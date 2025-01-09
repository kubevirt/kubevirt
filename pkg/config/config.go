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
	"fmt"
	"path/filepath"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	utildisk "kubevirt.io/kubevirt/pkg/util/disk"

	v1 "kubevirt.io/api/core/v1"
)

type (
	// Type represents allowed config types like ConfigMap or Secret
	Type string

	isoCreationFunc func(output string, volID string, files []string) error
)

const (
	// ConfigMap respresents a configmap type,
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
	ConfigMap Type = "configmap"
	// Secret represents a secret type,
	// https://kubernetes.io/docs/concepts/configuration/secret/
	Secret Type = "secret"
	// DownwardAPI represents a DownwardAPI type,
	// https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/
	DownwardAPI Type = "downwardapi"
	// ServiceAccount represents a secret type,
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	ServiceAccount Type = "serviceaccount"

	mountBaseDir = "/var/run/kubevirt-private"
)

var (
	// ConfigMapSourceDir represents a location where ConfigMap is attached to the pod
	ConfigMapSourceDir = filepath.Join(mountBaseDir, "config-map")
	// SysprepSourceDir represents a location where a Sysprep is attached to the pod
	SysprepSourceDir = filepath.Join(mountBaseDir, "sysprep")
	// SecretSourceDir represents a location where Secrets is attached to the pod
	SecretSourceDir = filepath.Join(mountBaseDir, "secret")
	// DownwardAPISourceDir represents a location where downwardapi is attached to the pod
	DownwardAPISourceDir = filepath.Join(mountBaseDir, "downwardapi")
	// ServiceAccountSourceDir represents the location where the ServiceAccount token is attached to the pod
	ServiceAccountSourceDir = "/var/run/secrets/kubernetes.io/serviceaccount/"

	// ConfigMapDisksDir represents a path to ConfigMap iso images
	ConfigMapDisksDir = filepath.Join(mountBaseDir, "config-map-disks")
	// SecretDisksDir represents a path to Secrets iso images
	SecretDisksDir = filepath.Join(mountBaseDir, "secret-disks")
	// SysprepDisksDir represents a path to Syspreps iso images
	SysprepDisksDir = filepath.Join(mountBaseDir, "sysprep-disks")
	// DownwardAPIDisksDir represents a path to DownwardAPI iso images
	DownwardAPIDisksDir = filepath.Join(mountBaseDir, "downwardapi-disks")
	// DownwardMetricDisksDir represents a path to DownwardMetric block disk
	DownwardMetricDisksDir = filepath.Join(mountBaseDir, "downwardmetric-disk")
	// DownwardMetricDisks represents the disk location for the DownwardMetric disk
	DownwardMetricDisk = filepath.Join(DownwardAPIDisksDir, "vhostmd0")
	// ServiceAccountDiskDir represents a path to the ServiceAccount iso image
	ServiceAccountDiskDir = filepath.Join(mountBaseDir, "service-account-disk")
	// ServiceAccountDiskName represents the name of the ServiceAccount iso image
	ServiceAccountDiskName = "service-account.iso"

	createISOImage      = utildisk.CreateIsoImage
	createEmptyISOImage = utildisk.CreateEmptyIsoImage
)

// The unit test suite uses this function
func setIsoCreationFunction(isoFunc isoCreationFunc) {
	createISOImage = isoFunc
}

func createIsoConfigImage(output string, volID string, files []string, size int64) error {
	var err error
	if size == 0 {
		err = createISOImage(output, volID, files)
	} else {
		err = createEmptyISOImage(output, size)
	}
	if err != nil {
		return err
	}
	return nil
}

func findIsoSize(vmi *v1.VirtualMachineInstance, volume *v1.Volume, emptyIso bool) (int64, error) {
	if emptyIso {
		for _, vs := range vmi.Status.VolumeStatus {
			if vs.Name == volume.Name {
				return vs.Size, nil
			}
		}
		return 0, fmt.Errorf("failed to find the status of volume %s", volume.Name)
	}
	return 0, nil
}

type volumeInfo interface {
	isValidType(*v1.Volume) bool
	getSourcePath(*v1.Volume) string
	getIsoPath(*v1.Volume) string
	getLabel(*v1.Volume) string
}

func createIsoDisksForConfigVolumes(vmi *v1.VirtualMachineInstance, emptyIso bool, info volumeInfo) error {
	volumes := make(map[string]v1.Volume)
	for _, volume := range vmi.Spec.Volumes {
		if info.isValidType(&volume) {
			volumes[volume.Name] = volume
		}
	}

	for _, disk := range vmi.Spec.Domain.Devices.Disks {
		volume, ok := volumes[disk.Name]
		if !ok {
			continue
		}

		filesPath, err := utildisk.GetFilesLayoutForISO(info.getSourcePath(&volume))
		if err != nil {
			return err
		}

		isoPath := info.getIsoPath(&volume)
		vmiIsoSize, err := findIsoSize(vmi, &volume, emptyIso)
		if err != nil {
			return err
		}

		label := info.getLabel(&volume)
		if err := createIsoConfigImage(isoPath, label, filesPath, vmiIsoSize); err != nil {
			return err
		}

		if err := ephemeraldiskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(isoPath); err != nil {
			return err
		}
	}

	return nil
}
