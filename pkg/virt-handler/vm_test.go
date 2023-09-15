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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virthandler

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/pkg/safepath"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"

	api2 "kubevirt.io/client-go/api"

	netcache "kubevirt.io/kubevirt/pkg/network/cache"
	neterrors "kubevirt.io/kubevirt/pkg/network/errors"
	"kubevirt.io/kubevirt/pkg/util"
	container_disk "kubevirt.io/kubevirt/pkg/virt-handler/container-disk"
	hotplug_volume "kubevirt.io/kubevirt/pkg/virt-handler/hotplug-disk"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/certificates"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	notifyserver "kubevirt.io/kubevirt/pkg/virt-handler/notify-server"
	notifyclient "kubevirt.io/kubevirt/pkg/virt-launcher/notify-client"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/precond"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	migrationproxy "kubevirt.io/kubevirt/pkg/virt-handler/migration-proxy"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

var _ = Describe("VirtualMachineInstance", func() {
	var client *cmdclient.MockLauncherClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var virtClient *kubecli.MockKubevirtClient
	var clientTest *fake.Clientset

	var ctrl *gomock.Controller
	var controller *VirtualMachineController
	var vmiSource *framework.FakeControllerSource
	var vmiSourceInformer cache.SharedIndexInformer
	var vmiTargetInformer cache.SharedIndexInformer
	var domainSource *framework.FakeControllerSource
	var domainInformer cache.SharedIndexInformer
	var gracefulShutdownInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue
	var mockWatchdog *MockWatchdog
	var mockGracefulShutdown *MockGracefulShutdown
	var mockIsolationDetector *isolation.MockPodIsolationDetector
	var mockIsolationResult *isolation.MockIsolationResult
	var mockContainerDiskMounter *container_disk.MockMounter
	var mockHotplugVolumeMounter *hotplug_volume.MockVolumeMounter

	var vmiFeeder *testutils.VirtualMachineFeeder
	var domainFeeder *testutils.DomainFeeder

	var recorder *record.FakeRecorder

	var err error
	var shareDir string
	var privateDir string
	var vmiShareDir string
	var podsDir string
	var sockFile string
	var ghostCacheDir string
	var vmiTestUUID types.UID
	var podTestUUID types.UID
	var stop chan struct{}
	var wg *sync.WaitGroup
	var eventChan chan watch.Event

	var host string

	var certDir string

	BeforeEach(func() {
		diskutils.MockDefaultOwnershipManager()

		wg = &sync.WaitGroup{}
		stop = make(chan struct{})
		eventChan = make(chan watch.Event, 100)
		shareDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		privateDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		podsDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		certDir, err = os.MkdirTemp("", "migrationproxytest")
		Expect(err).ToNot(HaveOccurred())
		vmiShareDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())
		ghostCacheDir, err = os.MkdirTemp("", "")
		Expect(err).ToNot(HaveOccurred())

		err = virtcache.InitializeGhostRecordCache(ghostCacheDir)
		Expect(err).ToNot(HaveOccurred())

		os.MkdirAll(filepath.Join(vmiShareDir, "var", "run", "kubevirt"), 0755)

		cmdclient.SetLegacyBaseDir(shareDir)
		cmdclient.SetPodsBaseDir(podsDir)

		store, err := certificates.GenerateSelfSignedCert(certDir, "test", "test")

		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			GetCertificate: func(info *tls.ClientHelloInfo) (certificate *tls.Certificate, e error) {
				return store.Current()
			},
		}
		Expect(err).ToNot(HaveOccurred())

		host = "master"
		podIpAddress := "10.10.10.10"

		Expect(err).ToNot(HaveOccurred())

		vmiSourceInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		vmiTargetInformer, _ = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		domainInformer, domainSource = testutils.NewFakeInformerFor(&api.Domain{})
		gracefulShutdownInformer, _ = testutils.NewFakeInformerFor(&api.Domain{})
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		clientTest = fake.NewSimpleClientset()
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().CoreV1().Return(clientTest.CoreV1()).AnyTimes()
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()

		mockWatchdog = &MockWatchdog{shareDir}
		mockGracefulShutdown = &MockGracefulShutdown{shareDir}
		config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		Expect(os.MkdirAll(filepath.Join(vmiShareDir, "dev"), 0755)).To(Succeed())
		f, err := os.OpenFile(filepath.Join(vmiShareDir, "dev", "kvm"), os.O_CREATE, 0755)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)
		mockIsolationResult.EXPECT().Pid().Return(1).AnyTimes()
		rootDir, err := safepath.JoinAndResolveWithRelativeRoot(vmiShareDir)
		Expect(err).ToNot(HaveOccurred())
		mockIsolationResult.EXPECT().MountRoot().Return(rootDir, nil).AnyTimes()

		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationDetector.EXPECT().Detect(gomock.Any()).Return(mockIsolationResult, nil).AnyTimes()
		mockIsolationDetector.EXPECT().AdjustResources(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		mockContainerDiskMounter = container_disk.NewMockMounter(ctrl)
		mockHotplugVolumeMounter = hotplug_volume.NewMockVolumeMounter(ctrl)

		migrationProxy := migrationproxy.NewMigrationProxyManager(tlsConfig, tlsConfig, config)
		fakeDownwardMetricsManager := newFakeManager()

		controller, _ = NewController(recorder,
			virtClient,
			host,
			podIpAddress,
			shareDir,
			privateDir,
			podsDir,
			vmiSourceInformer,
			vmiTargetInformer,
			domainInformer,
			gracefulShutdownInformer,
			1,
			10,
			config,
			mockIsolationDetector,
			migrationProxy,
			fakeDownwardMetricsManager,
			nil,
			"",
		)
		controller.hotplugVolumeMounter = mockHotplugVolumeMounter
		controller.virtLauncherFSRunDirPattern = filepath.Join(shareDir, "%d")

		controller.netConf = &netConfStub{}
		controller.netStat = &netStatStub{}

		vmiTestUUID = uuid.NewUUID()
		podTestUUID = uuid.NewUUID()
		sockFile = cmdclient.SocketFilePathOnHost(string(podTestUUID))
		Expect(os.MkdirAll(filepath.Dir(sockFile), 0755)).To(Succeed())
		f, err = os.Create(sockFile)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
		domainFeeder = testutils.NewDomainFeeder(mockQueue, domainSource)

		wg.Add(5)
		go func() { vmiSourceInformer.Run(stop); wg.Done() }()
		go func() { vmiTargetInformer.Run(stop); wg.Done() }()
		go func() { domainInformer.Run(stop); wg.Done() }()
		go func() { gracefulShutdownInformer.Run(stop); wg.Done() }()
		Expect(cache.WaitForCacheSync(stop, vmiSourceInformer.HasSynced, vmiTargetInformer.HasSynced, domainInformer.HasSynced, gracefulShutdownInformer.HasSynced)).To(BeTrue())

		go func() {
			notifyserver.RunServer(shareDir, stop, eventChan, nil, nil)
			wg.Done()
		}()
		time.Sleep(1 * time.Second)

		client = cmdclient.NewMockLauncherClient(ctrl)
		clientInfo := &virtcache.LauncherClientInfo{
			Client:             client,
			SocketFile:         sockFile,
			DomainPipeStopChan: make(chan struct{}),
			Ready:              true,
		}
		controller.addLauncherClient(vmiTestUUID, clientInfo)

	})

	AfterEach(func() {
		close(stop)
		wg.Wait()
		os.RemoveAll(shareDir)
		os.RemoveAll(privateDir)
		os.RemoveAll(vmiShareDir)
		os.RemoveAll(podsDir)
		os.RemoveAll(certDir)
		os.RemoveAll(ghostCacheDir)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
	})

	expectEvent := func(substring string, shouldExist bool) {
		found := false
		done := false
		for found == false && done == false {
			select {
			case event := <-recorder.Events:
				if strings.Contains(event, substring) {
					found = true
				}
			default:
				done = true
			}
		}
		Expect(found).To(Equal(shouldExist))
	}

	initGracePeriodHelper := func(gracePeriod int64, vmi *v1.VirtualMachineInstance, dom *api.Domain) {
		vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
		dom.Spec.Features = &api.Features{
			ACPI: &api.FeatureEnabled{},
		}
		dom.Spec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{}
		dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = gracePeriod
	}

	Context("VirtualMachineInstance controller gets informed about a Domain change through the Domain controller", func() {
		BeforeEach(func() {
			diskutils.MockDefaultOwnershipManager()
		})

		It("should delete non-running Domains if no cluster wide equivalent and no grace period info exists", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().DeleteDomain(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			controller.Execute()
			testutils.ExpectEvent(recorder, VMISignalDeletion)
		})

		It("should delete running Domains if no cluster wide equivalent exists and no grace period info exists", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStopping)
		})

		It("should handle cleanup of legacy graceful shutdown and watchdog files", func() {

			name := "testvmi"
			namespace := "default"
			uid := "1234"

			legacyMockSockFile := filepath.Join(shareDir, "sockets", uid+"_sock")
			clientInfo := &virtcache.LauncherClientInfo{
				Client:             client,
				SocketFile:         legacyMockSockFile,
				DomainPipeStopChan: make(chan struct{}),
				Ready:              true,
			}
			controller.addLauncherClient(types.UID(uid), clientInfo)
			err := virtcache.AddGhostRecord(namespace, name, legacyMockSockFile, types.UID(uid))
			Expect(err).ToNot(HaveOccurred())

			os.MkdirAll(filepath.Dir(legacyMockSockFile), 0755)
			os.MkdirAll(filepath.Join(shareDir, "watchdog-files"), 0755)
			os.MkdirAll(filepath.Join(shareDir, "graceful-shutdown-trigger"), 0755)

			watchdogFile := filepath.Join(shareDir, "watchdog-files", namespace+"_"+name)
			gracefulTrigger := filepath.Join(shareDir, "graceful-shutdown-trigger", namespace+"_"+name)

			f, err := os.Create(legacyMockSockFile)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			f, err = os.Create(watchdogFile)
			Expect(err).ToNot(HaveOccurred())
			_, err = f.WriteString(uid)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			f, err = os.Create(gracefulTrigger)
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			exists, err := diskutils.FileExists(filepath.Join(shareDir, "sockets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			mockQueue.Add(namespace + "/" + name)
			mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)
			client.EXPECT().Close()
			controller.Execute()

			exists, err = diskutils.FileExists(filepath.Join(shareDir, "sockets"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = diskutils.FileExists(filepath.Join(shareDir, "graceful-shutdown-trigger"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = diskutils.FileExists(filepath.Join(shareDir, "watchdog-files"))
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = diskutils.FileExists(legacyMockSockFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

			exists, err = diskutils.FileExists(watchdogFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())

			exists, err = diskutils.FileExists(gracefulTrigger)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should not attempt graceful shutdown of Domain if domain is already down.", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Running

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Crashed

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().DeleteDomain(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMISignalDeletion)
		})
		It("should attempt graceful shutdown of Domain if trigger file exists.", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Running

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any())

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(vmi)
			domainFeeder.Add(domain)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIGracefulShutdown)
		})

		It("should attempt graceful shutdown of Domain if no cluster wide equivalent exists", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIGracefulShutdown)
		})

		It("should do nothing if vmi and domain do not match", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = "other uuid"
			oldVMI := api2.NewMinimalVMI("testvmi")
			oldVMI.UID = vmiTestUUID
			domain := api.NewMinimalDomainWithUUID("testvmi", oldVMI.UID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(oldVMI)
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(oldVMI))
			Expect(errors.Is(err, os.ErrNotExist)).To(BeFalse())
		})

		It("should silently retry if the command socket is not yet ready", func() {
			vmi := NewScheduledVMI(vmiTestUUID, "notexisingpoduid", host)
			// the socket dir must exist, to not go immediately to failed
			sockFile = cmdclient.SocketFilePathOnHost("notexisingpoduid")
			Expect(os.MkdirAll(filepath.Dir(sockFile), 0755)).To(Succeed())

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Scheduled))
			})

			//Did not initialize yet
			clientInfo := &virtcache.LauncherClientInfo{
				DomainPipeStopChan:  make(chan struct{}),
				Ready:               false,
				NotInitializedSince: time.Now().Add(-1 * time.Minute),
			}
			controller.addLauncherClient(vmi.UID, clientInfo)

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			Expect(mockQueue.GetAddAfterEnqueueCount()).To(BeNumerically(">", 1))
		})

		It("should fail if the command socket is not ready after the suppress timeout of three minutes", func() {
			vmi := NewScheduledVMI(vmiTestUUID, "notexisingpoduid", host)
			// the socket dir must exist, to not go immediately to failed
			sockFile = cmdclient.SocketFilePathOnHost("notexisingpoduid")
			Expect(os.MkdirAll(filepath.Dir(sockFile), 0755)).To(Succeed())

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})

			//Did not initialize yet
			clientInfo := &virtcache.LauncherClientInfo{
				DomainPipeStopChan:  make(chan struct{}),
				Ready:               false,
				NotInitializedSince: time.Now().Add(-4 * time.Minute),
			}
			controller.addLauncherClient(vmi.UID, clientInfo)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMICrashed)
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			Expect(mockQueue.GetAddAfterEnqueueCount()).To(Equal(0))
		})

		It("should cleanup if vmi and domain do not match and watchdog is expired", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			oldVMI := api2.NewMinimalVMI("testvmi")
			oldVMI.UID = "other uuid"
			domain := api.NewMinimalDomainWithUUID("testvmi", "other uuid")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(oldVMI)
			// the domain is dead because the watchdog is expired
			mockWatchdog.Expire(oldVMI)

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)
			mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(oldVMI))
			Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
		})

		It("should cleanup if vmi is finalized and domain does not exist", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Succeeded

			mockWatchdog.CreateFile(vmi)

			vmiFeeder.Add(vmi)
			mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)
			client.EXPECT().Close()
			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(vmi))
			Expect(errors.Is(err, os.ErrNotExist)).To(BeTrue())
		})

		It("should do final cleanup if vmi is being deleted and not finalized", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Scheduled
			now := metav1.Time{Time: time.Now()}
			vmi.DeletionTimestamp = &now
			vmi.Status.MigrationMethod = v1.LiveMigration
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)

			vmiFeeder.Add(vmi)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIDefined)
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(vmi))
			Expect(errors.Is(err, os.ErrNotExist)).To(BeFalse())
		})

		It("should attempt force terminate Domain if grace period expires", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			metav1.Now()
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix()-3, 0)}
			domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now

			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStopping)
		})

		It("should immediately kill domain with grace period of 0", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID

			initGracePeriodHelper(0, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)
			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStopping)
		})

		It("should re-enqueue if the Key is unparseable", func() {
			Expect(mockQueue.Len()).Should(Equal(0))
			mockQueue.Add("a/b/c/d/e")
			controller.Execute()
			Expect(mockQueue.NumRequeues("a/b/c/d/e")).To(Equal(1))
		})

		It("should create the Domain if it sees the first time on a new VirtualMachineInstance", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)
			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) {
				Expect(options.VirtualMachineSMBios.Family).To(Equal(virtconfig.SmbiosConfigDefaultFamily))
				Expect(options.VirtualMachineSMBios.Product).To(Equal(virtconfig.SmbiosConfigDefaultProduct))
				Expect(options.VirtualMachineSMBios.Manufacturer).To(Equal(virtconfig.SmbiosConfigDefaultManufacturer))
			})
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, VMIDefined)
		})

		It("should update the qemu machine type on the VMI status", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.OS.Type.Machine = "q35-123"

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Machine = &v1.Machine{Type: "q35-123"}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) {
				Expect(options.VirtualMachineSMBios.Family).To(Equal(virtconfig.SmbiosConfigDefaultFamily))
				Expect(options.VirtualMachineSMBios.Product).To(Equal(virtconfig.SmbiosConfigDefaultProduct))
				Expect(options.VirtualMachineSMBios.Manufacturer).To(Equal(virtconfig.SmbiosConfigDefaultManufacturer))
			})

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).DoAndReturn(func(_ context.Context, obj interface{}) (*v1.VirtualMachineInstance, error) {
				vmi := obj.(*v1.VirtualMachineInstance)
				Expect(vmi.Status.Machine).To(Equal(updatedVMI.Status.Machine))
				return vmi, nil
			})
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
		})

		It("should update from Scheduled to Running, if it sees a running Domain", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi = addActivePods(vmi, podTestUUID, host)

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Phase = v1.Running
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			updatedVMI.Status.MigrationMethod = v1.LiveMigration
			updatedVMI.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, obj interface{}) (*v1.VirtualMachineInstance, error) {
				vmi := obj.(*v1.VirtualMachineInstance)
				Expect(vmi.Status.PhaseTransitionTimestamps).ToNot(BeEmpty())
				updatedVMI.Status.PhaseTransitionTimestamps = vmi.Status.PhaseTransitionTimestamps

				Expect(vmi).To(Equal(updatedVMI))
				return vmi, nil
			})

			node := &k8sv1.Node{
				Status: k8sv1.NodeStatus{
					Addresses: []k8sv1.NodeAddress{
						{
							Type:    k8sv1.NodeInternalIP,
							Address: "127.0.0.1",
						},
					},
				},
			}
			fakeClient := fake.NewSimpleClientset(node).CoreV1()
			virtClient.EXPECT().CoreV1().Return(fakeClient).AnyTimes()

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStarted)
		})

		It("should add guest agent condition when sees the channel connected", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Devices.Channels = []api.Channel{
				{
					Type: "unix",
					Target: &api.ChannelTarget{
						Name:  "org.qemu.guest_agent.0",
						State: "connected",
					},
				},
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
				{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:          v1.VirtualMachineInstanceUnsupportedAgent,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) {
				Expect(options.VirtualMachineSMBios.Family).To(Equal(virtconfig.SmbiosConfigDefaultFamily))
				Expect(options.VirtualMachineSMBios.Product).To(Equal(virtconfig.SmbiosConfigDefaultProduct))
				Expect(options.VirtualMachineSMBios.Manufacturer).To(Equal(virtconfig.SmbiosConfigDefaultManufacturer))
			})
			vmiInterface.EXPECT().Update(context.Background(), NewVMICondMatcher(*updatedVMI))
			client.EXPECT().GetGuestInfo().Return(&v1.VirtualMachineInstanceGuestAgentInfo{}, nil)
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
		})

		It("should maintain unsupported user agent condition when it's already set", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
				{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:          v1.VirtualMachineInstanceUnsupportedAgent,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
			}
			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Devices.Channels = []api.Channel{
				{
					Type: "unix",
					Target: &api.ChannelTarget{
						Name:  "org.qemu.guest_agent.0",
						State: "connected",
					},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) {
				Expect(options.VirtualMachineSMBios.Family).To(Equal(virtconfig.SmbiosConfigDefaultFamily))
				Expect(options.VirtualMachineSMBios.Product).To(Equal(virtconfig.SmbiosConfigDefaultProduct))
				Expect(options.VirtualMachineSMBios.Manufacturer).To(Equal(virtconfig.SmbiosConfigDefaultManufacturer))
			})
			client.EXPECT().GetGuestInfo().Return(&v1.VirtualMachineInstanceGuestAgentInfo{}, nil)
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
		})

		It("should remove guest agent condition when there is no channel connected", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:          v1.VirtualMachineInstanceAgentConnected,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Devices.Channels = []api.Channel{
				{
					Type: "unix",
					Target: &api.ChannelTarget{
						Name:  "org.qemu.guest_agent.0",
						State: "disconnected",
					},
				},
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) {
				Expect(options.VirtualMachineSMBios.Family).To(Equal(virtconfig.SmbiosConfigDefaultFamily))
				Expect(options.VirtualMachineSMBios.Product).To(Equal(virtconfig.SmbiosConfigDefaultProduct))
				Expect(options.VirtualMachineSMBios.Manufacturer).To(Equal(virtconfig.SmbiosConfigDefaultManufacturer))
			})
			vmiInterface.EXPECT().Update(context.Background(), NewVMICondMatcher(*updatedVMI))
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
		})

		It("should add access credential synced condition when credentials report success", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Metadata.KubeVirt.AccessCredential = &api.AccessCredentialMetadata{
				Succeeded: true,
				Message:   "",
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:          v1.VirtualMachineInstanceAccessCredentialsSynchronized,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			vmiInterface.EXPECT().Update(context.Background(), NewVMICondMatcher(*updatedVMI))
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()

			expectEvent(string(v1.AccessCredentialsSyncSuccess), true)
		})

		It("should do nothing if access credential condition already exists", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:          v1.VirtualMachineInstanceAccessCredentialsSynchronized,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Metadata.KubeVirt.AccessCredential = &api.AccessCredentialMetadata{
				Succeeded: true,
				Message:   "",
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
			// should not make another event entry unless something changes
			expectEvent(string(v1.AccessCredentialsSyncSuccess), false)
		})

		It("should update access credential condition if agent disconnects", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:          v1.VirtualMachineInstanceAccessCredentialsSynchronized,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionTrue,
				},
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Metadata.KubeVirt.AccessCredential = &api.AccessCredentialMetadata{
				Succeeded: false,
				Message:   "some message",
			}

			vmiCopy := vmi.DeepCopy()
			vmiCopy.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
				{
					Type:          v1.VirtualMachineInstanceAccessCredentialsSynchronized,
					LastProbeTime: metav1.Now(),
					Status:        k8sv1.ConditionFalse,
					Message:       "some message",
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			vmiInterface.EXPECT().Update(context.Background(), NewVMICondMatcher(*vmiCopy))
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)

			controller.Execute()
			expectEvent(string(v1.AccessCredentialsSyncFailed), true)
		})

		It("should add and remove paused condition", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)

			By("pausing domain")
			domain.Status.Status = api.Paused
			domain.Status.Reason = api.ReasonPausedUser

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
				{
					Type:   v1.VirtualMachineInstancePaused,
					Status: k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
			vmiInterface.EXPECT().Update(context.Background(), NewVMICondMatcher(*updatedVMI))

			controller.Execute()

			By("unpausing domain")
			domain.Status.Status = api.Running
			domain.Status.Reason = ""

			updatedVMI = vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
			vmiInterface.EXPECT().Update(context.Background(), NewVMICondMatcher(*updatedVMI))

			controller.Execute()
		})

		It("should move VirtualMachineInstance from Scheduled to Failed if watchdog file is missing", func() {
			cmdclient.MarkSocketUnresponsive(sockFile)
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Scheduled

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			controller.Execute()
			testutils.ExpectEvent(recorder, VMICrashed)
		})
		It("should move VirtualMachineInstance from Scheduled to Failed if watchdog file is expired", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Scheduled

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			time.Sleep(2 * time.Second)
			controller.Execute()
			testutils.ExpectEvent(recorder, VMICrashed)
		})

		It("should move VirtualMachineInstance from Running to Failed if domain does not exist in cache", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Running

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			controller.Execute()
			testutils.ExpectEvent(recorder, VMICrashed)
		})

		It("should move VirtualMachineInstance to Failed if configuring the networks on the virt-launcher fails with critical error", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.ActivePods = map[types.UID]string{podTestUUID: ""}
			vmi.Spec.Networks = []v1.Network{{Name: "foo"}}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "foo"}}

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)
			controller.netConf = &netConfStub{SetupError: &neterrors.CriticalNetworkError{}}

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
			controller.Execute()
			testutils.ExpectEvent(recorder, "failed to configure vmi network:")
			testutils.ExpectEvent(recorder, VMICrashed)
		})

		It("should remove an error condition if a synchronization run succeeds", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceSynchronized,
					Status: k8sv1.ConditionFalse,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			updatedVMI.Status.MigrationMethod = v1.LiveMigration

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance, options *cmdv1.VirtualMachineOptions) {
				Expect(options.VirtualMachineSMBios.Family).To(Equal(virtconfig.SmbiosConfigDefaultFamily))
				Expect(options.VirtualMachineSMBios.Product).To(Equal(virtconfig.SmbiosConfigDefaultProduct))
				Expect(options.VirtualMachineSMBios.Manufacturer).To(Equal(virtconfig.SmbiosConfigDefaultManufacturer))
			})
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
			vmiInterface.EXPECT().Update(context.Background(), updatedVMI)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIDefined)
		})

		Context("reacting to a VMI with a containerDisk", func() {
			BeforeEach(func() {
				controller.containerDiskMounter = mockContainerDiskMounter
			})
			It("should retry silently if a containerDisk is not yet ready", func() {
				vmi := NewScheduledVMIWithContainerDisk(vmiTestUUID, podTestUUID, host)

				mockWatchdog.CreateFile(vmi)
				vmiFeeder.Add(vmi)
				mockContainerDiskMounter.EXPECT().ContainerDisksReady(vmi, gomock.Any()).Return(false, nil)
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).AnyTimes()

				controller.Execute()
				Expect(mockQueue.GetAddAfterEnqueueCount()).To(Equal(1))
				Expect(mockQueue.Len()).To(Equal(0))
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			})

			It("should retry noisy if a containerDisk is not yet ready and the suppress timeout is over", func() {
				vmi := NewScheduledVMIWithContainerDisk(vmiTestUUID, podTestUUID, host)

				mockWatchdog.CreateFile(vmi)
				vmiFeeder.Add(vmi)
				mockContainerDiskMounter.EXPECT().ContainerDisksReady(vmi, gomock.Any()).DoAndReturn(func(vmi *v1.VirtualMachineInstance, notReadySince time.Time) (bool, error) {
					Expect(notReadySince.Before(time.Now())).To(BeTrue())
					return false, fmt.Errorf("out of time")
				})
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).AnyTimes()

				controller.Execute()
				testutils.ExpectEvent(recorder, "out of time")
				Expect(mockQueue.GetAddAfterEnqueueCount()).To(Equal(0))
				Expect(mockQueue.Len()).To(Equal(0))
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))
			})

			It("should continue to mount containerDisks if the containerDisks are ready", func() {
				vmi := NewScheduledVMIWithContainerDisk(vmiTestUUID, podTestUUID, host)

				mockWatchdog.CreateFile(vmi)
				vmiFeeder.Add(vmi)
				mockContainerDiskMounter.EXPECT().ContainerDisksReady(vmi, gomock.Any()).DoAndReturn(func(vmi *v1.VirtualMachineInstance, notReadySince time.Time) (bool, error) {
					Expect(notReadySince.Before(time.Now())).To(BeTrue())
					return true, nil
				})
				mockContainerDiskMounter.EXPECT().MountAndVerify(gomock.Any()).Return(nil, fmt.Errorf("aborting since we only want to reach this point"))
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).AnyTimes()

				controller.Execute()
				testutils.ExpectEvent(recorder, "aborting since we only want to reach this point")
				Expect(mockQueue.GetAddAfterEnqueueCount()).To(Equal(0))
				Expect(mockQueue.Len()).To(Equal(0))
				Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))
			})
		})

		Context("reacting to a VMI with hotplug", func() {
			BeforeEach(func() {
				controller.hotplugVolumeMounter = mockHotplugVolumeMounter
			})

			It("should call mount if VMI is scheduled to run", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Scheduled
				vmiFeeder.Add(vmi)
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any())
				mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
				client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())

				controller.Execute()
				testutils.ExpectEvent(recorder, VMIDefined)
			})

			It("should call mount and unmount if VMI is running", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any())
				mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
				mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
				client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())

				controller.Execute()
			})

			It("should call mount, fail if mount fails", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				vmiInterface.EXPECT().Update(context.Background(), gomock.Any())
				mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(fmt.Errorf("Error"))

				controller.Execute()
				testutils.ExpectEvent(recorder, v1.SyncFailed.String())
			})

			It("should call unmountAll from processVmCleanup", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)
				client.EXPECT().Close()
				controller.processVmCleanup(vmi)
			})
		})

		Context("hotplug status events", func() {
			It("should have hashotplug false without hotplugged volumes", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name: "test",
				})
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeFalse())
			})

			It("should have hashotplug true with hotplugged volumes", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name:   "test",
					Target: "sda",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
				})
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				testutils.ExpectEvent(recorder, VolumeReadyReason)
				Expect(hasHotplug).To(BeTrue())
			})

			DescribeTable("should generate a mount event, when able to move to mount", func(currentPhase v1.VolumePhase) {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "test",
				})
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name:    "test",
					Phase:   currentPhase,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
				})
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, api.Disk{
					Alias:  api.NewUserDefinedAlias("test"),
					Target: api.DiskTarget{},
				})
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeTrue())
				Expect(vmi.Status.VolumeStatus[0].Phase).To(Equal(v1.HotplugVolumeMounted))
				testutils.ExpectEvent(recorder, "Volume test has been mounted in virt-launcher pod")
				By("Calling it again with updated status, no new events are generated")
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				controller.updateVolumeStatusesFromDomain(vmi, domain)
			},
				Entry("When current phase is bound", v1.VolumeBound),
				Entry("When current phase is pending", v1.VolumePending),
				Entry("When current phase is bound for hotplug volume", v1.HotplugVolumeAttachedToNode),
			)

			DescribeTable("should generate an unmount event, when able to move to unmount", func(currentPhase v1.VolumePhase) {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name:    "test",
					Phase:   currentPhase,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
				})
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, api.Disk{
					Alias:  api.NewUserDefinedAlias("test"),
					Target: api.DiskTarget{},
				})
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(false, nil)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeTrue())
				Expect(vmi.Status.VolumeStatus[0].Phase).To(Equal(v1.HotplugVolumeUnMounted))
				testutils.ExpectEvent(recorder, "Volume test has been unmounted from virt-launcher pod")
				By("Calling it again with updated status, no new events are generated")
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(false, nil)
				controller.updateVolumeStatusesFromDomain(vmi, domain)
			},
				Entry("When current phase is bound", v1.VolumeReady),
				Entry("When current phase is pending", v1.HotplugVolumeMounted),
				Entry("When current phase is bound for hotplug volume", v1.HotplugVolumeAttachedToNode),
			)

			It("Should generate a ready event when target is assigned", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "test",
				})
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name:    "test",
					Phase:   v1.HotplugVolumeMounted,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
				})
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				domain.Spec.Devices.Disks = append(domain.Spec.Devices.Disks, api.Disk{
					Alias: api.NewUserDefinedAlias("test"),
					Target: api.DiskTarget{
						Device: "vdbbb",
					},
				})
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeTrue())
				Expect(vmi.Status.VolumeStatus[0].Phase).To(Equal(v1.VolumeReady))
				Expect(vmi.Status.VolumeStatus[0].Target).To(Equal("vdbbb"))
				testutils.ExpectEvent(recorder, "Successfully attach hotplugged volume test to VM")
				By("Calling it again with updated status, no new events are generated")
				controller.updateVolumeStatusesFromDomain(vmi, domain)
			})

			It("generateEventsForVolumeStatusChange should not modify arguments", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "test",
				})
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name:    "test",
					Phase:   v1.HotplugVolumeMounted,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
				})
				testStatusMap := make(map[string]v1.VolumeStatus)
				testStatusMap["test"] = vmi.Status.VolumeStatus[0]
				testStatusMap["test2"] = vmi.Status.VolumeStatus[0]
				testStatusMap["test3"] = vmi.Status.VolumeStatus[0]
				Expect(testStatusMap).To(HaveLen(3))
				controller.generateEventsForVolumeStatusChange(vmi, testStatusMap)
				testutils.ExpectEvent(recorder, "message")
				testutils.ExpectEvent(recorder, "message")
				Expect(testStatusMap).To(HaveLen(3))
			})
		})

		Context("memory dump status events", func() {
			It("Should trigger memory dump and generate InProgress event once mounted", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "test",
				})
				volumeStatus := v1.VolumeStatus{
					Name:    "test",
					Phase:   v1.HotplugVolumeMounted,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
					MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
						ClaimName: "test",
					},
				}
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, volumeStatus)
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)

				updatedVolumeStatus := *volumeStatus.DeepCopy()
				updatedVolumeStatus.MemoryDumpVolume.TargetFileName = dumpTargetFile(vmi.Name, volumeStatus.Name)
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeTrue())

				Expect(vmi.Status.VolumeStatus[0].Phase).To(Equal(v1.MemoryDumpVolumeInProgress))
				Expect(vmi.Status.VolumeStatus[0].MemoryDumpVolume.TargetFileName).To(Equal(dumpTargetFile(vmi.Name, volumeStatus.Name)))
				testutils.ExpectEvent(recorder, "Memory dump Volume test is attached, getting memory dump")
				By("Calling it again with updated status, no new events are generated as long as memory dump not completed")
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				controller.updateVolumeStatusesFromDomain(vmi, domain)
			})

			It("Should generate memory dump completed event once memory dump completed", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "test",
				})
				volumeStatus := v1.VolumeStatus{
					Name:    "test",
					Phase:   v1.MemoryDumpVolumeInProgress,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
					MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
						ClaimName:      "test",
						TargetFileName: dumpTargetFile(vmi.Name, "test"),
					},
				}
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, volumeStatus)
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				now := metav1.Now()
				domain.Spec.Metadata.KubeVirt.MemoryDump = &api.MemoryDumpMetadata{
					FileName:       dumpTargetFile(vmi.Name, "test"),
					StartTimestamp: &now,
					EndTimestamp:   &now,
					Completed:      true,
				}
				domain.Status.Status = api.Running
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)

				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeTrue())

				Expect(vmi.Status.VolumeStatus[0].Phase).To(Equal(v1.MemoryDumpVolumeCompleted))
				Expect(vmi.Status.VolumeStatus[0].MemoryDumpVolume.StartTimestamp).ToNot(BeNil())
				Expect(vmi.Status.VolumeStatus[0].MemoryDumpVolume.EndTimestamp).ToNot(BeNil())
				testutils.ExpectEvent(recorder, "Memory dump to Volume test has completed successfully")
				By("Calling it again with updated status, no new events are generated as long as memory dump not completed")
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				controller.updateVolumeStatusesFromDomain(vmi, domain)
			})

			It("Should generate memory dump failed event if memory dump failed", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.UID = vmiTestUUID
				vmi.Status.Phase = v1.Running
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: "test",
				})
				volumeStatus := v1.VolumeStatus{
					Name:    "test",
					Phase:   v1.MemoryDumpVolumeInProgress,
					Reason:  "reason",
					Message: "message",
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod",
						AttachPodUID:  "1234",
					},
					MemoryDumpVolume: &v1.DomainMemoryDumpInfo{
						ClaimName:      "test",
						TargetFileName: dumpTargetFile(vmi.Name, "test"),
					},
				}
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, volumeStatus)
				domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
				domain.Status.Status = api.Running
				now := metav1.Now()
				failureReason := "memory dump failed"
				domain.Spec.Metadata.KubeVirt.MemoryDump = &api.MemoryDumpMetadata{
					FileName:       dumpTargetFile(vmi.Name, "test"),
					StartTimestamp: &now,
					EndTimestamp:   &now,
					Failed:         true,
					FailureReason:  failureReason,
				}
				vmiFeeder.Add(vmi)
				domainFeeder.Add(domain)

				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				hasHotplug := controller.updateVolumeStatusesFromDomain(vmi, domain)
				Expect(hasHotplug).To(BeTrue())

				Expect(vmi.Status.VolumeStatus[0].Phase).To(Equal(v1.MemoryDumpVolumeFailed))
				Expect(vmi.Status.VolumeStatus[0].MemoryDumpVolume.StartTimestamp).ToNot(BeNil())
				Expect(vmi.Status.VolumeStatus[0].MemoryDumpVolume.EndTimestamp).ToNot(BeNil())
				testutils.ExpectEvent(recorder, fmt.Sprintf("Memory dump to pvc %s failed: %s", volumeStatus.Name, failureReason))
				By("Calling it again with updated status, no new events are generated as long as memory dump not completed")
				mockHotplugVolumeMounter.EXPECT().IsMounted(vmi, "test", gomock.Any()).Return(true, nil)
				controller.updateVolumeStatusesFromDomain(vmi, domain)
			})

		})

		DescribeTable("should leave the VirtualMachineInstance alone if it is in the final phase", func(phase v1.VirtualMachineInstancePhase) {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Status.Phase = phase
			vmiFeeder.Add(vmi)
			mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)
			controller.Execute()
			// expect no errors and no mock interactions
			Expect(mockQueue.NumRequeues("default/testvmi")).To(Equal(0))
		},
			Entry("succeeded", v1.Succeeded),
			Entry("failed", v1.Failed),
		)

		It("should leave VirtualMachineInstance phase alone if not the current active node", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Status.NodeName = "othernode"

			// no domain would result in a failure, but the NodeName is not
			// equal to controller.host's node, so we know that this node
			// does not own the vmi right now.

			vmiFeeder.Add(vmi)
			controller.Execute()
		})

		It("should prepare migration target", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = "othernode"
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:   host,
				SourceNode:   "othernode",
				MigrationUID: "123",
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			// something has to be listening to the cmd socket
			// for the proxy to work.
			os.MkdirAll(cmdclient.SocketDirectoryOnHost(string(podTestUUID)), os.ModePerm)

			socketFile := cmdclient.SocketFilePathOnHost(string(podTestUUID))
			os.RemoveAll(socketFile)
			socket, err := net.Listen("unix", socketFile)
			Expect(err).NotTo(HaveOccurred())
			defer socket.Close()

			// since a random port is generated, we have to create the proxy
			// here in order to know what port will be in the update.
			err = controller.handleTargetMigrationProxy(vmi)
			Expect(err).NotTo(HaveOccurred())

			destSrcPorts := controller.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
			updatedVmi := vmi.DeepCopy()
			updatedVmi.Status.MigrationState.TargetNodeAddress = controller.migrationIpAddress
			updatedVmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPorts

			client.EXPECT().Ping()
			client.EXPECT().SyncMigrationTarget(vmi, gomock.Any())
			vmiInterface.EXPECT().Update(context.Background(), updatedVmi)
			controller.Execute()
			testutils.ExpectEvent(recorder, VMIMigrationTargetPrepared)
			testutils.ExpectEvent(recorder, "Migration Target is listening")
		})

		It("should signal target pod to early exit on failed migration", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = "othernode"
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:   host,
				SourceNode:   "othernode",
				MigrationUID: "123",
				Failed:       true,
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			client.EXPECT().Ping()
			client.EXPECT().SignalTargetPodCleanup(vmi)
			controller.Execute()
		})

		It("should abort target prep if VMI is deleted", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = "othernode"
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:   host,
				SourceNode:   "othernode",
				MigrationUID: "123",
			}
			now := metav1.Time{Time: time.Now()}
			vmi.DeletionTimestamp = &now
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			// something has to be listening to the cmd socket
			// for the proxy to work.
			os.MkdirAll(cmdclient.SocketDirectoryOnHost(string(podTestUUID)), os.ModePerm)

			socketFile := cmdclient.SocketFilePathOnHost(string(podTestUUID))
			os.RemoveAll(socketFile)
			socket, err := net.Listen("unix", socketFile)
			Expect(err).NotTo(HaveOccurred())
			defer socket.Close()

			// since a random port is generated, we have to create the proxy
			// here in order to know what port will be in the update.
			err = controller.handleTargetMigrationProxy(vmi)
			Expect(err).NotTo(HaveOccurred())

			destSrcPorts := controller.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
			updatedVmi := vmi.DeepCopy()
			updatedVmi.Status.MigrationState.TargetNodeAddress = controller.migrationIpAddress
			updatedVmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPorts
			mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)

			client.EXPECT().Close()
			controller.Execute()
		})

		// handles case where a failed migration to this node has left overs still on local storage
		It("should clean stale clients when preparing migration target", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = "othernode"
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:   host,
				SourceNode:   "othernode",
				MigrationUID: "123",
			}

			stalePodUUID := uuid.NewUUID()
			vmi = addActivePods(vmi, podTestUUID, host)
			vmi = addActivePods(vmi, stalePodUUID, host)

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			// Create stale socket ghost file
			err := virtcache.AddGhostRecord(vmi.Namespace, vmi.Name, "made/up/path", vmi.UID)
			Expect(err).NotTo(HaveOccurred())
			exists := virtcache.HasGhostRecord(vmi.Namespace, vmi.Name)
			Expect(exists).To(BeTrue())

			// Create new socket
			socketFile := cmdclient.SocketFilePathOnHost(string(podTestUUID))
			os.RemoveAll(socketFile)
			socket, err := net.Listen("unix", socketFile)
			Expect(err).NotTo(HaveOccurred())
			defer socket.Close()

			mockHotplugVolumeMounter.EXPECT().UnmountAll(gomock.Any()).Return(nil)

			client.EXPECT().Ping().Return(fmt.Errorf("disconnected"))
			client.EXPECT().Close()

			controller.Execute()

			// ensure cleanup occurred of previous connection
			exists = virtcache.HasGhostRecord(vmi.Namespace, vmi.Name)
			Expect(exists).To(BeFalse())

		})

		It("should migrate vmi once target address is known", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)
			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("0Mi"),
				ProgressTimeout:         150,
				CompletionTimeoutPerGiB: 800,
				UnsafeMigration:         false,
				AllowPostCopy:           false,
			}
			client.EXPECT().MigrateVirtualMachine(vmi, options)
			controller.Execute()
			testutils.ExpectEvent(recorder, VMIMigrating)
		})

		It("should not try to migrate a vmi twice", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			startTimestamp := metav1.Now()
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
				StartTimestamp:                 &startTimestamp,
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				StartTimestamp: &startTimestamp,
				UID:            "123",
			}
			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)
			controller.Execute()
		})

		It("should abort vmi migration vmi when migration object indicates deletion", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				AbortRequested:                 true,
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &now,
			}
			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			client.EXPECT().CancelVirtualMachineMigration(vmi)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any())
			controller.Execute()
			testutils.ExpectEvent(recorder, VMIAbortingMigration)
		})

		It("Handoff domain to other node after completed migration", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetNodeDomainDetected:       true,
				TargetNodeDomainReadyTimestamp: &now,
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Shutoff
			domain.Status.Reason = api.ReasonMigrated

			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &now,
				EndTimestamp:   &now,
				Completed:      true,
			}

			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			vmiUpdated := vmi.DeepCopy()
			vmiUpdated.Status.MigrationState.Completed = true
			vmiUpdated.Status.MigrationTransport = v1.MigrationTransportUnix
			vmiUpdated.Status.MigrationState.StartTimestamp = &now
			vmiUpdated.Status.MigrationState.EndTimestamp = &now
			vmiUpdated.Status.NodeName = "othernode"
			vmiUpdated.Labels[v1.NodeNameLabel] = "othernode"
			vmiUpdated.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			vmiInterface.EXPECT().Update(context.Background(), vmiUpdated)

			controller.Execute()
			testutils.ExpectEvent(recorder, "The VirtualMachineInstance migrated to node")
		})

		It("should apply post-migration operations on guest VM after migration completed", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = "othernode"
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			pastTime := metav1.NewTime(metav1.Now().Add(time.Duration(-10) * time.Second))
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:               host,
				TargetNodeAddress:        "127.0.0.1:12345",
				SourceNode:               "othernode",
				MigrationUID:             "123",
				TargetNodeDomainDetected: false,
				StartTimestamp:           &pastTime,
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &pastTime,
			}

			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			vmiUpdated := vmi.DeepCopy()
			vmiUpdated.Status.MigrationState.TargetNodeDomainDetected = true
			client.EXPECT().Ping().AnyTimes()
			client.EXPECT().FinalizeVirtualMachineMigration(gomock.Any())
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmiObj *v1.VirtualMachineInstance) {

				Expect(vmiObj.Status.MigrationState.TargetNodeDomainReadyTimestamp).ToNot(BeNil())
				Expect(vmiObj.Status.CurrentCPUTopology).To(BeNil())
				vmiUpdated.Status.MigrationState.TargetNodeDomainReadyTimestamp = vmiObj.Status.MigrationState.TargetNodeDomainReadyTimestamp

				Expect(vmiObj).To(Equal(vmiUpdated))
			})

			controller.Execute()
		})

		It("should hotplug CPU in post-migration when target pod has the required conditions", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = "othernode"
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
			pastTime := metav1.NewTime(metav1.Now().Add(time.Duration(-10) * time.Second))
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:               host,
				TargetNodeAddress:        "127.0.0.1:12345",
				SourceNode:               "othernode",
				MigrationUID:             "123",
				TargetNodeDomainDetected: false,
				StartTimestamp:           &pastTime,
			}
			cpuTopology := &v1.CPU{
				Sockets: 1,
				Cores:   1,
				Threads: 1,
			}

			vmiConditions := virtcontroller.NewVirtualMachineInstanceConditionManager()
			vmi.Spec.Domain.CPU = cpuTopology

			vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
				Type:   v1.VirtualMachineInstanceVCPUChange,
				Status: k8sv1.ConditionTrue,
			})

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            "123",
				StartTimestamp: &pastTime,
			}

			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			vmiUpdated := vmi.DeepCopy()
			vmiUpdated.Status.MigrationState.TargetNodeDomainDetected = true

			client.EXPECT().Ping().AnyTimes()
			client.EXPECT().FinalizeVirtualMachineMigration(gomock.Any())
			client.EXPECT().SyncVirtualMachineCPUs(gomock.Any(), gomock.Any())
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmiObj *v1.VirtualMachineInstance) {

				Expect(vmiObj.Status.MigrationState.TargetNodeDomainReadyTimestamp).ToNot(BeNil())
				Expect(vmiObj.Status.CurrentCPUTopology).NotTo(BeNil())
				Expect(vmiConditions.HasCondition(vmiObj, v1.VirtualMachineInstanceVCPUChange)).To(BeFalse())
				vmiUpdated.Status.MigrationState.TargetNodeDomainReadyTimestamp = vmiObj.Status.MigrationState.TargetNodeDomainReadyTimestamp
				vmiUpdated.Status.CurrentCPUTopology = vmiObj.Status.CurrentCPUTopology
				vmiConditions.RemoveCondition(vmiUpdated, v1.VirtualMachineInstanceVCPUChange)

				Expect(vmiObj).To(Equal(vmiUpdated))
			})

			controller.Execute()
		})
	})

	It("should always remove the VirtualMachineInstanceVCPUChange condition even if hotplug CPU has failed", func() {
		vmi := api2.NewMinimalVMI("testvmi")
		vmi.UID = vmiTestUUID
		vmi.ObjectMeta.ResourceVersion = "1"
		vmi.Status.Phase = v1.Running
		vmi.Labels = make(map[string]string)
		vmi.Status.NodeName = "othernode"
		vmi.Labels[v1.MigrationTargetNodeNameLabel] = host
		pastTime := metav1.NewTime(metav1.Now().Add(time.Duration(-10) * time.Second))
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetNode:               host,
			TargetNodeAddress:        "127.0.0.1:12345",
			SourceNode:               "othernode",
			MigrationUID:             "123",
			TargetNodeDomainDetected: false,
			StartTimestamp:           &pastTime,
		}

		cpuTopology := &v1.CPU{
			Sockets: 1,
			Cores:   1,
			Threads: 1,
		}

		vmiConditions := virtcontroller.NewVirtualMachineInstanceConditionManager()
		vmi.Spec.Domain.CPU = cpuTopology

		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:   v1.VirtualMachineInstanceVCPUChange,
			Status: k8sv1.ConditionTrue,
		})

		vmi.Status.CurrentCPUTopology = &v1.CPUTopology{
			Cores:   cpuTopology.Cores,
			Sockets: cpuTopology.Sockets,
			Threads: cpuTopology.Threads,
		}

		mockWatchdog.CreateFile(vmi)
		domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
		domain.Status.Status = api.Running

		domain.Spec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
			UID:            "123",
			StartTimestamp: &pastTime,
		}
		domainFeeder.Add(domain)
		vmiFeeder.Add(vmi)

		vmiUpdated := vmi.DeepCopy()
		vmiUpdated.Status.MigrationState.TargetNodeDomainDetected = true

		client.EXPECT().Ping().AnyTimes()
		client.EXPECT().FinalizeVirtualMachineMigration(gomock.Any())
		client.EXPECT().SyncVirtualMachineCPUs(gomock.Any(), gomock.Any()).Return(fmt.Errorf("some error"))
		vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmiObj *v1.VirtualMachineInstance) {

			Expect(vmiObj.Status.MigrationState.TargetNodeDomainReadyTimestamp).ToNot(BeNil())
			Expect(vmiObj.Status.CurrentCPUTopology).NotTo(BeNil())
			vmiUpdated.Status.MigrationState.TargetNodeDomainReadyTimestamp = vmiObj.Status.MigrationState.TargetNodeDomainReadyTimestamp
			vmiConditions.RemoveCondition(vmiUpdated, v1.VirtualMachineInstanceVCPUChange)

			Expect(vmiObj).To(Equal(vmiUpdated))
		})

		controller.Execute()
		testutils.ExpectEvent(recorder, "failed to change vCPUs")
	})

	Context("check if migratable", func() {

		var testBlockPvc *k8sv1.PersistentVolumeClaim

		BeforeEach(func() {
			kubeClient := fake.NewSimpleClientset()
			virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

			// create a test block pvc
			mode := k8sv1.PersistentVolumeBlock
			testBlockPvc = &k8sv1.PersistentVolumeClaim{
				TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Namespace: k8sv1.NamespaceDefault, Name: "testblock"},
				Spec: k8sv1.PersistentVolumeClaimSpec{
					VolumeMode:  &mode,
					AccessModes: []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteMany},
				},
			}

		})
		It("should block migrate non-shared disks ", func() {

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should migrate shared disks without blockMigration flag", func() {

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name: "myvolume",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						AccessModes: testBlockPvc.Spec.AccessModes,
						VolumeMode:  testBlockPvc.Spec.VolumeMode,
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should fail migration for non-shared PVCs", func() {

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
					},
				},
			}

			testBlockPvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name: "myvolume",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						AccessModes: testBlockPvc.Spec.AccessModes,
						VolumeMode:  testBlockPvc.Spec.VolumeMode,
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI: PVC testblock is not shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)")))
		})
		It("should fail migration for non-shared data volume PVCs", func() {

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "testblock",
						},
					},
				},
			}

			testBlockPvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name: "myvolume",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						AccessModes: testBlockPvc.Spec.AccessModes,
						VolumeMode:  testBlockPvc.Spec.VolumeMode,
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI: PVC testblock is not shared, live migration requires that all PVCs must be shared (using ReadWriteMany access mode)")))
		})
		It("should be allowed to migrate a mix of shared and non-shared disks", func() {

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
					},
				},
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
			}

			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name: "myvolume",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						AccessModes: testBlockPvc.Spec.AccessModes,
						VolumeMode:  testBlockPvc.Spec.VolumeMode,
					},
				},
			}
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should be allowed to migrate a mix of non-shared and shared disks", func() {

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name: "myvolume",
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						AccessModes: testBlockPvc.Spec.AccessModes,
						VolumeMode:  testBlockPvc.Spec.VolumeMode,
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should be allowed to live-migrate shared HostDisks ", func() {
			_true := true
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "myvolume",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_true,
						},
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not be allowed to live-migrate shared and non-shared HostDisks ", func() {
			_true := true
			_false := false
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_true,
						},
					},
				},
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume31/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_false,
						},
					},
				},
			}

			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI with non-shared HostDisk")))
		})
		DescribeTable("with host model", func(hostCpuModel string) {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}

			controller.hostCpuModel = hostCpuModel

			err := controller.isHostModelMigratable(vmi)

			if hostCpuModel == "" {
				Expect(err).Should(HaveOccurred())
			} else {
				Expect(err).ShouldNot(HaveOccurred())
			}

		},
			Entry("exist migration should succeed", "Westmere"),
			Entry("don't exist migration should fail", ""),
		)

		It("should not be allowed to live-migrate if the VMI uses virtiofs ", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Filesystems = []v1.Filesystem{
				{
					Name:     "VIRTIOFS",
					Virtiofs: &v1.FilesystemVirtiofs{},
				},
			}

			condition, isBlockMigration := controller.calculateLiveMigrationCondition(vmi)
			Expect(isBlockMigration).To(BeFalse())
			Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceIsMigratable))
			Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
			Expect(condition.Reason).To(Equal(v1.VirtualMachineInstanceReasonVirtIOFSNotMigratable))
		})

		It("should not be allowed to live-migrate if the VMI does not use masquerade to connect to the pod network", func() {
			vmi := api2.NewMinimalVMI("testvmi")

			strategy := v1.EvictionStrategyLiveMigrate
			vmi.Spec.EvictionStrategy = &strategy

			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

			conditionManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
			controller.updateLiveMigrationConditions(vmi, conditionManager)

			testutils.ExpectEvent(recorder, fmt.Sprintf("cannot migrate VMI which does not use masquerade to connect to the pod network or bridge with %s VM annotation", v1.AllowPodBridgeNetworkLiveMigrationAnnotation))
		})
		Context("with AllowLiveMigrationBridgePodNetwork annotation", func() {
			It("should allow to live-migrate if the VMI use bridge to connect to the pod network", func() {
				vmi := api2.NewMinimalVMI("testvmi")

				vmi.Annotations = map[string]string{v1.AllowPodBridgeNetworkLiveMigrationAnnotation: ""}

				strategy := v1.EvictionStrategyLiveMigrate
				vmi.Spec.EvictionStrategy = &strategy

				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
				vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}

				conditionManager := virtcontroller.NewVirtualMachineInstanceConditionManager()
				controller.updateLiveMigrationConditions(vmi, conditionManager)
				Expect(vmi.Status.Conditions).To(ContainElement(v1.VirtualMachineInstanceCondition{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				}))
				Expect(vmi.Status.MigrationMethod).To(Equal(v1.LiveMigration))
			})
		})

		Context("check that migration is not supported when using Host Devices", func() {
			envName := util.ResourceNameToEnvVar(v1.PCIResourcePrefix, "dev1")

			BeforeEach(func() {
				_ = os.Setenv(envName, "0000:81:01.0")
			})

			AfterEach(func() {
				_ = os.Unsetenv(envName)
			})

			It("should not be allowed to live-migrate if the VMI uses PCI host device", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				vmi.Spec.Domain.Devices.HostDevices = []v1.HostDevice{
					{
						Name:       "name1",
						DeviceName: "dev1",
					},
				}

				condition, isBlockMigration := controller.calculateLiveMigrationCondition(vmi)
				Expect(isBlockMigration).To(BeFalse())
				Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceIsMigratable))
				Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
				Expect(condition.Reason).To(Equal(v1.VirtualMachineInstanceReasonHostDeviceNotMigratable))
			})

			It("should not be allowed to live-migrate if the VMI uses PCI GPU", func() {
				envName := util.ResourceNameToEnvVar(v1.PCIResourcePrefix, "dev1")
				_ = os.Setenv(envName, "0000:81:01.0")
				defer func() {
					_ = os.Unsetenv(envName)
				}()

				vmi := api2.NewMinimalVMI("testvmi")
				vmi.Spec.Domain.Devices.GPUs = []v1.GPU{
					{
						Name:       "name1",
						DeviceName: "dev1",
					},
				}

				condition, isBlockMigration := controller.calculateLiveMigrationCondition(vmi)
				Expect(isBlockMigration).To(BeFalse())
				Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceIsMigratable))
				Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
				Expect(condition.Reason).To(Equal(v1.VirtualMachineInstanceReasonHostDeviceNotMigratable))
			})
		})

		It("should not be allowed to live-migrate if the VMI uses SEV", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{},
			}

			condition, isBlockMigration := controller.calculateLiveMigrationCondition(vmi)
			Expect(isBlockMigration).To(BeFalse())
			Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceIsMigratable))
			Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
			Expect(condition.Reason).To(Equal(v1.VirtualMachineInstanceReasonSEVNotMigratable))
		})

		It("should not be allowed to live-migrate if the VMI uses SCSI persistent reservation", func() {
			vmi := api2.NewMinimalVMI("testvmi")

			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
				v1.Disk{
					Name: "scsi0",
					DiskDevice: v1.DiskDevice{
						LUN: &v1.LunTarget{
							Bus:         "scsi",
							Reservation: true,
						},
					},
				})
			condition, isBlockMigration := controller.calculateLiveMigrationCondition(vmi)
			Expect(isBlockMigration).To(BeFalse())
			Expect(condition.Type).To(Equal(v1.VirtualMachineInstanceIsMigratable))
			Expect(condition.Status).To(Equal(k8sv1.ConditionFalse))
			Expect(condition.Reason).To(Equal(v1.VirtualMachineInstanceReasonPRNotMigratable))
		})

		Context("with network configuration", func() {
			It("should block migration for bridge binding assigned to the pod network", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				interface_name := "interface_name"

				vmi.Spec.Networks = []v1.Network{
					{
						Name:          interface_name,
						NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
					},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{
						Name: interface_name,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{},
						},
					},
				}

				err := controller.checkNetworkInterfacesForMigration(vmi)
				Expect(err).To(HaveOccurred())
			})

			It("should not block migration for masquerade binding assigned to the pod network", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				interface_name := "interface_name"

				vmi.Spec.Networks = []v1.Network{
					{
						Name:          interface_name,
						NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
					},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{
						Name: interface_name,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Masquerade: &v1.InterfaceMasquerade{},
						},
					},
				}

				err := controller.checkNetworkInterfacesForMigration(vmi)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not block migration for bridge binding assigned to a multus network", func() {
				vmi := api2.NewMinimalVMI("testvmi")
				interface_name := "interface_name"

				vmi.Spec.Networks = []v1.Network{
					{
						Name:          interface_name,
						NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}},
					},
				}
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{
						Name: interface_name,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{},
						},
					},
				}

				err := controller.checkNetworkInterfacesForMigration(vmi)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("check right migration mode is used when using container disk volume with", func() {
			const volumeAndDiskName = "volume-disk-name"

			getCDRomDisk := func(isReadOnly *bool) v1.Disk {
				return v1.Disk{Name: volumeAndDiskName, DiskDevice: v1.DiskDevice{CDRom: &v1.CDRomTarget{ReadOnly: isReadOnly}}}
			}

			getContainerDiskVolume := func() v1.Volume {
				return v1.Volume{Name: volumeAndDiskName, VolumeSource: v1.VolumeSource{ContainerDisk: &v1.ContainerDiskSource{}}}
			}

			DescribeTable("using", func(disk v1.Disk, volume v1.Volume, migrationMethod v1.VirtualMachineInstanceMigrationMethod) {
				migrationMethodExpected := fmt.Sprintf("Migration method is expected to be %s", migrationMethod)

				vmi := api2.NewMinimalVMI("testvmi")
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{disk}
				vmi.Spec.Volumes = []v1.Volume{volume}

				isBlockMigration, err := controller.checkVolumesForMigration(vmi)
				Expect(err).ShouldNot(HaveOccurred())
				switch migrationMethod {
				case v1.BlockMigration:
					Expect(isBlockMigration).To(BeTrue(), migrationMethodExpected)
				case v1.LiveMigration:
					Expect(isBlockMigration).To(BeFalse(), migrationMethodExpected)
				default:
					Expect(true).To(BeFalse(), "Shouldn't reach here - BlockMigration and LiveMigration are the only migration methods supported")
				}
			},
				Entry("CDROM", getCDRomDisk(nil), getContainerDiskVolume(), v1.LiveMigration),
				Entry("CDROM with read-only=true", getCDRomDisk(pointer.BoolPtr(true)), getContainerDiskVolume(), v1.LiveMigration),
				Entry("CDROM with read-only=false", getCDRomDisk(pointer.BoolPtr(false)), getContainerDiskVolume(), v1.BlockMigration),
			)
		})

		It("HyperV reenlightenment shouldn't be migratable when tsc frequency is missing", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Features = &v1.Features{Hyperv: &v1.FeatureHyperv{Reenlightenment: &v1.FeatureState{Enabled: pointer.Bool(true)}}}
			vmi.Status.TopologyHints = nil

			cond, _ := controller.calculateLiveMigrationCondition(vmi)
			Expect(cond).ToNot(BeNil())
			Expect(cond.Type).To(Equal(v1.VirtualMachineInstanceIsMigratable))
			Expect(cond.Status).To(Equal(k8sv1.ConditionFalse))
		})

	})

	Context("VirtualMachineInstance network status", func() {
		var (
			vmi    *v1.VirtualMachineInstance
			domain *api.Domain
		)

		BeforeEach(func() {
			const vmiName = "testvmi"
			vmi = api2.NewMinimalVMI(vmiName)
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)

			mockWatchdog.CreateFile(vmi)
			domain = api.NewMinimalDomainWithUUID(vmiName, vmiTestUUID)
			domain.Status.Status = api.Running

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			domain.Status.Interfaces = append(domain.Status.Interfaces, api.InterfaceStatus{
				Mac:           "01:00:00:00:00:10",
				Ip:            "10.10.10.10",
				IPs:           []string{"10.10.10.10", "1::1/128"},
				InterfaceName: "nic0",
			})
		})

		It("should report interfaces status", func() {
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, vmiObj *v1.VirtualMachineInstance) {
				Expect(vmiObj.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
					{
						IP:            domain.Status.Interfaces[0].Ip,
						MAC:           domain.Status.Interfaces[0].Mac,
						IPs:           domain.Status.Interfaces[0].IPs,
						InterfaceName: domain.Status.Interfaces[0].InterfaceName,
					},
				}))
			})

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStarted)
		})
	})

	Context("VirtualMachineInstance controller gets informed about changes in a Domain", func() {
		It("should update Guest OS Information in VMI status", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			guestOSName := "TestGuestOS"

			vmi.Status.GuestOSInfo = v1.VirtualMachineInstanceGuestOSInfo{
				Name: guestOSName,
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Status.OSInfo = api.GuestOSInfo{Name: guestOSName}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.GuestOSInfo.Name).To(Equal(guestOSName))
			}).Return(vmi, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStarted)
		})

		It("should update Guest FSFreeze Status in VMI status if fs frozen", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			guestFSFreeezeStatus := "frozen"

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Status.FSFreezeStatus = api.FSFreeze{Status: guestFSFreeezeStatus}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.FSFreezeStatus).To(Equal(guestFSFreeezeStatus))
			}).Return(vmi, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStarted)
		})

		It("should not show Guest FSFreeze Status in VMI status if fs not frozen", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			guestFSFreeezeStatus := api.FSThawed

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Status.FSFreezeStatus = api.FSFreeze{Status: guestFSFreeezeStatus}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.FSFreezeStatus).To(BeEmpty())
			}).Return(vmi, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIStarted)
		})
	})

	Context("VirtualMachineInstance controller gets informed about disk information", func() {
		It("should update existing volume status with target", func() {
			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running

			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume",
					Phase: v1.VolumeReady,
				},
				{
					Name:  "hpvolume",
					Phase: v1.VolumeReady,
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "pod",
						AttachPodUID:  "abcd",
					},
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Spec.Devices.Disks = []api.Disk{
				{
					Device: "disk",
					Type:   "file",
					Source: api.DiskSource{
						File: "/var/run/kubevirt-private/vmi-disks/permvolume1/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    v1.DiskBusVirtio,
						Device: "vda",
					},
					Driver: &api.DiskDriver{
						Cache: "none",
						Name:  "qemu",
						Type:  "raw",
					},
					Alias: api.NewUserDefinedAlias("permvolume"),
				},
				{
					Device: "disk",
					Type:   "file",
					Source: api.DiskSource{
						File: filepath.Join(v1.HotplugDiskDir, "hpvolume1/disk.img"),
					},
					Target: api.DiskTarget{
						Bus:    v1.DiskBusSCSI,
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Cache: "none",
						Name:  "qemu",
						Type:  "raw",
					},
					Alias: api.NewUserDefinedAlias("hpvolume"),
					Address: &api.Address{
						Type:       "drive",
						Bus:        "0",
						Controller: "0",
						Unit:       "0",
					},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			client.EXPECT().SyncVirtualMachine(vmi, gomock.Any())
			mockHotplugVolumeMounter.EXPECT().Unmount(gomock.Any()).Return(nil)
			mockHotplugVolumeMounter.EXPECT().Mount(gomock.Any()).Return(nil)
			vmiInterface.EXPECT().Update(context.Background(), gomock.Any()).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.VolumeStatus).To(HaveLen(2))
				for _, status := range arg.(*v1.VirtualMachineInstance).Status.VolumeStatus {
					if status.Name == "hpvolume" {
						Expect(status.Target).To(Equal("sda"))
					}
					if status.Name == "permvolume" {
						Expect(status.Target).To(Equal("vda"))
					}
				}
			}).Return(vmi, nil)
			controller.Execute()
		})
	})

	Context("Guest Agent Compatibility", func() {
		var vmi *v1.VirtualMachineInstance
		var vmiWithPassword *v1.VirtualMachineInstance
		var vmiWithSSH *v1.VirtualMachineInstance
		var basicCommands []v1.GuestAgentCommandInfo
		var allCommands []v1.GuestAgentCommandInfo
		const agentSupported = "This guest agent is supported"

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{}
			vmiWithPassword = &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					AccessCredentials: []v1.AccessCredential{
						{
							UserPassword: &v1.UserPasswordAccessCredential{
								PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
									QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
								},
							},
						},
					},
				},
			}
			vmiWithSSH = &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					AccessCredentials: []v1.AccessCredential{
						{
							SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
								PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
									QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{},
								},
							},
						},
					},
				},
			}
			basicCommands = []v1.GuestAgentCommandInfo{}
			allCommands = []v1.GuestAgentCommandInfo{}

			for _, cmdName := range RequiredGuestAgentCommands {
				cmd := v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				}
				basicCommands = append(basicCommands, cmd)
				allCommands = append(allCommands, cmd)
			}
			for _, cmdName := range SSHRelatedGuestAgentCommands {
				cmd := v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				}
				allCommands = append(allCommands, cmd)
			}
			for _, cmdName := range PasswordRelatedGuestAgentCommands {
				cmd := v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				}
				allCommands = append(allCommands, cmd)
			}
		})

		It("should succeed with empty VMI and basic commands", func() {
			result, reason := isGuestAgentSupported(vmi, basicCommands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should succeed with empty VMI and all commands", func() {
			result, reason := isGuestAgentSupported(vmi, allCommands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should fail with password and basic commands", func() {
			result, reason := isGuestAgentSupported(vmiWithPassword, basicCommands)
			Expect(result).To(BeFalse())
			Expect(reason).To(Equal("This guest agent doesn't support required password commands"))
		})

		It("should succeed with password and all commands", func() {
			result, reason := isGuestAgentSupported(vmiWithPassword, allCommands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should fail with SSH and basic commands", func() {
			result, reason := isGuestAgentSupported(vmiWithSSH, basicCommands)
			Expect(result).To(BeFalse())
			Expect(reason).To(Equal("This guest agent doesn't support required public key commands"))
		})

		It("should succeed with SSH and all commands", func() {
			result, reason := isGuestAgentSupported(vmiWithSSH, allCommands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})
	})

	Context("Migration options", func() {
		It("multi-threaded qemu migrations", func() {
			const threadCount uint = 123

			vmi := api2.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Running
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:                     "othernode",
				TargetNodeAddress:              "127.0.0.1:12345",
				SourceNode:                     host,
				MigrationUID:                   "123",
				TargetDirectMigrationNodePorts: map[string]int{"49152": 12132},
			}
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceIsMigratable,
					Status: k8sv1.ConditionTrue,
				},
			}
			vmi = addActivePods(vmi, podTestUUID, host)

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)
			vmiFeeder.Add(vmi)

			vmi.Annotations = map[string]string{cmdclient.MultiThreadedQemuMigrationAnnotation: fmt.Sprintf("%d", threadCount)}

			client.EXPECT().MigrateVirtualMachine(gomock.Any(), gomock.Any()).Do(func(_ *v1.VirtualMachineInstance, options *cmdclient.MigrationOptions) {
				Expect(options.ParallelMigrationThreads).ToNot(BeNil())
				Expect(*options.ParallelMigrationThreads).To(Equal(threadCount))
			}).Times(1).Return(nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, VMIMigrating)
		})
	})

})

