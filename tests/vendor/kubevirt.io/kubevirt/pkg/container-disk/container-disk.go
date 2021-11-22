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

	"kubevirt.io/client-go/log"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"

	v1 "kubevirt.io/client-go/apis/core/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

var containerDiskOwner = "qemu"

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

func GetVolumeMountDirOnHost(vmi *v1.VirtualMachineInstance) (string, bool, error) {
	basepath := ""
	foundEntries := 0
	foundBasepath := ""
	for podUID := range vmi.Status.ActivePods {
		basepath = fmt.Sprintf("%s/%s/volumes/kubernetes.io~empty-dir/container-disks", podsBaseDir, string(podUID))
		exists, err := diskutils.FileExists(basepath)
		if err != nil {
			return "", false, err
		} else if exists {
			foundEntries++
			foundBasepath = basepath
		}
	}

	if foundEntries == 1 {
		return foundBasepath, true, nil
	} else if foundEntries > 1 {
		// Don't mount until outdated pod environments are removed
		// otherwise we might stomp on a previous cleanup
		return "", false, fmt.Errorf("Found multiple pods active for vmi %s/%s. Waiting on outdated pod directories to be removed", vmi.Namespace, vmi.Name)
	}
	return "", false, nil
}

func GetDiskTargetDirFromHostView(vmi *v1.VirtualMachineInstance) (string, error) {
	basepath, found, err := GetVolumeMountDirOnHost(vmi)
	if err != nil {
		return "", err
	} else if !found {
		return "", fmt.Errorf("container disk volume for vmi not found")
	}

	return basepath, nil
}

func GetDiskTargetPathFromHostView(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
	basepath, err := GetDiskTargetDirFromHostView(vmi)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/disk_%d.img", basepath, volumeIndex), nil
}

func GetDiskTargetPathFromLauncherView(volumeIndex int) string {
	return filepath.Join(mountBaseDir, fmt.Sprintf("disk_%d.img", volumeIndex))
}

func GetKernelBootArtifactPathFromLauncherView(artifact string) string {
	artifactBase := filepath.Base(artifact)
	return filepath.Join(mountBaseDir, KernelBootName, artifactBase)
}

func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
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

// The unit test suite uses this function
func setLocalDataOwner(user string) {
	containerDiskOwner = user
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

func GetImage(root string, imagePath string) (string, error) {
	fallbackPath := filepath.Join(root, DiskSourceFallbackPath)
	if imagePath != "" {
		imagePath = filepath.Join(root, imagePath)
		if _, err := os.Stat(imagePath); err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("No image on path %s", imagePath)
			} else {
				return "", fmt.Errorf("Failed to check if an image exists at %s", imagePath)
			}
		}
	} else {
		files, err := os.ReadDir(fallbackPath)
		if err != nil {
			return "", fmt.Errorf("Failed to check %s for disks: %v", fallbackPath, err)
		}
		if len(files) > 1 {
			return "", fmt.Errorf("More than one file found in folder %s, only one disk is allowed", DiskSourceFallbackPath)
		}
		imagePath = filepath.Join(fallbackPath, files[0].Name())
	}
	return imagePath, nil
}

func GenerateInitContainers(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName string, binVolumeName string) []kubev1.Container {
	return generateContainersHelper(vmi, imageIDs, podVolumeName, binVolumeName, true)
}

func GenerateContainers(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName string, binVolumeName string) []kubev1.Container {
	return generateContainersHelper(vmi, imageIDs, podVolumeName, binVolumeName, false)
}

func GenerateKernelBootContainer(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName string, binVolumeName string) *kubev1.Container {
	return generateKernelBootContainerHelper(vmi, imageIDs, podVolumeName, binVolumeName, false)
}

func GenerateKernelBootInitContainer(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName string, binVolumeName string) *kubev1.Container {
	return generateKernelBootContainerHelper(vmi, imageIDs, podVolumeName, binVolumeName, true)
}

func generateKernelBootContainerHelper(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName string, binVolumeName string, isInit bool) *kubev1.Container {
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
	return generateContainerFromVolume(vmi, imageIDs, podVolumeName, binVolumeName, isInit, true, &kernelBootVolume, fakeVolumeIdx)
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func generateContainersHelper(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName string, binVolumeName string, isInit bool) []kubev1.Container {
	var containers []kubev1.Container

	// Make VirtualMachineInstance Image Wrapper Containers
	for index, volume := range vmi.Spec.Volumes {
		if volume.Name == KernelBootVolumeName {
			continue
		}
		if container := generateContainerFromVolume(vmi, imageIDs, podVolumeName, binVolumeName, isInit, false, &volume, index); container != nil {
			containers = append(containers, *container)
		}
	}
	return containers
}

func generateContainerFromVolume(vmi *v1.VirtualMachineInstance, imageIDs map[string]string, podVolumeName, binVolumeName string, isInit, isKernelBoot bool, volume *v1.Volume, volumeIdx int) *kubev1.Container {
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
	resources.Limits = make(kubev1.ResourceList)
	resources.Requests = make(kubev1.ResourceList)
	resources.Limits[kubev1.ResourceMemory] = resource.MustParse("40M")
	resources.Requests[kubev1.ResourceCPU] = resource.MustParse("10m")
	resources.Requests[kubev1.ResourceEphemeralStorage] = resource.MustParse(ephemeralStorageOverheadSize)

	var mountedDiskName string
	if isKernelBoot {
		mountedDiskName = KernelBootName
	} else {
		mountedDiskName = "disk_" + strconv.Itoa(volumeIdx)
	}

	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		resources.Limits[kubev1.ResourceCPU] = resource.MustParse("10m")
		resources.Requests[kubev1.ResourceMemory] = resource.MustParse("40M")
	} else {
		resources.Limits[kubev1.ResourceCPU] = resource.MustParse("100m")
		resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1M")
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

		log.Log.Object(vmi).Infof("arguments for container-disk \"%s\": --copy-path %s", name, copyPathArg)
	}

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
			RunAsUser:    &userId,
			RunAsNonRoot: &nonRoot,
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
// which is recorded in the status section of a started pod.
// It returns a map where the key is the vlume name and the value is the imageID
func ExtractImageIDsFromSourcePod(vmi *v1.VirtualMachineInstance, sourcePod *kubev1.Pod) (imageIDs map[string]string, err error) {
	imageIDs = map[string]string{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk == nil {
			continue
		}
		imageIDs[volume.Name] = ""
	}

	if util.HasKernelBootContainerImage(vmi) {
		imageIDs[KernelBootVolumeName] = ""
	}

	for _, status := range sourcePod.Status.ContainerStatuses {
		if !isImageVolume(status.Name) {
			continue
		}
		key := toVolumeName(status.Name)
		if _, exists := imageIDs[key]; !exists {
			continue
		}
		imageID, err := toImageWithDigest(status.Image, status.ImageID)
		if err != nil {
			return nil, err
		}
		imageIDs[key] = imageID
	}
	return
}

func toImageWithDigest(image string, imageID string) (string, error) {
	baseImage := image
	if strings.LastIndex(image, "@sha256:") != -1 {
		baseImage = strings.Split(image, "@sha256:")[0]
	} else if colonIndex := strings.LastIndex(image, ":"); colonIndex > strings.LastIndex(image, "/") {
		baseImage = image[:colonIndex]
	}

	digestMatches := digestRegex.FindStringSubmatch(imageID)
	if len(digestMatches) < 2 {
		return "", fmt.Errorf("failed to identify image digest for container %q with id %q", image, imageID)
	}
	return fmt.Sprintf("%s@sha256:%s", baseImage, digestMatches[1]), nil
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
