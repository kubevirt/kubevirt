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

package registrydisk

import (
	"errors"
	"fmt"
	"os"

	"github.com/jeevatkm/go-model"

	kubev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/precond"
)

const registryDiskV1Alpha = "ContainerRegistryDisk:v1alpha"
const defaultIqn = "iqn.2017-01.io.kubevirt:wrapper/1"
const defaultPort = 3261
const defaultPortStr = "3261"
const filePrefix = "disk-image"
const defaultHost = "127.0.0.1"

var registryDiskOwner = "qemu"

var mountBaseDir = "/var/run/libvirt/kubevirt-disk-dir"

func generateVMBaseDir(vm *v1.VirtualMachine) string {
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	return fmt.Sprintf("%s/%s/%s", mountBaseDir, namespace, domain)
}
func generateVolumeMountDir(vm *v1.VirtualMachine, diskCount int) string {
	baseDir := generateVMBaseDir(vm)
	return fmt.Sprintf("%s/disk%d", baseDir, diskCount)
}

func k8sSecretName(vm *v1.VirtualMachine) string {
	return fmt.Sprintf("registrydisk-iscsi-%s-%s", vm.GetObjectMeta().GetNamespace(), vm.GetObjectMeta().GetName())
}

func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
	return nil
}

// The unit test suite uses this function
func SetLocalDataOwner(user string) {
	registryDiskOwner = user
}

func getFilePath(basePath string) (string, string, error) {
	rawPath := basePath + "/" + filePrefix + ".raw"
	qcow2Path := basePath + "/" + filePrefix + ".qcow2"

	exists, err := diskutils.FileExists(rawPath)
	if err != nil {
		return "", "", err
	} else if exists {
		return rawPath, "raw", nil
	}

	exists, err = diskutils.FileExists(qcow2Path)
	if err != nil {
		return "", "", err
	} else if exists {
		return qcow2Path, "qcow2", nil
	}

	return "", "", errors.New(fmt.Sprintf("no supported file disk found in directory %s", basePath))
}

func CleanupEphemeralDisks(vm *v1.VirtualMachine) error {
	volumeMountDir := generateVMBaseDir(vm)
	err := os.RemoveAll(volumeMountDir)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// The virt-handler converts registry disks to their corresponding iscsi network
// disks when the VM spec is being defined as a domain with libvirt.
// The ports and host of the iscsi disks are already provided here by the controller.
func MapRegistryDisks(vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	vmCopy := &v1.VirtualMachine{}
	model.Copy(vmCopy, vm)

	for diskCount, disk := range vmCopy.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {
			volumeMountDir := generateVolumeMountDir(vm, diskCount)

			diskPath, diskType, err := getFilePath(volumeMountDir)
			if err != nil {
				return vm, err
			}

			// Rename file to release management of it from container process.
			oldDiskPath := diskPath
			diskPath = oldDiskPath + ".virt"
			err = os.Rename(oldDiskPath, diskPath)
			if err != nil {
				return vm, err
			}

			err = diskutils.SetFileOwnership(registryDiskOwner, diskPath)
			if err != nil {
				return vm, err
			}

			newDisk := v1.Disk{}
			newDisk.Type = "file"
			newDisk.Device = "disk"
			newDisk.Driver = &v1.DiskDriver{
				Type: diskType,
				Name: "qemu",
			}
			newDisk.Source.File = diskPath
			newDisk.Target = disk.Target
			vmCopy.Spec.Domain.Devices.Disks[diskCount] = newDisk
		}
	}

	return vmCopy, nil
}

// The controller uses this function to generate the container
// specs for hosting the container registry disks.
func GenerateContainers(vm *v1.VirtualMachine) ([]kubev1.Container, []kubev1.Volume, error) {
	var containers []kubev1.Container
	var volumes []kubev1.Volume

	initialDelaySeconds := 2
	timeoutSeconds := 5
	periodSeconds := 5
	successThreshold := 2
	failureThreshold := 5

	// Make VM Image Wrapper Containers
	for diskCount, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.Type == registryDiskV1Alpha {

			volumeMountDir := generateVolumeMountDir(vm, diskCount)
			volumeName := fmt.Sprintf("disk%d-volume", diskCount)
			diskContainerName := fmt.Sprintf("disk%d", diskCount)
			// container image is disk.Source.Name
			diskContainerImage := disk.Source.Name

			volumes = append(volumes, kubev1.Volume{
				Name: volumeName,
				VolumeSource: kubev1.VolumeSource{
					HostPath: &kubev1.HostPathVolumeSource{
						Path: volumeMountDir,
					},
				},
			})
			containers = append(containers, kubev1.Container{
				Name:            diskContainerName,
				Image:           diskContainerImage,
				ImagePullPolicy: kubev1.PullIfNotPresent,
				Command:         []string{"/entry-point.sh"},
				Env: []kubev1.EnvVar{
					kubev1.EnvVar{
						Name:  "COPY_PATH",
						Value: volumeMountDir + "/" + filePrefix,
					},
				},
				VolumeMounts: []kubev1.VolumeMount{
					{
						Name:      volumeName,
						MountPath: volumeMountDir,
					},
				},
				// The readiness probes ensure the disk coversion and copy finished
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
			})
		}
	}
	return containers, volumes, nil
}
