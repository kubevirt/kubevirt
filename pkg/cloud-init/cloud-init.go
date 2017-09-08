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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	model "github.com/jeevatkm/go-model"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/precond"
)

type IsoCreationFunc func(isoOutFile string, inFiles []string) error

var cloudInitLocalDir = "/var/run/libvirt/cloud-init-dir"
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

	err := cmd.Start()
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("genisoimage cmd failed to start while generating iso file %s", isoOutFile))
		return err
	}

	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			logging.DefaultLogger().V(2).Error().Msg(fmt.Sprintf("Timed out generating cloud-init iso at path %s", isoOutFile))
			cmd.Process.Kill()
		case err := <-done:
			if err != nil {
				logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("genisoimage returned non-zero exit code while generating iso file %s", isoOutFile))
				return err
			}
			return nil
		}
	}
}

func removeFile(path string) error {
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("failed to remove cloud-init temporary data file %s", path))
		return err
	}
	return nil
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
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to look up username %s", username))
		return err
	}

	uid, err := strconv.Atoi(usrObj.Uid)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to find uid for username %s", username))
		return err
	}

	gid, err := strconv.Atoi(usrObj.Gid)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to find gid for username %s", username))
		return err
	}

	return os.Chown(file, uid, gid)
}

func filesAreEqual(path1 string, path2 string) (bool, error) {
	exists, err := fileExists(path1)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unexpected error encountered while attempting to determine if %s exists", path1))
		return false, err
	} else if exists == false {
		return false, nil
	}

	exists, err = fileExists(path2)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unexpected error encountered while attempting to determine if %s exists", path2))
		return false, err
	} else if exists == false {
		return false, nil
	}

	sum1, err := md5CheckSum(path1)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("calculating md5 checksum failed for %s", path1))
		return false, err
	}
	sum2, err := md5CheckSum(path2)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("calculating md5 checksum failed for %s", path2))
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

func SetLocalDirectory(dir string) error {

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return errors.New(fmt.Sprintf("Unable to initalize cloudInit local cache directory (%s). %v", dir, err))
	}

	exists, err := fileExists(dir)
	if err != nil {
		return errors.New(fmt.Sprintf("CloudInit local cache directory (%s) does not exist or is inaccessible. %v", dir, err))
	} else if exists == false {
		return errors.New(fmt.Sprintf("CloudInit local cache directory (%s) does not exist or is inaccessible.", dir))
	}

	cloudInitLocalDir = dir
	return nil
}

func GetDomainBasePath(domain string, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", cloudInitLocalDir, namespace, domain)
}

// This is called right before a VM is defined with libvirt.
// If the cloud-init type requires altering the domain, this
// is the place to do that.
func MapCloudInitDisks(vm *v1.VM) (*v1.VM, error) {
	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	spec := GetCloudInitSpec(vm)
	if spec == nil {
		return vm, nil
	}

	err := ValidateArgs(spec)
	if err != nil {
		return vm, err
	}

	dataSource := getDataSource(spec)
	switch dataSource {
	case dataSourceNoCloud:
		vmCopy := &v1.VM{}
		model.Copy(vmCopy, vm)
		filePath := fmt.Sprintf("%s/%s", GetDomainBasePath(domain, namespace), noCloudFile)

		for idx, disk := range vmCopy.Spec.Domain.Devices.Disks {
			if disk.Type == "file" && disk.CloudInit != nil {
				newDisk := v1.Disk{}
				newDisk.Type = "file"
				newDisk.Device = "disk"
				newDisk.Driver = &v1.DiskDriver{
					Type: "raw",
					Name: "qemu",
				}
				newDisk.Source.File = filePath
				newDisk.Target = disk.Target
				vmCopy.Spec.Domain.Devices.Disks[idx] = newDisk
			}
		}
		return vmCopy, nil
	default:
		return vm, errors.New(fmt.Sprintf("Unknown CloudInit type %s", dataSource))
	}
}

