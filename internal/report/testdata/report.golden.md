# examwatch run report

## Verdict: FAIL [FAIL]

| Field | Value |
|---|---|
| Start time | 2026-07-28 14:00:00 UTC |
| Duration | 1h0m0s |
| Interval | 1s |
| Hostname | test-macbook |

## Charts

```
RTT (ms)         ▁▁▁█▁
Loss (%)         ▁█▁
```

## Timeline

- `2026-07-28 14:05:00 UTC` AC_DROP (WARN) started: AC power disconnected during the run
- `2026-07-28 14:10:00 UTC` SUSTAINED_LOSS (FAIL) started: 100% packet loss sustained for consecutive samples
- `2026-07-28 14:11:00 UTC` SUSTAINED_LOSS (FAIL) ended
- `2026-07-28 14:15:00 UTC` AC_DROP (WARN) ended

## Signatures

| Signature | Status | Evidence | Detail |
|---|---|---|---|
| SUSTAINED_LOSS | FAIL [FAIL] | 2026-07-28 14:10:00 UTC → 2026-07-28 14:11:00 UTC | 100% packet loss sustained for consecutive samples |
| AC_DROP | WARN [WARN] | 2026-07-28 14:05:00 UTC → 2026-07-28 14:15:00 UTC | AC power disconnected during the run |
| IP_CHURN | PASS [PASS] | — | — |

