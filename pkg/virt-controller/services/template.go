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
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/kubectl/pkg/cmd/util/podcmd"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"

	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	"kubevirt.io/kubevirt/pkg/config"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/util/types"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	containerDisks   = "container-disks"
	hotplugDisks     = "hotplug-disks"
	hookSidecarSocks = "hook-sidecar-sockets"
	varRun           = "/var/run"
	virtBinDir       = "virt-bin-share-dir"
	hotplugDisk      = "hotplug-disk"
)

const KvmDevice = "devices.kubevirt.io/kvm"
const TunDevice = "devices.kubevirt.io/tun"
const VhostNetDevice = "devices.kubevirt.io/vhost-net"
const SevDevice = "devices.kubevirt.io/sev"

const debugLogs = "debugLogs"
const logVerbosity = "logVerbosity"
const virtiofsDebugLogs = "virtiofsdDebugLogs"

const MultusNetworksAnnotation = "k8s.v1.cni.cncf.io/networks"

const qemuTimeoutJitterRange = 120

const (
	CAP_NET_BIND_SERVICE = "NET_BIND_SERVICE"
	CAP_NET_RAW          = "NET_RAW"
	CAP_SYS_ADMIN        = "SYS_ADMIN"
	CAP_SYS_NICE         = "SYS_NICE"
)

// LibvirtStartupDelay is added to custom liveness and readiness probes initial delay value.
// Libvirt needs roughly 10 seconds to start.
const LibvirtStartupDelay = 10

//These perfixes for node feature discovery, are used in a NodeSelector on the pod
//to match a VirtualMachineInstance CPU model(Family) and/or features to nodes that support them.
const NFD_CPU_MODEL_PREFIX = "cpu-model.node.kubevirt.io/"
const NFD_CPU_FEATURE_PREFIX = "cpu-feature.node.kubevirt.io/"
const NFD_KVM_INFO_PREFIX = "hyperv.node.kubevirt.io/"
const IntelVendorName = "Intel"

const MULTUS_RESOURCE_NAME_ANNOTATION = "k8s.v1.cni.cncf.io/resourceName"
const MULTUS_DEFAULT_NETWORK_CNI_ANNOTATION = "v1.multus-cni.io/default-network"

// Istio list of virtual interfaces whose inbound traffic (from VM) will be treated as outbound traffic in envoy
const ISTIO_KUBEVIRT_ANNOTATION = "traffic.sidecar.istio.io/kubevirtInterfaces"

const VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION = "pre.hook.backup.velero.io/container"
const VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION = "pre.hook.backup.velero.io/command"
const VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION = "post.hook.backup.velero.io/container"
const VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION = "post.hook.backup.velero.io/command"

const ENV_VAR_LIBVIRT_DEBUG_LOGS = "LIBVIRT_DEBUG_LOGS"
const ENV_VAR_VIRTIOFSD_DEBUG_LOGS = "VIRTIOFSD_DEBUG_LOGS"
const ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY = "VIRT_LAUNCHER_LOG_VERBOSITY"

const ENV_VAR_POD_NAME = "POD_NAME"

// extensive log verbosity threshold after which libvirt debug logs will be enabled
const EXT_LOG_VERBOSITY_THRESHOLD = 5

const ephemeralStorageOverheadSize = "50M"

type TemplateService interface {
	RenderMigrationManifest(vmi *v1.VirtualMachineInstance, sourcePod *k8sv1.Pod) (*k8sv1.Pod, error)
	RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error)
	RenderHotplugAttachmentPodTemplate(volume []*v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, claimMap map[string]*k8sv1.PersistentVolumeClaim, tempPod bool) (*k8sv1.Pod, error)
	RenderHotplugAttachmentTriggerPodTemplate(volume *v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, pvcName string, isBlock bool, tempPod bool) (*k8sv1.Pod, error)
	RenderLaunchManifestNoVm(*v1.VirtualMachineInstance) (*k8sv1.Pod, error)
	GetLauncherImage() string
	IsPPC64() bool
	IsARM64() bool
}

type templateService struct {
	launcherImage              string
	launcherQemuTimeout        int
	virtShareDir               string
	virtLibDir                 string
	ephemeralDiskDir           string
	containerDiskDir           string
	hotplugDiskDir             string
	imagePullSecret            string
	persistentVolumeClaimStore cache.Store
	virtClient                 kubecli.KubevirtClient
	clusterConfig              *virtconfig.ClusterConfig
	launcherSubGid             int64
}

type PvcNotFoundError struct {
	Reason string
}

func (e PvcNotFoundError) Error() string {
	return e.Reason
}

type DataVolumeNotFoundError struct {
	Reason string
}

func (e DataVolumeNotFoundError) Error() string {
	return e.Reason
}

func isFeatureStateEnabled(fs *v1.FeatureState) bool {
	return fs != nil && fs.Enabled != nil && *fs.Enabled
}

type hvFeatureLabel struct {
	Feature *v1.FeatureState
	Label   string
}

// makeHVFeatureLabelTable creates the mapping table between the VMI hyperv state and the label names.
// The table needs pointers to v1.FeatureHyperv struct, so it has to be generated and can't be a
// static var
func makeHVFeatureLabelTable(vmi *v1.VirtualMachineInstance) []hvFeatureLabel {
	// The following HyperV features don't require support from the host kernel, according to inspection
	// of the QEMU sources (4.0 - adb3321bfd)
	// VAPIC, Relaxed, Spinlocks, VendorID
	// VPIndex, SyNIC: depend on both MSR and capability
	// IPI, TLBFlush: depend on KVM Capabilities
	// Runtime, Reset, SyNICTimer, Frequencies, Reenlightenment: depend on KVM MSRs availability
	// EVMCS: depends on KVM capability, but the only way to know that is enable it, QEMU doesn't do
	// any check before that, so we leave it out
	//
	// see also https://schd.ws/hosted_files/devconfcz2019/cf/vkuznets_enlightening_kvm_devconf2019.pdf
	// to learn about dependencies between enlightenments

	hyperv := vmi.Spec.Domain.Features.Hyperv // shortcut

	syNICTimer := &v1.FeatureState{}
	if hyperv.SyNICTimer != nil {
		syNICTimer.Enabled = hyperv.SyNICTimer.Enabled
	}

	return []hvFeatureLabel{
		{
			Feature: hyperv.VPIndex,
			Label:   "vpindex",
		},
		{
			Feature: hyperv.Runtime,
			Label:   "runtime",
		},
		{
			Feature: hyperv.Reset,
			Label:   "reset",
		},
		{
			// TODO: SyNIC depends on vp-index on QEMU level. We should enforce this constraint.
			Feature: hyperv.SyNIC,
			Label:   "synic",
		},
		{
			// TODO: SyNICTimer depends on SyNIC and Relaxed. We should enforce this constraint.
			Feature: syNICTimer,
			Label:   "synictimer",
		},
		{
			Feature: hyperv.Frequencies,
			Label:   "frequencies",
		},
		{
			Feature: hyperv.Reenlightenment,
			Label:   "reenlightenment",
		},
		{
			Feature: hyperv.TLBFlush,
			Label:   "tlbflush",
		},
		{
			Feature: hyperv.IPI,
			Label:   "ipi",
		},
	}
}

func getHypervNodeSelectors(vmi *v1.VirtualMachineInstance) map[string]string {
	nodeSelectors := make(map[string]string)
	if vmi.Spec.Domain.Features == nil || vmi.Spec.Domain.Features.Hyperv == nil {
		return nodeSelectors
	}

	hvFeatureLabels := makeHVFeatureLabelTable(vmi)
	for _, hv := range hvFeatureLabels {
		if isFeatureStateEnabled(hv.Feature) {
			nodeSelectors[NFD_KVM_INFO_PREFIX+hv.Label] = "true"
		}
	}

	if vmi.Spec.Domain.Features.Hyperv.EVMCS != nil {
		nodeSelectors[v1.CPUModelVendorLabel+IntelVendorName] = "true"
	}

	return nodeSelectors
}

func CPUModelLabelFromCPUModel(vmi *v1.VirtualMachineInstance) (label string, err error) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" {
		err = fmt.Errorf("Cannot create CPU Model label, vmi spec is mising CPU model")
		return
	}
	label = NFD_CPU_MODEL_PREFIX + vmi.Spec.Domain.CPU.Model
	return
}

