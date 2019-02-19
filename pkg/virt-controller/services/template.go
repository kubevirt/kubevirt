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
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-config"
)

const configMapName = "kubevirt-config"
const UseEmulationKey = "debug.useEmulation"
const ImagePullPolicyKey = "dev.imagePullPolicy"
const LessPVCSpaceTolerationKey = "pvc-tolerate-less-space-up-to-percent"
const KvmDevice = "devices.kubevirt.io/kvm"
const TunDevice = "devices.kubevirt.io/tun"
const VhostNetDevice = "devices.kubevirt.io/vhost-net"

const CAP_NET_ADMIN = "NET_ADMIN"
const CAP_SYS_NICE = "SYS_NICE"

// LibvirtStartupDelay is added to custom liveness and readiness probes initial delay value.
// Libvirt needs roughly 10 seconds to start.
const LibvirtStartupDelay = 10

//This is a perfix for node feature discovery, used in a NodeSelector on the pod
//to match a VirtualMachineInstance CPU model(Family) to nodes that support this model.
const NFD_CPU_MODEL_PREFIX = "feature.node.kubernetes.io/cpu-model-"

const MULTUS_RESOURCE_NAME_ANNOTATION = "k8s.v1.cni.cncf.io/resourceName"
const MULTUS_DEFAULT_NETWORK_CNI_ANNOTATION = "v1.multus-cni.io/default-network"

type TemplateService interface {
	RenderLaunchManifest(*v1.VirtualMachineInstance) (*k8sv1.Pod, error)
}

type templateService struct {
	launcherImage              string
	virtShareDir               string
	ephemeralDiskDir           string
	imagePullSecret            string
	configMapStore             cache.Store
	persistentVolumeClaimStore cache.Store
	virtClient                 kubecli.KubevirtClient
}

type PvcNotFoundError error

func getConfigMapEntry(store cache.Store, key string) (string, error) {

	namespace, err := util.GetNamespace()
	if err != nil {
		return "", err
	}

	if obj, exists, err := store.GetByKey(namespace + "/" + configMapName); err != nil {
		return "", err
	} else if !exists {
		return "", nil
	} else {
		return obj.(*k8sv1.ConfigMap).Data[key], nil
	}
}

func IsEmulationAllowed(store cache.Store) (useEmulation bool, err error) {
	var value string
	value, err = getConfigMapEntry(store, UseEmulationKey)
	if strings.ToLower(value) == "true" {
		useEmulation = true
	}
	return
}

func GetImagePullPolicy(store cache.Store) (policy k8sv1.PullPolicy, err error) {
	var value string
	if value, err = getConfigMapEntry(store, ImagePullPolicyKey); err != nil || value == "" {
		policy = k8sv1.PullIfNotPresent // Default if not specified
	} else {
		switch value {
		case "Always":
			policy = k8sv1.PullAlways
		case "Never":
			policy = k8sv1.PullNever
		case "IfNotPresent":
			policy = k8sv1.PullIfNotPresent
		default:
			err = fmt.Errorf("Invalid ImagePullPolicy in ConfigMap: %s", value)
		}
	}
	return
}

func GetlessPVCSpaceToleration(store cache.Store) (toleration int, err error) {
	var value string
	if value, err = getConfigMapEntry(store, LessPVCSpaceTolerationKey); err != nil || value == "" {
		toleration = 10 // Default if not specified
	} else {
		toleration, err = strconv.Atoi(value)
		if err != nil || toleration < 0 || toleration > 100 {
			err = fmt.Errorf("Invalid lessPVCSpaceToleration in ConfigMap: %s", value)
			return
		}
	}
	return
}

func isSRIOVVmi(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.SRIOV != nil {
			return true
		}
	}
	return false
}

func IsCPUNodeDiscoveryEnabled(store cache.Store) bool {
	if value, err := getConfigMapEntry(store, virtconfig.FeatureGatesKey); err != nil {
		return false
	} else if strings.Contains(value, virtconfig.CPUNodeDiscoveryGate) {
		return true
	}
	return false
}

