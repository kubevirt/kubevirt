// SPDX-License-Identifier: Apache-2.0
// Copyright(c) 2018-2020 Red Hat, Inc, Intel Corp.

//
// This module provides the library functions to read and write
// annotations by call calling KubeAPI. The annotations are used
// to pass data between the host and a container.
//

package annotations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/go-logfmt/logfmt"

	"github.com/intel/userspace-cni-network-plugin/logging"
	"github.com/intel/userspace-cni-network-plugin/pkg/k8sclient"
	"github.com/intel/userspace-cni-network-plugin/pkg/types"
)

// Annotation
// These structures are used to document the set of annotations used in
// the Userspace CNI pod spec to pass data from Admission Controller to
// the CNI and from the CNI to the Container.

// List of Annotations supported in the podSpec
const (
	annotKeyNetwork         = "k8s.v1.cni.cncf.io/networks"
	annotKeyNetworkStatus   = "k8s.v1.cni.cncf.io/networks-status"
	AnnotKeyUsrspConfigData = "userspace/configuration-data"
	AnnotKeyUsrspMappedDir  = "userspace/mapped-dir"
	volMntKeySharedDir      = "shared-dir"

	DefaultBaseCNIDir  = "/var/lib/cni/usrspcni"
	DefaultLocalCNIDir = "/var/lib/cni/usrspcni/data"

	DefaultHostkubeletPodBaseDir  = "/var/lib/kubelet/pods/"
	DefaultHostEmptyDirVolumeName = "volumes/kubernetes.io~empty-dir/"
)

// Errors returned from this module
type NoSharedDirProvidedError struct {
	message string
}

func (e *NoSharedDirProvidedError) Error() string { return string(e.message) }

type NoKubeClientProvidedError struct {
	message string
}

func (e *NoKubeClientProvidedError) Error() string { return string(e.message) }

type NoPodProvidedError struct {
	message string
}

func (e *NoPodProvidedError) Error() string { return string(e.message) }

func GetPodVolumeMountHostSharedDir(pod *v1.Pod) (string, error) {
	var hostSharedDir string

	if pod == nil {
		return hostSharedDir, &NoPodProvidedError{"Error: Pod not provided."}
	}

	logging.Verbosef("GetPodVolumeMountSharedDir: type=%T Volumes=%v", pod.Spec.Volumes, pod.Spec.Volumes)

	if len(pod.Spec.Volumes) == 0 {
		return hostSharedDir, &NoSharedDirProvidedError{"Error: No Volumes. Need \"shared-dir\" in podSpec \"Volumes\""}
	}

	for _, volumeMount := range pod.Spec.Volumes {
		if volumeMount.Name == volMntKeySharedDir {
			if volumeMount.HostPath != nil {
				hostSharedDir = volumeMount.HostPath.Path
			} else if volumeMount.EmptyDir != nil {
				hostSharedDir = DefaultHostkubeletPodBaseDir + string(pod.UID) + "/" + DefaultHostEmptyDirVolumeName + volMntKeySharedDir
			} else {
				return hostSharedDir, &NoSharedDirProvidedError{"Error: Volume is invalid"}
			}
			break
		}
	}

	if len(hostSharedDir) == 0 {
		return hostSharedDir, &NoSharedDirProvidedError{"Error: No shared-dir. Need \"shared-dir\" in podSpec \"Volumes\""}
	}

	return hostSharedDir, nil
}

func getPodVolumeMountHostMappedSharedDir(pod *v1.Pod) (string, error) {
	var mappedSharedDir string

	if pod == nil {
		return mappedSharedDir, &NoPodProvidedError{"Error: Pod not provided."}
	}

	logging.Verbosef("getPodVolumeMountHostMappedSharedDir: Containers=%v", pod.Spec.Containers)

	if len(pod.Spec.Containers) == 0 {
		return mappedSharedDir, &NoSharedDirProvidedError{"Error: No Containers. Need \"shared-dir\" in podSpec \"Volumes\""}
	}

	for _, container := range pod.Spec.Containers {
		if len(container.VolumeMounts) != 0 {
			for _, volumeMount := range container.VolumeMounts {
				if volumeMount.Name == volMntKeySharedDir {
					mappedSharedDir = volumeMount.MountPath
					break
				}
			}
		}
	}

	if len(mappedSharedDir) == 0 {
		return mappedSharedDir, &NoSharedDirProvidedError{"Error: No mapped shared-dir. Need \"shared-dir\" in podSpec \"Volumes\""}
	}

	return mappedSharedDir, nil
}