func CPUFeatureLabelsFromCPUFeatures(vmi *v1.VirtualMachineInstance) []string {
	var labels []string
	if vmi.Spec.Domain.CPU != nil && vmi.Spec.Domain.CPU.Features != nil {
		for _, feature := range vmi.Spec.Domain.CPU.Features {
			if feature.Policy == "" || feature.Policy == "require" {
				labels = append(labels, NFD_CPU_FEATURE_PREFIX+feature.Name)
			}
		}
	}
	return labels
}

func SetNodeAffinityForForbiddenFeaturePolicy(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {

	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Features == nil {
		return
	}

	for _, feature := range vmi.Spec.Domain.CPU.Features {
		if feature.Policy == "forbid" {

			requirement := k8sv1.NodeSelectorRequirement{
				Key:      NFD_CPU_FEATURE_PREFIX + feature.Name,
				Operator: k8sv1.NodeSelectorOpDoesNotExist,
			}
			term := k8sv1.NodeSelectorTerm{
				MatchExpressions: []k8sv1.NodeSelectorRequirement{requirement}}

			nodeAffinity := &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{term},
				},
			}

			if pod.Spec.Affinity != nil && pod.Spec.Affinity.NodeAffinity != nil {
				if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
					terms := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
					// Since NodeSelectorTerms are ORed , the anti affinity requirement will be added to each term.
					for i, selectorTerm := range terms {
						pod.Spec.Affinity.NodeAffinity.
							RequiredDuringSchedulingIgnoredDuringExecution.
							NodeSelectorTerms[i].MatchExpressions = append(selectorTerm.MatchExpressions, requirement)
					}
				} else {
					pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sv1.NodeSelector{
						NodeSelectorTerms: []k8sv1.NodeSelectorTerm{term},
					}
				}

			} else if pod.Spec.Affinity != nil {
				pod.Spec.Affinity.NodeAffinity = nodeAffinity
			} else {
				pod.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: nodeAffinity,
				}

			}
		}
	}
}

func sysprepVolumeSource(sysprepVolume v1.SysprepSource) (k8sv1.VolumeSource, error) {
	logger := log.DefaultLogger()
	if sysprepVolume.Secret != nil {
		return k8sv1.VolumeSource{
			Secret: &k8sv1.SecretVolumeSource{
				SecretName: sysprepVolume.Secret.Name,
			},
		}, nil
	} else if sysprepVolume.ConfigMap != nil {
		return k8sv1.VolumeSource{
			ConfigMap: &k8sv1.ConfigMapVolumeSource{
				LocalObjectReference: k8sv1.LocalObjectReference{
					Name: sysprepVolume.ConfigMap.Name,
				},
			},
		}, nil
	}
	errorStr := fmt.Sprintf("Sysprep must have Secret or ConfigMap reference set %v", sysprepVolume)
	logger.Errorf(errorStr)
	return k8sv1.VolumeSource{}, fmt.Errorf(errorStr)
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
// should be equal to requests; and 2) QoS rules do not apVFIO//
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

func (t *templateService) GetLauncherImage() string {
	return t.launcherImage
}

func (t *templateService) RenderLaunchManifestNoVm(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	return t.renderLaunchManifest(vmi, nil, true)
}

func (t *templateService) RenderMigrationManifest(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) (*k8sv1.Pod, error) {
	reproducibleImageIDs, err := containerdisk.ExtractImageIDsFromSourcePod(vmi, pod)
	if err != nil {
		return nil, fmt.Errorf("can not proceed with the migration when no reproducible image digest can be detected: %v", err)
	}
	return t.renderLaunchManifest(vmi, reproducibleImageIDs, false)
}

func (t *templateService) RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	return t.renderLaunchManifest(vmi, nil, false)
}

func (t *templateService) IsPPC64() bool {
	return t.clusterConfig.GetClusterCPUArch() == "ppc64le"
}

func (t *templateService) IsARM64() bool {
	return t.clusterConfig.GetClusterCPUArch() == "arm64"
}

func generateQemuTimeoutWithJitter(qemuTimeoutBaseSeconds int) string {
	timeout := rand.Intn(qemuTimeoutJitterRange) + qemuTimeoutBaseSeconds

	return fmt.Sprintf("%ds", timeout)
}

func (t *templateService) addPVCToLaunchManifest(volume v1.Volume, claimName string, namespace string, volumeMounts *[]k8sv1.VolumeMount, volumeDevices *[]k8sv1.VolumeDevice) error {
	logger := log.DefaultLogger()
	_, exists, isBlock, err := types.IsPVCBlockFromStore(t.persistentVolumeClaimStore, namespace, claimName)
	if err != nil {
		logger.Errorf("error getting PVC: %v", claimName)
		return err
	} else if !exists {
		logger.Errorf("didn't find PVC %v", claimName)
		return PvcNotFoundError{Reason: fmt.Sprintf("didn't find PVC %v", claimName)}
	} else if isBlock {
		devicePath := filepath.Join(string(filepath.Separator), "dev", volume.Name)
		device := k8sv1.VolumeDevice{
			Name:       volume.Name,
			DevicePath: devicePath,
		}
		*volumeDevices = append(*volumeDevices, device)
	} else {
		volumeMount := k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: hostdisk.GetMountedHostDiskDir(volume.Name),
		}
		*volumeMounts = append(*volumeMounts, volumeMount)
	}
	return nil
}

