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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"kubevirt.io/kubevirt/pkg/util"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

func StubOutNetworkForTest() {
	network.SetupPodNetworkPhase1 = func(vm *v1.VirtualMachineInstance, pid int) error { return nil }
}

var _ = Describe("VirtualMachineInstance", func() {
	var client *cmdclient.MockLauncherClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var virtClient *kubecli.MockKubevirtClient

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

	var vmiFeeder *testutils.VirtualMachineFeeder
	var domainFeeder *testutils.DomainFeeder

	var recorder record.EventRecorder

	var err error
	var shareDir string
	var privateDir string
	var vmiShareDir string
	var podsDir string
	var ghostCacheDir string
	var vmiTestUUID types.UID
	var podTestUUID types.UID
	var stop chan struct{}
	var eventChan chan watch.Event

	var host string

	log.Log.SetIOWriter(GinkgoWriter)

	var certDir string

	BeforeEach(func() {
		stop = make(chan struct{})
		eventChan = make(chan watch.Event, 100)
		shareDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		privateDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		podsDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		certDir, err = ioutil.TempDir("", "migrationproxytest")
		Expect(err).ToNot(HaveOccurred())
		vmiShareDir, err = ioutil.TempDir("", "")
		Expect(err).ToNot(HaveOccurred())
		ghostCacheDir, err = ioutil.TempDir("", "")
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

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()

		mockWatchdog = &MockWatchdog{shareDir}
		mockGracefulShutdown = &MockGracefulShutdown{shareDir}
		config, _, _, _ := testutils.NewFakeClusterConfig(&k8sv1.ConfigMap{})

		mockIsolationResult = isolation.NewMockIsolationResult(ctrl)
		mockIsolationResult.EXPECT().DoNetNS(gomock.Any()).Return(nil).AnyTimes()
		mockIsolationResult.EXPECT().Pid().Return(1).AnyTimes()
		mockIsolationResult.EXPECT().MountRoot().Return(vmiShareDir).AnyTimes()

		mockIsolationDetector = isolation.NewMockPodIsolationDetector(ctrl)
		mockIsolationDetector.EXPECT().Detect(gomock.Any()).Return(mockIsolationResult, nil).AnyTimes()
		mockIsolationDetector.EXPECT().AdjustResources(gomock.Any()).Return(nil).AnyTimes()

		controller = NewController(recorder,
			virtClient,
			host,
			podIpAddress,
			shareDir,
			privateDir,
			vmiSourceInformer,
			vmiTargetInformer,
			domainInformer,
			gracefulShutdownInformer,
			1,
			10,
			config,
			tlsConfig,
			tlsConfig,
			mockIsolationDetector,
		)

		vmiTestUUID = uuid.NewUUID()
		podTestUUID = uuid.NewUUID()
		sockFile := cmdclient.SocketFilePathOnHost(string(podTestUUID))
		os.MkdirAll(filepath.Dir(sockFile), 0755)
		f, err := os.Create(sockFile)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
		domainFeeder = testutils.NewDomainFeeder(mockQueue, domainSource)

		go vmiSourceInformer.Run(stop)
		go vmiTargetInformer.Run(stop)
		go domainInformer.Run(stop)
		go gracefulShutdownInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmiSourceInformer.HasSynced, vmiTargetInformer.HasSynced, domainInformer.HasSynced, gracefulShutdownInformer.HasSynced)).To(BeTrue())

		StubOutNetworkForTest()

		go func() {
			notifyserver.RunServer(shareDir, stop, eventChan, nil, nil)
		}()
		time.Sleep(1 * time.Second)

		client = cmdclient.NewMockLauncherClient(ctrl)
		domainPipe, err := controller.startDomainNotifyPipe(nil)
		Expect(err).ToNot(HaveOccurred())
		controller.addLauncherClient(vmiTestUUID, client, sockFile, domainPipe)

	})

	AfterEach(func() {
		close(stop)
		ctrl.Finish()
		os.RemoveAll(shareDir)
		os.RemoveAll(privateDir)
		os.RemoveAll(vmiShareDir)
		os.RemoveAll(podsDir)
		os.RemoveAll(certDir)
		os.RemoveAll(ghostCacheDir)
	})

	initGracePeriodHelper := func(gracePeriod int64, vmi *v1.VirtualMachineInstance, dom *api.Domain) {
		vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
		dom.Spec.Features = &api.Features{
			ACPI: &api.FeatureEnabled{},
		}
		dom.Spec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{}
		dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = gracePeriod
	}

	Context("VirtualMachineInstance controller gets informed about a Domain change through the Domain controller", func() {

		It("should delete non-running Domains if no cluster wide equivalent and no grace period info exists", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().DeleteDomain(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			controller.Execute()
		})

		It("should delete running Domains if no cluster wide equivalent exists and no grace period info exists", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))

			controller.Execute()
		})

		It("should handle cleanup of legacy graceful shutdown and watchdog files", func() {

			name := "testvmi"
			namespace := "default"
			uid := "1234"

			legacyMockSockFile := filepath.Join(shareDir, "sockets", uid+"_sock")

			controller.addLauncherClient(types.UID(uid), client, legacyMockSockFile, nil)
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
			vmi := v1.NewMinimalVMI("testvmi")
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
		}, 3)
		It("should attempt graceful shutdown of Domain if trigger file exists.", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Running

			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should attempt graceful shutdown of Domain if no cluster wide equivalent exists", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should do nothing if vmi and domain do not match", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = "other uuid"
			oldVMI := v1.NewMinimalVMI("testvmi")
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
			Expect(os.IsNotExist(err)).To(BeFalse())
		}, 3)

		It("should cleanup if vmi and domain do not match and watchdog is expired", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			oldVMI := v1.NewMinimalVMI("testvmi")
			oldVMI.UID = "other uuid"
			domain := api.NewMinimalDomainWithUUID("testvmi", "other uuid")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(oldVMI)
			// the domain is dead because the watchdog is expired
			mockWatchdog.Expire(oldVMI)
			controller.phase1NetworkSetupCache[oldVMI.UID] = 1
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(oldVMI))
			Expect(os.IsNotExist(err)).To(BeTrue())
			Expect(len(controller.phase1NetworkSetupCache)).To(Equal(0))
		}, 3)

		It("should cleanup if vmi is finalized and domain does not exist", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.Status.Phase = v1.Succeeded

			mockWatchdog.CreateFile(vmi)

			controller.phase1NetworkSetupCache[vmi.UID] = 1
			vmiFeeder.Add(vmi)

			client.EXPECT().Close()
			controller.Execute()
			Expect(mockQueue.Len()).To(Equal(0))
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			_, err := os.Stat(mockWatchdog.File(vmi))
			Expect(os.IsNotExist(err)).To(BeTrue())
			Expect(len(controller.phase1NetworkSetupCache)).To(Equal(0))
		}, 3)

		It("should attempt force terminate Domain if grace period expires", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
		}, 3)

		It("should immediately kill domain with grace period of 0", func() {
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID

			initGracePeriodHelper(0, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceWithUUID(metav1.NamespaceDefault, "testvmi", vmiTestUUID))
			domainFeeder.Add(domain)
			controller.Execute()
		}, 3)

		It("should re-enqueue if the Key is unparseable", func() {
			Expect(mockQueue.Len()).Should(Equal(0))
			mockQueue.Add("a/b/c/d/e")
			controller.Execute()
			Expect(mockQueue.NumRequeues("a/b/c/d/e")).To(Equal(1))
		})

		It("should create the Domain if it sees the first time on a new VirtualMachineInstance", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			controller.Execute()
			Expect(len(controller.phase1NetworkSetupCache)).To(Equal(1))
		})

		It("should update from Scheduled to Running, if it sees a running Domain", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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

			vmiInterface.EXPECT().Update(updatedVMI)

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
		})

		It("should add guest agent condition when sees the channel connected", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			vmiInterface.EXPECT().Update(NewVMICondMatcher(*updatedVMI))
			client.EXPECT().GetGuestInfo().Return(&v1.VirtualMachineInstanceGuestAgentInfo{}, nil)

			controller.Execute()
		})

		It("should maintain unsupported user agent condition when it's already set", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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

			controller.Execute()
		})

		It("should remove guest agent condition when there is no channel connected", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			vmiInterface.EXPECT().Update(NewVMICondMatcher(*updatedVMI))

			controller.Execute()
		})

		It("should add and remove paused condition", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			vmiInterface.EXPECT().Update(NewVMICondMatcher(*updatedVMI))

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
			vmiInterface.EXPECT().Update(NewVMICondMatcher(*updatedVMI))

			controller.Execute()
		})

		It("should move VirtualMachineInstance from Scheduled to Failed if watchdog file is missing", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			controller.Execute()
		})
		It("should move VirtualMachineInstance from Scheduled to Failed if watchdog file is expired", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			time.Sleep(2 * time.Second)
			controller.Execute()
		}, 2)

		It("should move VirtualMachineInstance from Running to Failed if domain does not exist in cache", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running

			vmiFeeder.Add(vmi)
			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(vmi *v1.VirtualMachineInstance) {
				Expect(vmi.Status.Phase).To(Equal(v1.Failed))
			})
			controller.Execute()
		})

		It("should remove an error condition if a synchronization run succeeds", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			vmiInterface.EXPECT().Update(updatedVMI)

			controller.Execute()
		})

		table.DescribeTable("should leave the VirtualMachineInstance alone if it is in the final phase", func(phase v1.VirtualMachineInstancePhase) {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = phase
			vmiFeeder.Add(vmi)
			controller.Execute()
			// expect no errors and no mock interactions
			Expect(mockQueue.NumRequeues("default/testvmi")).To(Equal(0))
		},
			table.Entry("succeeded", v1.Succeeded),
			table.Entry("failed", v1.Failed),
		)

		It("should leave VirtualMachineInstance phase alone if not the current active node", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			vmi := v1.NewMinimalVMI("testvmi")
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
			err = controller.handleMigrationProxy(vmi)
			Expect(err).NotTo(HaveOccurred())
			err = controller.handlePostSyncMigrationProxy(vmi)
			Expect(err).NotTo(HaveOccurred())

			destSrcPorts := controller.migrationProxy.GetTargetListenerPorts(string(vmi.UID))
			fmt.Println("destSrcPorts: ", destSrcPorts)
			updatedVmi := vmi.DeepCopy()
			updatedVmi.Status.MigrationState.TargetNodeAddress = controller.ipAddress
			updatedVmi.Status.MigrationState.TargetDirectMigrationNodePorts = destSrcPorts

			client.EXPECT().Ping()
			client.EXPECT().SyncMigrationTarget(vmi)
			vmiInterface.EXPECT().Update(updatedVmi)
			controller.Execute()
		}, 3)

		It("should migrate vmi once target address is known", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         150,
				CompletionTimeoutPerGiB: 800,
				UnsafeMigration:         false,
			}
			client.EXPECT().MigrateVirtualMachine(vmi, options)
			controller.Execute()
		}, 3)

		It("should abort vmi migration vmi when migration object indicates deletion", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			vmiInterface.EXPECT().Update(gomock.Any())
			controller.Execute()
		}, 3)

		It("Handoff domain to other node after completed migration", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Running
			vmi.Labels = make(map[string]string)
			vmi.Status.NodeName = host
			vmi.Labels[v1.MigrationTargetNodeNameLabel] = "othernode"
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				TargetNode:               "othernode",
				TargetNodeAddress:        "127.0.0.1:12345",
				SourceNode:               host,
				MigrationUID:             "123",
				TargetNodeDomainDetected: true,
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Shutoff
			domain.Status.Reason = api.ReasonMigrated

			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
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
			vmiUpdated.Status.MigrationState.StartTimestamp = &now
			vmiUpdated.Status.MigrationState.EndTimestamp = &now
			vmiUpdated.Status.NodeName = "othernode"
			vmiUpdated.Labels[v1.NodeNameLabel] = "othernode"
			vmiUpdated.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)
			vmiInterface.EXPECT().Update(vmiUpdated)

			controller.Execute()
		}, 3)
		It("update guest time after completed migration", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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
			client.EXPECT().SetVirtualMachineGuestTime(vmi)
			vmiInterface.EXPECT().Update(vmiUpdated)

			controller.Execute()
		}, 3)
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

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
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
			Expect(err).To(BeNil())
		})
		It("should migrate shared disks without blockMigration flag", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
			}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeFalse())
			Expect(err).To(BeNil())
		})
		It("should fail migration for non-shared PVCs", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
			}

			testBlockPvc.Spec.AccessModes = []k8sv1.PersistentVolumeAccessMode{k8sv1.ReadWriteOnce}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI with non-shared PVCs")))
		})
		It("should fail migration for non-shared data volume PVCs", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
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

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(Equal(fmt.Errorf("cannot migrate VMI with non-shared PVCs")))
		})
		It("should be allowed to migrate a mix of shared and non-shared disks", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
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

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(BeNil())
		})
		It("should be allowed to migrate a mix of non-shared and shared disks", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
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
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
			}

			virtClient.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(testBlockPvc)
			blockMigrate, err := controller.checkVolumesForMigration(vmi)
			Expect(blockMigrate).To(BeTrue())
			Expect(err).To(BeNil())
		})
		It("should be allowed to live-migrate shared HostDisks ", func() {
			_true := true
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "myvolume",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
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
			Expect(err).To(BeNil())
		})
		It("should not be allowed to live-migrate shared and non-shared HostDisks ", func() {
			_true := true
			_false := false
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name: "mydisk1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
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

		Context("with network configuration", func() {
			It("should block migration for bridge binding assigned to the pod network", func() {
				vmi := v1.NewMinimalVMI("testvmi")
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
				vmi := v1.NewMinimalVMI("testvmi")
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
				vmi := v1.NewMinimalVMI("testvmi")
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

	})
	Context("When VirtualMachineInstance is connected to a network", func() {

		It("should only report the pod network in status", func() {

			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			interfaceName := "interface_name"
			vmi.Spec.Networks = []v1.Network{
				{
					Name:          interfaceName,
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Interfaces = make([]v1.VirtualMachineInstanceNetworkInterface, 0)

			podCacheInterface := network.PodCacheInterface{
				Iface: &v1.Interface{
					Name: interfaceName,
				},
				PodIP: "1.1.1.1",
			}
			podJson, err := json.Marshal(podCacheInterface)
			Expect(err).ToNot(HaveOccurred())
			err = os.MkdirAll(fmt.Sprintf(util.VMIInterfaceDir, vmi.UID), 0755)
			Expect(err).ToNot(HaveOccurred())
			vmiInterfacepath := fmt.Sprintf(util.VMIInterfacepath, vmi.UID, interfaceName)
			f, err := os.Create(vmiInterfacepath)
			Expect(err).ToNot(HaveOccurred())
			_, err = f.WriteString(string(podJson))
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(podCacheInterface.Iface.Name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IP).To(Equal(podCacheInterface.PodIP))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IPs[0]).To(Equal(podCacheInterface.PodIP))

			}).Return(vmi, nil)

			controller.Execute()
		})

		It("Should update existing interface with latest IP", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			interfaceName := "interface_name"
			MAC := "1C:CE:C0:01:BE:E7"
			vmi.Spec.Networks = []v1.Network{
				{
					Name:          interfaceName,
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				{
					Name: interfaceName,
					InterfaceBindingMethod: v1.InterfaceBindingMethod{
						Masquerade: &v1.InterfaceMasquerade{},
					},
				},
			}
			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					Name: interfaceName,
					IP:   "1.1.1.1",
					MAC:  MAC,
				},
			}

			podCacheInterface := network.PodCacheInterface{
				Iface: &v1.Interface{
					Name: interfaceName,
				},
				PodIP: "2.2.2.2",
			}
			podJson, err := json.Marshal(podCacheInterface)
			Expect(err).ToNot(HaveOccurred())
			err = os.MkdirAll(fmt.Sprintf(util.VMIInterfaceDir, vmi.UID), 0755)
			Expect(err).ToNot(HaveOccurred())
			vmiInterfacepath := fmt.Sprintf(util.VMIInterfacepath, vmi.UID, interfaceName)
			f, err := os.Create(vmiInterfacepath)
			Expect(err).ToNot(HaveOccurred())
			_, err = f.WriteString(string(podJson))
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running
			domain.Spec.Devices.Interfaces = []api.Interface{
				{
					MAC:   &api.MAC{MAC: MAC},
					Alias: &api.Alias{Name: interfaceName},
				},
			}
			domain.Status.Interfaces = []api.InterfaceStatus{
				{
					Name: interfaceName,
					Mac:  MAC,
					Ip:   "1.1.1.1",
					IPs:  []string{"1.1.1.1"},
				},
			}
			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(podCacheInterface.Iface.Name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IP).To(Equal(podCacheInterface.PodIP))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(domain.Status.Interfaces[0].Mac))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IPs[0]).To(Equal(podCacheInterface.PodIP))
			}).Return(vmi, nil)

			controller.Execute()
		})
	})
	Context("VirtualMachineInstance controller gets informed about interfaces in a Domain", func() {
		It("should update existing interface with MAC", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			interface_name := "interface_name"

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					IP:   "1.1.1.1",
					MAC:  "C0:01:BE:E7:15:G0:0D",
					Name: interface_name,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			new_MAC := "1C:CE:C0:01:BE:E7"

			domain.Spec.Devices.Interfaces = []api.Interface{
				{
					MAC:   &api.MAC{MAC: new_MAC},
					Alias: &api.Alias{Name: interface_name},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(new_MAC))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should update existing interface with IPs", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			interfaceName := "interface_name"
			mac := "C0:01:BE:E7:15:G0:0D"
			ip := "2.2.2.2/24"
			ips := []string{"2.2.2.2/24", "3.3.3.3/16"}

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					IP:   "1.1.1.1",
					MAC:  mac,
					Name: interfaceName,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			domain.Spec.Devices.Interfaces = []api.Interface{
				{
					MAC:   &api.MAC{MAC: mac},
					Alias: &api.Alias{Name: interfaceName},
				},
			}
			domain.Status.Interfaces = []api.InterfaceStatus{
				{
					Name: interfaceName,
					Mac:  mac,
					Ip:   ip,
					IPs:  ips,
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(interfaceName))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(mac))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IP).To(Equal(ip))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].IPs).To(Equal(ips))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should update Guest OS Information in VMI status", func() {
			vmi := v1.NewMinimalVMI("testvmi")
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

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).Status.GuestOSInfo.Name).To(Equal(guestOSName))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should add new vmi interfaces for new domain interfaces", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			old_interface_name := "old_interface_name"
			old_MAC := "C0:01:BE:E7:15:G0:0D"

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					IP:   "1.1.1.1",
					MAC:  old_MAC,
					Name: old_interface_name,
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			new_interface_name := "new_interface_name"
			new_MAC := "1C:CE:C0:01:BE:E7"

			domain.Spec.Devices.Interfaces = []api.Interface{
				{
					MAC:   &api.MAC{MAC: old_MAC},
					Alias: &api.Alias{Name: old_interface_name},
				},
				{
					MAC:   &api.MAC{MAC: new_MAC},
					Alias: &api.Alias{Name: new_interface_name},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(2))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(old_interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(old_MAC))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[1].Name).To(Equal(new_interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[1].MAC).To(Equal(new_MAC))
			}).Return(vmi, nil)

			controller.Execute()
		})

		It("should update name on status interfaces with no name", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.UID = vmiTestUUID
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			interface_name := "interface_name"

			vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					IP:  "1.1.1.1",
					MAC: "C0:01:BE:E7:15:G0:0D",
				},
			}

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomainWithUUID("testvmi", vmiTestUUID)
			domain.Status.Status = api.Running

			new_MAC := "1C:CE:C0:01:BE:E7"

			domain.Spec.Devices.Interfaces = []api.Interface{
				{
					MAC:   &api.MAC{MAC: new_MAC},
					Alias: &api.Alias{Name: interface_name},
				},
			}

			vmi.Spec.Networks = []v1.Network{
				{
					Name:          "other_name",
					NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}},
				},
				{
					Name:          interface_name,
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
			}

			vmiFeeder.Add(vmi)
			domainFeeder.Add(domain)

			vmiInterface.EXPECT().Update(gomock.Any()).Do(func(arg interface{}) {
				Expect(len(arg.(*v1.VirtualMachineInstance).Status.Interfaces)).To(Equal(1))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].Name).To(Equal(interface_name))
				Expect(arg.(*v1.VirtualMachineInstance).Status.Interfaces[0].MAC).To(Equal(new_MAC))
			}).Return(vmi, nil)

			controller.Execute()
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
