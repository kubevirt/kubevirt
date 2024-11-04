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

package containerdisk

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/safepath"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var podsBaseDir = util.KubeletPodsDir

var mountBaseDir = filepath.Join(util.VirtShareDir, "/container-disks")

type SocketPathGetter func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error)
type KernelBootSocketPathGetter func(vmi *v1.VirtualMachineInstance) (string, error)

const KernelBootName = "kernel-boot"
const KernelBootVolumeName = KernelBootName + "-volume"

const ephemeralStorageOverheadSize = "50M"

var digestRegex = regexp.MustCompile(`sha256:([a-zA-Z0-9]+)`)

func GetLegacyVolumeMountDirOnHost(vmi *v1.VirtualMachineInstance) string {
	return filepath.Join(mountBaseDir, string(vmi.UID))
}

func GetVolumeMountDirOnGuest(vmi *v1.VirtualMachineInstance) string {
	return filepath.Join(mountBaseDir, string(vmi.UID))
}

func GetVolumeMountDirOnHost(vmi *v1.VirtualMachineInstance) (*safepath.Path, error) {
	basepath := ""
	foundEntries := 0
	foundBasepath := ""
	for podUID := range vmi.Status.ActivePods {
		basepath = fmt.Sprintf("%s/%s/volumes/kubernetes.io~empty-dir/container-disks", podsBaseDir, string(podUID))
		exists, err := diskutils.FileExists(basepath)
		if err != nil {
			return nil, err
		} else if exists {
			foundEntries++
			foundBasepath = basepath
		}
	}

	if foundEntries == 1 {
		return safepath.JoinAndResolveWithRelativeRoot("/", foundBasepath)
	} else if foundEntries > 1 {
		// Don't mount until outdated pod environments are removed
		// otherwise we might stomp on a previous cleanup
		return nil, fmt.Errorf("Found multiple pods active for vmi %s/%s. Waiting on outdated pod directories to be removed", vmi.Namespace, vmi.Name)
	}
	return nil, os.ErrNotExist
}

func GetDiskTargetDirFromHostView(vmi *v1.VirtualMachineInstance) (*safepath.Path, error) {
	basepath, err := GetVolumeMountDirOnHost(vmi)
	if err != nil {
		return nil, err
	}
	return basepath, nil
}

func GetDiskTargetName(volumeIndex int) string {
	return fmt.Sprintf("disk_%d.img", volumeIndex)
}

func GetDiskTargetPathFromLauncherView(volumeIndex int) string {
	return filepath.Join(mountBaseDir, GetDiskTargetName(volumeIndex))
}

func GetKernelBootArtifactPathFromLauncherView(artifact string) string {
	artifactBase := filepath.Base(artifact)
	return filepath.Join(mountBaseDir, KernelBootName, artifactBase)
}

// SetLocalDirectoryOnly TODO: Refactor this package. This package is used by virt-controller
// to set proper paths on the virt-launcher template and by virt-launcher to create directories
// at the right location. The functions have side-effects and mix path setting and creation
// which makes it hard to differentiate the usage per component.
func SetLocalDirectoryOnly(dir string) {
	mountBaseDir = dir
}

func SetLocalDirectory(dir string) error {
	SetLocalDirectoryOnly(dir)
	return os.MkdirAll(dir, 0750)
}

func SetKubeletPodsDirectory(dir string) {
	podsBaseDir = dir
}

// used for testing - we don't want to MkdirAll on a production host mount
func setPodsDirectory(dir string) error {
	podsBaseDir = dir
	return os.MkdirAll(dir, 0750)
}

// GetDiskTargetPartFromLauncherView returns (path to disk image, image type, and error)
func GetDiskTargetPartFromLauncherView(volumeIndex int) (string, error) {

	path := GetDiskTargetPathFromLauncherView(volumeIndex)
	exists, err := diskutils.FileExists(path)
	if err != nil {
		return "", err
	} else if exists {
		return path, nil
	}

	return "", fmt.Errorf("no supported file disk found for volume with index %d", volumeIndex)
}

