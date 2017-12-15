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

	kubev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"

	"strings"

	"kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/precond"
)

const filePrefix = "disk-image"

var registryDiskOwner = "qemu"

var mountBaseDir = "/var/run/libvirt/kubevirt-disk-dir"

func generateVMBaseDir(vm *v1.VirtualMachine) string {
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	return fmt.Sprintf("%s/%s/%s", mountBaseDir, namespace, domain)
}
func generateVolumeMountDir(vm *v1.VirtualMachine, volumeName string) string {
	baseDir := generateVMBaseDir(vm)
	return fmt.Sprintf("%s/disk_%s", baseDir, volumeName)
}

func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
	return nil
}

// The unit test suite uses this function
func SetLocalDataOwner(user string) {
	registryDiskOwner = user
}

func GetFilePath(vm *v1.VirtualMachine, volumeName string) (string, string, error) {

	volumeMountDir := generateVolumeMountDir(vm, volumeName)
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

	return "", "", errors.New(fmt.Sprintf("no supported file disk found in directory %s", volumeMountDir))
}

func CleanupOrphanedEphemeralDisks(indexer cache.Store) error {
	vms, err := diskutils.ListVmWithEphemeralDisk(mountBaseDir)
	if err != nil {
		return err
	}

	for _, vm := range vms {
		cleanup := false
		key, err := cache.MetaNamespaceKeyFunc(vm)
		if err != nil {
			return err
		}
		obj, exists, _ := indexer.GetByKey(key)
		if exists == false {
			cleanup = true
		} else {
			vm := obj.(*v1.VirtualMachine)
			if vm.IsFinal() {
				cleanup = true
			}
		}

		if cleanup {
			err := CleanupEphemeralDisks(vm)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CleanupEphemeralDisks(vm *v1.VirtualMachine) error {
	volumeMountDir := generateVMBaseDir(vm)
	err := os.RemoveAll(volumeMountDir)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// The virt-handler renames all registry disks in order to indicate to virt-launcher
// that it took the ownership of the image
func TakeOverRegistryDisks(vm *v1.VirtualMachine) error {

	for _, volume := range vm.Spec.Volumes {
		if volume.RegistryDisk != nil {
			diskPath, _, err := GetFilePath(vm, volume.Name)
			if err != nil {
				return err
			}

			// Rename file to release management of it from container process.
			if !strings.HasSuffix(diskPath, ".virt") {
				oldDiskPath := diskPath
				diskPath = oldDiskPath + ".virt"
				err = os.Rename(oldDiskPath, diskPath)
				if err != nil {
					return err
				}
			}

			err = diskutils.SetFileOwnership(registryDiskOwner, diskPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
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
	for _, volume := range vm.Spec.Volumes {
		if volume.RegistryDisk != nil {

			volumeMountDir := generateVolumeMountDir(vm, volume.Name)
			volumeName := fmt.Sprintf("volume%s-volume", volume.Name)
			diskContainerName := fmt.Sprintf("volume%s", volume.Name)
			// container image is volume.Source.Name
			diskContainerImage := volume.Name

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
			})
		}
	}
	return containers, volumes, nil
}
