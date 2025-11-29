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

	"kubevirt.io/kubevirt/pkg/testutils"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
)

var _ = Describe("VirtualMachineInstance migration target", func() {
	var _ = Describe("DomainNotifyServerRestarts", func() {
		Context("should establish a notify server pipe", func() {
			var shareDir string
			var serverStopChan chan struct{}
			var serverIsStoppedChan chan struct{}
			var stoppedServer bool
			var domainPipeStopChan chan struct{}
			var stoppedPipe bool
			var eventChan chan watch.Event
			var client *notifyclient.Notifier
			var recorder *record.FakeRecorder
			var vmiStore cache.Store

			BeforeEach(func() {
				var err error
				serverStopChan = make(chan struct{})
				domainPipeStopChan = make(chan struct{})
				serverIsStoppedChan = make(chan struct{})
				eventChan = make(chan watch.Event, 100)
				stoppedServer = false
				stoppedPipe = false
				shareDir, err = os.MkdirTemp("", "kubevirt-share")
				Expect(err).ToNot(HaveOccurred())

				recorder = record.NewFakeRecorder(10)
				recorder.IncludeObject = true
				vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
				vmiStore = vmiInformer.GetStore()

				go func(serverIsStoppedChan chan struct{}) {
					notifyserver.RunServer(shareDir, serverStopChan, eventChan, recorder, vmiStore)
					close(serverIsStoppedChan)
				}(serverIsStoppedChan)

				time.Sleep(3)
			})

			AfterEach(func() {
				if stoppedServer == false {
					close(serverStopChan)
				}
				if stoppedPipe == false {
					close(domainPipeStopChan)
				}
				client.Close()
				os.RemoveAll(shareDir)
			})

			It("should get notify events", func() {
				vmi := api2.NewMinimalVMI("fake-vmi")
				vmi.UID = "4321"
				vmiStore.Add(vmi)

				eventType := "Normal"
				eventReason := "fooReason"
				eventMessage := "barMessage"

				pipePath := filepath.Join(shareDir, "client_path", "domain-notify-pipe.sock")
				pipeDir := filepath.Join(shareDir, "client_path")
				err := os.MkdirAll(pipeDir, 0755)
				Expect(err).ToNot(HaveOccurred())

				listener, err := net.Listen("unix", pipePath)
				Expect(err).ToNot(HaveOccurred())

				handleDomainNotifyPipe(domainPipeStopChan, listener, shareDir, vmi)
				time.Sleep(1)

				client = notifyclient.NewNotifier(pipeDir)

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

				pipePath := filepath.Join(shareDir, "client_path", "domain-notify-pipe.sock")
				pipeDir := filepath.Join(shareDir, "client_path")
				err := os.MkdirAll(pipeDir, 0755)
				Expect(err).ToNot(HaveOccurred())

				// Client should fail when pipe is offline
				client = notifyclient.NewNotifier(pipeDir)

				client.SetCustomTimeouts(1*time.Second, 1*time.Second, 3*time.Second)

				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).To(HaveOccurred())

				// Client should automatically come online when pipe is established
				listener, err := net.Listen("unix", pipePath)
				Expect(err).ToNot(HaveOccurred())

				handleDomainNotifyPipe(domainPipeStopChan, listener, shareDir, vmi)
				time.Sleep(1)

				// Expect the client to reconnect and succeed despite initial failure
				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should be resilient to notify server restarts", func() {
				vmi := api2.NewMinimalVMI("fake-vmi")
				vmi.UID = "4321"
				vmiStore.Add(vmi)

				eventType := "Normal"
				eventReason := "fooReason"
				eventMessage := "barMessage"

				pipePath := filepath.Join(shareDir, "client_path", "domain-notify-pipe.sock")
				pipeDir := filepath.Join(shareDir, "client_path")
				err := os.MkdirAll(pipeDir, 0755)
				Expect(err).ToNot(HaveOccurred())

				listener, err := net.Listen("unix", pipePath)
				Expect(err).ToNot(HaveOccurred())

				handleDomainNotifyPipe(domainPipeStopChan, listener, shareDir, vmi)
				time.Sleep(1)

				client = notifyclient.NewNotifier(pipeDir)

				for i := 1; i < 5; i++ {
					// close and wait for server to stop
					close(serverStopChan)
					<-serverIsStoppedChan

					client.SetCustomTimeouts(1*time.Second, 1*time.Second, 1*time.Second)
					// Expect a client error to occur here because the server is down
					err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
					Expect(err).To(HaveOccurred())

					// Restart the server now that it is down.
					serverStopChan = make(chan struct{})
					serverIsStoppedChan = make(chan struct{})
					go func() {
						notifyserver.RunServer(shareDir, serverStopChan, eventChan, recorder, vmiStore)
						close(serverIsStoppedChan)
					}()

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

	Describe("LauncherClientInfo Close", func() {
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
			for i := 0; i < 5; i++ {
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

			for i := 0; i < 5; i++ {
				<-done
			}
		})
	})
})
