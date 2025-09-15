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

package services

import (
	"context"
	"fmt"
	"maps"
	"math/rand"
	"os"
	"strconv"
	"strings"

	"github.com/openshift/library-go/pkg/build/naming"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubectl/pkg/cmd/util/podcmd"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	drautil "kubevirt.io/kubevirt/pkg/dra"
	"kubevirt.io/kubevirt/pkg/pointer"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/hooks"
	metrics "kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-controller"
	"kubevirt.io/kubevirt/pkg/network/downwardapi"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/multus"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/descheduler"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

const (
	containerDisks               = "container-disks"
	hotplugDisks                 = "hotplug-disks"
	hookSidecarSocks             = "hook-sidecar-sockets"
	varRun                       = "/var/run"
	virtBinDir                   = "virt-bin-share-dir"
	hotplugDisk                  = "d8v-hotplug-disk"
	virtExporter                 = "virt-exporter"
	hotplugContainerDisks        = "hotplug-container-disks"
	HotplugContainerDisk         = "hotplug-container-disk-"
	varLog                       = "/var/log"
	etcLibvirt                   = "/etc/libvirt"
	varLibLibvirt                = "/var/lib/libvirt"
	varCacheLibvirt              = "/var/cache/libvirt"
	tmp                          = "/tmp"
	varLibSWTPMLocalCA           = "/var/lib/swtpm-localca"
	varLogVolumeName             = "var-log"
	etcLibvirtVolumeName         = "etc-libvirt"
	varLibLibvirtVolumeName      = "var-lib-libvirt"
	varCacheLibvirtVolumeName    = "var-cache-libvirt"
	varRunVolumeName             = "var-run"
	tmpVolumeName                = "tmp"
	varLibSWTPMLocalCAVolumeName = "var-lib-swtpm-localca"
)

const KvmDevice = "devices.virtualization.deckhouse.io/kvm"
const TunDevice = "devices.virtualization.deckhouse.io/tun"
const VhostNetDevice = "devices.virtualization.deckhouse.io/vhost-net"
const SevDevice = "devices.virtualization.deckhouse.io/sev"
const VhostVsockDevice = "devices.virtualization.deckhouse.io/vhost-vsock"
const PrDevice = "devices.virtualization.deckhouse.io/pr-helper"

const debugLogs = "debugLogs"
const logVerbosity = "logVerbosity"
const virtiofsDebugLogs = "virtiofsdDebugLogs"

const qemuTimeoutJitterRange = 120

const (
	CAP_NET_BIND_SERVICE = "NET_BIND_SERVICE"
	CAP_SYS_NICE         = "SYS_NICE"
)

// LibvirtStartupDelay is added to custom liveness and readiness probes initial delay value.
// Libvirt needs roughly 10 seconds to start.
const LibvirtStartupDelay = 10

const IntelVendorName = "Intel"

const ENV_VAR_LIBVIRT_DEBUG_LOGS = "LIBVIRT_DEBUG_LOGS"
const ENV_VAR_VIRTIOFSD_DEBUG_LOGS = "VIRTIOFSD_DEBUG_LOGS"
const ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY = "VIRT_LAUNCHER_LOG_VERBOSITY"
const ENV_VAR_SHARED_FILESYSTEM_PATHS = "SHARED_FILESYSTEM_PATHS"

const ENV_VAR_POD_NAME = "POD_NAME"

// extensive log verbosity threshold after which libvirt debug logs will be enabled
const EXT_LOG_VERBOSITY_THRESHOLD = 5

const ephemeralStorageOverheadSize = "50M"

const (
	VirtLauncherMonitorOverhead = "25Mi"  // The `ps` RSS for virt-launcher-monitor
	VirtLauncherOverhead        = "100Mi" // The `ps` RSS for the virt-launcher process
	VirtlogdOverhead            = "25Mi"  // The `ps` RSS for virtlogd
	VirtqemudOverhead           = "40Mi"  // The `ps` RSS for virtqemud
	QemuOverhead                = "30Mi"  // The `ps` RSS for qemu, minus the RAM of its (stressed) guest, minus the virtual page table
	// Default: limits.memory = 2*requests.memory
	DefaultMemoryLimitOverheadRatio = float64(2.0)

	FailedToRenderLaunchManifestErrFormat = "failed to render launch manifest: %v"
)

type netBindingPluginMemoryCalculator interface {
	Calculate(vmi *v1.VirtualMachineInstance, registeredPlugins map[string]v1.InterfaceBindingPlugin) resource.Quantity
}

type annotationsGenerator interface {
	Generate(vmi *v1.VirtualMachineInstance) (map[string]string, error)
}

type targetAnnotationsGenerator interface {
	GenerateFromSource(vmi *v1.VirtualMachineInstance, sourcePod *k8sv1.Pod) (map[string]string, error)
}

type TemplateService interface {
	RenderMigrationManifest(vmi *v1.VirtualMachineInstance, migration *v1.VirtualMachineInstanceMigration, sourcePod *k8sv1.Pod) (*k8sv1.Pod, error)
	RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error)
	RenderHotplugAttachmentPodTemplate(volumes []*v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, claimMap map[string]*k8sv1.PersistentVolumeClaim) (*k8sv1.Pod, error)
	RenderHotplugAttachmentTriggerPodTemplate(volume *v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, pvcName string, isBlock bool, tempPod bool) (*k8sv1.Pod, error)
	RenderLaunchManifestNoVm(*v1.VirtualMachineInstance) (*k8sv1.Pod, error)
	RenderExporterManifest(vmExport *exportv1.VirtualMachineExport, namePrefix string) *k8sv1.Pod
	GetLauncherImage() string
	IsPPC64() bool
}

type templateService struct {
	launcherImage              string
	exporterImage              string
	launcherQemuTimeout        int
	virtShareDir               string
	ephemeralDiskDir           string
	containerDiskDir           string
	hotplugDiskDir             string
	imagePullSecret            string
	persistentVolumeClaimStore cache.Store
	virtClient                 kubecli.KubevirtClient
	clusterConfig              *virtconfig.ClusterConfig
	launcherSubGid             int64
	resourceQuotaStore         cache.Store
	namespaceStore             cache.Store

	sidecarCreators                  []SidecarCreatorFunc
	netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator
	annotationsGenerators            []annotationsGenerator
	netTargetAnnotationsGenerator    targetAnnotationsGenerator
}

func isFeatureStateEnabled(fs *v1.FeatureState) bool {
	return fs != nil && fs.Enabled != nil && *fs.Enabled
}

