# KubeVirt metrics

### kubevirt_allocatable_nodes
The number of allocatable nodes in the cluster. Type: Gauge.

### kubevirt_api_request_deprecated_total
The total number of requests to deprecated KubeVirt APIs. Type: Counter.

### kubevirt_configuration_emulation_enabled
Indicates whether the Software Emulation is enabled in the configuration. Type: Gauge.

### kubevirt_console_active_connections
Amount of active Console connections, broken down by namespace and vmi name. Type: Gauge.

### kubevirt_info
Version information. Type: Gauge.

### kubevirt_memory_delta_from_requested_bytes
The delta between the pod with highest memory working set or rss and its requested memory for each container, virt-controller, virt-handler, virt-api and virt-operator. Type: Gauge.

### kubevirt_node_deprecated_machine_types
List of deprecated machine types based on the capabilities of individual nodes, as detected by virt-handler. Type: Gauge.

### kubevirt_nodes_with_kvm
The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available. Type: Gauge.

### kubevirt_number_of_vms
The number of VMs in the cluster by namespace. Type: Gauge.

### kubevirt_portforward_active_tunnels
Amount of active portforward tunnels, broken down by namespace and vmi name. Type: Gauge.

### kubevirt_rest_client_rate_limiter_duration_seconds
Client side rate limiter latency in seconds. Broken down by verb and URL. Type: Histogram.

### kubevirt_rest_client_request_latency_seconds
Request latency in seconds. Broken down by verb and URL. Type: Histogram.

### kubevirt_rest_client_requests_total
Number of HTTP requests, partitioned by status code, method, and host. Type: Counter.

### kubevirt_usbredir_active_connections
Amount of active USB redirection connections, broken down by namespace and vmi name. Type: Gauge.

### kubevirt_virt_api_up
The number of virt-api pods that are up. Type: Gauge.

### kubevirt_virt_controller_leading_status
Indication for an operating virt-controller. Type: Gauge.

### kubevirt_virt_controller_ready
The number of virt-controller pods that are ready. Type: Gauge.

### kubevirt_virt_controller_ready_status
Indication for a virt-controller that is ready to take the lead. Type: Gauge.

### kubevirt_virt_controller_up
The number of virt-controller pods that are up. Type: Gauge.

### kubevirt_virt_handler_up
The number of virt-handler pods that are up. Type: Gauge.

### kubevirt_virt_operator_leading
The number of virt-operator pods that are leading. Type: Gauge.

### kubevirt_virt_operator_leading_status
Indication for an operating virt-operator. Type: Gauge.

### kubevirt_virt_operator_ready
The number of virt-operator pods that are ready. Type: Gauge.

### kubevirt_virt_operator_ready_status
Indication for a virt-operator that is ready to take the lead. Type: Gauge.

### kubevirt_virt_operator_up
The number of virt-operator pods that are up. Type: Gauge.

### kubevirt_vm_container_free_memory_bytes_based_on_rss
The current available memory of the VM containers based on the rss. Type: Gauge.

### kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes
The current available memory of the VM containers based on the working set. Type: Gauge.

### kubevirt_vm_create_date_timestamp_seconds
Virtual Machine creation timestamp. Type: Gauge.

### kubevirt_vm_created_by_pod_total
The total number of VMs created by namespace and virt-api pod, since install. Type: Counter.

### kubevirt_vm_created_total
The total number of VMs created by namespace, since install. Type: Counter.

### kubevirt_vm_disk_allocated_size_bytes
Allocated disk size of a Virtual Machine in bytes, based on its PersistentVolumeClaim. Includes persistentvolumeclaim (PVC name), volume_mode (disk presentation mode: Filesystem or Block), and device (disk name). Type: Gauge.

### kubevirt_vm_error_status_last_transition_timestamp_seconds
Virtual Machine last transition timestamp to error status. Type: Counter.

### kubevirt_vm_info
Information about Virtual Machines. Type: Gauge.

### kubevirt_vm_migrating_status_last_transition_timestamp_seconds
Virtual Machine last transition timestamp to migrating status. Type: Counter.

### kubevirt_vm_non_running_status_last_transition_timestamp_seconds
Virtual Machine last transition timestamp to paused/stopped status. Type: Counter.

### kubevirt_vm_resource_limits
Resources limits by Virtual Machine. Reports memory and CPU limits. Type: Gauge.

### kubevirt_vm_resource_requests
Resources requested by Virtual Machine. Reports memory and CPU requests. Type: Gauge.

### kubevirt_vm_running_status_last_transition_timestamp_seconds
Virtual Machine last transition timestamp to running status. Type: Counter.

### kubevirt_vm_starting_status_last_transition_timestamp_seconds
Virtual Machine last transition timestamp to starting status. Type: Counter.

### kubevirt_vm_vnic_info
Details of Virtual Machine (VM) vNIC interfaces, such as vNIC name, binding type, network name, and binding name for each vNIC defined in the VM's configuration. Type: Gauge.

