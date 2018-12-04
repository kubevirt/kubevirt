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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package ephemeraldisk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

var mountBaseDir = "/var/run/libvirt/kubevirt-ephemeral-disk"
var pvcBaseDir = "/var/run/kubevirt-private/vmi-disks"
var ephemeralImageDiskOwner = "qemu"

func generateBaseDir() string {
	return fmt.Sprintf("%s", mountBaseDir)
}
func generateVolumeMountDir(volumeName string) string {
	baseDir := generateBaseDir()
	return filepath.Join(baseDir, volumeName)
}

func getBackingFilePath(volumeName string) string {
	return filepath.Join(pvcBaseDir, volumeName, "disk.img")
}

func SetLocalDirectory(dir string) error {
	mountBaseDir = dir
	return os.MkdirAll(dir, 0755)
}

// Used by tests.
func setBackingDirectory(dir string) error {
	pvcBaseDir = dir
	return os.MkdirAll(dir, 0755)
}

// Used by tests.
func SetLocalDataOwner(user string) {
	ephemeralImageDiskOwner = user
}

func createVolumeDirectory(volumeName string) error {
	dir := generateVolumeMountDir(volumeName)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	return nil
}

func GetFilePath(volumeName string) string {
	volumeMountDir := generateVolumeMountDir(volumeName)
	return filepath.Join(volumeMountDir, "disk.qcow2")
}

func CreateEphemeralImages(vmi *v1.VirtualMachineInstance) error {
	// The domain is setup to use the COW image instead of the base image. What we have
	// to do here is only create the image where the domain expects it (GetFilePath)
	// for each disk that requires it.
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.Ephemeral != nil {
			err := createVolumeDirectory(volume.Name)
			if err != nil {
				return err
			}

			imagePath := GetFilePath(volume.Name)

			var args []string

			args = append(args, "create")
			args = append(args, "-f")
			args = append(args, "qcow2")
			args = append(args, "-b")
			args = append(args, getBackingFilePath(volume.Name))
			args = append(args, imagePath)

			cmd := exec.Command("qemu-img", args...)
			err = cmd.Run()

			// Cleanup of previous images isn't really necessary as they're all on EmptyDir.
			if err != nil {
				return err
			}

			// We need to ensure that the permissions are setup correctly.
			err = diskutils.SetFileOwnership(ephemeralImageDiskOwner, imagePath)
			if err != nil {
				return err
			}
		}

	}

	return nil
}
