// Package probe collects network, power, and system telemetry samples.
package probe

import (
	"context"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

type NetworkSample struct {
	Timestamp   time.Time     `json:"timestamp"`
	RTT1111     time.Duration `json:"rtt_1111"`
	RTT8888     time.Duration `json:"rtt_8888"`
	LossPercent float64       `json:"loss_percent"`
	Jitter      time.Duration `json:"jitter"`
	PublicIP    string        `json:"public_ip"`
	DNSLatency  time.Duration `json:"dns_latency"`
}

type Pinger interface {
	Ping(ctx context.Context, host string) (time.Duration, error)
}

type Resolver interface {
	LookupHost(ctx context.Context, host string) (time.Duration, error)
}

type IPFetcher interface {
	GetIP(ctx context.Context) (string, error)
}

type NetworkProbe struct {
	Pinger    Pinger
	Resolver  Resolver
	IPFetcher IPFetcher
	Interval  time.Duration
}

func NewNetworkProbe(interval time.Duration) *NetworkProbe {
	return &NetworkProbe{
		Pinger:    &defaultPinger{},
		Resolver:  &defaultResolver{},
		IPFetcher: &defaultIPFetcher{},
		Interval:  interval,
	}
}

func (p *NetworkProbe) Start(ctx context.Context, out chan<- interface{}) {
	ticker := time.NewTicker(p.Interval)
	defer ticker.Stop()

	dnsTicker := time.NewTicker(5 * time.Second)
	defer dnsTicker.Stop()

	ipTicker := time.NewTicker(15 * time.Second)
	defer ipTicker.Stop()

	// Rolling windows
	var rtts []time.Duration
	var lastIP string
	var lastDNS time.Duration

	// Initial fetch
	lastIP, _ = p.IPFetcher.GetIP(ctx)
	lastDNS, _ = p.Resolver.LookupHost(ctx, "google.com")

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			rtt1, err1 := p.Pinger.Ping(ctx, "1.1.1.1")
			rtt2, err2 := p.Pinger.Ping(ctx, "8.8.8.8")

			// Record RTT for rolling window (using 1.1.1.1 for simplicity, or 8.8.8.8 if 1.1.1.1 fails)
			rtt := rtt1
			if err1 != nil && err2 == nil {
				rtt = rtt2
			}
			
			isLoss := err1 != nil && err2 != nil
			if isLoss {
				rtt = 0 // represents loss in our window
			}

			rtts = append(rtts, rtt)
			if len(rtts) > 10 {
				rtts = rtts[1:]
			}

			loss, jitter := calculateNetworkStats(rtts)

			sample := NetworkSample{
				Timestamp:   t,
				RTT1111:     rtt1,
				RTT8888:     rtt2,
				LossPercent: loss,
				Jitter:      jitter,
				PublicIP:    lastIP,
				DNSLatency:  lastDNS,
			}
			select {
			case out <- sample:
			case <-ctx.Done():
				return
			}
		case <-dnsTicker.C:
			if d, err := p.Resolver.LookupHost(ctx, "google.com"); err == nil {
				lastDNS = d
			} else {
				lastDNS = 0 // indicates failure
			}
		case <-ipTicker.C:
			if ip, err := p.IPFetcher.GetIP(ctx); err == nil {
				lastIP = ip
			}
		}
	}
}

func calculateNetworkStats(rtts []time.Duration) (loss float64, jitter time.Duration) {
	if len(rtts) == 0 {
		return 0, 0
	}
	var lost int
	var validRTTs []time.Duration
	var sum time.Duration
	for _, r := range rtts {
		if r == 0 {
			lost++
		} else {
			validRTTs = append(validRTTs, r)
			sum += r
		}
	}
	loss = float64(lost) / float64(len(rtts)) * 100

	if len(validRTTs) > 1 {
		mean := float64(sum) / float64(len(validRTTs))
		var varianceSum float64
		for _, r := range validRTTs {
			diff := float64(r) - mean
			varianceSum += diff * diff
		}
		variance := varianceSum / float64(len(validRTTs))
		jitter = time.Duration(math.Sqrt(variance))
	}

	return loss, jitter
}

// defaultPinger uses pro-bing
type defaultPinger struct{}

func (dp *defaultPinger) Ping(ctx context.Context, host string) (time.Duration, error) {
	pinger, err := probing.NewPinger(host)
	if err != nil {
		return 0, err
	}
	pinger.SetPrivileged(false)
	pinger.Count = 1
	pinger.Timeout = 800 * time.Millisecond // 800ms timeout for individual ping
	
	errCh := make(chan error, 1)
	go func() {
		errCh <- pinger.Run()
	}()
	
	select {
	case <-ctx.Done():
		pinger.Stop()
		return 0, ctx.Err()
	case err := <-errCh:
		if err != nil {
			return 0, err
		}
	}
	
	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		// Fallback to TCP dial
		start := time.Now()
		d := net.Dialer{Timeout: 800 * time.Millisecond}
		conn, err2 := d.DialContext(ctx, "tcp", host+":443")
		if err2 != nil {
			return 0, fmt.Errorf("ping and tcp dial both failed")
		}
		_ = conn.Close()
		return time.Since(start), nil
	}
	return stats.AvgRtt, nil
}

// defaultResolver uses net.Resolver
type defaultResolver struct{}

func (dr *defaultResolver) LookupHost(ctx context.Context, host string) (time.Duration, error) {
	start := time.Now()
	var r net.Resolver
	_, err := r.LookupHost(ctx, host)
	return time.Since(start), err
}

// defaultIPFetcher uses an external API
type defaultIPFetcher struct{}

func (di *defaultIPFetcher) GetIP(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
