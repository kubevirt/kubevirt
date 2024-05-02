# Hyperconverged Cluster Operator metrics

### cluster:vmi_request_cpu_cores:sum
Sum of CPU core requests for all running virt-launcher VMIs across the entire Kubevirt cluster. Type: Gauge.

### cnv_abnormal
Monitors resources for potential problems. Type: Gauge.

### kubevirt_hco_hyperconverged_cr_exists
Indicates whether the HyperConverged custom resource exists (1) or not (0). Type: Gauge.

### kubevirt_hco_out_of_band_modifications_total
Count of out-of-band modifications overwritten by HCO. Type: Counter.

### kubevirt_hco_single_stack_ipv6
Indicates whether the underlying cluster is single stack IPv6 (1) or not (0). Type: Gauge.

### kubevirt_hco_system_health_status
Indicates whether the system health status is healthy (0), warning (1), or error (2), by aggregating the conditions of HCO and its secondary resources. Type: Gauge.

### kubevirt_hco_unsafe_modifications
Count of unsafe modifications in the HyperConverged annotations. Type: Gauge.

### kubevirt_hyperconverged_operator_health_status
Indicates whether HCO and its secondary resources health status is healthy (0), warning (1) or critical (2), based both on the firing alerts that impact the operator health, and on kubevirt_hco_system_health_status metric. Type: Gauge.

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
