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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
)

type IsoCreationFunc func(isoOutFile, volumeID string, inDir string) error

var cloudInitLocalDir = "/var/run/libvirt/cloud-init-dir"
var cloudInitIsoFunc = defaultIsoFunc

// Locations of data source disk files
const (
	noCloudFile     = "noCloud.iso"
	configDriveFile = "configdrive.iso"
)

type DataSourceType string
type DeviceMetadataType string

const (
	DataSourceNoCloud     DataSourceType     = "noCloud"
	DataSourceConfigDrive DataSourceType     = "configDrive"
	NICMetadataType       DeviceMetadataType = "nic"
	DiskMetadataType      DeviceMetadataType = "disk"
)

// CloudInitData is a data source independent struct that
// holds cloud-init user and network data
type CloudInitData struct {
	DataSource  DataSourceType
	MetaData    *Metadata
	UserData    string
	NetworkData string
	DevicesData *[]DeviceData
}

type Metadata struct {
	InstanceID    string        `json:"instance-id"`
	LocalHostname string        `json:"local-hostname,omitempty"`
	Hostname      string        `json:"hostname,omitempty"`
	UUID          string        `json:"uuid,omitempty"`
	Devices       *[]DeviceData `json:"devices,omitempty"`
}

type DeviceData struct {
	Type    DeviceMetadataType `json:"type"`
	Bus     string             `json:"bus"`
	Address string             `json:"address"`
	MAC     string             `json:"mac,omitempty"`
	Serial  string             `json:"serial,omitempty"`
	Tags    []string           `json:"tags"`
}

// IsValidCloudInitData checks if the given CloudInitData object is valid in the sense that GenerateLocalData can be called with it.
func IsValidCloudInitData(cloudInitData *CloudInitData) bool {
	return cloudInitData != nil && cloudInitData.UserData != "" && cloudInitData.MetaData != nil
}

// ReadCloudInitVolumeDataSource scans the given VMI for CloudInit volumes and
// reads their content into a CloudInitData struct. Does not resolve secret refs.
// To ensure that secrets are read correctly, call ResolveNoCloudSecrets beforehand.
func ReadCloudInitVolumeDataSource(vmi *v1.VirtualMachineInstance) (cloudInitData *CloudInitData, err error) {
	precond.MustNotBeNil(vmi)
	hostname := dns.SanitizeHostname(vmi)

	for _, volume := range vmi.Spec.Volumes {
		if volume.CloudInitNoCloud != nil {
			cloudInitData, err = readCloudInitNoCloudSource(volume.CloudInitNoCloud)
			cloudInitData.MetaData = readCloudInitNoCloudMetaData(vmi.Name, hostname, vmi.Namespace)
			return cloudInitData, err
		}
		if volume.CloudInitConfigDrive != nil {
			cloudInitData, err = readCloudInitConfigDriveSource(volume.CloudInitConfigDrive)
			cloudInitData.MetaData = readCloudInitConfigDriveMetaData(string(vmi.UID), vmi.Name, hostname, vmi.Namespace)
			return cloudInitData, err
		}
	}
	return nil, nil
}

// ResolveNoCloudSecrets is looking for CloudInitNoCloud volumes with UserDataSecretRef
// requests. It reads the `userdata` secret the corresponds to the given CloudInitNoCloud
// volume and sets the UserData field on that volume.
//
// Note: when using this function, make sure that your code can access the secret volumes.
func ResolveNoCloudSecrets(vmi *v1.VirtualMachineInstance, secretSourceDir string) error {
	volume := findCloudInitNoCloudSecretVolume(vmi.Spec.Volumes)
	if volume == nil {
		return nil
	}

	baseDir := filepath.Join(secretSourceDir, volume.Name)
	userData, userDataError := readFileFromDir(baseDir, "userdata")
	networkData, networkDataError := readFileFromDir(baseDir, "networkdata")
	if userDataError != nil && networkDataError != nil {
		return fmt.Errorf("no cloud-init data-source found at volume: %s", volume.Name)
	}

	if userData != "" {
		volume.CloudInitNoCloud.UserData = userData
	}
	if networkData != "" {
		volume.CloudInitNoCloud.NetworkData = networkData
	}

	return nil
}

