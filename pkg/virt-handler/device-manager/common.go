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
 * Copyright 2020 Red Hat, Inc.
 *
 */

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package device_manager

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"
)

type DeviceHandler interface {
	GetDeviceIOMMUGroup(basepath string, pciAddress string) (string, error)
	GetDeviceDriver(basepath string, pciAddress string) (string, error)
	GetDeviceNumaNode(basepath string, pciAddress string) (numaNode int)
	GetDevicePCIID(basepath string, pciAddress string) (string, error)
	GetMdevParentPCIAddr(mdevUUID string) (string, error)
}

type DeviceUtilsHandler struct{}

var Handler DeviceHandler

// getDeviceIOMMUGroup gets devices iommu_group
// e.g. /sys/bus/pci/devices/0000\:65\:00.0/iommu_group -> ../../../../../kernel/iommu_groups/45
func (h *DeviceUtilsHandler) GetDeviceIOMMUGroup(basepath string, pciAddress string) (string, error) {
	iommuLink := filepath.Join(basepath, pciAddress, "iommu_group")
	iommuPath, err := os.Readlink(iommuLink)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to read iommu_group link %s for device %s", iommuLink, pciAddress)
		return "", err
	}
	_, iommuGroup := filepath.Split(iommuPath)
	return iommuGroup, nil
}

// gets device driver
func (h *DeviceUtilsHandler) GetDeviceDriver(basepath string, pciAddress string) (string, error) {
	driverLink := filepath.Join(basepath, pciAddress, "driver")
	driverPath, err := os.Readlink(driverLink)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to read driver link %s for device %s", driverLink, pciAddress)
		return "", err
	}
	_, driver := filepath.Split(driverPath)
	return driver, nil
}

func (h *DeviceUtilsHandler) GetDeviceNumaNode(basepath string, pciAddress string) (numaNode int) {
	numaNode = -1
	numaNodePath := filepath.Join(basepath, pciAddress, "numa_node")
	// #nosec No risk for path injection. Reading static path of NUMA node info
	numaNodeStr, err := os.ReadFile(numaNodePath)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to read numa_node %s for device %s", numaNodePath, pciAddress)
		return
	}
	numaNodeStr = bytes.TrimSpace(numaNodeStr)
	numaNode, err = strconv.Atoi(string(numaNodeStr))
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to convert numa node value %v of device %s", numaNodeStr, pciAddress)
		return
	}
	return
}

func (h *DeviceUtilsHandler) GetDevicePCIID(basepath string, pciAddress string) (string, error) {
	// #nosec No risk for path injection. Reading static path of PCI data
	file, err := os.Open(filepath.Join(basepath, pciAddress, "uevent"))
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PCI_ID") {
			equal := strings.Index(line, "=")
			value := strings.TrimSpace(line[equal+1:])
			return strings.ToLower(value), nil
		}
	}
	return "", fmt.Errorf("no pci_id is found")
}

// /sys/class/mdev_bus/0000:00:03.0/53764d0e-85a0-42b4-af5c-2046b460b1dc
func (h *DeviceUtilsHandler) GetMdevParentPCIAddr(mdevUUID string) (string, error) {
	mdevLink, err := os.Readlink(filepath.Join(mdevBasePath, mdevUUID))
	if err != nil {
		return "", err
	}
	linkParts := strings.Split(mdevLink, "/")
	return linkParts[len(linkParts)-2], nil
}

func initHandler() {
	if Handler == nil {
		Handler = &DeviceUtilsHandler{}
	}
}
