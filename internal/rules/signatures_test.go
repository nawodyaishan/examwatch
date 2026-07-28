package rules

import (
	"testing"
	"time"
)

var base = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

func at(seconds int) time.Time {
	return base.Add(time.Duration(seconds) * time.Second)
}

// buildLossSeries builds a series of n consecutive 1-second samples, the
// first `lossSeconds` of which are at 100% loss and the rest at 0%.
func buildLossSeries(lossSeconds, total int) Series {
	s := make(Series, 0, total)
	for i := 0; i < total; i++ {
		loss := 0.0
		if i < lossSeconds {
			loss = 100.0
		}
		s = append(s, Sample{Timestamp: at(i), LossPercent: loss, DNSLatencyMillis: -1})
	}
	return s
}

func TestEvalSustainedLoss(t *testing.T) {
	tests := []struct {
		name        string
		lossSeconds int
		want        Verdict
	}{
		{"one below threshold (4s)", SustainedLossMinConsecutiveSeconds - 1, PASS},
		{"exactly at threshold (5s)", SustainedLossMinConsecutiveSeconds, FAIL},
		{"one above threshold (6s)", SustainedLossMinConsecutiveSeconds + 1, FAIL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			series := buildLossSeries(tt.lossSeconds, tt.lossSeconds+3)
			got := EvalSustainedLoss(series)
			if got.Verdict != tt.want {
				t.Fatalf("lossSeconds=%d: got %v, want %v", tt.lossSeconds, got.Verdict, tt.want)
			}
			if tt.want == FAIL {
				if got.Evidence.Start.IsZero() || got.Evidence.End.IsZero() {
					t.Fatalf("expected non-zero evidence window on FAIL")
				}
			}
		})
	}

	t.Run("partial loss below 100 percent never triggers", func(t *testing.T) {
		s := make(Series, 0)
		for i := 0; i < 10; i++ {
			s = append(s, Sample{Timestamp: at(i), LossPercent: 99.9, DNSLatencyMillis: -1})
		}
		if got := EvalSustainedLoss(s); got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})

	t.Run("empty series", func(t *testing.T) {
		if got := EvalSustainedLoss(nil); got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})
}

func TestEvalIPChurn(t *testing.T) {
	tests := []struct {
		name string
		ips  []string
		want Verdict
	}{
		{"zero changes (below threshold)", []string{"1.1.1.1", "1.1.1.1", "1.1.1.1"}, PASS},
		{"exactly one change (at threshold)", []string{"1.1.1.1", "1.1.1.1", "2.2.2.2"}, WARN},
		{"two changes (above threshold)", []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}, WARN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := make(Series, 0, len(tt.ips))
			for i, ip := range tt.ips {
				s = append(s, Sample{Timestamp: at(i), PublicIP: ip, DNSLatencyMillis: -1})
			}
			got := EvalIPChurn(s)
			if got.Verdict != tt.want {
				t.Fatalf("ips=%v: got %v, want %v", tt.ips, got.Verdict, tt.want)
			}
		})
	}

	t.Run("unmeasured samples (empty PublicIP) are ignored", func(t *testing.T) {
		s := Series{
			{Timestamp: at(0), PublicIP: "1.1.1.1", DNSLatencyMillis: -1},
			{Timestamp: at(1), PublicIP: "", DNSLatencyMillis: -1},
			{Timestamp: at(2), PublicIP: "1.1.1.1", DNSLatencyMillis: -1},
		}
		if got := EvalIPChurn(s); got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})
}

// buildRTTSeries builds one sample per second for `total` seconds, using
// the given constant RTT except the samples in [spikeStart, spikeEnd)
// (indices) which alternate to create the desired stddev.
func buildRTTSeries(total int, rtts []float64) Series {
	s := make(Series, 0, total)
	for i := 0; i < total; i++ {
		s = append(s, Sample{Timestamp: at(i), RTTMillis: rtts[i], DNSLatencyMillis: -1})
	}
	return s
}