func (t *templateService) renderLaunchManifest(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, tempPod bool) (*k8sv1.Pod, error) {
	precond.MustNotBeNil(vmi)
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	nodeSelector := map[string]string{}

	var volumes []k8sv1.Volume
	var volumeDevices []k8sv1.VolumeDevice
	var volumeMounts []k8sv1.VolumeMount
	var imagePullSecrets []k8sv1.LocalObjectReference

	var userId int64 = util.RootUser
	var privileged bool = false

	nonRoot := util.IsNonRootVMI(vmi)
	if nonRoot {
		userId = util.NonRootUID
	}

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "private",
		MountPath: util.VirtPrivateDir,
	})
	volumes = append(volumes, k8sv1.Volume{
		Name: "private",
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "public",
		MountPath: util.VirtShareDir,
	})
	volumes = append(volumes, k8sv1.Volume{
		Name: "public",
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})

	hotplugVolumes := make(map[string]bool)
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.HotplugVolume != nil {
			hotplugVolumes[volumeStatus.Name] = true
		}
	}
	// This detects hotplug volumes for a started but not ready VMI
	for _, volume := range vmi.Spec.Volumes {
		if (volume.DataVolume != nil && volume.DataVolume.Hotpluggable) || (volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable) {
			hotplugVolumes[volume.Name] = true
		}
	}

	// Need to run in privileged mode in Power or libvirt will fail to lock memory for VMI
	if t.IsPPC64() {
		privileged = true
	}

	gracePeriodSeconds := v1.DefaultGracePeriodSeconds
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		gracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
	}

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "ephemeral-disks",
		MountPath: t.ephemeralDiskDir,
	})

	prop := k8sv1.MountPropagationHostToContainer
	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:             containerDisks,
		MountPath:        t.containerDiskDir,
		MountPropagation: &prop,
	})
	if !vmi.Spec.Domain.Devices.DisableHotplug {
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:             hotplugDisks,
			MountPath:        t.hotplugDiskDir,
			MountPropagation: &prop,
		})
	}

	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "libvirt-runtime",
		MountPath: "/var/run/libvirt",
	})

	// virt-launcher cmd socket dir
	volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
		Name:      "sockets",
		MountPath: filepath.Join(t.virtShareDir, "sockets"),
	})
	volumes = append(volumes, k8sv1.Volume{
		Name: "sockets",
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})

	serviceAccountName := ""

	for _, volume := range vmi.Spec.Volumes {
		if hotplugVolumes[volume.Name] {
			continue
		}
		if volume.PersistentVolumeClaim != nil {
			claimName := volume.PersistentVolumeClaim.ClaimName
			if err := t.addPVCToLaunchManifest(volume, claimName, namespace, &volumeMounts, &volumeDevices); err != nil {
				return nil, err
			}
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: volume.PersistentVolumeClaim.ClaimName,
						ReadOnly:  volume.PersistentVolumeClaim.ReadOnly,
					},
				},
			})
		}
		if volume.Ephemeral != nil {
			claimName := volume.Ephemeral.PersistentVolumeClaim.ClaimName
			if err := t.addPVCToLaunchManifest(volume, claimName, namespace, &volumeMounts, &volumeDevices); err != nil {
				return nil, err
			}
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
				MountPath: hostdisk.GetMountedHostDiskDir(volume.Name),
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
			claimName := volume.DataVolume.Name
			if err := t.addPVCToLaunchManifest(volume, claimName, namespace, &volumeMounts, &volumeDevices); err != nil {
				return nil, err
			}
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: claimName,
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

		if volume.DownwardAPI != nil {
			// attach a Secret to the pod
			volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
				Name:      volume.Name,
				MountPath: filepath.Join(config.DownwardAPISourceDir, volume.Name),
				ReadOnly:  true,
			})
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					DownwardAPI: &k8sv1.DownwardAPIVolumeSource{
						Items: volume.DownwardAPI.Fields,
					},
				},
			})
		}

		if volume.ServiceAccount != nil {
			serviceAccountName = volume.ServiceAccount.ServiceAccountName
		}

		if volume.DownwardMetrics != nil {
			sizeLimit := resource.MustParse("1Mi")
			volumes = append(volumes, k8sv1.Volume{
				Name: volume.Name,
				VolumeSource: k8sv1.VolumeSource{
					EmptyDir: &k8sv1.EmptyDirVolumeSource{
						Medium:    "Memory",
						SizeLimit: &sizeLimit,
					},
				},
			})
			volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
				Name:      volume.Name,
				MountPath: config.DownwardMetricDisksDir,
			})
		}

		if volume.CloudInitNoCloud != nil {
			if volume.CloudInitNoCloud.UserDataSecretRef != nil {
				// attach a secret referenced by the user
				volumeName := volume.Name + "-udata"
				volumes = append(volumes, k8sv1.Volume{
					Name: volumeName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: volume.CloudInitNoCloud.UserDataSecretRef.Name,
						},
					},
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userdata"),
					SubPath:   "userdata",
					ReadOnly:  true,
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userData"),
					SubPath:   "userData",
					ReadOnly:  true,
				})
			}
			if volume.CloudInitNoCloud.NetworkDataSecretRef != nil {
				// attach a secret referenced by the networkdata
				volumeName := volume.Name + "-ndata"
				volumes = append(volumes, k8sv1.Volume{
					Name: volumeName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: volume.CloudInitNoCloud.NetworkDataSecretRef.Name,
						},
					},
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkdata"),
					SubPath:   "networkdata",
					ReadOnly:  true,
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkData"),
					SubPath:   "networkData",
					ReadOnly:  true,
				})
			}
		}

		if volume.Sysprep != nil {
			var volumeSource k8sv1.VolumeSource
			// attach a Secret or ConfigMap referenced by the user
			volumeSource, err := sysprepVolumeSource(*volume.Sysprep)
			if err != nil {
				return nil, err
			}
			volumes = append(volumes, k8sv1.Volume{
				Name:         volume.Name,
				VolumeSource: volumeSource,
			})
			volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
				Name:      volume.Name,
				MountPath: filepath.Join(config.SysprepSourceDir, volume.Name),
				ReadOnly:  true,
			})
		}

		if volume.CloudInitConfigDrive != nil {
			if volume.CloudInitConfigDrive.UserDataSecretRef != nil {
				// attach a secret referenced by the user
				volumeName := volume.Name + "-udata"
				volumes = append(volumes, k8sv1.Volume{
					Name: volumeName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: volume.CloudInitConfigDrive.UserDataSecretRef.Name,
						},
					},
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userdata"),
					SubPath:   "userdata",
					ReadOnly:  true,
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "userData"),
					SubPath:   "userData",
					ReadOnly:  true,
				})
			}
			if volume.CloudInitConfigDrive.NetworkDataSecretRef != nil {
				// attach a secret referenced by the networkdata
				volumeName := volume.Name + "-ndata"
				volumes = append(volumes, k8sv1.Volume{
					Name: volumeName,
					VolumeSource: k8sv1.VolumeSource{
						Secret: &k8sv1.SecretVolumeSource{
							SecretName: volume.CloudInitConfigDrive.NetworkDataSecretRef.Name,
						},
					},
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkdata"),
					SubPath:   "networkdata",
					ReadOnly:  true,
				})
				volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
					Name:      volumeName,
					MountPath: filepath.Join(config.SecretSourceDir, volume.Name, "networkData"),
					SubPath:   "networkData",
					ReadOnly:  true,
				})
			}
		}
	}

	for _, accessCred := range vmi.Spec.AccessCredentials {
		secretName := ""
		if accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.Source.Secret != nil {
			secretName = accessCred.SSHPublicKey.Source.Secret.SecretName
		} else if accessCred.UserPassword != nil && accessCred.UserPassword.Source.Secret != nil {
			secretName = accessCred.UserPassword.Source.Secret.SecretName
		}

		if secretName == "" {
			continue
		}
		volumeName := secretName + "-access-cred"
		volumes = append(volumes, k8sv1.Volume{
			Name: volumeName,
			VolumeSource: k8sv1.VolumeSource{
				Secret: &k8sv1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		})
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      volumeName,
			MountPath: filepath.Join(config.SecretSourceDir, volumeName),
			ReadOnly:  true,
		})
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
	memoryOverhead := GetMemoryOverhead(vmi, t.clusterConfig.GetClusterCPUArch())

	// Consider CPU and memory requests and limits for pod scheduling
	resources := k8sv1.ResourceRequirements{}
	vmiResources := vmi.Spec.Domain.Resources

	resources.Requests = make(k8sv1.ResourceList)
	resources.Limits = make(k8sv1.ResourceList)

	// Set Default CPUs request
	if !vmi.IsCPUDedicated() {
		vcpus := int64(1)
		if vmi.Spec.Domain.CPU != nil {
			vcpus = hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU)
		}
		cpuAllocationRatio := t.clusterConfig.GetCPUAllocationRatio()
		if vcpus != 0 && cpuAllocationRatio > 0 {
			val := float64(vcpus) / float64(cpuAllocationRatio)
			vcpusStr := fmt.Sprintf("%g", val)
			if val < 1 {
				val *= 1000
				vcpusStr = fmt.Sprintf("%gm", val)
			}
			resources.Requests[k8sv1.ResourceCPU] = resource.MustParse(vcpusStr)
		}
	}
	// Copy vmi resources requests to a container
	for key, value := range vmiResources.Requests {
		resources.Requests[key] = value
	}

	// Copy vmi resources limits to a container
	for key, value := range vmiResources.Limits {
		resources.Limits[key] = value
	}

	// Add ephemeral storage request to container to be used by Kubevirt. This amount of ephemeral storage
	// should be added to the user's request.
	ephemeralStorageOverhead := resource.MustParse(ephemeralStorageOverheadSize)
	ephemeralStorageRequested := resources.Requests[k8sv1.ResourceEphemeralStorage]
	ephemeralStorageRequested.Add(ephemeralStorageOverhead)
	resources.Requests[k8sv1.ResourceEphemeralStorage] = ephemeralStorageRequested

	if ephemeralStorageLimit, ephemeralStorageLimitDefined := resources.Limits[k8sv1.ResourceEphemeralStorage]; ephemeralStorageLimitDefined {
		ephemeralStorageLimit.Add(ephemeralStorageOverhead)
		resources.Limits[k8sv1.ResourceEphemeralStorage] = ephemeralStorageLimit
	}

	// Consider hugepages resource for pod scheduling
	if util.HasHugePages(vmi) {
		hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + vmi.Spec.Domain.Memory.Hugepages.PageSize)
		hugepagesMemReq := vmi.Spec.Domain.Resources.Requests.Memory()

		// If requested, use the guest memory to allocate hugepages
		if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
			requests := vmi.Spec.Domain.Resources.Requests.Memory().Value()
			guest := vmi.Spec.Domain.Memory.Guest.Value()
			if requests > guest {
				hugepagesMemReq = vmi.Spec.Domain.Memory.Guest
			}
		}
		resources.Requests[hugepageType] = *hugepagesMemReq
		resources.Limits[hugepageType] = *hugepagesMemReq

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

		reqMemDiff := resource.NewScaledQuantity(0, resource.Kilo)
		limMemDiff := resource.NewScaledQuantity(0, resource.Kilo)
		// In case the guest memory and the requested memeory are different, add the difference
		// to the to the overhead
		if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
			requests := vmi.Spec.Domain.Resources.Requests.Memory().Value()
			limits := vmi.Spec.Domain.Resources.Limits.Memory().Value()
			guest := vmi.Spec.Domain.Memory.Guest.Value()
			if requests > guest {
				reqMemDiff.Add(*vmi.Spec.Domain.Resources.Requests.Memory())
				reqMemDiff.Sub(*vmi.Spec.Domain.Memory.Guest)
			}
			if limits > guest {
				limMemDiff.Add(*vmi.Spec.Domain.Resources.Limits.Memory())
				limMemDiff.Sub(*vmi.Spec.Domain.Memory.Guest)
			}
		}
		// Set requested memory equals to overhead memory
		reqMemDiff.Add(*memoryOverhead)
		resources.Requests[k8sv1.ResourceMemory] = *reqMemDiff
		if _, ok := resources.Limits[k8sv1.ResourceMemory]; ok {
			limMemDiff.Add(*memoryOverhead)
			resources.Limits[k8sv1.ResourceMemory] = *limMemDiff
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
			Name: hookSidecarSocks,
			VolumeSource: k8sv1.VolumeSource{
				EmptyDir: &k8sv1.EmptyDirVolumeSource{},
			},
		})
		volumeMounts = append(volumeMounts, k8sv1.VolumeMount{
			Name:      hookSidecarSocks,
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
		// allocate 1 more pcpu if IsolateEmulatorThread request
		if vmi.Spec.Domain.CPU.IsolateEmulatorThread {
			emulatorThreadCPU := resource.NewQuantity(1, resource.BinarySI)
			limits := resources.Limits[k8sv1.ResourceCPU]
			limits.Add(*emulatorThreadCPU)
			resources.Limits[k8sv1.ResourceCPU] = limits
			if cpuRequest, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
				cpuRequest.Add(*emulatorThreadCPU)
				resources.Requests[k8sv1.ResourceCPU] = cpuRequest
			}
		}

		resources.Limits[k8sv1.ResourceMemory] = *resources.Requests.Memory()
	}

	ovmfPath := t.clusterConfig.GetOVMFPath()

	var command []string
	if tempPod {
		logger := log.DefaultLogger()
		logger.Infof("RUNNING doppleganger pod for %s", vmi.Name)
		command = []string{"/bin/bash",
			"-c",
			"echo", "bound PVCs"}
	} else {
		command = []string{"/usr/bin/virt-launcher",
			"--qemu-timeout", generateQemuTimeoutWithJitter(t.launcherQemuTimeout),
			"--name", domain,
			"--uid", string(vmi.UID),
			"--namespace", namespace,
			"--kubevirt-share-dir", t.virtShareDir,
			"--ephemeral-disk-dir", t.ephemeralDiskDir,
			"--container-disk-dir", t.containerDiskDir,
			"--grace-period-seconds", strconv.Itoa(int(gracePeriodSeconds)),
			"--hook-sidecars", strconv.Itoa(len(requestedHookSidecarList)),
			"--ovmf-path", ovmfPath,
		}
		if nonRoot {
			command = append(command, "--run-as-nonroot")
		}
		if customDebugFilters, exists := vmi.Annotations[v1.CustomLibvirtLogFiltersAnnotation]; exists {
			log.Log.Object(vmi).Infof("Applying custom debug filters for vmi %s: %s", vmi.Name, customDebugFilters)
			command = append(command, "--libvirt-log-filters", customDebugFilters)
		}
	}

	allowEmulation := t.clusterConfig.AllowEmulation()
	imagePullPolicy := t.clusterConfig.GetImagePullPolicy()

	if resources.Limits == nil {
		resources.Limits = make(k8sv1.ResourceList)
	}

	extraResources := getRequiredResources(vmi, allowEmulation)
	for key, val := range extraResources {
		resources.Limits[key] = val
	}

	if allowEmulation {
		command = append(command, "--allow-emulation")
	} else {
		resources.Limits[KvmDevice] = resource.MustParse("1")
	}

	if checkForKeepLauncherAfterFailure(vmi) {
		command = append(command, "--keep-after-failure")
	}

	_, ok := vmi.Annotations[v1.FuncTestLauncherFailFastAnnotation]
	if ok {
		command = append(command, "--simulate-crash")
	}

	// Add ports from interfaces to the pod manifest
	ports := getPortsFromVMI(vmi)

	capabilities := getRequiredCapabilities(vmi, t.clusterConfig)

	networkToResourceMap, err := getNetworkToResourceMap(t.virtClient, vmi)
	if err != nil {
		return nil, err
	}

	// Register resource requests and limits corresponding to attached multus networks.
	for _, resourceName := range networkToResourceMap {
		if resourceName != "" {
			requestResource(&resources, resourceName)
		}
	}

	err = validatePermittedHostDevices(&vmi.Spec, t.clusterConfig)
	if err != nil {
		return nil, err
	}

	if util.IsGPUVMI(vmi) {
		for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
			requestResource(&resources, gpu.DeviceName)
		}
	}

	if util.IsHostDevVMI(vmi) {
		for _, hostDev := range vmi.Spec.Domain.Devices.HostDevices {
			requestResource(&resources, hostDev.DeviceName)
		}
	}

	if util.IsSEVVMI(vmi) {
		requestResource(&resources, SevDevice)
	}

	// VirtualMachineInstance target container
	compute := k8sv1.Container{
		Name:            "compute",
		Image:           t.launcherImage,
		ImagePullPolicy: imagePullPolicy,
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:  &userId,
			Privileged: &privileged,
			Capabilities: &k8sv1.Capabilities{
				Add:  capabilities,
				Drop: []k8sv1.Capability{CAP_NET_RAW},
			},
		},
		Command:       command,
		VolumeDevices: volumeDevices,
		VolumeMounts:  volumeMounts,
		Resources:     resources,
		Ports:         ports,
	}
	if nonRoot {
		compute.SecurityContext.RunAsGroup = &userId
		compute.SecurityContext.RunAsNonRoot = &nonRoot
		compute.Env = append(compute.Env,
			k8sv1.EnvVar{
				Name:  "XDG_CACHE_HOME",
				Value: varRun,
			},
			k8sv1.EnvVar{
				Name:  "XDG_CONFIG_HOME",
				Value: varRun,
			},
			k8sv1.EnvVar{
				Name:  "XDG_RUNTIME_DIR",
				Value: varRun,
			},
		)
	}

	if vmi.Spec.ReadinessProbe != nil {
		v1.SetDefaults_Probe(vmi.Spec.ReadinessProbe)
		compute.ReadinessProbe = copyProbe(vmi.Spec.ReadinessProbe)
		updateReadinessProbe(vmi, compute.ReadinessProbe)
	}

	if vmi.Spec.LivenessProbe != nil {
		v1.SetDefaults_Probe(vmi.Spec.LivenessProbe)
		compute.LivenessProbe = copyProbe(vmi.Spec.LivenessProbe)
		updateLivenessProbe(vmi, compute.LivenessProbe)
	}

	for networkName, resourceName := range networkToResourceMap {
		varName := fmt.Sprintf("KUBEVIRT_RESOURCE_NAME_%s", networkName)
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: varName, Value: resourceName})
	}

	virtLauncherLogVerbosity := t.clusterConfig.GetVirtLauncherVerbosity()

	if verbosity, isSet := vmi.Labels[logVerbosity]; isSet || virtLauncherLogVerbosity != virtconfig.DefaultVirtLauncherLogVerbosity {
		// Override the cluster wide verbosity level if a specific value has been provided for this VMI
		verbosityStr := fmt.Sprint(virtLauncherLogVerbosity)
		if isSet {
			verbosityStr = verbosity

			verbosityInt, err := strconv.Atoi(verbosity)
			if err != nil {
				return nil, fmt.Errorf("verbosity %s cannot cast to int: %v", verbosity, err)
			}

			virtLauncherLogVerbosity = uint(verbosityInt)
		}
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, Value: verbosityStr})
	}

	if labelValue, ok := vmi.Labels[debugLogs]; (ok && strings.EqualFold(labelValue, "true")) || virtLauncherLogVerbosity > EXT_LOG_VERBOSITY_THRESHOLD {
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: ENV_VAR_LIBVIRT_DEBUG_LOGS, Value: "1"})
	}
	if labelValue, ok := vmi.Labels[virtiofsDebugLogs]; (ok && strings.EqualFold(labelValue, "true")) || virtLauncherLogVerbosity > EXT_LOG_VERBOSITY_THRESHOLD {
		compute.Env = append(compute.Env, k8sv1.EnvVar{Name: ENV_VAR_VIRTIOFSD_DEBUG_LOGS, Value: "1"})
	}

	compute.Env = append(compute.Env, k8sv1.EnvVar{
		Name: ENV_VAR_POD_NAME,
		ValueFrom: &k8sv1.EnvVarSource{
			FieldRef: &k8sv1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})

	// Make sure the compute container is always the first since the mutating webhook shipped with the sriov operator
	// for adding the requested resources to the pod will add them to the first container of the list
	containers := []k8sv1.Container{compute}
	containersDisks := containerdisk.GenerateContainers(vmi, imageIDs, containerDisks, virtBinDir)
	containers = append(containers, containersDisks...)

	kernelBootContainer := containerdisk.GenerateKernelBootContainer(vmi, imageIDs, containerDisks, virtBinDir)
	if kernelBootContainer != nil {
		log.Log.Object(vmi).Infof("kernel boot container generated")
		containers = append(containers, *kernelBootContainer)
	}

	volumes = append(volumes,
		k8sv1.Volume{
			Name: virtBinDir,
			VolumeSource: k8sv1.VolumeSource{
				EmptyDir: &k8sv1.EmptyDirVolumeSource{},
			},
		},
	)
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
	volumes = append(volumes, k8sv1.Volume{
		Name: containerDisks,
		VolumeSource: k8sv1.VolumeSource{
			EmptyDir: &k8sv1.EmptyDirVolumeSource{},
		},
	})
	if !vmi.Spec.Domain.Devices.DisableHotplug {
		volumes = append(volumes, k8sv1.Volume{
			Name: hotplugDisks,
			VolumeSource: k8sv1.VolumeSource{
				EmptyDir: &k8sv1.EmptyDirVolumeSource{},
			},
		})
	}

	for k, v := range vmi.Spec.NodeSelector {
		nodeSelector[k] = v

	}
	if t.clusterConfig.CPUNodeDiscoveryEnabled() {
		if cpuModelLabel, err := CPUModelLabelFromCPUModel(vmi); err == nil {
			if vmi.Spec.Domain.CPU.Model != v1.CPUModeHostModel && vmi.Spec.Domain.CPU.Model != v1.CPUModeHostPassthrough {
				nodeSelector[cpuModelLabel] = "true"
			}
		}
		for _, cpuFeatureLable := range CPUFeatureLabelsFromCPUFeatures(vmi) {
			nodeSelector[cpuFeatureLable] = "true"
		}
	}
	if t.clusterConfig.HypervStrictCheckEnabled() {
		hvNodeSelectors := getHypervNodeSelectors(vmi)
		for k, v := range hvNodeSelectors {
			nodeSelector[k] = v
		}
	}

	if vmi.Status.TopologyHints != nil {
		if vmi.Status.TopologyHints.TSCFrequency != nil {
			nodeSelector[topology.ToTSCSchedulableLabel(*vmi.Status.TopologyHints.TSCFrequency)] = "true"
		}
	}

	nodeSelector[v1.NodeSchedulable] = "true"
	nodeSelectors := t.clusterConfig.GetNodeSelectors()
	for k, v := range nodeSelectors {
		nodeSelector[k] = v
	}

	hostName := dns.SanitizeHostname(vmi)

	podLabels := map[string]string{}

	for k, v := range vmi.Labels {
		podLabels[k] = v
	}
	podLabels[v1.AppLabel] = "virt-launcher"
	podLabels[v1.CreatedByLabel] = string(vmi.UID)
	podLabels[v1.VirtualMachineNameLabel] = hostName

	for i, requestedHookSidecar := range requestedHookSidecarList {
		resources := k8sv1.ResourceRequirements{}
		// add default cpu and memory limits to enable cpu pinning if requested
		// TODO(vladikr): make the hookSidecar express resources
		if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
			resources.Limits = make(k8sv1.ResourceList)
			resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("200m")
			resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("64M")
		}
		sidecar := k8sv1.Container{
			Name:            fmt.Sprintf("hook-sidecar-%d", i),
			Image:           requestedHookSidecar.Image,
			ImagePullPolicy: requestedHookSidecar.ImagePullPolicy,
			Command:         requestedHookSidecar.Command,
			Args:            requestedHookSidecar.Args,
			Resources:       resources,
			SecurityContext: &k8sv1.SecurityContext{
				RunAsUser:  &userId,
				Privileged: &privileged,
			},
			VolumeMounts: []k8sv1.VolumeMount{
				{
					Name:      hookSidecarSocks,
					MountPath: hooks.HookSocketsSharedDirectory,
				},
			},
		}
		if nonRoot {
			sidecar.SecurityContext.RunAsGroup = &userId
			sidecar.SecurityContext.RunAsNonRoot = &nonRoot
		}
		containers = append(containers, sidecar)
	}

	podAnnotations, err := generatePodAnnotations(vmi)
	if err != nil {
		return nil, err
	}
	if tempPod {
		// mark pod as temp - only used for provisioning
		podAnnotations[v1.EphemeralProvisioningObject] = "true"
	}

	var initContainers []k8sv1.Container

	if HaveContainerDiskVolume(vmi.Spec.Volumes) || util.HasKernelBootContainerImage(vmi) {

		initContainerVolumeMounts := []k8sv1.VolumeMount{
			{
				Name:      virtBinDir,
				MountPath: "/init/usr/bin",
			},
		}

		initContainerResources := k8sv1.ResourceRequirements{}
		if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
			initContainerResources.Limits = make(k8sv1.ResourceList)
			initContainerResources.Limits[k8sv1.ResourceCPU] = resource.MustParse("10m")
			initContainerResources.Limits[k8sv1.ResourceMemory] = resource.MustParse("40M")
			initContainerResources.Requests = make(k8sv1.ResourceList)
			initContainerResources.Requests[k8sv1.ResourceCPU] = resource.MustParse("10m")
			initContainerResources.Requests[k8sv1.ResourceMemory] = resource.MustParse("40M")
		} else {
			initContainerResources.Limits = make(k8sv1.ResourceList)
			initContainerResources.Limits[k8sv1.ResourceCPU] = resource.MustParse("100m")
			initContainerResources.Limits[k8sv1.ResourceMemory] = resource.MustParse("40M")
			initContainerResources.Requests = make(k8sv1.ResourceList)
			initContainerResources.Requests[k8sv1.ResourceCPU] = resource.MustParse("10m")
			initContainerResources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
		}
		initContainerCommand := []string{"/usr/bin/cp",
			"/usr/bin/container-disk",
			"/init/usr/bin/container-disk",
		}
		cpInitContainer := k8sv1.Container{
			Name:            "container-disk-binary",
			Image:           t.launcherImage,
			ImagePullPolicy: imagePullPolicy,
			SecurityContext: &k8sv1.SecurityContext{
				RunAsUser:  &userId,
				Privileged: &privileged,
			},
			Command:      initContainerCommand,
			VolumeMounts: initContainerVolumeMounts,
			Resources:    initContainerResources,
		}
		if nonRoot {
			cpInitContainer.SecurityContext.RunAsGroup = &userId
			cpInitContainer.SecurityContext.RunAsNonRoot = &nonRoot
		}

		initContainers = append(initContainers, cpInitContainer)

		// this causes containerDisks to be pre-pulled before virt-launcher starts.
		initContainers = append(initContainers, containerdisk.GenerateInitContainers(vmi, imageIDs, containerDisks, virtBinDir)...)

		kernelBootInitContainer := containerdisk.GenerateKernelBootInitContainer(vmi, imageIDs, containerDisks, virtBinDir)
		if kernelBootInitContainer != nil {
			initContainers = append(initContainers, *kernelBootInitContainer)
		}
	}

	readinessGates := []k8sv1.PodReadinessGate{
		{
			ConditionType: v1.VirtualMachineUnpaused,
		},
	}

	// TODO use constants for podLabels
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-",
			Labels:       podLabels,
			Annotations:  podAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
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
			InitContainers:                initContainers,
			NodeSelector:                  nodeSelector,
			Volumes:                       volumes,
			ImagePullSecrets:              imagePullSecrets,
			DNSConfig:                     vmi.Spec.DNSConfig,
			DNSPolicy:                     vmi.Spec.DNSPolicy,
			ReadinessGates:                readinessGates,
		},
	}

	if nonRoot {
		if util.HasHugePages(vmi) {
			pod.Spec.SecurityContext.FSGroup = &userId
		}
		pod.Spec.SecurityContext.RunAsGroup = &userId
		pod.Spec.SecurityContext.RunAsNonRoot = &nonRoot
	}

	// If an SELinux type was specified, use that--otherwise don't set an SELinux type
	selinuxType := t.clusterConfig.GetSELinuxLauncherType()
	if selinuxType != "" {
		alignPodMultiCategorySecurity(&pod, selinuxType)
	}

	// If we have a runtime class specified, use it, otherwise don't set a runtimeClassName
	runtimeClassName := t.clusterConfig.GetDefaultRuntimeClass()
	if runtimeClassName != "" {
		pod.Spec.RuntimeClassName = &runtimeClassName
	}

	if vmi.Spec.PriorityClassName != "" {
		pod.Spec.PriorityClassName = vmi.Spec.PriorityClassName
	}

	if vmi.Spec.Affinity != nil {
		pod.Spec.Affinity = vmi.Spec.Affinity.DeepCopy()
	}

	if t.clusterConfig.CPUNodeDiscoveryEnabled() {
		SetNodeAffinityForForbiddenFeaturePolicy(vmi, &pod)
	}

	pod.Spec.Tolerations = vmi.Spec.Tolerations

	pod.Spec.SchedulerName = vmi.Spec.SchedulerName

	enableServiceLinks := false
	pod.Spec.EnableServiceLinks = &enableServiceLinks

	if len(serviceAccountName) > 0 {
		pod.Spec.ServiceAccountName = serviceAccountName
		automount := true
		pod.Spec.AutomountServiceAccountToken = &automount
	} else if istio.ProxyInjectionEnabled(vmi) {
		automount := true
		pod.Spec.AutomountServiceAccountToken = &automount
	} else {
		automount := false
		pod.Spec.AutomountServiceAccountToken = &automount
	}

	return &pod, nil
}

