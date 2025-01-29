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

package featuregate

import "fmt"

const (
	LiveMigrationGate      = "LiveMigration"
	SRIOVLiveMigrationGate = "SRIOVLiveMigration"
	NonRoot                = "NonRoot"
	PSA                    = "PSA"
	CPUNodeDiscoveryGate   = "CPUNodeDiscovery"
	NUMAFeatureGate        = "NUMA"
	GPUGate                = "GPU"
	// VMLiveUpdateFeaturesGate allows updating certain VM fields, such as CPU sockets to enable hot-plug functionality.
	// GA:    v1.4.0
	VMLiveUpdateFeaturesGate = "VMLiveUpdateFeatures"

	// CommonInstancetypesDeploymentGate enables the deployment of common-instancetypes by virt-operator
	// Owner: @lyarwood
	// Alpha: v1.1.0
	// Beta:  v1.2.0
	// GA:	  v1.4.0
	CommonInstancetypesDeploymentGate = "CommonInstancetypesDeploymentGate"

	// HotplugNetworkIfacesGate controls the network interface hotplug feature lifecycle.
	// Alpha: v1.1.0
	// Beta:  v1.3.0
	// GA:    v1.4.0
	HotplugNetworkIfacesGate = "HotplugNICs"

	// BochsDisplayForEFIGuests instructs EFI guests to start with Bochs display (instead of VGA)
	// GA:    v1.4.0
	BochsDisplayForEFIGuests = "BochsDisplayForEFIGuests"

	// AutoResourceLimitsGate enables automatic setting of vmi limits if there is a ResourceQuota with limits associated with the vmi namespace.
	// GA:    v1.5.0
	AutoResourceLimitsGate = "AutoResourceLimitsGate"

	// DockerSELinuxMCSWorkaround sets the SELinux level of all the non-compute virt-launcher containers to "s0".
	// Deprecated: v1.4.0
	DockerSELinuxMCSWorkaround = "DockerSELinuxMCSWorkaround"

	// NetworkBindingPlugingsGate enables using a plugin to bind the pod and the VM network
	// Alpha: v1.1.0
	// Beta:  v1.4.0
	// GA:    v1.5.0
	NetworkBindingPlugingsGate = "NetworkBindingPlugins"

	// DynamicPodInterfaceNamingGate enables a mechanism to dynamically determine the primary pod interface for KubeVirt virtual machines.
	// Beta:  v1.4.0
	// GA:    v1.5.0
	DynamicPodInterfaceNamingGate = "DynamicPodInterfaceNaming"

	PasstGate   = "Passt"
	MacvtapGate = "Macvtap"

	VirtIOFSGate = "ExperimentalVirtiofsSupport"
	// VolumesUpdateStrategy enables to specify the strategy on the volume updates.
	// Introduced in v1.3.0
	VolumesUpdateStrategy = "VolumesUpdateStrategy"
	// VolumeMigration enables to migrate the storage. It depends on the VolumesUpdateStrategy feature.
	// Introduced in v1.3.0
	VolumeMigration = "VolumeMigration"

	// DisableCustomSELinuxPolicy disables the installation of the custom SELinux policy for virt-launcher
	DisableCustomSELinuxPolicy = "DisableCustomSELinuxPolicy"
)

func init() {
	RegisterFeatureGate(FeatureGate{Name: LiveMigrationGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: SRIOVLiveMigrationGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: NonRoot, State: GA})
	RegisterFeatureGate(FeatureGate{Name: PSA, State: GA})
	RegisterFeatureGate(FeatureGate{Name: CPUNodeDiscoveryGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: NUMAFeatureGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: CommonInstancetypesDeploymentGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: GPUGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: HotplugNetworkIfacesGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: BochsDisplayForEFIGuests, State: GA})
	RegisterFeatureGate(FeatureGate{Name: VMLiveUpdateFeaturesGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: NetworkBindingPlugingsGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: DynamicPodInterfaceNamingGate, State: GA})
	RegisterFeatureGate(FeatureGate{Name: VolumesUpdateStrategy, State: GA})
	RegisterFeatureGate(FeatureGate{Name: VolumeMigration, State: GA})
	RegisterFeatureGate(FeatureGate{Name: DisableCustomSELinuxPolicy, State: GA})
	RegisterFeatureGate(FeatureGate{Name: AutoResourceLimitsGate, State: GA})

	RegisterFeatureGate(FeatureGate{Name: DockerSELinuxMCSWorkaround, State: Deprecated, Message: fmt.Sprintf(
		"DockerSELinuxMCSWorkaround has been deprecated since v1.4.")})
	RegisterFeatureGate(FeatureGate{Name: VirtIOFSGate, State: Deprecated, Message: VirtioFsFeatureGateDeprecationMessage})

	RegisterFeatureGate(FeatureGate{Name: PasstGate, State: Discontinued, Message: PasstDiscontinueMessage, VmiSpecUsed: passtApiUsed})
	RegisterFeatureGate(FeatureGate{Name: MacvtapGate, State: Discontinued, Message: MacvtapDiscontinueMessage, VmiSpecUsed: macvtapApiUsed})
}
