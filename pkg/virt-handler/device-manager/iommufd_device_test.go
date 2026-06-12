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
	"context"
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sys/unix"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("IOMMUFD Device Plugin", func() {
	var origRelabelFunc func(string) error

	BeforeEach(func() {
		origRelabelFunc = relabelPathFunc
		relabelPathFunc = func(_ string) error { return nil }
	})

	AfterEach(func() {
		relabelPathFunc = origRelabelFunc
	})

	Describe("NewIOMMUFDDevicePlugin", func() {
		It("should create a plugin with the correct number of devices", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			Expect(dp.devs).To(HaveLen(10))
		})

		It("should set all devices to healthy", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			for _, dev := range dp.devs {
				Expect(dev.Health).To(Equal(pluginapi.Healthy))
			}
		})

		It("should assign unique device IDs", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			ids := make(map[string]struct{})
			for _, dev := range dp.devs {
				_, exists := ids[dev.ID]
				Expect(exists).To(BeFalse(), "duplicate device ID: %s", dev.ID)
				ids[dev.ID] = struct{}{}
			}
		})

		It("should set the correct resource name", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			Expect(dp.resourceName).To(Equal("devices.kubevirt.io/iommufd"))
		})

		It("should set the correct socket path", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			Expect(dp.socketPath).To(Equal(SocketPath("iommufd")))
		})
	})

	Describe("GetDevicePluginOptions", func() {
		It("should return options with PreStartRequired false", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			opts, err := dp.GetDevicePluginOptions(context.Background(), &pluginapi.Empty{})
			Expect(err).NotTo(HaveOccurred())
			Expect(opts.PreStartRequired).To(BeFalse())
		})
	})

	Describe("PreStartContainer", func() {
		It("should return an empty response", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			resp, err := dp.PreStartContainer(context.Background(), &pluginapi.PreStartContainerRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).NotTo(BeNil())
		})
	})

	Describe("GetInitialized", func() {
		It("should return false by default", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			Expect(dp.GetInitialized()).To(BeFalse())
		})

		It("should return true after setInitialized(true)", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			dp.setInitialized(true)
			Expect(dp.GetInitialized()).To(BeTrue())
		})

		It("should return false after setInitialized(false)", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			dp.setInitialized(true)
			dp.setInitialized(false)
			Expect(dp.GetInitialized()).To(BeFalse())
		})
	})

	Describe("cleanup", func() {
		It("should remove the socket file", func() {
			tmpDir := GinkgoT().TempDir()
			dp := NewIOMMUFDDevicePlugin(10)
			dp.socketPath = filepath.Join(tmpDir, "test.sock")

			f, err := os.Create(dp.socketPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Close()).To(Succeed())

			Expect(dp.cleanup()).To(Succeed())
			_, err = os.Stat(dp.socketPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("should succeed when socket file does not exist", func() {
			dp := NewIOMMUFDDevicePlugin(10)
			dp.socketPath = "/tmp/nonexistent-socket-path.sock"
			Expect(dp.cleanup()).To(Succeed())
		})
	})

	Describe("Allocate", func() {
		Context("when IOMMUFD is not supported", func() {
			var origCheckPath string

			BeforeEach(func() {
				origCheckPath = iommuDeviceCheckPath
				iommuDeviceCheckPath = "/dev/nonexistent-iommu-device"
			})

			AfterEach(func() {
				iommuDeviceCheckPath = origCheckPath
			})

			It("should return an error", func() {
				dp := NewIOMMUFDDevicePlugin(10)
				req := &pluginapi.AllocateRequest{
					ContainerRequests: []*pluginapi.ContainerAllocateRequest{
						{DevicesIDs: []string{"iommufd0"}},
					},
				}
				_, err := dp.Allocate(context.Background(), req)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("IOMMUFD not supported"))
			})
		})
	})

	Describe("supportsIOMMUFD", func() {
		It("should return false when the device does not exist", func() {
			original := iommuDeviceCheckPath
			iommuDeviceCheckPath = "/dev/nonexistent-iommu-device"
			defer func() { iommuDeviceCheckPath = original }()

			Expect(supportsIOMMUFD()).To(BeFalse())
		})
	})

	Describe("updateDeviceHealth", func() {
		It("should set devices to unhealthy when /dev/iommu is absent", func() {
			original := iommuDeviceCheckPath
			iommuDeviceCheckPath = "/dev/nonexistent-iommu-device"
			defer func() { iommuDeviceCheckPath = original }()

			dp := NewIOMMUFDDevicePlugin(3)
			dp.updateDeviceHealth()
			for _, dev := range dp.devs {
				Expect(dev.Health).To(Equal(pluginapi.Unhealthy))
			}
		})
	})

	Describe("createIOMMUFDSocket", func() {
		It("should create a socket and pass an FD to a connecting client", func() {
			tmpDir := GinkgoT().TempDir()

			r, w, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			defer w.Close()
			fd := int(r.Fd())

			socketPath, err := createIOMMUFDSocket(fd, tmpDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(socketPath).To(HavePrefix(filepath.Join(tmpDir, "iommufd-")))
			Expect(socketPath).To(HaveSuffix(".sock"))

			conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()

			buf := make([]byte, 1)
			oob := make([]byte, 24)
			_, oobn, _, _, err := conn.ReadMsgUnix(buf, oob)
			Expect(err).NotTo(HaveOccurred())
			Expect(oobn).To(BeNumerically(">", 0))

			scms, err := unix.ParseSocketControlMessage(oob[:oobn])
			Expect(err).NotTo(HaveOccurred())
			Expect(scms).NotTo(BeEmpty())

			fds, err := unix.ParseUnixRights(&scms[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(fds).To(HaveLen(1))
			defer unix.Close(fds[0])

			_, err = conn.Write([]byte{1})
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				_, err := os.Stat(socketPath)
				return os.IsNotExist(err)
			}, 5*time.Second, 100*time.Millisecond).Should(BeTrue())
		})

		It("should create the socket directory if it does not exist", func() {
			tmpDir := GinkgoT().TempDir()
			socketDir := filepath.Join(tmpDir, "subdir")

			r, w, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			defer w.Close()

			socketPath, err := createIOMMUFDSocket(int(r.Fd()), socketDir)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(socketDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())

			conn, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
			Expect(err).NotTo(HaveOccurred())
			conn.Close()
		})

		It("should create unique socket paths for concurrent allocations", func() {
			tmpDir := GinkgoT().TempDir()

			r1, w1, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			defer w1.Close()

			r2, w2, err := os.Pipe()
			Expect(err).NotTo(HaveOccurred())
			defer w2.Close()

			path1, err := createIOMMUFDSocket(int(r1.Fd()), tmpDir)
			Expect(err).NotTo(HaveOccurred())

			path2, err := createIOMMUFDSocket(int(r2.Fd()), tmpDir)
			Expect(err).NotTo(HaveOccurred())

			Expect(path1).NotTo(Equal(path2))
		})
	})

	Describe("ensureDirWithRelabel", func() {
		It("should create the directory", func() {
			tmpDir := GinkgoT().TempDir()
			target := tmpDir + "/subdir"
			Expect(ensureDirWithRelabel(target)).To(Succeed())
			info, err := os.Stat(target)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})

		It("should succeed when directory already exists", func() {
			tmpDir := GinkgoT().TempDir()
			Expect(ensureDirWithRelabel(tmpDir)).To(Succeed())
		})

		It("should create nested directories", func() {
			tmpDir := GinkgoT().TempDir()
			target := tmpDir + "/a/b/c"
			Expect(ensureDirWithRelabel(target)).To(Succeed())
			info, err := os.Stat(target)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})
	})
})
