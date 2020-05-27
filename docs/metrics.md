# Exposed Metrics

Sometimes the Help text on `/metrics` endpoint just isn't enough to explain what a certain metric means. This document's objective is to give further explanation to KubeVirt related metrics.

#### kubevirt_info

Kubevirt's version information

| Label       	| Description                                              	|
|-------------	|----------------------------------------------------------	|
| goversion   	| GO version used to compile this version of KubeVirt      	|
| kubeversion 	| Git commit refspec that created this version of KubeVirt 	|

#### kubevirt_vmi_memory_resident_bytes

Total resident memory in bytes. Usually set on VMI manifest's [like this example](../examples/vmi-ephemeral.yaml#L8-19)

| Label     	| Description                                                                                                                                	|
|-----------	|--------------------------------------------------------------------------------------------------------------------------------------------	|
| domain    	| Unique name used by `libvirtd` to identify the given VMI. This unique name is created with the VMI's name concatenated with it's namespace. 	|
| name      	| VMI's name given on it's specification.                                                                                                    	|
| namespace 	| Namespace which the given VMI is related to.                                                                                               	|
| node      	| Node where the VMI is running on.                                                                                                          	|

#### kubevirt_vmi_network_errors_total

Counter of network errors when transmitting and receiving data.

| Label     	| Description                                                                                                                                	|
|-----------	|--------------------------------------------------------------------------------------------------------------------------------------------	|
| domain    	| Unique name used by `libvirtd` to identify the given VMI. This unique name is created with the VMI's name concatenated with it's namespace 	|
| name      	| VMI's name given on it's specification.                                                                                                    	|
| namespace 	| Namespace which the given VMI is related to.                                                                                               	|
| node      	| Node where the VMI is running on.                                                                                                          	|
| interface 	| Which network interface that errors are occurring                                                                                          	|
| type      	| Identify if the error occurred when transmitting or receiving data. `tx` when transmitting and `rx` when receiving.                        	|

#### kubevirt_vmi_network_traffic_bytes_total

How much traffic is being transmitted and received, in bytes.

| Label     	| Description                                                                                                                                	|
|-----------	|--------------------------------------------------------------------------------------------------------------------------------------------	|
| domain    	| Unique name used by `libvirtd` to identify the given VMI. This unique name is created with the VMI's name concatenated with it's namespace 	|
| name      	| VMI's name given on it's specification.                                                                                                    	|
| namespace 	| Namespace which the given VMI is related to.                                                                                               	|
| node      	| Node where the VMI is running on.                                                                                                          	|
| interface 	| Which network interface that data is being transmitted/received                                                                            	|
| type      	| Identify if the data is being transmitted or received. `tx` when transmitting and `rx` when receiving.                                     	|

#### kubevirt_vmi_network_traffic_packets_total

How much packets are being transmitted and received, in bytes.

| Label     	| Description                                                                                                                                	|
|-----------	|--------------------------------------------------------------------------------------------------------------------------------------------	|
| domain    	| Unique name used by `libvirtd` to identify the given VMI. This unique name is created with the VMI's name concatenated with it's namespace 	|
| name      	| VMI's name given on it's specification.                                                                                                    	|
| namespace 	| Namespace which the given VMI is related to.                                                                                               	|
| node      	| Node where the VMI is running on.                                                                                                          	|
| interface 	| Which network interface that packets are being transmitted/received                                                                            	|
| type      	| Identify if the packet are being transmitted or received. `tx` when transmitting and `rx` when receiving.                                     	|

#### kubevirt_vmi_phase_count

This metric will return the total amount of VMIs per node and phase.

| Label 	| Description                                                 	|
|-------	|-------------------------------------------------------------	|
| phase 	| Phase of the VMI. It can be one of [kubernetes pod lifecycle](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/) 	|
| node  	| Node where the VMI is running on.                           	|
