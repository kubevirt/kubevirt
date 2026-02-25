# KubeVirt metrics

| Name | Kind | Type | Description |
|------|------|------|-------------|
| kubevirt_configuration_emulation_enabled | Metric | Gauge | Indicates whether the Software Emulation is enabled in the configuration. |
| kubevirt_console_active_connections | Metric | Gauge | Amount of active Console connections, broken down by namespace and vmi name. |
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
| kubevirt_vm_migrating_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to migrating status. |
| kubevirt_vm_non_running_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to paused/stopped status. |
| kubevirt_vm_resource_limits | Metric | Gauge | Resources limits by Virtual Machine. Reports memory and CPU limits. |
| kubevirt_vm_resource_requests | Metric | Gauge | Resources requested by Virtual Machine. Reports memory and CPU requests. |
| kubevirt_vm_running_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to running status. |
| kubevirt_vm_starting_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to starting status. |
| kubevirt_vm_vnic_info | Metric | Gauge | Details of Virtual Machine (VM) vNIC interfaces, such as vNIC name, binding type, network name, and binding name for each vNIC defined in the VM's configuration. |
| kubevirt_vmi_info | Metric | Gauge | Information about VirtualMachineInstances. |
| kubevirt_vmi_last_api_connection_timestamp_seconds | Metric | Gauge | Virtual Machine Instance last API connection timestamp. Including VNC, console, portforward, SSH and usbredir connections. |
| kubevirt_vmi_launcher_memory_overhead_bytes | Metric | Gauge | Estimation of the memory amount required for virt-launcher's infrastructure components (e.g. libvirt, QEMU). |
| kubevirt_vmi_migration_end_time_seconds | Metric | Gauge | The time at which the migration ended. |
| kubevirt_vmi_migration_failed | Metric | Gauge | Indicates if the VMI migration failed. |
| kubevirt_vmi_migration_phase_transition_time_from_creation_seconds | Metric | Histogram | Histogram of VM migration phase transitions duration from creation time in seconds. |
| kubevirt_vmi_migration_start_time_seconds | Metric | Gauge | The time at which the migration started. |
| kubevirt_vmi_migration_succeeded | Metric | Gauge | Indicates if the VMI migration succeeded. |
| kubevirt_vmi_migrations_in_pending_phase | Metric | Gauge | Number of current pending migrations. |
| kubevirt_vmi_migrations_in_running_phase | Metric | Gauge | Number of current running migrations. |
| kubevirt_vmi_migrations_in_scheduling_phase | Metric | Gauge | Number of current scheduling migrations. |
| kubevirt_vmi_migrations_in_unset_phase | Metric | Gauge | Number of current unset migrations. These are pending items the virt-controller hasn’t processed yet from the queue. |
| kubevirt_vmi_non_evictable | Metric | Gauge | Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable. |
| kubevirt_vmi_number_of_outdated | Metric | Gauge | Indication for the total number of VirtualMachineInstance workloads that are not running within the most up-to-date version of the virt-launcher environment. |
| kubevirt_vmi_phase_transition_time_from_creation_seconds | Metric | Histogram | Histogram of VM phase transitions duration from creation time in seconds. |
| kubevirt_vmi_phase_transition_time_from_deletion_seconds | Metric | Histogram | Histogram of VM phase transitions duration from deletion time in seconds. |
| kubevirt_vmi_phase_transition_time_seconds | Metric | Histogram | Histogram of VM phase transitions duration between different phases in seconds. |
| kubevirt_vmi_status_addresses | Metric | Gauge | The addresses of a VirtualMachineInstance. This metric provides the address of an available network interface associated with the VMI in the 'address' label, and about the type of address, such as internal IP, in the 'type' label. |
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
| kubevirt_allocatable_nodes | Recording rule | Gauge | The number of allocatable nodes in the cluster. |
| kubevirt_api_request_deprecated_total | Recording rule | Counter | The total number of requests to deprecated KubeVirt APIs. |
| kubevirt_memory_delta_from_requested_bytes | Recording rule | Gauge | The delta between the pod with highest memory working set or rss and its requested memory for each container, virt-controller, virt-handler, virt-api and virt-operator. |
| kubevirt_nodes_with_kvm | Recording rule | Gauge | The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available. |
| kubevirt_number_of_vms | Recording rule | Gauge | The number of VMs in the cluster by namespace. |
| kubevirt_virt_api_up | Recording rule | Gauge | The number of virt-api pods that are up. |
| kubevirt_virt_controller_ready | Recording rule | Gauge | The number of virt-controller pods that are ready. |
| kubevirt_virt_controller_up | Recording rule | Gauge | The number of virt-controller pods that are up. |
| kubevirt_virt_handler_up | Recording rule | Gauge | The number of virt-handler pods that are up. |
| kubevirt_virt_operator_leading | Recording rule | Gauge | The number of virt-operator pods that are leading. |
| kubevirt_virt_operator_ready | Recording rule | Gauge | The number of virt-operator pods that are ready. |
| kubevirt_virt_operator_up | Recording rule | Gauge | The number of virt-operator pods that are up. |
| kubevirt_vm_container_free_memory_bytes_based_on_rss | Recording rule | Gauge | The current available memory of the VM containers based on the rss. |
| kubevirt_vm_container_free_memory_bytes_based_on_working_set_bytes | Recording rule | Gauge | The current available memory of the VM containers based on the working set. |
| kubevirt_vm_created_total | Recording rule | Counter | The total number of VMs created by namespace, since install. |
| kubevirt_vmi_memory_used_bytes | Recording rule | Gauge | Amount of `used` memory as seen by the domain. |
| kubevirt_vmi_phase_count | Recording rule | Gauge | Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`]. |
| kubevirt_vmsnapshot_disks_restored_from_source | Recording rule | Gauge | Returns the total number of virtual machine disks restored from the source virtual machine. |
| kubevirt_vmsnapshot_disks_restored_from_source_bytes | Recording rule | Gauge | Returns the amount of space in bytes restored from the source virtual machine. |
| kubevirt_vmsnapshot_persistentvolumeclaim_labels | Recording rule | Gauge | Returns the labels of the persistent volume claims that are used for restoring virtual machines. |

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