// ResolveConfigDriveSecrets is looking for CloudInitConfigDriveSource volume source with
// UserDataSecretRef and NetworkDataSecretRef and resolves the secret from the corresponding
// VolumeMount.
//
// Note: when using this function, make sure that your code can access the secret volumes.
func ResolveConfigDriveSecrets(vmi *v1.VirtualMachineInstance, secretSourceDir string) error {
	volume := findCloudInitConfigDriveSecretVolume(vmi.Spec.Volumes)
	if volume == nil {
		return nil
	}

	baseDir := filepath.Join(secretSourceDir, volume.Name)
	userData, userDataError := readFileFromDir(baseDir, "userdata")
	networkData, networkDataError := readFileFromDir(baseDir, "networkdata")
	if userDataError != nil && networkDataError != nil {
		return fmt.Errorf("no cloud-init data-source found at volume: %s", volume.Name)
	}

	if userData != "" {
		volume.CloudInitConfigDrive.UserData = userData
	}
	if networkData != "" {
		volume.CloudInitConfigDrive.NetworkData = networkData
	}

	return nil
}

// findCloudInitConfigDriveSecretVolume loops over a given list of volumes and return a pointer
// to the first volume with a CloudInitConfigDrive source and UserDataSecretRef field set.
func findCloudInitConfigDriveSecretVolume(volumes []v1.Volume) *v1.Volume {
	for _, volume := range volumes {
		if volume.CloudInitConfigDrive == nil {
			continue
		}
		if volume.CloudInitConfigDrive.UserDataSecretRef != nil ||
			volume.CloudInitConfigDrive.NetworkDataSecretRef != nil {
			return &volume
		}
	}

	return nil
}

func readFileFromDir(basedir, secretFile string) (string, error) {
	userDataSecretFile := filepath.Join(basedir, secretFile)
	userDataSecret, err := ioutil.ReadFile(userDataSecretFile)
	if err != nil {
		log.Log.V(2).Reason(err).
			Errorf("could not read secret data from source: %s", userDataSecretFile)
		return "", err
	}
	return string(userDataSecret), nil
}

// findCloudInitNoCloudSecretVolume loops over a given list of volumes and return a pointer
// to the first CloudInitNoCloud volume with a UserDataSecretRef field set.
func findCloudInitNoCloudSecretVolume(volumes []v1.Volume) *v1.Volume {
	for _, volume := range volumes {
		if volume.CloudInitNoCloud != nil && volume.CloudInitNoCloud.UserDataSecretRef != nil {
			return &volume
		}
	}

	return nil
}

func readRawOrBase64Data(rawData, base64Data string) (string, error) {
	if rawData != "" {
		return rawData, nil
	} else if base64Data != "" {
		bytes, err := base64.StdEncoding.DecodeString(base64Data)
		return string(bytes), err
	}
	return "", nil
}

// readCloudInitData reads user and network data raw or in base64 encoding,
// regardless from which data source they are coming from
func readCloudInitData(userData, userDataBase64, networkData, networkDataBase64 string) (string, string, error) {
	readUserData, err := readRawOrBase64Data(userData, userDataBase64)
	if err != nil {
		return "", "", err
	}
	if readUserData == "" {
		return "", "", fmt.Errorf("userDataBase64 or userData is required for a cloud-init data source")
	}

	readNetworkData, err := readRawOrBase64Data(networkData, networkDataBase64)
	if err != nil {
		return "", "", err
	}

	return readUserData, readNetworkData, nil
}

func readCloudInitNoCloudSource(source *v1.CloudInitNoCloudSource) (*CloudInitData, error) {
	userData, networkData, err := readCloudInitData(source.UserData,
		source.UserDataBase64, source.NetworkData, source.NetworkDataBase64)
	if err != nil {
		return &CloudInitData{}, err
	}

	return &CloudInitData{
		DataSource:  DataSourceNoCloud,
		UserData:    userData,
		NetworkData: networkData,
	}, nil
}

func readCloudInitConfigDriveSource(source *v1.CloudInitConfigDriveSource) (*CloudInitData, error) {
	userData, networkData, err := readCloudInitData(source.UserData,
		source.UserDataBase64, source.NetworkData, source.NetworkDataBase64)
	if err != nil {
		return &CloudInitData{}, err
	}

	return &CloudInitData{
		DataSource:  DataSourceConfigDrive,
		UserData:    userData,
		NetworkData: networkData,
	}, nil
}

func readCloudInitNoCloudMetaData(name, hostname, namespace string) *Metadata {
	return &Metadata{
		InstanceID:    fmt.Sprintf("%s.%s", name, namespace),
		LocalHostname: hostname,
	}
}

