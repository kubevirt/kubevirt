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
 * Copyright The KubeVirt Authors.
 *
 */

package libstorage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/errorhandling"
	"kubevirt.io/kubevirt/tests/flags"
)

// KubeVirtTestsConfiguration contains the configuration for KubeVirt tests
type KubeVirtTestsConfiguration struct {
	// StorageClass to use to create rhel PVCs
	StorageClassRhel string `json:"storageClassRhel"`
	// StorageClass to use to create windows PVCs
	StorageClassWindows string `json:"storageClassWindows"`
	// StorageClass supporting RWX Filesystem
	StorageRWXFileSystem string `json:"storageRWXFileSystem"`
	// StorageClass supporting RWX Block
	StorageRWXBlock string `json:"storageRWXBlock"`
	// StorageClass supporting RWO Filesystem
	StorageRWOFileSystem string `json:"storageRWOFileSystem"`
	// StorageClass supporting RWO Block
	StorageRWOBlock string `json:"storageRWOBlock"`
	// StorageClass supporting snapshot
	StorageSnapshot string `json:"storageSnapshot"`
	// StorageVMState is the storage class for backend PVCs (TPM/EFI)
	StorageVMState string `json:"storageVMState"`
	// StorageClass supporting CSI
	StorageClassCSI string `json:"storageClassCSI"`
}

const kubevirtIoTest = "kubevirt.io/test"

var Config *KubeVirtTestsConfiguration

func LoadConfig() (*KubeVirtTestsConfiguration, error) {
	// open configuration file
	jsonFile, err := os.Open(flags.ConfigFile)
	if err != nil {
		return nil, err
	}

	defer errorhandling.SafelyCloseFile(jsonFile)

	// read the configuration file as a byte array
	byteValue, _ := io.ReadAll(jsonFile)

	// convert the byte array to a KubeVirtTestsConfiguration struct
	config := &KubeVirtTestsConfiguration{}
	err = json.Unmarshal(byteValue, config)

	return config, err
}

// DiscoverStorageCapabilitiesFromSC queries the CDI StorageProfile for the given storage class
// and populates the test configuration based on its capabilities.
func DiscoverStorageCapabilitiesFromSC(virtClient kubecli.KubevirtClient, storageClassName string) error {
	if storageClassName == "" {
		return fmt.Errorf("storage class name cannot be empty")
	}

	fmt.Printf("Auto-discovering storage capabilities from StorageClass: %s\n", storageClassName)

	sp, err := virtClient.CdiClient().CdiV1beta1().StorageProfiles().Get(
		context.Background(), storageClassName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get StorageProfile for %s: %w", storageClassName, err)
	}

	// Parse capabilities from StorageProfile
	var rwxFilesystem, rwxBlock, rwoFilesystem, rwoBlock bool
	for _, propSet := range sp.Status.ClaimPropertySets {
		if propSet.VolumeMode == nil {
			continue
		}
		for _, accessMode := range propSet.AccessModes {
			switch {
			case *propSet.VolumeMode == k8sv1.PersistentVolumeFilesystem && accessMode == k8sv1.ReadWriteOnce:
				rwoFilesystem = true
			case *propSet.VolumeMode == k8sv1.PersistentVolumeFilesystem && accessMode == k8sv1.ReadWriteMany:
				rwxFilesystem = true
			case *propSet.VolumeMode == k8sv1.PersistentVolumeBlock && accessMode == k8sv1.ReadWriteOnce:
				rwoBlock = true
			case *propSet.VolumeMode == k8sv1.PersistentVolumeBlock && accessMode == k8sv1.ReadWriteMany:
				rwxBlock = true
			}
		}
	}
	hasSnapshot := sp.Status.SnapshotClass != nil && *sp.Status.SnapshotClass != ""
	hasCSI := sp.Status.Provisioner != nil && *sp.Status.Provisioner != ""

	// Clear existing config and apply only what this storage class supports
	*Config = KubeVirtTestsConfiguration{}

	if rwoFilesystem {
		Config.StorageRWOFileSystem = storageClassName
		Config.StorageVMState = storageClassName
	}
	if rwoBlock {
		Config.StorageRWOBlock = storageClassName
	}
	if rwxFilesystem {
		Config.StorageRWXFileSystem = storageClassName
	}
	if rwxBlock {
		Config.StorageRWXBlock = storageClassName
	}
	if hasSnapshot {
		Config.StorageSnapshot = storageClassName
	}
	if hasCSI {
		Config.StorageClassCSI = storageClassName
	}
	if rwoBlock || rwoFilesystem {
		Config.StorageClassRhel = storageClassName
		Config.StorageClassWindows = storageClassName
	}

	// Print discovered configuration
	fmt.Println("Discovered storage configuration:")
	fmt.Printf("  StorageRWOFileSystem: %s\n", valueOrNotAvailable(Config.StorageRWOFileSystem))
	fmt.Printf("  StorageRWOBlock: %s\n", valueOrNotAvailable(Config.StorageRWOBlock))
	fmt.Printf("  StorageRWXFileSystem: %s\n", valueOrNotAvailable(Config.StorageRWXFileSystem))
	fmt.Printf("  StorageRWXBlock: %s\n", valueOrNotAvailable(Config.StorageRWXBlock))
	fmt.Printf("  StorageSnapshot: %s\n", valueOrNotAvailable(Config.StorageSnapshot))
	fmt.Printf("  StorageClassCSI: %s\n", valueOrNotAvailable(Config.StorageClassCSI))
	fmt.Printf("  StorageVMState: %s\n", valueOrNotAvailable(Config.StorageVMState))
	fmt.Printf("  StorageClassRhel: %s\n", valueOrNotAvailable(Config.StorageClassRhel))
	fmt.Printf("  StorageClassWindows: %s\n", valueOrNotAvailable(Config.StorageClassWindows))

	return nil
}

func valueOrNotAvailable(s string) string {
	if s == "" {
		return "(not available)"
	}
	return s
}
