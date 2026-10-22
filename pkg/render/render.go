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

package render

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/hooks"
	netbinding "kubevirt.io/kubevirt/pkg/network/netbinding"
	netannotations "kubevirt.io/kubevirt/pkg/network/pod/annotations"
	netresources "kubevirt.io/kubevirt/pkg/network/resources"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/storage/memorydump"
	storageannotations "kubevirt.io/kubevirt/pkg/storage/pod/annotations"
	storagevmispec "kubevirt.io/kubevirt/pkg/storage/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

const (
	// Matches virt-controller's launcherQemuTimeout in watch/application.go.
	defaultQemuTimeout      = 240
	computeContainerName    = "compute"
	defaultEphemeralDiskDir = util.VirtShareDir + "-ephemeral-disks"
)

// Options configures offline Pod rendering. It is the public/offline
// surface; virt-controller keeps passing real caches and clients to
// TemplateService directly.
type Options struct {
	// LauncherImage is the virt-launcher container image. Required.
	LauncherImage string

	// FeatureGates lists KubeVirt feature gates to enable.
	FeatureGates []string

	// DisabledFeatureGates lists feature gates to explicitly disable,
	// including Beta gates that are on by default.
	DisabledFeatureGates []string

	// LauncherQemuTimeout is the QEMU process timeout in seconds.
	// Default: 240
	LauncherQemuTimeout int

	// LauncherSubGid is the supplemental GID for the launcher process.
	// Default: 107
	LauncherSubGid int64

	// RunAsUser is the UID for the virt-launcher process. When unset,
	// mutations choose 107 for non-root VMs and 0 when the Root feature
	// gate is enabled. Non-zero values are written to
	// vmi.Status.RuntimeUser before rendering so launcher mode matches,
	// then applied to the Pod. TemplateService still hardcodes UID 107
	// for any non-root VMI; a follow-up will teach it to use RuntimeUser.
	RunAsUser int64

	// ExporterImage is the VM export server image. Optional; pkg/render
	// does not render export pods today.
	ExporterImage string

	// Architecture is the target cluster CPU architecture (amd64, arm64,
	// or s390x). It selects the machine type, node selector, and EFI
	// firmware path. Empty uses runtime.GOARCH.
	Architecture string

	// NetworkBindings is the cluster's binding-plugin table, copied from
	// KubeVirt.spec.configuration.network.binding. A VMI interface that
	// names a plugin needs that entry here so the sidecar and its memory
	// overhead can be rendered. Core bindings (bridge, masquerade, passt)
	// do not use this map.
	NetworkBindings map[string]v1.InterfaceBindingPlugin

	// PVCs resolve PersistentVolumeClaim, DataVolume, and ephemeral
	// volumes. VolumeMode (Filesystem vs Block) is taken from each
	// claim. Claims are matched by namespace/name. Referenced claims
	// missing here are stubbed as Filesystem RWO in the VMI namespace.
	PVCs []*k8sv1.PersistentVolumeClaim

	// Instancetype is applied to the VMI when the VM references an
	// instancetype matcher. Required when Spec.Instancetype is set.
	Instancetype *v1beta1.VirtualMachineInstancetypeSpec

	// Preference is applied to the VMI when the VM references a
	// preference matcher. Required when Spec.Preference is set.
	Preference *v1beta1.VirtualMachinePreferenceSpec
}

func (o Options) withDefaults() (Options, error) {
	if o.LauncherImage == "" {
		return o, fmt.Errorf("LauncherImage is required")
	}
	if o.LauncherQemuTimeout == 0 {
		o.LauncherQemuTimeout = defaultQemuTimeout
	}
	if o.LauncherSubGid == 0 {
		o.LauncherSubGid = util.NonRootUID
	}
	return o, nil
}

// Result is the defaulted VMI together with the virt-launcher Pod
// rendered from it. Standalone runtimes that need to attach the VMI
// (for example as STANDALONE_VMI) should use Result.VMI, not a
// separately defaulted object.
type Result struct {
	VMI *v1.VirtualMachineInstance
	Pod *k8sv1.Pod
}

