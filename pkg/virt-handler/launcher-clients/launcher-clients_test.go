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
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

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

		var wg sync.WaitGroup
		for range 5 {
			wg.Go(func() {
				defer GinkgoRecover()
				clientInfo.Close()
			})
		}
		wg.Wait()
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

		It("should return the FindSocket error as recoverable when it is not ErrNoSocketFound", func() {
			// Two sockets on the same node make FindSocket ambiguous, which is a
			// recoverable condition: one of the two pods is expected to go away.
			vmi.Status.ActivePods[types.UID(otherPodUID)] = nodeName
			createSocketFile(socketPath)
			createSocketFile(cmdclient.SocketFilePathOnHost(otherPodUID))

			mockClient := cmdclient.NewMockLauncherClient(gomock.NewController(GinkgoT()))
			mockClient.EXPECT().Ping().Return(fmt.Errorf("connection refused"))

			manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
				Client:     mockClient,
				SocketFile: socketPath,
				Ready:      true,
			})

			client, err := manager.GetVerifiedLauncherClient(vmi)

			Expect(err).To(MatchError(ContainSubstring("Found multiple sockets")))
			Expect(err).ToNot(MatchError(IrrecoverableError))
			Expect(client).To(BeNil())
		})

		It("should propagate the error of a failed client lookup", func() {
			// Nothing is cached and no socket exists, so GetLauncherClient fails
			// before Ping is ever attempted.
			client, err := manager.GetVerifiedLauncherClient(vmi)

			Expect(err).To(MatchError(IrrecoverableError))
			Expect(client).To(BeNil())
		})
	})
})