// NewSocketPathGetter get the socket pat of a containerDisk. For testing a baseDir
// can be provided which can for instance point to /tmp.
func NewSocketPathGetter(baseDir string) SocketPathGetter {
	return func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
		for podUID := range vmi.Status.ActivePods {
			basePath := getContainerDiskSocketBasePath(baseDir, string(podUID))
			socketPath := filepath.Join(basePath, fmt.Sprintf("disk_%d.sock", volumeIndex))
			exists, _ := diskutils.FileExists(socketPath)
			if exists {
				return socketPath, nil
			}
		}
		return "", fmt.Errorf("container disk socket path not found for vmi \"%s\"", vmi.Name)
	}
}

// NewKernelBootSocketPathGetter get the socket pat of the kernel-boot containerDisk. For testing a baseDir
// can be provided which can for instance point to /tmp.
func NewKernelBootSocketPathGetter(baseDir string) KernelBootSocketPathGetter {
	return func(vmi *v1.VirtualMachineInstance) (string, error) {
		for podUID := range vmi.Status.ActivePods {
			basePath := getContainerDiskSocketBasePath(baseDir, string(podUID))
			socketPath := filepath.Join(basePath, KernelBootName+".sock")
			exists, _ := diskutils.FileExists(socketPath)
			if exists {
				return socketPath, nil
			}
		}
		return "", fmt.Errorf("kernel boot socket path not found for vmi \"%s\"", vmi.Name)
	}
}

func GetImage(root *safepath.Path, imagePath string) (*safepath.Path, error) {
	if imagePath != "" {
		var err error
		resolvedPath, err := root.AppendAndResolveWithRelativeRoot(imagePath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine custom image path %s: %v", imagePath, err)
		}
		return resolvedPath, nil
	} else {
		fallbackPath, err := root.AppendAndResolveWithRelativeRoot(DiskSourceFallbackPath)
		if err != nil {
			return nil, fmt.Errorf("failed to determine default image path %v: %v", fallbackPath, err)
		}
		var files []os.DirEntry
		err = fallbackPath.ExecuteNoFollow(func(safePath string) (err error) {
			files, err = os.ReadDir(safePath)
			return err
		})
		if err != nil {
			return nil, fmt.Errorf("failed to check default image path %s: %v", fallbackPath, err)
		}
		if len(files) == 0 {
			return nil, fmt.Errorf("no file found in folder %s, no disk present", DiskSourceFallbackPath)
		} else if len(files) > 1 {
			return nil, fmt.Errorf("more than one file found in folder %s, only one disk is allowed", DiskSourceFallbackPath)
		}
		fileName := files[0].Name()
		resolvedPath, err := root.AppendAndResolveWithRelativeRoot(DiskSourceFallbackPath, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to check default image path %s: %v", imagePath, err)
		}
		return resolvedPath, nil
	}
}

func GenerateInitContainers(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) []kubev1.Container {
	return generateContainersHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, true)
}

func GenerateContainers(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) []kubev1.Container {
	return generateContainersHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, false)
}

func GenerateKernelBootContainer(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) *kubev1.Container {
	return generateKernelBootContainerHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, false)
}

func GenerateKernelBootInitContainer(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string) *kubev1.Container {
	return generateKernelBootContainerHelper(vmi, config, imageIDs, podVolumeName, binVolumeName, true)
}

