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

const (
	CPUManager            = "CPUManager"
	IgnitionGate          = "ExperimentalIgnitionSupport"
	HypervStrictCheckGate = "HypervStrictCheck"
	SidecarGate           = "Sidecar"
	HostDevicesGate       = "HostDevices"

	// Owner: sig-storage
	// Alpha: v0.30.0
	// Beta: v1.3.0
	SnapshotGate = "Snapshot"

	HostDiskGate = "HostDisk"

	// Owner: sig-storage
	// Alpha: v1.7.0
	//
	// UtilityVolumes enables utility volumes feature which provides a general capability
	// of hot-plugging volumes directly into the virt-launcher Pod for operational workflows
	UtilityVolumesGate = "UtilityVolumes"

	DownwardMetricsFeatureGate = "DownwardMetrics"
	Root                       = "Root"

	// Owner: sig-compute / @alancaldelas
	// Alpha: v0.49.0
	// Beta: v1.9.0
	WorkloadEncryptionSEV = "WorkloadEncryptionSEV"
	WorkloadEncryptionTDX = "WorkloadEncryptionTDX"
	VSOCKGate             = "VSOCK"
	// KubevirtSeccompProfile indicate that Kubevirt will install its custom profile and
	// user can tell Kubevirt to use it
	KubevirtSeccompProfile = "KubevirtSeccompProfile"
	// AlignCPUsGate allows emulator thread to assign two extra CPUs if needed to complete even parity.
	AlignCPUsGate = "AlignCPUs"

	// Owner: @xpivarc
	// Alpha: v1.3.0
	// Beta: v1.6.0
	//
	// NodeRestriction enables Kubelet's like NodeRestriction but for Kubevirt's virt-handler.
	// This feature requires following Kubernetes feature gate "ServiceAccountTokenPodNodeInfo". The feature gate is available
	// in Kubernetes 1.30 as Beta and was graduated in 1.32.
	NodeRestrictionGate = "NodeRestriction"

	// Owner: @Barakmor1
	// Alpha: v1.6.0
	// Beta: v1.7.0
	//
	// ImageVolume The ImageVolume FG in KubeVirt uses Kubernetes ImageVolume FG to eliminate
	// the need for an extra container for containerDisk, improving security by avoiding
	// bind mounts in virt-handler.
	ImageVolume = "ImageVolume"

	// Owner: @Barakmor1
	// Alpha: v1.8.0
	// Beta: v1.9.0
	//
	// LibvirtHooksServerAndClient The LibvirtHooksServerAndClient FG enables running pre-migration
	// hooks on the target virt-launcher pod, allowing domain XML mutations to be applied
	// on the target before migration starts.
	LibvirtHooksServerAndClient = "LibvirtHooksServerAndClient"

	// Owner: @shellyka13
	// Alpha: v1.6.0
	//
	// IncrementalBackup feature gate enables creating full and incremental backups for virtual machines.
	// These backups leverage libvirt's native backup capabilities, providing a storage-agnostic solution.
	// To support incremental backups, a QCOW2 overlay must be created on top of the VM's raw disk image.
	IncrementalBackupGate = "IncrementalBackup"

	VirtIOFSStorageVolumeGate = "EnableVirtioFsStorageVolumes"

	// Owner: @alaypatel07
	// Alpha: v1.6.0
	// Beta: v1.9.0
	//
	// GPUsWithDRAGate allows users to create VMIs with DRA provisioned GPU devices
	GPUsWithDRAGate = "GPUsWithDRA"

	// Owner: @alaypatel07
	// Alpha: v1.6.0
	// Beta: v1.9.0
	//
	// HostDevicesWithDRAGate allows users to create VMIs with DRA provisioned Host devices
	HostDevicesWithDRAGate = "HostDevicesWithDRA"

	// Owner: @mresvanis
	// Alpha: v1.6.0
	//
	// PCINUMAAwareTopologyEnabled enables NUMA-aware PCIe topology mapping for passthrough devices
	PCINUMAAwareTopologyEnabled = "PCINUMAAwareTopology"

	// Owner: SIG network
	// Alpha: v1.9.0
	//
	// NetworkDevicesWithDRAGate allows users to create VMIs with DRA provisioned Network devices
	// specified in spec.networks with resourceClaim type. This enables DRA-managed network
	// resources to be attached to VMs using the natural networks API.
	NetworkDevicesWithDRAGate = "NetworkDevicesWithDRA"

	DecentralizedLiveMigration = "DecentralizedLiveMigration"

	// Owner: sig-storage / @alromeros
	// Alpha: v1.6.0
	//
	// ObjectGraph introduces a new subresource for VMs and VMIs.
	// This subresource returns a structured list of k8s objects that are related
	// to the specified VM or VMI, enabling better dependency tracking.
	ObjectGraph = "ObjectGraph"

	// Owner: sig-storage / @mhenriks
	// Alpha: v1.6.0
	// Beta: v1.9.0
	//
	// DeclarativeHotplugVolumes enables adding/removing volumes declaratively
	// also implicitly handles inject/eject CDROM
	DeclarativeHotplugVolumesGate = "DeclarativeHotplugVolumes"

	// Beta: v1.8.0
	//
	// PasstBinding enables the use of passt core network binding
	PasstBinding = "PasstBinding"

	// Owner: @harshitgupta1337
	// Alpha: v1.8.0
	// This feature is disabled by default. When enabled, it allows using
	// hypervisors other than KVM for running VMs.
	// Details of the new hypervisors should be specified via the
	// HypervisorConfigurations field in KubeVirtConfiguration.
	ConfigurableHypervisor = "ConfigurableHypervisor"

	// PodSecondaryInterfaceNamingUpgrade enables the upgrade mechanism for VMs
	// stuck with the obsolete ordinal naming scheme for their pod secondary networks
	// Owner: SIG network
	// Beta: v1.8
	PodSecondaryInterfaceNamingUpgrade = "PodSecondaryInterfaceNamingUpgrade"

	// ExternalNetResourceInjection disables the VMI controller query of NetworkAttachmentDefinition objects and
	// the deployment of related RBAC rules by virt-operator.
	// Owner: SIG network
	// Beta: v1.8.0
	ExternalNetResourceInjection = "ExternalNetResourceInjection"

	// Owner: sig-compute / @MarSik
	// Alpha: v1.8.0
	// Beta: v1.9.0
	//
	// RebootPolicy enables setting the RebootPolicy field on VMI's DomainSpec
	// which allows terminating the VMI on guest reboot instead of silently rebooting,
	// enabling the VM controller to recreate the VMI with updated configuration.
	RebootPolicy = "RebootPolicy"

	// Owner: sig-compute / @0xFelix
	// Template enables the deployment of virt-template components by virt-operator.
	// Alpha: v1.8.0
	// Beta: v1.9.0
	Template = "Template"

	// Owner: @bmordeha
	// Alpha: v1.8.0
	// Beta: v1.9.0
	//
	// VmiMemoryOverheadReport enables reporting the memory overhead in the VMI status.
	// When enabled, the memory overhead is calculated and set in the VMI status.Memory.MemoryOverhead field.
	VmiMemoryOverheadReport = "VmiMemoryOverheadReport"

	// Owner: sig-storage / @mhenriks
	// Alpha: v1.8.0
	//
	// ContainerPathVolumes enables exposing virt-launcher volumeMount paths to the VM
	// via virtiofs. This allows VMs to access credentials and tokens injected into pods
	// by external systems such as AWS IRSA, GKE Workload Identity, or TEE attestation.
	ContainerPathVolumesGate = "ContainerPathVolumes"

	// Enables using the spec.domain.memory.ReservedOverhead field which
	// can specify some required memory overhead as well as whether VM
	// memory (and overhead) needs to be locked or not
	// Owner: sig-compute / @bgartzi
	// Alpha: v1.8.0
	ReservedOverheadMemlock = "ReservedOverheadMemlock"

	// Owner: @orenc1
	// Alpha: v1.8.0
	// Beta: v1.9.0
	//
	// OptOutRoleAggregation enables the RoleAggregationStrategy field in KubeVirtConfiguration,
	// allowing users to opt out of aggregating KubeVirt ClusterRoles to the default Kubernetes roles.
	OptOutRoleAggregation = "OptOutRoleAggregation"

	// Owner: @csomani1
	// Alpha: v1.8.0
	//
	// The VGPULiveMigration fg enables the vGPU hook to run for vGPU live migrations, allowing the
	// target XML's mdev UUID to be mutated.
	VGPULiveMigration = "VGPULiveMigration"

	// Owner: sig-compute / @enp0s3
	// Alpha: v1.9.0
	//
	// VMStatsCollector enables the additional guest agent polling workers
	// (frequent/medium/infrequent tiers) that collect raw monitoring data
	// for the GetVMStats gRPC RPC.
	VMStatsCollector = "VMStatsCollector"

	// Owner: sig-compute, sig-storage / @0xFelix
	// Alpha: v1.9.0
	//
	// OCIExport enables exporting VM disks as OCI image layout TAR archives.
	OCIExport = "OCIExport"

	// Owner: @iholder101
	// Alpha: v1.9.0
	//
	// Plugins enables the Plugin CRD for declarative VM extension
	// via domain hooks, node hooks, and admission references (VEP-190).
	PluginsGate = "Plugins"

	// Owner: sig-compute / @fanzhangio
	// Alpha: v1.9.0
	// GraceIOVirtualization enables GPU passthrough optimized for NVIDIA Grace
	// platforms (i.e ARM64 architectures by utilizing SMMUv3 IOMMU).
	// It utilizes SMMUv3 IOMMU, IOMMUFD device binding on ARM64, and ACPI Generic Initiator NUMA topology.
	GraceIOVirtualization = "GraceIOVirtualization"

	// Owner: sig-compute / @fossedihelm
	// Alpha: v1.9.0
	//
	// IOMMUFD enables the IOMMUFD device plugin for passthrough devices.
	// When enabled, virt-controller requests devices.kubevirt.io/iommufd for
	// every launcher pod. Nodes without /dev/iommu (kernel <6.2) will report
	// unhealthy devices, making pods unschedulable there.
	// This feature also emits domain-level <iommufd enabled='yes' fdgroup='iommu'/>
	// and uses virDomainFDAssociate, which require a libvirt/QEMU stack that supports
	// fdgroup-based IOMMUFD, currently libvirt >= 12.2. Enable this gate only on
	// clusters where target nodes and virt-launcher images provide compatible
	// kernel, libvirt, and QEMU support.
	IOMMUFDGate = "IOMMUFD"

	// Owner: sig-compute / @lyarwood
	// Alpha: v1.9.0
	//
	// FirmwareAutoSelection uses libvirt's firmware auto-selection feature for
	// EFI Secure Boot instead of hardcoded OVMF firmware paths.
	FirmwareAutoSelection = "FirmwareAutoSelection"

	// Owner: @aseeef
	// Alpha: v1.9.0
	//
	// MigrationStallDetection enables iteration-aligned stall detection and migration convergence tuning.
	MigrationStallDetection = "MigrationStallDetection"

	// Owner: @michalskrivanek
	// Alpha: v1.9.0
	//
	// MigrationDowntimeTuning enables iteration-aware downtime ramping for live
	// migration convergence via the ExperimentalMigrationOptions.DowntimeTuning field.
	MigrationDowntimeTuning = "MigrationDowntimeTuning"

	// Owner: sig-compute / @lyarwood
	// Alpha: v1.9.0
	//
	// CrossArchitectureVirtualization enables cross-architecture VM execution.
	// When enabled, VMs can run on nodes with a different CPU architecture than
	// the guest (e.g., ARM64 guests on AMD64 hosts) via software emulation or
	// hardware-accelerated virtualization. Independent of useEmulation.
	// See VEP #172.
	CrossArchitectureVirtualization = "CrossArchitectureVirtualization"

	// Owner: sig-network
	// Alpha: v1.9.0
	//
	// PortRangesSpec enables the portRanges field, initially only on masquerade interfaces,
	// allowing compact specification of contiguous port intervals to forward to the VM guest.
	PortRangesSpec = "PortRangesSpec"
)

