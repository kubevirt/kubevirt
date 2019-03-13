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

	isoCreationFunc func(output string, files []string) error
)

const (
	// ConfigMap respresents a configmap type,
	// https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/
	ConfigMap Type = "configmap"
	// Secret represents a secret type,
	// https://kubernetes.io/docs/concepts/configuration/secret/
	Secret Type = "secret"
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
	// ServiceAccountSourceDir represents the location where the ServiceAccount token is attached to the pod
	ServiceAccountSourceDir = "/var/run/secrets/kubernetes.io/serviceaccount/"

	// ConfigMapDisksDir represents a path to ConfigMap iso images
	ConfigMapDisksDir = mountBaseDir + "/config-map-disks"
	// SecretDisksDir represents a path to Secrets iso images
	SecretDisksDir = mountBaseDir + "/secret-disks"
	// ServiceAccountDisksDir represents a path to the ServiceAccount iso image
	ServiceAccountDiskDir = mountBaseDir + "/service-account-disk"
	// ServiceAccountDisksName represents the name of the ServiceAccount iso image
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

func defaultCreateIsoImage(output string, files []string) error {
	var args []string
	args = append(args, "-output")
	args = append(args, output)
	args = append(args, "-volid")
	args = append(args, "cfgdata")
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

func createIsoConfigImage(output string, files []string) error {
	err := createISOImage(output, files)
	if err != nil {
		return err
	}
	return nil
}