func CPUModelLabelFromCPUModel(vmi *v1.VirtualMachineInstance) (label string, err error) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		err = fmt.Errorf("Cannot create CPU Model label, vmi spec is mising CPU model")
		return
	}
	label = NFD_CPU_MODEL_PREFIX + vmi.Spec.Domain.CPU.Model
	return
}

// Request a resource by name. This function bumps the number of resources,
// both its limits and requests attributes.
//
// If we were operating with a regular resource (CPU, memory, network
// bandwidth), we would need to take care of QoS. For example,
// https://kubernetes.io/docs/tasks/configure-pod-container/quality-service-pod/#create-a-pod-that-gets-assigned-a-qos-class-of-guaranteed
// explains that when Limits are set but Requests are not then scheduler
// assumes that Requests are the same as Limits for a particular resource.
//
// But this function is not called for this standard resources but for
// resources managed by device plugins. The device plugin design document says
// the following on the matter:
// https://github.com/kubernetes/community/blob/master/contributors/design-proposals/resource-management/device-plugin.md#end-user-story
//
// ```
// Devices can be selected using the same process as for OIRs in the pod spec.
// Devices have no impact on QOS. However, for the alpha, we expect the request
// to have limits == requests.
// ```
//
// Which suggests that, for resources managed by device plugins, 1) limits
// should be equal to requests; and 2) QoS rules do not apply.
//
// Hence we don't copy Limits value to Requests if the latter is missing.
func requestResource(resources *k8sv1.ResourceRequirements, resourceName string) {
	name := k8sv1.ResourceName(resourceName)

	// assume resources are countable, singular, and cannot be divided
	unitQuantity := *resource.NewQuantity(1, resource.DecimalSI)

	// Fill in limits
	val, ok := resources.Limits[name]
	if ok {
		val.Add(unitQuantity)
		resources.Limits[name] = val
	} else {
		resources.Limits[name] = unitQuantity
	}

	// Fill in requests
	val, ok = resources.Requests[name]
	if ok {
		val.Add(unitQuantity)
		resources.Requests[name] = val
	} else {
		resources.Requests[name] = unitQuantity
	}
}

