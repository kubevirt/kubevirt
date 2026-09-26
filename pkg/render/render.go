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
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

const (
	defaultLauncherImage          = "quay.io/kubevirt/virt-launcher:latest"
	defaultExporterImage          = "quay.io/kubevirt/vm-export:latest"
	defaultQemuTimeout            = 240
	defaultLauncherSubGid   int64 = 107
	defaultVirtShareDir           = "/var/run/kubevirt"
	defaultEphemeralDir           = "/var/run/kubevirt-ephemeral-disks"
	defaultContainerDiskDir       = "/var/run/kubevirt/container-disks"
)

// Options configures offline Pod rendering. It is the public/offline
// surface; virt-controller keeps passing real caches and clients to
// TemplateService directly.
type Options struct {
	// LauncherImage is the virt-launcher container image.
	// Default: quay.io/kubevirt/virt-launcher:latest
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

	// ExporterImage is the VM export server image.
	// Default: quay.io/kubevirt/vm-export:latest
	ExporterImage string

	// PVCs resolve PersistentVolumeClaim volumes. VolumeMode (Filesystem
	// vs Block) is taken from each claim. Claims referenced by the VMI
	// but missing here are stubbed as Filesystem RWO.
	PVCs []*k8sv1.PersistentVolumeClaim
}

func (o Options) withDefaults() Options {
	if o.LauncherImage == "" {
		o.LauncherImage = defaultLauncherImage
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
	return o
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
func LaunchManifest(vmi *virtv1.VirtualMachineInstance, renderer ManifestRenderer) (*k8sv1.Pod, error) {
	return renderer.RenderLaunchManifest(vmi)
}

// PrepareVMI applies the same mutations the VMI admission webhook applies
// (defaults, feature-gate annotations, non-root marking) and then attaches
// a default input device when requested.
func PrepareVMI(vmi *virtv1.VirtualMachineInstance, cfg mutators.ClusterConfigProvider) error {
	if err := mutators.ApplyNewVMIMutations(vmi, cfg); err != nil {
		return fmt.Errorf("failed to apply VMI mutations: %w", err)
	}
	AutoAttachInputDevice(vmi)
	return nil
}

// VMIFromVM extracts a VirtualMachineInstance from a VirtualMachine,
// applying VM defaults and VMI mutations. The returned VMI is the same
// object PodFromVM would render a Pod from.
func VMIFromVM(vm *virtv1.VirtualMachine, opts Options) (*virtv1.VirtualMachineInstance, error) {
	opts = opts.withDefaults()
	if vm.Spec.Template == nil {
		return nil, fmt.Errorf("VM %q has no template spec", vm.Name)
	}

	cfg := newOfflineConfig(opts)
	vmCopy := vm.DeepCopy()
	if vmCopy.Namespace == "" {
		vmCopy.Namespace = "default"
	}
	defaults.SetVirtualMachineDefaults(vmCopy, cfg, nil)

	vmi := NewVMI(vmCopy)
	if err := PrepareVMI(vmi, cfg); err != nil {
		return nil, err
	}
	return vmi, nil
}

// PodFromVM renders a virt-launcher Pod from a VirtualMachine definition.
// No running cluster is required. The returned Result.VMI is the fully
// defaulted instance the Pod was built from.
func PodFromVM(vm *virtv1.VirtualMachine, opts Options) (*Result, error) {
	opts = opts.withDefaults()
	vmi, err := VMIFromVM(vm, opts)
	if err != nil {
		return nil, err
	}
	return launch(vmi, opts)
}

// PodFromVMI renders a virt-launcher Pod from a VirtualMachineInstance.
// VMI defaults and mutations are applied. No running cluster is required.
func PodFromVMI(vmi *virtv1.VirtualMachineInstance, opts Options) (*Result, error) {
	opts = opts.withDefaults()
	vmiCopy := vmi.DeepCopy()
	if vmiCopy.Namespace == "" {
		vmiCopy.Namespace = "default"
	}
	cfg := newOfflineConfig(opts)
	if err := PrepareVMI(vmiCopy, cfg); err != nil {
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
	return &Result{VMI: vmi, Pod: pod}, nil
}

func newOfflineRenderer(vmi *virtv1.VirtualMachineInstance, opts Options) ManifestRenderer {
	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	loadPVCs(pvcCache, vmi, opts.PVCs)

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
		newOfflineConfig(opts),
		opts.LauncherSubGid,
		opts.ExporterImage,
		cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
		cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc),
	)
}

func loadPVCs(pvcCache cache.Indexer, vmi *virtv1.VirtualMachineInstance, provided []*k8sv1.PersistentVolumeClaim) {
	ns := vmi.Namespace
	if ns == "" {
		ns = "default"
	}

	byName := make(map[string]*k8sv1.PersistentVolumeClaim, len(provided))
	for _, pvc := range provided {
		if pvc == nil {
			continue
		}
		_ = pvcCache.Add(pvc)
		byName[pvc.Name] = pvc
	}

	filesystemMode := k8sv1.PersistentVolumeFilesystem
	for _, vol := range vmi.Spec.Volumes {
		if vol.PersistentVolumeClaim == nil {
			continue
		}
		if _, ok := byName[vol.PersistentVolumeClaim.ClaimName]; ok {
			continue
		}
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vol.PersistentVolumeClaim.ClaimName,
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
