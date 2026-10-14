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

package notifymanager

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"
)

var _ = Describe("startDomainNotifyPipe", func() {

	const (
		shareDirName = "share"
		settling     = 200 * time.Millisecond
		timeout      = 5 * time.Second
	)

	var (
		manager    *pipeManager
		detector   *isolation.MockPodIsolationDetector
		isoResult  *isolation.MockIsolationResult
		safeRoot   *safepath.Path
		vmi        *v1.VirtualMachineInstance
		stopChan   chan struct{}
		socketPath string
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())

		// podRoot stands in for the launcher pod's mount root
		podRoot := GinkgoT().TempDir()
		Expect(os.MkdirAll(filepath.Join(podRoot, shareDirName), 0777)).To(Succeed())
		socketPath = filepath.Join(podRoot, shareDirName, "domain-notify-pipe.sock")

		var err error
		safeRoot, err = safepath.JoinAndResolveWithRelativeRoot(podRoot)
		Expect(err).To(Succeed())

		isoResult = isolation.NewMockIsolationResult(ctrl)
		detector = isolation.NewMockPodIsolationDetector(ctrl)

		manager = &pipeManager{
			virtShareDir:         shareDirName,
			podIsolationDetector: detector,
		}

		vmi = api.NewMinimalVMI("testvmi")
		vmi.UID = "1234"

		stopChan = make(chan struct{})
	})

	// expectPipeInjection programs the detector to report an isolation result
	// rooted at the stand-in pod mount root.
	expectPipeInjection := func() {
		detector.EXPECT().Detect(vmi).Return(isoResult, nil)
		isoResult.EXPECT().MountRoot().Return(safeRoot, nil)
	}

	Context("failures", func() {

		It("should wrap the error when isolation detection fails", func() {
			detectErr := errors.New("no such pod")
			detector.EXPECT().Detect(vmi).Return(nil, detectErr)

			err := manager.startDomainNotifyPipe(stopChan, vmi)

			Expect(err).To(MatchError(ContainSubstring("failed to detect isolation for launcher pod when setting up notify pipe")))
			Expect(err).To(MatchError(ContainSubstring(detectErr.Error())))
			Expect(socketPath).ToNot(BeAnExistingFile())
		})

		It("should return the injection error unchanged when the mount root is unavailable", func() {
			mountErr := errors.New("mount root unavailable")
			detector.EXPECT().Detect(vmi).Return(isoResult, nil)
			isoResult.EXPECT().MountRoot().Return(nil, mountErr)

			err := manager.startDomainNotifyPipe(stopChan, vmi)

			Expect(err).To(MatchError(mountErr))
			Expect(socketPath).ToNot(BeAnExistingFile())
		})
	})

	Context("success", func() {

		It("should create the pipe socket in the pod share directory", func() {
			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
			Expect(socketPath).To(BeAnExistingFile())
		})

		It("should accept connections on the pipe socket", func() {
			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())

			conn, err := net.Dial("unix", socketPath)
			Expect(err).To(Succeed())
			Expect(conn.Close()).To(Succeed())
		})
	})

	Context("socket ownership", func() {

		// mockOwnership swaps in a mock ownership manager for the duration of
		// the spec and returns it.
		mockOwnership := func() *diskutils.MockOwnershipManagerInterface {
			original := diskutils.DefaultOwnershipManager
			DeferCleanup(func() {
				diskutils.DefaultOwnershipManager = original
			})
			manager := diskutils.NewMockOwnershipManagerInterface(gomock.NewController(GinkgoT()))
			diskutils.DefaultOwnershipManager = manager
			return manager
		}

		DescribeTable("should hand the pipe socket to the ownership manager for a nonroot VMI",
			func(makeNonRoot func(*v1.VirtualMachineInstance)) {
				makeNonRoot(vmi)

				var owned *safepath.Path
				mockOwnership().EXPECT().SetFileOwnership(gomock.Any()).Times(1).
					Do(func(path *safepath.Path) { owned = path })

				expectPipeInjection()
				DeferCleanup(func() { close(stopChan) })

				Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
				Expect(owned).ToNot(BeNil())
				Expect(owned.String()).To(ContainSubstring("domain-notify-pipe.sock"))
			},
			Entry("marked by a nonzero runtime user", func(vmi *v1.VirtualMachineInstance) {
				vmi.Status.RuntimeUser = 107
			}),
			Entry("marked by the deprecated nonroot annotation", func(vmi *v1.VirtualMachineInstance) {
				vmi.Annotations = map[string]string{v1.DeprecatedNonRootVMIAnnotation: ""}
			}),
		)

		It("should not touch ownership for a root VMI", func() {
			vmi.Status.RuntimeUser = 0
			vmi.Annotations = nil

			mockOwnership().EXPECT().SetFileOwnership(gomock.Any()).Times(0)

			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
		})
	})

	Context("shutdown", func() {

		It("should keep the pipe open while the stop channel is open", func() {
			expectPipeInjection()
			DeferCleanup(func() { close(stopChan) })

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())

			Consistently(socketPath, settling).Should(BeAnExistingFile())
		})

		It("should tear the pipe down when the stop channel is closed", func() {
			expectPipeInjection()

			Expect(manager.startDomainNotifyPipe(stopChan, vmi)).To(Succeed())
			Expect(socketPath).To(BeAnExistingFile())

			close(stopChan)

			Eventually(socketPath, timeout).ShouldNot(BeAnExistingFile())

			_, err := net.Dial("unix", socketPath)
			Expect(err).To(MatchError(syscall.ENOENT))
		})
	})
})

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
			vmi := api.NewMinimalVMI("fake-vmi")
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

			notifyClient := notifyclient.NewOldNotifyClient(pipeDir)
			client := notifyclient.NewNotifier(&notifyClient)
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
			vmi := api.NewMinimalVMI("fake-vmi")
			vmi.UID = "4321"
			vmiStore.Add(vmi)

			eventType := "Normal"
			eventReason := "fooReason"
			eventMessage := "barMessage"

			pipeDir, pipePath := preparePipe()

			// Client should fail when pipe is offline
			notifyClient := notifyclient.NewOldNotifyClientWithCustomTimeouts(pipeDir,
				1*time.Second, 1*time.Second, 3*time.Second,
			)
			client := notifyclient.NewNotifier(&notifyClient)
			defer client.Close()

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

			vmi := api.NewMinimalVMI("fake-vmi")
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

			notifyClient := notifyclient.NewOldNotifyClientWithCustomTimeouts(pipeDir,
				1*time.Second, 1*time.Second, 1*time.Second,
			)
			client := notifyclient.NewNotifier(&notifyClient)
			defer client.Close()

			for range 4 {
				// close and wait for server to stop
				close(serverStopChan)
				<-serverIsStoppedChan

				// Expect a client error to occur here because the server is down
				err = client.SendK8sEvent(vmi, eventType, eventReason, eventMessage)
				Expect(err).To(HaveOccurred())

				// Restart the server now that it is down.
				serverIsStoppedChan, serverStopChan = startServer(recorder, vmiStore)

				// Expect the client to reconnect and succeed despite server restarts
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