### kubevirt_vmi_cpu_system_usage_seconds_total
Total CPU time spent in system mode. Type: Counter.

### kubevirt_vmi_cpu_usage_seconds_total
Total CPU time spent in all modes (sum of both vcpu and hypervisor usage). Type: Counter.

### kubevirt_vmi_cpu_user_usage_seconds_total
Total CPU time spent in user mode. Type: Counter.

### kubevirt_vmi_dirty_rate_bytes_per_second
Guest dirty-rate in bytes per second. Type: Gauge.

### kubevirt_vmi_filesystem_capacity_bytes
Total VM filesystem capacity in bytes. Type: Gauge.

### kubevirt_vmi_filesystem_used_bytes
Used VM filesystem capacity in bytes. Type: Gauge.

### kubevirt_vmi_guest_load_15m
Guest system load average over 15 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Type: Gauge.

### kubevirt_vmi_guest_load_1m
Guest system load average over 1 minute as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Type: Gauge.

### kubevirt_vmi_guest_load_5m
Guest system load average over 5 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Type: Gauge.

### kubevirt_vmi_guest_vcpu_queue
Guest queue length. Type: Gauge.

### kubevirt_vmi_info
Information about VirtualMachineInstances. Type: Gauge.

### kubevirt_vmi_last_api_connection_timestamp_seconds
Virtual Machine Instance last API connection timestamp. Including VNC, console, portforward, SSH and usbredir connections. Type: Gauge.

### kubevirt_vmi_launcher_memory_overhead_bytes
Estimation of the memory amount required for virt-launcher's infrastructure components (e.g. libvirt, QEMU). Type: Gauge.

### kubevirt_vmi_memory_actual_balloon_bytes
Current balloon size in bytes. Type: Gauge.

### kubevirt_vmi_memory_available_bytes
Amount of usable memory as seen by the domain. This value may not be accurate if a balloon driver is in use or if the guest OS does not initialize all assigned pages Type: Gauge.

### kubevirt_vmi_memory_cached_bytes
The amount of memory that is being used to cache I/O and is available to be reclaimed, corresponds to the sum of `Buffers` + `Cached` + `SwapCached` in `/proc/meminfo`. Type: Gauge.

### kubevirt_vmi_memory_domain_bytes
The amount of memory in bytes allocated to the domain. The `memory` value in domain xml file. Type: Gauge.

### kubevirt_vmi_memory_pgmajfault_total
The number of page faults when disk IO was required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is required, it is considered as major fault. Type: Counter.

### kubevirt_vmi_memory_pgminfault_total
The number of other page faults, when disk IO was not required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is NOT required, it is considered as minor fault. Type: Counter.

### kubevirt_vmi_memory_resident_bytes
Resident set size of the process running the domain. Type: Gauge.

### kubevirt_vmi_memory_swap_in_traffic_bytes
The total amount of data read from swap space of the guest in bytes. Type: Gauge.

### kubevirt_vmi_memory_swap_out_traffic_bytes
The total amount of memory written out to swap space of the guest in bytes. Type: Gauge.

### kubevirt_vmi_memory_unused_bytes
The amount of memory left completely unused by the system. Memory that is available but used for reclaimable caches should NOT be reported as free. Type: Gauge.

### kubevirt_vmi_memory_usable_bytes
The amount of memory which can be reclaimed by balloon without pushing the guest system to swap, corresponds to 'Available' in /proc/meminfo. Type: Gauge.

### kubevirt_vmi_memory_used_bytes
Amount of `used` memory as seen by the domain. Type: Gauge.

### kubevirt_vmi_migration_data_processed_bytes
The total Guest OS data processed and migrated to the new VM. Type: Gauge.

### kubevirt_vmi_migration_data_remaining_bytes
The remaining guest OS data to be migrated to the new VM. Type: Gauge.

### kubevirt_vmi_migration_data_total_bytes
The total Guest OS data to be migrated to the new VM. Type: Counter.

### kubevirt_vmi_migration_dirty_memory_rate_bytes
The rate of memory being dirty in the Guest OS. Type: Gauge.

### kubevirt_vmi_migration_disk_transfer_rate_bytes
The rate at which the memory is being transferred. Type: Gauge.

### kubevirt_vmi_migration_end_time_seconds
The time at which the migration ended. Type: Gauge.

### kubevirt_vmi_migration_failed
Indicates if the VMI migration failed. Type: Gauge.

### kubevirt_vmi_migration_phase_transition_time_from_creation_seconds
Histogram of VM migration phase transitions duration from creation time in seconds. Type: Histogram.

### kubevirt_vmi_migration_start_time_seconds
The time at which the migration started. Type: Gauge.

### kubevirt_vmi_migration_succeeded
Indicates if the VMI migration succeeded. Type: Gauge.

### kubevirt_vmi_migrations_in_pending_phase
Number of current pending migrations. Type: Gauge.

