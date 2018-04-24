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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package services

import (
	"path/filepath"
	"strconv"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/registry-disk"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VirtualMachine) *kubev1.Pod
}

type templateService struct {
	launcherImage   string
	virtShareDir    string
	imagePullSecret string
}

func (t *templateService) RenderLaunchManifest(vm *v1.VirtualMachine) *kubev1.Pod {
	precond.MustNotBeNil(vm)
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())

	initialDelaySeconds := 2
	timeoutSeconds := 5
	periodSeconds := 2
	successThreshold := 1
	failureThreshold := 5

	var volumes []kubev1.Volume
	var userId int64 = 0
	var privileged = true
	var volumesMounts []kubev1.VolumeMount
	var imagePullSecrets []kubev1.LocalObjectReference

	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vm.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vm.Spec.TerminationGracePeriodSeconds
	}

	volumesMounts = append(volumesMounts, kubev1.VolumeMount{
		Name:      "virt-share-dir",
		MountPath: t.virtShareDir,
	})
	volumesMounts = append(volumesMounts, kubev1.VolumeMount{
		Name:      "libvirt-runtime",
		MountPath: "/var/run/libvirt",
	})
	for _, volume := range vm.Spec.Volumes {
		volumeMount := kubev1.VolumeMount{
			Name:      volume.Name,
			MountPath: filepath.Join("/var/run/kubevirt-private", "vm-disks", volume.Name),
		}
		if volume.PersistentVolumeClaim != nil {
			volumesMounts = append(volumesMounts, volumeMount)
			volumes = append(volumes, kubev1.Volume{
				Name: volume.Name,
				VolumeSource: kubev1.VolumeSource{
					PersistentVolumeClaim: volume.PersistentVolumeClaim,
				},
			})
		}
		if volume.Ephemeral != nil {
			volumesMounts = append(volumesMounts, volumeMount)
			volumes = append(volumes, kubev1.Volume{
				Name: volume.Name,
				VolumeSource: kubev1.VolumeSource{
					PersistentVolumeClaim: volume.Ephemeral.PersistentVolumeClaim,
				},
			})
		}
		if volume.RegistryDisk != nil && volume.RegistryDisk.ImagePullSecret != "" {
			imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, kubev1.LocalObjectReference{
				Name: volume.RegistryDisk.ImagePullSecret,
			})
		}
	}

	if t.imagePullSecret != "" {
		imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, kubev1.LocalObjectReference{
			Name: t.imagePullSecret,
		})
	}

	// Pad the virt-launcher grace period.
	// Ideally we want virt-handler to handle tearing down
	// the vm without virt-launcher's termination forcing
	// the vm down.
	gracePeriodSeconds = gracePeriodSeconds + int64(15)
	gracePeriodKillAfter := gracePeriodSeconds + int64(15)

	// Consider CPU and memory requests and limits for pod scheduling
	resources := kubev1.ResourceRequirements{}
	vmResources := vm.Spec.Domain.Resources

	resources.Requests = make(kubev1.ResourceList)

	// Copy vm resources requests to a container
	for key, value := range vmResources.Requests {
		resources.Requests[key] = value
	}

	// Copy vm resources limits to a container
	if vmResources.Limits != nil {
		resources.Limits = make(kubev1.ResourceList)
	}
	for key, value := range vmResources.Limits {
		resources.Limits[key] = value
	}

	// Add memory overhead
	setMemoryOverhead(vm.Spec.Domain, &resources)

	// VM target container
	container := kubev1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: kubev1.PullIfNotPresent,
		// Privileged mode is required for /dev/kvm and the
		// ability to create macvtap devices
		SecurityContext: &kubev1.SecurityContext{
			RunAsUser:  &userId,
			Privileged: &privileged,
		},
		Command: []string{"/entrypoint.sh",
			"--qemu-timeout", "5m",
			"--name", domain,
			"--namespace", namespace,
			"--kubevirt-share-dir", t.virtShareDir,
			"--readiness-file", "/tmp/healthy",
			"--grace-period-seconds", strconv.Itoa(int(gracePeriodSeconds)),
		},
		VolumeMounts: volumesMounts,
		ReadinessProbe: &kubev1.Probe{
			Handler: kubev1.Handler{
				Exec: &kubev1.ExecAction{
					Command: []string{
						"cat",
						"/tmp/healthy",
					},
				},
			},
			InitialDelaySeconds: int32(initialDelaySeconds),
			PeriodSeconds:       int32(periodSeconds),
			TimeoutSeconds:      int32(timeoutSeconds),
			SuccessThreshold:    int32(successThreshold),
			FailureThreshold:    int32(failureThreshold),
		},
		Resources: resources,
	}

	containers := registrydisk.GenerateContainers(vm, "libvirt-runtime", "/var/run/libvirt")

	volumes = append(volumes, kubev1.Volume{
		Name: "virt-share-dir",
		VolumeSource: kubev1.VolumeSource{
			HostPath: &kubev1.HostPathVolumeSource{
				Path: t.virtShareDir,
			},
		},
	})
	volumes = append(volumes, kubev1.Volume{
		Name: "libvirt-runtime",
		VolumeSource: kubev1.VolumeSource{
			EmptyDir: &kubev1.EmptyDirVolumeSource{},
		},
	})

	nodeSelector := map[string]string{}
	for k, v := range vm.Spec.NodeSelector {
		nodeSelector[k] = v

	}
	nodeSelector[v1.NodeSchedulable] = "true"

	podLabels := map[string]string{}

	for k, v := range vm.Labels {
		podLabels[k] = v
	}
	podLabels[v1.AppLabel] = "virt-launcher"
	podLabels[v1.DomainLabel] = domain

	containers = append(containers, container)

	// TODO use constants for podLabels
	pod := kubev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-",
			Labels:       podLabels,
			Annotations: map[string]string{
				v1.CreatedByAnnotation: string(vm.UID),
				v1.OwnedByAnnotation:   "virt-controller",
			},
		},
		Spec: kubev1.PodSpec{
			SecurityContext: &kubev1.PodSecurityContext{
				RunAsUser: &userId,
			},
			TerminationGracePeriodSeconds: &gracePeriodKillAfter,
			RestartPolicy:                 kubev1.RestartPolicyNever,
			Containers:                    containers,
			NodeSelector:                  nodeSelector,
			Volumes:                       volumes,
			ImagePullSecrets:              imagePullSecrets,
		},
	}

	if vm.Spec.Affinity != nil {
		pod.Spec.Affinity = &kubev1.Affinity{}

		if vm.Spec.Affinity.NodeAffinity != nil {
			pod.Spec.Affinity.NodeAffinity = vm.Spec.Affinity.NodeAffinity
		}
	}

	return &pod
}

