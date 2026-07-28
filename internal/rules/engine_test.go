package rules

import (
	"testing"
	"time"
)

func TestEngineEvaluate(t *testing.T) {
	e := NewEngine()
	s := Series{
		{Timestamp: time.Unix(0, 0), ACConnected: true, HasACReading: true, DNSLatencyMillis: 0},
		{Timestamp: time.Unix(1, 0), ACConnected: false, HasACReading: true, DNSLatencyMillis: 0}, // triggers ACDrop (WARN)
		{Timestamp: time.Unix(2, 0), DNSLatencyMillis: 2500},                                      // triggers DNSStall (FAIL)
	}

	res := e.Evaluate(s)
	if res.Verdict != FAIL {
		t.Fatalf("expected FAIL, got %v", res.Verdict)
	}
}

func TestCalcStdDevLess2(t *testing.T) {
	if calcStdDev([]float64{1.0}) != 0 {
		t.Fatalf("expected 0 for < 2 samples")
	}
}