func readCloudInitConfigDriveMetaData(uid, name, hostname, namespace string) *Metadata {
	return &Metadata{
		UUID:       uid,
		InstanceID: fmt.Sprintf("%s.%s", name, namespace),
		Hostname:   hostname,
	}
}

func defaultIsoFunc(isoOutFile, volumeID string, inDir string) error {

	var args []string

	args = append(args, "-output")
	args = append(args, isoOutFile)
	args = append(args, "-volid")
	args = append(args, volumeID)
	args = append(args, "-joliet")
	args = append(args, "-rock")
	args = append(args, inDir)

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

func getDomainBasePath(domain string, namespace string) string {
	return fmt.Sprintf("%s/%s/%s", cloudInitLocalDir, namespace, domain)
}

func GetIsoFilePath(source DataSourceType, domain, namespace string) string {
	switch source {
	case DataSourceNoCloud:
		return fmt.Sprintf("%s/%s", getDomainBasePath(domain, namespace), noCloudFile)
	case DataSourceConfigDrive:
		return fmt.Sprintf("%s/%s", getDomainBasePath(domain, namespace), configDriveFile)
	}
	return fmt.Sprintf("%s/%s", getDomainBasePath(domain, namespace), noCloudFile)
}

func removeLocalData(domain string, namespace string) error {
	domainBasePath := getDomainBasePath(domain, namespace)
	err := os.RemoveAll(domainBasePath)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

func GenerateLocalData(vmiName string, namespace string, data *CloudInitData) error {
	precond.MustNotBeEmpty(vmiName)
	precond.MustNotBeNil(data)

	domainBasePath := getDomainBasePath(vmiName, namespace)
	dataBasePath := fmt.Sprintf("%s/data", domainBasePath)

	var dataPath, metaFile, userFile, networkFile, iso, isoStaging string
	switch data.DataSource {
	case DataSourceNoCloud:
		dataPath = dataBasePath
		metaFile = fmt.Sprintf("%s/%s", dataPath, "meta-data")
		userFile = fmt.Sprintf("%s/%s", dataPath, "user-data")
		networkFile = fmt.Sprintf("%s/%s", dataPath, "network-config")
		iso = GetIsoFilePath(DataSourceNoCloud, vmiName, namespace)
		isoStaging = fmt.Sprintf("%s.staging", iso)
	case DataSourceConfigDrive:
		dataPath = fmt.Sprintf("%s/openstack/latest", dataBasePath)
		metaFile = fmt.Sprintf("%s/%s", dataPath, "meta_data.json")
		userFile = fmt.Sprintf("%s/%s", dataPath, "user_data")
		networkFile = fmt.Sprintf("%s/%s", dataPath, "network_data.json")
		iso = GetIsoFilePath(DataSourceConfigDrive, vmiName, namespace)
		isoStaging = fmt.Sprintf("%s.staging", iso)
	default:
		return fmt.Errorf("Invalid cloud-init data source: '%v'", data.DataSource)
	}

	err := os.MkdirAll(dataPath, 0755)
	if err != nil {
		log.Log.V(2).Reason(err).Errorf("unable to create cloud-init base path %s", domainBasePath)
		return err
	}

	if data.UserData == "" {
		return fmt.Errorf("UserData is required for cloud-init data source")
	}
	userData := []byte(data.UserData)

	var networkData []byte
	if data.NetworkData != "" {
		networkData = []byte(data.NetworkData)
	}

	var metaData []byte
	if data.MetaData == nil {
		log.Log.V(2).Infof("No metadata found in cloud-init data. Create minimal metadata with instance-id.")
		data.MetaData = &Metadata{
			InstanceID: fmt.Sprintf("%s.%s", vmiName, namespace),
		}
	}
	data.MetaData.Devices = data.DevicesData
	metaData, err = json.Marshal(data.MetaData)
	if err != nil {
		return err
	}

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

	switch data.DataSource {
	case DataSourceNoCloud:
		err = cloudInitIsoFunc(isoStaging, "cidata", dataBasePath)
	case DataSourceConfigDrive:
		err = cloudInitIsoFunc(isoStaging, "config-2", dataBasePath)
	}
	if err != nil {
		return err
	}
	diskutils.RemoveFile(metaFile)
	diskutils.RemoveFile(userFile)
	diskutils.RemoveFile(networkFile)

	if err := diskutils.DefaultOwnershipManager.SetFileOwnership(isoStaging); err != nil {
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
func listVmWithLocalData() ([]*v1.VirtualMachineInstance, error) {
	return diskutils.ListVmWithEphemeralDisk(cloudInitLocalDir)
}