func generateKernelBootContainerHelper(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string, isInit bool) *kubev1.Container {
	if !util.HasKernelBootContainerImage(vmi) {
		return nil
	}

	kernelBootContainer := vmi.Spec.Domain.Firmware.KernelBoot.Container

	kernelBootVolume := v1.Volume{
		Name: KernelBootVolumeName,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image:           kernelBootContainer.Image,
				ImagePullSecret: kernelBootContainer.ImagePullSecret,
				Path:            "/",
				ImagePullPolicy: kernelBootContainer.ImagePullPolicy,
			},
		},
	}

	const fakeVolumeIdx = 0 // volume index makes no difference for kernel-boot container
	return generateContainerFromVolume(vmi, config, imageIDs, podVolumeName, binVolumeName, isInit, true, &kernelBootVolume, fakeVolumeIdx)
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func generateContainersHelper(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName string, binVolumeName string, isInit bool) []kubev1.Container {
	var containers []kubev1.Container

	// Make VirtualMachineInstance Image Wrapper Containers
	for index, volume := range vmi.Spec.Volumes {
		if volume.Name == KernelBootVolumeName {
			continue
		}
		if container := generateContainerFromVolume(vmi, config, imageIDs, podVolumeName, binVolumeName, isInit, false, &volume, index); container != nil {
			containers = append(containers, *container)
		}
	}
	return containers
}

func generateContainerFromVolume(vmi *v1.VirtualMachineInstance, config *virtconfig.ClusterConfig, imageIDs map[string]string, podVolumeName, binVolumeName string, isInit, isKernelBoot bool, volume *v1.Volume, volumeIdx int) *kubev1.Container {
	if volume.ContainerDisk == nil {
		return nil
	}

	volumeMountDir := GetVolumeMountDirOnGuest(vmi)
	diskContainerName := toContainerName(volume.Name)
	diskContainerImage := volume.ContainerDisk.Image
	if img, exists := imageIDs[volume.Name]; exists {
		diskContainerImage = img
	}

	resources := kubev1.ResourceRequirements{}
	resources.Requests = make(kubev1.ResourceList)
	resources.Limits = make(kubev1.ResourceList)

	resources.Requests[kubev1.ResourceCPU] = resource.MustParse("1m")
	if cpuRequest := config.GetSupportContainerRequest(v1.ContainerDisk, kubev1.ResourceCPU); cpuRequest != nil {
		resources.Requests[kubev1.ResourceCPU] = *cpuRequest
	}
	resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1M")
	if memRequest := config.GetSupportContainerRequest(v1.ContainerDisk, kubev1.ResourceMemory); memRequest != nil {
		resources.Requests[kubev1.ResourceMemory] = *memRequest
	}
	resources.Requests[kubev1.ResourceEphemeralStorage] = resource.MustParse(ephemeralStorageOverheadSize)

	resources.Limits[kubev1.ResourceCPU] = resource.MustParse("10m")
	if cpuLimit := config.GetSupportContainerLimit(v1.ContainerDisk, kubev1.ResourceCPU); cpuLimit != nil {
		resources.Limits[kubev1.ResourceCPU] = *cpuLimit
	}
	resources.Limits[kubev1.ResourceMemory] = resource.MustParse("40M")
	if memLimit := config.GetSupportContainerLimit(v1.ContainerDisk, kubev1.ResourceMemory); memLimit != nil {
		resources.Limits[kubev1.ResourceMemory] = *memLimit
	}

	var mountedDiskName string
	if isKernelBoot {
		mountedDiskName = KernelBootName
	} else {
		mountedDiskName = "disk_" + strconv.Itoa(volumeIdx)
	}

	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		resources.Requests[kubev1.ResourceCPU] = resources.Limits[kubev1.ResourceCPU]
		resources.Requests[kubev1.ResourceMemory] = resources.Limits[kubev1.ResourceMemory]
	}
	var args []string
	var name string
	if isInit {
		name = diskContainerName + "-init"
		args = []string{"--no-op"}
	} else {
		name = diskContainerName
		copyPathArg := path.Join(volumeMountDir, mountedDiskName)
		args = []string{"--copy-path", copyPathArg}
	}

	noPrivilegeEscalation := false
	nonRoot := true
	var userId int64 = util.NonRootUID

	container := &kubev1.Container{
		Name:            name,
		Image:           diskContainerImage,
		ImagePullPolicy: volume.ContainerDisk.ImagePullPolicy,
		Command:         []string{"/usr/bin/container-disk"},
		Args:            args,
		VolumeMounts: []kubev1.VolumeMount{
			{
				Name:      podVolumeName,
				MountPath: volumeMountDir,
			},
			{
				Name:      binVolumeName,
				MountPath: "/usr/bin",
			},
		},
		Resources: resources,
		SecurityContext: &kubev1.SecurityContext{
			RunAsUser:                &userId,
			RunAsNonRoot:             &nonRoot,
			AllowPrivilegeEscalation: &noPrivilegeEscalation,
			Capabilities: &kubev1.Capabilities{
				Drop: []kubev1.Capability{"ALL"},
			},
		},
	}

	return container
}

