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
	"strings"

	"github.com/google/uuid"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/defaults"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-api/webhooks/mutating-webhook/mutators"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

// Options configures Pod rendering behavior.
type Options struct {
	// LauncherImage is the virt-launcher container image.
	// Default: "quay.io/kubevirt/virt-launcher:latest"
	LauncherImage string

	// FeatureGates lists KubeVirt feature gates to enable.
	FeatureGates []string

	// LauncherQemuTimeout is the QEMU process timeout in seconds.
	// Default: 240
	LauncherQemuTimeout int

	// LauncherSubGid is the supplemental GID for the launcher process.
	// Default: 107
	LauncherSubGid int64

	// ExporterImage is the VM export server image.
	// Default: "quay.io/kubevirt/vm-export:latest"
	ExporterImage string
}

func (o *Options) withDefaults() Options {
	out := *o
	if out.LauncherImage == "" {
		out.LauncherImage = "quay.io/kubevirt/virt-launcher:latest"
	}
	if out.LauncherQemuTimeout == 0 {
		out.LauncherQemuTimeout = 240
	}
	if out.LauncherSubGid == 0 {
		out.LauncherSubGid = 107
	}
	if out.ExporterImage == "" {
		out.ExporterImage = "quay.io/kubevirt/vm-export:latest"
	}
	return out
}

// PodFromVM renders a Pod spec from a VirtualMachine definition.
// It applies VM defaults, extracts the VMI, applies VMI defaults and
// mutations, and renders the launch manifest. No running Kubernetes
// cluster or KubeVirt controllers are required.
func PodFromVM(vm *virtv1.VirtualMachine, opts Options) (*k8sv1.Pod, error) {
	opts = opts.withDefaults()

	if vm.Spec.Template == nil {
		return nil, fmt.Errorf("VM %q has no template spec", vm.Name)
	}

	config := newOfflineRenderConfig(opts)

	vmCopy := vm.DeepCopy()
	if vmCopy.Namespace == "" {
		vmCopy.Namespace = "default"
	}

	defaults.SetVirtualMachineDefaults(vmCopy, config, nil)

	vmi := setupVMIFromVM(vmCopy)
	renderer := newOfflineRenderer(vmi, config, opts)

	return renderPod(vmi, config, renderer, opts)
}

// PodFromVMI renders a Pod spec from a VirtualMachineInstance definition.
// It applies VMI defaults and mutations, and renders the launch manifest.
// No running Kubernetes cluster or KubeVirt controllers are required.
func PodFromVMI(vmi *virtv1.VirtualMachineInstance, opts Options) (*k8sv1.Pod, error) {
	opts = opts.withDefaults()

	config := newOfflineRenderConfig(opts)
	vmiCopy := vmi.DeepCopy()
	renderer := newOfflineRenderer(vmiCopy, config, opts)

	return renderPod(vmiCopy, config, renderer, opts)
}

func renderPod(vmi *virtv1.VirtualMachineInstance, config RenderConfig, renderer ManifestRenderer, opts Options) (*k8sv1.Pod, error) {
	if vmi.Namespace == "" {
		vmi.Namespace = "default"
	}

	if err := mutators.ApplyNewVMIMutations(vmi, config); err != nil {
		return nil, fmt.Errorf("failed to apply VMI mutations: %w", err)
	}

	if err := vmispec.SetDefaultNetworkInterface(config, &vmi.Spec); err != nil {
		return nil, fmt.Errorf("failed to set default network: %w", err)
	}

	util.SetDefaultVolumeDisk(&vmi.Spec)
	autoAttachInputDevice(vmi)

	pod, err := renderer.RenderLaunchManifest(vmi)
	if err != nil {
		return nil, err
	}

	pod.TypeMeta = metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"}

	// RenderLaunchManifest sets GenerateName (e.g. "virt-launcher-myvm-") but not
	// Name. Standalone runtimes like podman kube play require a concrete Name, so
	// we strip the trailing dash to produce one.
	if pod.GenerateName != "" && pod.Name == "" {
		pod.Name = strings.TrimRight(pod.GenerateName, "-")
		pod.GenerateName = ""
	}

	return pod, nil
}

func newOfflineRenderer(vmi *virtv1.VirtualMachineInstance, config RenderConfig, opts Options) ManifestRenderer {
	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	stubPVCs(pvcCache, vmi)

	resourceQuotaStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
	namespaceStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

	return services.NewTemplateService(
		opts.LauncherImage,
		opts.LauncherQemuTimeout,
		"/var/run/kubevirt",
		"/var/run/kubevirt-ephemeral-disks",
		"/var/run/kubevirt/container-disks",
		virtv1.HotplugDiskDir,
		"",
		pvcCache,
		nil,
		config,
		opts.LauncherSubGid,
		opts.ExporterImage,
		resourceQuotaStore,
		namespaceStore,
	)
}

var firmwareUUIDns = uuid.MustParse("6a1a24a1-4061-4607-8bf4-a3963d0c5895")

func setupVMIFromVM(vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {
	vmi := &virtv1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: virtv1.GroupVersion.String(),
			Kind:       "VirtualMachineInstance",
		},
		ObjectMeta: *vm.Spec.Template.ObjectMeta.DeepCopy(),
		Spec:       *vm.Spec.Template.Spec.DeepCopy(),
	}
	vmi.ObjectMeta.Name = vm.ObjectMeta.Name
	vmi.ObjectMeta.GenerateName = ""
	vmi.ObjectMeta.Namespace = vm.ObjectMeta.Namespace
	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	if vmi.Spec.Domain.Firmware == nil {
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{}
	}
	if vmi.Spec.Domain.Firmware.UUID == "" {
		vmi.Spec.Domain.Firmware.UUID = types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(vmi.Name)).String())
	}

	util.SetDefaultVolumeDisk(&vmi.Spec)

	return vmi
}

func autoAttachInputDevice(vmi *virtv1.VirtualMachineInstance) {
	if vmi.Spec.Domain.Devices.AutoattachInputDevice == nil ||
		!*vmi.Spec.Domain.Devices.AutoattachInputDevice ||
		len(vmi.Spec.Domain.Devices.Inputs) > 0 {
		return
	}
	vmi.Spec.Domain.Devices.Inputs = append(vmi.Spec.Domain.Devices.Inputs,
		virtv1.Input{Name: "default-0"})
}

// stubPVCs pre-populates the PVC cache so RenderLaunchManifest can look up
// PersistentVolumeClaim volumes without a real Kubernetes API.
func stubPVCs(pvcCache cache.Indexer, vmi *virtv1.VirtualMachineInstance) {
	ns := vmi.Namespace
	if ns == "" {
		ns = "default"
	}
	filesystemMode := k8sv1.PersistentVolumeFilesystem
	for _, vol := range vmi.Spec.Volumes {
		if vol.PersistentVolumeClaim == nil {
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
