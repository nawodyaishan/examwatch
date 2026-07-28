package rules

import (
	"math"
	"time"
)

type Verdict int

const (
	PASS Verdict = iota
	WARN
	FAIL
)

func (v Verdict) String() string {
	switch v {
	case PASS:
		return "PASS"
	case WARN:
		return "WARN"
	case FAIL:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

type Sample struct {
	Timestamp        time.Time
	LossPercent      float64
	PublicIP         string
	RTTMillis        float64
	ACConnected      bool
	HasACReading     bool
	DNSLatencyMillis float64
}

type Series []Sample

type Evidence struct {
	Start time.Time
	End   time.Time
}

type SignatureResult struct {
	Verdict  Verdict
	Evidence Evidence
}

// worse returns the higher severity verdict.
func worse(a, b Verdict) Verdict {
	if b > a {
		return b
	}
	return a
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Evaluate(series Series) SignatureResult {
	sigs := []SignatureResult{
		EvalSustainedLoss(series),
		EvalIPChurn(series),
		EvalJitterSpike(series),
		EvalACDrop(series),
		EvalDNSStall(series),
	}

	final := PASS
	for _, res := range sigs {
		final = worse(final, res.Verdict)
	}

	return SignatureResult{Verdict: final} // Could add evidence if needed
}

func EvalSustainedLoss(series Series) SignatureResult {
	var start time.Time
	inLoss := false

	for _, s := range series {
		if s.LossPercent == 100 {
			if !inLoss {
				start = s.Timestamp
				inLoss = true
			}
			if s.Timestamp.Sub(start) >= time.Duration(SustainedLossMinConsecutiveSeconds)*time.Second {
				return SignatureResult{
					Verdict:  FAIL,
					Evidence: Evidence{Start: start, End: s.Timestamp},
				}
			}
		} else if inLoss {
			if s.Timestamp.Sub(start) >= time.Duration(SustainedLossMinConsecutiveSeconds)*time.Second {
				return SignatureResult{
					Verdict:  FAIL,
					Evidence: Evidence{Start: start, End: s.Timestamp},
				}
			}
			inLoss = false
		}
	}

	if inLoss {
		last := series[len(series)-1]
		if last.Timestamp.Sub(start) >= time.Duration(SustainedLossMinConsecutiveSeconds)*time.Second {
			return SignatureResult{
				Verdict:  FAIL,
				Evidence: Evidence{Start: start, End: last.Timestamp},
			}
		}
	}
	return SignatureResult{Verdict: PASS}
}

func EvalIPChurn(series Series) SignatureResult {
	var initialIP string
	var start time.Time

	for _, s := range series {
		if s.PublicIP != "" {
			if initialIP == "" {
				initialIP = s.PublicIP
				start = s.Timestamp
			} else if s.PublicIP != initialIP {
				// WARN in test
				return SignatureResult{
					Verdict:  WARN,
					Evidence: Evidence{Start: start, End: s.Timestamp},
				}
			}
		}
	}
	return SignatureResult{Verdict: PASS}
}

func calcStdDev(samples []float64) float64 {
	if len(samples) < 2 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += s
	}
	mean := sum / float64(len(samples))
	var variance float64
	for _, s := range samples {
		diff := s - mean
		variance += diff * diff
	}
	variance /= float64(len(samples))
	return math.Sqrt(variance)
}

func EvalJitterSpike(series Series) SignatureResult {
	var rtts []float64
	var spikeStart time.Time
	inSpike := false

	for _, s := range series {
		rtts = append(rtts, s.RTTMillis)

		if len(rtts) > JitterSpikeWindowSize {
			rtts = rtts[1:]
		}

		if len(rtts) == JitterSpikeWindowSize {
			sd := calcStdDev(rtts)
			if sd > JitterSpikeStdDevThreshold {
				if !inSpike {
					spikeStart = s.Timestamp
					inSpike = true
				} else if int(s.Timestamp.Sub(spikeStart).Seconds()) > JitterSpikeDurationSeconds {
					return SignatureResult{
						Verdict:  FAIL,
						Evidence: Evidence{Start: spikeStart, End: s.Timestamp},
					}
				}
			} else {
				inSpike = false
			}
		}
	}
	return SignatureResult{Verdict: PASS}
}

func EvalACDrop(series Series) SignatureResult {
	hasSeenAC := false
	for _, s := range series {
		if s.HasACReading {
			if s.ACConnected {
				hasSeenAC = true
			} else if hasSeenAC {
				return SignatureResult{
					Verdict:  WARN, // test expects WARN
					Evidence: Evidence{Start: s.Timestamp, End: s.Timestamp},
				}
			}
		}
	}
	return SignatureResult{Verdict: PASS}
}

func EvalDNSStall(series Series) SignatureResult {
	for _, s := range series {
		// Test says: unmeasured samples (negative latency) are ignored
		if s.DNSLatencyMillis >= 0 {
			if s.DNSLatencyMillis > DNSStallThresholdMillis {
				return SignatureResult{
					Verdict:  FAIL,
					Evidence: Evidence{Start: s.Timestamp, End: s.Timestamp},
				}
			}
		}
	}
	return SignatureResult{Verdict: PASS}
}
