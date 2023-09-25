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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pborman/uuid"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
)

const isoStagingFmt = "%s.staging"

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
	HostDevMetadataType   DeviceMetadataType = "hostdev"
)

// CloudInitData is a data source independent struct that
// holds cloud-init user and network data
type CloudInitData struct {
	DataSource          DataSourceType
	NoCloudMetaData     *NoCloudMetadata
	ConfigDriveMetaData *ConfigDriveMetadata
	UserData            string
	NetworkData         string
	DevicesData         *[]DeviceData
	VolumeName          string
}

type PublicSSHKey struct {
	string
}

type NoCloudMetadata struct {
	InstanceType  string            `json:"instance-type,omitempty"`
	InstanceID    string            `json:"instance-id"`
	LocalHostname string            `json:"local-hostname,omitempty"`
	PublicSSHKeys map[string]string `json:"public-keys,omitempty"`
}

type ConfigDriveMetadata struct {
	InstanceType  string            `json:"instance_type,omitempty"`
	InstanceID    string            `json:"instance_id"`
	LocalHostname string            `json:"local_hostname,omitempty"`
	Hostname      string            `json:"hostname,omitempty"`
	UUID          string            `json:"uuid,omitempty"`
	Devices       *[]DeviceData     `json:"devices,omitempty"`
	PublicSSHKeys map[string]string `json:"public_keys,omitempty"`
}

type DeviceData struct {
	Type        DeviceMetadataType `json:"type"`
	Bus         string             `json:"bus"`
	Address     string             `json:"address"`
	MAC         string             `json:"mac,omitempty"`
	Serial      string             `json:"serial,omitempty"`
	NumaNode    uint32             `json:"numaNode,omitempty"`
	AlignedCPUs []uint32           `json:"alignedCPUs,omitempty"`
	Tags        []string           `json:"tags"`
}

// IsValidCloudInitData checks if the given CloudInitData object is valid in the sense that GenerateLocalData can be called with it.
func IsValidCloudInitData(cloudInitData *CloudInitData) bool {
	return cloudInitData != nil && cloudInitData.UserData != "" && (cloudInitData.NoCloudMetaData != nil || cloudInitData.ConfigDriveMetaData != nil)
}

func cloudInitUUIDFromVMI(vmi *v1.VirtualMachineInstance) string {
	if vmi.Spec.Domain.Firmware == nil {
		return uuid.NewRandom().String()
	}
	return string(vmi.Spec.Domain.Firmware.UUID)
}

// ReadCloudInitVolumeDataSource scans the given VMI for CloudInit volumes and
// reads their content into a CloudInitData struct. Does not resolve secret refs.
func ReadCloudInitVolumeDataSource(vmi *v1.VirtualMachineInstance, secretSourceDir string) (cloudInitData *CloudInitData, err error) {
	precond.MustNotBeNil(vmi)
	// ClusterInstancetypeAnnotation will take precedence over a namespaced Instancetype
	// for setting instance_type in the metadata
	instancetype := vmi.Annotations[v1.ClusterInstancetypeAnnotation]
	if instancetype == "" {
		instancetype = vmi.Annotations[v1.InstancetypeAnnotation]
	}

	hostname := dns.SanitizeHostname(vmi)

	for _, volume := range vmi.Spec.Volumes {
		if volume.CloudInitNoCloud != nil {
			keys, err := resolveNoCloudSecrets(vmi, secretSourceDir)
			if err != nil {
				return nil, err
			}

			cloudInitData, err = readCloudInitNoCloudSource(volume.CloudInitNoCloud)
			cloudInitData.NoCloudMetaData = readCloudInitNoCloudMetaData(hostname, cloudInitUUIDFromVMI(vmi), instancetype, keys)
			cloudInitData.VolumeName = volume.Name
			return cloudInitData, err
		}
		if volume.CloudInitConfigDrive != nil {

			keys, err := resolveConfigDriveSecrets(vmi, secretSourceDir)
			if err != nil {
				return nil, err
			}

			uuid := cloudInitUUIDFromVMI(vmi)
			cloudInitData, err = readCloudInitConfigDriveSource(volume.CloudInitConfigDrive)
			cloudInitData.ConfigDriveMetaData = readCloudInitConfigDriveMetaData(vmi.Name, uuid, hostname, vmi.Namespace, keys, instancetype)
			cloudInitData.VolumeName = volume.Name
			return cloudInitData, err
		}
	}
	return nil, nil
}

func isNoCloudAccessCredential(accessCred v1.AccessCredential) bool {
	return accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.PropagationMethod.NoCloud != nil
}

