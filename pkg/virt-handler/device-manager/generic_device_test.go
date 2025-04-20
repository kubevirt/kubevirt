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
 */

package device_manager

import (
	"os"
	"path"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Generic Device", func() {
	var workDir string
	var err error
	var dpi *GenericDevicePlugin
	var stop chan struct{}
	var devicePath string

	BeforeEach(func() {
		workDir, err = os.MkdirTemp("", "kubevirt-test")
		Expect(err).ToNot(HaveOccurred())

		devicePath = path.Join(workDir, "foo")
		fileObj, err := os.Create(devicePath)
		Expect(err).ToNot(HaveOccurred())
		fileObj.Close()

		dpi = NewGenericDevicePlugin("foo", devicePath, 1, "rw", true)
		dpi.socketPath = filepath.Join(workDir, "test.sock")
		dpi.server = grpc.NewServer([]grpc.ServerOption{}...)
		dpi.done = make(chan struct{})
		dpi.deviceRoot = "/"
		stop = make(chan struct{})
		dpi.stop = stop

	})

	AfterEach(func() {
		close(stop)
		os.RemoveAll(workDir)
	})

	It("Should stop if the device plugin socket file is deleted", func() {
		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		errChan := make(chan error, 1)
		go func(errChan chan error) {
			errChan <- dpi.healthCheck()
		}(errChan)
		Consistently(func() string {
			return dpi.devs[0].Health
		}, 2*time.Second, 500*time.Millisecond).Should(Equal(pluginapi.Healthy))
		Expect(os.Remove(dpi.socketPath)).To(Succeed())

		Expect(<-errChan).ToNot(HaveOccurred())
	})

	It("Should monitor health of device node", func() {

		os.OpenFile(dpi.socketPath, os.O_RDONLY|os.O_CREATE, 0666)

		go dpi.healthCheck()
		Expect(dpi.devs[0].Health).To(Equal(pluginapi.Healthy))

		time.Sleep(1 * time.Second)
		By("Removing a (fake) device node")
		os.Remove(devicePath)

		By("waiting for healthcheck to send Unhealthy message")
		Eventually(func() string {
			return (<-dpi.health).Health
		}, 5*time.Second).Should(Equal(pluginapi.Unhealthy))

		By("Creating a new (fake) device node")
		fileObj, err := os.Create(devicePath)
		Expect(err).ToNot(HaveOccurred())
		fileObj.Close()

		By("waiting for healthcheck to send Healthy message")
		Eventually(func() string {
			return (<-dpi.health).Health
		}, 5*time.Second).Should(Equal(pluginapi.Healthy))
	})
})
