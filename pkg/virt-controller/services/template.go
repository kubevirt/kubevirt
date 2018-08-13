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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package services

import (
	"path/filepath"
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/registry-disk"
)

type TemplateService interface {
	RenderLaunchManifest(*v1.VirtualMachineInstance) (*k8sv1.Pod, error)
}

type templateService struct {
	launcherImage   string
	virtShareDir    string
	imagePullSecret string
	clusterConfig   *config.ClusterConfig
}

func (t *templateService) RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	precond.MustNotBeNil(vmi)
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())

	initialDelaySeconds := 2
	timeoutSeconds := 5
	periodSeconds := 2
	successThreshold := 1
	failureThreshold := 5

	var volumes []k8sv1.Volume
	var userId int64 = 0
	var privileged bool = true
	var volumesMounts []k8sv1.VolumeMount
	var imagePullSecrets []k8sv1.LocalObjectReference

	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}

	volumesMounts = append(volumesMounts, k8sv1.VolumeMount{
		Name:      "virt-share-dir",
		MountPath: t.virtShareDir,
	})
	volumesMounts = append(volumesMounts, k8sv1.VolumeMount{
		Name:      "libvirt-runtime",
		MountPath: "/var/run/libvirt",
	})
	for _, volume := range vmi.Spec.Volumes {
		volumeMount := k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: filepath.Join("/var/run/kubevirt-private", "vmi-disks", volume.Name),
		}
		if volume.PersistentVolumeClaim != nil {
			volumesMounts = append(volumesMounts, volumeMount)
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: volume.PersistentVolumeClaim,
				},
			})
		}
		if volume.Ephemeral != nil {
			volumesMounts = append(volumesMounts, volumeMount)
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: volume.Ephemeral.PersistentVolumeClaim,
				},
			})
		}
		if volume.RegistryDisk != nil && volume.RegistryDisk.ImagePullSecret != "" {
			imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
				Name: volume.RegistryDisk.ImagePullSecret,
			})
		}
	}

	if t.imagePullSecret != "" {
		imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
			Name: t.imagePullSecret,
		})
	}

	// Pad the virt-launcher grace period.
	// Ideally we want virt-handler to handle tearing down
	// the vmi without virt-launcher's termination forcing
	// the vmi down.
	gracePeriodSeconds = gracePeriodSeconds + int64(15)
	gracePeriodKillAfter := gracePeriodSeconds + int64(15)

	// Get memory overhead
	memoryOverhead := getMemoryOverhead(vmi.Spec.Domain)

	// Consider CPU and memory requests and limits for pod scheduling
	resources := k8sv1.ResourceRequirements{}
	vmiResources := vmi.Spec.Domain.Resources

	resources.Requests = make(k8sv1.ResourceList)

	// Copy vmi resources requests to a container
	for key, value := range vmiResources.Requests {
		resources.Requests[key] = value
	}

	// Copy vmi resources limits to a container
	if vmiResources.Limits != nil {
		resources.Limits = make(k8sv1.ResourceList)
	}

	for key, value := range vmiResources.Limits {
		resources.Limits[key] = value
	}

	// Consider hugepages resource for pod scheduling
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		if resources.Limits == nil {
			resources.Limits = make(k8sv1.ResourceList)
		}

		hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + vmi.Spec.Domain.Memory.Hugepages.PageSize)
		resources.Requests[hugepageType] = resources.Requests[k8sv1.ResourceMemory]
		resources.Limits[hugepageType] = resources.Requests[k8sv1.ResourceMemory]

		// Configure hugepages mount on a pod
		volumesMounts = append(volumesMounts, k8sv1.VolumeMount{
			Name:      "hugepages",
			MountPath: filepath.Join("/dev/hugepages"),
		})
		volumes = append(volumes, k8sv1.Volume{
			Name: "hugepages",
			VolumeSource: k8sv1.VolumeSource{
				EmptyDir: &k8sv1.EmptyDirVolumeSource{
					Medium: k8sv1.StorageMediumHugePages,
				},
			},
		})

		// Set requested memory equals to overhead memory
		resources.Requests[k8sv1.ResourceMemory] = *memoryOverhead
		if _, ok := resources.Limits[k8sv1.ResourceMemory]; ok {
			resources.Limits[k8sv1.ResourceMemory] = *memoryOverhead
		}
	} else {
		// Add overhead memory
		memoryRequest := resources.Requests[k8sv1.ResourceMemory]
		if !vmi.Spec.Domain.Resources.OvercommitGuestOverhead {
			memoryRequest.Add(*memoryOverhead)
		}
		resources.Requests[k8sv1.ResourceMemory] = memoryRequest

		if memoryLimit, ok := resources.Limits[k8sv1.ResourceMemory]; ok {
			memoryLimit.Add(*memoryOverhead)
			resources.Limits[k8sv1.ResourceMemory] = memoryLimit
		}
	}

	command := []string{"/usr/share/kubevirt/virt-launcher/entrypoint.sh",
		"--qemu-timeout", "5m",
		"--name", domain,
		"--namespace", namespace,
		"--kubevirt-share-dir", t.virtShareDir,
		"--readiness-file", "/tmp/healthy",
		"--grace-period-seconds", strconv.Itoa(int(gracePeriodSeconds)),
	}

	imagePullPolicy, err := t.clusterConfig.GetImagePullPolicy()
	if err != nil {
		return nil, err
	}

	// VirtualMachineInstance target container
	container := k8sv1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: imagePullPolicy,
		// Privileged mode is required for /dev/kvm and the
		// ability to create macvtap devices
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:  &userId,
			Privileged: &privileged,
		},
		Command:      command,
		VolumeMounts: volumesMounts,
		ReadinessProbe: &k8sv1.Probe{
			Handler: k8sv1.Handler{
				Exec: &k8sv1.ExecAction{
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

	containers := registrydisk.GenerateContainers(vmi, "libvirt-runtime", "/var/run/libvirt")

	volumes = append(volumes, k8sv1.Volume{
		Name: "virt-share-dir",
		VolumeSource: k8sv1.VolumeSource{
			HostPath: &k8sv1.HostPathVolumeSource{
				Path: t.virtShareDir,
			},
		},
	})
	volumes = append(volumes, k8sv1.Volume{
		Name: "libvirt-runtime",
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})

	nodeSelector := map[string]string{}
	for k, v := range vmi.Spec.NodeSelector {
		nodeSelector[k] = v

	}
	nodeSelector[v1.NodeSchedulable] = "true"

	podLabels := map[string]string{}

	for k, v := range vmi.Labels {
		podLabels[k] = v
	}
	podLabels[v1.AppLabel] = "virt-launcher"
	podLabels[v1.CreatedByLabel] = string(vmi.UID)

	containers = append(containers, container)

	hostName := vmi.Name
	if vmi.Spec.Hostname != "" {
		hostName = vmi.Spec.Hostname
	}

	// TODO use constants for podLabels
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-",
			Labels:       podLabels,
			Annotations: map[string]string{
				v1.DomainAnnotation:  domain,
				v1.OwnedByAnnotation: "virt-controller",
			},
		},
		Spec: k8sv1.PodSpec{
			Hostname:  hostName,
			Subdomain: vmi.Spec.Subdomain,
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsUser: &userId,
			},
			TerminationGracePeriodSeconds: &gracePeriodKillAfter,
			RestartPolicy:                 k8sv1.RestartPolicyNever,
			Containers:                    containers,
			NodeSelector:                  nodeSelector,
			Volumes:                       volumes,
			ImagePullSecrets:              imagePullSecrets,
		},
	}

	if vmi.Spec.Affinity != nil {
		pod.Spec.Affinity = &k8sv1.Affinity{}

		if vmi.Spec.Affinity.NodeAffinity != nil {
			pod.Spec.Affinity.NodeAffinity = vmi.Spec.Affinity.NodeAffinity
		}

		if vmi.Spec.Affinity.PodAffinity != nil {
			pod.Spec.Affinity.PodAffinity = vmi.Spec.Affinity.PodAffinity
		}

		if vmi.Spec.Affinity.PodAntiAffinity != nil {
			pod.Spec.Affinity.PodAntiAffinity = vmi.Spec.Affinity.PodAntiAffinity
		}
	}

	return &pod, nil
}