func CreateEphemeralImages(
	vmi *v1.VirtualMachineInstance,
	diskCreator ephemeraldisk.EphemeralDiskCreatorInterface,
	disksInfo map[string]*DiskInfo,
) error {
	// The domain is setup to use the COW image instead of the base image. What we have
	// to do here is only create the image where the domain expects it (GetDiskTargetPartFromLauncherView)
	// for each disk that requires it.

	for i, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk != nil {
			info, _ := disksInfo[volume.Name]
			if info == nil {
				return fmt.Errorf("no disk info provided for volume %s", volume.Name)
			}
			if backingFile, err := GetDiskTargetPartFromLauncherView(i); err != nil {
				return err
			} else if err := diskCreator.CreateBackedImageForVolume(volume, backingFile, info.Format); err != nil {
				return err
			}
		}
	}

	return nil
}

func getContainerDiskSocketBasePath(baseDir, podUID string) string {
	return fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks", baseDir, podUID)
}

// ExtractImageIDsFromSourcePod takes the VMI and its source pod to determine the exact image used by containerdisks and boot container images,
// which is recorded in the status section of a started pod; if the status section does not contain this info the tag is used.
// It returns a map where the key is the vlume name and the value is the imageID
func ExtractImageIDsFromSourcePod(vmi *v1.VirtualMachineInstance, sourcePod *kubev1.Pod) (imageIDs map[string]string) {
	imageIDs = map[string]string{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk == nil {
			continue
		}
		imageIDs[volume.Name] = volume.ContainerDisk.Image
	}

	if util.HasKernelBootContainerImage(vmi) {
		imageIDs[KernelBootVolumeName] = vmi.Spec.Domain.Firmware.KernelBoot.Container.Image
	}

	for _, status := range sourcePod.Status.ContainerStatuses {
		if !isImageVolume(status.Name) {
			continue
		}
		key := toVolumeName(status.Name)
		image, exists := imageIDs[key]
		if !exists {
			continue
		}
		imageIDs[key] = toPullableImageReference(image, status.ImageID)
	}
	return
}

func toPullableImageReference(image string, imageID string) string {
	baseImage := image
	if strings.LastIndex(image, "@sha256:") != -1 {
		baseImage = strings.Split(image, "@sha256:")[0]
	} else if colonIndex := strings.LastIndex(image, ":"); colonIndex > strings.LastIndex(image, "/") {
		baseImage = image[:colonIndex]
	}

	digestMatches := digestRegex.FindStringSubmatch(imageID)
	if len(digestMatches) < 2 {
		// failed to identify image digest for container, will use the image tag
		// as virt-handler will anyway check the checksum of the root disk image
		return image
	}
	return fmt.Sprintf("%s@sha256:%s", baseImage, digestMatches[1])
}

func isImageVolume(containerName string) bool {
	return strings.HasPrefix(containerName, "volume")
}

func toContainerName(volumeName string) string {
	return fmt.Sprintf("volume%s", volumeName)
}

func toVolumeName(containerName string) string {
	return strings.TrimPrefix(containerName, "volume")
}