var _ = Describe("DomainNotifyServerRestarts", func() {
	Context("should establish a notify server pipe", func() {
		var shareDir string
		var err error
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

type MockGracefulShutdown struct {
	baseDir string
}

func (m *MockGracefulShutdown) TriggerShutdown(vmi *v1.VirtualMachineInstance) {
	Expect(os.MkdirAll(filepath.Join(m.baseDir, "graceful-shutdown-trigger"), os.ModePerm)).To(Succeed())

	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	triggerFile := gracefulShutdownTriggerFromNamespaceName(m.baseDir, namespace, domain)
	f, err := os.Create(triggerFile)
	Expect(err).NotTo(HaveOccurred())
	f.Close()
}

type MockWatchdog struct {
	baseDir string
}

func (m *MockWatchdog) CreateFile(vmi *v1.VirtualMachineInstance) {
	Expect(os.MkdirAll(watchdog.WatchdogFileDirectory(m.baseDir), os.ModePerm)).To(Succeed())
	err := watchdog.WatchdogFileUpdate(
		watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name), string(vmi.UID),
	)
	Expect(err).NotTo(HaveOccurred())
}

func (m *MockWatchdog) Expire(vmi *v1.VirtualMachineInstance) {
	watchdogFile := watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name)
	expireDate := time.Now().Add(time.Duration(-1) * time.Minute)
	os.Chtimes(watchdogFile, expireDate, expireDate)
}

func (m *MockWatchdog) File(vmi *v1.VirtualMachineInstance) string {
	return watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name)
}

type vmiCondMatcher struct {
	vmi v1.VirtualMachineInstance
}

func NewVMICondMatcher(vmi v1.VirtualMachineInstance) gomock.Matcher {
	return &vmiCondMatcher{vmi}
}

func (m *vmiCondMatcher) Matches(x interface{}) bool {
	vmi, ok := x.(*v1.VirtualMachineInstance)
	if !ok {
		return false
	}

	if len(m.vmi.Status.Conditions) != len(vmi.Status.Conditions) {
		return false
	}

	for ind := range vmi.Status.Conditions {
		if m.vmi.Status.Conditions[ind].Type != vmi.Status.Conditions[ind].Type {
			return false
		}
	}

	return true
}

func (m *vmiCondMatcher) String() string {
	return "conditions matches on vmis"
}

func addActivePods(vmi *v1.VirtualMachineInstance, podUID types.UID, hostName string) *v1.VirtualMachineInstance {

	if vmi.Status.ActivePods != nil {
		vmi.Status.ActivePods[podUID] = hostName
	} else {
		vmi.Status.ActivePods = map[types.UID]string{
			podUID: hostName,
		}
	}
	return vmi
}

func NewScheduledVMIWithContainerDisk(vmiUID types.UID, podUID types.UID, hostname string) *v1.VirtualMachineInstance {
	vmi := NewScheduledVMI(vmiUID, podUID, hostname)

	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "test",
		VolumeSource: v1.VolumeSource{
			ContainerDisk: testutils.NewFakeContainerDiskSource(),
		},
	})
	return vmi
}
func NewScheduledVMI(vmiUID types.UID, podUID types.UID, hostname string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMI("testvmi")
	vmi.UID = vmiUID
	vmi.ObjectMeta.ResourceVersion = "1"
	vmi.Status.Phase = v1.Scheduled

	vmi = addActivePods(vmi, podUID, hostname)
	return vmi
}