func (t *templateService) RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	precond.MustNotBeNil(vmi)
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	nodeSelector := map[string]string{}

	initialDelaySeconds := 2
	timeoutSeconds := 5
	periodSeconds := 2
	successThreshold := 1
	failureThreshold := 5

	var volumes []k8sv1.Volume
	var volumeDevices []k8sv1.VolumeDevice
	var userId int64 = 0
	// Privileged mode is disabled by default.
	var privileged bool = false
	var volumeMounts []k8sv1.VolumeMount
	var imagePullSecrets []k8sv1.LocalObjectReference

	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "ephemeral-disks",
		MountPath: t.ephemeralDiskDir,
	})

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "virt-share-dir",
		MountPath: t.virtShareDir,
	})

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "libvirt-runtime",
		MountPath: "/var/run/libvirt",
	})

	if isSRIOVVmi(vmi) {
		// libvirt needs this volume to unbind the device from kernel
		// driver, and register it with vfio userspace driver
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      "pci-bus",
			MountPath: "/sys/bus/pci/",
		})
		volumes = append(volumes, k8sv1.Volume{
			Name: "pci-bus",
			VolumeSource: k8sv1.VolumeSource{
				HostPath: &k8sv1.HostPathVolumeSource{
					Path: "/sys/bus/pci/",
				},
			},
		})

		// libvirt needs this volume to determine iommu group assigned
		// to the device
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      "pci-devices",
			MountPath: "/sys/devices/",
		})
		volumes = append(volumes, k8sv1.Volume{
			Name: "pci-devices",
			VolumeSource: k8sv1.VolumeSource{
				HostPath: &k8sv1.HostPathVolumeSource{
					Path: "/sys/devices/",
				},
			},
		})

		// libvirt uses vfio-pci to pass host devices through
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      "dev-vfio",
			MountPath: "/dev/vfio/",
		})
		volumes = append(volumes, k8sv1.Volume{
			Name: "dev-vfio",
			VolumeSource: k8sv1.VolumeSource{
				HostPath: &k8sv1.HostPathVolumeSource{
					Path: "/dev/vfio/",
				},
			},
		})

		// todo: revisit when SR-IOV DP registers /dev/vfio/NN with pod
		// device group:
		// https://github.com/intel/sriov-network-device-plugin/pull/26
		//
		// Run virt-launcher compute container privileged to allow qemu
		// to open /dev/vfio/NN for PCI passthrough
		privileged = true
	}

	serviceAccountName := ""

	for _, volume := range vmi.Spec.Volumes {
		volumeMount := k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: filepath.Join("/var/run/kubevirt-private", "vmi-disks", volume.Name),
		}
		if volume.PersistentVolumeClaim != nil {
			logger := log.DefaultLogger()
			claimName := volume.PersistentVolumeClaim.ClaimName
			_, exists, isBlock, err := types.IsPVCBlockFromStore(t.persistentVolumeClaimStore, namespace, claimName)
			if err != nil {
				logger.Errorf("error getting PVC: %v", claimName)
				return nil, err
			} else if !exists {
				logger.Errorf("didn't find PVC %v", claimName)
				return nil, PvcNotFoundError(fmt.Errorf("didn't find PVC %v", claimName))
			} else if isBlock {
				devicePath := filepath.Join(string(filepath.Separator), "dev", volume.Name)
				device := k8sv1.VolumeDevice{
					Name:       volume.Name,
					DevicePath: devicePath,
				}
				volumeDevices = append(volumeDevices, device)
			} else {
				volumeMounts = append(volumeMounts, volumeMount)
			}
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: volume.PersistentVolumeClaim,
				},
			})
		}
		if volume.Ephemeral != nil {
			volumeMounts = append(volumeMounts, volumeMount)
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: volume.Ephemeral.PersistentVolumeClaim,
				},
			})
		}
		if volume.ContainerDisk != nil && volume.ContainerDisk.ImagePullSecret != "" {
			imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
				Name: volume.ContainerDisk.ImagePullSecret,
			})
		}
		if volume.HostDisk != nil {
			var hostPathType k8sv1.HostPathType

			switch hostType := volume.HostDisk.Type; hostType {
			case v1.HostDiskExists:
				hostPathType = k8sv1.HostPathDirectory
			case v1.HostDiskExistsOrCreate:
				hostPathType = k8sv1.HostPathDirectoryOrCreate
			}

			volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
				Name:      volume.Name,
				MountPath: filepath.Dir(volume.HostDisk.Path),
			})
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					HostPath: &k8sv1.HostPathVolumeSource{
						Path: filepath.Dir(volume.HostDisk.Path),
						Type: &hostPathType,
					},
				},
			})
		}
		if volume.DataVolume != nil {
			volumeMounts = append(volumeMounts, volumeMount)
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: volume.DataVolume.Name,
					},
				},
			})
		}
		if volume.ConfigMap != nil {
			// attach a ConfigMap to the pod
			volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
				Name:      volume.Name,
				MountPath: filepath.Join(config.ConfigMapSourceDir, volume.Name),
				ReadOnly:  true,
			})
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					ConfigMap: &k8sv1.ConfigMapVolumeSource{
						LocalObjectReference: volume.ConfigMap.LocalObjectReference,
						Optional:             volume.ConfigMap.Optional,
					},
				},
			})
		}

		if volume.Secret != nil {
			// attach a Secret to the pod
			volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
				Name:      volume.Name,
				MountPath: filepath.Join(config.SecretSourceDir, volume.Name),
				ReadOnly:  true,
			})
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					Secret: &k8sv1.SecretVolumeSource{
						SecretName: volume.Secret.SecretName,
						Optional:   volume.Secret.Optional,
					},
				},
			})
		}

		if volume.ServiceAccount != nil {
			serviceAccountName = volume.ServiceAccount.ServiceAccountName
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
	resources.Limits = make(k8sv1.ResourceList)

	// Copy vmi resources requests to a container
	for key, value := range vmiResources.Requests {
		resources.Requests[key] = value
	}

	// Copy vmi resources limits to a container
	for key, value := range vmiResources.Limits {
		resources.Limits[key] = value
	}

	// Consider hugepages resource for pod scheduling
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil {
		hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + vmi.Spec.Domain.Memory.Hugepages.PageSize)
		resources.Requests[hugepageType] = resources.Requests[k8sv1.ResourceMemory]
		resources.Limits[hugepageType] = resources.Requests[k8sv1.ResourceMemory]

		// Configure hugepages mount on a pod
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
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

	// Read requested hookSidecars from VMI meta
	requestedHookSidecarList, err := hooks.UnmarshalHookSidecarList(vmi)
	if err != nil {
		return nil, err
	}

	if len(requestedHookSidecarList) != 0 {
		volumes = append(volumes, k8sv1.Volume{
			Name: "hook-sidecar-sockets",
			VolumeSource: k8sv1.VolumeSource{
				EmptyDir: &k8sv1.EmptyDirVolumeSource{},
			},
		})
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      "hook-sidecar-sockets",
			MountPath: hooks.HookSocketsSharedDirectory,
		})
	}

	// Handle CPU pinning
	if vmi.IsCPUDedicated() {
		// schedule only on nodes with a running cpu manager
		nodeSelector[v1.CPUManager] = "true"

		vcpus := hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU)

		if vcpus != 0 {
			resources.Limits[k8sv1.ResourceCPU] = *resource.NewQuantity(vcpus, resource.BinarySI)
		} else {
			if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
				resources.Requests[k8sv1.ResourceCPU] = cpuLimit
			} else if cpuRequest, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
				resources.Limits[k8sv1.ResourceCPU] = cpuRequest
			}
		}
		resources.Limits[k8sv1.ResourceMemory] = *resources.Requests.Memory()
	}

	lessPVCSpaceToleration, err := GetlessPVCSpaceToleration(t.configMapStore)
	if err != nil {
		return nil, err
	}

	command := []string{"/usr/bin/virt-launcher",
		"--qemu-timeout", "5m",
		"--name", domain,
		"--uid", string(vmi.UID),
		"--namespace", namespace,
		"--kubevirt-share-dir", t.virtShareDir,
		"--ephemeral-disk-dir", t.ephemeralDiskDir,
		"--readiness-file", "/var/run/kubevirt-infra/healthy",
		"--grace-period-seconds", strconv.Itoa(int(gracePeriodSeconds)),
		"--hook-sidecars", strconv.Itoa(len(requestedHookSidecarList)),
		"--less-pvc-space-toleration", strconv.Itoa(lessPVCSpaceToleration),
	}

	useEmulation, err := IsEmulationAllowed(t.configMapStore)
	if err != nil {
		return nil, err
	}

	imagePullPolicy, err := GetImagePullPolicy(t.configMapStore)
	if err != nil {
		return nil, err
	}

	if resources.Limits == nil {
		resources.Limits = make(k8sv1.ResourceList)
	}

	extraResources := getRequiredResources(vmi, useEmulation)
	for key, val := range extraResources {
		resources.Limits[key] = val
	}

	if useEmulation {
		command = append(command, "--use-emulation")
	} else {
		resources.Limits[KvmDevice] = resource.MustParse("1")
	}

	// Add ports from interfaces to the pod manifest
	ports := getPortsFromVMI(vmi)

	capabilities := getRequiredCapabilities(vmi)

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "infra-ready-mount",
		MountPath: "/var/run/kubevirt-infra",
	})

	defaultReadinessProbe := &k8sv1.Probe{
		Handler: k8sv1.Handler{
			Exec: &k8sv1.ExecAction{
				Command: []string{
					"cat",
					"/var/run/kubevirt-infra/healthy",
				},
			},
		},
		InitialDelaySeconds: int32(initialDelaySeconds),
		PeriodSeconds:       int32(periodSeconds),
		TimeoutSeconds:      int32(timeoutSeconds),
		SuccessThreshold:    int32(successThreshold),
		FailureThreshold:    int32(failureThreshold),
	}

	volumes = append(volumes, k8sv1.Volume{Name: "infra-ready-mount", VolumeSource: k8sv1.VolumeSource{EmptyDir: &k8sv1.EmptyDirVolumeSource{}}})

	containers := containerdisk.GenerateContainers(vmi, "ephemeral-disks", t.ephemeralDiskDir)

	networkToResourceMap, err := getNetworkToResourceMap(t.virtClient, vmi)
	if err != nil {
		return nil, err
	}

	// Register resource requests and limits corresponding to attached multus networks.
	// TODO(ihar) remove when we adopt Multus mutating webhook that handles the job.
	for _, resourceName := range networkToResourceMap {
		if resourceName != "" {
			requestResource(&resources, resourceName)
		}
	}

	// VirtualMachineInstance target container
	container := k8sv1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: imagePullPolicy,
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:  &userId,
			Privileged: &privileged,
			Capabilities: &k8sv1.Capabilities{
				Add: capabilities,
			},
		},
		Command:        command,
		VolumeDevices:  volumeDevices,
		VolumeMounts:   volumeMounts,
		Resources:      resources,
		Ports:          ports,
		ReadinessProbe: defaultReadinessProbe,
	}

	if vmi.Spec.ReadinessProbe != nil {
		container.ReadinessProbe = copyProbe(vmi.Spec.ReadinessProbe)
		container.ReadinessProbe.InitialDelaySeconds = container.ReadinessProbe.InitialDelaySeconds + LibvirtStartupDelay
	}

	if vmi.Spec.LivenessProbe != nil {
		container.LivenessProbe = copyProbe(vmi.Spec.LivenessProbe)
		container.LivenessProbe.InitialDelaySeconds = container.LivenessProbe.InitialDelaySeconds + LibvirtStartupDelay
	}

	for networkName, resourceName := range networkToResourceMap {
		varName := fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", networkName)
		container.Env = append(container.Env, k8sv1.EnvVar{Name: varName, Value: resourceName})
	}

	containers = append(containers, container)

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
	volumes = append(volumes, k8sv1.Volume{
		Name: "ephemeral-disks",
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})

	for k, v := range vmi.Spec.NodeSelector {
		nodeSelector[k] = v

	}
	if IsCPUNodeDiscoveryEnabled(t.configMapStore) {
		if cpuModelLabel, err := CPUModelLabelFromCPUModel(vmi); err == nil {
			if vmi.Spec.Domain.CPU.Model != v1.CPUModeHostModel && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostPassthrough {
				nodeSelector[cpuModelLabel] = "true"
			}
		}
	}

	nodeSelector[v1.NodeSchedulable] = "true"

	podLabels := map[string]string{}

	for k, v := range vmi.Labels {
		podLabels[k] = v
	}
	podLabels[v1.AppLabel] = "virt-launcher"
	podLabels[v1.CreatedByLabel] = string(vmi.UID)

	for i, requestedHookSidecar := range requestedHookSidecarList {
		resources := k8sv1.ResourceRequirements{}
		// add default cpu and memory limits to enable cpu pinning if requested
		// TODO(vladikr): make the hookSidecar express resources
		if vmi.IsCPUDedicated() {
			resources.Limits = make(k8sv1.ResourceList)
			resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("200m")
			resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("64M")
		}
		containers = append(containers, k8sv1.Container{
			Name:            fmt.Sprintf("hook-sidecar-%d", i),
			Image:           requestedHookSidecar.Image,
			ImagePullPolicy: requestedHookSidecar.ImagePullPolicy,
			Resources:       resources,
			VolumeMounts: []k8sv1.VolumeMount{
				k8sv1.VolumeMount{
					Name:      "hook-sidecar-sockets",
					MountPath: hooks.HookSocketsSharedDirectory,
				},
			},
		})
	}

	// XXX: reduce test time. Adding one more container delays the start.
	// First stdci has issues with that and second we don't want to increase the startup time even more.
	// At the end the infra container needs to be always there, to allow better default readiness checks.
	if vmi.Spec.ReadinessProbe != nil {
		// Infra-ready container
		readyContainer := k8sv1.Container{
			Name:            "kubevirt-infra",
			Image:           t.launcherImage,
			ImagePullPolicy: imagePullPolicy,
			SecurityContext: &k8sv1.SecurityContext{
				RunAsUser: &userId,
			},
			Resources: k8sv1.ResourceRequirements{
				Limits: map[k8sv1.ResourceName]resource.Quantity{
					k8sv1.ResourceCPU:    resource.MustParse("1m"),
					k8sv1.ResourceMemory: resource.MustParse("5Mi"),
				},
			},
			Command:        []string{"/usr/bin/tail", "-f", "/dev/null"},
			VolumeDevices:  volumeDevices,
			VolumeMounts:   volumeMounts,
			ReadinessProbe: defaultReadinessProbe,
		}
		containers = append(containers, readyContainer)
	}

	hostName := dns.SanitizeHostname(vmi)

	annotationsList := map[string]string{
		v1.DomainAnnotation: domain,
	}

	cniNetworks, cniAnnotation := getCniInterfaceList(vmi)
	if len(cniNetworks) > 0 {
		annotationsList[cniAnnotation] = cniNetworks
	}

	for _, network := range vmi.Spec.Networks {
		if network.Multus != nil && network.Multus.Default {
			annotationsList[MULTUS_DEFAULT_NETWORK_CNI_ANNOTATION] = network.Multus.NetworkName
		}
	}

	// TODO use constants for podLabels
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-",
			Labels:       podLabels,
			Annotations:  annotationsList,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
		},
		Spec: k8sv1.PodSpec{
			Hostname:  hostName,
			Subdomain: vmi.Spec.Subdomain,
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsUser: &userId,
				SELinuxOptions: &k8sv1.SELinuxOptions{
					Type: "spc_t",
				},
			},
			TerminationGracePeriodSeconds: &gracePeriodKillAfter,
			RestartPolicy:                 k8sv1.RestartPolicyNever,
			Containers:                    containers,
			NodeSelector:                  nodeSelector,
			Volumes:                       volumes,
			ImagePullSecrets:              imagePullSecrets,
			DNSConfig:                     vmi.Spec.DNSConfig,
			DNSPolicy:                     vmi.Spec.DNSPolicy,
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

	if vmi.Spec.Tolerations != nil {
		pod.Spec.Tolerations = []k8sv1.Toleration{}
		for _, v := range vmi.Spec.Tolerations {
			pod.Spec.Tolerations = append(pod.Spec.Tolerations, v)
		}
	}

	if len(serviceAccountName) > 0 {
		pod.Spec.ServiceAccountName = serviceAccountName
		automount := true
		pod.Spec.AutomountServiceAccountToken = &automount
	} else {
		automount := false
		pod.Spec.AutomountServiceAccountToken = &automount
	}

	return &pod, nil
}