var _ = Describe("GetLauncherClient", func() {
	var (
		manager          *launcherClientsManager
		notify           *fakeNotifyManager
		ghostRecordStore *virtcache.GhostRecordStore
		vmi              *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		vmi = newTestVMI()
		setupPodsBaseDir()
		ghostRecordStore = newGhostRecordStore()
		notify = &fakeNotifyManager{}
		manager = NewLauncherClientsManager(nil, notify, ghostRecordStore)
	})

	It("should return the cached client without creating a new one", func() {
		mockClient := cmdclient.NewMockLauncherClient(gomock.NewController(GinkgoT()))
		manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{Client: mockClient})

		client, err := manager.GetLauncherClient(vmi)

		Expect(err).ToNot(HaveOccurred())
		Expect(client).To(Equal(mockClient))
		Expect(notify.callCount()).To(BeZero())
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
	})

	It("should return an irrecoverable error when no socket exists", func() {
		client, err := manager.GetLauncherClient(vmi)

		Expect(err).To(MatchError(IrrecoverableError))
		Expect(err).To(MatchError(cmdclient.ErrNoSocketFound))
		Expect(client).To(BeNil())
		Expect(manager.GetLauncherClientInfo(vmi)).To(BeNil())
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
	})

	It("should return a recoverable error when multiple sockets are found", func() {
		vmi.Status.ActivePods[types.UID(otherPodUID)] = nodeName
		createSocketFile(cmdclient.SocketFilePathOnHost(podUID))
		createSocketFile(cmdclient.SocketFilePathOnHost(otherPodUID))

		client, err := manager.GetLauncherClient(vmi)

		Expect(err).To(MatchError(ContainSubstring("Found multiple sockets")))
		Expect(err).ToNot(MatchError(IrrecoverableError))
		Expect(client).To(BeNil())
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
	})

	It("should fail when a ghost record of a previous VMI incarnation exists", func() {
		const staleUID types.UID = "stale-vmi-uid"
		Expect(ghostRecordStore.Add(vmi.Namespace, vmi.Name, "/stale/socket", staleUID)).To(Succeed())
		// A plain file would make the gRPC dial time out, so a quick ghost record
		// error also proves the record is stored before connecting.
		createSocketFile(cmdclient.SocketFilePathOnHost(podUID))

		client, err := manager.GetLauncherClient(vmi)

		Expect(err).To(MatchError(ContainSubstring("differing UID")))
		Expect(client).To(BeNil())
		Expect(notify.callCount()).To(BeZero())
		Expect(manager.GetLauncherClientInfo(vmi)).To(BeNil())
		Expect(ghostRecordStore.LastKnownUID(ghostRecordKey(vmi))).To(Equal(staleUID))
	})

	It("should fail when the client can not be created", func() {
		// A plain file instead of a listening socket makes the gRPC dial fail.
		createSocketFile(cmdclient.SocketFilePathOnHost(podUID))

		client, err := manager.GetLauncherClient(vmi)

		Expect(err).To(MatchError("context deadline exceeded"))
		Expect(client).To(BeNil())
		Expect(notify.callCount()).To(BeZero())
		Expect(manager.GetLauncherClientInfo(vmi)).To(BeNil())
		// The socket was found, so the record stays until CloseLauncherClient.
		Expect(ghostRecordStore.LastKnownUID(ghostRecordKey(vmi))).To(Equal(vmi.UID))
	})

	When("a launcher is listening on the socket", func() {
		var socketPath string

		BeforeEach(func() {
			socketPath = cmdclient.SocketFilePathOnHost(podUID)
			startFakeLauncher(socketPath)
			DeferCleanup(func() {
				manager.GetLauncherClientInfo(vmi).Close()
			})
		})

		It("should create, cache and start notifying for a new client", func() {
			client, err := manager.GetLauncherClient(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())

			clientInfo := manager.GetLauncherClientInfo(vmi)
			Expect(clientInfo).ToNot(BeNil())
			Expect(clientInfo.Client).To(Equal(client))
			Expect(clientInfo.SocketFile).To(Equal(socketPath))
			Expect(clientInfo.Ready).To(BeTrue())
			Expect(clientInfo.DomainPipeStopChan).ToNot(BeClosed())

			Expect(notify.callCount()).To(Equal(1))
			Expect(notify.lastVMI()).To(Equal(vmi))
			Expect(ghostRecordStore.LastKnownUID(ghostRecordKey(vmi))).To(Equal(vmi.UID))

			By("returning the very same client on a subsequent call")
			cachedClient, err := manager.GetLauncherClient(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(cachedClient).To(Equal(client))
			Expect(notify.callCount()).To(Equal(1))
		})

		It("should tear down the client when starting the domain notifier fails", func() {
			notify.err = fmt.Errorf("pipe failed")

			client, err := manager.GetLauncherClient(vmi)

			Expect(err).To(MatchError("pipe failed"))
			Expect(err).ToNot(MatchError(IrrecoverableError))
			Expect(client).To(BeNil())
			Expect(manager.GetLauncherClientInfo(vmi)).To(BeNil())
			Expect(notify.lastStopChan()).To(BeClosed())
			// The socket was found, so the record stays until CloseLauncherClient.
			Expect(ghostRecordStore.LastKnownUID(ghostRecordKey(vmi))).To(Equal(vmi.UID))
		})

		It("should create a single client when called concurrently", func() {
			const concurrency = 10

			clients := make([]cmdclient.LauncherClient, concurrency)
			errs := make([]error, concurrency)
			start := make(chan struct{})

			var wg sync.WaitGroup
			for i := range concurrency {
				wg.Go(
					func() {
						defer GinkgoRecover()
						<-start
						clients[i], errs[i] = manager.GetLauncherClient(vmi)
					})
			}
			close(start)
			wg.Wait()

			for i := range concurrency {
				Expect(errs[i]).ToNot(HaveOccurred())
				Expect(clients[i]).To(Equal(clients[0]))
			}
			Expect(notify.callCount()).To(Equal(1))
		})
	})
})

