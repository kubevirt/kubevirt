# virt-exportproxy horizontal pod autoscaling

`virt-exportproxy` is managed by an operator-owned HorizontalPodAutoscaler
(`virt-exportproxy-hpa`) on multi-node clusters. Single-node clusters skip the
HPA and keep a single replica. Pinning replicas with a
`CustomizeComponents` JSON patch on `/spec/replicas` also removes the HPA
(opting out of autoscaling).

## Metrics profiles

The operator auto-detects which metrics `custom.metrics.k8s.io` can serve and
annotates the HPA with `kubevirt.io/export-proxy-hpa-metrics-profile`:

| Profile | When selected | Scale signal |
|---------|---------------|--------------|
| `custom-metrics` | prometheus-adapter (or equivalent) exposes the transfer metrics below | Active transfers plus CPU utilization |
| `resource` | Custom metrics are unavailable | CPU utilization vs requests |

Detection is cached so brief adapter outages do not flip the HPA on every
reconcile.

Memory is **not** used as an HPA signal: Go rarely returns heap to the OS, so
RSS after a burst can keep replicas pinned high indefinitely.

## Custom transfer metrics

Preferred scale signals (see `pkg/monitoring/metrics/virt-exportproxy`):

- `kubevirt_exportproxy_active_transfers` (Pods average, target **130** per pod)
- `kubevirt_exportproxy_active_transfers_pod_max` (namespace max, gated, target **150** per pod)

The custom-metrics profile also keeps resource CPU so soft CPU admission
pressure can still drive scale-out when active transfers are below the
transfer targets.

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
are still covered by soft CPU admission and the resource HPA CPU metric.

A working prometheus-adapter rule fragment lives at:

[examples/export-proxy-prometheus-adapter-rules.yaml](examples/export-proxy-prometheus-adapter-rules.yaml)

Keep rule names and the gated-max floor synchronized with
`pkg/virt-operator/resource/generate/components/hpa.go` and
`pkg/exportproxy/admission/limits.go`.

## Resource fallback

When custom metrics are missing, the HPA uses only CPU utilization.
Export-proxy pods are Burstable (requests below limits). HPA
`AverageUtilization` is relative to **requests**, so the CPU target is above
100% of request in order to align scale-out with soft admission near 70% of
**limits**. See the constants in
`pkg/virt-operator/resource/generate/components/hpa.go`.

## Scale-down and graceful drain

The HPA uses a conservative scale-down policy (5-minute stabilization,
at most one pod or 10% per minute). Pods use a 10-minute
`terminationGracePeriodSeconds`. On SIGTERM the proxy:

1. Fails `/readyz` (removed from Service endpoints)
2. Rejects new transfers with HTTP 429
3. Waits for in-flight transfers (up to ~9 minutes)
4. Shuts down the HTTP server

## Placement

`virt-exportproxy` schedules with `AnyNode` so HPA scale-out can land on
worker nodes. Node selectors / tolerations come from `spec.workloads` when
set, otherwise from `spec.infra`. The previous default control-plane
preference (`RequireControlPlanePreferNonWorker`) is no longer applied.

Clusters that previously relied on control-plane-only placement should set
`spec.workloads` (or `spec.infra`) node selectors / tolerations explicitly,
or pin replicas with `CustomizeComponents` to opt out of the HPA.

## Unauthenticated traffic and soft admission

`virt-exportproxy` does **not** validate export tokens. Auth stays on
`virt-exportserver` (per-namespace token secrets). The proxy only routes to a
ready export backend.

To avoid unauthenticated floods driving HPA scale-out and to rebalance load:

1. **Reserve before proxying** – soft transfer / CPU / memory limits run
   before the backend TLS dial. Rejected requests get HTTP `429` with
   `Connection: close` so keep-alive clients reconnect and land on other
   pods.
2. **Auth-gated confirmation** – `transfers_total` and sustained active
   slots are confirmed only after the backend returns a response that is
   not `401`/`403`. Failed auth releases the reserved slot immediately.
3. **Periodic utilization sampling** – cgroup CPU/memory is refreshed on a
   background ticker so soft admission does not do synchronous sysfs reads
   per request.