func setNodeAffinityForPod(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {
	setNodeAffinityForHostModelCpuModel(vmi, pod)
	setNodeAffinityForbiddenFeaturePolicy(vmi, pod)
}

func setNodeAffinityForHostModelCpuModel(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Model == "" || vmi.Spec.Domain.CPU.Model == v1.CPUModeHostModel {
		pod.Spec.Affinity = modifyNodeAffintyToRejectLabel(pod.Spec.Affinity, v1.NodeHostModelIsObsoleteLabel)
	}
}

func setNodeAffinityForbiddenFeaturePolicy(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) {
	if vmi.Spec.Domain.CPU == nil || vmi.Spec.Domain.CPU.Features == nil {
		return
	}

	for _, feature := range vmi.Spec.Domain.CPU.Features {
		if feature.Policy == "forbid" {
			pod.Spec.Affinity = modifyNodeAffintyToRejectLabel(pod.Spec.Affinity, v1.CPUFeatureLabel+feature.Name)
		}
	}
}

func modifyNodeAffintyToRejectLabel(origAffinity *k8sv1.Affinity, labelToReject string) *k8sv1.Affinity {
	affinity := origAffinity.DeepCopy()
	requirement := k8sv1.NodeSelectorRequirement{
		Key:      labelToReject,
		Operator: k8sv1.NodeSelectorOpDoesNotExist,
	}
	term := k8sv1.NodeSelectorTerm{
		MatchExpressions: []k8sv1.NodeSelectorRequirement{requirement}}

	nodeAffinity := &k8sv1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
			NodeSelectorTerms: []k8sv1.NodeSelectorTerm{term},
		},
	}
	if affinity != nil && affinity.NodeAffinity != nil {
		if affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			terms := affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
			// Since NodeSelectorTerms are ORed , the anti affinity requirement will be added to each term.
			for i, selectorTerm := range terms {
				affinity.NodeAffinity.
					RequiredDuringSchedulingIgnoredDuringExecution.
					NodeSelectorTerms[i].MatchExpressions = append(selectorTerm.MatchExpressions, requirement)
			}
		} else {
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{term},
			}
		}

	} else if affinity != nil {
		affinity.NodeAffinity = nodeAffinity
	} else {
		affinity = &k8sv1.Affinity{
			NodeAffinity: nodeAffinity,
		}
	}
	return affinity
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

func (t *templateService) GetLauncherImage() string {
	return t.launcherImage
}

func (t *templateService) RenderLaunchManifestNoVm(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	backendStoragePVCName := ""
	if backendstorage.IsBackendStorageNeededForVMI(&vmi.Spec) {
		backendStoragePVC := backendstorage.PVCForVMI(t.persistentVolumeClaimStore, vmi)
		if backendStoragePVC == nil {
			return nil, fmt.Errorf("can't generate manifest without backend-storage PVC, waiting for the PVC to be created")
		}
		backendStoragePVCName = backendStoragePVC.Name
	}
	return t.renderLaunchManifest(vmi, nil, backendStoragePVCName, true)
}

func (t *templateService) RenderMigrationManifest(vmi *v1.VirtualMachineInstance, migration *v1.VirtualMachineInstanceMigration, sourcePod *k8sv1.Pod) (*k8sv1.Pod, error) {
	reproducibleImageIDs, err := containerdisk.ExtractImageIDsFromSourcePod(vmi, sourcePod)
	if err != nil {
		return nil, fmt.Errorf("can not proceed with the migration when no reproducible image digest can be detected: %v", err)
	}
	backendStoragePVCName := ""
	if backendstorage.IsBackendStorageNeededForVMI(&vmi.Spec) {
		backendStoragePVC := backendstorage.PVCForMigrationTarget(t.persistentVolumeClaimStore, migration)
		if backendStoragePVC == nil {
			return nil, fmt.Errorf("can't generate manifest without backend-storage PVC, waiting for the PVC to be created")
		}
		backendStoragePVCName = backendStoragePVC.Name
	}
	targetPod, err := t.renderLaunchManifest(vmi, reproducibleImageIDs, backendStoragePVCName, false)
	if err != nil {
		return nil, err
	}

	if t.netTargetAnnotationsGenerator != nil {
		netAnnotations, err := t.netTargetAnnotationsGenerator.GenerateFromSource(vmi, sourcePod)
		if err != nil {
			return nil, err
		}

		maps.Copy(targetPod.Annotations, netAnnotations)
	}

	return targetPod, err
}

func (t *templateService) RenderLaunchManifest(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
	backendStoragePVCName := ""
	if backendstorage.IsBackendStorageNeededForVMI(&vmi.Spec) {
		backendStoragePVC := backendstorage.PVCForVMI(t.persistentVolumeClaimStore, vmi)
		if backendStoragePVC == nil {
			return nil, fmt.Errorf("can't generate manifest without backend-storage PVC, waiting for the PVC to be created")
		}
		backendStoragePVCName = backendStoragePVC.Name
	}
	return t.renderLaunchManifest(vmi, nil, backendStoragePVCName, false)
}

func (t *templateService) IsPPC64() bool {
	return t.clusterConfig.GetClusterCPUArch() == "ppc64le"
}

func generateQemuTimeoutWithJitter(qemuTimeoutBaseSeconds int) string {
	timeout := rand.Intn(qemuTimeoutJitterRange) + qemuTimeoutBaseSeconds

	return fmt.Sprintf("%ds", timeout)
}

func computePodSecurityContext(vmi *v1.VirtualMachineInstance, seccomp *k8sv1.SeccompProfile) *k8sv1.PodSecurityContext {
	psc := &k8sv1.PodSecurityContext{}

	// virtiofs container will run unprivileged even if the pod runs as root,
	// so we need to allow the NonRootUID for virtiofsd to be able to write into the PVC
	psc.FSGroup = pointer.P(int64(util.NonRootUID))

	if util.IsNonRootVMI(vmi) {
		nonRootUser := int64(util.NonRootUID)
		psc.RunAsUser = &nonRootUser
		psc.RunAsGroup = &nonRootUser
		psc.RunAsNonRoot = pointer.P(true)
	} else {
		rootUser := int64(util.RootUser)
		psc.RunAsUser = &rootUser
	}
	psc.SeccompProfile = seccomp

	return psc
}