func ValidateArgs(spec *v1.CloudInitSpec) error {
	if spec == nil {
		return nil
	}

	dataSource := getDataSource(spec)
	switch dataSource {
	case dataSourceNoCloud:
		if spec.NoCloudData == nil {
			return errors.New(fmt.Sprintf("DataSource %s does not have the required data initialized", dataSource))
		}
		if spec.NoCloudData.UserDataBase64 == "" {
			return errors.New(fmt.Sprintf("userDataBase64 is required for cloudInit type %s", dataSource))
		}
		if spec.NoCloudData.MetaDataBase64 == "" {
			return errors.New(fmt.Sprintf("metaDataBase64 is required for cloudInit type %s", dataSource))
		}
	default:
		return errors.New(fmt.Sprintf("Unknown CloudInit dataSource %s", dataSource))
	}

	return nil
}

// Place metadata auto-generation code in here
func ApplyMetadata(vm *v1.VM) {
	spec := GetCloudInitSpec(vm)
	if spec == nil {
		return
	}

	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())

	switch getDataSource(spec) {
	case dataSourceNoCloud:
		// Only autogenerate metadata if user defined metadata does not exist
		if spec.NoCloudData.MetaDataBase64 != "" {
			return
		}
		// TODO Put local-hostname in MetaData once we get pod DNS working with VMs
		msg := fmt.Sprintf("instance-id: %s-%s\n", domain, namespace)
		spec.NoCloudData.MetaDataBase64 = base64.StdEncoding.EncodeToString([]byte(msg))
	}
}

func RemoveLocalData(domain string, namespace string) error {
	domainBasePath := GetDomainBasePath(domain, namespace)
	err := os.RemoveAll(domainBasePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func GetCloudInitSpec(vm *v1.VM) *v1.CloudInitSpec {
	// search various places cloud init spec may live.
	// at the moment, cloud init only exists on disks.
	for _, disk := range vm.Spec.Domain.Devices.Disks {
		if disk.CloudInit != nil {
			return disk.CloudInit
		}
	}
	return nil
}

func getDataSource(spec *v1.CloudInitSpec) string {
	if spec == nil {
		return ""
	}

	if spec.NoCloudData != nil {
		return dataSourceNoCloud
	}
	return ""
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
	err = os.MkdirAll(domainBasePath, 0755)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to create cloud-init base path %s", domainBasePath))
		return err
	}

	switch getDataSource(spec) {
	case dataSourceNoCloud:
		metaFile := fmt.Sprintf("%s/%s", domainBasePath, "meta-data")
		userFile := fmt.Sprintf("%s/%s", domainBasePath, "user-data")
		iso := fmt.Sprintf("%s/%s", domainBasePath, noCloudFile)
		isoStaging := fmt.Sprintf("%s/%s.staging", domainBasePath, noCloudFile)
		userData64 := spec.NoCloudData.UserDataBase64
		metaData64 := spec.NoCloudData.MetaDataBase64

		removeFile(userFile)
		removeFile(metaFile)
		removeFile(isoStaging)

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
		err = cloudInitIsoFunc(isoStaging, files)
		if err != nil {
			return err
		}
		removeFile(metaFile)
		removeFile(userFile)

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
			removeFile(isoStaging)
		} else {
			removeFile(iso)
			err = os.Rename(isoStaging, iso)
			if err != nil {
				// This error is not something we need to block iso creation for.
				logging.DefaultLogger().Error().Reason(err).Msg(fmt.Sprintf("Cloud-init failed to rename file %s to %s", isoStaging, iso))
				return err
			}
		}

		logging.DefaultLogger().V(2).Info().Msg(fmt.Sprintf("generated nocloud iso file %s", iso))
	}
	return nil
}

// Lists all vms cloud-init has local data for
func ListVmWithLocalData() ([]*v1.VM, error) {
	var keys []*v1.VM

	err := filepath.Walk(cloudInitLocalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() == false {
			return nil
		}

		relativePath := strings.TrimPrefix(path, cloudInitLocalDir+"/")
		if relativePath == "" {
			return nil
		}
		dirs := strings.Split(relativePath, "/")
		if len(dirs) != 2 {
			return nil
		}

		namespace := dirs[0]
		domain := dirs[1]
		if namespace == "" || domain == "" {
			return nil
		}
		keys = append(keys, v1.NewVMReferenceFromNameWithNS(dirs[0], dirs[1]))
		return nil
	})

	return keys, err
}
