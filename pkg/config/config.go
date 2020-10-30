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
	"io/ioutil"
	"os/exec"
	"path/filepath"
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
	ConfigMapSourceDir = mountBaseDir + "/config-map"
	// SecretSourceDir represents a location where Secrets is attached to the pod
	SecretSourceDir = mountBaseDir + "/secret"
	// DownwardAPISourceDir represents a location where downwardapi is attached to the pod
	DownwardAPISourceDir = mountBaseDir + "/downwardapi"
	// ServiceAccountSourceDir represents the location where the ServiceAccount token is attached to the pod
	ServiceAccountSourceDir = "/var/run/secrets/kubernetes.io/serviceaccount/"

	// ConfigMapDisksDir represents a path to ConfigMap iso images
	ConfigMapDisksDir = mountBaseDir + "/config-map-disks"
	// SecretDisksDir represents a path to Secrets iso images
	SecretDisksDir = mountBaseDir + "/secret-disks"
	// DownwardAPIDisksDir represents a path to DownwardAPI iso images
	DownwardAPIDisksDir = mountBaseDir + "/downwardapi-disks"
	// ServiceAccountDiskDir represents a path to the ServiceAccount iso image
	ServiceAccountDiskDir = mountBaseDir + "/service-account-disk"
	// ServiceAccountDiskName represents the name of the ServiceAccount iso image
	ServiceAccountDiskName = "service-account.iso"

	createISOImage = defaultCreateIsoImage
)

// The unit test suite uses this function
func setIsoCreationFunction(isoFunc isoCreationFunc) {
	createISOImage = isoFunc
}

func getFilesLayout(dirPath string) ([]string, error) {
	var filesPath []string
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fileName := file.Name()
		filesPath = append(filesPath, fileName+"="+filepath.Join(dirPath, fileName))
	}
	return filesPath, nil
}

func defaultCreateIsoImage(output string, volID string, files []string) error {

	if volID == "" {
		volID = "cfgdata"
	}

	var args []string
	args = append(args, "-output")
	args = append(args, output)
	args = append(args, "-follow-links")
	args = append(args, "-volid")
	args = append(args, volID)
	args = append(args, "-joliet")
	args = append(args, "-rock")
	args = append(args, "-graft-points")
	args = append(args, files...)

	cmd := exec.Command("genisoimage", args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func createIsoConfigImage(output string, volID string, files []string) error {
	err := createISOImage(output, volID, files)
	if err != nil {
		return err
	}
	return nil
}
