# virt-exportproxy horizontal pod autoscaling

`virt-exportproxy` is managed by an operator-owned HorizontalPodAutoscaler
(`virt-exportproxy-hpa`) on multi-node clusters. Single-node clusters skip the
HPA and keep a single replica.

## Metrics profiles

The operator auto-detects which metrics `custom.metrics.k8s.io` can serve and
annotates the HPA with `kubevirt.io/export-proxy-hpa-metrics-profile`:

| Profile | When selected | Scale signal |
|---------|---------------|--------------|
| `custom-metrics` | prometheus-adapter (or equivalent) exposes the transfer metrics below | Active transfers plus CPU/memory utilization |
| `resource` | Custom metrics are unavailable | CPU and memory utilization vs requests |

Detection is cached so brief adapter outages do not flip the HPA on every
reconcile.

## Custom transfer metrics

Preferred scale signals (see `pkg/monitoring/metrics/virt-exportproxy`):

- `kubevirt_exportproxy_active_transfers` (Pods average)
- `kubevirt_exportproxy_active_transfers_pod_max` (namespace max, gated)

The custom-metrics profile also keeps resource CPU/memory metrics so soft
CPU/memory admission pressure can still drive scale-out when active transfers
are below the transfer targets.

A working prometheus-adapter rule fragment lives at:

[examples/export-proxy-prometheus-adapter-rules.yaml](examples/export-proxy-prometheus-adapter-rules.yaml)

Keep rule names and the gated-max floor synchronized with
`pkg/virt-operator/resource/generate/components/hpa.go` and
`pkg/exportproxy/admission/limits.go`.

## Resource fallback

When custom metrics are missing, the HPA uses only CPU and memory utilization.
Export-proxy pods are Burstable (requests below limits). HPA
`AverageUtilization` is relative to **requests**, so targets are above 100% of
request in order to align scale-out with soft admission near 70% of **limits**.
See the constants in `pkg/virt-operator/resource/generate/components/hpa.go`.