func (t *templateService) renderLaunchManifest(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, backendStoragePVCName string, tempPod bool) (*k8sv1.Pod, error) {
	precond.MustNotBeNil(vmi)
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())

	var userId int64 = util.RootUser

	nonRoot := util.IsNonRootVMI(vmi)
	if nonRoot {
		userId = util.NonRootUID
	}

	// Pad the virt-launcher grace period.
	// Ideally we want virt-handler to handle tearing down
	// the vmi without virt-launcher's termination forcing
	// the vmi down.
	const gracePeriodPaddingSeconds int64 = 15
	gracePeriodSeconds := gracePeriodInSeconds(vmi) + gracePeriodPaddingSeconds
	gracePeriodKillAfter := gracePeriodSeconds + gracePeriodPaddingSeconds

	imagePullSecrets := imgPullSecrets(vmi.Spec.Volumes...)
	if util.HasKernelBootContainerImage(vmi) && vmi.Spec.Domain.Firmware.KernelBoot.Container.ImagePullSecret != "" {
		imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
			Name: vmi.Spec.Domain.Firmware.KernelBoot.Container.ImagePullSecret,
		})
	}
	if t.imagePullSecret != "" {
		imagePullSecrets = appendUniqueImagePullSecret(imagePullSecrets, k8sv1.LocalObjectReference{
			Name: t.imagePullSecret,
		})
	}

	networkToResourceMap, err := multus.NetworkToResource(t.virtClient, vmi)
	if err != nil {
		return nil, err
	}
	resourceRenderer, err := t.newResourceRenderer(vmi, networkToResourceMap)
	if err != nil {
		return nil, err
	}
	resources := resourceRenderer.ResourceRequirements()

	ovmfPath := t.clusterConfig.GetOVMFPath(vmi.Spec.Architecture)

	var requestedHookSidecarList hooks.HookSidecarList
	for _, sidecarCreator := range t.sidecarCreators {
		sidecars, err := sidecarCreator(vmi, t.clusterConfig.GetConfig())
		if err != nil {
			return nil, err
		}
		requestedHookSidecarList = append(requestedHookSidecarList, sidecars...)
	}

	var command []string
	if tempPod {
		logger := log.DefaultLogger()
		logger.Infof("RUNNING doppleganger pod for %s", vmi.Name)
		command = []string{"temp_pod"}
	} else {
		command = []string{"/usr/bin/tini", "--", "/usr/bin/virt-launcher-monitor",
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
			"--disk-memory-limit", strconv.Itoa(int(t.clusterConfig.GetDiskVerification().MemoryLimit.Value())),
		}
		if nonRoot {
			command = append(command, "--run-as-nonroot")
		}
		if t.clusterConfig.ImageVolumeEnabled() {
			command = append(command, "--image-volume")
		}
		if customDebugFilters, exists := vmi.Annotations[v1.CustomLibvirtLogFiltersAnnotation]; exists {
			log.Log.Object(vmi).Infof("Applying custom debug filters for vmi %s: %s", vmi.Name, customDebugFilters)
			command = append(command, "--libvirt-log-filters", customDebugFilters)
		}
	}

	if t.clusterConfig.AllowEmulation() {
		command = append(command, "--allow-emulation")
	}

	if checkForKeepLauncherAfterFailure(vmi) {
		command = append(command, "--keep-after-failure")
	}

	_, ok := vmi.Annotations[v1.FuncTestLauncherFailFastAnnotation]
	if ok {
		command = append(command, "--simulate-crash")
	}

	volumeRenderer, err := t.newVolumeRenderer(vmi, namespace, requestedHookSidecarList, backendStoragePVCName)
	if err != nil {
		return nil, err
	}

	compute := t.newContainerSpecRenderer(vmi, volumeRenderer, resources, userId).Render(command)

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
	if !t.clusterConfig.ImageVolumeEnabled() {
		containersDisks := containerdisk.GenerateContainers(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)
		containers = append(containers, containersDisks...)

		kernelBootContainer := containerdisk.GenerateKernelBootContainer(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)
		if kernelBootContainer != nil {
			log.Log.Object(vmi).Infof("kernel boot container generated")
			containers = append(containers, *kernelBootContainer)
		}
	}

	virtiofsContainers := generateVirtioFSContainers(vmi, t.launcherImage, t.clusterConfig)
	if virtiofsContainers != nil {
		containers = append(containers, virtiofsContainers...)
	}

	var sidecarVolumes []k8sv1.Volume
	for i, requestedHookSidecar := range requestedHookSidecarList {
		sidecarContainer := newSidecarContainerRenderer(
			sidecarContainerName(i), vmi, sidecarResources(vmi, t.clusterConfig), requestedHookSidecar, userId).Render(requestedHookSidecar.Command)

		if requestedHookSidecar.ConfigMap != nil {
			cm, err := t.virtClient.CoreV1().ConfigMaps(vmi.Namespace).Get(context.TODO(), requestedHookSidecar.ConfigMap.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			volumeSource := k8sv1.VolumeSource{
				ConfigMap: &k8sv1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{Name: cm.Name},
					DefaultMode:          pointer.P(int32(0755)),
				},
			}
			vol := k8sv1.Volume{
				Name:         cm.Name,
				VolumeSource: volumeSource,
			}
			sidecarVolumes = append(sidecarVolumes, vol)
		}
		if requestedHookSidecar.PVC != nil {
			volumeSource := k8sv1.VolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: requestedHookSidecar.PVC.Name,
				},
			}
			vol := k8sv1.Volume{
				Name:         requestedHookSidecar.PVC.Name,
				VolumeSource: volumeSource,
			}
			sidecarVolumes = append(sidecarVolumes, vol)
			if requestedHookSidecar.PVC.SharedComputePath != "" {
				containers[0].VolumeMounts = append(containers[0].VolumeMounts,
					k8sv1.VolumeMount{
						Name:      requestedHookSidecar.PVC.Name,
						MountPath: requestedHookSidecar.PVC.SharedComputePath,
					})
			}
		}
		containers = append(containers, sidecarContainer)
	}

	podAnnotations, err := t.generatePodAnnotations(vmi)
	if err != nil {
		return nil, err
	}
	if tempPod {
		// mark pod as temp - only used for provisioning
		podAnnotations[v1.EphemeralProvisioningObject] = "true"
	}

	var initContainers []k8sv1.Container

	sconsolelogContainer := generateSerialConsoleLogContainer(vmi, t.launcherImage, t.clusterConfig, virtLauncherLogVerbosity)
	if sconsolelogContainer != nil {
		initContainers = append(initContainers, *sconsolelogContainer)
	}

	if !t.clusterConfig.ImageVolumeEnabled() && (HaveContainerDiskVolume(vmi.Spec.Volumes) || util.HasKernelBootContainerImage(vmi)) {
		initContainerCommand := []string{"/usr/bin/cp",
			"/usr/bin/container-disk",
			"/init/usr/bin/container-disk",
		}

		initContainers = append(
			initContainers,
			t.newInitContainerRenderer(vmi,
				initContainerVolumeMount(),
				initContainerResourceRequirementsForVMI(vmi, v1.ContainerDisk, t.clusterConfig),
				userId).Render(initContainerCommand))

		// this causes containerDisks to be pre-pulled before virt-launcher starts.
		initContainers = append(initContainers, containerdisk.GenerateInitContainers(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)...)

		kernelBootInitContainer := containerdisk.GenerateKernelBootInitContainer(vmi, t.clusterConfig, imageIDs, containerDisks, virtBinDir)
		if kernelBootInitContainer != nil {
			initContainers = append(initContainers, *kernelBootInitContainer)
		}
	}

	hostName := dns.SanitizeHostname(vmi)
	enableServiceLinks := false

	var podSeccompProfile *k8sv1.SeccompProfile = nil
	if seccompConf := t.clusterConfig.GetConfig().SeccompConfiguration; seccompConf != nil && seccompConf.VirtualMachineInstanceProfile != nil {
		vmProfile := seccompConf.VirtualMachineInstanceProfile
		if customProfile := vmProfile.CustomProfile; customProfile != nil {
			if customProfile.LocalhostProfile != nil {
				podSeccompProfile = &k8sv1.SeccompProfile{
					Type:             k8sv1.SeccompProfileTypeLocalhost,
					LocalhostProfile: customProfile.LocalhostProfile,
				}
			} else if customProfile.RuntimeDefaultProfile {
				podSeccompProfile = &k8sv1.SeccompProfile{
					Type: k8sv1.SeccompProfileTypeRuntimeDefault,
				}
			}
		}

	}

	// Set ReadOnlyRootFilesystem
	setReadOnlyRootFilesystem := func(ctrs []k8sv1.Container) {
		for i := range ctrs {
			ctr := &ctrs[i]
			if ctr.SecurityContext == nil {
				ctr.SecurityContext = &k8sv1.SecurityContext{}
			}
			ctr.SecurityContext.ReadOnlyRootFilesystem = pointer.P(true)
		}
	}
	setReadOnlyRootFilesystem(initContainers)
	setReadOnlyRootFilesystem(containers)

	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "virt-launcher-" + domain + "-",
			Labels:       podLabels(vmi, hostName),
			Annotations:  podAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmi, v1.VirtualMachineInstanceGroupVersionKind),
			},
		},
		Spec: k8sv1.PodSpec{
			Hostname:                      hostName,
			Subdomain:                     vmi.Spec.Subdomain,
			SecurityContext:               computePodSecurityContext(vmi, podSeccompProfile),
			TerminationGracePeriodSeconds: &gracePeriodKillAfter,
			RestartPolicy:                 k8sv1.RestartPolicyNever,
			Containers:                    containers,
			InitContainers:                initContainers,
			NodeSelector:                  t.newNodeSelectorRenderer(vmi).Render(),
			Volumes:                       volumeRenderer.Volumes(),
			ImagePullSecrets:              imagePullSecrets,
			DNSConfig:                     vmi.Spec.DNSConfig,
			DNSPolicy:                     vmi.Spec.DNSPolicy,
			ReadinessGates:                readinessGates(),
			EnableServiceLinks:            &enableServiceLinks,
			SchedulerName:                 vmi.Spec.SchedulerName,
			Tolerations:                   vmi.Spec.Tolerations,
			TopologySpreadConstraints:     vmi.Spec.TopologySpreadConstraints,
			ResourceClaims:                vmi.Spec.ResourceClaims,
		},
	}

	alignPodMultiCategorySecurity(&pod, t.clusterConfig.GetSELinuxLauncherType(), t.clusterConfig.DockerSELinuxMCSWorkaroundEnabled())

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

	setNodeAffinityForPod(vmi, &pod)

	serviceAccountName := serviceAccount(vmi.Spec.Volumes...)
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

	pod.Spec.Volumes = append(pod.Spec.Volumes, sidecarVolumes...)

	return &pod, nil
}

