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



## kubevirt_vmi_cpu_system_seconds_total
#### HELP kubevirt_vmi_cpu_system_seconds_total system cpu time spent in seconds.
The system cpu time spent (seconds)
## kubevirt_vmi_cpu_usage_seconds_total
#### HELP kubevirt_vmi_cpu_usage_seconds_total total cpu time spent for this domain in seconds.
The total cpu time spent (seconds)
## kubevirt_vmi_cpu_user_seconds_total
#### HELP kubevirt_vmi_cpu_user_seconds_total user cpu time spent in seconds.
The user cpu time spent (seconds)
## kubevirt_vmi_memory_actual_balloon_bytes
#### HELP kubevirt_vmi_memory_actual_balloon_bytes current balloon bytes.
The current balloon value (bytes)
## kubevirt_vmi_memory_available_bytes
#### HELP kubevirt_vmi_memory_available_bytes amount of usable memory as seen by the domain.
The amount of usable memory (bytes)
## kubevirt_vmi_memory_resident_bytes
#### HELP kubevirt_vmi_memory_resident_bytes resident set size of the process running the domain.
Resident Set Size of the process running the VMI (bytes)
## kubevirt_vmi_memory_swap_in_traffic_bytes_total
#### HELP kubevirt_vmi_memory_swap_in_traffic_bytes_total Swap in memory traffic in bytes
The total amount of data read from swap space (bytes)
## kubevirt_vmi_memory_unused_bytes
#### HELP kubevirt_vmi_memory_unused_bytes amount of unused memory as seen by the domain.
The amount of memory left completely unused by the system (bytes)
## kubevirt_vmi_memory_used_total_bytes
#### HELP kubevirt_vmi_memory_used_total_bytes The amount of memory in bytes used by the domain.
The memory used by the VMI (bytes)## kubevirt_vmi_network_receive_bytes_total
#### HELP kubevirt_vmi_network_receive_bytes_total Network traffic receive in bytes
The amount of traffic received per interface (bytes)
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_receive_errors_total
#### HELP kubevirt_vmi_network_receive_errors_total Network receive error packets
Counter of network errors when receiving data
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_receive_packets_dropped_total
#### HELP kubevirt_vmi_network_receive_packets_dropped_total The number of rx packets dropped on vNIC interfaces.
Counter of dropped network packets when receiving data
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_receive_packets_total
#### HELP kubevirt_vmi_network_receive_packets_total Network traffic receive packets
Counter of received network packets
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_transmit_bytes_total
#### HELP kubevirt_vmi_network_transmit_bytes_total Network traffic transmit in bytes
The amount of traffic transmitted per interface (bytes)
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_transmit_errors_total
#### HELP kubevirt_vmi_network_transmit_errors_total Network transmit error packets
Counter of network errors when transmitting data
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_transmit_packets_dropped_total
#### HELP kubevirt_vmi_network_transmit_packets_dropped_total The number of tx packets dropped on vNIC interfaces.
Counter of dropped network packets when transmitting data
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_network_transmit_packets_total
#### HELP kubevirt_vmi_network_transmit_packets_total Network traffic transmit packets
Counter of transmitting network packets
Extra labels:
* `interface` - Network interface name
## kubevirt_vmi_storage_iops_read_total
#### HELP kubevirt_vmi_storage_iops_read_total I/O read operations
The number of read requests
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_iops_write_total
#### HELP kubevirt_vmi_storage_iops_write_total I/O write operations
The number of write requests
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_read_times_ms_total
#### HELP kubevirt_vmi_storage_read_times_ms_total Storage read operation time
The total time spend on cache reads of the block device (ms)
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_read_traffic_bytes_total
#### HELP kubevirt_vmi_storage_read_traffic_bytes_total Storage read traffic in bytes
The total number of read bytes of the block device
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_flush_times_ms_total
#### HELP kubevirt_vmi_storage_flush_times_ms_total total time (ms) spent on cache flushing.
The total time spent on block cache flushes
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_flush_requests_total
#### HELP kubevirt_vmi_storage_flush_requests_total storage flush requests.
The total flush requests of the block device
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_write_times_ms_total
#### HELP kubevirt_vmi_storage_write_times_ms_total Storage write operation time
The total time spend on cache writes of the block device (ms)
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_storage_write_traffic_bytes_total
#### HELP kubevirt_vmi_storage_write_traffic_bytes_total Storage write traffic in bytes
The total number of write bytes of the block device
Extra labels:
* `drive` - Disk device name
## kubevirt_vmi_vcpu_seconds
#### HELP kubevirt_vmi_vcpu_seconds Vcpu elapsed time.
Virtual cpu time spent by virtual CPU (seconds)
Extra labels:
* `id` - Id of the virtual CPU
* `state` - State of the virtual CPU
## kubevirt_vmi_vcpu_wait_seconds
#### HELP kubevirt_vmi_vcpu_wait_seconds vcpu time spent by waiting on I/O.
Time the virtual CPU wants to run, but the host scheduler has something else running
Extra labels:
* `id` - Id of the virtual CPU
 # RoadMap
Improving Kubevirt's Observability is a important topic and we are currently working on new metrics.

A design proposal and its implementation history can be seen [here](https://docs.google.com/document/d/1bEwrnZZkVsCtz0PSyzlxOdhupL6GTurkUYcz7TXFM1g/edit)

 # Other Metrics 
## kubevirt_vmi_network_traffic_bytes_total
#### HELP kubevirt_vmi_network_traffic_bytes_total network traffic.
## leading_virt_controller
#### HELP leading_virt_controller Indication for an operating virt-controller.
## ready_virt_controller
#### HELP ready_virt_controller Indication for a virt-controller that is ready to take the lead.