func validatePermittedHostDevices(spec *v1.VirtualMachineInstanceSpec, config *virtconfig.ClusterConfig) error {
	errors := make([]string, 0)

	if hostDevs := config.GetPermittedHostDevices(); hostDevs != nil {
		// build a map of all permitted host devices
		supportedHostDevicesMap := make(map[string]bool)
		for _, dev := range hostDevs.PciHostDevices {
			supportedHostDevicesMap[dev.ResourceName] = true
		}
		for _, dev := range hostDevs.MediatedDevices {
			supportedHostDevicesMap[dev.ResourceName] = true
		}
		for _, hostDev := range spec.Domain.Devices.GPUs {
			if _, exist := supportedHostDevicesMap[hostDev.DeviceName]; !exist {
				errors = append(errors, fmt.Sprintf("GPU %s is not permitted in permittedHostDevices configuration", hostDev.DeviceName))
			}
		}
		for _, hostDev := range spec.Domain.Devices.HostDevices {
			if _, exist := supportedHostDevicesMap[hostDev.DeviceName]; !exist {
				errors = append(errors, fmt.Sprintf("HostDevice %s is not permitted in permittedHostDevices configuration", hostDev.DeviceName))
			}
		}
	}

	if len(errors) != 0 {
		return fmt.Errorf(strings.Join(errors, " "))
	}

	return nil
}

