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

package device_manager

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Secure Guest Capacity Device Plugin", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "test-cvm-capacity-")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	It("misc.capacity doesn't exist", func() {
		nonExistentPath := filepath.Join(tmpDir, "nonexistent")
		plugin, err := newSecureGuestCapacityDevicePlugin(nonExistentPath)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("No secure guest capacity available"))
		Expect(plugin).To(BeNil())
	})

	It("misc.capacity has TDX capacity", func() {
		capacityFile := filepath.Join(tmpDir, "misc.capacity")
		err := os.WriteFile(capacityFile, []byte("tdx 15\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		plugin, err := newSecureGuestCapacityDevicePlugin(capacityFile)

		Expect(err).ToNot(HaveOccurred())
		Expect(plugin).ToNot(BeNil())
		Expect(plugin.secureGuestType).To(Equal(TDX))
		Expect(plugin.capacity).To(Equal(15))
		Expect(plugin.resourceName).To(Equal(TDXResourceName))
		Expect(plugin.socketPath).To(Equal(SocketPath("tdx.intel.com-keys")))
	})

	It("misc.capacity has SEV-SNP capacity", func() {
		capacityFile := filepath.Join(tmpDir, "misc.capacity")
		err := os.WriteFile(capacityFile, []byte("sev 509\nsev_es 99\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		plugin, err := newSecureGuestCapacityDevicePlugin(capacityFile)

		Expect(err).ToNot(HaveOccurred())
		Expect(plugin).ToNot(BeNil())
		Expect(plugin.secureGuestType).To(Equal(SNP))
		Expect(plugin.capacity).To(Equal(99))
		Expect(plugin.resourceName).To(Equal(SNPResourceName))
		Expect(plugin.socketPath).To(Equal(SocketPath("sev-snp.amd.com-esids")))
	})

	It("misc.capacity has unsupported types", func() {
		capacityFile := filepath.Join(tmpDir, "misc.capacity")
		err := os.WriteFile(capacityFile, []byte("unknown_type 10\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		plugin, err := newSecureGuestCapacityDevicePlugin(capacityFile)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("No secure guest capacity available"))
		Expect(plugin).To(BeNil())
	})

	It("misc.capacity is empty", func() {
		capacityFile := filepath.Join(tmpDir, "misc.capacity")
		err := os.WriteFile(capacityFile, []byte(""), 0644)
		Expect(err).ToNot(HaveOccurred())

		plugin, err := newSecureGuestCapacityDevicePlugin(capacityFile)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("No secure guest capacity available"))
		Expect(plugin).To(BeNil())
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		capacityFile := filepath.Join(tmpDir, "misc.capacity")
		err := os.WriteFile(capacityFile, []byte("tdx 15\n"), 0644)
		Expect(err).ToNot(HaveOccurred())

		plugin, err := newSecureGuestCapacityDevicePlugin(capacityFile)
		Expect(err).ToNot(HaveOccurred())

		plugin.socketPath = filepath.Join(tmpDir, "kubevirt-test.sock")
		fh, err := os.OpenFile(plugin.socketPath, os.O_RDONLY|os.O_CREATE, 0666)
		Expect(err).ToNot(HaveOccurred())
		fh.Close()
		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- plugin.healthCheck()
		}(errChan)
		Consistently(func() string {
			return plugin.devs[0].Health
		}, 500*time.Millisecond, 100*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(plugin.socketPath)).To(Succeed())
		Expect(<-errChan).ToNot(HaveOccurred())
	})
})

type mockListAndWatchServer struct {
	pluginapi.DevicePlugin_ListAndWatchServer
	devices chan []*pluginapi.Device
}

func (m *mockListAndWatchServer) Send(resp *pluginapi.ListAndWatchResponse) error {
	m.devices <- resp.Devices
	return nil
}
