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
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
)

type IsoCreationFunc func(isoOutFile string, inFiles []string) error

var cloudInitLocalDir = "/var/run/libvirt/kubevirt"
var cloudInitOwner = "qemu"
var cloudInitIsoFunc = defaultIsoFunc

const noCloudFile = "noCloud.iso"

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
	return cmd.Run()
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	exists := false

	if err == nil {
		exists = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return exists, err
}

func md5CheckSum(filePath string) ([]byte, error) {
	var result []byte

	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)

	if err != nil {
		return result, err
	}

	result = hash.Sum(result)
	return result, nil
}

func setFileOwnership(username string, file string) error {
	usrObj, err := user.Lookup(username)
	if err != nil {
		return err
	}

	uid, err := strconv.Atoi(usrObj.Uid)
	if err != nil {
		return err
	}

	gid, err := strconv.Atoi(usrObj.Gid)
	if err != nil {
		return err
	}

	return os.Chown(file, uid, gid)
}

func filesAreEqual(path1 string, path2 string) (bool, error) {
	exists, err := fileExists(path1)
	if err != nil {
		return false, err
	} else if exists == false {
		return false, nil
	}

	exists, err = fileExists(path2)
	if err != nil {
		return false, err
	} else if exists == false {
		return false, nil
	}

	sum1, err := md5CheckSum(path1)
	if err != nil {
		return false, err
	}
	sum2, err := md5CheckSum(path2)
	if err != nil {
		return false, err
	}

	return bytes.Equal(sum1, sum2), nil
}

// The unit test suite uses this function
func SetIsoCreationFunction(isoFunc IsoCreationFunc) {
	cloudInitIsoFunc = isoFunc
}

// The unit test suite uses this function
func SetLocalDataOwner(user string) {
	cloudInitOwner = user
}

func SetLocalDirectory(dir string) {
	cloudInitLocalDir = dir
}

func GetDomainBasePath(domain string, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", cloudInitLocalDir, namespace, domain)
}

// This is called right before a VM is defined with libvirt.
// If the cloud-init type requires altering the domain, this
// is the place to do that.
func InjectDomainData(vm *v1.VM) (*v1.VM, error) {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	if vm.Spec.CloudInit == nil {
		return vm, nil
	}

	err := ValidateArgs(vm.Spec.CloudInit)
	if err != nil {
		return vm, err
	}

	switch vm.Spec.CloudInit.DataSource {
	case dataSourceNoCloud:
		filePath := fmt.Sprintf("%s/%s", GetDomainBasePath(domain, namespace), noCloudFile)

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

func ValidateArgs(spec *v1.CloudInitSpec) error {
	if spec == nil {
		return nil
	}

	switch spec.DataSource {
	case dataSourceNoCloud:
		if spec.NoCloudData == nil {
			return errors.New(fmt.Sprintf("DataSource %s does not have the required data initialized", spec.DataSource))
		}
		if spec.NoCloudData.UserDataBase64 == "" {
			return errors.New(fmt.Sprintf("userDataBase64 is required for cloudInit type %s", spec.DataSource))
		}
		if spec.NoCloudData.MetaDataBase64 == "" {
			return errors.New(fmt.Sprintf("metaDataBase64 is required for cloudInit type %s", spec.DataSource))
		}
		if spec.NoCloudData.DiskTarget == "" {
			return errors.New(fmt.Sprintf("noCloudTarget is required for cloudInit type %s", spec.DataSource))
		}
	default:
		return errors.New(fmt.Sprintf("Unknown CloudInit dataSource %s", spec.DataSource))
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

func RemoveLocalData(domain string, namespace string) {
	domainBasePath := GetDomainBasePath(domain, namespace)
	os.RemoveAll(domainBasePath)
}

func GenerateLocalData(domain string, namespace string, spec *v1.CloudInitSpec) error {
	if spec == nil {
		return nil
	}

	err := ValidateArgs(spec)
	if err != nil {
		return err
	}

	domainBasePath := GetDomainBasePath(domain, namespace)
	os.MkdirAll(domainBasePath, 0755)

	switch spec.DataSource {
	case dataSourceNoCloud:
		metaFile := fmt.Sprintf("%s/%s", domainBasePath, "meta-data")
		userFile := fmt.Sprintf("%s/%s", domainBasePath, "user-data")
		iso := fmt.Sprintf("%s/%s", domainBasePath, noCloudFile)
		isoStaging := fmt.Sprintf("%s/%s.staging", domainBasePath, noCloudFile)
		userData64 := spec.NoCloudData.UserDataBase64
		metaData64 := spec.NoCloudData.MetaDataBase64

		os.Remove(userFile)
		os.Remove(metaFile)
		os.Remove(isoStaging)

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

		files := make([]string, 0, 2)
		files = append(files, metaFile)
		files = append(files, userFile)
		cloudInitIsoFunc(isoStaging, files)
		os.Remove(metaFile)
		os.Remove(userFile)
		if err != nil {
			return err
		}

		err = setFileOwnership(cloudInitOwner, isoStaging)
		if err != nil {
			return err
		}

		isEqual, err := filesAreEqual(iso, isoStaging)
		if err != nil {
			return err
		}

		// Only replace the dynamically generated iso if it has a different checksum
		if isEqual {
			os.Remove(isoStaging)
		} else {
			os.Remove(iso)
			os.Rename(isoStaging, iso)
		}

		logging.DefaultLogger().V(2).Info().Msg(fmt.Sprintf("generated nocloud iso file %s", iso))
	}
	return nil
}
