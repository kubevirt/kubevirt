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

package cloudinit

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
)

type IsoCreationFunc func(isoOutFile string, inFiles []string) error

var cloudInitLocalDir = "/var/run/libvirt/cloud-init-dir"
var cloudInitOwner = "qemu"
var cloudInitIsoFunc = defaultIsoFunc

const NoCloudFile = "noCloud.iso"

// Supported DataSources
const (
	dataSourceNoCloud = "noCloud"
)

func defaultIsoFunc(isoOutFile string, inFiles []string) error {

	var args []string

	args = append(args, "-output")
	args = append(args, isoOutFile)
	args = append(args, "-volid")
	args = append(args, "cidata")
	args = append(args, "-joliet")
	args = append(args, "-rock")
	args = append(args, inFiles...)

	cmd := exec.Command("genisoimage", args...)

	err := cmd.Start()
	if err != nil {
		log.Log.V(2).Reason(err).Errorf("genisoimage cmd failed to start while generating iso file %s", isoOutFile)
		return err
	}

	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			log.Log.V(2).Errorf("Timed out generating cloud-init iso at path %s", isoOutFile)
			cmd.Process.Kill()
		case err := <-done:
			if err != nil {
				log.Log.V(2).Reason(err).Errorf("genisoimage returned non-zero exit code while generating iso file %s", isoOutFile)
				return err
			}
			return nil
		}
	}
}

// The unit test suite uses this function
func SetIsoCreationFunction(isoFunc IsoCreationFunc) {
	cloudInitIsoFunc = isoFunc
}

// The unit test suite uses this function
func SetLocalDataOwner(user string) {
	cloudInitOwner = user
}

func SetLocalDirectory(dir string) error {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("unable to initialize cloudInit local cache directory (%s). %v", dir, err)
	}

	exists, err := diskutils.FileExists(dir)
	if err != nil {
		return fmt.Errorf("CloudInit local cache directory (%s) does not exist or is inaccessible. %v", dir, err)
	} else if exists == false {
		return fmt.Errorf("CloudInit local cache directory (%s) does not exist or is inaccessible", dir)
	}

	cloudInitLocalDir = dir
	return nil
}

func GetDomainBasePath(domain string, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", cloudInitLocalDir, namespace, domain)
}

func RemoveLocalData(domain string, namespace string) error {
	domainBasePath := GetDomainBasePath(domain, namespace)
	err := os.RemoveAll(domainBasePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func GetCloudInitNoCloudSource(vmi *v1.VirtualMachineInstance) *v1.CloudInitNoCloudSource {
	precond.MustNotBeNil(vmi)
	// search various places cloud init spec may live.
	// at the moment, cloud init only exists on disks.
	for _, volume := range vmi.Spec.Volumes {
		if volume.CloudInitNoCloud != nil {
			return volume.CloudInitNoCloud
		}
	}
	return nil
}

func ResolveSecrets(source *v1.CloudInitNoCloudSource, namespace string, clientset kubecli.KubevirtClient) error {
	precond.CheckNotNil(source)
	precond.CheckNotEmpty(namespace)
	precond.CheckNotNil(clientset)

	if source.UserDataSecretRef == nil && source.NetworkDataSecretRef == nil {
		return nil
	}

	if source.UserDataSecretRef != nil {
		secretID := source.UserDataSecretRef.Name

		secret, err := clientset.CoreV1().Secrets(namespace).Get(secretID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		userData, ok := secret.Data["userdata"]
		if ok == false {
			return fmt.Errorf("userdata key not found in k8s secret %s %v", secretID, err)
		}
		source.UserData = string(userData)
	}

	if source.NetworkDataSecretRef != nil {
		secretID := source.NetworkDataSecretRef.Name

		secret, err := clientset.CoreV1().Secrets(namespace).Get(secretID, metav1.GetOptions{})
		if err != nil {
			return err
		}

		networkData, ok := secret.Data["networkdata"]
		if ok == false {
			return fmt.Errorf("networkdata key not found in k8s secret %s %v", secretID, err)
		}
		source.NetworkData = string(networkData)
	}

	return nil
}

func GenerateLocalData(vmiName string, hostname string, namespace string, source *v1.CloudInitNoCloudSource) error {
	precond.MustNotBeEmpty(vmiName)
	precond.MustNotBeNil(source)

	domainBasePath := GetDomainBasePath(vmiName, namespace)
	err := os.MkdirAll(domainBasePath, 0755)
	if err != nil {
		log.Log.V(2).Reason(err).Errorf("unable to create cloud-init base path %s", domainBasePath)
		return err
	}

	metaFile := fmt.Sprintf("%s/%s", domainBasePath, "meta-data")
	userFile := fmt.Sprintf("%s/%s", domainBasePath, "user-data")
	networkFile := fmt.Sprintf("%s/%s", domainBasePath, "network-config")
	iso := fmt.Sprintf("%s/%s", domainBasePath, NoCloudFile)
	isoStaging := fmt.Sprintf("%s/%s.staging", domainBasePath, NoCloudFile)

	var userData []byte
	if source.UserData != "" {
		userData = []byte(source.UserData)
	} else if source.UserDataBase64 != "" {
		userData, err = base64.StdEncoding.DecodeString(source.UserDataBase64)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("userDataBase64 or userData is required for no-cloud data source")
	}

	var networkData []byte
	if source.NetworkData != "" {
		networkData = []byte(source.NetworkData)
	} else if source.NetworkDataBase64 != "" {
		networkData, err = base64.StdEncoding.DecodeString(source.NetworkDataBase64)
		if err != nil {
			return err
		}
	}

	metaData := []byte(fmt.Sprintf("{ \"instance-id\": \"%s.%s\", \"local-hostname\": \"%s\" }\n", vmiName, namespace, hostname))

	diskutils.RemoveFile(userFile)
	diskutils.RemoveFile(metaFile)
	diskutils.RemoveFile(networkFile)
	diskutils.RemoveFile(isoStaging)

	err = ioutil.WriteFile(userFile, userData, 0644)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(metaFile, metaData, 0644)
	if err != nil {
		return err
	}

	files := make([]string, 0, 3)
	files = append(files, metaFile)
	files = append(files, userFile)

	if len(networkData) > 0 {
		err = ioutil.WriteFile(networkFile, networkData, 0644)
		if err != nil {
			return err
		}
		files = append(files, networkFile)
	}

	err = cloudInitIsoFunc(isoStaging, files)
	if err != nil {
		return err
	}
	diskutils.RemoveFile(metaFile)
	diskutils.RemoveFile(userFile)
	diskutils.RemoveFile(networkFile)

	err = diskutils.SetFileOwnership(cloudInitOwner, isoStaging)
	if err != nil {
		return err
	}

	isEqual, err := diskutils.FilesAreEqual(iso, isoStaging)
	if err != nil {
		return err
	}

	// Only replace the dynamically generated iso if it has a different checksum
	if isEqual {
		diskutils.RemoveFile(isoStaging)
	} else {
		diskutils.RemoveFile(iso)
		err = os.Rename(isoStaging, iso)
		if err != nil {
			// This error is not something we need to block iso creation for.
			log.Log.Reason(err).Errorf("Cloud-init failed to rename file %s to %s", isoStaging, iso)
			return err
		}
	}

	log.Log.V(2).Infof("generated nocloud iso file %s", iso)
	return nil
}

// Lists all vmis cloud-init has local data for
func ListVmWithLocalData() ([]*v1.VirtualMachineInstance, error) {
	return diskutils.ListVmWithEphemeralDisk(cloudInitLocalDir)
}