func getRequiredCapabilities(vmi *v1.VirtualMachineInstance) []k8sv1.Capability {
	res := []k8sv1.Capability{}
	if (len(vmi.Spec.Domain.Devices.Interfaces) > 0) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface == nil) ||
		(*vmi.Spec.Domain.Devices.AutoattachPodInterface == true) {
		res = append(res, CAP_NET_ADMIN)
	}
	// add a CAP_SYS_NICE capability to allow setting cpu affinity
	res = append(res, CAP_SYS_NICE)
	return res
}

func getRequiredResources(vmi *v1.VirtualMachineInstance, useEmulation bool) k8sv1.ResourceList {
	res := k8sv1.ResourceList{}
	if (len(vmi.Spec.Domain.Devices.Interfaces) > 0) ||
		(vmi.Spec.Domain.Devices.AutoattachPodInterface == nil) ||
		(*vmi.Spec.Domain.Devices.AutoattachPodInterface == true) {
		res[TunDevice] = resource.MustParse("1")
	}
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if !useEmulation && (iface.Model == "" || iface.Model == "virtio") {
			// Note that about network interface, useEmulation does not make
			// any difference on eventual Domain xml, but uniformly making
			// /dev/vhost-net unavailable and libvirt implicitly fallback
			// to use QEMU userland NIC emulation.
			res[VhostNetDevice] = resource.MustParse("1")
		}
	}
	return res
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
	overhead.Add(resource.MustParse("128M"))

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