func (t *templateService) newNodeSelectorRenderer(vmi *v1.VirtualMachineInstance) *NodeSelectorRenderer {
	var opts []NodeSelectorRendererOption
	if vmi.IsCPUDedicated() {
		opts = append(opts, WithDedicatedCPU())
	}
	if t.clusterConfig.HypervStrictCheckEnabled() {
		opts = append(opts, WithHyperv(vmi.Spec.Domain.Features))
	}

	if modelLabel, err := CPUModelLabelFromCPUModel(vmi); err == nil {
		opts = append(
			opts,
			WithModelAndFeatureLabels(modelLabel, CPUFeatureLabelsFromCPUFeatures(vmi)...),
		)
	}

	var machineType string
	if vmi.Status.Machine != nil && vmi.Status.Machine.Type != "" {
		machineType = vmi.Status.Machine.Type
	} else if vmi.Spec.Domain.Machine != nil && vmi.Spec.Domain.Machine.Type != "" {
		machineType = vmi.Spec.Domain.Machine.Type
	}

	if machineType != "" {
		opts = append(opts, WithMachineType(machineType))
	}

	if topology.IsManualTSCFrequencyRequired(vmi) {
		opts = append(opts, WithTSCTimer(vmi.Status.TopologyHints.TSCFrequency))
	}

	if vmi.IsRealtimeEnabled() {
		log.Log.V(4).Info("Add realtime node label selector")
		opts = append(opts, WithRealtime())
	}
	if util.IsSEVVMI(vmi) {
		log.Log.V(4).Info("Add SEV node label selector")
		opts = append(opts, WithSEVSelector())
	}
	if isSEVESVMI(vmi) {
		log.Log.V(4).Info("Add SEV-ES node label selector")
		opts = append(opts, WithSEVESSelector())
	}
	if util.IsSecureExecutionVMI(vmi) {
		log.Log.V(4).Info("Add Secure Execution node label selector")
		opts = append(opts, WithSecureExecutionSelector())
	}

	return NewNodeSelectorRenderer(
		vmi.Spec.NodeSelector,
		t.clusterConfig.GetNodeSelectors(),
		vmi.Spec.Architecture,
		opts...,
	)
}

func initContainerVolumeMount() k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      virtBinDir,
		MountPath: "/init/usr/bin",
	}
}

func newSidecarContainerRenderer(sidecarName string, vmiSpec *v1.VirtualMachineInstance, resources k8sv1.ResourceRequirements, requestedHookSidecar hooks.HookSidecar, userId int64) *ContainerSpecRenderer {
	sidecarOpts := []Option{
		WithResourceRequirements(resources),
		WithArgs(requestedHookSidecar.Args),
		WithExtraEnvVars([]k8sv1.EnvVar{
			k8sv1.EnvVar{
				Name:  hooks.ContainerNameEnvVar,
				Value: sidecarName,
			}}),
	}

	var mounts []k8sv1.VolumeMount
	mounts = append(mounts, sidecarVolumeMount(sidecarName))
	if requestedHookSidecar.DownwardAPI == v1.DeviceInfo {
		mounts = append(mounts, mountPath(downwardapi.NetworkInfoVolumeName, downwardapi.MountPath))
	}
	if requestedHookSidecar.ConfigMap != nil {
		mounts = append(mounts, configMapVolumeMount(*requestedHookSidecar.ConfigMap))
	}
	if requestedHookSidecar.PVC != nil {
		mounts = append(mounts, pvcVolumeMount(*requestedHookSidecar.PVC))
	}
	sidecarOpts = append(sidecarOpts, WithVolumeMounts(mounts...))

	if util.IsNonRootVMI(vmiSpec) {
		sidecarOpts = append(sidecarOpts, WithNonRoot(userId))
		sidecarOpts = append(sidecarOpts, WithDropALLCapabilities())
	}
	if requestedHookSidecar.Image == "" {
		requestedHookSidecar.Image = os.Getenv(operatorutil.SidecarShimImageEnvName)
	}

	return NewContainerSpecRenderer(
		sidecarName,
		requestedHookSidecar.Image,
		requestedHookSidecar.ImagePullPolicy,
		sidecarOpts...)
}

func (t *templateService) newInitContainerRenderer(vmiSpec *v1.VirtualMachineInstance, initContainerVolumeMount k8sv1.VolumeMount, initContainerResources k8sv1.ResourceRequirements, userId int64) *ContainerSpecRenderer {
	const containerDisk = "d8v-container-disk-binary"
	cpInitContainerOpts := []Option{
		WithVolumeMounts(initContainerVolumeMount),
		WithResourceRequirements(initContainerResources),
		WithNoCapabilities(),
	}

	if util.IsNonRootVMI(vmiSpec) {
		cpInitContainerOpts = append(cpInitContainerOpts, WithNonRoot(userId))
	}
	if t.IsPPC64() {
		cpInitContainerOpts = append(cpInitContainerOpts, WithPrivileged())
	}

	return NewContainerSpecRenderer(containerDisk, t.launcherImage, t.clusterConfig.GetImagePullPolicy(), cpInitContainerOpts...)
}

func (t *templateService) newContainerSpecRenderer(vmi *v1.VirtualMachineInstance, volumeRenderer *VolumeRenderer, resources k8sv1.ResourceRequirements, userId int64) *ContainerSpecRenderer {
	computeContainerOpts := []Option{
		WithVolumeDevices(volumeRenderer.VolumeDevices()...),
		WithVolumeMounts(volumeRenderer.Mounts()...),
		WithSharedFilesystems(volumeRenderer.SharedFilesystemPaths()...),
		WithResourceRequirements(resources),
		WithPorts(vmi),
		WithCapabilities(vmi),
	}
	if util.IsNonRootVMI(vmi) {
		computeContainerOpts = append(computeContainerOpts, WithNonRoot(userId))
		computeContainerOpts = append(computeContainerOpts, WithDropALLCapabilities())
	}
	if t.IsPPC64() {
		computeContainerOpts = append(computeContainerOpts, WithPrivileged())
	}
	if vmi.Spec.ReadinessProbe != nil {
		computeContainerOpts = append(computeContainerOpts, WithReadinessProbe(vmi))
	}

	if vmi.Spec.LivenessProbe != nil {
		computeContainerOpts = append(computeContainerOpts, WithLivelinessProbe(vmi))
	}

	const computeContainerName = "d8v-compute"
	containerRenderer := NewContainerSpecRenderer(
		computeContainerName, t.launcherImage, t.clusterConfig.GetImagePullPolicy(), computeContainerOpts...)
	return containerRenderer
}

