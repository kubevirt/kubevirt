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

package launcher_clients

import (
	"fmt"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("LauncherClientInfo Close", func() {
	It("should safely handle multiple Close calls without panicking", func() {
		stopChan := make(chan struct{})
		clientInfo := &virtcache.LauncherClientInfo{
			DomainPipeStopChan: stopChan,
		}

		clientInfo.Close()

		Expect(func() {
			clientInfo.Close()
		}).ToNot(Panic())

		Expect(func() {
			clientInfo.Close()
		}).ToNot(Panic())
	})

	It("should handle concurrent Close calls without panicking", func() {
		stopChan := make(chan struct{})
		clientInfo := &virtcache.LauncherClientInfo{
			DomainPipeStopChan: stopChan,
		}

		done := make(chan bool, 5)
		for range 5 {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						Fail(fmt.Sprintf("Panic occurred during concurrent Close: %v", r))
					}
					done <- true
				}()
				clientInfo.Close()
			}()
		}

		for range 5 {
			<-done
		}
	})
})

var _ = Describe("GetVerifiedLauncherClient FindSocket check", func() {
	var (
		manager    *launcherClientsManager
		vmi        *v1.VirtualMachineInstance
		socketPath string
	)

	BeforeEach(func() {
		vmi = newTestVMI()

		// Set up socket directory and make FindSocket look at this node
		setupPodsBaseDir()
		socketPath = cmdclient.SocketFilePathOnHost(podUID)

		manager = NewLauncherClientsManager(nil, nil, nil)
	})

	It("should succeed when Ping succeed without checking the socket", func() {
		// Don't create the socket file, so we can check FindSocket is not performed

		// Create mock client that succeeds on Ping
		ctrl := gomock.NewController(GinkgoT())
		mockClient := cmdclient.NewMockLauncherClient(ctrl)
		mockClient.EXPECT().Ping().Return(nil)

		// Store the client in the manager
		manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
			Client:     mockClient,
			SocketFile: "/nonexistent/socket",
			Ready:      true,
		})

		// Call GetVerifiedLauncherClient
		client, err := manager.GetVerifiedLauncherClient(vmi)

		// Ping succeeded, so no error
		Expect(err).ToNot(HaveOccurred())
		Expect(client).To(Equal(mockClient))
	})

	When("Ping fails", func() {
		It("should check FindSocket and return the Ping(non irrecoverable) error if it succeeds", func() {
			// Create the socket file structure
			createSocketFile(socketPath)

			// Create mock client that fails on Ping
			ctrl := gomock.NewController(GinkgoT())
			mockClient := cmdclient.NewMockLauncherClient(ctrl)
			pingErr := fmt.Errorf("connection refused")
			mockClient.EXPECT().Ping().Return(pingErr)

			// Store the client in the manager
			manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
				Client:     mockClient,
				SocketFile: socketPath,
				Ready:      true,
			})

			// Call GetVerifiedLauncherClient
			client, err := manager.GetVerifiedLauncherClient(vmi)

			Expect(err).To(MatchError(ContainSubstring("connection refused")))
			Expect(err).ToNot(MatchError(IrrecoverableError))
			Expect(client).To(BeNil())
		})

		It("should check FindSocket and return it as irrecoverable error if it fails", func() {
			// Don't create the socket file, so FindSocket will fail

			// Create mock client that fails on Ping
			ctrl := gomock.NewController(GinkgoT())
			mockClient := cmdclient.NewMockLauncherClient(ctrl)
			pingErr := fmt.Errorf("connection refused")
			mockClient.EXPECT().Ping().Return(pingErr)

			// Store the client in the manager
			manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
				Client:     mockClient,
				SocketFile: "/nonexistent/socket",
				Ready:      true,
			})

			// Call GetVerifiedLauncherClient
			client, err := manager.GetVerifiedLauncherClient(vmi)

			// Both Ping and FindSocket failed, so we should get an error
			Expect(err).To(MatchError(ContainSubstring("no sockets found for VMI")))
			Expect(err).To(MatchError(IrrecoverableError))
			Expect(client).To(BeNil())
		})
	})
})

const (
	podUID      = "test-pod-uid"
	otherPodUID = "other-pod-uid"
	nodeName    = "test-node"
)

func newTestVMI() *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithName("test-vmi"),
		libvmi.WithNamespace(k8sv1.NamespaceDefault),
		libvmi.WithUID("test-vmi-uid"),
		libvmistatus.WithStatus(libvmistatus.New(
			libvmistatus.WithActivePod(types.UID(podUID), nodeName),
		)),
	)
}

func setupPodsBaseDir() {
	podsDir := GinkgoT().TempDir()
	DeferCleanup(func() {
		Expect(os.RemoveAll(podsDir)).To(Succeed())
	})

	cmdclient.SetPodsBaseDir(podsDir)
	Expect(os.Setenv("NODE_NAME", nodeName)).To(Succeed())
	DeferCleanup(func() {
		Expect(os.Unsetenv("NODE_NAME")).To(Succeed())
	})
}

func createSocketFile(socketPath string) {
	Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())
	f, err := os.Create(socketPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(f.Close()).To(Succeed())
}
