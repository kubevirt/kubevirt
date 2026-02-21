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

package mutators

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virtiofs"
)

const (
	computeContainerName = "compute"
)

// VirtLauncherPodMutator mutates virt-launcher pods to inject virtiofs containers
// for containerPath volumes.
type VirtLauncherPodMutator struct {
	ClusterConfig *virtconfig.ClusterConfig
	VirtClient    kubecli.KubevirtClient
}

func NewVirtLauncherPodMutator(clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient) *VirtLauncherPodMutator {
	return &VirtLauncherPodMutator{
		ClusterConfig: clusterConfig,
		VirtClient:    virtClient,
	}
}

func (m *VirtLauncherPodMutator) Mutate(ar *admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	if ar.Request.Resource.Resource != "pods" {
		return allowedResponse()
	}

	pod := &k8sv1.Pod{}
	if err := json.Unmarshal(ar.Request.Object.Raw, pod); err != nil {
		log.Log.Reason(err).Error("Failed to unmarshal pod")
		return allowedResponse()
	}

	// Only mutate virt-launcher pods
	if !isVirtLauncherPod(pod) {
		return allowedResponse()
	}

	// Get the VMI from the owner reference
	vmi, err := m.getVMIFromOwnerRef(pod)
	if err != nil {
		log.Log.Reason(err).Warningf("Failed to get VMI for pod %s/%s", pod.Namespace, pod.Name)
		return allowedResponse()
	}

	if vmi == nil {
		return allowedResponse()
	}

	// Check which virtiofs containers are missing (idempotency check)
	missingContainers := virtiofs.MissingContainerPathContainers(vmi, pod)
	if len(missingContainers) == 0 {
		return allowedResponse()
	}

	// Find the compute container to get the image and volumeMounts
	computeContainer := findContainer(pod, computeContainerName)
	if computeContainer == nil {
		log.Log.Warningf("Compute container not found in pod %s/%s", pod.Namespace, pod.Name)
		return allowedResponse()
	}

	// Get containerPath volumes that need virtiofs containers
	containerPathVolumes := virtiofs.GetContainerPathVolumesWithFilesystems(vmi)

	// Generate virtiofs containers for missing containerPath volumes
	containersToAdd := m.generateContainerPathVirtiofsContainers(vmi, pod, containerPathVolumes, computeContainer, missingContainers)
	if len(containersToAdd) == 0 {
		return allowedResponse()
	}

	// Create patch to add containers
	patchSet := patch.New()
	for _, container := range containersToAdd {
		patchSet.AddOption(patch.WithAdd("/spec/containers/-", container))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		log.Log.Reason(err).Error("Failed to generate patch")
		return allowedResponse()
	}

	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: pointer.P(admissionv1.PatchTypeJSONPatch),
	}
}

func (m *VirtLauncherPodMutator) generateContainerPathVirtiofsContainers(
	vmi *v1.VirtualMachineInstance,
	pod *k8sv1.Pod,
	volumes []v1.Volume,
	computeContainer *k8sv1.Container,
	missingContainers []string,
) []k8sv1.Container {
	var containers []k8sv1.Container

	// Build set of missing container names for efficient lookup
	missingSet := make(map[string]struct{}, len(missingContainers))
	for _, name := range missingContainers {
		missingSet[name] = struct{}{}
	}

	for _, volume := range volumes {
		containerName := virtiofs.ContainerPathVirtiofsContainerName(volume.Name)

		// Skip if container is not in the missing set
		if _, isMissing := missingSet[containerName]; !isMissing {
			continue
		}

		// Only create virtiofs container when readOnly is explicitly true
		if volume.ContainerPath.ReadOnly == nil || !*volume.ContainerPath.ReadOnly {
			log.Log.Warningf("ContainerPath volume %s does not have readOnly=true; skipping virtiofs container", volume.Name)
			continue
		}

		// Find the volumeMount in compute container that matches the containerPath
		volumeMount, subPath := virtiofs.FindVolumeMountForPath(computeContainer, volume.ContainerPath.Path)
		if volumeMount == nil {
			log.Log.Warningf("No volumeMount found for containerPath %s in volume %s", volume.ContainerPath.Path, volume.Name)
			continue
		}

		// Validate that the pod volume is a supported type for ContainerPath
		podVolume := virtiofs.FindPodVolumeByName(pod, volumeMount.Name)
		if podVolume == nil {
			log.Log.Warningf("Pod volume %s not found for containerPath volume %s", volumeMount.Name, volume.Name)
			continue
		}
		if !virtiofs.IsSupportedContainerPathVolumeType(podVolume) {
			log.Log.Warningf("Pod volume %s has unsupported type for containerPath volume %s; supported types are: ConfigMap, Secret, Projected, DownwardAPI, EmptyDir", volumeMount.Name, volume.Name)
			continue
		}

		container := m.createVirtiofsContainer(vmi, volume, computeContainer.Image, volumeMount, subPath)
		containers = append(containers, container)
	}

	return containers
}