func TestEvalJitterSpike(t *testing.T) {
	// Alternate between two RTT values so the trailing-window stddev is
	// deterministic: for values alternating a,b the population stddev is
	// |a-b|/2.
	makeAlternating := func(n int, low, high float64) []float64 {
		out := make([]float64, n)
		for i := range out {
			if i%2 == 0 {
				out[i] = low
			} else {
				out[i] = high
			}
		}
		return out
	}

	t.Run("just below stddev threshold never triggers", func(t *testing.T) {
		// diff/2 = 149 => diff = 298
		rtts := makeAlternating(30, 0, 298)
		s := buildRTTSeries(30, rtts)
		got := EvalJitterSpike(s)
		if got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})

	t.Run("stddev above threshold but not sustained past 10s never triggers", func(t *testing.T) {
		// High stddev only for a short burst (well under 10s of sustained duration).
		rtts := make([]float64, 30)
		for i := range rtts {
			rtts[i] = 0
		}
		// Small burst around index 15-16 to create a brief stddev spike.
		rtts[15] = 1000
		s := buildRTTSeries(30, rtts)
		got := EvalJitterSpike(s)
		if got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})

	t.Run("stddev above threshold sustained over 10s triggers FAIL", func(t *testing.T) {
		// diff/2 = 151 => diff = 302, sustained across the whole series.
		rtts := makeAlternating(30, 0, 302)
		s := buildRTTSeries(30, rtts)
		got := EvalJitterSpike(s)
		if got.Verdict != FAIL {
			t.Fatalf("got %v, want FAIL", got.Verdict)
		}
		if got.Evidence.Start.IsZero() || got.Evidence.End.IsZero() {
			t.Fatalf("expected non-zero evidence window on FAIL")
		}
	})
}

func TestEvalACDrop(t *testing.T) {
	tests := []struct {
		name    string
		samples []bool // AC connected state per sample; all HasACReading=true
		want    Verdict
	}{
		{"never flips to false (below threshold)", []bool{true, true, true}, PASS},
		{"flips once (at threshold)", []bool{true, true, false}, WARN},
		{"flips twice (above threshold)", []bool{true, false, true, false}, WARN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := make(Series, 0, len(tt.samples))
			for i, ac := range tt.samples {
				s = append(s, Sample{Timestamp: at(i), ACConnected: ac, HasACReading: true, DNSLatencyMillis: -1})
			}
			got := EvalACDrop(s)
			if got.Verdict != tt.want {
				t.Fatalf("samples=%v: got %v, want %v", tt.samples, got.Verdict, tt.want)
			}
		})
	}

	t.Run("starting on battery (false) is not itself a flip", func(t *testing.T) {
		s := Series{
			{Timestamp: at(0), ACConnected: false, HasACReading: true, DNSLatencyMillis: -1},
			{Timestamp: at(1), ACConnected: false, HasACReading: true, DNSLatencyMillis: -1},
		}
		if got := EvalACDrop(s); got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})

	t.Run("unmeasured samples (HasACReading=false) are ignored", func(t *testing.T) {
		s := Series{
			{Timestamp: at(0), ACConnected: true, HasACReading: true, DNSLatencyMillis: -1},
			{Timestamp: at(1), HasACReading: false, DNSLatencyMillis: -1},
			{Timestamp: at(2), ACConnected: true, HasACReading: true, DNSLatencyMillis: -1},
		}
		if got := EvalACDrop(s); got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})
}

func TestEvalDNSStall(t *testing.T) {
	tests := []struct {
		name    string
		latency float64
		want    Verdict
	}{
		{"just below threshold (1999.9ms)", DNSStallThresholdMillis - 0.1, PASS},
		{"exactly at threshold (2000ms)", DNSStallThresholdMillis, PASS}, // strictly-greater-than per spec
		{"just above threshold (2000.1ms)", DNSStallThresholdMillis + 0.1, FAIL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Series{{Timestamp: at(0), DNSLatencyMillis: tt.latency}}
			got := EvalDNSStall(s)
			if got.Verdict != tt.want {
				t.Fatalf("latency=%v: got %v, want %v", tt.latency, got.Verdict, tt.want)
			}
		})
	}

	t.Run("unmeasured samples (negative latency) are ignored", func(t *testing.T) {
		s := Series{{Timestamp: at(0), DNSLatencyMillis: -1}}
		if got := EvalDNSStall(s); got.Verdict != PASS {
			t.Fatalf("got %v, want PASS", got.Verdict)
		}
	})
}

func TestVerdictString(t *testing.T) {
	tests := map[Verdict]string{PASS: "PASS", WARN: "WARN", FAIL: "FAIL", Verdict(99): "UNKNOWN"}
	for v, want := range tests {
		if got := v.String(); got != want {
			t.Fatalf("Verdict(%d).String() = %q, want %q", v, got, want)
		}
	}
}

func TestWorse(t *testing.T) {
	tests := []struct {
		a, b, want Verdict
	}{
		{PASS, PASS, PASS},
		{PASS, WARN, WARN},
		{WARN, PASS, WARN},
		{WARN, FAIL, FAIL},
		{FAIL, WARN, FAIL},
		{PASS, FAIL, FAIL},
		{FAIL, PASS, FAIL},
	}
	for _, tt := range tests {
		if got := worse(tt.a, tt.b); got != tt.want {
			t.Fatalf("worse(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
