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
	"os"
	"os/exec"
	"path/filepath"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

	"kubevirt.io/kubevirt/pkg/util"

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

	createISOImage      = defaultCreateIsoImage
	createEmptyISOImage = defaultCreateEmptyIsoImage
)

// The unit test suite uses this function
func setIsoCreationFunction(isoFunc isoCreationFunc) {
	createISOImage = isoFunc
}

func getFilesLayout(dirPath string) ([]string, error) {
	var filesPath []string
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fileName := file.Name()
		filesPath = append(filesPath, fileName+"="+filepath.Join(dirPath, fileName))
	}
	return filesPath, nil
}

func defaultCreateIsoImage(iso string, volID string, files []string) error {
	if volID == "" {
		volID = "cfgdata"
	}

	isoStaging := fmt.Sprintf("%s.staging", iso)

	var args []string
	args = append(args, "-output")
	args = append(args, isoStaging)
	args = append(args, "-follow-links")
	args = append(args, "-volid")
	args = append(args, volID)
	args = append(args, "-joliet")
	args = append(args, "-rock")
	args = append(args, "-graft-points")
	args = append(args, "-partition_cyl_align")
	args = append(args, "on")
	args = append(args, files...)

	isoBinary := "xorrisofs"

	// #nosec No risk for attacket injection. Parameters are predefined strings
	cmd := exec.Command(isoBinary, args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	err = os.Rename(isoStaging, iso)

	return err
}

func defaultCreateEmptyIsoImage(iso string, size int64) error {
	isoStaging := fmt.Sprintf("%s.staging", iso)

	f, err := os.Create(isoStaging)
	if err != nil {
		return fmt.Errorf("failed to create empty iso: '%s'", isoStaging)
	}
	err = util.WriteBytes(f, 0, size)
	if err != nil {
		return err
	}
	util.CloseIOAndCheckErr(f, &err)
	if err != nil {
		return err
	}
	err = os.Rename(isoStaging, iso)

	return err
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

		filesPath, err := getFilesLayout(info.getSourcePath(&volume))
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
