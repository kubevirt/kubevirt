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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

var _ = Describe("GetLauncherClient", func() {
	const podUID = "test-pod-uid"

	var (
		manager          *launcherClientsManager
		vmi              *v1.VirtualMachineInstance
		socketPath       string
		ghostRecordStore *virtcache.GhostRecordStore
		infoServer       *fakeCmdInfoServer
		notifyManager    *fakeNotifyManager
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

		cmdclient.SetPodsBaseDir(GinkgoT().TempDir())
		socketPath = cmdclient.SocketFilePathOnHost(podUID)
		GinkgoT().Setenv("NODE_NAME", "test-node")

		ghostRecordStore = virtcache.InitializeGhostRecordCache(
			virtcache.NewIterableCheckpointManager(GinkgoT().TempDir(), GinkgoT().TempDir()),
		)
		infoServer = &fakeCmdInfoServer{}
		notifyManager = &fakeNotifyManager{}

		manager = NewLauncherClientsManager(nil, notifyManager, ghostRecordStore)
		DeferCleanup(manager.CloseLauncherClient, vmi)
	})

	// startLauncher serves the cmd Info service on the VMI's launcher socket,
	// which is all cmdclient.NewClient needs to hand out a client.
	startLauncher := func() {
		stop := make(chan struct{})
		done, err := grpcserver.RunServer(stop, socketPath, func(s *grpc.Server) {
			info.RegisterCmdInfoServer(s, infoServer)
		})
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() {
			close(stop)
			Eventually(done).Should(BeClosed())
		})
	}

	expectNothingCached := func() {
		ExpectWithOffset(1, manager.GetLauncherClientInfo(vmi)).To(BeNil())
	}

	expectClientCreatedAndCached := func() {
		client, err := manager.GetLauncherClient(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(client).ToNot(BeNil())

		clientInfo := manager.GetLauncherClientInfo(vmi)
		Expect(clientInfo).ToNot(BeNil())
		Expect(clientInfo.Client).To(BeIdenticalTo(client))
		Expect(clientInfo.SocketFile).To(Equal(socketPath))
		Expect(clientInfo.Ready).To(BeTrue())
		Expect(clientInfo.DomainPipeStopChan).ToNot(BeClosed())
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeTrue())

		By("returning the cached client on subsequent calls without reconnecting")
		infoCalls := infoServer.Calls()
		notifyCalls := len(notifyManager.stopChans)

		cached, err := manager.GetLauncherClient(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(cached).To(BeIdenticalTo(client))
		Expect(infoServer.Calls()).To(Equal(infoCalls))
		Expect(notifyManager.stopChans).To(HaveLen(notifyCalls))
	}

	It("should return a client once the launcher socket appears", func() {
		By("failing while the launcher socket does not exist")
		_, err := manager.GetLauncherClient(vmi)
		Expect(err).To(MatchError(IrrecoverableError))
		Expect(err).To(MatchError(cmdclient.ErrNoSocketFound))
		expectNothingCached()
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
		Expect(notifyManager.stopChans).To(BeEmpty())

		By("succeeding once the launcher serves the socket")
		startLauncher()
		expectClientCreatedAndCached()
	})

	It("should return a client when the first connection to the launcher failed", func() {
		infoServer.failures = 1
		startLauncher()

		By("failing when the launcher rejects the version check")
		_, err := manager.GetLauncherClient(vmi)
		Expect(err).To(MatchError(ContainSubstring("could not check cmd server version")))
		Expect(err).ToNot(MatchError(IrrecoverableError))
		expectNothingCached()
		Expect(notifyManager.stopChans).To(BeEmpty())
		// The failed attempt already recorded the ghost record, the retry has to accept it.
		Expect(ghostRecordStore.Exists(vmi.Namespace, vmi.Name)).To(BeTrue())

		By("succeeding on the next attempt")
		expectClientCreatedAndCached()
		Expect(infoServer.Calls()).To(Equal(2))
	})

	It("should return a client when starting domain notify failed on the first try", func() {
		notifyErr := errors.New("domain notify failed")
		notifyManager.errs = []error{notifyErr}
		startLauncher()

		By("failing when domain notify cannot be started")
		_, err := manager.GetLauncherClient(vmi)
		Expect(err).To(MatchError(notifyErr))
		Expect(err).ToNot(MatchError(IrrecoverableError))
		expectNothingCached()
		Expect(notifyManager.stopChans).To(HaveLen(1))
		Expect(notifyManager.stopChans[0]).To(BeClosed(), "the failed attempt should release its domain notify")

		By("succeeding on the next attempt")
		expectClientCreatedAndCached()
		Expect(notifyManager.stopChans).To(HaveLen(2))
	})
})

// fakeCmdInfoServer answers the cmd version handshake performed by cmdclient.NewClient.
// The first `failures` calls are rejected.
type fakeCmdInfoServer struct {
	lock     sync.Mutex
	failures int
	calls    int
}

func (s *fakeCmdInfoServer) Info(context.Context, *info.CmdInfoRequest) (*info.CmdInfoResponse, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.calls++
	if s.failures > 0 {
		s.failures--
		return nil, errors.New("launcher not ready")
	}
	return &info.CmdInfoResponse{SupportedCmdVersions: []uint32{1}}, nil
}

func (s *fakeCmdInfoServer) Calls() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.calls
}

// fakeNotifyManager records every StartDomainNotify call and returns the queued errors in order.
type fakeNotifyManager struct {
	errs      []error
	stopChans []<-chan struct{}
}

func (f *fakeNotifyManager) StartDomainNotify(stop <-chan struct{}, _ *v1.VirtualMachineInstance) error {
	f.stopChans = append(f.stopChans, stop)
	if len(f.errs) == 0 {
		return nil
	}
	err := f.errs[0]
	f.errs = f.errs[1:]
	return err
}