func appendUniqueImagePullSecret(secrets []k8sv1.LocalObjectReference, newsecret k8sv1.LocalObjectReference) []k8sv1.LocalObjectReference {
	for _, oldsecret := range secrets {
		if oldsecret == newsecret {
			return secrets
		}
	}
	return append(secrets, newsecret)
}

// getMemoryOverhead computes the estimation of total
// memory needed for the domain to operate properly.
// This includes the memory needed for the guest and memory
// for Qemu and OS overhead.
//
// The return value is overhead memory quantity
//
// Note: This is the best estimation we were able to come up with
//       and is still not 100% accurate
func getMemoryOverhead(domain v1.DomainSpec) *resource.Quantity {
	vmiMemoryReq := domain.Resources.Requests.Memory()

	overhead := resource.NewScaledQuantity(0, resource.Kilo)

	// Add the memory needed for pagetables (one bit for every 512b of RAM size)
	pagetableMemory := resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo)
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
	if domain.Devices.AutoattachGraphicsDevice == nil || *domain.Devices.AutoattachGraphicsDevice == true {
		overhead.Add(resource.MustParse("16Mi"))
	}

	return overhead
}

func NewTemplateService(launcherImage string, virtShareDir string, imagePullSecret string, clusterConfig *config.ClusterConfig) TemplateService {
	precond.MustNotBeEmpty(launcherImage)
	svc := templateService{
		launcherImage:   launcherImage,
		virtShareDir:    virtShareDir,
		imagePullSecret: imagePullSecret,
		clusterConfig:   clusterConfig,
	}
	return &svc
}
