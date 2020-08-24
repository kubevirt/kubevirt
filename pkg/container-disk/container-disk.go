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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"

	v1 "kubevirt.io/client-go/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
)

var containerDiskOwner = "qemu"

var podsBaseDir = util.KubeletPodsDir

var mountBaseDir = filepath.Join(util.VirtShareDir, "/container-disks")

type SocketPathGetter func(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error)

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
	for podUID, _ := range vmi.Status.ActivePods {
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

func GetDiskTargetPathFromHostView(vmi *v1.VirtualMachineInstance, volumeIndex int) (string, error) {
	basepath, found, err := GetVolumeMountDirOnHost(vmi)
	if err != nil {
		return "", err
	} else if !found {
		return "", fmt.Errorf("container disk volume for vmi not found")
	}

	return fmt.Sprintf("%s/disk_%d.img", basepath, volumeIndex), nil
}

func GetDiskTargetPathFromLauncherView(volumeIndex int) string {
	return filepath.Join(mountBaseDir, fmt.Sprintf("disk_%d.img", volumeIndex))
}

func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
	return os.MkdirAll(dir, 0755)
}

func SetKubeletPodsDirectory(dir string) {
	podsBaseDir = dir
}

// used for testing - we don't want to MkdirAll on a production host mount
func setPodsDirectory(dir string) error {
	podsBaseDir = dir
	return os.MkdirAll(dir, 0755)
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
		for podUID, _ := range vmi.Status.ActivePods {
			basepath := fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~empty-dir/container-disks", baseDir, string(podUID))
			socketPath := filepath.Join(basepath, fmt.Sprintf("disk_%d.sock", volumeIndex))
			exists, _ := diskutils.FileExists(socketPath)
			if exists {
				return socketPath, nil
			}
		}
		return "", fmt.Errorf("container disk socket path not found for vmi")
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
		files, err := ioutil.ReadDir(fallbackPath)
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

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func GenerateContainers(vmi *v1.VirtualMachineInstance, podVolumeName string, binVolumeName string) []kubev1.Container {
	var containers []kubev1.Container

	// Make VirtualMachineInstance Image Wrapper Containers
	for index, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {

			volumeMountDir := GetVolumeMountDirOnGuest(vmi)
			diskContainerName := fmt.Sprintf("volume%s", volume.Name)
			diskContainerImage := volume.ContainerDisk.Image
			resources := kubev1.ResourceRequirements{}
			if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
				resources.Limits = make(kubev1.ResourceList)
				resources.Limits[kubev1.ResourceCPU] = resource.MustParse("10m")
				resources.Limits[kubev1.ResourceMemory] = resource.MustParse("40M")
				resources.Requests = make(kubev1.ResourceList)
				resources.Requests[kubev1.ResourceCPU] = resource.MustParse("10m")
				resources.Requests[kubev1.ResourceMemory] = resource.MustParse("40M")
			} else {
				resources.Limits = make(kubev1.ResourceList)
				resources.Limits[kubev1.ResourceCPU] = resource.MustParse("100m")
				resources.Limits[kubev1.ResourceMemory] = resource.MustParse("40M")
				resources.Requests = make(kubev1.ResourceList)
				resources.Requests[kubev1.ResourceCPU] = resource.MustParse("10m")
				resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1M")
			}
			container := kubev1.Container{
				Name:            diskContainerName,
				Image:           diskContainerImage,
				ImagePullPolicy: volume.ContainerDisk.ImagePullPolicy,
				Command:         []string{"/usr/bin/container-disk"},
				Args:            []string{"--copy-path", volumeMountDir + "/disk_" + strconv.Itoa(index)},
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
			}

			containers = append(containers, container)
		}
	}
	return containers
}

func CreateEphemeralImages(vmi *v1.VirtualMachineInstance) error {
	// The domain is setup to use the COW image instead of the base image. What we have
	// to do here is only create the image where the domain expects it (GetDiskTargetPartFromLauncherView)
	// for each disk that requires it.

	for i, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.ContainerDisk != nil {
			if backingFile, err := GetDiskTargetPartFromLauncherView(i); err != nil {
				return err
			} else if err := ephemeraldisk.CreateBackedImageForVolume(volume, backingFile); err != nil {
				return err
			}
		}
	}

	return nil
}
