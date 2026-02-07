## KubeVirt Monitoring

### Observability Compatibility Policy

This policy covers all KubeVirt observability signal. Like: metrics (and their names, label sets, and types), Prometheus recording rules, and alerting rules. Unless explicitly stated otherwise, these signals are considered implementation details and are subject to change.

- Stability: KubeVirt does not guarantee long-term backwards compatibility for observability signals. Names, labels, types, and semantics may change between releases to improve correctness, performance, or operability.
- Deprecation: When feasible, we will deprecate renamed or removed signals by:
  - Marking the old name as Deprecated in documentation.
  - Optionally providing short-lived compatibility recording rules (aliases) that map new signals to old names.
  - Keeping deprecated signals for at least one minor release when possible. In exceptional cases (security, correctness, or scalability), changes may occur without a deprecation window.
- Communication: Material changes will be documented in release notes and reflected in `docs/observability/metrics.md`. Alert and rule updates will also be surfaced via PR descriptions.
- Consumer guidance: Dashboards and alerts should:
  - When creating PromQL queries, expect label sets to change; avoid relying on exhaustive or fixed labels. Select, join, and group by the minimum labels required.
    - Example: Prefer `sum by (namespace)(...)` over `sum by (namespace,pod,container,instance)(...)` when possible.
  - Treat deprecated signals as temporary and migrate to replacements promptly.

Contributors adding or changing observability signals should update documentation, consider temporary compatibility rules if practical, and include migration notes in the PR and release notes.

### KubeVirt Metrics
#### Naming a New KubeVirt Metrics:

The KubeVirt metrics should align with the Kubernetes metrics names.

The KubeVirt Users should have the same experience when searching for a node, container, pod and virtual machine metrics.

**Naming requirements:**
1. Check if a similar Kubernetes metric, for node, container or pod, exists and try to align to it.
2. KubeVirt metric for a running VM should have a `kubevirt_vmi_` prefix

For Example, see the following Kubernetes network metrics:
- **node**_network_receive_packets_total
- **node**_network_transmit_packets_total
- **container**_network_receive_packets_total
- **container**_network_transmit_packets_total

The KubeVirt metrics for vmi should be:
- **kubevirt_vmi**_network_receive_packets_total
- **kubevirt_vmi**_network_transmit_packets_total

### KubeVirt Recording Rules

#### Naming a New KubeVirt Recording Rule:

The Prometheus recording rules appear in Prometheus as metrics.

Recording rules should be in the format of <level>:<metric>:<operation>, based on the Prometheus naming best practices.
In order to easily identify the KubeVirt recording rules, they should include a `kubevirt_` prefix in the <metric> section like in metrics.

### KubeVirt Alerts Rules

When creating a KubeVirt alert rule, please see the following :

1. Use [recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#recording-rules) when doing calculations.
2. Create an alert runbook at [KubeVirt runbooks](https://github.com/kubevirt/monitoring/tree/main/docs/runbooks).
3. Alert rule must include `runbook_url` with the link to your runbook from step #2.
4. Alert rule must include `severity`. One of: `critical`, `warning`, `info`.

    NOTE:
     - Critical alerts - When the service is down and you loss critical functionality, an action is required immediately.
     - Warning alerts - When an alert require user intervention. A more serious issue may develop if this is not resolved soon.
     - Info alerts - When a minor problem has been detected. It should be resolved relatively soon and not ignored.

5. Alert `message` must be verbose, since it is being propagated to the [observability/metrics.md](https://github.com/kubevirt/kubevirt/blob/main/docs/observability/metrics.md) file, when running `make-generate`.