func (t *templateService) RenderHotplugAttachmentPodTemplate(volumes []*v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, claimMap map[string]*k8sv1.PersistentVolumeClaim, tempPod bool) (*k8sv1.Pod, error) {
	zero := int64(0)
	sharedMount := k8sv1.MountPropagationHostToContainer
	command := []string{"/bin/sh", "-c", "/usr/bin/container-disk --copy-path /path/hp"}

	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hp-volume-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   k8sv1.SchemeGroupVersion.Group,
					Version: k8sv1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
			Labels: map[string]string{
				v1.AppLabel: hotplugDisk,
			},
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:    hotplugDisk,
					Image:   t.launcherImage,
					Command: command,
					Resources: k8sv1.ResourceRequirements{ //Took the request and limits from containerDisk init container.
						Limits: map[k8sv1.ResourceName]resource.Quantity{
							k8sv1.ResourceCPU:    resource.MustParse("100m"),
							k8sv1.ResourceMemory: resource.MustParse("80M"),
						},
						Requests: map[k8sv1.ResourceName]resource.Quantity{
							k8sv1.ResourceCPU:    resource.MustParse("10m"),
							k8sv1.ResourceMemory: resource.MustParse("2M"),
						},
					},
					SecurityContext: &k8sv1.SecurityContext{
						SELinuxOptions: &k8sv1.SELinuxOptions{
							Level: "s0",
							Type:  t.clusterConfig.GetSELinuxLauncherType(),
						},
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:             hotplugDisks,
							MountPath:        "/path",
							MountPropagation: &sharedMount,
						},
					},
				},
			},
			Affinity: &k8sv1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
						NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
							{
								MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: k8sv1.NodeSelectorOpIn,
										Values:   []string{ownerPod.Spec.NodeName},
									},
								},
							},
						},
					},
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: hotplugDisks,
					VolumeSource: k8sv1.VolumeSource{
						EmptyDir: &k8sv1.EmptyDirVolumeSource{},
					},
				},
			},
			TerminationGracePeriodSeconds: &zero,
		},
	}

	hotplugVolumeStatusMap := make(map[string]v1.VolumePhase)
	for _, status := range vmi.Status.VolumeStatus {
		if status.HotplugVolume != nil {
			hotplugVolumeStatusMap[status.Name] = status.Phase
		}
	}
	for _, volume := range volumes {
		claimName := types.PVCNameFromVirtVolume(volume)
		if claimName == "" {
			continue
		}
		skipMount := false
		if hotplugVolumeStatusMap[volume.Name] == v1.VolumeReady || hotplugVolumeStatusMap[volume.Name] == v1.HotplugVolumeMounted {
			skipMount = true
		}
		pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
			Name: volume.Name,
			VolumeSource: k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		})
		if !skipMount {
			pvc := claimMap[volume.Name]
			if pvc != nil {
				if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == k8sv1.PersistentVolumeBlock {
					pod.Spec.Containers[0].VolumeDevices = append(pod.Spec.Containers[0].VolumeDevices, k8sv1.VolumeDevice{
						Name:       volume.Name,
						DevicePath: fmt.Sprintf("/path/%s/%s", volume.Name, pvc.GetUID()),
					})
					pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{
						RunAsUser: &[]int64{0}[0],
					}
				} else {
					pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
						Name:      volume.Name,
						MountPath: fmt.Sprintf("/%s", volume.Name),
					})
				}
			}
		}
	}

	return pod, nil
}