func (t *templateService) newVolumeRenderer(vmi *v1.VirtualMachineInstance, namespace string, requestedHookSidecarList hooks.HookSidecarList, backendStoragePVCName string) (*VolumeRenderer, error) {
	imageVolumeFeatureGateEnabled := t.clusterConfig.ImageVolumeEnabled()
	volumeOpts := []VolumeRendererOption{
		withVMIConfigVolumes(vmi.Spec.Domain.Devices.Disks, vmi.Spec.Volumes),
		withVMIVolumes(t.persistentVolumeClaimStore, vmi.Spec.Volumes, vmi.Status.VolumeStatus),
		withAccessCredentials(vmi.Spec.AccessCredentials),
		withBackendStorage(vmi, backendStoragePVCName),
	}
	if imageVolumeFeatureGateEnabled {
		volumeOpts = append(volumeOpts, withImageVolumes(vmi))
	}
	if len(requestedHookSidecarList) != 0 {
		volumeOpts = append(volumeOpts, withSidecarVolumes(requestedHookSidecarList))
	}

	if hasHugePages(vmi) {
		volumeOpts = append(volumeOpts, withHugepages())
	}

	if !vmi.Spec.Domain.Devices.DisableHotplug {
		volumeOpts = append(volumeOpts, withHotplugSupport(t.hotplugDiskDir))
	}

	if vmispec.BindingPluginNetworkWithDeviceInfoExist(vmi.Spec.Domain.Devices.Interfaces, t.clusterConfig.GetNetworkBindings()) ||
		vmispec.SRIOVInterfaceExist(vmi.Spec.Domain.Devices.Interfaces) {
		volumeOpts = append(volumeOpts, func(renderer *VolumeRenderer) error {
			renderer.podVolumeMounts = append(renderer.podVolumeMounts, mountPath(downwardapi.NetworkInfoVolumeName, downwardapi.MountPath))
			return nil
		})
		volumeOpts = append(volumeOpts, withNetworkDeviceInfoMapAnnotation())
	}

	if util.IsVMIVirtiofsEnabled(vmi) {
		volumeOpts = append(volumeOpts, withVirioFS())
	}

	volumeRenderer, err := NewVolumeRenderer(
		imageVolumeFeatureGateEnabled,
		namespace,
		t.ephemeralDiskDir,
		t.containerDiskDir,
		t.virtShareDir,
		volumeOpts...)

	if err != nil {
		return nil, err
	}
	return volumeRenderer, nil
}

func (t *templateService) newResourceRenderer(vmi *v1.VirtualMachineInstance, networkToResourceMap map[string]string) (*ResourceRenderer, error) {
	vmiResources := vmi.Spec.Domain.Resources
	baseOptions := []ResourceRendererOption{
		WithEphemeralStorageRequest(),
		WithVirtualizationResources(getRequiredResources(vmi, t.clusterConfig.AllowEmulation())),
	}

	if err := validatePermittedHostDevices(&vmi.Spec, t.clusterConfig); err != nil {
		return nil, err
	}

	options := append(baseOptions, t.VMIResourcePredicates(vmi, networkToResourceMap).Apply()...)
	return NewResourceRenderer(vmiResources.Limits, vmiResources.Requests, options...), nil
}

func sidecarVolumeMount(containerName string) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      hookSidecarSocks,
		MountPath: hooks.HookSocketsSharedDirectory,
		SubPath:   containerName,
	}
}

func configMapVolumeMount(v hooks.ConfigMap) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      v.Name,
		MountPath: v.HookPath,
		SubPath:   v.Key,
	}
}

func pvcVolumeMount(v hooks.PVC) k8sv1.VolumeMount {
	return k8sv1.VolumeMount{
		Name:      v.Name,
		MountPath: v.VolumePath,
	}
}

func gracePeriodInSeconds(vmi *v1.VirtualMachineInstance) int64 {
	if vmi.Spec.TerminationGracePeriodSeconds != nil {
		return *vmi.Spec.TerminationGracePeriodSeconds
	}
	return v1.DefaultGracePeriodSeconds
}

func sidecarContainerName(i int) string {
	return fmt.Sprintf("hook-sidecar-%d", i)
}

func sidecarContainerHotplugContainerdDiskName(id int) string {
	return fmt.Sprintf("%s%d", HotplugContainerDisk, id)
}
func sidecarContainerHotplugContainerdDiskVolumeName(name string) string {
	return fmt.Sprintf("%s%s", HotplugContainerDisk, name)
}

func (t *templateService) containerForHotplugContainerDisk(ctrName, volName string, cd *v1.ContainerDiskSource, vmi *v1.VirtualMachineInstance) k8sv1.Container {
	runUser := int64(util.NonRootUID)
	sharedMount := k8sv1.MountPropagationHostToContainer
	path := fmt.Sprintf("/path/%s", volName)
	command := []string{"/init/usr/bin/container-disk"}
	args := []string{"--copy-path", path}

	return k8sv1.Container{
		Name:      ctrName,
		Image:     cd.Image,
		Command:   command,
		Args:      args,
		Resources: hotplugContainerResourceRequirementsForVMI(t.clusterConfig),
		SecurityContext: &k8sv1.SecurityContext{
			AllowPrivilegeEscalation: pointer.P(false),
			ReadOnlyRootFilesystem:   pointer.P(true),
			RunAsNonRoot:             pointer.P(true),
			RunAsUser:                &runUser,
			SeccompProfile: &k8sv1.SeccompProfile{
				Type: k8sv1.SeccompProfileTypeRuntimeDefault,
			},
			Capabilities: &k8sv1.Capabilities{
				Drop: []k8sv1.Capability{"ALL"},
			},
			SELinuxOptions: &k8sv1.SELinuxOptions{
				Type:  t.clusterConfig.GetSELinuxLauncherType(),
				Level: "s0",
			},
		},
		VolumeMounts: []k8sv1.VolumeMount{
			initContainerVolumeMount(),
			{
				Name:             hotplugContainerDisks,
				MountPath:        "/path",
				MountPropagation: &sharedMount,
			},
		},
	}
}

