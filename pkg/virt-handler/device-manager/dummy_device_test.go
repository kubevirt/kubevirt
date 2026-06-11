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
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Dummy Device", func() {
	var dpi *DummyDevicePlugin

	BeforeEach(func() {
		workDir, err := os.MkdirTemp("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		dpi = NewDummyDevicePlugin("foo", 3)
		dpi.socketPath = filepath.Join(workDir, "test.sock")
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.done = make(chan struct{})
		stop := make(chan struct{})
		dpi.stop = stop
		DeferCleanup(func() {
			close(stop)
			os.RemoveAll(workDir)
		})
	})

	It("Should advertise the requested capacity as healthy devices", func() {
		Expect(dpi.devs).To(HaveLen(3))
		for _, dev := range dpi.devs {
			Expect(dev.Health).To(Equal(pluginapi.Healthy))
		}
	})

	It("Should allocate without exposing any device", func() {
		response, err := dpi.Allocate(context.Background(), &pluginapi.AllocateRequest{
			ContainerRequests: []*pluginapi.ContainerAllocateRequest{
				{DevicesIDs: []string{"foo0"}},
				{DevicesIDs: []string{"foo1"}},
			},
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(response.ContainerResponses).To(HaveLen(2))
		for _, containerResponse := range response.ContainerResponses {
			Expect(containerResponse.Devices).To(BeEmpty())
			Expect(containerResponse.Mounts).To(BeEmpty())
		}
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 500*time.Millisecond, 100*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Expect(<-errChan).ToNot(HaveOccurred())
	})
})