### kubevirt_vmi_migrations_in_running_phase
Number of current running migrations. Type: Gauge.

### kubevirt_vmi_migrations_in_scheduling_phase
Number of current scheduling migrations. Type: Gauge.

### kubevirt_vmi_migrations_in_unset_phase
Number of current unset migrations. These are pending items the virt-controller hasn’t processed yet from the queue. Type: Gauge.

### kubevirt_vmi_network_receive_bytes_total
Total network traffic received in bytes. Type: Counter.

### kubevirt_vmi_network_receive_errors_total
Total network received error packets. Type: Counter.

### kubevirt_vmi_network_receive_packets_dropped_total
The total number of rx packets dropped on vNIC interfaces. Type: Counter.

### kubevirt_vmi_network_receive_packets_total
Total network traffic received packets. Type: Counter.

### kubevirt_vmi_network_traffic_bytes_total
[Deprecated] Total number of bytes sent and received. Type: Counter.

### kubevirt_vmi_network_transmit_bytes_total
Total network traffic transmitted in bytes. Type: Counter.

### kubevirt_vmi_network_transmit_errors_total
Total network transmitted error packets. Type: Counter.

### kubevirt_vmi_network_transmit_packets_dropped_total
The total number of tx packets dropped on vNIC interfaces. Type: Counter.

### kubevirt_vmi_network_transmit_packets_total
Total network traffic transmitted packets. Type: Counter.

### kubevirt_vmi_node_cpu_affinity
Number of VMI CPU affinities to node physical cores. Type: Gauge.

### kubevirt_vmi_non_evictable
Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable. Type: Gauge.

### kubevirt_vmi_number_of_outdated
Indication for the total number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment. Type: Gauge.

### kubevirt_vmi_phase_count
Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`]. Type: Gauge.

### kubevirt_vmi_phase_transition_time_from_creation_seconds
Histogram of VM phase transitions duration from creation time in seconds. Type: Histogram.

### kubevirt_vmi_phase_transition_time_from_deletion_seconds
Histogram of VM phase transitions duration from deletion time in seconds. Type: Histogram.

### kubevirt_vmi_phase_transition_time_seconds
Histogram of VM phase transitions duration between different phases in seconds. Type: Histogram.

### kubevirt_vmi_status_addresses
The addresses of a VirtualMachineInstance. This metric provides the address of an available network interface associated with the VMI in the 'address' label, and about the type of address, such as internal IP, in the 'type' label. Type: Gauge.

### kubevirt_vmi_storage_flush_requests_total
Total storage flush requests. Type: Counter.

### kubevirt_vmi_storage_flush_times_seconds_total
Total time spent on cache flushing. Type: Counter.

### kubevirt_vmi_storage_iops_read_total
Total number of I/O read operations. Type: Counter.

### kubevirt_vmi_storage_iops_write_total
Total number of I/O write operations. Type: Counter.

### kubevirt_vmi_storage_read_times_seconds_total
Total time spent on read operations. Type: Counter.

### kubevirt_vmi_storage_read_traffic_bytes_total
Total number of bytes read from storage. Type: Counter.

### kubevirt_vmi_storage_write_times_seconds_total
Total time spent on write operations. Type: Counter.

### kubevirt_vmi_storage_write_traffic_bytes_total
Total number of written bytes. Type: Counter.

### kubevirt_vmi_vcpu_count
The number of the VMI vCPUs. Type: Gauge.

### kubevirt_vmi_vcpu_delay_seconds_total
Amount of time spent by each vcpu waiting in the queue instead of running. Type: Counter.

### kubevirt_vmi_vcpu_seconds_total
Total amount of time spent in each state by each vcpu (cpu_time excluding hypervisor time). Where `id` is the vcpu identifier and `state` can be one of the following: [`OFFLINE`, `RUNNING`, `BLOCKED`]. Type: Counter.

### kubevirt_vmi_vcpu_wait_seconds_total
Amount of time spent by each vcpu while waiting on I/O. Type: Counter.

### kubevirt_vmi_vnic_info
Details of VirtualMachineInstance (VMI) vNIC interfaces, such as vNIC name, binding type, network name, and binding name for each vNIC of a running instance. Type: Gauge.

### kubevirt_vmsnapshot_disks_restored_from_source
Returns the total number of virtual machine disks restored from the source virtual machine. Type: Gauge.

### kubevirt_vmsnapshot_disks_restored_from_source_bytes
Returns the amount of space in bytes restored from the source virtual machine. Type: Gauge.

### kubevirt_vmsnapshot_persistentvolumeclaim_labels
Returns the labels of the persistent volume claims that are used for restoring virtual machines. Type: Gauge.

### kubevirt_vmsnapshot_succeeded_timestamp_seconds
Returns the timestamp of successful virtual machine snapshot. Type: Gauge.

### kubevirt_vnc_active_connections
Amount of active VNC connections, broken down by namespace and vmi name. Type: Gauge.

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