func init() {
	RegisterFeatureGate(FeatureGate{Name: LibvirtHooksServerAndClient, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: ImageVolume, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: CPUManager, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: IgnitionGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HypervStrictCheckGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SidecarGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HostDevicesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SnapshotGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: HostDiskGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DownwardMetricsFeatureGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: Root, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: WorkloadEncryptionSEV, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: WorkloadEncryptionTDX, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VSOCKGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: KubevirtSeccompProfile, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: AlignCPUsGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: NodeRestrictionGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: VirtIOFSStorageVolumeGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: GPUsWithDRAGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: HostDevicesWithDRAGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: PCINUMAAwareTopologyEnabled, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: NetworkDevicesWithDRAGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DecentralizedLiveMigration, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DeclarativeHotplugVolumesGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: ObjectGraph, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: UtilityVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: ConfigurableHypervisor, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PasstBinding, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: IncrementalBackupGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PodSecondaryInterfaceNamingUpgrade, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: ExternalNetResourceInjection, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: RebootPolicy, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: Template, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: VmiMemoryOverheadReport, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: ContainerPathVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: ReservedOverheadMemlock, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: OptOutRoleAggregation, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: VGPULiveMigration, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VMStatsCollector, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: OCIExport, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PluginsGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: GraceIOVirtualization, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: IOMMUFDGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: FirmwareAutoSelection, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: MigrationStallDetection, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: MigrationDowntimeTuning, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: CrossArchitectureVirtualization, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PortRangesSpec, State: Alpha})
}
