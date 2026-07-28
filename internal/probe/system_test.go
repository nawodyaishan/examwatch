package probe

import (
	"context"
	"testing"
	"time"
)

type mockSysSampler struct {
	cpu float64
	mem float64
}

func (m *mockSysSampler) SampleCPU(ctx context.Context) (float64, error) {
	return m.cpu, nil
}

func (m *mockSysSampler) SampleMem(ctx context.Context) (float64, error) {
	return m.mem, nil
}

func TestSystemProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	probe := &SystemProbe{
		Sampler: &mockSysSampler{
			cpu: 15.5,
			mem: 45.0,
		},
		Interval: 10 * time.Millisecond,
	}

	out := make(chan interface{}, 10)
	go probe.Start(ctx, out)

	select {
	case v := <-out:
		sample, ok := v.(SystemSample)
		if !ok {
			t.Fatalf("expected SystemSample")
		}
		if sample.CPUPercent != 15.5 {
			t.Errorf("expected CPU 15.5, got %v", sample.CPUPercent)
		}
		if sample.MemUsedPercent != 45.0 {
			t.Errorf("expected Mem 45.0, got %v", sample.MemUsedPercent)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("timeout waiting for sample")
	}
}