func (t *templateService) RenderHotplugAttachmentTriggerPodTemplate(volume *v1.Volume, ownerPod *k8sv1.Pod, _ *v1.VirtualMachineInstance, pvcName string, isBlock bool, tempPod bool) (*k8sv1.Pod, error) {
	zero := int64(0)
	sharedMount := k8sv1.MountPropagationHostToContainer
	var command []string
	if tempPod {
		command = []string{"/bin/bash",
			"-c",
			"exit", "0"}
	} else {
		command = []string{"/bin/sh", "-c", "/usr/bin/container-disk --copy-path /path/hp"}
	}

	annotationsList := make(map[string]string)
	if tempPod {
		// mark pod as temp - only used for provisioning
		annotationsList[v1.EphemeralProvisioningObject] = "true"
	}

	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "hp-volume-",
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(ownerPod, schema.GroupVersionKind{
					Group:   k8sv1.SchemeGroupVersion.Group,
					Version: k8sv1.SchemeGroupVersion.Version,
					Kind:    "Pod",
				}),
			},
			Labels: map[string]string{
				v1.AppLabel: hotplugDisk,
			},
			Annotations: annotationsList,
		},
		Spec: k8sv1.PodSpec{
			Containers: []k8sv1.Container{
				{
					Name:    hotplugDisk,
					Image:   t.launcherImage,
					Command: command,
					Resources: k8sv1.ResourceRequirements{ //Took the request and limits from containerDisk init container.
						Limits: map[k8sv1.ResourceName]resource.Quantity{
							k8sv1.ResourceCPU:    resource.MustParse("100m"),
							k8sv1.ResourceMemory: resource.MustParse("80M"),
						},
						Requests: map[k8sv1.ResourceName]resource.Quantity{
							k8sv1.ResourceCPU:    resource.MustParse("10m"),
							k8sv1.ResourceMemory: resource.MustParse("2M"),
						},
					},
					SecurityContext: &k8sv1.SecurityContext{
						SELinuxOptions: &k8sv1.SELinuxOptions{
							Level: "s0",
							Type:  t.clusterConfig.GetSELinuxLauncherType(),
						},
					},
					VolumeMounts: []k8sv1.VolumeMount{
						{
							Name:             hotplugDisks,
							MountPath:        "/path",
							MountPropagation: &sharedMount,
						},
					},
				},
			},
			Affinity: &k8sv1.Affinity{
				PodAffinity: &k8sv1.PodAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: ownerPod.GetLabels(),
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: volume.Name,
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName,
							ReadOnly:  false,
						},
					},
				},
				{
					Name: hotplugDisks,
					VolumeSource: k8sv1.VolumeSource{
						EmptyDir: &k8sv1.EmptyDirVolumeSource{},
					},
				},
			},
			TerminationGracePeriodSeconds: &zero,
		},
	}

	if isBlock {
		pod.Spec.Containers[0].VolumeDevices = []k8sv1.VolumeDevice{
			{
				Name:       volume.Name,
				DevicePath: "/dev/hotplugblockdevice",
			},
		}
		pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{
			RunAsUser: &[]int64{0}[0],
		}
	} else {
		pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
			Name:      volume.Name,
			MountPath: "/pvc",
		})
	}
	return pod, nil
}