func (t *templateService) RenderHotplugAttachmentPodTemplate(volumes []*v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, claimMap map[string]*k8sv1.PersistentVolumeClaim) (*k8sv1.Pod, error) {
	zero := int64(0)
	runUser := int64(util.NonRootUID)
	sharedMount := k8sv1.MountPropagationHostToContainer
	command := []string{"/usr/bin/container-disk", "--copy-path", "/path/hp"}

	tmpTolerations := make([]k8sv1.Toleration, len(ownerPod.Spec.Tolerations))
	copy(tmpTolerations, ownerPod.Spec.Tolerations)

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
					Name:      hotplugDisk,
					Image:     t.launcherImage,
					Command:   command,
					Resources: hotplugContainerResourceRequirementsForVMI(t.clusterConfig),
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						ReadOnlyRootFilesystem:   pointer.P(true),
						RunAsNonRoot:             pointer.P(true),
						RunAsUser:                &runUser,
						SeccompProfile: &k8sv1.SeccompProfile{
							Type: k8sv1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &k8sv1.Capabilities{
							Drop: []k8sv1.Capability{"ALL"},
						},
						SELinuxOptions: &k8sv1.SELinuxOptions{
							// If SELinux is enabled on the host, this level will be adjusted below to match the level
							// of its companion virt-launcher pod to allow it to consume our disk images.
							Type:  t.clusterConfig.GetSELinuxLauncherType(),
							Level: "s0",
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
										Key:      k8sv1.LabelHostname,
										Operator: k8sv1.NodeSelectorOpIn,
										Values:   []string{ownerPod.Spec.NodeName},
									},
								},
							},
						},
					},
				},
			},
			Tolerations:                   tmpTolerations,
			Volumes:                       []k8sv1.Volume{emptyDirVolume(hotplugDisks)},
			TerminationGracePeriodSeconds: &zero,
		},
	}
	first := true
	for i, vol := range vmi.Spec.Volumes {
		if vol.ContainerDisk == nil || !vol.ContainerDisk.Hotpluggable {
			continue
		}
		ctrName := sidecarContainerHotplugContainerdDiskName(i)
		volName := sidecarContainerHotplugContainerdDiskVolumeName(vol.Name)

		annos := pod.GetAnnotations()
		if annos == nil {
			annos = make(map[string]string)
		}
		annos[ctrName] = vol.Name

		pod.SetAnnotations(annos)
		pod.Spec.Containers = append(pod.Spec.Containers, t.containerForHotplugContainerDisk(ctrName, volName, vol.ContainerDisk, vmi))
		if first {
			first = false
			userId := int64(util.NonRootUID)
			initContainerCommand := []string{"/usr/bin/cp",
				"/usr/bin/container-disk",
				"/init/usr/bin/container-disk",
			}
			pod.Spec.InitContainers = append(
				pod.Spec.InitContainers,
				t.newInitContainerRenderer(vmi,
					initContainerVolumeMount(),
					initContainerResourceRequirementsForVMI(vmi, v1.ContainerDisk, t.clusterConfig),
					userId).Render(initContainerCommand))
			pod.Spec.Volumes = append(pod.Spec.Volumes, emptyDirVolume(hotplugContainerDisks))
			pod.Spec.Volumes = append(pod.Spec.Volumes, emptyDirVolume(virtBinDir))
		}
	}

	err := matchSELinuxLevelOfVMI(pod, vmi)
	if err != nil {
		return nil, err
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
		pvc := claimMap[volume.Name]
		if pvc == nil {
			continue
		}
		if types.IsPVCBlock(pvc.Spec.VolumeMode) {
			pod.Spec.Containers[0].VolumeDevices = append(pod.Spec.Containers[0].VolumeDevices, k8sv1.VolumeDevice{
				Name:       volume.Name,
				DevicePath: fmt.Sprintf("/path/%s/%s", volume.Name, pvc.GetUID()),
			})
		} else {
			if !skipMount {
				pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
					Name:      volume.Name,
					MountPath: fmt.Sprintf("/%s", volume.Name),
				})
			}
		}
	}

	return pod, nil
}

func (t *templateService) RenderHotplugAttachmentTriggerPodTemplate(volume *v1.Volume, ownerPod *k8sv1.Pod, vmi *v1.VirtualMachineInstance, pvcName string, isBlock bool, tempPod bool) (*k8sv1.Pod, error) {
	zero := int64(0)
	runUser := int64(util.NonRootUID)
	sharedMount := k8sv1.MountPropagationHostToContainer
	var command []string
	if tempPod {
		command = []string{"temp_pod"}
	} else {
		command = []string{"/usr/bin/container-disk", "--copy-path", "/path/hp"}
	}

	annotationsList := make(map[string]string)
	if tempPod {
		// mark pod as temp - only used for provisioning
		annotationsList[v1.EphemeralProvisioningObject] = "true"
	}

	tmpTolerations := make([]k8sv1.Toleration, len(ownerPod.Spec.Tolerations))
	copy(tmpTolerations, ownerPod.Spec.Tolerations)

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
					Name:      hotplugDisk,
					Image:     t.launcherImage,
					Command:   command,
					Resources: hotplugContainerResourceRequirementsForVMI(t.clusterConfig),
					SecurityContext: &k8sv1.SecurityContext{
						ReadOnlyRootFilesystem:   pointer.P(true),
						AllowPrivilegeEscalation: pointer.P(false),
						RunAsNonRoot:             pointer.P(true),
						RunAsUser:                &runUser,
						SeccompProfile: &k8sv1.SeccompProfile{
							Type: k8sv1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &k8sv1.Capabilities{
							Drop: []k8sv1.Capability{"ALL"},
						},
						SELinuxOptions: &k8sv1.SELinuxOptions{
							Level: "s0",
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
							TopologyKey: k8sv1.LabelHostname,
						},
					},
				},
			},
			Tolerations: tmpTolerations,
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
				emptyDirVolume(hotplugDisks),
			},
			TerminationGracePeriodSeconds: &zero,
		},
	}

	err := matchSELinuxLevelOfVMI(pod, vmi)
	if err != nil {
		return nil, err
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

func (t *templateService) RenderExporterManifest(vmExport *exportv1.VirtualMachineExport, namePrefix string) *k8sv1.Pod {
	exporterPod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			// Use of DNS1035LabelMaxLength here to align with
			// VMExportController{}.getExportPodName
			Name:      naming.GetName(namePrefix, vmExport.Name, validation.DNS1035LabelMaxLength),
			Namespace: vmExport.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(vmExport, schema.GroupVersionKind{
					Group:   exportv1.SchemeGroupVersion.Group,
					Version: exportv1.SchemeGroupVersion.Version,
					Kind:    "VirtualMachineExport",
				}),
			},
			Labels: map[string]string{
				v1.AppLabel: virtExporter,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:            "exporter",
					Image:           t.exporterImage,
					ImagePullPolicy: t.clusterConfig.GetImagePullPolicy(),
					Env: []k8sv1.EnvVar{
						{
							Name: "POD_NAME",
							ValueFrom: &k8sv1.EnvVarSource{
								FieldRef: &k8sv1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
					},
					SecurityContext: &k8sv1.SecurityContext{
						AllowPrivilegeEscalation: pointer.P(false),
						ReadOnlyRootFilesystem:   pointer.P(true),
						Capabilities:             &k8sv1.Capabilities{Drop: []k8sv1.Capability{"ALL"}},
					},
					Resources: vmExportContainerResourceRequirements(t.clusterConfig),
				},
			},
		},
	}
	return exporterPod
}

func appendUniqueImagePullSecret(secrets []k8sv1.LocalObjectReference, newsecret k8sv1.LocalObjectReference) []k8sv1.LocalObjectReference {
	for _, oldsecret := range secrets {
		if oldsecret == newsecret {
			return secrets
		}
	}
	return append(secrets, newsecret)
}