// ManifestRenderer produces a virt-launcher Pod from a prepared VMI.
// *services.TemplateService satisfies it.
type ManifestRenderer interface {
	RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error)
}

var _ ManifestRenderer = (*services.TemplateService)(nil)

// LaunchManifest renders a virt-launcher Pod from an already-prepared VMI.
// renderer is typically *services.TemplateService; a future virt-launcher
// plugin can implement ManifestRenderer and be passed in instead.
func LaunchManifest(vmi *v1.VirtualMachineInstance, renderer ManifestRenderer) (*k8sv1.Pod, error) {
	return renderer.RenderLaunchManifest(vmi)
}

// setFirmwareDefaultsIfEmpty matches the VM mutator: a new VM with an
// empty firmware UUID or serial gets random values. The name-based
// legacy UUID is not used.
func setFirmwareDefaultsIfEmpty(vm *v1.VirtualMachine) {
	if vm.Spec.Template == nil {
		return
	}
	if vm.Spec.Template.Spec.Domain.Firmware == nil {
		vm.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{}
	}
	fw := vm.Spec.Template.Spec.Domain.Firmware
	if fw.UUID == "" {
		fw.UUID = types.UID(uuid.New().String())
	}
	if fw.Serial == "" {
		fw.Serial = uuid.New().String()
	}
}

func prepareVMI(vmi *v1.VirtualMachineInstance, cfg *offlineConfig) error {
	if err := mutators.ApplyNewVMIMutations(vmi, cfg); err != nil {
		return fmt.Errorf("failed to apply VMI mutations: %w", err)
	}
	return nil
}

// VMIFromVM extracts a VirtualMachineInstance from a VirtualMachine,
// applying VM defaults and VMI mutations. It is an offline
// transformation: no cluster client or informers are required. The
// returned VMI is the same object PodFromVM would render a Pod from.
// Instancetype and preference matchers are applied when the matching
// spec is supplied in Options; otherwise VMs that reference them are
// rejected.
func VMIFromVM(vm *v1.VirtualMachine, opts Options) (*v1.VirtualMachineInstance, error) {
	if vm == nil {
		return nil, fmt.Errorf("VM is nil")
	}
	opts, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}
	if vm.Spec.Template == nil {
		return nil, fmt.Errorf("VM %q has no template spec", vm.Name)
	}
	if err := rejectUnsupportedVM(vm, opts); err != nil {
		return nil, err
	}

	cfg := newOfflineConfig(opts)
	vmCopy := vm.DeepCopy()
	if vmCopy.Namespace == "" {
		vmCopy.Namespace = "default"
	}
	setFirmwareDefaultsIfEmpty(vmCopy)
	// Architecture and machine type preferences are applied before VM
	// defaults, matching the VM admission mutator.
	applyAdmissionPreferences(vmCopy, opts.Preference)
	defaults.SetVirtualMachineDefaults(vmCopy, cfg, nil)

	vmi := NewVMI(vmCopy)
	// Start-paused stays in the VM controller. It comes from a runtime
	// stateChangeRequest, not from the VM spec. A completed memory dump
	// is recorded on the VM status; drop that volume here, before
	// mutations pair disks, so offline render matches the controller.
	if memorydump.HasCompleted(vmCopy) {
		vmi.Spec = *memorydump.RemoveMemoryDumpVolumeFromVMISpec(&vmi.Spec, vmCopy.Status.MemoryDumpRequest.ClaimName)
	}
	// Pair disks before preferences so PreferredDiskBus can fill an empty
	// bus. Later VMI defaulting would otherwise set the architecture bus.
	storagevmispec.SetDefaultVolumeDisk(&vmi.Spec)
	applyPreferenceAutoAttach(vmi, opts.Preference)
	// Only the VM path attaches a default input. A standalone VMI does
	// not, matching the VMI admission webhook.
	AutoAttachInputDevice(vmi)
	// Add the default pod interface before instancetype and preference
	// expansion, matching virt-controller. Interface preferences then
	// apply to that interface instead of missing it.
	if err := netvmispec.SetDefaultNetworkInterface(cfg, &vmi.Spec); err != nil {
		return nil, fmt.Errorf("failed to set default network interface: %w", err)
	}
	if err := applyInstancetypeToVMI(vmCopy, vmi, opts); err != nil {
		return nil, err
	}
	if err := mutators.ApplyNewVMIMutations(vmi, cfg); err != nil {
		return nil, fmt.Errorf("failed to apply VMI mutations: %w", err)
	}
	applyRunAsUserToVMI(vmi, opts.RunAsUser)
	return vmi, nil
}

