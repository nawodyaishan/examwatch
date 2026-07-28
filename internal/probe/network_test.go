package probe

import (
	"context"
	"testing"
	"time"
)

type mockPinger struct {
	rtt time.Duration
	err error
}

func (m *mockPinger) Ping(ctx context.Context, host string) (time.Duration, error) {
	return m.rtt, m.err
}

type mockResolver struct {
	rtt time.Duration
	err error
}

func (m *mockResolver) LookupHost(ctx context.Context, host string) (time.Duration, error) {
	return m.rtt, m.err
}

type mockIPFetcher struct {
	ip  string
	err error
}

func (m *mockIPFetcher) GetIP(ctx context.Context) (string, error) {
	return m.ip, m.err
}

func TestNetworkProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe := &NetworkProbe{
		Pinger:    &mockPinger{rtt: 10 * time.Millisecond},
		Resolver:  &mockResolver{rtt: 5 * time.Millisecond},
		IPFetcher: &mockIPFetcher{ip: "1.2.3.4"},
		Interval:  10 * time.Millisecond,
	}

	out := make(chan interface{}, 10)
	go probe.Start(ctx, out)

	select {
	case v := <-out:
		sample, ok := v.(NetworkSample)
		if !ok {
			t.Fatalf("expected NetworkSample")
		}
		if sample.PublicIP != "1.2.3.4" {
			t.Errorf("expected 1.2.3.4, got %s", sample.PublicIP)
		}
		if sample.RTT1111 != 10*time.Millisecond {
			t.Errorf("expected 10ms, got %v", sample.RTT1111)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for sample")
	}
}

func TestCalculateNetworkStats(t *testing.T) {
	rtts := []time.Duration{
		10 * time.Millisecond,
		20 * time.Millisecond,
		15 * time.Millisecond,
		0, // lost
		25 * time.Millisecond,
	}
	loss, jitter := calculateNetworkStats(rtts)
	if loss != 20.0 {
		t.Errorf("expected 20%% loss, got %v", loss)
	}
	// valid: 10, 20, 15, 25. Mean: 17.5
	// diffs: -7.5, 2.5, -2.5, 7.5
	// sq: 56.25, 6.25, 6.25, 56.25 => sum = 125
	// variance: 125 / 4 = 31.25
	// sqrt(31.25) ~ 5.59ms
	if jitter == 0 {
		t.Errorf("expected non-zero jitter")
	}
}
