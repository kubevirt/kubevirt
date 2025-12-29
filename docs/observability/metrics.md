# KubeVirt metrics

| Name | Kind | Type | Description |
|------|------|------|-------------|
| kubevirt_configuration_emulation_enabled | Metric | Gauge | Indicates whether the Software Emulation is enabled in the configuration. |
| kubevirt_console_active_connections | Metric | Gauge | Amount of active Console connections, broken down by namespace and vmi name. |
| kubevirt_info | Metric | Gauge | Version information. |
| kubevirt_node_deprecated_machine_types | Metric | Gauge | List of deprecated machine types based on the capabilities of individual nodes, as detected by virt-handler. |
| kubevirt_portforward_active_tunnels | Metric | Gauge | Amount of active portforward tunnels, broken down by namespace and vmi name. |
| kubevirt_rest_client_rate_limiter_duration_seconds | Metric | Histogram | Client side rate limiter latency in seconds. Broken down by verb and URL. |
| kubevirt_rest_client_request_latency_seconds | Metric | Histogram | Request latency in seconds. Broken down by verb and URL. |
| kubevirt_rest_client_requests_total | Metric | Counter | Number of HTTP requests, partitioned by status code, method, and host. |
| kubevirt_usbredir_active_connections | Metric | Gauge | Amount of active USB redirection connections, broken down by namespace and vmi name. |
| kubevirt_virt_controller_leading_status | Metric | Gauge | Indication for an operating virt-controller. |
| kubevirt_virt_controller_ready_status | Metric | Gauge | Indication for a virt-controller that is ready to take the lead. |
| kubevirt_virt_operator_leading_status | Metric | Gauge | Indication for an operating virt-operator. |
| kubevirt_virt_operator_ready_status | Metric | Gauge | Indication for a virt-operator that is ready to take the lead. |
| kubevirt_vm_create_date_timestamp_seconds | Metric | Gauge | Virtual Machine creation timestamp. |
| kubevirt_vm_created_by_pod_total | Metric | Counter | The total number of VMs created by namespace and virt-api pod, since install. |
| kubevirt_vm_disk_allocated_size_bytes | Metric | Gauge | Allocated disk size of a Virtual Machine in bytes, based on its PersistentVolumeClaim. Includes persistentvolumeclaim (PVC name), volume_mode (disk presentation mode: Filesystem or Block), and device (disk name). |
| kubevirt_vm_error_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to error status. |
| kubevirt_vm_info | Metric | Gauge | Information about Virtual Machines. |
| kubevirt_vm_labels | Metric | Gauge | The metric exposes the VM labels as Prometheus labels. Configure allowed and ignored labels via the 'kubevirt-vm-labels-config' ConfigMap. |
| kubevirt_vm_migrating_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to migrating status. |
| kubevirt_vm_non_running_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to paused/stopped status. |
| kubevirt_vm_resource_limits | Metric | Gauge | Resources limits by Virtual Machine. Reports memory and CPU limits. |
| kubevirt_vm_resource_requests | Metric | Gauge | Resources requested by Virtual Machine. Reports memory and CPU requests. |
| kubevirt_vm_running_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to running status. |
| kubevirt_vm_starting_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to starting status. |
| kubevirt_vm_vnic_info | Metric | Gauge | Details of Virtual Machine (VM) vNIC interfaces, such as vNIC name, binding type, network name, and binding name for each vNIC defined in the VM's configuration. |
| kubevirt_vmi_contains_ephemeral_hotplug_volume | Metric | Gauge | Reported only for VMIs that contain an ephemeral hotplug volume. |
| kubevirt_vmi_cpu_system_usage_seconds_total | Metric | Counter | Total CPU time spent in system mode. |
| kubevirt_vmi_cpu_usage_seconds_total | Metric | Counter | Total CPU time spent in all modes (sum of both vcpu and hypervisor usage). |
| kubevirt_vmi_cpu_user_usage_seconds_total | Metric | Counter | Total CPU time spent in user mode. |
| kubevirt_vmi_dirty_rate_bytes_per_second | Metric | Gauge | Guest dirty-rate in bytes per second. |
| kubevirt_vmi_filesystem_capacity_bytes | Metric | Gauge | Total VM filesystem capacity in bytes. |
| kubevirt_vmi_filesystem_used_bytes | Metric | Gauge | Used VM filesystem capacity in bytes. |
| kubevirt_vmi_guest_load_15m | Metric | Gauge | Guest system load average over 15 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Requires qemu-guest-agent version 10.0.0 or above. |
| kubevirt_vmi_guest_load_1m | Metric | Gauge | Guest system load average over 1 minute as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Requires qemu-guest-agent version 10.0.0 or above. |
| kubevirt_vmi_guest_load_5m | Metric | Gauge | Guest system load average over 5 minutes as reported by the guest agent. Load is defined as the number of processes in the runqueue or waiting for disk I/O. Requires qemu-guest-agent version 10.0.0 or above. |
| kubevirt_vmi_info | Metric | Gauge | Information about VirtualMachineInstances. |
| kubevirt_vmi_last_api_connection_timestamp_seconds | Metric | Gauge | Virtual Machine Instance last API connection timestamp. Including VNC, console, portforward, SSH and usbredir connections. |
| kubevirt_vmi_launcher_memory_overhead_bytes | Metric | Gauge | Estimation of the memory amount required for virt-launcher's infrastructure components (e.g. libvirt, QEMU). |
| kubevirt_vmi_memory_actual_balloon_bytes | Metric | Gauge | Current balloon size in bytes. |
| kubevirt_vmi_memory_available_bytes | Metric | Gauge | Amount of usable memory as seen by the domain. This value may not be accurate if a balloon driver is in use or if the guest OS does not initialize all assigned pages |
| kubevirt_vmi_memory_cached_bytes | Metric | Gauge | The amount of memory that is being used to cache I/O and is available to be reclaimed, corresponds to the sum of `Buffers` + `Cached` + `SwapCached` in `/proc/meminfo`. |
| kubevirt_vmi_memory_domain_bytes | Metric | Gauge | The amount of memory in bytes allocated to the domain. The `memory` value in domain xml file. |
| kubevirt_vmi_memory_pgmajfault_total | Metric | Counter | The number of page faults when disk IO was required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is required, it is considered as major fault. |
| kubevirt_vmi_memory_pgminfault_total | Metric | Counter | The number of other page faults, when disk IO was not required. Page faults occur when a process makes a valid access to virtual memory that is not available. When servicing the page fault, if disk IO is NOT required, it is considered as minor fault. |
| kubevirt_vmi_memory_resident_bytes | Metric | Gauge | Resident set size of the process running the domain. |
| kubevirt_vmi_memory_swap_in_traffic_bytes | Metric | Gauge | The total amount of data read from swap space of the guest in bytes. |
| kubevirt_vmi_memory_swap_out_traffic_bytes | Metric | Gauge | The total amount of memory written out to swap space of the guest in bytes. |
| kubevirt_vmi_memory_unused_bytes | Metric | Gauge | The amount of memory left completely unused by the system. Memory that is available but used for reclaimable caches should NOT be reported as free. |
| kubevirt_vmi_memory_usable_bytes | Metric | Gauge | The amount of memory which can be reclaimed by balloon without pushing the guest system to swap, corresponds to 'Available' in /proc/meminfo. |
| kubevirt_vmi_migration_data_bytes_total | Metric | Counter | The total Guest OS data to be migrated to the new VM. |
| kubevirt_vmi_migration_data_processed_bytes | Metric | Gauge | The total Guest OS data processed and migrated to the new VM. |
| kubevirt_vmi_migration_data_remaining_bytes | Metric | Gauge | The remaining guest OS data to be migrated to the new VM. |
| kubevirt_vmi_migration_dirty_memory_rate_bytes | Metric | Gauge | The rate of memory being dirty in the Guest OS. |
| kubevirt_vmi_migration_end_time_seconds | Metric | Gauge | The time at which the migration ended. |
| kubevirt_vmi_migration_failed | Metric | Gauge | Indicates if the VMI migration failed. |
| kubevirt_vmi_migration_memory_transfer_rate_bytes | Metric | Gauge | The rate at which the memory is being transferred. |
| kubevirt_vmi_migration_phase_transition_time_from_creation_seconds | Metric | Histogram | Histogram of VM migration phase transitions duration from creation time in seconds. |
| kubevirt_vmi_migration_start_time_seconds | Metric | Gauge | The time at which the migration started. |
| kubevirt_vmi_migration_succeeded | Metric | Gauge | Indicates if the VMI migration succeeded. |
| kubevirt_vmi_migrations_in_pending_phase | Metric | Gauge | Number of current pending migrations. |
| kubevirt_vmi_migrations_in_running_phase | Metric | Gauge | Number of current running migrations. |
| kubevirt_vmi_migrations_in_scheduling_phase | Metric | Gauge | Number of current scheduling migrations. |
| kubevirt_vmi_migrations_in_unset_phase | Metric | Gauge | Number of current unset migrations. These are pending items the virt-controller hasn’t processed yet from the queue. |
| kubevirt_vmi_network_receive_bytes_total | Metric | Counter | Total network traffic received in bytes. |
| kubevirt_vmi_network_receive_errors_total | Metric | Counter | Total network received error packets. |
| kubevirt_vmi_network_receive_packets_dropped_total | Metric | Counter | The total number of rx packets dropped on vNIC interfaces. |
| kubevirt_vmi_network_receive_packets_total | Metric | Counter | Total network traffic received packets. |
| kubevirt_vmi_network_traffic_bytes_total | Metric | Counter | [Deprecated] Total number of bytes sent and received. |
| kubevirt_vmi_network_transmit_bytes_total | Metric | Counter | Total network traffic transmitted in bytes. |
| kubevirt_vmi_network_transmit_errors_total | Metric | Counter | Total network transmitted error packets. |
| kubevirt_vmi_network_transmit_packets_dropped_total | Metric | Counter | The total number of tx packets dropped on vNIC interfaces. |
| kubevirt_vmi_network_transmit_packets_total | Metric | Counter | Total network traffic transmitted packets. |
| kubevirt_vmi_node_cpu_affinity | Metric | Gauge | Number of VMI CPU affinities to node physical cores. |
| kubevirt_vmi_non_evictable | Metric | Gauge | Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable. |
| kubevirt_vmi_number_of_outdated | Metric | Gauge | Indication for the total number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment. |
| kubevirt_vmi_phase_transition_time_from_creation_seconds | Metric | Histogram | Histogram of VM phase transitions duration from creation time in seconds. |
| kubevirt_vmi_phase_transition_time_from_deletion_seconds | Metric | Histogram | Histogram of VM phase transitions duration from deletion time in seconds. |
| kubevirt_vmi_phase_transition_time_seconds | Metric | Histogram | Histogram of VM phase transitions duration between different phases in seconds. |
| kubevirt_vmi_status_addresses | Metric | Gauge | The addresses of a VirtualMachineInstance. This metric provides the address of an available network interface associated with the VMI in the 'address' label, and about the type of address, such as internal IP, in the 'type' label. |
| kubevirt_vmi_storage_flush_requests_total | Metric | Counter | Total storage flush requests. |
| kubevirt_vmi_storage_flush_times_seconds_total | Metric | Counter | Total time spent on cache flushing. |
| kubevirt_vmi_storage_iops_read_total | Metric | Counter | Total number of I/O read operations. |
| kubevirt_vmi_storage_iops_write_total | Metric | Counter | Total number of I/O write operations. |
| kubevirt_vmi_storage_read_times_seconds_total | Metric | Counter | Total time spent on read operations. |
| kubevirt_vmi_storage_read_traffic_bytes_total | Metric | Counter | Total number of bytes read from storage. |
| kubevirt_vmi_storage_write_times_seconds_total | Metric | Counter | Total time spent on write operations. |
| kubevirt_vmi_storage_write_traffic_bytes_total | Metric | Counter | Total number of written bytes. |
| kubevirt_vmi_vcpu_delay_seconds_total | Metric | Counter | Amount of time spent by each vcpu waiting in the queue instead of running. |
| kubevirt_vmi_vcpu_seconds_total | Metric | Counter | Total amount of time spent in each state by each vcpu (cpu_time excluding hypervisor time). Where `id` is the vcpu identifier and `state` can be one of the following: [`OFFLINE`, `RUNNING`, `BLOCKED`]. |
| kubevirt_vmi_vcpu_wait_seconds_total | Metric | Counter | Amount of time spent by each vcpu while waiting on I/O. |
| kubevirt_vmi_vnic_info | Metric | Gauge | Details of VirtualMachineInstance (VMI) vNIC interfaces, such as vNIC name, binding type, network name, and binding name for each vNIC of a running instance. |
| kubevirt_vmsnapshot_succeeded_timestamp_seconds | Metric | Gauge | Returns the timestamp of successful virtual machine snapshot. |
| kubevirt_vnc_active_connections | Metric | Gauge | Amount of active VNC connections, broken down by namespace and vmi name. |
| kubevirt_workqueue_adds_total | Metric | Counter | Total number of adds handled by workqueue |
| kubevirt_workqueue_depth | Metric | Gauge | Current depth of workqueue |
| kubevirt_workqueue_longest_running_processor_seconds | Metric | Gauge | How many seconds has the longest running processor for workqueue been running. |
| kubevirt_workqueue_queue_duration_seconds | Metric | Histogram | How long an item stays in workqueue before being requested. |
| kubevirt_workqueue_retries_total | Metric | Counter | Total number of retries handled by workqueue |
| kubevirt_workqueue_unfinished_work_seconds | Metric | Gauge | How many seconds of work has done that is in progress and hasn't been observed by work_duration. Large values indicate stuck threads. One can deduce the number of stuck threads by observing the rate at which this increases. |
| kubevirt_workqueue_work_duration_seconds | Metric | Histogram | How long in seconds processing an item from workqueue takes. |
| cluster:kubevirt_virt_controller_pods_running:count | Recording rule | Gauge | The number of virt-controller pods that are running. |
| kubevirt_allocatable_nodes | Recording rule | Gauge | The number of allocatable nodes in the cluster. |
| kubevirt_api_request_deprecated_total | Recording rule | Counter | The total number of requests to deprecated KubeVirt APIs. |
| kubevirt_memory_delta_from_requested_bytes | Recording rule | Gauge | The delta between the pod with highest memory working set or rss and its requested memory for each container, virt-controller, virt-handler, virt-api, virt-operator and compute(virt-launcher). |
| kubevirt_nodes_with_kvm | Recording rule | Gauge | The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available. |
| kubevirt_number_of_vms | Recording rule | Gauge | The number of VMs in the cluster by namespace. |
| kubevirt_virt_api_up | Recording rule | Gauge | The number of virt-api pods that are up. |
| kubevirt_virt_controller_ready | Recording rule | Gauge | The number of virt-controller pods that are ready. |
| kubevirt_virt_controller_up | Recording rule | Gauge | The number of virt-controller pods that are up. |
| kubevirt_virt_handler_up | Recording rule | Gauge | The number of virt-handler pods that are up. |
| kubevirt_virt_operator_leading | Recording rule | Gauge | The number of virt-operator pods that are leading. |
| kubevirt_virt_operator_ready | Recording rule | Gauge | The number of virt-operator pods that are ready. |
| kubevirt_virt_operator_up | Recording rule | Gauge | The number of virt-operator pods that are up. |
| kubevirt_vm_container_memory_request_margin_based_on_rss_bytes | Recording rule | Gauge | Difference between requested memory and rss for VM containers (request margin). Can be negative when usage exceeds request. |
| kubevirt_vm_container_memory_request_margin_based_on_working_set_bytes | Recording rule | Gauge | Difference between requested memory and working set for VM containers (request margin). Can be negative when usage exceeds request. |
| kubevirt_vm_created_total | Recording rule | Counter | The total number of VMs created by namespace, since install. |
| kubevirt_vmi_guest_vcpu_queue | Recording rule | Gauge | Guest queue length. |
| kubevirt_vmi_memory_used_bytes | Recording rule | Gauge | Amount of `used` memory as seen by the domain. |
| kubevirt_vmi_migration_data_total_bytes | Recording rule | Counter | [Deprecated] Replaced by kubevirt_vmi_migration_data_bytes_total. |
| kubevirt_vmi_phase_count | Recording rule | Gauge | Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`]. |
| kubevirt_vmsnapshot_disks_restored_from_source | Recording rule | Gauge | Returns the total number of virtual machine disks restored from the source virtual machine. |
| kubevirt_vmsnapshot_disks_restored_from_source_bytes | Recording rule | Gauge | Returns the amount of space in bytes restored from the source virtual machine. |
| kubevirt_vmsnapshot_persistentvolumeclaim_labels | Recording rule | Gauge | Returns the labels of the persistent volume claims that are used for restoring virtual machines. |
| vmi:kubevirt_vmi_memory_available_bytes:sum | Recording rule | Gauge | Sum of available memory bytes per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_memory_headroom_ratio:sum | Recording rule | Gauge | Usable memory to available memory ratio per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_pgmajfaults:rate30m | Recording rule | Gauge | Rate of major page faults over 30 minutes per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_pgmajfaults:rate5m | Recording rule | Gauge | Rate of major page faults over 5 minutes per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_swap_traffic_bytes:rate30m | Recording rule | Gauge | Total swap I/O traffic rate over 30 minutes per VMI (swap in + swap out, aggregated by name, namespace). |
| vmi:kubevirt_vmi_swap_traffic_bytes:rate5m | Recording rule | Gauge | Total swap I/O traffic rate over 5 minutes per VMI (swap in + swap out, aggregated by name, namespace). |
| vmi:kubevirt_vmi_vcpu:count | Recording rule | Gauge | The number of the VMI vCPUs. |

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
