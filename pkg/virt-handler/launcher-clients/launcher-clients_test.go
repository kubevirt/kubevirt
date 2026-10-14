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

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
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

var _ = Describe("CloseLauncherClient", func() {
	var manager *launcherClientsManager

	BeforeEach(func() {
		virtcache.InitializeGhostRecordCache(virtcache.NewIterableCheckpointManager(GinkgoT().TempDir(), GinkgoT().TempDir()))
		manager = NewLauncherClientsManager(GinkgoT().TempDir(), nil).(*launcherClientsManager)
	})

	It("should remove the ghost record and client entry of the closed incarnation", func() {
		vmi := libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace("default"), libvmi.WithUID("uid-a"))
		Expect(virtcache.GhostRecordGlobalStore.Add(vmi.Namespace, vmi.Name, "/tmp/socket-a", vmi.UID)).To(Succeed())
		stopChan := make(chan struct{})
		manager.launcherClients.Store(vmi.UID, &virtcache.LauncherClientInfo{DomainPipeStopChan: stopChan})

		Expect(manager.CloseLauncherClient(vmi)).To(Succeed())

		Expect(virtcache.GhostRecordGlobalStore.Exists(vmi.Namespace, vmi.Name)).To(BeFalse())
		_, exists := manager.launcherClients.Load(vmi.UID)
		Expect(exists).To(BeFalse())
		Expect(stopChan).To(BeClosed())
	})

	It("should keep the ghost record of a newer incarnation with the same name", func() {
		older := libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace("default"), libvmi.WithUID("uid-a"))
		newer := libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace("default"), libvmi.WithUID("uid-b"))
		Expect(virtcache.GhostRecordGlobalStore.Add(newer.Namespace, newer.Name, "/tmp/socket-b", newer.UID)).To(Succeed())

		Expect(manager.CloseLauncherClient(older)).To(Succeed())

		record, exists := virtcache.GhostRecordGlobalStore.Get(newer.Namespace, newer.Name)
		Expect(exists).To(BeTrue())
		Expect(record.UID).To(Equal(newer.UID))
	})

	It("should not touch ghost records when the VMI has no UID", func() {
		vmi := libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace("default"))
		Expect(virtcache.GhostRecordGlobalStore.Add(vmi.Namespace, vmi.Name, "/tmp/socket-a", "uid-a")).To(Succeed())

		Expect(manager.CloseLauncherClient(vmi)).To(Succeed())

		Expect(virtcache.GhostRecordGlobalStore.Exists(vmi.Namespace, vmi.Name)).To(BeTrue())
	})

	It("should return a checkpoint deletion error and keep the ghost record", func() {
		checkpointDir := GinkgoT().TempDir()
		virtcache.InitializeGhostRecordCache(virtcache.NewIterableCheckpointManager(checkpointDir, GinkgoT().TempDir()))
		vmi := libvmi.New(libvmi.WithName("testvmi"), libvmi.WithNamespace("default"), libvmi.WithUID("uid-a"))
		Expect(virtcache.GhostRecordGlobalStore.Add(vmi.Namespace, vmi.Name, "/tmp/socket-a", vmi.UID)).To(Succeed())
		checkpointPath := filepath.Join(checkpointDir, "uid-a")
		Expect(os.Remove(checkpointPath)).To(Succeed())
		// A non-empty directory makes os.Remove fail with an error other than ENOENT.
		Expect(os.MkdirAll(filepath.Join(checkpointPath, "child"), 0755)).To(Succeed())

		Expect(manager.CloseLauncherClient(vmi)).To(HaveOccurred())

		Expect(virtcache.GhostRecordGlobalStore.Exists(vmi.Namespace, vmi.Name)).To(BeTrue())
	})
})