var _ = Describe("GetLauncherClientInfo", func() {
	var (
		manager *launcherClientsManager
		vmi     *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		vmi = newTestVMI()
		manager = NewLauncherClientsManager(nil, nil, nil)
	})

	It("should return nil for an unknown VMI", func() {
		Expect(manager.GetLauncherClientInfo(vmi)).To(BeNil())
	})

	It("should return the stored info for a known VMI", func() {
		clientInfo := &virtcache.LauncherClientInfo{SocketFile: "/some/socket"}
		manager.launcherClients.Store(vmi.UID, clientInfo)

		Expect(manager.GetLauncherClientInfo(vmi)).To(BeIdenticalTo(clientInfo))
	})
})

var _ = Describe("CloseLauncherClient", func() {
	var (
		manager          *launcherClientsManager
		ghostRecordStore *virtcache.GhostRecordStore
		vmi              *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		vmi = newTestVMI()
		setupPodsBaseDir()
		ghostRecordStore = newGhostRecordStore()
		manager = NewLauncherClientsManager(nil, nil, ghostRecordStore)
	})

	It("should do nothing for a VMI without UID", func() {
		vmi.UID = ""
		Expect(ghostRecordStore.Add(vmi.Namespace, vmi.Name, "/some/socket", "other-uid")).To(Succeed())

		manager.CloseLauncherClient(vmi)

		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeTrue())
	})

	It("should close the client and drop all of its records", func() {
		mockClient := cmdclient.NewMockLauncherClient(gomock.NewController(GinkgoT()))
		mockClient.EXPECT().Close()
		stopChan := make(chan struct{})
		manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
			Client:             mockClient,
			DomainPipeStopChan: stopChan,
		})
		Expect(ghostRecordStore.Add(vmi.Namespace, vmi.Name, "/some/socket", vmi.UID)).To(Succeed())

		manager.CloseLauncherClient(vmi)

		Expect(stopChan).To(BeClosed())
		Expect(manager.GetLauncherClientInfo(vmi)).To(BeNil())
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
	})

	It("should drop the ghost record even if no client was ever created", func() {
		Expect(ghostRecordStore.Add(vmi.Namespace, vmi.Name, "/some/socket", vmi.UID)).To(Succeed())

		manager.CloseLauncherClient(vmi)

		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
	})
})