func isConfigDriveAccessCredential(accessCred v1.AccessCredential) bool {
	return accessCred.SSHPublicKey != nil && accessCred.SSHPublicKey.PropagationMethod.ConfigDrive != nil
}

func resolveSSHPublicKeys(accessCredentials []v1.AccessCredential, secretSourceDir string, isAccessCredentialValidFunc func(v1.AccessCredential) bool) (map[string]string, error) {
	keys := make(map[string]string)
	count := 0
	for _, accessCred := range accessCredentials {

		if !isAccessCredentialValidFunc(accessCred) {
			continue
		}

		secretName := ""
		if accessCred.SSHPublicKey.Source.Secret != nil {
			secretName = accessCred.SSHPublicKey.Source.Secret.SecretName
		}

		if secretName == "" {
			continue
		}

		baseDir := filepath.Join(secretSourceDir, secretName+"-access-cred")
		files, err := os.ReadDir(baseDir)
		if err != nil {
			return keys, err
		}

		for _, file := range files {
			if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
				continue
			}
			keyData, err := readFileFromDir(baseDir, file.Name())

			if err != nil {
				return keys, fmt.Errorf("Unable to read public keys found at volume: %s/%s error: %v", baseDir, file.Name(), err)
			}

			if keyData == "" {
				continue
			}
			keys[strconv.Itoa(count)] = keyData
			count++
		}
	}
	return keys, nil
}

// resolveNoCloudSecrets is looking for CloudInitNoCloud volumes with UserDataSecretRef
// requests. It reads the `userdata` secret the corresponds to the given CloudInitNoCloud
// volume and sets the UserData field on that volume.
//
// Note: when using this function, make sure that your code can access the secret volumes.
func resolveNoCloudSecrets(vmi *v1.VirtualMachineInstance, secretSourceDir string) (map[string]string, error) {
	keys, err := resolveSSHPublicKeys(vmi.Spec.AccessCredentials, secretSourceDir, isNoCloudAccessCredential)
	if err != nil {
		return keys, err
	}

	volume := findCloudInitNoCloudSecretVolume(vmi.Spec.Volumes)
	if volume == nil {
		return keys, nil
	}

	baseDir := filepath.Join(secretSourceDir, volume.Name)
	var userDataError, networkDataError error
	var userData, networkData string
	if volume.CloudInitNoCloud.UserDataSecretRef != nil {
		userData, userDataError = readFirstFoundFileFromDir(baseDir, []string{"userdata", "userData"})
	}
	if volume.CloudInitNoCloud.NetworkDataSecretRef != nil {
		networkData, networkDataError = readFirstFoundFileFromDir(baseDir, []string{"networkdata", "networkData"})
	}
	if userDataError != nil && networkDataError != nil {
		return keys, fmt.Errorf("no cloud-init data-source found at volume: %s", volume.Name)
	}

	if userData != "" {
		volume.CloudInitNoCloud.UserData = userData
	}
	if networkData != "" {
		volume.CloudInitNoCloud.NetworkData = networkData
	}

	return keys, nil
}

// resolveConfigDriveSecrets is looking for CloudInitConfigDriveSource volume source with
// UserDataSecretRef and NetworkDataSecretRef and resolves the secret from the corresponding
// VolumeMount.
//
// Note: when using this function, make sure that your code can access the secret volumes.
func resolveConfigDriveSecrets(vmi *v1.VirtualMachineInstance, secretSourceDir string) (map[string]string, error) {
	keys, err := resolveSSHPublicKeys(vmi.Spec.AccessCredentials, secretSourceDir, isConfigDriveAccessCredential)
	if err != nil {
		return keys, err
	}

	if err != nil {
		return keys, err
	}

	volume := findCloudInitConfigDriveSecretVolume(vmi.Spec.Volumes)
	if volume == nil {
		return keys, nil
	}

	baseDir := filepath.Join(secretSourceDir, volume.Name)
	var userDataError, networkDataError error
	var userData, networkData string
	if volume.CloudInitConfigDrive.UserDataSecretRef != nil {
		userData, userDataError = readFirstFoundFileFromDir(baseDir, []string{"userdata", "userData"})
	}
	if volume.CloudInitConfigDrive.NetworkDataSecretRef != nil {
		networkData, networkDataError = readFirstFoundFileFromDir(baseDir, []string{"networkdata", "networkData"})
	}
	if userDataError != nil && networkDataError != nil {
		return keys, fmt.Errorf("no cloud-init data-source found at volume: %s", volume.Name)
	}
	if userData != "" {
		volume.CloudInitConfigDrive.UserData = userData
	}
	if networkData != "" {
		volume.CloudInitConfigDrive.NetworkData = networkData
	}

	return keys, nil
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

func readFirstFoundFileFromDir(basedir string, files []string) (string, error) {
	var err error
	var data string
	for _, file := range files {
		data, err = readFileFromDir(basedir, file)
		if err == nil {
			break
		}
	}
	return data, err
}
func readFileFromDir(basedir, file string) (string, error) {
	filePath := filepath.Join(basedir, file)
	// #nosec No risk for path injection: basedir & secretFile are static strings
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Log.Reason(err).Errorf("could not read data from source: %s", filePath)
		return "", err
	}
	return string(data), nil
}

