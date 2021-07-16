## KubeVirt Monitoring
 
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

In order to easily identify the KubeVirt recording rules, they should have a `kubevirt_` prefix.

### KubeVirt Alerts Rules

When creating a KubeVirt alert rule, please see the following :

1. Use [recording rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#recording-rules) when doing calculations.
2. Create an alert runbook at [KubeVirt runbooks](https://github.com/kubevirt/monitoring/tree/main/runbooks).
3. Alert rule must include `runbook_url` with the link to your runbook from step #2.
4. Alert rule must include `severity`. One of: `critical`, `warning`, `info`.

    NOTE:
     - Critical alerts - When the service is down and you loss critical functionality, an action is required immediately.
     - Warning alerts - When an alert require user intervention. A more serious issue may develop if this is not resolved soon.
     - Info alerts - When a minor problem has been detected. It should be resolved relatively soon and not ignored.

5. Alert `message` must be verbose, since it is being propagated to the [metrics.md](https://github.com/kubevirt/kubevirt/blob/main/docs/monitoring-guidelines.md) file, when running `make-generate`.
