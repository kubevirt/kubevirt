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

// GetConfigMapSourcePath returns a path to ConfigMap mounted on a pod
func GetConfigMapSourcePath(volumeName string) string {
	return filepath.Join(ConfigMapSourceDir, volumeName)
}

// GetConfigMapDiskPath returns a path to ConfigMap iso image created based on a volume name
func GetConfigMapDiskPath(volumeName string) string {
	return filepath.Join(ConfigMapDisksDir, volumeName+".iso")
}

// CreateConfigMapDisks creates ConfigMap iso disks which are attached to vmis
func CreateConfigMapDisks(vmi *v1.VirtualMachineInstance, emptyIso bool) error {
	for _, volume := range vmi.Spec.Volumes {
		if volume.ConfigMap != nil {
			var filesPath []string
			filesPath, err := getFilesLayout(GetConfigMapSourcePath(volume.Name))
			if err != nil {
				return err
			}

			disk := GetConfigMapDiskPath(volume.Name)
			vmiIsoSize, err := findIsoSize(vmi, &volume, emptyIso)
			if err != nil {
				return err
			}
			if err := createIsoConfigImage(disk, volume.ConfigMap.VolumeLabel, filesPath, vmiIsoSize); err != nil {
				return err
			}

			if err := ephemeraldiskutils.DefaultOwnershipManager.SetFileOwnership(disk); err != nil {
				return err
			}
		}
	}

	return nil
}
