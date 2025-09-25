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
	ExpandDisksGate       = "ExpandDisks"
	CPUManager            = "CPUManager"
	IgnitionGate          = "ExperimentalIgnitionSupport"
	HypervStrictCheckGate = "HypervStrictCheck"
	SidecarGate           = "Sidecar"
	HostDevicesGate       = "HostDevices"

	// Owner: sig-storage
	// Alpha: v0.30.0
	// Beta: v1.3.0
	SnapshotGate = "Snapshot"

	// Owner: sig-storage
	// Alpha: v0.55.0
	// Beta: v1.3.0
	VMExportGate       = "VMExport"
	HotplugVolumesGate = "HotplugVolumes"
	HostDiskGate       = "HostDisk"

	// Owner: sig-storage
	// Alpha: v1.7.0
	//
	// UtilityVolumes enables utility volumes feature which provides a general capability
	// of hot-plugging volumes directly into the virt-launcher Pod for operational workflows
	UtilityVolumesGate = "UtilityVolumes"

	DownwardMetricsFeatureGate = "DownwardMetrics"
	Root                       = "Root"
	WorkloadEncryptionSEV      = "WorkloadEncryptionSEV"
	WorkloadEncryptionTDX      = "WorkloadEncryptionTDX"
	VSOCKGate                  = "VSOCK"
	// KubevirtSeccompProfile indicate that Kubevirt will install its custom profile and
	// user can tell Kubevirt to use it
	KubevirtSeccompProfile = "KubevirtSeccompProfile"
	// DisableMediatedDevicesHandling disables the handling of mediated
	// devices, its creation and deletion
	DisableMediatedDevicesHandling = "DisableMDEVConfiguration"
	// PersistentReservation enables the use of the SCSI persistent reservation with the pr-helper daemon
	PersistentReservation = "PersistentReservation"

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

	// Owner: @shellyka13
	// Alpha: v1.6.0
	//
	// IncrementalBackup feature gate enables creating full and incremental backups for virtual machines.
	// These backups leverage libvirt's native backup capabilities, providing a storage-agnostic solution.
	// To support incremental backups, a QCOW2 overlay must be created on top of the VM's raw disk image.
	IncrementalBackupGate = "IncrementalBackup"

	VirtIOFSConfigVolumesGate = "EnableVirtioFsConfigVolumes"
	VirtIOFSStorageVolumeGate = "EnableVirtioFsStorageVolumes"

	// Owner: @alaypatel07
	// Alpha: v1.6.0
	//
	// GPUsWithDRAGate allows users to create VMIs with DRA provisioned GPU devices
	GPUsWithDRAGate = "GPUsWithDRA"

	// Owner: @alaypatel07
	// Alpha: v1.6.0
	//
	// HostDevicesWithDRAGate allows users to create VMIs with DRA provisioned Host devices
	HostDevicesWithDRAGate = "HostDevicesWithDRA"

	DecentralizedLiveMigration = "DecentralizedLiveMigration"

	// Owner: sig-storage / @alromeros
	// Alpha: v1.6.0
	//
	// ObjectGraph introduces a new subresource for VMs and VMIs.
	// This subresource returns a structured list of k8s objects that are related
	// to the specified VM or VMI, enabling better dependency tracking.
	ObjectGraph = "ObjectGraph"

	// DeclarativeHotplugVolumes enables adding/removing volumes declaratively
	// also implicitly handles inject/eject CDROM
	DeclarativeHotplugVolumesGate = "DeclarativeHotplugVolumes"

	// Owner: sig-conpute / @jschintag
	// Alpha: v1.6.0
	// Beta: v1.7.0
	//
	// SecureExecution introduces secure execution of VMs on IBM Z architecture
	SecureExecution = "SecureExecution"

	// VideoConfig enables VM owners to specify a video device type (e.g., virtio, vga, bochs, ramfb) via the `Video` field, overriding default settings.
	// Requires `autoattachGraphicsDevice` to be true or unset. Alpha feature, defaults unchanged.
	// Owner: @dasionov
	// Alpha: v1.6.0
	// Beta: v1.7.0
	//
	VideoConfig = "VideoConfig"

	// Owner: @varunrsekar
	// Alpha: v1.6.0
	// Beta: v1.7.0
	//
	// PanicDevices allows defining panic devices for signaling crashes in the guest for a VirtualMachineInstance.
	PanicDevicesGate = "PanicDevices"

	// Alpha: v1.6.0
	//
	// PasstIPStackMigration enables seamless migration with passt network binding.
	PasstIPStackMigration = "PasstIPStackMigration"

	// MigrationPriorityQueue enables controllers to assign priorities to migrations,
	// ensuring system-initiated migrations (e.g., node drains, upgrades) take precedence
	// over user-initiated ones (e.g., hot plug operations).
	// Owner: sig-compute / @fossedihelm
	// Alpha: v1.7.0
	//
	MigrationPriorityQueue = "MigrationPriorityQueue"

	// Owner: @DariuszPorowski
	// Alpha: v1.?.0
	// This feature is disabled by default. When enabled, it allows setting specific Hyper-V
	// fields on the VMI spec which are otherwise restricted.
	HyperVLayered = "HyperVLayered"

	// Owner: @harshitgupta1337
	// Alpha: v1.?.0
	// This feature is disabled by default. When enabled, it allows using a
	// hypervisor other than KVM for running VMs.
	// Details of the new hypervisor should be specified via the
	// HypervisorConfiguration field in KubeVirtConfiguration.
	ConfigurableHypervisor = "ConfigurableHypervisor"
)

func init() {
	RegisterFeatureGate(FeatureGate{Name: ImageVolume, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: ExpandDisksGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: CPUManager, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: IgnitionGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HypervStrictCheckGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SidecarGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HostDevicesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SnapshotGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: VMExportGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: HotplugVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HostDiskGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DownwardMetricsFeatureGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: Root, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: WorkloadEncryptionSEV, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: WorkloadEncryptionTDX, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VSOCKGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: KubevirtSeccompProfile, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: DisableMediatedDevicesHandling, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PersistentReservation, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: AlignCPUsGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: NodeRestrictionGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: VirtIOFSConfigVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: VirtIOFSStorageVolumeGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: GPUsWithDRAGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HostDevicesWithDRAGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DecentralizedLiveMigration, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: DeclarativeHotplugVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: SecureExecution, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: VideoConfig, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: PanicDevicesGate, State: Beta})
	RegisterFeatureGate(FeatureGate{Name: UtilityVolumesGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: PasstIPStackMigration, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: HyperVLayered, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: ConfigurableHypervisor, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: IncrementalBackupGate, State: Alpha})
	RegisterFeatureGate(FeatureGate{Name: MigrationPriorityQueue, State: Alpha})
}