func WritePodAnnotation(kubeClient kubernetes.Interface,
	pod *v1.Pod,
	configData *types.ConfigurationData) (*v1.Pod, error) {
	var err error
	var modifiedConfig bool
	var modifiedMappedDir bool

	if pod == nil {
		return pod, &NoPodProvidedError{"Error: Pod not provided."}
	}

	//
	// Write configuration data that will be consumed by container
	//
	if kubeClient != nil {
		//
		// Write configuration data into annotation
		//
		logging.Debugf("SaveRemoteConfig(): Store in PodSpec")

		modifiedConfig, err = setPodAnnotationConfigData(pod, configData)
		if err != nil {
			logging.Errorf("SaveRemoteConfig: Error formatting annotation configData: %v", err)
			return pod, err
		}

		// Retrieve the mappedSharedDir from the Containers in podSpec. Directory
		// in container Socket Files will be read from. Write this data back as an
		// annotation so container knows where directory is located.
		mappedSharedDir, err := getPodVolumeMountHostMappedSharedDir(pod)
		if err != nil {
			mappedSharedDir = DefaultBaseCNIDir
			logging.Warningf("SaveRemoteConfig: Error reading VolumeMount: %v", err)
			logging.Warningf("SaveRemoteConfig: VolumeMount \"shared-dir\" not provided, defaulting to: %s", mappedSharedDir)
			err = nil
		}
		modifiedMappedDir, err = setPodAnnotationMappedDir(pod, mappedSharedDir)
		if err != nil {
			logging.Errorf("SaveRemoteConfig: Error formatting annotation mappedSharedDir - %v", err)
			return pod, err
		}

		if modifiedConfig == true || modifiedMappedDir == true {
			pod, err = commitAnnotation(kubeClient, pod)
			if err != nil {
				logging.Errorf("SaveRemoteConfig: Error writing annotations - %v", err)
				return pod, err
			}
		}
	} else {
		return pod, &NoKubeClientProvidedError{"Error: KubeClient not provided."}
	}

	return pod, err
}

//
// Local Utility Functions
//
func setPodAnnotationMappedDir(pod *v1.Pod,
	mappedDir string) (bool, error) {
	var modified bool

	if pod == nil {
		return false, &NoPodProvidedError{"Error: Pod not provided."}
	}

	logging.Verbosef("SetPodAnnotationMappedDir: inputMappedDir=%s Annot - type=%T mappedDir=%v", mappedDir, pod.Annotations[AnnotKeyUsrspMappedDir], pod.Annotations[AnnotKeyUsrspMappedDir])

	// If pod annotations is empty, make sure it allocatable
	if len(pod.Annotations) == 0 {
		pod.Annotations = make(map[string]string)
	}

	// Get current data, if any. The current data is a string containing the
	// directory in the container to find shared files. If the data already exists,
	// it should be the same as the input data.
	annotDataStr := pod.Annotations[AnnotKeyUsrspMappedDir]
	if len(annotDataStr) != 0 {
		if filepath.Clean(annotDataStr) == filepath.Clean(mappedDir) {
			logging.Verbosef("SetPodAnnotationMappedDir: Existing matches input. Do nothing.")
			return modified, nil
		} else {
			return modified, logging.Errorf("SetPodAnnotationMappedDir: Input \"%s\" does not match existing \"%s\"", mappedDir, annotDataStr)
		}
	}

	// Append the just converted JSON string to any existing strings and
	// store in the annotation in the pod.
	pod.Annotations[AnnotKeyUsrspMappedDir] = mappedDir
	modified = true

	return modified, nil
}