// PodFromVM renders a virt-launcher Pod from a VirtualMachine definition.
// No running cluster is required. The returned Result.VMI is the fully
// defaulted instance the Pod was built from.
func PodFromVM(vm *v1.VirtualMachine, opts Options) (*Result, error) {
	if vm == nil {
		return nil, fmt.Errorf("VM is nil")
	}
	opts, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}
	vmi, err := VMIFromVM(vm, opts)
	if err != nil {
		return nil, err
	}
	return launch(vmi, opts)
}

// PodFromVMI renders a virt-launcher Pod from a VirtualMachineInstance.
// VMI defaults and mutations are applied. No running cluster is required.
func PodFromVMI(vmi *v1.VirtualMachineInstance, opts Options) (*Result, error) {
	if vmi == nil {
		return nil, fmt.Errorf("VMI is nil")
	}
	opts, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}
	vmiCopy := vmi.DeepCopy()
	if vmiCopy.Namespace == "" {
		vmiCopy.Namespace = "default"
	}
	cfg := newOfflineConfig(opts)
	if err := prepareVMI(vmiCopy, cfg); err != nil {
		return nil, err
	}
	applyRunAsUserToVMI(vmiCopy, opts.RunAsUser)
	return launch(vmiCopy, opts)
}

func launch(vmi *v1.VirtualMachineInstance, opts Options) (*Result, error) {
	renderer := newOfflineRenderer(vmi, opts)
	pod, err := LaunchManifest(vmi, renderer)
	if err != nil {
		return nil, err
	}
	pod.TypeMeta = metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}
	applyRunAsUser(pod, opts.RunAsUser)
	return &Result{VMI: vmi, Pod: pod}, nil
}

func applyRunAsUserToVMI(vmi *v1.VirtualMachineInstance, uid int64) {
	if uid == 0 {
		return
	}
	vmi.Status.RuntimeUser = uint64(uid)
}

// applyRunAsUser overrides the Pod UID after TemplateService renders.
// TemplateService currently hardcodes util.NonRootUID (107) for any
// non-root VMI. A follow-up will teach it to use status.runtimeUser.
func applyRunAsUser(pod *k8sv1.Pod, uid int64) {
	if uid == 0 {
		return
	}
	if pod.Spec.SecurityContext == nil {
		pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
	}
	pod.Spec.SecurityContext.RunAsUser = &uid
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name != computeContainerName {
			continue
		}
		if pod.Spec.Containers[i].SecurityContext == nil {
			pod.Spec.Containers[i].SecurityContext = &k8sv1.SecurityContext{}
		}
		pod.Spec.Containers[i].SecurityContext.RunAsUser = &uid
	}
}

func newOfflineRenderer(vmi *v1.VirtualMachineInstance, opts Options) ManifestRenderer {
	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	loadPVCs(pvcCache, vmi, opts.PVCs)
	// Persistent TPM, persistent EFI, and CBT state need a backend-storage
	// PVC. In a cluster, virt-controller creates persistent-state-for-<vm>
	// and retries until it exists. Offline, PodFromVM and PodFromVMI stub
	// that claim when Options.PVCs does not already contain it (matched by
	// name, or by the persistent-state-for label). The returned Pod
	// references the claim. The caller creates that PVC where the Pod
	// runs, the same way it creates disk PVCs named by the Pod.
	ensureBackendStoragePVC(pvcCache, vmi)
	cfg := newOfflineConfig(opts)

	return services.NewTemplateService(
		opts.LauncherImage,
		opts.LauncherQemuTimeout,
		util.VirtShareDir,
		defaultEphemeralDiskDir,
		filepath.Join(util.VirtShareDir, "container-disks"),
		v1.HotplugDiskDir,
		"",
		pvcCache,
		nil,
		cfg,
		opts.LauncherSubGid,
		opts.ExporterImage,
		cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
		cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
		services.WithSidecarCreator(offlineHookSidecars),
		services.WithSidecarCreator(netbinding.NetBindingPluginSidecarList),
		services.WithMemoryOverheadCalculators(netresources.NewMemoryCalculator(cfg)),
		services.WithAnnotationsGenerators(netannotations.NewGenerator(cfg), storageannotations.Generator{}),
	)
}

