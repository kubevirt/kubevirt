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
#### HELP kubevirt_vmi_phase_count VMI phase.

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
#### HELP kubevirt_vmi_memory_resident_bytes resident set size of the process running the domain.

Total resident memory of the process running the VMI.

#### kubevirt_vmi_memory_available_bytes
#### HELP kubevirt_vmi_memory_available_bytes amount of usable memory as seen by the domain.

The total amount of usable memory.

#### kubevirt_vmi_memory_unused_bytes
#### HELP kubevirt_vmi_memory_unused_bytes amount of unused memory as seen by the domain.

The total amount of unused memory as seen by the domain.

#### kubevirt_vmi_memory_swap_traffic_bytes_total
#### HELP kubevirt_vmi_memory_swap_traffic_bytes_total swap memory traffic.

The amount of traffic that is being read and written in swap memory.

Extra labels:
* `type` - Whether the data is being transmitted or received. `in` when transmitting and `out` when receiving.

#### kubevirt_vmi_network_errors_total
#### HELP kubevirt_vmi_network_errors_total network errors.

Counter of network errors when transmitting and receiving data.

Extra labels:
* `interface` - Which network interface that errors are occurring.
* `type` - Whether the error occurred when transmitting or receiving data. `tx` when transmitting and `rx` when receiving.

#### kubevirt_vmi_network_traffic_bytes_total
#### HELP kubevirt_vmi_network_traffic_bytes_total network traffic.

The total amount of traffic that is being transmitted and received.

Extra labels:
* `interface` - Which network interface that errors are occurring.
* `type` - Whether the error occurred when transmitting or receiving data. `tx` when transmitting and `rx` when receiving.

#### kubevirt_vmi_network_traffic_packets_total
#### HELP kubevirt_vmi_network_traffic_packets_total network traffic packets.

The total amount of packets that are being transmitted and received.

Extra labels:
* `interface` - Which network interface that errors are occurring.
* `type` - Whether the error occurred when transmitting or receiving data. `tx` when transmitting and `rx` when receiving.

#### kubevirt_vmi_storage_iops_total
#### HELP kubevirt_vmi_storage_iops_total I/O operation performed.

Counter of read and write operations per disk device.

Extra labels:
* `drive` - Disk device that is being written/read.
* `type` - Whether it's a read or write operation.

#### kubevirt_vmi_storage_times_ms_total
#### HELP kubevirt_vmi_storage_times_ms_total storage operation time.

Total time spent on read and write operations per disk device.

Extra labels:
* `drive` - Disk device that is being written/read.
* `type` - Whether it's a read or write operation.

#### kubevirt_vmi_storage_traffic_bytes_total
#### HELP kubevirt_vmi_storage_traffic_bytes_total storage traffic.

The total amount of data read and written per disk device.

Extra labels:
* `drive` - Disk device that is being written/read.
* `type` - Whether it's a read or write operation.

#### kubevirt_vmi_vcpu_seconds
#### HELP kubevirt_vmi_vcpu_seconds Vcpu elapsed time.

The total amount of time spent in each vcpu state

Extra labels:
* `id` - Identifier to a single Virtual CPU.
* `state` - Identify the Virtual CPU state. It can be one of libvirt vcpu's states: `OFFLINE`, `RUNNING` or `BLOCKED`



## RoadMap

Improving Kubevirt's Observability is a important topic and we are currently working on new metrics.

