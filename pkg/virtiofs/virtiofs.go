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

package virtiofs

import (
	"fmt"
	//"os"
	//"path"
	"path/filepath"
	//"regexp"
	//"strconv"
	//"strings"

	//"kubevirt.io/client-go/log"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/util"
)

const (
	varRun = "/var/run"
)

func GenerateContainers(vmi *v1.VirtualMachineInstance, podVolumeName string, virtLauncherImage string) []k8sv1.Container {
	return generateContainersHelper(vmi, podVolumeName, virtLauncherImage)
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func generateContainersHelper(vmi *v1.VirtualMachineInstance, podVolumeName string, virtLauncherImage string) []k8sv1.Container {
	var containers []k8sv1.Container
	passthoughFSVolumes := make(map[string]struct{})
	for i := range vmi.Spec.Domain.Devices.Filesystems {
		passthoughFSVolumes[vmi.Spec.Domain.Devices.Filesystems[i].Name] = struct{}{}
	}

	// Make VirtualMachineInstance Image Wrapper Containers
	for index, volume := range vmi.Spec.Volumes {
		if _, isPassthoughFSVolume := passthoughFSVolumes[volume.Name]; isPassthoughFSVolume {
			//log.Log.V(4).Infof("this volume %s is mapped as a filesystem passthrough, will not be replaced by HostDisk", volumeName)
			if container := generateContainerFromVolume(vmi, podVolumeName, virtLauncherImage, &volume, index); container != nil {
				containers = append(containers, *container)
			}
		}
	}
	return containers
}

var mountBaseDir string

func GetVolumeMountDirOnGuest(vmi *v1.VirtualMachineInstance) string {
	return filepath.Join(mountBaseDir, string(vmi.UID))
}

func getVirtiofsCapabilities() []k8sv1.Capability {
	return []k8sv1.Capability{
		"SYS_CHROOT",
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

func generateContainerFromVolume(vmi *v1.VirtualMachineInstance, podVolumeName, virtLauncherImage string, volume *v1.Volume, volumeIdx int) *k8sv1.Container {

	mountBaseDir = filepath.Join(util.VirtShareDir, podVolumeName)
	//volumeMountDir := GetVolumeMountDirOnGuest(vmi)
	diskContainerName := fmt.Sprintf("sharedfs-%s", volume.Name)

	resources := k8sv1.ResourceRequirements{}
	resources.Limits = make(k8sv1.ResourceList)
	resources.Requests = make(k8sv1.ResourceList)
	resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("40M")
	resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("10m")
	//resources.Requests[k8sv1.ResourceEphemeralStorage] = resource.MustParse(ephemeralStorageOverheadSize)

	socketName := fmt.Sprintf("%s.sock", volume.Name)
	socketPath := filepath.Join(mountBaseDir, socketName)

	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("10m")
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("40M")
	} else {
		resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("100m")
		resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1M")
	}
	var args []string
	var name string
	name = diskContainerName
	socketPathArg := fmt.Sprintf("--socket-path=%s", socketPath)
	volumeNameArg := fmt.Sprintf("--volume-name=%s", volume.Name)
	args = []string{socketPathArg, volumeNameArg}
	//"-o", optionsArg, "-o", "sandbox=chroot", "-o", "xattr", "-o", "xattrmap=:map::user.virtiofsd.:"}

	var userId int64 = util.RootUser

	nonRoot := false
	/*	nonRoot := util.IsNonRootVMI(vmi)
		if nonRoot {
			userId = util.NonRootUID
		}*/

	capabilities := getVirtiofsCapabilities()
	container := &k8sv1.Container{
		Name:            name,
		Image:           virtLauncherImage,
		ImagePullPolicy: k8sv1.PullIfNotPresent,
		Command:         []string{"/usr/bin/virtiofsd-monitor"},
		Args:            args,
		VolumeMounts: []k8sv1.VolumeMount{
			{
				Name:      podVolumeName,
				MountPath: mountBaseDir,
			},
			{
				Name:      "virtiofs-runtime",
				MountPath: "/var/run/virtiofsd",
			},
			{
				Name:      volume.Name,
				MountPath: fmt.Sprintf("/%s", volume.Name),
			},
		},
		Resources: resources,
		SecurityContext: &k8sv1.SecurityContext{
			RunAsUser:    &userId,
			RunAsNonRoot: &nonRoot,
			Capabilities: &k8sv1.Capabilities{
				Add: capabilities,
			},
		},
	}
	if nonRoot {
		//container.SecurityContext.FSGroup = &userId
		container.SecurityContext.RunAsGroup = &userId
		container.SecurityContext.RunAsNonRoot = &nonRoot
		container.Env = append(container.Env,
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

	return container
}
