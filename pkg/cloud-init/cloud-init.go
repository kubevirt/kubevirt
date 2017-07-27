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
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	kubev1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/precond"
)

const noCloudBasePath = "/var/run/libvirt/kubevirt"
const noCloudFile = "noCloud.iso"

// Supported DataSources
const (
	dataSourceNoCloud = "noCloud"
)

// This is called right before a VM is defined with libvirt.
// If the cloud-init type requires altering the domain, this
// is the place to do that.
func InjectDomainData(vm *v1.VM) (*v1.VM, error) {
	if vm.Spec.CloudInit == nil {
		return vm, nil
	}

	err := ValidateArgs(vm)
	if err != nil {
		return vm, err
	}

	switch vm.Spec.CloudInit.DataSource {
	case dataSourceNoCloud:
		filePath := fmt.Sprintf("%s/%s", getDiskPath(vm), noCloudFile)

		newDisk := v1.Disk{}
		newDisk.Type = "file"
		newDisk.Device = "disk"
		newDisk.Driver = &v1.DiskDriver{
			Type: "raw",
			Name: "qemu",
		}
		newDisk.Source.File = filePath
		newDisk.Target = v1.DiskTarget{
			Device: vm.Spec.CloudInit.NoCloudData.DiskTarget,
			Bus:    "virtio",
		}

		vm.Spec.Domain.Devices.Disks = append(vm.Spec.Domain.Devices.Disks, newDisk)
	default:
		return vm, errors.New(fmt.Sprintf("Unknown CloudInit type %s", vm.Spec.CloudInit.DataSource))
	}

	return vm, nil
}

// TODO must make directory something that is passed in.
func getDiskPath(vm *v1.VM) string {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	return fmt.Sprintf("%s/%s/%s", noCloudBasePath, namespace, domain)
}

func ValidateArgs(vm *v1.VM) error {
	if vm.Spec.CloudInit == nil {
		return nil
	}

	switch vm.Spec.CloudInit.DataSource {
	case dataSourceNoCloud:
		if vm.Spec.CloudInit.NoCloudData == nil {
			return errors.New(fmt.Sprintf("DataSource %s does not have the required data initialized", vm.Spec.CloudInit.DataSource))
		}
		if vm.Spec.CloudInit.NoCloudData.UserDataBase64 == "" {
			return errors.New(fmt.Sprintf("userDataBase64 is required for cloudInit type %s", vm.Spec.CloudInit.DataSource))
		}
		if vm.Spec.CloudInit.NoCloudData.MetaDataBase64 == "" {
			return errors.New(fmt.Sprintf("metaDataBase64 is required for cloudInit type %s", vm.Spec.CloudInit.DataSource))
		}
		if vm.Spec.CloudInit.NoCloudData.DiskTarget == "" {
			return errors.New(fmt.Sprintf("noCloudTarget is required for cloudInit type %s", vm.Spec.CloudInit.DataSource))
		}
	default:
		return errors.New(fmt.Sprintf("Unknown CloudInit dataSource %s", vm.Spec.CloudInit.DataSource))
	}

	return nil
}

func ApplyMetadata(vm *v1.VM) {
	if vm.Spec.CloudInit == nil {
		return
	}

	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	// TODO Put local-hostname in MetaData once we get pod DNS working with VMs
	msg := fmt.Sprintf("instance-id: %s-%s\n", namespace, domain)
	vm.Spec.CloudInit.NoCloudData.MetaDataBase64 = base64.StdEncoding.EncodeToString([]byte(msg))
}

// This function removes any local data associated with cloud-init
// Not all cloud-init types require local data.
func RemoveLocalData(directory string) {
	dataSource := os.Getenv("CLOUD_INIT_DS")
	if dataSource == "" {
		return
	}

	switch dataSource {
	case dataSourceNoCloud:
		metaFile := fmt.Sprintf("%s/%s", directory, "meta-data")
		userFile := fmt.Sprintf("%s/%s", directory, "user-data")
		iso := fmt.Sprintf("%s/%s", directory, os.Getenv("NO_CLOUD_FILE"))

		os.Remove(metaFile)
		os.Remove(userFile)
		os.Remove(iso)
		log.Printf("Removed nocloud local data files")
	}
}

func GenerateLocalData(directory string) error {
	dataSource := os.Getenv("CLOUD_INIT_DS")
	if dataSource == "" {
		return nil
	}

	switch dataSource {
	case dataSourceNoCloud:
		metaFile := fmt.Sprintf("%s/%s", directory, "meta-data")
		userFile := fmt.Sprintf("%s/%s", directory, "user-data")
		iso := fmt.Sprintf("%s/%s", directory, os.Getenv("NO_CLOUD_FILE"))
		userData64 := os.Getenv("USER_DATA_BASE64")
		metaData64 := os.Getenv("META_DATA_BASE64")

		userDataBytes, err := base64.StdEncoding.DecodeString(userData64)
		if err != nil {
			return err
		}
		metaDataBytes, err := base64.StdEncoding.DecodeString(metaData64)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(userFile, userDataBytes, 0644)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(metaFile, metaDataBytes, 0644)
		if err != nil {
			return err
		}

		//genisoimage -output $ISO -volid cidata -joliet -rock $USER_DATA $META_DATA
		cmd := exec.Command("genisoimage",
			"-output",
			iso,
			"-volid",
			"cidata",
			"-joliet",
			"-rock",
			userFile,
			metaFile)

		err = cmd.Run()
		if err != nil {
			return err
		}
		os.Chown(iso, 1111, 1111)
		log.Printf("Generated nocloud local data files")
	}
	return nil
}

func GenerateEnvVars(vm *v1.VM) ([]kubev1.EnvVar, error) {
	var containerVars []kubev1.EnvVar

	if vm.Spec.CloudInit == nil {
		return containerVars, nil
	}

	err := ValidateArgs(vm)
	if err != nil {
		return containerVars, err
	}

	switch vm.Spec.CloudInit.DataSource {
	case dataSourceNoCloud:
		containerVars = append(containerVars, kubev1.EnvVar{
			Name:  "CLOUD_INIT_DS",
			Value: dataSourceNoCloud,
		})
		containerVars = append(containerVars, kubev1.EnvVar{
			Name:  "NO_CLOUD_PATH",
			Value: getDiskPath(vm),
		})

		containerVars = append(containerVars, kubev1.EnvVar{
			Name:  "NO_CLOUD_FILE",
			Value: noCloudFile,
		})
		containerVars = append(containerVars, kubev1.EnvVar{
			Name:  "USER_DATA_BASE64",
			Value: vm.Spec.CloudInit.NoCloudData.UserDataBase64,
		})
		containerVars = append(containerVars, kubev1.EnvVar{
			Name:  "META_DATA_BASE64",
			Value: vm.Spec.CloudInit.NoCloudData.MetaDataBase64,
		})
	}
	return containerVars, nil
}
