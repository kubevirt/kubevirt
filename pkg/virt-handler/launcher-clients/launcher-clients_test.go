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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
)

var _ = Describe("DomainNotifyServer integration", func() {
	var preparePipe = func() (string, string) {
		pipeDir := GinkgoT().TempDir()
		pipePath := filepath.Join(pipeDir, "domain-notify-pipe.sock")
		err := os.MkdirAll(pipeDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		return pipeDir, pipePath
	}

	var (
		ctx       context.Context
		cancel    context.CancelFunc
		notifyDir string
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		var err error
		notifyDir, err = os.MkdirTemp("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

	})

	AfterEach(func() {
		cancel()
		os.RemoveAll(notifyDir)
	})

	startServer := func(recorder *record.FakeRecorder,
		vmiStore cache.Store) (chan struct{}, chan struct{}) {
		serverIsStoppedChan := make(chan struct{})
		serverStopChan := make(chan struct{})

		recorder.IncludeObject = true

		go func() {
			notifyserver.RunServer(notifyDir, serverStopChan, make(chan watch.Event, 100), recorder, vmiStore)
			close(serverIsStoppedChan)
		}()

		return serverIsStoppedChan, serverStopChan
	}

	Context("with running Server", func() {

		var recorder *record.FakeRecorder
		var vmiStore cache.Store

		BeforeEach(func() {
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore = vmiInformer.GetStore()

			recorder = record.NewFakeRecorder(10)
			serverIsStoppedChan, serverStopChan := startServer(recorder, vmiStore)
			time.Sleep(3)
			DeferCleanup(func() {
				close(serverStopChan)
				<-serverIsStoppedChan
			})
		})

		It("should get notify events", func() {
			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir, pipePath := preparePipe()

			listener, err := net.Listen("unix", pipePath)
			Expect(err).ToNot(HaveOccurred())

			handleDomainNotifyPipe(ctx, listener, notifyDir, vmi)
			time.Sleep(1)

			client := notifyclient.NewNotifier(pipeDir)
			defer client.Close()

			err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

			timedOut := false
			timeout := time.After(4 * time.Second)
			select {
			case <-timeout:
				timedOut = true
			case event := <-recorder.Events:
				Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
			}

			Expect(timedOut).To(BeFalse(), "should not time out")
		})

		It("should eventually get notify events once pipe is online", func() {
			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir, pipePath := preparePipe()

			// Client should fail when pipe is offline
			client := notifyclient.NewNotifier(pipeDir)
			defer client.Close()

			client.SetCustomTimeouts(1*time.Second, 1*time.Second, 3*time.Second)

			err := client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).To(HaveOccurred())

			// Client should automatically come online when pipe is established
			listener, err := net.Listen("unix", pipePath)
			Expect(err).ToNot(HaveOccurred())

			handleDomainNotifyPipe(ctx, listener, notifyDir, vmi)
			time.Sleep(1)

			// Expect the client to reconnect and succeed despite initial failure
			err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
			Expect(err).ToNot(HaveOccurred())

		})

	})
	Context("", func() {
		It("should be resilient to notify server restarts", func() {
			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmiStore := vmiInformer.GetStore()

			recorder := record.NewFakeRecorder(10)
			serverIsStoppedChan, serverStopChan := startServer(recorder, vmiStore)

			time.Sleep(3)

			vmi := api2.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir, pipePath := preparePipe()

			listener, err := net.Listen("unix", pipePath)
			Expect(err).ToNot(HaveOccurred())

			handleDomainNotifyPipe(ctx, listener, notifyDir, vmi)
			time.Sleep(1)

			client := notifyclient.NewNotifier(pipeDir)
			defer client.Close()

			for range 4 {
				// close and wait for server to stop
				close(serverStopChan)
				<-serverIsStoppedChan

				client.SetCustomTimeouts(1*time.Second, 1*time.Second, 1*time.Second)
				// Expect a client error to occur here because the server is down
				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).To(HaveOccurred())

				// Restart the server now that it is down.
				serverIsStoppedChan, serverStopChan = startServer(recorder, vmiStore)

				// Expect the client to reconnect and succeed despite server restarts
				client.SetCustomTimeouts(1*time.Second, 1*time.Second, 3*time.Second)
				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).ToNot(HaveOccurred())

				timedOut := false
				timeout := time.After(4 * time.Second)
				select {
				case <-timeout:
					timedOut = true
				case event := <-recorder.Events:
					Expect(event).To(Equal(fmt.Sprintf("%s %s %s involvedObject{kind=VirtualMachineInstance,apiVersion=kubevirt.io/v1}", eventType, eventReason, eventMessage)))
				}
				Expect(timedOut).To(BeFalse(), "should not time out")
			}
		})
	})
})

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
		vmi = api2.NewMinimalVMI("test-vmi")
		vmi.UID = "test-vmi-uid"
		vmi.Status.ActivePods = map[types.UID]string{
			types.UID(podUID): "test-node",
		}

		// Set up socket directory
		podsDir := GinkgoT().TempDir()
		cmdclient.SetPodsBaseDir(podsDir)
		socketPath = cmdclient.SocketFilePathOnHost(podUID)

		// Set environment for FindSocket to find the correct path
		_ = os.Setenv("NODE_NAME", "test-node")
		DeferCleanup(func() {
			_ = os.Unsetenv("NODE_NAME")
		})

		manager = &launcherClientsManager{
			launcherClients: virtcache.LauncherClientInfoByVMI{},
		}
	})

	It("should succeed when Ping succeed without checking the socket", func() {
		// Don't create the socket file, so we can check FindSocket is not performed

		// Create mock client that succeeds on Ping
		ctrl := gomock.NewController(GinkgoT())
		defer ctrl.Finish()
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
			defer ctrl.Finish()
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

			Expect(err).To(HaveOccurred())
			Expect(err).ToNot(MatchError(IrrecoverableError))
			Expect(err.Error()).To(ContainSubstring("connection refused"))
			Expect(client).To(Equal(mockClient))
		})

		It("should check FindSocket and return it as irrecoverable error if it fails", func() {
			// Don't create the socket file, so FindSocket will fail

			// Create mock client that fails on Ping
			ctrl := gomock.NewController(GinkgoT())
			defer ctrl.Finish()
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
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(IrrecoverableError))
			Expect(err.Error()).To(ContainSubstring("No command socket found"))
			Expect(client).To(Equal(mockClient))
		})
	})
})
