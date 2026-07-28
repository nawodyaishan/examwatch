package probe

import (
	"context"
	"testing"
	"time"
)

func TestParsePmset(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantErr     bool
		acConnected bool
		batteryPct  int
		charging    bool
		remain      string
	}{
		{
			name: "AC connected charging",
			output: `Now drawing from 'AC Power'
 -InternalBattery-0 (id=123)	85%; charging; 1:12 remaining present: true`,
			wantErr:     false,
			acConnected: true,
			batteryPct:  85,
			charging:    true,
			remain:      "1:12",
		},
		{
			name: "AC connected charged",
			output: `Now drawing from 'AC Power'
 -InternalBattery-0 (id=123)	100%; charged; 0:00 remaining present: true`,
			wantErr:     false,
			acConnected: true,
			batteryPct:  100,
			charging:    true, // treated as not discharging
			remain:      "0:00",
		},
		{
			name: "Battery discharging",
			output: `Now drawing from 'Battery Power'
 -InternalBattery-0 (id=123)	45%; discharging; 2:30 remaining present: true`,
			wantErr:     false,
			acConnected: false,
			batteryPct:  45,
			charging:    false,
			remain:      "2:30",
		},
		{
			name:    "Invalid output",
			output:  `Some weird output`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePmset(tt.output)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParsePmset() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				if got.ACConnected != tt.acConnected {
					t.Errorf("ACConnected = %v, want %v", got.ACConnected, tt.acConnected)
				}
				if got.BatteryPct != tt.batteryPct {
					t.Errorf("BatteryPct = %v, want %v", got.BatteryPct, tt.batteryPct)
				}
				if got.Charging != tt.charging {
					t.Errorf("Charging = %v, want %v", got.Charging, tt.charging)
				}
				if got.TimeRemaining != tt.remain {
					t.Errorf("TimeRemaining = %v, want %v", got.TimeRemaining, tt.remain)
				}
			}
		})
	}
}

type mockBattReader struct {
	outputs []string
	idx     int
}

func (m *mockBattReader) Read(ctx context.Context) (string, error) {
	if m.idx >= len(m.outputs) {
		return m.outputs[len(m.outputs)-1], nil
	}
	out := m.outputs[m.idx]
	m.idx++
	return out, nil
}

func TestPowerProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockOutput1 := `Now drawing from 'AC Power'
 -InternalBattery-0	100%; charged; 0:00 remaining present: true`
	mockOutput2 := `Now drawing from 'Battery Power'
 -InternalBattery-0	99%; discharging; 5:00 remaining present: true`

	probe := &PowerProbe{
		Reader: &mockBattReader{
			outputs: []string{mockOutput1, mockOutput1, mockOutput2},
		},
		Interval: 10 * time.Millisecond,
	}

	out := make(chan interface{}, 10)
	go probe.Start(ctx, out)

	// Expect two events: initial (AC) and transition to Battery
	
	// Wait for first event
	select {
	case v := <-out:
		ev := v.(PowerEvent)
		if !ev.ACConnected {
			t.Errorf("expected first event to be AC connected")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for event 1")
	}

	// Wait for second event
	select {
	case v := <-out:
		ev := v.(PowerEvent)
		if ev.ACConnected {
			t.Errorf("expected second event to be Battery Power")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for event 2")
	}
}
