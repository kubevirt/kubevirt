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

package hardware

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	VFIOCDevBasePath = "/dev/vfio/devices"
)

// GetDeviceVFIOCDevName gets the name of the associated VFIO cdev for PCI and
// Mediated devices if have
// e.g. /sys/bus/pci/devices/0000\:65\:00.0/vfio-dev/vfio0 <-
func GetDeviceVFIOCDevName(devPath string) (string, error) {
	dirPath := filepath.Join(devPath, "vfio-dev")
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), "vfio") {
			return entry.Name(), nil
		}
	}
	return "", fmt.Errorf("no VFIO cdev found for device")
}

// VFIOCDevExists checks whether a specified VFIO cdev is present in the
// environment (e.g. a container)
func VFIOCDevExists(name string) (bool, error) {
	path := filepath.Join(VFIOCDevBasePath, name)
	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
