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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/hooks"
	netbinding "kubevirt.io/kubevirt/pkg/network/netbinding"
	netannotations "kubevirt.io/kubevirt/pkg/network/pod/annotations"
	storageannotations "kubevirt.io/kubevirt/pkg/storage/pod/annotations"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

const (
	defaultExporterImage          = "quay.io/kubevirt/vm-export:latest"
	defaultQemuTimeout            = 240
	defaultLauncherSubGid   int64 = 107
	defaultVirtShareDir           = "/var/run/kubevirt"
	defaultEphemeralDir           = "/var/run/kubevirt-ephemeral-disks"
	defaultContainerDiskDir       = "/var/run/kubevirt/container-disks"
	computeContainerName          = "compute"
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
	// TemplateService chooses 107 for non-root VMs and 0 when the Root
	// feature gate is enabled.
	RunAsUser int64

	// ExporterImage is the VM export server image.
	// Default: quay.io/kubevirt/vm-export:latest
	ExporterImage string

	// PVCs resolve PersistentVolumeClaim, DataVolume, and ephemeral
	// volumes. VolumeMode (Filesystem vs Block) is taken from each
	// claim. Claims are matched by namespace/name. Referenced claims
	// missing here are stubbed as Filesystem RWO in the VMI namespace.
	PVCs []*k8sv1.PersistentVolumeClaim
}

func (o Options) withDefaults() (Options, error) {
	if o.LauncherImage == "" {
		return o, fmt.Errorf("LauncherImage is required")
	}
	if o.LauncherQemuTimeout == 0 {
		o.LauncherQemuTimeout = defaultQemuTimeout
	}
	if o.LauncherSubGid == 0 {
		o.LauncherSubGid = defaultLauncherSubGid
	}
	if o.ExporterImage == "" {
		o.ExporterImage = defaultExporterImage
	}
	return o, nil
}

// Result is the defaulted VMI together with the virt-launcher Pod
// rendered from it. Standalone runtimes that need to attach the VMI
// (for example as STANDALONE_VMI) should use Result.VMI, not a
// separately defaulted object.
type Result struct {
	VMI *virtv1.VirtualMachineInstance
	Pod *k8sv1.Pod
}

// ManifestRenderer produces a virt-launcher Pod from a prepared VMI.
// *services.TemplateService satisfies it.
type ManifestRenderer interface {
	RenderLaunchManifest(vmi *virtv1.VirtualMachineInstance) (*k8sv1.Pod, error)
}

var _ ManifestRenderer = (*services.TemplateService)(nil)

// LaunchManifest renders a virt-launcher Pod from an already-prepared VMI.
// renderer is typically *services.TemplateService; a future virt-launcher
// plugin can implement ManifestRenderer and be passed in instead.
func LaunchManifest(vmi *virtv1.VirtualMachineInstance, renderer ManifestRenderer) (*k8sv1.Pod, error) {
	return renderer.RenderLaunchManifest(vmi)
}

func prepareVMI(vmi *virtv1.VirtualMachineInstance, cfg *offlineConfig) error {
	AutoAttachInputDevice(vmi)
	if err := mutators.ApplyNewVMIMutations(vmi, cfg); err != nil {
		return fmt.Errorf("failed to apply VMI mutations: %w", err)
	}
	return nil
}

// VMIFromVM extracts a VirtualMachineInstance from a VirtualMachine,
// applying VM defaults and VMI mutations. It is an offline
// transformation: no cluster client or informers are required. The
// returned VMI is the same object PodFromVM would render a Pod from.
// Instancetype and preference matchers are not applied; VMs that
// reference them are rejected.
func VMIFromVM(vm *virtv1.VirtualMachine, opts Options) (*virtv1.VirtualMachineInstance, error) {
	opts, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}
	if vm.Spec.Template == nil {
		return nil, fmt.Errorf("VM %q has no template spec", vm.Name)
	}
	if err := rejectUnsupportedVM(vm); err != nil {
		return nil, err
	}

	cfg := newOfflineConfig(opts)
	vmCopy := vm.DeepCopy()
	if vmCopy.Namespace == "" {
		vmCopy.Namespace = "default"
	}
	defaults.SetVirtualMachineDefaults(vmCopy, cfg, nil)

	vmi := NewVMI(vmCopy)
	if err := prepareVMI(vmi, cfg); err != nil {
		return nil, err
	}
	return vmi, nil
}

// PodFromVM renders a virt-launcher Pod from a VirtualMachine definition.
// No running cluster is required. The returned Result.VMI is the fully
// defaulted instance the Pod was built from.
func PodFromVM(vm *virtv1.VirtualMachine, opts Options) (*Result, error) {
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
func PodFromVMI(vmi *virtv1.VirtualMachineInstance, opts Options) (*Result, error) {
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
	return launch(vmiCopy, opts)
}

func launch(vmi *virtv1.VirtualMachineInstance, opts Options) (*Result, error) {
	renderer := newOfflineRenderer(vmi, opts)
	pod, err := LaunchManifest(vmi, renderer)
	if err != nil {
		return nil, err
	}
	pod.TypeMeta = metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}
	applyRunAsUser(pod, opts.RunAsUser)
	return &Result{VMI: vmi, Pod: pod}, nil
}

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

func newOfflineRenderer(vmi *virtv1.VirtualMachineInstance, opts Options) ManifestRenderer {
	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	loadPVCs(pvcCache, vmi, opts.PVCs)
	cfg := newOfflineConfig(opts)

	return services.NewTemplateService(
		opts.LauncherImage,
		opts.LauncherQemuTimeout,
		defaultVirtShareDir,
		defaultEphemeralDir,
		defaultContainerDiskDir,
		virtv1.HotplugDiskDir,
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
		services.WithAnnotationsGenerators(netannotations.NewGenerator(cfg), storageannotations.Generator{}),
	)
}

func loadPVCs(pvcCache cache.Indexer, vmi *virtv1.VirtualMachineInstance, provided []*k8sv1.PersistentVolumeClaim) {
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

func claimNameForVolume(vol virtv1.Volume) string {
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

func rejectUnsupportedVM(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype != nil {
		return fmt.Errorf("offline render does not apply instancetype matchers")
	}
	if vm.Spec.Preference != nil {
		return fmt.Errorf("offline render does not apply preference matchers")
	}
	return nil
}

func offlineHookSidecars(vmi *virtv1.VirtualMachineInstance, _ *virtv1.KubeVirtConfiguration) (hooks.HookSidecarList, error) {
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
