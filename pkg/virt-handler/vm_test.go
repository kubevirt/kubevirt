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
	"io/ioutil"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/precond"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

var _ = Describe("VirtualMachineInstance", func() {
	var client *cmdclient.MockLauncherClient
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var virtClient *kubecli.MockKubevirtClient

	var ctrl *gomock.Controller
	var controller *VirtualMachineController
	var vmiSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var domainSource *framework.FakeControllerSource
	var domainInformer cache.SharedIndexInformer
	var gracefulShutdownInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue
	var mockWatchdog *MockWatchdog
	var mockGracefulShutdown *MockGracefulShutdown

	var vmiFeeder *testutils.VirtualMachineFeeder
	var domainFeeder *testutils.DomainFeeder

	var recorder record.EventRecorder

	var err error
	var shareDir string
	var stop chan struct{}

	log.Log.SetIOWriter(GinkgoWriter)

	BeforeEach(func() {
		stop = make(chan struct{})
		shareDir, err = ioutil.TempDir("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		host := "master"

		Expect(err).ToNot(HaveOccurred())

		vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
		domainInformer, domainSource = testutils.NewFakeInformerFor(&api.Domain{})
		gracefulShutdownInformer, _ = testutils.NewFakeInformerFor(&api.Domain{})
		recorder = record.NewFakeRecorder(100)

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()

		mockWatchdog = &MockWatchdog{shareDir}
		mockGracefulShutdown = &MockGracefulShutdown{shareDir}

		controller = NewController(recorder,
			virtClient,
			host,
			shareDir,
			vmiInformer,
			domainInformer,
			gracefulShutdownInformer,
			1,
			10,
		)

		client = cmdclient.NewMockLauncherClient(ctrl)
		sockFile := cmdclient.SocketFromNamespaceName(shareDir, "default", "testvmi")
		controller.addLauncherClient(client, sockFile)

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)
		domainFeeder = testutils.NewDomainFeeder(mockQueue, domainSource)

		go vmiInformer.Run(stop)
		go domainInformer.Run(stop)
		go gracefulShutdownInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, domainInformer.HasSynced, gracefulShutdownInformer.HasSynced)).To(BeTrue())
	})

	AfterEach(func() {
		close(stop)
		ctrl.Finish()
		os.RemoveAll(shareDir)
	})

	initGracePeriodHelper := func(gracePeriod int64, vmi *v1.VirtualMachineInstance, dom *api.Domain) {
		vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
		dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = gracePeriod
	}

	Context("VirtualMachineInstance controller gets informed about a Domain change through the Domain controller", func() {

		It("should delete non-running Domains if no cluster wide equivalent and no grace period info exists", func() {
			domain := api.NewMinimalDomain("testvmi")
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceFromName("testvmi"))
			client.EXPECT().Close()
			controller.Execute()
		})

		It("should delete running Domains if no cluster wide equivalent exists and no grace period info exists", func() {
			domain := api.NewMinimalDomain("testvmi")
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceFromName("testvmi"))
			client.EXPECT().Close()

			controller.Execute()
		})

		It("should perform cleanup of local ephemeral data if domain and vmi are deleted", func() {
			mockQueue.Add("default/testvmi")
			client.EXPECT().Close()
			controller.Execute()
		})

		It("should attempt graceful shutdown of Domain if trigger file exists.", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.Status.Phase = v1.Running

			domain := api.NewMinimalDomain("testvmi")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceFromName("testvmi"))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should attempt graceful shutdown of Domain if no cluster wide equivalent exists", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			domain := api.NewMinimalDomain("testvmi")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			mockWatchdog.CreateFile(vmi)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMIReferenceFromName("testvmi"))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should attempt force terminate Domain if grace period expires", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			domain := api.NewMinimalDomain("testvmi")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vmi, domain)
			metav1.Now()
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix()-3, 0)}
			domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now

			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceFromName("testvmi"))
			client.EXPECT().Close()
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should immediately kill domain with grace period of 0", func() {
			domain := api.NewMinimalDomain("testvmi")
			domain.Status.Status = api.Running
			vmi := v1.NewMinimalVMI("testvmi")

			initGracePeriodHelper(0, vmi, domain)
			mockWatchdog.CreateFile(vmi)
			mockGracefulShutdown.TriggerShutdown(vmi)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMIReferenceFromName("testvmi"))
			client.EXPECT().Close()
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
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			client.EXPECT().SyncVirtualMachine(vmi)

			controller.Execute()
		})

		It("should update from Scheduled to Running, if it sees a running Domain", func() {
			vmi := v1.NewMinimalVMI("testvmi")
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Phase = v1.Running

			mockWatchdog.CreateFile(vmi)
			domain := api.NewMinimalDomain("testvmi")
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
			vmi.ObjectMeta.ResourceVersion = "1"
			vmi.Status.Phase = v1.Scheduled
			vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
				{
					Type:   v1.VirtualMachineInstanceSynchronized,
					Status: k8sv1.ConditionFalse,
				},
			}

			updatedVMI := vmi.DeepCopy()
			updatedVMI.Status.Conditions = nil

			mockWatchdog.CreateFile(vmi)
			vmiFeeder.Add(vmi)

			client.EXPECT().SyncVirtualMachine(vmi)
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
	})
})

type MockGracefulShutdown struct {
	baseDir string
}

func (m *MockGracefulShutdown) TriggerShutdown(vmi *v1.VirtualMachineInstance) {
	Expect(os.MkdirAll(virtlauncher.GracefulShutdownTriggerDir(m.baseDir), os.ModePerm)).To(Succeed())

	namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())
	triggerFile := virtlauncher.GracefulShutdownTriggerFromNamespaceName(m.baseDir, namespace, domain)
	err := virtlauncher.GracefulShutdownTriggerInitiate(triggerFile)
	Expect(err).NotTo(HaveOccurred())
}

type MockWatchdog struct {
	baseDir string
}

func (m *MockWatchdog) CreateFile(vmi *v1.VirtualMachineInstance) {
	Expect(os.MkdirAll(watchdog.WatchdogFileDirectory(m.baseDir), os.ModePerm)).To(Succeed())
	err := watchdog.WatchdogFileUpdate(
		watchdog.WatchdogFileFromNamespaceName(m.baseDir, vmi.ObjectMeta.Namespace, vmi.ObjectMeta.Name),
	)
	Expect(err).NotTo(HaveOccurred())
}
