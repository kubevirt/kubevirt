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

- `kubevirt_exportproxy_active_transfers` (Pods average, target **130** per pod)
- `kubevirt_exportproxy_active_transfers_pod_max` (namespace max, gated, target **150** per pod)

The custom-metrics profile also keeps resource CPU/memory metrics so soft
CPU/memory admission pressure can still drive scale-out when active transfers
are below the transfer targets.

### Transfer limits vs HPA targets

Per-pod transfer constants in `pkg/exportproxy/admission/limits.go` serve
different roles:

| Constant | Value | Role |
|----------|-------|------|
| `HPATargetAverageTransfers` | 130 | Fleet-average HPA scale target |
| `HPATargetMaxTransfers` | 150 | Hottest-pod HPA scale target (load-test capacity) |
| `SoftTransferLimit` | 170 | HTTP 429 admission ceiling and CAS cap on the metric |

`SoftTransferLimit` is intentionally above `HPATargetMaxTransfers`. Kubernetes
HPA applies a default **10% tolerance** and only scales when
`metric / target` is outside `[0.9, 1.1]`. With a CAS cap equal to the max
target (150/150 = 1.0), the hottest-pod metric could never clear tolerance.
Allowing the metric to reach **170** while the HPA target stays **150** gives
a ratio of ~1.13 at saturation, which triggers scale-out. CPU-bound workloads
are still covered by soft CPU/memory admission and the resource HPA metrics.

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

## Unauthenticated traffic and DoS controls

`virt-exportproxy` does **not** validate export tokens. Auth stays on
`virt-exportserver` (per-namespace token secrets). The proxy only routes to a
ready export backend.

To avoid unauthenticated floods driving HPA scale-out:

1. **Auth-gated admission** – `TryRecordTransferStarted` (soft transfer /
   CPU / memory limits and `kubevirt_exportproxy_active_transfers`) runs only
   after the backend returns a response that is not `401`/`403`. Failed auth
   does not consume admission slots or move HPA metrics.
2. **Per-source-IP concurrency** – at most
   `MaxConcurrentRequestsPerIP` (32) in-flight proxied requests per
   `Request.RemoteAddr` peer. Excess requests get HTTP `429` before the
   backend is contacted. `X-Forwarded-For` is not trusted (spoofable); real
   client IPs require the ingress/load balancer to present them as the TCP
   peer.