func (m *VirtLauncherPodMutator) createVirtiofsContainer(
	vmi *v1.VirtualMachineInstance,
	volume v1.Volume,
	image string,
	sourceMount *k8sv1.VolumeMount,
	subPath string,
) k8sv1.Container {
	// Use an internal mount path to avoid conflicts with third-party webhooks
	// that may inject volumes at the same containerPath (e.g., IRSA, Vault).
	// virtiofsd will serve from this internal path.
	internalMountPath := fmt.Sprintf("/virtiofs-data/%s", volume.Name)

	socketPath := virtiofs.VirtioFSSocketPath(volume.Name)

	args := []string{
		fmt.Sprintf("--socket-path=%s", socketPath),
		fmt.Sprintf("--shared-dir=%s", internalMountPath),
		"--sandbox=none",
		"--cache=auto",
		"--migration-on-error=guest-error",
		"--migration-mode=find-paths",
	}

	// Volume mounts for the virtiofs container
	volumeMounts := []k8sv1.VolumeMount{
		// Socket directory shared with compute container
		{
			Name:      virtiofs.VirtioFSContainers,
			MountPath: virtiofs.VirtioFSContainersMountBaseDir,
		},
		// The actual volume containing the data - mounted at internal path
		// to avoid conflicts with third-party webhook injections
		{
			Name:      sourceMount.Name,
			MountPath: internalMountPath,
			SubPath:   subPath,
			ReadOnly:  true,
		},
	}

	// Get resources based on VMI QOS settings
	dedicatedCPUs := vmi.IsCPUDedicated()
	guaranteedQOS := dedicatedCPUs || vmi.WantsToHaveQOSGuaranteed()
	resources := virtiofs.ResourcesForVirtioFSContainer(dedicatedCPUs, guaranteedQOS, m.ClusterConfig)

	return k8sv1.Container{
		Name:            virtiofs.ContainerPathVirtiofsContainerName(volume.Name),
		Image:           image,
		ImagePullPolicy: k8sv1.PullIfNotPresent,
		Command:         []string{"/usr/libexec/virtiofsd"},
		Args:            args,
		VolumeMounts:    volumeMounts,
		Resources:       resources,
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:                pointer.P(int64(util.NonRootUID)),
			RunAsGroup:               pointer.P(int64(util.NonRootUID)),
			RunAsNonRoot:             pointer.P(true),
			AllowPrivilegeEscalation: pointer.P(false),
			Capabilities: &k8sv1.Capabilities{
				Drop: []k8sv1.Capability{"ALL"},
			},
		},
	}
}

func (m *VirtLauncherPodMutator) getVMIFromOwnerRef(pod *k8sv1.Pod) (*v1.VirtualMachineInstance, error) {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == v1.VirtualMachineInstanceGroupVersionKind.Kind &&
			ownerRef.APIVersion == v1.VirtualMachineInstanceGroupVersionKind.GroupVersion().String() {
			vmi, err := m.VirtClient.VirtualMachineInstance(pod.Namespace).Get(context.Background(), ownerRef.Name, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get VMI %s/%s: %w", pod.Namespace, ownerRef.Name, err)
			}
			return vmi, nil
		}
	}
	return nil, nil
}

func isVirtLauncherPod(pod *k8sv1.Pod) bool {
	if pod.Labels == nil {
		return false
	}
	return pod.Labels[v1.AppLabel] == "virt-launcher"
}

func findContainer(pod *k8sv1.Pod, name string) *k8sv1.Container {
	for i := range pod.Spec.Containers {
		if pod.Spec.Containers[i].Name == name {
			return &pod.Spec.Containers[i]
		}
	}
	return nil
}

func allowedResponse() *admissionv1.AdmissionResponse {
	return &admissionv1.AdmissionResponse{
		Allowed: true,
	}
}
