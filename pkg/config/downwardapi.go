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

package config

import (
	"path/filepath"

	v1 "kubevirt.io/client-go/apis/core/v1"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
)

// GetDownwardAPISourcePath returns a path to downwardAPI mounted on a pod
func GetDownwardAPISourcePath(volumeName string) string {
	return filepath.Join(DownwardAPISourceDir, volumeName)
}

// GetDownwardAPIDiskPath returns a path to downwardAPI iso image created based on volume name
func GetDownwardAPIDiskPath(volumeName string) string {
	return filepath.Join(DownwardAPIDisksDir, volumeName+".iso")
}

// CreateDownwardAPIDisks creates DownwardAPI iso disks which are attached to vmis
func CreateDownwardAPIDisks(vmi *v1.VirtualMachineInstance, emptyIso bool) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.DownwardAPI != nil {

			var filesPath []string
			filesPath, err := getFilesLayout(GetDownwardAPISourcePath(volume.Name))
			if err != nil {
				return err
			}

			disk := GetDownwardAPIDiskPath(volume.Name)
			vmiIsoSize, err := findIsoSize(vmi, &volume, emptyIso)
			if err != nil {
				return err
			}
			if err := createIsoConfigImage(disk, volume.DownwardAPI.VolumeLabel, filesPath, vmiIsoSize); err != nil {
				return err
			}

			if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(disk); err != nil {
				return err
			}
		}
	}

	return nil
}