func getVirtiofsCapabilities() []k8sv1.Capability {
	return []k8sv1.Capability{
		"CHOWN",
		"DAC_OVERRIDE",
		"FOWNER",
		"FSETID",
		"SETGID",
		"SETUID",
		"MKNOD",
		"SETFCAP",
	}
}

func requireDHCP(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Bridge != nil || iface.Masquerade != nil {
			return true
		}
	}
	return false
}

func haveSlirp(vmi *v1.VirtualMachineInstance) bool {
	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Slirp != nil {
			return true
		}
	}
	return false
}

func getRequiredCapabilities(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig) []k8sv1.Capability {
	if util.IsNonRootVMI(vmi) {
		return []k8sv1.Capability{CAP_NET_BIND_SERVICE}
	}
	capabilities := []k8sv1.Capability{}
	if requireDHCP(vmi) || haveSlirp(vmi) {
		capabilities = append(capabilities, CAP_NET_BIND_SERVICE)
	}
	// add a CAP_SYS_NICE capability to allow setting cpu affinity
	capabilities = append(capabilities, CAP_SYS_NICE)
	// add CAP_SYS_ADMIN capability to allow virtiofs
	if util.IsVMIVirtiofsEnabled(vmi) {
		capabilities = append(capabilities, CAP_SYS_ADMIN)
		capabilities = append(capabilities, getVirtiofsCapabilities()...)
	}
	return capabilities
}

func getRequiredResources(vmi *v1.VirtualMachineInstance, allowEmulation bool) k8sv1.ResourceList {
	res := k8sv1.ResourceList{}
	if util.NeedTunDevice(vmi) {
		res[TunDevice] = resource.MustParse("1")
	}
	if util.NeedVirtioNetDevice(vmi, allowEmulation) {
		// Note that about network interface, allowEmulation does not make
		// any difference on eventual Domain xml, but uniformly making
		// /dev/vhost-net unavailable and libvirt implicitly fallback
		// to use QEMU userland NIC emulation.
		res[VhostNetDevice] = resource.MustParse("1")

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

// GetMemoryOverhead computes the estimation of total
// memory needed for the domain to operate properly.
// This includes the memory needed for the guest and memory
// for Qemu and OS overhead.
//
// The return value is overhead memory quantity
//
// Note: This is the best estimation we were able to come up with
//       and is still not 100% accurate
func GetMemoryOverhead(vmi *v1.VirtualMachineInstance, cpuArch string) *resource.Quantity {
	domain := vmi.Spec.Domain
	vmiMemoryReq := domain.Resources.Requests.Memory()

	overhead := resource.NewScaledQuantity(0, resource.Kilo)

	// Add the memory needed for pagetables (one bit for every 512b of RAM size)
	pagetableMemory := resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo)
	pagetableMemory.Set(pagetableMemory.Value() / 512)
	overhead.Add(*pagetableMemory)

	// Add fixed overhead for shared libraries and such
	// TODO account for the overhead of kubevirt components running in the pod
	overhead.Add(resource.MustParse("138Mi"))

	// Add CPU table overhead (8 MiB per vCPU and 8 MiB per IO thread)
	// overhead per vcpu in MiB
	coresMemory := resource.MustParse("8Mi")
	var vcpus int64
	if domain.CPU != nil {
		vcpus = hardware.GetNumberOfVCPUs(domain.CPU)
	} else {
		// Currently, a default guest CPU topology is set by the API webhook mutator, if not set by a user.
		// However, this wasn't always the case.
		// In case when the guest topology isn't set, take value from resources request or limits.
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			vcpus = cpuLimit.Value()
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			vcpus = cpuRequests.Value()
		}
	}

	// if neither CPU topology nor request or limits provided, set vcpus to 1
	if vcpus < 1 {
		vcpus = 1
	}
	value := coresMemory.Value() * vcpus
	coresMemory = *resource.NewQuantity(value, coresMemory.Format)
	overhead.Add(coresMemory)

	// static overhead for IOThread
	overhead.Add(resource.MustParse("8Mi"))

	// Add video RAM overhead
	if domain.Devices.AutoattachGraphicsDevice == nil || *domain.Devices.AutoattachGraphicsDevice == true {
		overhead.Add(resource.MustParse("16Mi"))
	}

	// When use uefi boot on aarch64 with edk2 package, qemu will create 2 pflash(64Mi each, 128Mi in total)
	// it should be considered for memory overhead
	// Additional information can be found here: https://github.com/qemu/qemu/blob/master/hw/arm/virt.c#L120
	if cpuArch == "arm64" {
		overhead.Add(resource.MustParse("128Mi"))
	}

	// Additional overhead of 1G for VFIO devices. VFIO requires all guest RAM to be locked
	// in addition to MMIO memory space to allow DMA. 1G is often the size of reserved MMIO space on x86 systems.
	// Additial information can be found here: https://www.redhat.com/archives/libvir-list/2015-November/msg00329.html
	if util.IsVFIOVMI(vmi) {
		overhead.Add(resource.MustParse("1Gi"))
	}

	// DownardMetrics volumes are using emptyDirs backed by memory.
	// the max. disk size is only 256Ki.
	if downwardmetrics.HasDownwardMetricDisk(vmi) {
		overhead.Add(resource.MustParse("1Mi"))
	}

	addProbeOverheads(vmi, overhead)

	// Consider memory overhead for SEV guests.
	// Additional information can be found here: https://libvirt.org/kbase/launch_security_sev.html#memory
	if util.IsSEVVMI(vmi) {
		overhead.Add(resource.MustParse("256Mi"))
	}

	return overhead
}

// We need to add this overhead due to potential issues when using exec probes.
// In certain situations depending on things like node size and kernel versions
// the exec probe can cause a significant memory overhead that results in the pod getting OOM killed.
// To prevent this, we add this overhead until we have a better way of doing exec probes.
// The virtProbeTotalAdditionalOverhead is added for the virt-probe binary we use for probing and
// only added once, while the virtProbeOverhead is the general memory consumption of virt-probe
// that we add per added probe.
var virtProbeTotalAdditionalOverhead = resource.MustParse("100Mi")
var virtProbeOverhead = resource.MustParse("10Mi")

func addProbeOverheads(vmi *v1.VirtualMachineInstance, to *resource.Quantity) {
	hasLiveness := addProbeOverhead(vmi.Spec.LivenessProbe, to)
	hasReadiness := addProbeOverhead(vmi.Spec.ReadinessProbe, to)
	if hasLiveness || hasReadiness {
		to.Add(virtProbeTotalAdditionalOverhead)
	}
}

func addProbeOverhead(probe *v1.Probe, to *resource.Quantity) bool {
	if probe != nil && probe.Exec != nil {
		to.Add(virtProbeOverhead)
		return true
	}
	return false
}

func updateReadinessProbe(vmi *v1.VirtualMachineInstance, computeProbe *k8sv1.Probe) {
	if vmi.Spec.ReadinessProbe.GuestAgentPing != nil {
		wrapGuestAgentPingWithVirtProbe(vmi, computeProbe)
		computeProbe.InitialDelaySeconds = computeProbe.InitialDelaySeconds + LibvirtStartupDelay
		return
	}
	wrapExecProbeWithVirtProbe(vmi, computeProbe)
	computeProbe.InitialDelaySeconds = computeProbe.InitialDelaySeconds + LibvirtStartupDelay
}