// findCloudInitNoCloudSecretVolume loops over a given list of volumes and return a pointer
// to the first CloudInitNoCloud volume with a UserDataSecretRef field set.
func findCloudInitNoCloudSecretVolume(volumes []v1.Volume) *v1.Volume {
	for _, volume := range volumes {
		if volume.CloudInitNoCloud == nil {
			continue
		}
		if volume.CloudInitNoCloud.UserDataSecretRef != nil ||
			volume.CloudInitNoCloud.NetworkDataSecretRef != nil {
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

	readNetworkData, err := readRawOrBase64Data(networkData, networkDataBase64)
	if err != nil {
		return "", "", err
	}

	if readUserData == "" && readNetworkData == "" {
		return "", "", fmt.Errorf("userDataBase64, userData, networkDataBase64 or networkData is required for a cloud-init data source")
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

func readCloudInitNoCloudMetaData(hostname, instanceId string, instanceType string, keys map[string]string) *NoCloudMetadata {
	return &NoCloudMetadata{
		InstanceType:  instanceType,
		InstanceID:    instanceId,
		LocalHostname: hostname,
		PublicSSHKeys: keys,
	}
}

func readCloudInitConfigDriveMetaData(name, uuid, hostname, namespace string, keys map[string]string, instanceType string) *ConfigDriveMetadata {
	return &ConfigDriveMetadata{
		InstanceType:  instanceType,
		UUID:          uuid,
		InstanceID:    fmt.Sprintf("%s.%s", name, namespace),
		Hostname:      hostname,
		PublicSSHKeys: keys,
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
	args = append(args, "-partition_cyl_align")
	args = append(args, "on")
	args = append(args, inDir)

	isoBinary := "xorrisofs"

	// #nosec No risk for attacket injection. Parameters are predefined strings
	cmd := exec.Command(isoBinary, args...)

	err := cmd.Start()
	if err != nil {
		log.Log.Reason(err).Errorf("%s cmd failed to start while generating iso file %s", isoBinary, isoOutFile)
		return err
	}

	done := make(chan error)
	go func() { done <- cmd.Wait() }()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-timeout:
			log.Log.Errorf("Timed out generating cloud-init iso at path %s", isoOutFile)
			cmd.Process.Kill()
		case err := <-done:
			if err != nil {
				log.Log.Reason(err).Errorf("%s returned non-zero exit code while generating iso file %s with args '%s'", isoBinary, isoOutFile, strings.Join(cmd.Args, " "))
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
	err := util.MkdirAllWithNosec(dir)
	if err != nil {
		return fmt.Errorf("unable to initialize cloudInit local cache directory (%s). %v", dir, err)
	}

	exists, err := diskutils.FileExists(dir)
	if err != nil {
		return fmt.Errorf("CloudInit local cache directory (%s) does not exist or is inaccessible. %v", dir, err)
	} else if exists == false {
		return fmt.Errorf("CloudInit local cache directory (%s) does not exist or is inaccessible", dir)
	}

	SetLocalDirectoryOnly(dir)
	return nil
}

// XXX refactor this whole package
// This is just a cheap workaround to make e2e tests pass
func SetLocalDirectoryOnly(dir string) {
	cloudInitLocalDir = dir
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

func PrepareLocalPath(vmiName string, namespace string) error {
	return util.MkdirAllWithNosec(getDomainBasePath(vmiName, namespace))
}

func GenerateEmptyIso(vmiName string, namespace string, data *CloudInitData, size int64) error {
	precond.MustNotBeEmpty(vmiName)
	precond.MustNotBeNil(data)

	var err error
	var isoStaging, iso string

	switch data.DataSource {
	case DataSourceNoCloud, DataSourceConfigDrive:
		iso = GetIsoFilePath(data.DataSource, vmiName, namespace)
	default:
		return fmt.Errorf("invalid cloud-init data source: '%v'", data.DataSource)
	}
	isoStaging = fmt.Sprintf(isoStagingFmt, iso)

	err = diskutils.RemoveFilesIfExist(isoStaging)
	if err != nil {
		return err
	}

	err = util.MkdirAllWithNosec(path.Dir(isoStaging))
	if err != nil {
		log.Log.Reason(err).Errorf("unable to create cloud-init base path %s", path.Dir(isoStaging))
		return err
	}

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

	if err := diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(isoStaging); err != nil {
		return err
	}
	err = os.Rename(isoStaging, iso)
	if err != nil {
		log.Log.Reason(err).Errorf("Cloud-init failed to rename file %s to %s", isoStaging, iso)
		return err
	}

	log.Log.V(2).Infof("generated empty iso file %s", iso)
	return nil
}

func GenerateLocalData(vmi *v1.VirtualMachineInstance, instanceType string, data *CloudInitData) error {
	precond.MustNotBeEmpty(vmi.Name)
	precond.MustNotBeNil(data)

	var metaData []byte
	var err error

	domainBasePath := getDomainBasePath(vmi.Name, vmi.Namespace)
	dataBasePath := fmt.Sprintf("%s/data", domainBasePath)

	var dataPath, metaFile, userFile, networkFile, iso, isoStaging string
	switch data.DataSource {
	case DataSourceNoCloud:
		dataPath = dataBasePath
		metaFile = fmt.Sprintf("%s/%s", dataPath, "meta-data")
		userFile = fmt.Sprintf("%s/%s", dataPath, "user-data")
		networkFile = fmt.Sprintf("%s/%s", dataPath, "network-config")
		iso = GetIsoFilePath(DataSourceNoCloud, vmi.Name, vmi.Namespace)
		isoStaging = fmt.Sprintf(isoStagingFmt, iso)
		if data.NoCloudMetaData == nil {
			log.Log.V(2).Infof("No metadata found in cloud-init data. Create minimal metadata with instance-id.")
			data.NoCloudMetaData = &NoCloudMetadata{
				InstanceID: cloudInitUUIDFromVMI(vmi),
			}
			data.NoCloudMetaData.InstanceType = instanceType
		}
		metaData, err = json.Marshal(data.NoCloudMetaData)
		if err != nil {
			return err
		}
	case DataSourceConfigDrive:
		dataPath = fmt.Sprintf("%s/openstack/latest", dataBasePath)
		metaFile = fmt.Sprintf("%s/%s", dataPath, "meta_data.json")
		userFile = fmt.Sprintf("%s/%s", dataPath, "user_data")
		networkFile = fmt.Sprintf("%s/%s", dataPath, "network_data.json")
		iso = GetIsoFilePath(DataSourceConfigDrive, vmi.Name, vmi.Namespace)
		isoStaging = fmt.Sprintf(isoStagingFmt, iso)
		if data.ConfigDriveMetaData == nil {
			log.Log.V(2).Infof("No metadata found in cloud-init data. Create minimal metadata with instance-id.")
			instanceId := fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace)
			data.ConfigDriveMetaData = &ConfigDriveMetadata{
				InstanceID: instanceId,
				UUID:       cloudInitUUIDFromVMI(vmi),
			}
			data.ConfigDriveMetaData.InstanceType = instanceType
		}
		data.ConfigDriveMetaData.Devices = data.DevicesData
		metaData, err = json.Marshal(data.ConfigDriveMetaData)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("Invalid cloud-init data source: '%v'", data.DataSource)
	}

	err = util.MkdirAllWithNosec(dataPath)
	if err != nil {
		log.Log.Reason(err).Errorf("unable to create cloud-init base path %s", domainBasePath)
		return err
	}

	if data.UserData == "" && data.NetworkData == "" {
		return fmt.Errorf("UserData or NetworkData is required for cloud-init data source")
	}
	userData := []byte(data.UserData)

	var networkData []byte
	if data.NetworkData != "" {
		networkData = []byte(data.NetworkData)
	}

	err = diskutils.RemoveFilesIfExist(userFile, metaFile, networkFile, isoStaging)
	if err != nil {
		return err
	}

	err = os.WriteFile(userFile, userData, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(userFile)

	err = os.WriteFile(metaFile, metaData, 0600)
	if err != nil {
		return err
	}
	defer os.Remove(metaFile)

	if len(networkData) > 0 {
		err = os.WriteFile(networkFile, networkData, 0600)
		if err != nil {
			return err
		}
		defer os.Remove(networkFile)
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

	if err := diskutils.DefaultOwnershipManager.UnsafeSetFileOwnership(isoStaging); err != nil {
		return err
	}

	err = os.Rename(isoStaging, iso)
	if err != nil {
		log.Log.Reason(err).Errorf("Cloud-init failed to rename file %s to %s", isoStaging, iso)
		return err
	}

	log.Log.V(2).Infof("generated nocloud iso file %s", iso)
	return nil
}
