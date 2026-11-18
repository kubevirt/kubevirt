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

package device_manager_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	devicemanager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("PCI device plugin VFIO cdev allocation", func() {
	var tmpDir string
	var vfioDevicesDir string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		vfioDevicesDir = filepath.Join(tmpDir, "dev-vfio-devices")
		Expect(os.MkdirAll(vfioDevicesDir, 0755)).To(Succeed())
		DeferCleanup(devicemanager.SetPCIBasePath(tmpDir))
		DeferCleanup(devicemanager.SetVFIODevicesRoot(vfioDevicesDir))
	})

	It("should include VFIO cdev device specs in Allocate response", func() {
		pciAddr := "0000:08:00.0"
		iommuGroup := "42"
		vfioDevDir := filepath.Join(tmpDir, pciAddr, "vfio-dev")
		Expect(os.MkdirAll(vfioDevDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(vfioDevDir, "vfio0"), []byte{}, 0644)).To(Succeed())
		// Create the actual device file in the mocked /dev/vfio/devices directory
		Expect(os.WriteFile(filepath.Join(vfioDevicesDir, "vfio0"), []byte{}, 0644)).To(Succeed())

		dpi := devicemanager.NewPCIDevicePluginForTest(map[string]string{iommuGroup: pciAddr})

		resp, err := dpi.Allocate(context.Background(), &pluginapi.AllocateRequest{
			ContainerRequests: []*pluginapi.ContainerAllocateRequest{
				{DevicesIDs: []string{iommuGroup}},
			},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.ContainerResponses).To(HaveLen(1))

		expectedPath := filepath.Join(vfioDevicesDir, "vfio0")
		Expect(resp.ContainerResponses[0].Devices).To(ContainElement(
			BeEquivalentTo(&pluginapi.DeviceSpec{
				HostPath:      expectedPath,
				ContainerPath: expectedPath,
				Permissions:   "mrw",
			})))
	})

	It("should not fail when no VFIO cdev devices exist", func() {
		iommuGroup := "42"
		pciAddr := "0000:08:00.0"
		dpi := devicemanager.NewPCIDevicePluginForTest(map[string]string{iommuGroup: pciAddr})

		resp, err := dpi.Allocate(context.Background(), &pluginapi.AllocateRequest{
			ContainerRequests: []*pluginapi.ContainerAllocateRequest{
				{DevicesIDs: []string{iommuGroup}},
			},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.ContainerResponses).To(HaveLen(1))

		devicePaths := make([]string, 0)
		for _, spec := range resp.ContainerResponses[0].Devices {
			devicePaths = append(devicePaths, spec.ContainerPath)
		}
		Expect(devicePaths).ToNot(ContainElement(ContainSubstring(vfioDevicesDir)))
	})

	It("should skip VFIO cdev device specs when sysfs dir exists but device files don't", func() {
		// This simulates kernels without CONFIG_VFIO_DEVICE_CDEV where the
		// vfio-dev directory exists in sysfs but the actual character devices
		// in /dev/vfio/devices/ don't exist
		pciAddr := "0000:08:00.0"
		iommuGroup := "42"
		vfioDevDir := filepath.Join(tmpDir, pciAddr, "vfio-dev")
		Expect(os.MkdirAll(vfioDevDir, 0755)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(vfioDevDir, "vfio0"), []byte{}, 0644)).To(Succeed())

		// Note: we don't create /dev/vfio/devices/vfio0, simulating the missing device

		dpi := devicemanager.NewPCIDevicePluginForTest(map[string]string{iommuGroup: pciAddr})

		resp, err := dpi.Allocate(context.Background(), &pluginapi.AllocateRequest{
			ContainerRequests: []*pluginapi.ContainerAllocateRequest{
				{DevicesIDs: []string{iommuGroup}},
			},
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(resp.ContainerResponses).To(HaveLen(1))

		// Should only contain legacy VFIO group devices, not cdev devices
		devicePaths := make([]string, 0)
		for _, spec := range resp.ContainerResponses[0].Devices {
			devicePaths = append(devicePaths, spec.ContainerPath)
		}
		Expect(devicePaths).ToNot(ContainElement(ContainSubstring("/dev/vfio/devices/")))
		Expect(devicePaths).To(ContainElement("/dev/vfio/vfio"))
	})
})