func getPortsFromVMI(vmi *v1.VirtualMachineInstance) []k8sv1.ContainerPort {
	ports := make([]k8sv1.ContainerPort, 0)

	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Ports != nil {
			for _, port := range iface.Ports {
				if port.Protocol == "" {
					port.Protocol = "TCP"
				}

				ports = append(ports, k8sv1.ContainerPort{Protocol: k8sv1.Protocol(port.Protocol), Name: port.Name, ContainerPort: port.Port})
			}
		}
	}

	if len(ports) == 0 {
		return nil
	}

	return ports
}

func getResourceNameForNetwork(network *networkv1.NetworkAttachmentDefinition) string {
	resourceName, ok := network.Annotations[MULTUS_RESOURCE_NAME_ANNOTATION]
	if ok {
		return resourceName
	}
	return "" // meaning the network is not served by resources
}

func getNamespaceAndNetworkName(vmi *v1.VirtualMachineInstance, fullNetworkName string) (namespace string, networkName string) {
	if strings.Contains(fullNetworkName, "/") {
		res := strings.SplitN(fullNetworkName, "/", 2)
		namespace, networkName = res[0], res[1]
	} else {
		namespace = precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
		networkName = fullNetworkName
	}
	return
}

func getNetworkToResourceMap(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (networkToResourceMap map[string]string, err error) {
	networkToResourceMap = make(map[string]string)
	for _, network := range vmi.Spec.Networks {
		if network.Multus != nil {
			namespace, networkName := getNamespaceAndNetworkName(vmi, network.Multus.NetworkName)
			crd, err := virtClient.NetworkClient().K8sCniCncfIo().NetworkAttachmentDefinitions(namespace).Get(networkName, metav1.GetOptions{})
			if err != nil {
				return map[string]string{}, fmt.Errorf("Failed to locate network attachment definition %s/%s", namespace, networkName)
			}
			networkToResourceMap[network.Name] = getResourceNameForNetwork(crd)
		}
	}
	return
}

func getCniInterfaceList(vmi *v1.VirtualMachineInstance) (ifaceListString string, cniAnnotation string) {
	ifaceList := make([]string, 0)

	for _, network := range vmi.Spec.Networks {
		// set the type for the first network
		// all other networks must have same type
		if network.Multus != nil {
			if network.Multus.Default {
				continue
			}
			ifaceList = append(ifaceList, network.Multus.NetworkName)
			if cniAnnotation == "" {
				cniAnnotation = "k8s.v1.cni.cncf.io/networks"
			}
		} else if network.Genie != nil {
			ifaceList = append(ifaceList, network.Genie.NetworkName)
			if cniAnnotation == "" {
				cniAnnotation = "cni"
			}
		}
	}

	ifaceListString = strings.Join(ifaceList, ",")
	return
}

func NewTemplateService(launcherImage string,
	virtShareDir string,
	ephemeralDiskDir string,
	imagePullSecret string,
	configMapCache cache.Store,
	persistentVolumeClaimCache cache.Store,
	virtClient kubecli.KubevirtClient) TemplateService {

	precond.MustNotBeEmpty(launcherImage)
	svc := templateService{
		launcherImage:              launcherImage,
		virtShareDir:               virtShareDir,
		ephemeralDiskDir:           ephemeralDiskDir,
		imagePullSecret:            imagePullSecret,
		configMapStore:             configMapCache,
		persistentVolumeClaimStore: persistentVolumeClaimCache,
		virtClient:                 virtClient,
	}
	return &svc
}

func copyProbe(probe *v1.Probe) *k8sv1.Probe {
	if probe == nil {
		return nil
	}
	return &k8sv1.Probe{
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
		Handler: k8sv1.Handler{
			HTTPGet:   probe.HTTPGet,
			TCPSocket: probe.TCPSocket,
		},
	}
}
