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

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/precond"
)

const filePrefix = "disk-image"

var containerDiskOwner = "qemu"

var mountBaseDir = "/var/run/kubevirt-ephemeral-disks/container-disk-data"

func generateVMIBaseDir(vmi *v1.VirtualMachineInstance) string {
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	return fmt.Sprintf("%s/%s/%s", mountBaseDir, namespace, domain)
}
func generateVolumeMountDir(vmi *v1.VirtualMachineInstance, volumeName string) string {
	baseDir := generateVMIBaseDir(vmi)
	return fmt.Sprintf("%s/disk_%s", baseDir, volumeName)
}

func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
	return os.MkdirAll(dir, 0755)
}

// The unit test suite uses this function
func SetLocalDataOwner(user string) {
	containerDiskOwner = user
}

// GetFilePath returns  (path to disk image, image type, and error)
func GetFilePath(vmi *v1.VirtualMachineInstance, volumeName string) (string, string, error) {

	volumeMountDir := generateVolumeMountDir(vmi, volumeName)
	suffixes := map[string]string{".raw": "raw", ".qcow2": "qcow2", ".raw.virt": "raw", ".qcow2.virt": "qcow2"}

	for k, v := range suffixes {
		path := volumeMountDir + "/" + filePrefix + k
		exists, err := diskutils.FileExists(path)
		if err != nil {
			return "", "", err
		} else if exists {
			return path, v, nil
		}
	}

	return "", "", fmt.Errorf("no supported file disk found in directory %s", volumeMountDir)
}

func SetFilePermissions(vmi *v1.VirtualMachineInstance) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			diskPath, _, err := GetFilePath(vmi, volume.Name)
			if err != nil {
				return err
			}

			err = diskutils.SetFileOwnership(containerDiskOwner, diskPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func GenerateContainers(vmi *v1.VirtualMachineInstance, podVolumeName string, podVolumeMountDir string) []kubev1.Container {
	var containers []kubev1.Container

	initialDelaySeconds := 2
	timeoutSeconds := 5
	periodSeconds := 5
	successThreshold := 2
	failureThreshold := 5

	// Make VirtualMachineInstance Image Wrapper Containers
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {

			volumeMountDir := generateVolumeMountDir(vmi, volume.Name)
			diskContainerName := fmt.Sprintf("volume%s", volume.Name)
			diskContainerImage := volume.ContainerDisk.Image
			resources := kubev1.ResourceRequirements{}
			if vmi.IsCPUDedicated() {
				resources.Limits = make(kubev1.ResourceList)
				// TODO(vladikr): adjust the correct cpu/mem values - this is mainly needed to allow QemuImg to run correctly
				resources.Limits[kubev1.ResourceCPU] = resource.MustParse("400m")
				// k8s minimum memory reservation is linuxMinMemory = 4194304
				resources.Limits[kubev1.ResourceMemory] = resource.MustParse("128M")
			}
			container := kubev1.Container{
				Name:            diskContainerName,
				Image:           diskContainerImage,
				ImagePullPolicy: kubev1.PullIfNotPresent,
				Command:         []string{"/entry-point.sh"},
				Env: []kubev1.EnvVar{
					{
						Name:  "COPY_PATH",
						Value: volumeMountDir + "/" + filePrefix,
					},
				},
				VolumeMounts: []kubev1.VolumeMount{
					{
						Name:      podVolumeName,
						MountPath: podVolumeMountDir,
					},
				},
				Resources: resources,

				// The readiness probes ensure the volume coversion and copy finished
				// before the container is marked as "Ready: True"
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
			}

			if volume.ContainerDisk.Path != "" {
				container.Env = append(container.Env, kubev1.EnvVar{
					Name:  "IMAGE_PATH",
					Value: volume.ContainerDisk.Path,
				})
			}

			containers = append(containers, container)
		}
	}
	return containers
}
