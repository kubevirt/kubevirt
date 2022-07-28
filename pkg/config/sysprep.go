/*
 * This file is part of the kubevirt project
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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

// Assuming windows does not care what's the exact label.
var sysprepVolumeLabel = "unattendCD"

// GetSysprepSourcePath returns a path to the Sysprep volume mounted on a pod
func GetSysprepSourcePath(volumeName string) string {
	return filepath.Join(SysprepSourceDir, volumeName)
}

// GetSysprepDiskPath returns a path to a ConfigMap iso image created based on a volume name
func GetSysprepDiskPath(volumeName string) string {
	return filepath.Join(SysprepDisksDir, volumeName+".iso")
}

func sysprepVolumeHasContents(sysprepVolume *v1.SysprepSource) bool {
	return sysprepVolume.ConfigMap != nil || sysprepVolume.Secret != nil
}

// Explained here: https://docs.microsoft.com/en-us/windows-hardware/manufacture/desktop/windows-setup-automation-overview
const autounattendFilename = "autounattend.xml"
const unattendFilename = "unattend.xml"

func validateUnattendPresence(dirPath string) error {
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("Error validating that %s or %s have been provided: %w", autounattendFilename, unattendFilename, err)
	}
	for _, file := range files {
		if f := strings.ToLower(file.Name()); f == autounattendFilename || f == unattendFilename {
			return nil
		}
	}
	return fmt.Errorf("Sysprep drive should contain %s or %s but neither were found.", autounattendFilename, unattendFilename)
}

// CreateSysprepDisks creates Sysprep iso disks which are attached to vmis from either ConfigMap or Secret as a source
func CreateSysprepDisks(vmi *v1.VirtualMachineInstance, emptyIso bool) error {
	for _, volume := range vmi.Spec.Volumes {
		if !shouldCreateSysprepDisk(volume.Sysprep) {
			continue
		}
		vmiIsoSize, err := findIsoSize(vmi, &volume, emptyIso)
		if err != nil {
			return err
		}
		if err := createSysprepDisk(volume.Name, vmiIsoSize); err != nil {
			return err
		}
	}
	return nil
}

func shouldCreateSysprepDisk(volumeSysprep *v1.SysprepSource) bool {
	return volumeSysprep != nil && sysprepVolumeHasContents(volumeSysprep)
}

func createSysprepDisk(volumeName string, size int64) error {
	sysprepSourcePath := GetSysprepSourcePath(volumeName)
	if err := validateUnattendPresence(sysprepSourcePath); err != nil {
		return err
	}
	filesPath, err := getFilesLayout(sysprepSourcePath)
	if err != nil {
		return err
	}

	return createIsoImageAndSetFileOwnership(volumeName, filesPath, size)
}

func createIsoImageAndSetFileOwnership(volumeName string, filesPath []string, size int64) error {
	disk := GetSysprepDiskPath(volumeName)
	if err := createIsoConfigImage(disk, sysprepVolumeLabel, filesPath, size); err != nil {
		return err
	}
	if err := ephemeraldiskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(disk); err != nil {
		return err
	}

	return nil
}
