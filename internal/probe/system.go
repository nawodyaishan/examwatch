package probe

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

type SystemSample struct {
	Timestamp      time.Time `json:"timestamp"`
	CPUPercent     float64   `json:"cpu_percent"`
	MemUsedPercent float64   `json:"mem_used_percent"`
}

type SysSampler interface {
	SampleCPU(ctx context.Context) (float64, error)
	SampleMem(ctx context.Context) (float64, error)
}

type defaultSysSampler struct{}

func (d *defaultSysSampler) SampleCPU(ctx context.Context) (float64, error) {
	pcts, err := cpu.PercentWithContext(ctx, 0, false)
	if err != nil || len(pcts) == 0 {
		return 0, err
	}
	return pcts[0], nil
}

func (d *defaultSysSampler) SampleMem(ctx context.Context) (float64, error) {
	v, err := mem.VirtualMemoryWithContext(ctx)
	if err != nil {
		return 0, err
	}
	return v.UsedPercent, nil
}

type SystemProbe struct {
	Sampler  SysSampler
	Interval time.Duration
}

func NewSystemProbe(interval time.Duration) *SystemProbe {
	return &SystemProbe{
		Sampler:  &defaultSysSampler{},
		Interval: interval,
	}
}

func (p *SystemProbe) Start(ctx context.Context, out chan<- interface{}) {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			cpuPct, err1 := p.Sampler.SampleCPU(ctx)
			memPct, err2 := p.Sampler.SampleMem(ctx)

			if err1 == nil && err2 == nil {
				sample := SystemSample{
					Timestamp:      t,
					CPUPercent:     cpuPct,
					MemUsedPercent: memPct,
				}
				select {
				case out <- sample:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