var _ = Describe("IsLauncherClientUnresponsive", func() {
	var (
		manager    *launcherClientsManager
		vmi        *v1.VirtualMachineInstance
		socketPath string
	)

	BeforeEach(func() {
		vmi = newTestVMI()
		setupPodsBaseDir()
		socketPath = cmdclient.SocketFilePathOnHost(podUID)
		manager = NewLauncherClientsManager(nil, nil, nil)
	})

	Context("without a cached entry", func() {
		It("should report an existing socket as responsive and cache it", func() {
			createSocketFile(socketPath)

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeFalse())
			Expect(initialized).To(BeTrue())

			clientInfo := manager.GetLauncherClientInfo(vmi)
			Expect(clientInfo).ToNot(BeNil())
			Expect(clientInfo.Ready).To(BeTrue())
			Expect(clientInfo.SocketFile).To(Equal(socketPath))
		})

		It("should report a socket marked unresponsive as unresponsive", func() {
			createSocketFile(socketPath)
			Expect(cmdclient.MarkSocketUnresponsive(socketPath)).To(Succeed())

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeTrue())
			Expect(initialized).To(BeTrue())
		})

		It("should report unresponsive when neither socket nor pod dir exist", func() {
			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeTrue())
			Expect(initialized).To(BeTrue())
		})

		It("should report not initialized while the pod dir exists without a socket", func() {
			createPodDir()

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeFalse())
			Expect(initialized).To(BeFalse())
			Expect(manager.GetLauncherClientInfo(vmi).Ready).To(BeFalse())
		})
	})

	Context("with a cached ready entry", func() {
		BeforeEach(func() {
			manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{
				SocketFile: socketPath,
				Ready:      true,
			})
		})

		It("should use the cached socket and report it responsive", func() {
			createSocketFile(socketPath)

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeFalse())
			Expect(initialized).To(BeTrue())
		})

		It("should report unresponsive once the cached socket is gone", func() {
			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeTrue())
			Expect(initialized).To(BeTrue())
		})
	})

	Context("with a cached entry that is not ready yet", func() {
		var clientInfo *virtcache.LauncherClientInfo

		BeforeEach(func() {
			clientInfo = &virtcache.LauncherClientInfo{
				NotInitializedSince: time.Now(),
				Ready:               false,
			}
			manager.launcherClients.Store(vmi.UID, clientInfo)
		})

		It("should mark it ready once the socket shows up", func() {
			createSocketFile(socketPath)

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeFalse())
			Expect(initialized).To(BeTrue())
			Expect(clientInfo.Ready).To(BeTrue())
			Expect(clientInfo.SocketFile).To(Equal(socketPath))
		})

		It("should report unresponsive when the pod is gone", func() {
			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeTrue())
			Expect(initialized).To(BeTrue())
		})

		It("should keep waiting while the pod exists and the grace period did not pass", func() {
			createPodDir()

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeFalse())
			Expect(initialized).To(BeFalse())
		})

		It("should give up once the socket did not show up in time", func() {
			createPodDir()
			clientInfo.NotInitializedSince = time.Now().Add(-4 * time.Minute)

			unresponsive, initialized, err := manager.IsLauncherClientUnresponsive(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(unresponsive).To(BeTrue())
			Expect(initialized).To(BeTrue())
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
	cmdclient.SetPodsBaseDir(GinkgoT().TempDir())
	GinkgoT().Setenv("NODE_NAME", nodeName)
}

func newGhostRecordStore() *virtcache.GhostRecordStore {
	return virtcache.InitializeGhostRecordCache(
		virtcache.NewIterableCheckpointManager(GinkgoT().TempDir(), GinkgoT().TempDir()),
	)
}

func ghostRecordKey(vmi *v1.VirtualMachineInstance) string {
	return vmi.Namespace + "/" + vmi.Name
}

func createPodDir() {
	Expect(os.MkdirAll(cmdclient.SocketDirectoryOnHost(podUID), 0755)).To(Succeed())
}

func createSocketFile(socketPath string) {
	Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())
	f, err := os.Create(socketPath)
	Expect(err).ToNot(HaveOccurred())
	Expect(f.Close()).To(Succeed())
}

// startFakeLauncher serves just enough of the launcher gRPC API on socketPath
// for cmdclient.NewClient to negotiate a connection.
func startFakeLauncher(socketPath string) {
	Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())
	listener, err := net.Listen("unix", socketPath)
	Expect(err).ToNot(HaveOccurred())

	server := grpc.NewServer()
	info.RegisterCmdInfoServer(server, &fakeCmdInfoServer{})
	go func() {
		defer GinkgoRecover()
		_ = server.Serve(listener)
	}()
	DeferCleanup(server.Stop)
}

type fakeCmdInfoServer struct{}

func (*fakeCmdInfoServer) Info(context.Context, *info.CmdInfoRequest) (*info.CmdInfoResponse, error) {
	return &info.CmdInfoResponse{SupportedCmdVersions: []uint32{1}}, nil
}

type fakeNotifyManager struct {
	lock     sync.Mutex
	err      error
	calls    int
	stopChan <-chan struct{}
	vmi      *v1.VirtualMachineInstance
}

func (f *fakeNotifyManager) StartDomainNotify(stopChan <-chan struct{}, vmi *v1.VirtualMachineInstance) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.calls++
	f.stopChan = stopChan
	f.vmi = vmi
	return f.err
}

func (f *fakeNotifyManager) callCount() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.calls
}

func (f *fakeNotifyManager) lastStopChan() <-chan struct{} {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.stopChan
}

func (f *fakeNotifyManager) lastVMI() *v1.VirtualMachineInstance {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.vmi
}
