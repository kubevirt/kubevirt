# Exposed Metrics

Sometimes the Help text on `/metrics` endpoint just isn't enough to explain what a certain metric means. This document's objective is to give further explanation to KubeVirt related metrics.

## VM Related Metrics 

#### kubevirt_info

Kubevirt's version information

| Label | Description |
|-------------|----------------------------------------------------------|
| goversion | GO version used to compile this version of KubeVirt |
| kubeversion | Git commit refspec that created this version of KubeVirt |

#### kubevirt_vmi_memory_resident_bytes

Total resident memory of the process running the VMI. Usually set on VMI manifest's [like this example](../examples/vmi-ephemeral.yaml#L8-19)

| Label | Description |
|-----------|----------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |

#### kubevirt_vmi_memory_available_bytes

Total amount of usable memory.

| Label | Description |
|-----------|----------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |

#### kubevirt_vmi_memory_swap_traffic_bytes_total

How much traffic is being read and written in swap memory.

| Label | Description |
|-----------|----------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| type | Whether the data is being transmitted or received. `tx` when transmitting and `rx` when receiving. |

#### kubevirt_vmi_network_errors_total

Counter of network errors when transmitting and receiving data.

| Label | Description |
|-----------|-----------------------------------------------------------------------------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| interface | Which network interface that errors are occurring. |
| type | Whether the error occurred when transmitting or receiving data. `tx` when transmitting and `rx` when receiving. |

#### kubevirt_vmi_network_traffic_bytes_total

How much traffic is being transmitted and received.

| Label | Description |
|-----------|----------------------------------------------------------------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| interface | Which network interface that data is being transmitted/received. |
| type | Whether the data is being transmitted or received. `tx` when transmitting and `rx` when receiving. |

#### kubevirt_vmi_network_traffic_packets_total

How much packets are being transmitted and received.

| Label | Description |
|-----------|-------------------------------------------------------------------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| interface | Which network interface that packets are being transmitted/received. |
| type | Whether the packet are being transmitted or received. `tx` when transmitting and `rx` when receiving. |

#### kubevirt_vmi_storage_iops_total

Counter of read and write operations per disk device.

| Label | Description |
|-----------|----------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| drive | Disk device that is being written/read. |
| type | Whether it's a read or write operation. |

#### kubevirt_vmi_storage_times_ms_total

Total time spent on read and write operations per disk device.

| Label | Description |
|-----------|----------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| drive | Disk device that is being written/read. |
| type | Whether it's a read or write operation. |

#### kubevirt_vmi_storage_traffic_bytes_total

Total amount of data read and written per disk device.

| Label | Description |
|-----------|----------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| drive | Disk device that is being written/read. |
| type | Whether it's a read or write operation. |

#### kubevirt_vmi_vcpu_seconds

Total amount of time spent in each vcpu state

| Label | Description |
|-----------|------------------------------------------------------------------------------------------------------------|
| name | VMI's name given on it's specification. |
| namespace | Namespace which the given VMI is related to. |
| node | Node where the VMI is running on. |
| id | Indentifier to a single Virtual CPU. |
| state | Identify the Virtual CPU state. It can be one of libvirtd vcpu's states: `OFFLINE`, `RUNNING` or `BLOCKED` |

#### kubevirt_vmi_phase_count

Total amount of VMIs per node and phase.

| Label | Description |
|-------|-----------------------------------------------------------------------------------------------------------------------------------------|
| phase | Phase of the VMI. It can be one of [kubernetes pod lifecycle phases](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/) |
| node | Node where the VMI is running on. |

## Reflector Metrics

> To be done

## Client Metrics

> To be done

## Workqueue Metrics

> To be done

## RoadMap

List of metrics that are not implemented right now, but are on our RoadMap:

> To be done