func setPodAnnotationConfigData(pod *v1.Pod,
	configData *types.ConfigurationData) (bool, error) {
	var configDataStr []string
	var modified bool

	if pod == nil {
		return false, &NoPodProvidedError{"Error: Pod not provided."}
	}

	// check for empty configData, otherwise "null" string would be added to annotations
	if configData == nil {
		logging.Verbosef("SetPodAnnotationConfigData: ConfigData not provided: %v", configData)
		return false, nil
	}

	logging.Verbosef("SetPodAnnotationConfigData: type=%T configData=%v", pod.Annotations[AnnotKeyUsrspConfigData], pod.Annotations[AnnotKeyUsrspConfigData])

	// If pod annotations is empty, make sure it allocatable
	if len(pod.Annotations) == 0 {
		pod.Annotations = make(map[string]string)
	}

	// Get current data, if any. The current data is a string in JSON format with
	// data for multiple interfaces appended together. A given container can have
	// multiple interfaces, added one at a time. So existing data may be empty if
	// this is the first interface, or already contain data.
	annotDataStr := pod.Annotations[AnnotKeyUsrspConfigData]
	if len(annotDataStr) != 0 {
		// Strip wrapping [], will be added back around entire field.
		annotDataStr = strings.TrimLeft(annotDataStr, "[")
		annotDataStr = strings.TrimRight(annotDataStr, "]")

		// Add current string to slice of strings.
		configDataStr = append(configDataStr, annotDataStr)
	}

	// Marshal the input config data struct into a JSON string.
	data, err := json.MarshalIndent(configData, "", "    ")
	if err != nil {
		return modified, logging.Errorf("SetPodAnnotationConfigData: error with Marshal Indent: %v", err)
	}
	configDataStr = append(configDataStr, string(data))

	// Append the just converted JSON string to any existing strings and
	// store in the annotation in the pod.
	pod.Annotations[AnnotKeyUsrspConfigData] = fmt.Sprintf("[%s]", strings.Join(configDataStr, ","))
	modified = true

	return modified, err
}

func commitAnnotation(kubeClient kubernetes.Interface,
	pod *v1.Pod) (*v1.Pod, error) {
	// Write the modified data back to the pod.
	return k8sclient.WritePodAnnotation(kubeClient, pod)
}

//
// Container Access Functions
// These functions can be called from code running in a container. It reads
// the data from the exposed Downward API.
//
func getFileAnnotation(annotFile string, annotIndex string) ([]byte, error) {
	var rawData []byte

	fileData, err := ioutil.ReadFile(annotFile)
	if err != nil {
		logging.Errorf("getFileAnnotation: File Read ERROR - %v", err)
		return rawData, fmt.Errorf("error reading %s: %s", annotFile, err)
	}

	d := logfmt.NewDecoder(bytes.NewReader(fileData))
	for d.ScanRecord() {
		for d.ScanKeyval() {
			//fmt.Printf("k: %T %s v: %T %s\n", d.Key(), d.Key(), d.Value(), d.Value())
			//logging.Debugf("  k: %T %s v: %T %s\n", d.Key(), d.Key(), d.Value(), d.Value())

			if string(d.Key()) == annotIndex {
				rawData = d.Value()
				return rawData, nil
			}
		}
		//fmt.Println()
	}

	return rawData, fmt.Errorf("ERROR: \"%s\" missing from pod annotation", annotIndex)
}

func GetFileAnnotationMappedDir(annotFile string) (string, error) {
	rawData, err := getFileAnnotation(annotFile, AnnotKeyUsrspMappedDir)
	if err != nil {
		return "", err
	}

	return string(rawData), err
}

func GetFileAnnotationConfigData(annotFile string) ([]*types.ConfigurationData, error) {
	var configDataList []*types.ConfigurationData

	// Remove
	logging.Debugf("GetFileAnnotationConfigData: ENTER")

	rawData, err := getFileAnnotation(annotFile, AnnotKeyUsrspConfigData)
	if err != nil {
		return nil, err
	}

	rawString := string(rawData)
	if strings.IndexAny(rawString, "[{\"") >= 0 {
		if err := json.Unmarshal([]byte(rawString), &configDataList); err != nil {
			return nil, logging.Errorf("GetFileAnnotationConfigData: Failed to parse ConfigData Annotation JSON format: %v", err)
		}
	} else {
		return nil, logging.Errorf("GetFileAnnotationConfigData: Invalid formatted JSON data")
	}

	return configDataList, err
}

//func GetFileAnnotationNetworksStatus() ([]*multusTypes.NetworkStatus, error) {
//	var networkStatusList []*multusTypes.NetworkStatus
//
//	// Remove
//	logging.Debugf("GetFileAnnotationNetworksStatus: ENTER")
//
//	rawData, err := getFileAnnotation(annotKeyNetworkStatus)
//	if err != nil {
//		return nil, err
//	}
//
//	rawString := string(rawData)
//	if strings.IndexAny(rawString, "[{\"") >= 0 {
//		if err := json.Unmarshal([]byte(rawString), &networkStatusList); err != nil {
//			return nil, logging.Errorf("GetFileAnnotationNetworksStatus: Failed to parse networkStatusList Annotation JSON format: %v", err)
//		}
//	} else {
//		return nil, logging.Errorf("GetFileAnnotationNetworksStatus: Invalid formatted JSON data")
//	}
//
//	return networkStatusList, err
//}
