/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virtcontroller

const (
	metricLabelNode          = "node"
	metricLabelName          = "name"
	metricLabelNamespace     = "namespace"
	metricLabelPhase         = "phase"
	metricLabelVNICName      = "vnic_name"
	metricLabelFlavor        = "flavor"
	metricLabelInstanceType  = "instance_type"
	metricLabelMigrationName = "migration_name"
	metricLabelStatus        = "status"
	metricLabelBindingName   = "binding_name"
	metricLabelResource      = "resource"

	resourceNameCPU    = "cpu"
	resourceNameMemory = "memory"

	resourceUnitBytes   = "bytes"
	resourceUnitCores   = "cores"
	resourceUnitThreads = "threads"
	resourceUnitSockets = "sockets"

	resourceSourceDefault        = "default"
	resourceSourceDomain         = "domain"
	resourceSourceGuest          = "guest"
	resourceSourceGuestEffective = "guest_effective"
	resourceSourceHugepages      = "hugepages"
	resourceSourceRequests       = "requests"

	bindingNameBridge     = "bridge"
	bindingNameMasquerade = "masquerade"
	bindingNameSRIOV      = "sriov"
	networkNamePod        = "pod networking"

	vmiInterfaceTypeExternal = "ExternalInterface"
	vmiInterfaceTypeSystem   = "SystemInterface"

	vmStatusGroupRunning        = "running"
	migrationStatusSucceeded    = "succeeded"
	metricVMResourceLimits      = "kubevirt_vm_resource_limits"
	metricLabelMigrationVMIName = "vmi"
)
