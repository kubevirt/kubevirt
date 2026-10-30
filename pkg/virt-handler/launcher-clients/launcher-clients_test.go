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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/grpcserver"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
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
		podUID     = "test-pod-uid"
		socketPath string
	)

	BeforeEach(func() {
		vmi = libvmi.New(
			libvmi.WithName("test-vmi"),
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithUID("test-vmi-uid"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithActivePod(types.UID(podUID), "test-node"),
			)),
		)

		// Set up socket directory
		podsDir := GinkgoT().TempDir()
		cmdclient.SetPodsBaseDir(podsDir)
		socketPath = cmdclient.SocketFilePathOnHost(podUID)

		// Set environment for FindSocket to find the correct path
		_ = os.Setenv("NODE_NAME", "test-node")
		DeferCleanup(func() {
			_ = os.Unsetenv("NODE_NAME")
		})

		manager = NewLauncherClientsManager(nil, nil)
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
			Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())
			f, err := os.Create(socketPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Close()).To(Succeed())

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

var _ = Describe("GetLauncherClient concurrent calls", func() {
	const callers = 5

	var (
		manager    *launcherClientsManager
		vmi        *v1.VirtualMachineInstance
		infoServer *blockingInfoServer
	)

	BeforeEach(func() {
		const podUID = "test-pod-uid"
		vmi = libvmi.New(
			libvmi.WithName("test-vmi"),
			libvmi.WithNamespace(k8sv1.NamespaceDefault),
			libvmi.WithUID("test-vmi-uid"),
			libvmistatus.WithStatus(libvmistatus.New(
				libvmistatus.WithActivePod(types.UID(podUID), "test-node"),
			)),
		)

		_ = virtcache.InitializeGhostRecordCache(virtcache.NewIterableCheckpointManager(GinkgoT().TempDir(), GinkgoT().TempDir()))

		// A unix socket path is limited to 108 bytes, keep the base short.
		podsDir, err := os.MkdirTemp("", "lc")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(os.RemoveAll, podsDir)
		cmdclient.SetPodsBaseDir(podsDir)
		socketPath := cmdclient.SocketFilePathOnHost(podUID)
		Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())

		_ = os.Setenv("NODE_NAME", "test-node")
		DeferCleanup(func() {
			_ = os.Unsetenv("NODE_NAME")
		})

		// Every cmdclient.NewClient call issues exactly one Info RPC, so the
		// number of Info calls is the number of connections opened.
		infoServer = &blockingInfoServer{release: make(chan struct{})}
		stop := make(chan struct{})
		done, err := grpcserver.RunServer(stop, socketPath, func(s *grpc.Server) {
			info.RegisterCmdInfoServer(s, infoServer)
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			infoServer.unblock()
			close(stop)
			<-done
		})

		// Failing the domain notify setup keeps the client out of the cache,
		// so every caller has to go through connection setup.
		manager = NewLauncherClientsManager(nil, failingNotifyManager{})
	})

	It("should open a single connection per VMI", func() {
		var wg sync.WaitGroup
		errs := make([]error, callers)
		for i := range callers {
			wg.Go(func() {
				_, errs[i] = manager.GetLauncherClient(vmi)
			})
		}

		Eventually(infoServer.calls.Load).Should(BeEquivalentTo(1))
		Consistently(infoServer.calls.Load).WithTimeout(500 * time.Millisecond).Should(BeEquivalentTo(1))

		infoServer.unblock()
		wg.Wait()
		for _, err := range errs {
			Expect(err).To(MatchError(ContainSubstring("start domain notify failed")))
		}
	})
})

// blockingInfoServer answers the cmd version handshake only after unblock is
// called, which keeps connection setup in flight while other callers arrive.
type blockingInfoServer struct {
	calls   atomic.Int32
	release chan struct{}
	once    sync.Once
}

func (s *blockingInfoServer) Info(_ context.Context, _ *info.CmdInfoRequest) (*info.CmdInfoResponse, error) {
	s.calls.Add(1)
	<-s.release
	return &info.CmdInfoResponse{SupportedCmdVersions: []uint32{1}}, nil
}

func (s *blockingInfoServer) unblock() {
	s.once.Do(func() { close(s.release) })
}

type failingNotifyManager struct{}

func (failingNotifyManager) StartDomainNotify(<-chan struct{}, *v1.VirtualMachineInstance) error {
	return fmt.Errorf("start domain notify failed")
}