// loadPVCs puts caller-supplied claims into the cache, then stubs any
// PVC, DataVolume, or ephemeral claim that is still missing as
// Filesystem RWO. A DataVolume's volume mode lives on its PVC, not on
// the VM. Without that claim the stub is filesystem and the Pod gets a
// mount. Pass the Block PVC in Options.PVCs, with the same name as the
// DataVolume, to get a volume device instead.
func loadPVCs(pvcCache cache.Indexer, vmi *v1.VirtualMachineInstance, provided []*k8sv1.PersistentVolumeClaim) {
	ns := vmi.Namespace
	if ns == "" {
		ns = "default"
	}

	providedKeys := make(map[string]struct{}, len(provided))
	for _, pvc := range provided {
		if pvc == nil {
			continue
		}
		pvcCopy := pvc.DeepCopy()
		if pvcCopy.Namespace == "" {
			pvcCopy.Namespace = ns
		}
		_ = pvcCache.Add(pvcCopy)
		providedKeys[pvcCopy.Namespace+"/"+pvcCopy.Name] = struct{}{}
	}

	filesystemMode := k8sv1.PersistentVolumeFilesystem
	for _, vol := range vmi.Spec.Volumes {
		claimName := claimNameForVolume(vol)
		if claimName == "" {
			continue
		}
		if _, ok := providedKeys[ns+"/"+claimName]; ok {
			continue
		}
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      claimName,
				Namespace: ns,
			},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
				VolumeMode:  &filesystemMode,
			},
		}
		_ = pvcCache.Add(pvc)
	}
}

func ensureBackendStoragePVC(pvcCache cache.Indexer, vmi *v1.VirtualMachineInstance) {
	if !backendstorage.IsBackendStorageNeeded(vmi) {
		return
	}
	if backendstorage.PVCForVMI(pvcCache, vmi) != nil {
		return
	}

	ns := vmi.Namespace
	if ns == "" {
		ns = "default"
	}
	filesystemMode := k8sv1.PersistentVolumeFilesystem
	_ = pvcCache.Add(&k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backendstorage.PVCPrefix + "-" + vmi.Name,
			Namespace: ns,
			Labels:    map[string]string{backendstorage.PVCPrefix: vmi.Name},
		},
		Spec: k8sv1.PersistentVolumeClaimSpec{
			AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce},
			VolumeMode:  &filesystemMode,
		},
	})
}

func claimNameForVolume(vol v1.Volume) string {
	switch {
	case vol.PersistentVolumeClaim != nil:
		return vol.PersistentVolumeClaim.ClaimName
	case vol.DataVolume != nil:
		return vol.DataVolume.Name
	case vol.Ephemeral != nil && vol.Ephemeral.PersistentVolumeClaim != nil:
		return vol.Ephemeral.PersistentVolumeClaim.ClaimName
	default:
		return ""
	}
}

func offlineHookSidecars(vmi *v1.VirtualMachineInstance, _ *v1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
	list, err := hooks.UnmarshalHookSidecarList(vmi)
	if err != nil {
		return nil, err
	}
	for _, sidecar := range list {
		if sidecar.ConfigMap != nil {
			return nil, fmt.Errorf("offline render does not support ConfigMap-backed hook sidecars")
		}
	}
	return list, nil
}