A design proposal and its implementation history can be seen [here](https://docs.google.com/document/d/1bEwrnZZkVsCtz0PSyzlxOdhupL6GTurkUYcz7TXFM1g/edit)

 # Other Metrics
## kubevirt_vmi_vcpu_wait_seconds
#### HELP kubevirt_vmi_vcpu_wait_seconds vcpu time spent by waiting on I/O.
## leading_virt_controller
#### HELP leading_virt_controller Indication for an operating virt-controller.
## ready_virt_controller
#### HELP ready_virt_controller Indication for a virt-controller that is ready to take the lead.

 # Other Metrics
## kubevirt_vmi_outdated_count
#### HELP kubevirt_vmi_outdated_count Indication for the number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment.

 # Other Metrics
## kubevirt_vmi_network_receive_bytes_total
#### HELP kubevirt_vmi_network_receive_bytes_total Network traffic receive in bytes
## kubevirt_vmi_network_receive_errors_total
#### HELP kubevirt_vmi_network_receive_errors_total Network receive error packets
## kubevirt_vmi_network_receive_packets_dropped_total
#### HELP kubevirt_vmi_network_receive_packets_dropped_total The number of rx packets dropped on vNIC interfaces.
## kubevirt_vmi_network_receive_packets_total
#### HELP kubevirt_vmi_network_receive_packets_total Network traffic receive packets
## kubevirt_vmi_network_transmit_bytes_total
#### HELP kubevirt_vmi_network_transmit_bytes_total Network traffic transmit in bytes
## kubevirt_vmi_network_transmit_errors_total
#### HELP kubevirt_vmi_network_transmit_errors_total Network transmit error packets
## kubevirt_vmi_network_transmit_packets_dropped_total
#### HELP kubevirt_vmi_network_transmit_packets_dropped_total The number of tx packets dropped on vNIC interfaces.
## kubevirt_vmi_network_transmit_packets_total
#### HELP kubevirt_vmi_network_transmit_packets_total Network traffic transmit packets

 # Other Metrics
## kubevirt_vmi_memory_actual_balloon_bytes
#### HELP kubevirt_vmi_memory_actual_balloon_bytes current balloon bytes.
## kubevirt_vmi_memory_pgmajfault
#### HELP kubevirt_vmi_memory_pgmajfault The number of page faults when disk IO was required.
## kubevirt_vmi_memory_pgminfault
#### HELP kubevirt_vmi_memory_pgminfault The number of other page faults, when disk IO was not required.
## kubevirt_vmi_memory_swap_in_traffic_bytes_total
#### HELP kubevirt_vmi_memory_swap_in_traffic_bytes_total Swap in memory traffic in bytes
## kubevirt_vmi_memory_swap_out_traffic_bytes_total
#### HELP kubevirt_vmi_memory_swap_out_traffic_bytes_total Swap out memory traffic in bytes
## kubevirt_vmi_memory_usable_bytes
#### HELP kubevirt_vmi_memory_usable_bytes The amount of memory which can be reclaimed by balloon without causing host swapping in bytes.
## kubevirt_vmi_memory_used_total_bytes
#### HELP kubevirt_vmi_memory_used_total_bytes The amount of memory in bytes used by the domain.

 # Other Metrics
## kubevirt_vmi_cpu_affinity
#### HELP kubevirt_vmi_cpu_affinity vcpu affinity details

 # Other Metrics 
## kubevirt_virt_controller_leading_total
#### HELP kubevirt_virt_controller_leading Indication for an operating virt-controller.
## kubevirt_virt_controller_ready_total
#### HELP kubevirt_virt_controller_ready Indication for a virt-controller that is ready to take the lead.
## kubevirt_vmi_storage_flush_requests_total
#### HELP kubevirt_vmi_storage_flush_requests_total storage flush requests.
## kubevirt_vmi_storage_flush_times_ms_total
#### HELP kubevirt_vmi_storage_flush_times_ms_total total time (ms) spent on cache flushing.
## kubevirt_vmi_storage_iops_read_total
#### HELP kubevirt_vmi_storage_iops_read_total I/O read operations
## kubevirt_vmi_storage_iops_write_total
#### HELP kubevirt_vmi_storage_iops_write_total I/O write operations
## kubevirt_vmi_storage_read_times_ms_total
#### HELP kubevirt_vmi_storage_read_times_ms_total Storage read operation time
## kubevirt_vmi_storage_read_traffic_bytes_total
#### HELP kubevirt_vmi_storage_read_traffic_bytes_total Storage read traffic in bytes
## kubevirt_vmi_storage_write_times_ms_total
#### HELP kubevirt_vmi_storage_write_times_ms_total Storage write operation time
## kubevirt_vmi_storage_write_traffic_bytes_total
#### HELP kubevirt_vmi_storage_write_traffic_bytes_total Storage write traffic in bytes