func addProbeOverheads(vmi *v1.VirtualMachineInstance, quantity *resource.Quantity) {
	// We need to add this overhead due to potential issues when using exec probes.
	// In certain situations depending on things like node size and kernel versions
	// the exec probe can cause a significant memory overhead that results in the pod getting OOM killed.
	// To prevent this, we add this overhead until we have a better way of doing exec probes.
	// The virtProbeTotalAdditionalOverhead is added for the virt-probe binary we use for probing and
	// only added once, while the virtProbeOverhead is the general memory consumption of virt-probe
	// that we add per added probe.
	virtProbeTotalAdditionalOverhead := resource.MustParse("100Mi")
	virtProbeOverhead := resource.MustParse("10Mi")
	hasLiveness := vmi.Spec.LivenessProbe != nil && vmi.Spec.LivenessProbe.Exec != nil
	hasReadiness := vmi.Spec.ReadinessProbe != nil && vmi.Spec.ReadinessProbe.Exec != nil
	if hasLiveness {
		quantity.Add(virtProbeOverhead)
	}
	if hasReadiness {
		quantity.Add(virtProbeOverhead)
	}
	if hasLiveness || hasReadiness {
		quantity.Add(virtProbeTotalAdditionalOverhead)
	}
}

func HaveContainerDiskVolume(volumes []v1.Volume) bool {
	for _, volume := range volumes {
		if volume.ContainerDisk != nil {
			return true
		}
	}
	return false
}

type templateServiceOption func(*templateService)

func NewTemplateService(launcherImage string,
	launcherQemuTimeout int,
	virtShareDir string,
	ephemeralDiskDir string,
	containerDiskDir string,
	hotplugDiskDir string,
	imagePullSecret string,
	persistentVolumeClaimCache cache.Store,
	virtClient kubecli.KubevirtClient,
	clusterConfig *virtconfig.ClusterConfig,
	launcherSubGid int64,
	exporterImage string,
	resourceQuotaStore cache.Store,
	namespaceStore cache.Store,
	opts ...templateServiceOption,
) TemplateService {

	precond.MustNotBeEmpty(launcherImage)
	log.Log.V(1).Infof("Exporter Image: %s", exporterImage)
	svc := templateService{
		launcherImage:              launcherImage,
		launcherQemuTimeout:        launcherQemuTimeout,
		virtShareDir:               virtShareDir,
		ephemeralDiskDir:           ephemeralDiskDir,
		containerDiskDir:           containerDiskDir,
		hotplugDiskDir:             hotplugDiskDir,
		imagePullSecret:            imagePullSecret,
		persistentVolumeClaimStore: persistentVolumeClaimCache,
		virtClient:                 virtClient,
		clusterConfig:              clusterConfig,
		launcherSubGid:             launcherSubGid,
		exporterImage:              exporterImage,
		resourceQuotaStore:         resourceQuotaStore,
		namespaceStore:             namespaceStore,
	}

	for _, opt := range opts {
		opt(&svc)
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

func alignPodMultiCategorySecurity(pod *k8sv1.Pod, selinuxType string, dockerSELinuxMCSWorkaround bool) {
	if selinuxType == "" && !dockerSELinuxMCSWorkaround {
		// No SELinux type and no docker workaround, nothing to do
		return
	}

	if selinuxType != "" {
		if pod.Spec.SecurityContext == nil {
			pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{}
		}
		pod.Spec.SecurityContext.SELinuxOptions = &k8sv1.SELinuxOptions{Type: selinuxType}
	}

	if dockerSELinuxMCSWorkaround {
		// more info on https://github.com/kubernetes/kubernetes/issues/90759
		// Since the compute container needs to be able to communicate with the
		// rest of the pod, we loop over all the containers and remove their SELinux
		// categories.
		// This currently only affects Docker + SELinux use-cases, and requires a
		// feature gate to be set.
		for i := range pod.Spec.Containers {
			container := &pod.Spec.Containers[i]
			if !strings.HasSuffix(container.Name, "compute") {
				generateContainerSecurityContext(selinuxType, container)
			}
		}
	}
}

func matchSELinuxLevelOfVMI(pod *k8sv1.Pod, vmi *v1.VirtualMachineInstance) error {
	if vmi.Status.SelinuxContext == "" {
		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.SourceState != nil && vmi.Status.MigrationState.SourceState.SelinuxContext != "" {
			selinuxContext := vmi.Status.MigrationState.SourceState.SelinuxContext
			if selinuxContext != "none" {
				return setSELinuxContext(selinuxContext, pod)
			}
			return nil
		}
		return fmt.Errorf("VMI is missing SELinux context")
	} else if vmi.Status.SelinuxContext != "none" {
		return setSELinuxContext(vmi.Status.SelinuxContext, pod)
	}

	return nil
}

func setSELinuxContext(selinuxContext string, pod *k8sv1.Pod) error {
	ctx := strings.Split(selinuxContext, ":")
	if len(ctx) < 4 {
		return fmt.Errorf("VMI has invalid SELinux context: %s", selinuxContext)
	}
	pod.Spec.Containers[0].SecurityContext.SELinuxOptions.Level = strings.Join(ctx[3:], ":")
	return nil
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
	container.SecurityContext.ReadOnlyRootFilesystem = pointer.P(true)
}

func (t *templateService) generatePodAnnotations(vmi *v1.VirtualMachineInstance) (map[string]string, error) {
	annotationsSet := map[string]string{
		v1.DomainAnnotation: vmi.GetObjectMeta().GetName(),
	}
	maps.Copy(annotationsSet, filterVMIAnnotationsForPod(vmi.Annotations))

	annotationsSet[podcmd.DefaultContainerAnnotationName] = "d8v-compute"

	// Set this annotation now to indicate that the newly created virt-launchers will use
	// unix sockets as a transport for migration
	annotationsSet[v1.MigrationTransportUnixAnnotation] = "true"
	annotationsSet[descheduler.EvictOnlyAnnotation] = ""

	for _, generator := range t.annotationsGenerators {
		annotations, err := generator.Generate(vmi)
		if err != nil {
			return nil, err
		}

		maps.Copy(annotationsSet, annotations)
	}

	return annotationsSet, nil
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

func (t *templateService) doesVMIRequireAutoCPULimits(vmi *v1.VirtualMachineInstance) bool {
	if t.doesVMIRequireAutoResourceLimits(vmi, k8sv1.ResourceCPU) {
		return true
	}

	labelSelector := t.clusterConfig.GetConfig().AutoCPULimitNamespaceLabelSelector
	_, limitSet := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU]
	if labelSelector == nil || limitSet {
		return false
	}
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		log.DefaultLogger().Reason(err).Warning("invalid CPULimitNamespaceLabelSelector set, assuming none")
		return false
	}

	if t.namespaceStore == nil {
		log.DefaultLogger().Reason(err).Warning("empty namespace informer")
		return false
	}

	obj, exists, err := t.namespaceStore.GetByKey(vmi.Namespace)
	if err != nil {
		log.Log.Warning("Error retrieving namespace from informer")
		return false
	} else if !exists {
		log.Log.Warningf("namespace %s does not exist.", vmi.Namespace)
		return false
	}

	ns, ok := obj.(*k8sv1.Namespace)
	if !ok {
		log.Log.Errorf("couldn't cast object to Namespace: %+v", obj)
		return false
	}

	if selector.Matches(labels.Set(ns.Labels)) {
		return true
	}

	return false
}

func (t *templateService) VMIResourcePredicates(vmi *v1.VirtualMachineInstance, networkToResourceMap map[string]string) VMIResourcePredicates {
	// Set default with vmi Architecture. compatible with multi-architecture hybrid environments
	vmiCPUArch := vmi.Spec.Architecture
	if vmiCPUArch == "" {
		vmiCPUArch = t.clusterConfig.GetClusterCPUArch()
	}
	memoryOverhead := GetMemoryOverhead(vmi, vmiCPUArch, t.clusterConfig.GetConfig().AdditionalGuestMemoryOverheadRatio)

	if t.netBindingPluginMemoryCalculator != nil {
		memoryOverhead.Add(
			t.netBindingPluginMemoryCalculator.Calculate(vmi, t.clusterConfig.GetNetworkBindings()),
		)
	}

	metrics.SetVmiLaucherMemoryOverhead(vmi, memoryOverhead)
	withCPULimits := t.doesVMIRequireAutoCPULimits(vmi)
	additionalCPUs := uint32(0)
	if vmi.Spec.Domain.IOThreadsPolicy != nil &&
		*vmi.Spec.Domain.IOThreadsPolicy == v1.IOThreadsPolicySupplementalPool &&
		vmi.Spec.Domain.IOThreads != nil &&
		vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
		additionalCPUs = *vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount
	}
	return VMIResourcePredicates{
		vmi: vmi,
		resourceRules: []VMIResourceRule{
			// Run overcommit first to avoid overcommitting overhead memory
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.GetMemoryOvercommit() != 100
			}, WithMemoryOvercommit(t.clusterConfig.GetMemoryOvercommit())),
			NewVMIResourceRule(doesVMIRequireDedicatedCPU, WithCPUPinning(vmi, vmi.Annotations, additionalCPUs)),
			NewVMIResourceRule(not(doesVMIRequireDedicatedCPU), WithoutDedicatedCPU(vmi, t.clusterConfig.GetCPUAllocationRatio(), withCPULimits)),
			NewVMIResourceRule(hasHugePages, WithHugePages(vmi.Spec.Domain.Memory, memoryOverhead)),
			NewVMIResourceRule(not(hasHugePages), WithMemoryOverhead(vmi.Spec.Domain.Resources, memoryOverhead)),
			NewVMIResourceRule(t.doesVMIRequireAutoMemoryLimits, WithAutoMemoryLimits(vmi.Namespace, t.namespaceStore)),
			NewVMIResourceRule(func(*v1.VirtualMachineInstance) bool {
				return len(networkToResourceMap) > 0
			}, WithNetworkResources(networkToResourceMap)),
			NewVMIResourceRule(isGPUVMIDevicePlugins, WithGPUsDevicePlugins(vmi.Spec.Domain.Devices.GPUs)),
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.GPUsWithDRAGateEnabled() && isGPUVMIDRA(vmi)
			}, WithGPUsDRA(vmi.Spec.Domain.Devices.GPUs)),
			NewVMIResourceRule(isHostDevVMIDevicePlugins, WithHostDevicesDevicePlugins(vmi.Spec.Domain.Devices.HostDevices)),
			NewVMIResourceRule(func(vmi *v1.VirtualMachineInstance) bool {
				return t.clusterConfig.HostDevicesWithDRAEnabled() && isHostDevVMIDRA(vmi)
			}, WithHostDevicesDRA(vmi.Spec.Domain.Devices.HostDevices)),
			NewVMIResourceRule(util.IsSEVVMI, WithSEV()),
			NewVMIResourceRule(reservation.HasVMIPersistentReservation, WithPersistentReservation()),
		},
	}
}

