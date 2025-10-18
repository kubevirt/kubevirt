### Benchmarks

This document shares some of the performance benchmarks observed as part of the v1.6.0 release.
It will talk about what this means for an end-user's perf and scale story.

#### Background: How to interpret the numbers?

KubeVirt is an extension for Kubernetes that includes a collection of custom resource definitions
(CRDs) served by kubevirt-apiserver. These CRDs are managed by controllers. Due to the distributed
nature of the system, understanding performance and scalability data becomes challenging without
taking specific assumptions into account. This section aims to provide clarity on those assumptions.

1. The data presented in the document is collected from `periodic-kubevirt-e2e-k8s-1.29-sig-performance` and after December 5, 2024, data was collected from `periodic-kubevirt-e2e-k8s-1.31-sig-performance`
1. The test suite includes three tests:
   1. It creates 100 minimal VMIs, with a small pause of 100 ms between creation of 2 VMIs. The definition
      of minimal VMIs can be found [here](https://github.com/kubevirt/kubevirt/blob/20f6caaba4108733a2c3f216e3247202929c1ef9/tests/performance/density.go#L273).
      This is represented in the graphs as <Metric> for VMI, for example `vmiCreationToRunningSecondsP50` for VMI
   2. It creates 100 minimal VMs, with a small pause of 100 ms between creation of 2 VMIs. The definition
      of minimal VMs created can be found [here](https://github.com/kubevirt/kubevirt/blob/20f6caaba4108733a2c3f216e3247202929c1ef9/tests/performance/density.go#L219C1-L219C1).
      This is represented in the graphs as <Metric> for VM, for example `vmiCreationToRunningSecondsP50` for VM
   3. It creates VMs with instancetype and preference, the definition can be found [here](https://github.com/kubevirt/kubevirt/blob/20f6caaba4108733a2c3f216e3247202929c1ef9/tests/performance/density.go#L203).
      The benchmarks for this will be added in future releases.
1. The test waits for the VMIs to go into running state and collects a bunch of metrics
1. The collected metrics are categorized into two buckets, performance and scale
   1. Performance Metrics: This tells users how KubeVirt stack is performing. Examples include
      `vmiCreationToRunningSecondsP50` and `vmiCreationToRunningSecondsP95`. This helps users understand how KubeVirt 
       performance evolved over the releases; depending on the user deployment, the numbers will vary, because a real
       production workload could use other KubeVirt extension points like the device plugins, custom scheduler, 
       different version of kubelet etc. These numbers are just a guidance for how the KubeVirt codebase is performing 
       with minimal VMIs, provided all other variables(hardware, kubernetes version, cluster-size etc) remain the same.
   1. Scalability metrics: This helps users understand the KubeVirt scaling behaviors. Examples include, 
      `PATCH-pods-count` for VMI, `PATCH-virtualmachineinstances-count` for VMI and `UPDATE-virtualmachineinstances-count`
      for VMI. These metrics are measured on the client side to understand the load generated to apiserver by the 
      KubeVirt stack. This will help users and developers understand the cost of new features going into KubeVirt. It
      will also make end-users aware about the most expensive calls coming from KubeVirt in their deployment and 
      potentially act on it.  
1. The green dotted line in the graph is July 17, 2024, denoting release of v1.3.0.
1. The red dotted line in the graph is November 13, 2024, denoting release of v1.4.0.
1. The violet dotted line in the graph is December 5, 2024, denoting change in k8s provider from v1.29 to v1.31.
1. The grey dotted line in the graph is March 25, 2025, denoting release of v1.5.0


#### Performance benchmarks for v1.6.0 release

#### vmiCreationToRunningSecondsP50

![vmiCreationToRunningSecondsP50 for VMI](perf-scale-graphs/vmi/vmi-p50-Creation-to-Running.png "vmiCreationToRunningSecondsP50 for VMI")

![vmiCreationToRunningSecondsP50 for VM](perf-scale-graphs/vm/vm-p50-Creation-to-Running.png "vmiCreationToRunningSecondsP50 for VM")

#### vmiCreationToRunningSecondsP95

![vmiCreationToRunningSecondsP95 for VMI](perf-scale-graphs/vmi/vmi-p95-Creation-to-Running.png "vmiCreationToRunningSecondsP95 for VMI")

![vmiCreationToRunningSecondsP95 for VM](perf-scale-graphs/vm/vm-p95-Creation-to-Running.png "vmiCreationToRunningSecondsP95 for VM")

#### CPU and Memory usage of virt-api, virt-controller and virt-handler 

Note: virt-api and virt-controller memory usage is being collected from June 27th and virt-handler CPU and memory usage are being collected from December 6th

#### avgVirtAPICPUUsage

![avgVirtAPICPUUsage for VMI](perf-scale-graphs/vmi/vmi-avg-virt-api-cpu-usage.png "avgVirtAPICPUUsage for VMI")

![avgVirtAPICPUUsage for VM](perf-scale-graphs/vm/vm-avg-virt-api-cpu-usage.png "avgVirtAPICPUUsage for VM")

#### avgVirtControllerCPUUsage

![avgVirtControllerCPUUsage for VMI](perf-scale-graphs/vmi/vmi-avg-virt-controller-cpu-usage.png "avgVirtControlerCPUUsage for VMI")

![avgVirtControlerCPUUsage for VM](perf-scale-graphs/vm/vm-avg-virt-controller-cpu-usage.png "avgVirtControlerCPUUsage for VM")

#### avgVirtAPIMemoryUsageInMB

![avgVirtAPIMemoryUsageInMB for VMI](perf-scale-graphs/vmi/vmi-avg-virt-api-memory-usage.png "avgVirtAPIMemoryUsageInMB for VMI")

![avgVirtAPIMemoryUsageInMB for VM](perf-scale-graphs/vm/vm-avg-virt-api-memory-usage.png "avgVirtAPIMemoryUsageInMB for VM")

#### avgVirtControllerMemoryUsageInMB

![avgVirtAPIControllerMemoryUsageInMB for VMI](perf-scale-graphs/vmi/vmi-avg-virt-controller-memory-usage.png "avgVirtAPIControllerMemoryUsageInMB  for VMI")

![avgVirtAPIControllerMemoryUsageInMB for VM](perf-scale-graphs/vm/vm-avg-virt-controller-memory-usage.png "avgVirtAPIControllerMemoryUsageInMB  for VM")

#### avgVirtHandlerCPUUsage

![avgVirtHandlerCPUUsage for VMI](perf-scale-graphs/vmi/vmi-avg-virt-handler-cpu-usage.png "avgVirtHandlerCPUUsage for VMI")

![avgVirtHandlerCPUUsage for VM](perf-scale-graphs/vm/vm-avg-virt-handler-cpu-usage.png "avgVirtHandlerCPUUsage for VM")

#### avgVirtHandlerMemoryUsageInMB

![avgVirtHandlerMemoryUsageInMB for VMI](perf-scale-graphs/vmi/vmi-avg-virt-handler-memory-usage.png "avgVirtHandlerMemoryUsageInMB for VMI")

![avgVirtHandlerMemoryUsageInMB for VM](perf-scale-graphs/vm/vm-avg-virt-handler-memory-usage.png "avgVirtHandlerMemoryUsageInMB for VM")

#### Scalability benchmarks for v1.6.0 release

#### PATCH-pods-count

![PATCH-pods-count for VMI](perf-scale-graphs/vmi/vmi-patch-pods-count.png "PATCH-pods-count for VMI")

![PATCH-pods-count for VM](perf-scale-graphs/vm/vm-patch-pods-count.png "PATCH-pods-count for VM")

#### UPDATE-vmis-count

![UPDATE-vmis-count for VMI](perf-scale-graphs/vmi/vmi-update-vmis-count.png "UPDATE-vmis-count for VMI")

![UPDATE-vmis-count for VM](perf-scale-graphs/vm/vm-update-vmis-count.png "UPDATE-vmis-count for VM")

#### PATCH-vmis-count

![PATCH-vmis-count for VMI](perf-scale-graphs/vmi/vmi-patch-vmis-count.png "PATCH-vmis-count for VMI")

![PATCH-vmis-count for VM](perf-scale-graphs/vm/vm-patch-vmis-count.png "PATCH-vmis-count for VM")

#### PATCH-nodes-count

![PATCH-nodes-count for VMI](perf-scale-graphs/vmi/vmi-patch-nodes-count.png "PATCH-nodes-count for VMI")

![PATCH-nodes-count for VM](perf-scale-graphs/vm/vm-patch-nodes-count.png "PATCH-nodes-count for VM")

#### GET-nodes-count

![GET-nodes-count for VMI](perf-scale-graphs/vmi/vmi-get-nodes-count.png "GET-nodes-count for VMI")

![GET-nodes-count for VM](perf-scale-graphs/vm/vm-get-nodes-count.png "GET-nodes-count for VM")

#### Note :
The metrics we collected until October 2nd, 2024 had a bug where the "watch" metrics were being treated as list calls. This issue was fixed in this specific [PR](https://github.com/kubevirt/kubevirt/pull/12716), so the data collected up to October 1st is categorized under the list, and the data after that is under watch.

The bug was fixed on October 2nd 2024, and now the data is categorized correctly under the watch-virtualmachineinstances-count

#### WATCH-virtualmachineinstances-count

![WATCH-virtualmachineinstances-count for VM](perf-scale-graphs/vm/vm-watch-virtualmachineinstances-count.png "WATCH-virtualmachineinstances-count for VM")
