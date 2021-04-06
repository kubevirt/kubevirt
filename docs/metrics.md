# Exposed Metrics

Sometimes the Help text on `/metrics` endpoint just isn't enough to explain what a certain metric means. This document's objective is to give further explanation of KubeVirt related metrics.

## Kubevirt Metric

#### kubevirt_info

Kubevirt's version information

Labels:
* `goversion` - GO version used to compile this version of KubeVirt
* `kubeversion` - Git commit refspec that created this version of KubeVirt

## Node Metric

#### kubevirt_vmi_phase_count

The total amount of VMIs per node and phase.

Labels:
* `phase` - Phase of the VMI. It can be one of [Virtual Machine Instance Phases](https://github.com/kubevirt/kubevirt/blob/master/staging/src/kubevirt.io/client-go/api/v1/types.go#L415)
* `node` - Node where the VMI is running on.

## VMI Metrics

All VMI metrics listed below contain, but are not limited to, these three labels for identifying purposes:

* `name` - VMI's name given on its specification.
* `namespace` - Namespace which the given VMI is related to.
* `node` - Node where the VMI is running on.

#### kubevirt_vmi_memory_resident_bytes

Total resident memory of the process running the VMI.

#### kubevirt_vmi_memory_available_bytes

The total amount of usable memory.

#### kubevirt_vmi_memory_used_total_bytes

The amount of memory in bytes used by the domain.

#### kubevirt_vmi_memory_actual_balloon_bytes

The current balloon bytes.

#### kubevirt_vmi_memory_pgmajfault

The number of page faults when disk IO was required.

#### kubevirt_vmi_memory_pgminfault

The number of other page faults, when disk IO was not required.

#### kubevirt_vmi_memory_usable_bytes

The amount of memory which can be reclaimed by balloon without causing host swapping in bytes.

#### kubevirt_vmi_memory_unused_bytes

The total amount of unused memory as seen by the domain.

#### kubevirt_vmi_memory_swap_in_traffic_bytes_total

The amount of traffic that is brought back into memory from the swap memory.

#### kubevirt_vmi_memory_swap_out_traffic_bytes_total

The amount of traffic that is sent to the swap memory from the memory.

#### kubevirt_vmi_network_receive_errors_total

The counter of network errors when receiving data.

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_transmit_errors_total

The counter of network errors when transmitting data.

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_receive_bytes_total

The network traffic received in bytes

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_transmit_bytes_total

The network traffic transmitted in bytes

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_receive_packets_total

The total amount of packets that were received.

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_transmit_packets_total

The total amount of packets that were transmitted.

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_receive_packets_dropped_total

The number of rx packets dropped on vNIC interfaces.

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_network_transmit_packets_dropped_total

The number of tx packets dropped on vNIC interfaces.

Extra labels:
* `interface` - Which network interface that errors are occurring.

#### kubevirt_vmi_storage_iops_read_total

The total count of I/O read operations per disk device.

Extra labels:
* `drive` - Disk device that is being written/read.

#### kubevirt_vmi_storage_iops_write_total

The total count of I/O write operations per disk device.

Extra labels:
* `drive` - Disk device that is being written/read.

#### kubevirt_vmi_storage_read_times_ms_total

Total time spent on read operations per disk device in milliseconds.

Extra labels:
* `drive` - Disk device that is being written/read.

#### kubevirt_vmi_storage_write_times_ms_total

Total time spent on write operations per disk device in milliseconds.

Extra labels:
* `drive` - Disk device that is being written/read.

#### kubevirt_vmi_storage_read_traffic_bytes_total

The total amount of data read per disk device in bytes.

Extra labels:
* `drive` - Disk device that is being written/read.

#### kubevirt_vmi_storage_write_traffic_bytes_total

The total amount of data written per disk device in bytes.

Extra labels:
* `drive` - Disk device that is being written/read.

#### kubevirt_vmi_storage_flush_requests_total

The total count of the storage flush requests.

#### kubevirt_vmi_storage_flush_times_ms_total

The total time (ms) spent on cache flushing.

#### kubevirt_vmi_vcpu_seconds

The total amount of time spent in each vcpu state

Extra labels:
* `id` - Identifier to a single Virtual CPU.
* `state` - Identify the Virtual CPU state. It can be one of libvirt vcpu's states: `OFFLINE`, `RUNNING` or `BLOCKED`

#### kubevirt_vmi_vcpu_wait_seconds

The vcpu time spent by waiting on I/O.

#### kubevirt_vmi_outdated_count

The indication for the number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment.

#### kubevirt_vmi_cpu_affinity

The vcpu affinity details

#### leading_virt_controller

The indication for an operating virt-controller.

#### ready_virt_controller

The indication for a virt-controller that is ready to take the lead.


## RoadMap

Improving Kubevirt's Observability is a important topic and we are currently working on new metrics.

A design proposal and its implementation history can be seen [here](https://docs.google.com/document/d/1bEwrnZZkVsCtz0PSyzlxOdhupL6GTurkUYcz7TXFM1g/edit)
