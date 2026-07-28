package rules

const (
	SustainedLossMinConsecutiveSeconds = 5
	JitterSpikeStdDevThreshold         = 150.0
	JitterSpikeDurationSeconds         = 10
	JitterSpikeWindowSize              = 10
	DNSStallThresholdMillis            = 2000.0
)

const (
	SigSustainedLoss = "SUSTAINED_LOSS"
	SigIPChurn       = "IP_CHURN"
	SigJitterSpike   = "JITTER_SPIKE"
	SigACDrop        = "AC_DROP"
	SigDNSStall      = "DNS_STALL"
)
