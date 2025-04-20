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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package device_manager

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/uuid"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

type DeviceHandler interface {
	GetDeviceIOMMUGroup(basepath string, pciAddress string) (string, error)
	GetDeviceDriver(basepath string, pciAddress string) (string, error)
	GetDeviceNumaNode(basepath string, pciAddress string) (numaNode int)
	GetDevicePCIID(basepath string, pciAddress string) (string, error)
	GetMdevParentPCIAddr(mdevUUID string) (string, error)
	CreateMDEVType(mdevType string, parentID string) error
	RemoveMDEVType(mdevUUID string) error
	ReadMDEVAvailableInstances(mdevType string, parentID string) (int, error)
}

type DeviceUtilsHandler struct{}

var handler DeviceHandler = &DeviceUtilsHandler{}

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

func (h *DeviceUtilsHandler) CreateMDEVType(mdevType string, parentID string) error {
	uid := uuid.NewUUID()
	path := filepath.Join(mdevClassBusPath, parentID, "mdev_supported_types", mdevType, "create")
	_, err := virt_chroot.CreateMDEVType(mdevType, parentID, string(uid)).Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if len(e.Stderr) > 0 {
				msg := fmt.Sprintf("failed to create mdev type %s, err: %v", mdevType, string(e.Stderr))
				errMsg := fmt.Errorf(msg)
				log.Log.Reason(err).Errorf(msg)
				return errMsg
			}
		}
		log.Log.Reason(err).Errorf("failed to create mdev type %s, can't open path %s", mdevType, path)
		return err
	}
	log.Log.Infof("Successfully created mdev %s - %s", mdevType, uid)
	return nil
}

func (h *DeviceUtilsHandler) RemoveMDEVType(mdevUUID string) error {
	removePath := filepath.Join(mdevBasePath, mdevUUID, "remove")
	_, err := virt_chroot.RemoveMDEVType(mdevUUID).Output()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if len(e.Stderr) > 0 {
				msg := fmt.Sprintf("failed to remove mdev %s, can't write to %s, err: %v", mdevUUID, removePath, string(e.Stderr))
				errMsg := fmt.Errorf(msg)
				log.Log.Reason(err).Errorf(msg)
				return errMsg
			}
		}
		log.Log.Reason(err).Errorf("failed to remove mdev %s, can't write to %s", mdevUUID, removePath)
		return err
	}
	log.Log.Infof("Successfully removed mdev %s", mdevUUID)
	return nil
}

func (h *DeviceUtilsHandler) ReadMDEVAvailableInstances(mdevType string, parentID string) (int, error) {
	var lines []string
	path := filepath.Join(mdevClassBusPath, parentID, "mdev_supported_types", mdevType, "available_instances")
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	if err != nil {
		return 0, err
	}

	i, err := strconv.Atoi(string(lines[0]))
	if err != nil {
		return 0, err
	}

	return i, nil
}

func waitForGRPCServer(socketPath string, timeout time.Duration) error {
	conn, err := gRPCConnect(socketPath, timeout)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// dial establishes the gRPC communication with the registered device plugin.
func gRPCConnect(socketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(socketPath,
		grpc.WithInsecure(),
		grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func SocketPath(deviceName string) string {
	return filepath.Join(v1beta1.DevicePluginPath, fmt.Sprintf("kubevirt-%s.sock", deviceName))
}

func IsChanClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
	}

	return false
}

func formatVFIODeviceSpecs(devID string) []*v1beta1.DeviceSpec {
	// always add /dev/vfio/vfio device as well
	devSpecs := make([]*v1beta1.DeviceSpec, 0)
	devSpecs = append(devSpecs, &v1beta1.DeviceSpec{
		HostPath:      vfioMount,
		ContainerPath: vfioMount,
		Permissions:   "mrw",
	})

	vfioDevice := filepath.Join(vfioDevicePath, devID)
	devSpecs = append(devSpecs, &v1beta1.DeviceSpec{
		HostPath:      vfioDevice,
		ContainerPath: vfioDevice,
		Permissions:   "mrw",
	})
	return devSpecs
}

type deviceHealth struct {
	DevId  string
	Health string
}