func updateLivenessProbe(vmi *v1.VirtualMachineInstance, computeProbe *k8sv1.Probe) {
	if vmi.Spec.LivenessProbe.GuestAgentPing != nil {
		wrapGuestAgentPingWithVirtProbe(vmi, computeProbe)
		computeProbe.InitialDelaySeconds = computeProbe.InitialDelaySeconds + LibvirtStartupDelay
		return
	}
	wrapExecProbeWithVirtProbe(vmi, computeProbe)
	computeProbe.InitialDelaySeconds = computeProbe.InitialDelaySeconds + LibvirtStartupDelay
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

func HaveMasqueradeInterface(interfaces []v1.Interface) bool {
	for _, iface := range interfaces {
		if iface.Masquerade != nil {
			return true
		}
	}

	return false
}

func HaveContainerDiskVolume(volumes []v1.Volume) bool {
	for _, volume := range volumes {
		if volume.ContainerDisk != nil {
			return true
		}
	}
	return false
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
			crd, err := virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(namespace).Get(context.Background(), networkName, metav1.GetOptions{})
			if err != nil {
				return map[string]string{}, fmt.Errorf("Failed to locate network attachment definition %s/%s", namespace, networkName)
			}
			networkToResourceMap[network.Name] = getResourceNameForNetwork(crd)
		}
	}
	return
}

func NewTemplateService(launcherImage string,
	launcherQemuTimeout int,
	virtShareDir string,
	virtLibDir string,
	ephemeralDiskDir string,
	containerDiskDir string,
	hotplugDiskDir string,
	imagePullSecret string,
	persistentVolumeClaimCache cache.Store,
	virtClient kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
	launcherSubGid int64) TemplateService {

	precond.MustNotBeEmpty(launcherImage)
	svc := templateService{
		launcherImage:              launcherImage,
		launcherQemuTimeout:        launcherQemuTimeout,
		virtShareDir:               virtShareDir,
		virtLibDir:                 virtLibDir,
		ephemeralDiskDir:           ephemeralDiskDir,
		containerDiskDir:           containerDiskDir,
		hotplugDiskDir:             hotplugDiskDir,
		imagePullSecret:            imagePullSecret,
		persistentVolumeClaimStore: persistentVolumeClaimCache,
		virtClient:                 virtClient,
		clusterConfig:              clusterConfig,
		launcherSubGid:             launcherSubGid,
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
		ProbeHandler: k8sv1.ProbeHandler{
			Exec:      probe.Exec,
			HTTPGet:   probe.HTTPGet,
			TCPSocket: probe.TCPSocket,
		},
	}
}

func wrapGuestAgentPingWithVirtProbe(vmi *v1.VirtualMachineInstance, probe *k8sv1.Probe) {
	pingCommand := []string{
		"virt-probe",
		"--domainName", api.VMINamespaceKeyFunc(vmi),
		"--timeoutSeconds", strconv.FormatInt(int64(probe.TimeoutSeconds), 10),
		"--guestAgentPing",
	}
	probe.ProbeHandler.Exec = &k8sv1.ExecAction{Command: pingCommand}
	// we add 1s to the pod probe to compensate for the additional steps in probing
	probe.TimeoutSeconds += 1
	return
}

func wrapExecProbeWithVirtProbe(vmi *v1.VirtualMachineInstance, probe *k8sv1.Probe) {
	if probe == nil || probe.ProbeHandler.Exec == nil {
		return
	}

	originalCommand := probe.ProbeHandler.Exec.Command
	if len(originalCommand) < 1 {
		return
	}

	wrappedCommand := []string{
		"virt-probe",
		"--domainName", api.VMINamespaceKeyFunc(vmi),
		"--timeoutSeconds", strconv.FormatInt(int64(probe.TimeoutSeconds), 10),
		"--command", originalCommand[0],
		"--",
	}
	wrappedCommand = append(wrappedCommand, originalCommand[1:]...)

	probe.ProbeHandler.Exec.Command = wrappedCommand
	// we add 1s to the pod probe to compensate for the additional steps in probing
	probe.TimeoutSeconds += 1
}

func alignPodMultiCategorySecurity(pod *k8sv1.Pod, selinuxType string) {
	pod.Spec.SecurityContext.SELinuxOptions = &k8sv1.SELinuxOptions{Type: selinuxType}
	// more info on https://github.com/kubernetes/kubernetes/issues/90759
	// Since the compute container needs to be able to communicate with the
	// rest of the pod, we loop over all the containers and remove their SELinux
	// categories.
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		if container.Name != "compute" {
			generateContainerSecurityContext(selinuxType, container)
		}
	}
}

func generateContainerSecurityContext(selinuxType string, container *k8sv1.Container) {
	if container.SecurityContext == nil {
		container.SecurityContext = &k8sv1.SecurityContext{}
	}
	if container.SecurityContext.SELinuxOptions == nil {
		container.SecurityContext.SELinuxOptions = &k8sv1.SELinuxOptions{}
	}
	container.SecurityContext.SELinuxOptions.Type = selinuxType
	container.SecurityContext.SELinuxOptions.Level = "s0"
}

func generatePodAnnotations(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	annotationsSet := map[string]string{
		v1.DomainAnnotation: vmi.GetObjectMeta().GetName(),
	}
	for k, v := range filterVMIAnnotationsForPod(vmi.Annotations) {
		annotationsSet[k] = v
	}

	annotationsSet[podcmd.DefaultContainerAnnotationName] = "compute"

	multusAnnotation, err := generateMultusCNIAnnotation(vmi)
	if err != nil {
		return nil, err
	}
	if multusAnnotation != "" {
		annotationsSet[MultusNetworksAnnotation] = multusAnnotation
	}

	if multusDefaultNetwork := lookupMultusDefaultNetworkName(vmi.Spec.Networks); multusDefaultNetwork != "" {
		annotationsSet[MULTUS_DEFAULT_NETWORK_CNI_ANNOTATION] = multusDefaultNetwork
	}

	if HaveMasqueradeInterface(vmi.Spec.Domain.Devices.Interfaces) {
		annotationsSet[ISTIO_KUBEVIRT_ANNOTATION] = "k6t-eth0"
	}
	annotationsSet[VELERO_PREBACKUP_HOOK_CONTAINER_ANNOTATION] = "compute"
	annotationsSet[VELERO_PREBACKUP_HOOK_COMMAND_ANNOTATION] = fmt.Sprintf(
		"[\"/usr/bin/virt-freezer\", \"--freeze\", \"--name\", \"%s\", \"--namespace\", \"%s\"]",
		vmi.GetObjectMeta().GetName(),
		vmi.GetObjectMeta().GetNamespace())
	annotationsSet[VELERO_POSTBACKUP_HOOK_CONTAINER_ANNOTATION] = "compute"
	annotationsSet[VELERO_POSTBACKUP_HOOK_COMMAND_ANNOTATION] = fmt.Sprintf(
		"[\"/usr/bin/virt-freezer\", \"--unfreeze\", \"--name\", \"%s\", \"--namespace\", \"%s\"]",
		vmi.GetObjectMeta().GetName(),
		vmi.GetObjectMeta().GetNamespace())

	// Set this annotation now to indicate that the newly created virt-launchers will use
	// unix sockets as a transport for migration
	annotationsSet[v1.MigrationTransportUnixAnnotation] = "true"
	return annotationsSet, nil
}

func lookupMultusDefaultNetworkName(networks []v1.Network) string {
	for _, network := range networks {
		if network.Multus != nil && network.Multus.Default {
			return network.Multus.NetworkName
		}
	}
	return ""
}

func filterVMIAnnotationsForPod(vmiAnnotations map[string]string) map[string]string {
	annotationsList := map[string]string{}
	for k, v := range vmiAnnotations {
		if strings.HasPrefix(k, "kubectl.kubernetes.io") ||
			strings.HasPrefix(k, "kubevirt.io/storage-observed-api-version") ||
			strings.HasPrefix(k, "kubevirt.io/latest-observed-api-version") {
			continue
		}
		annotationsList[k] = v
	}
	return annotationsList
}

func checkForKeepLauncherAfterFailure(vmi *v1.VirtualMachineInstance) bool {
	keepLauncherAfterFailure := false
	for k, v := range vmi.Annotations {
		if strings.HasPrefix(k, v1.KeepLauncherAfterFailureAnnotation) {
			if v == "" || strings.HasPrefix(v, "true") {
				keepLauncherAfterFailure = true
				break
			}
		}
	}
	return keepLauncherAfterFailure
}