func (t *templateService) doesVMIRequireAutoMemoryLimits(vmi *v1.VirtualMachineInstance) bool {
	return t.doesVMIRequireAutoResourceLimits(vmi, k8sv1.ResourceMemory)
}

func (t *templateService) doesVMIRequireAutoResourceLimits(vmi *v1.VirtualMachineInstance, resource k8sv1.ResourceName) bool {
	if _, resourceLimitsExists := vmi.Spec.Domain.Resources.Limits[resource]; resourceLimitsExists {
		return false
	}

	for _, obj := range t.resourceQuotaStore.List() {
		if resourceQuota, ok := obj.(*k8sv1.ResourceQuota); ok {
			if _, exists := resourceQuota.Spec.Hard["limits."+resource]; exists && resourceQuota.Namespace == vmi.Namespace {
				return true
			}
		}
	}

	return false
}

func (p VMIResourcePredicates) Apply() []ResourceRendererOption {
	var options []ResourceRendererOption
	for _, rule := range p.resourceRules {
		if rule.predicate(p.vmi) {
			options = append(options, rule.option)
		}
	}
	return options
}

func podLabels(vmi *v1.VirtualMachineInstance, hostName string) map[string]string {
	labels := map[string]string{}

	for k, v := range vmi.Labels {
		labels[k] = v
	}
	labels[v1.AppLabel] = "virt-launcher"
	labels[v1.CreatedByLabel] = string(vmi.UID)
	labels[v1.VirtualMachineNameLabel] = hostName
	return labels
}

func readinessGates() []k8sv1.PodReadinessGate {
	return []k8sv1.PodReadinessGate{
		{
			ConditionType: v1.VirtualMachineUnpaused,
		},
	}
}

func WithNetBindingPluginMemoryCalculator(netBindingPluginMemoryCalculator netBindingPluginMemoryCalculator) templateServiceOption {
	return func(service *templateService) {
		service.netBindingPluginMemoryCalculator = netBindingPluginMemoryCalculator
	}
}

func WithAnnotationsGenerators(generators ...annotationsGenerator) templateServiceOption {
	return func(service *templateService) {
		service.annotationsGenerators = append(service.annotationsGenerators, generators...)
	}
}

func WithNetTargetAnnotationsGenerator(generator targetAnnotationsGenerator) templateServiceOption {
	return func(service *templateService) {
		service.netTargetAnnotationsGenerator = generator
	}
}

func hasHugePages(vmi *v1.VirtualMachineInstance) bool {
	return vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Hugepages != nil
}

// Check if a VMI spec requests AMD SEV-ES
func isSEVESVMI(vmi *v1.VirtualMachineInstance) bool {
	return util.IsSEVVMI(vmi) &&
		vmi.Spec.Domain.LaunchSecurity.SEV.Policy != nil &&
		vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState != nil &&
		*vmi.Spec.Domain.LaunchSecurity.SEV.Policy.EncryptedState
}

// isGPUVMIDevicePlugins checks if a VMI has any GPUs configured for device plugins
func isGPUVMIDevicePlugins(vmi *v1.VirtualMachineInstance) bool {
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if isGPUDevicePlugin(gpu) {
			return true
		}
	}
	return false
}

func isGPUDevicePlugin(gpu v1.GPU) bool {
	return gpu.DeviceName != "" && gpu.ClaimRequest == nil
}

// isGPUVMIDRA checks if a VMI has any GPUs configured for Dynamic Resource Allocation
func isGPUVMIDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if drautil.IsGPUDRA(gpu) {
			return true
		}
	}
	return false
}

// isHostDevVMIDevicePlugins checks if a VMI has any HostDevices configured for device plugins
func isHostDevVMIDevicePlugins(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.HostDevices == nil {
		return false
	}

	for _, hostDev := range vmi.Spec.Domain.Devices.HostDevices {
		if hostDev.DeviceName != "" && hostDev.ClaimRequest == nil {
			return true
		}
	}

	return false
}

// isHostDevVMIDRA checks if a VMI has any HostDevices configured for Dynamic Resource Allocation
func isHostDevVMIDRA(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.HostDevices == nil {
		return false
	}

	for _, hostDev := range vmi.Spec.Domain.Devices.HostDevices {
		if hostDev.DeviceName == "" && hostDev.ClaimRequest != nil {
			return true
		}
	}

	return false
}