func addNode(client *fake.Clientset, node *k8sv1.Node) {
	client.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
		return true, node, nil
	})
}

type netConfStub struct {
	vmiUID     types.UID
	SetupError error
}

func (nc *netConfStub) Setup(vmi *v1.VirtualMachineInstance, _ []v1.Network, launcherPid int, preSetup func() error) error {
	if nc.SetupError != nil {
		return nc.SetupError
	}
	return nil
}

func (nc *netConfStub) Teardown(vmi *v1.VirtualMachineInstance) error {
	nc.vmiUID = ""
	return nil
}

func (nc *netConfStub) HotUnplugInterfaces(vmi *v1.VirtualMachineInstance) error {
	return nil
}

type netStatStub struct{}

func (ns *netStatStub) UpdateStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if domain == nil || vmi == nil {
		return nil
	}
	if vmi.Status.Interfaces == nil {
		vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{}
	}
	if len(domain.Status.Interfaces) == 0 {
		return nil
	}
	ifaceStatus := v1.VirtualMachineInstanceNetworkInterface{
		IP:            domain.Status.Interfaces[0].Ip,
		MAC:           domain.Status.Interfaces[0].Mac,
		IPs:           domain.Status.Interfaces[0].IPs,
		InterfaceName: domain.Status.Interfaces[0].InterfaceName,
	}
	vmi.Status.Interfaces = append(vmi.Status.Interfaces, ifaceStatus)
	return nil
}

func (ns *netStatStub) Teardown(vmi *v1.VirtualMachineInstance) {}
func (ns *netStatStub) PodInterfaceVolatileDataIsCached(vmi *v1.VirtualMachineInstance, ifaceName string) bool {
	return false
}
func (ns *netStatStub) CachePodInterfaceVolatileData(vmi *v1.VirtualMachineInstance, ifaceName string, data *netcache.PodIfaceCacheData) {
}

func newFakeManager() *fakeManager {
	return &fakeManager{}
}

type fakeManager struct{}

func (*fakeManager) Run(_ chan struct{}) {}
func (*fakeManager) StartServer(_ *v1.VirtualMachineInstance, _ int) error {
	return nil
}
func (*fakeManager) StopServer(_ *v1.VirtualMachineInstance) {}
