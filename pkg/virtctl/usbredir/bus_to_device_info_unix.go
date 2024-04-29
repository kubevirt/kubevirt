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
 * Copyright the KubeVirt Authors.
 *
 */
package usbredir

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	pathToUSBDevices = "/sys/bus/usb/devices"
)

func busToDevicePlatform(bus, devnum string) (string, string, error) {
	var vendor, product string

	err := filepath.Walk(pathToUSBDevices, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Ignore named usb controllers
		if strings.HasPrefix(info.Name(), "usb") {
			return nil
		}

		// We are interested in actual USB devices information that
		// contains busnum and devnum. We can skip all others.
		if _, err := os.Stat(filepath.Join(path, "busnum")); err != nil {
			return nil
		}

		if vendor, product, err = getDeviceInfo(path, bus, devnum); err == nil {
			return filepath.SkipAll
		}

		return nil
	})

	if err == nil && vendor != "" && product != "" {
		return vendor, product, nil
	}

	return "", "", fmt.Errorf("Failed to findo vendor/product of bus=%s,devnum=%s", bus, devnum)
}

func getDeviceInfo(path, bus, devnum string) (string, string, error) {
	var busInt, devnumInt int64
	var fileBus, fileDevnum int64
	var err error

	if busInt, err = strconv.ParseInt(bus, 10, 32); err != nil {
		return "", "", err
	}
	if devnumInt, err = strconv.ParseInt(devnum, 10, 32); err != nil {
		return "", "", err
	}

	if buffer, err := os.ReadFile(filepath.Join(path, "busnum")); err != nil {
		return "", "", err
	} else if fileBus, err = strconv.ParseInt(strings.TrimSpace(string(buffer)), 10, 32); err != nil {
		return "", "", err
	} else if fileBus != busInt {
		return "", "", fmt.Errorf("Input bus %d and busnum %d do not match", busInt, fileBus)
	}

	if buffer, err := os.ReadFile(filepath.Join(path, "devnum")); err != nil {
		return "", "", err
	} else if fileDevnum, err = strconv.ParseInt(strings.TrimSpace(string(buffer)), 10, 32); err != nil {
		return "", "", err
	} else if fileDevnum != devnumInt {
		return "", "", fmt.Errorf("Input devnum %d and busnum %d do not match", devnumInt, fileDevnum)
	}

	// Matches! Just need to fetch Vendor and Product information now.
	var vendor, product []byte
	if vendor, err = os.ReadFile(filepath.Join(path, "idVendor")); err != nil {
		return "", "", err
	}
	if product, err = os.ReadFile(filepath.Join(path, "idProduct")); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(string(vendor)), strings.TrimSpace(string(product)), nil
}
