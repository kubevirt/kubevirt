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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	ephemeralDiskPVCBaseDir         = "/var/run/kubevirt-private/vmi-disks"
	ephemeralDiskBlockDeviceBaseDir = "/dev"
	ephemeralDiskFormat             = "raw"
)

type EphemeralDiskCreatorInterface interface {
	CreateBackedImageForVolume(volume v1.Volume, backingFile string, backingFormat string) error
	CreateEphemeralImages(vmi *v1.VirtualMachineInstance, domain *api.Domain) error
	GetFilePath(volumeName string) string
	Init() error
}

type ephemeralDiskCreator struct {
	mountBaseDir    string
	pvcBaseDir      string
	blockDevBaseDir string
	discCreateFunc  func(backingFile string, backingFormat string, imagePath string) ([]byte, error)
}

func NewEphemeralDiskCreator(mountBaseDir string) *ephemeralDiskCreator {
	return &ephemeralDiskCreator{
		mountBaseDir:    mountBaseDir,
		pvcBaseDir:      ephemeralDiskPVCBaseDir,
		blockDevBaseDir: ephemeralDiskBlockDeviceBaseDir,
		discCreateFunc:  createBackingDisk,
	}
}

func (c *ephemeralDiskCreator) Init() error {
	return os.MkdirAll(c.mountBaseDir, 0755)
}

func (c *ephemeralDiskCreator) generateVolumeMountDir(volumeName string) string {
	return filepath.Join(c.mountBaseDir, volumeName)
}

func (c *ephemeralDiskCreator) getBackingFilePath(volumeName string, isBlockVolume bool) string {
	if isBlockVolume {
		return filepath.Join(c.blockDevBaseDir, volumeName)
	}
	return filepath.Join(c.pvcBaseDir, volumeName, "disk.img")
}

func (c *ephemeralDiskCreator) createVolumeDirectory(volumeName string) error {
	dir := c.generateVolumeMountDir(volumeName)

	err := util.MkdirAllWithNosec(dir)
	if err != nil {
		return err
	}

	return nil
}

func (c *ephemeralDiskCreator) GetFilePath(volumeName string) string {
	volumeMountDir := c.generateVolumeMountDir(volumeName)
	return filepath.Join(volumeMountDir, "disk.qcow2")
}

func (c *ephemeralDiskCreator) CreateBackedImageForVolume(volume v1.Volume, backingFile string, backingFormat string) error {
	err := c.createVolumeDirectory(volume.Name)
	if err != nil {
		return err
	}

	imagePath := c.GetFilePath(volume.Name)

	if _, err := os.Stat(imagePath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	output, err := c.discCreateFunc(backingFile, backingFormat, imagePath)

	// Cleanup of previous images isn't really necessary as they're all on EmptyDir.
	if err != nil {
		return fmt.Errorf("qemu-img failed with output '%s': %v", string(output), err)
	}

	// #nosec G302: Poor file permissions used with chmod. Safe permission setting for files shared between virt-launcher and qemu.
	if err = os.Chmod(imagePath, 0640); err != nil {
		return fmt.Errorf("failed to change permissions on %s", imagePath)
	}

	// We need to ensure that the permissions are setup correctly.
	err = diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(imagePath)
	return err
}

func (c *ephemeralDiskCreator) CreateEphemeralImages(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	// The domain is setup to use the COW image instead of the base image. What we have
	// to do here is only create the image where the domain expects it (GetFilePath)
	// for each disk that requires it.
	isBlockVolumes := diskutils.GetEphemeralBackingSourceBlockDevices(domain)
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.Ephemeral != nil {
			if err := c.CreateBackedImageForVolume(volume, c.getBackingFilePath(volume.Name, isBlockVolumes[volume.Name]), ephemeralDiskFormat); err != nil {
				return err
			}
		}
	}

	return nil
}

func createBackingDisk(backingFile string, backingFormat string, imagePath string) ([]byte, error) {
	// #nosec No risk for attacket injection. Parameters are predefined strings
	cmd := exec.Command("qemu-img",
		"create",
		"-f",
		"qcow2",
		"-b",
		backingFile,
		"-F",
		backingFormat,
		imagePath,
	)
	return cmd.CombinedOutput()
}