func appendUniqueImagePullSecret(secrets []kubev1.LocalObjectReference, newsecret kubev1.LocalObjectReference) []kubev1.LocalObjectReference {
	for _, oldsecret := range secrets {
		if oldsecret == newsecret {
			return secrets
		}
	}
	return append(secrets, newsecret)
}

// setMemoryOverhead computes the estimation of total
// memory needed for the domain to operate properly.
// This includes the memory needed for the guest and memory
// for Qemu and OS overhead.
//
// The return values are requested memory and limit memory quantities
//
// Note: This is the best estimation we were able to come up with
//       and is still not 100% accurate
func setMemoryOverhead(domain v1.DomainSpec, resources *kubev1.ResourceRequirements) error {
	vmMemoryReq := domain.Resources.Requests.Memory()

	overhead := resource.NewScaledQuantity(0, resource.Kilo)

	// Add the memory needed for pagetables (one bit for every 512b of RAM size)
	pagetableMemory := resource.NewScaledQuantity(vmMemoryReq.ScaledValue(resource.Kilo), resource.Kilo)
	pagetableMemory.Set(pagetableMemory.Value() / 512)
	overhead.Add(*pagetableMemory)

	// Add fixed overhead for shared libraries and such
	// TODO account for the overhead of kubevirt components running in the pod
	overhead.Add(resource.MustParse("64M"))

	// Add CPU table overhead (8 MiB per vCPU and 8 MiB per IO thread)
	// overhead per vcpu in MiB
	coresMemory := uint32(8)
	if domain.CPU != nil {
		coresMemory *= domain.CPU.Cores
	}
	overhead.Add(resource.MustParse(strconv.Itoa(int(coresMemory)) + "Mi"))

	// static overhead for IOThread
	overhead.Add(resource.MustParse("8Mi"))

	// Add video RAM overhead
	overhead.Add(resource.MustParse("16Mi"))

	// Add overhead to memory request
	memoryRequest := resources.Requests[kubev1.ResourceMemory]
	memoryRequest.Add(*overhead)
	resources.Requests[kubev1.ResourceMemory] = memoryRequest

	// Add overhead to memory limits, if exists
	if memoryLimit, ok := resources.Limits[kubev1.ResourceMemory]; ok {
		memoryLimit.Add(*overhead)
		resources.Limits[kubev1.ResourceMemory] = memoryLimit
	}

	return nil
}

func NewTemplateService(launcherImage string, virtShareDir string, imagePullSecret string) TemplateService {
	precond.MustNotBeEmpty(launcherImage)
	svc := templateService{
		launcherImage:   launcherImage,
		virtShareDir:    virtShareDir,
		imagePullSecret: imagePullSecret,
	}
	return &svc
}
