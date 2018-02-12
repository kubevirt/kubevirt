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
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/watchdog"
)

var _ = Describe("VM", func() {
	var client *cmdclient.MockLauncherClient
	var vmInterface *kubecli.MockVMInterface
	var virtClient *kubecli.MockKubevirtClient

	var ctrl *gomock.Controller
	var controller *VirtualMachineController
	var vmSource *framework.FakeControllerSource
	var vmInformer cache.SharedIndexInformer
	var domainSource *framework.FakeControllerSource
	var domainInformer cache.SharedIndexInformer
	var gracefulShutdownSource *framework.FakeControllerSource
	var gracefulShutdownInformer cache.SharedIndexInformer
	var mockQueue *testutils.MockWorkQueue
	var mockWatchdog *MockWatchdog
	var mockGracefulShutdown *MockGracefulShutdown

	var vmFeeder *testutils.VirtualMachineFeeder
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

		vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
		domainInformer, domainSource = testutils.NewFakeInformerFor(&api.Domain{})
		gracefulShutdownInformer, gracefulShutdownSource = testutils.NewFakeInformerFor(&api.Domain{})
		recorder = record.NewFakeRecorder(100)

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)
		virtClient.EXPECT().VM(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

		mockWatchdog = &MockWatchdog{shareDir}
		mockGracefulShutdown = &MockGracefulShutdown{shareDir}

		controller = NewController(recorder,
			virtClient,
			host,
			shareDir,
			vmInformer,
			domainInformer,
			gracefulShutdownInformer)

		client = cmdclient.NewMockLauncherClient(ctrl)
		sockFile := cmdclient.SocketFromNamespaceName(shareDir, "default", "testvm")
		controller.addLauncherClient(client, sockFile)

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)
		domainFeeder = testutils.NewDomainFeeder(mockQueue, domainSource)

		go vmInformer.Run(stop)
		go domainInformer.Run(stop)
		go gracefulShutdownInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, domainInformer.HasSynced, gracefulShutdownInformer.HasSynced)).To(BeTrue())
	})

	AfterEach(func() {
		close(stop)
		ctrl.Finish()
		os.RemoveAll(shareDir)
	})

	initGracePeriodHelper := func(gracePeriod int64, vm *v1.VirtualMachine, dom *api.Domain) {
		vm.Spec.TerminationGracePeriodSeconds = &gracePeriod
		dom.Spec.Metadata.KubeVirt.GracePeriod.DeletionGracePeriodSeconds = gracePeriod
	}

	Context("VM controller gets informed about a Domain change through the Domain controller", func() {

		It("should delete non-running Domains if no cluster wide equivalent and no grace period info exists", func() {
			domain := api.NewMinimalDomain("testvm")
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMReferenceFromName("testvm"))
			client.EXPECT().Close()
			controller.Execute()
		})

		It("should delete running Domains if no cluster wide equivalent exists and no grace period info exists", func() {
			domain := api.NewMinimalDomain("testvm")
			domain.Status.Status = api.Running
			domainFeeder.Add(domain)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMReferenceFromName("testvm"))
			client.EXPECT().Close()

			controller.Execute()
		})

		It("should perform cleanup of local ephemeral data if domain and vm are deleted", func() {
			mockQueue.Add("default/testvm")
			client.EXPECT().Close()
			controller.Execute()
		})

		It("should attempt graceful shutdown of Domain if trigger file exists.", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = v1.Running

			domain := api.NewMinimalDomain("testvm")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vm, domain)
			mockWatchdog.CreateFile(vm)
			mockGracefulShutdown.TriggerShutdown(vm)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMReferenceFromName("testvm"))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should attempt graceful shutdown of Domain if no cluster wide equivalent exists", func() {
			vm := v1.NewMinimalVM("testvm")
			domain := api.NewMinimalDomain("testvm")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vm, domain)
			mockWatchdog.CreateFile(vm)

			client.EXPECT().Ping()
			client.EXPECT().ShutdownVirtualMachine(v1.NewVMReferenceFromName("testvm"))
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should attempt force terminate Domain if grace period expires", func() {
			vm := v1.NewMinimalVM("testvm")
			domain := api.NewMinimalDomain("testvm")
			domain.Status.Status = api.Running

			initGracePeriodHelper(1, vm, domain)
			metav1.Now()
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix()-3, 0)}
			domain.Spec.Metadata.KubeVirt.GracePeriod.DeletionTimestamp = &now

			mockWatchdog.CreateFile(vm)
			mockGracefulShutdown.TriggerShutdown(vm)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMReferenceFromName("testvm"))
			client.EXPECT().Close()
			domainFeeder.Add(domain)

			controller.Execute()
		}, 3)

		It("should immediately kill domain with grace period of 0", func() {
			domain := api.NewMinimalDomain("testvm")
			domain.Status.Status = api.Running
			vm := v1.NewMinimalVM("testvm")

			initGracePeriodHelper(0, vm, domain)
			mockWatchdog.CreateFile(vm)
			mockGracefulShutdown.TriggerShutdown(vm)

			client.EXPECT().Ping()
			client.EXPECT().KillVirtualMachine(v1.NewVMReferenceFromName("testvm"))
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

		It("should create the Domain if it sees the first time on a new VM", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.ResourceVersion = "1"
			vm.Status.Phase = v1.Scheduled

			mockWatchdog.CreateFile(vm)
			vmFeeder.Add(vm)

			secrets := map[string]*k8sv1.Secret{}
			client.EXPECT().SyncVirtualMachine(vm, secrets)

			controller.Execute()
		})

		It("should update from Scheduled to Running, if it sees a running Domain", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.ResourceVersion = "1"
			vm.Status.Phase = v1.Scheduled

			updatedVM := vm.DeepCopy()
			updatedVM.Status.Phase = v1.Running

			mockWatchdog.CreateFile(vm)
			domain := api.NewMinimalDomain("testvm")
			domain.Status.Status = api.Running
			vmFeeder.Add(vm)
			domainFeeder.Add(domain)

			vmInterface.EXPECT().Update(updatedVM)

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

		It("should detect a missing watchdog file and report the error on the VM", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.ResourceVersion = "1"
			vm.Status.Phase = v1.Scheduled

			vmFeeder.Add(vm)
			vmInterface.EXPECT().Update(gomock.Any()).Do(func(vm *v1.VirtualMachine) {
				Expect(vm.Status.Conditions).To(HaveLen(1))
			})
			controller.Execute()
		})

		It("should remove an error condition if a synchronization run succeeds", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.ResourceVersion = "1"
			vm.Status.Phase = v1.Scheduled
			vm.Status.Conditions = []v1.VirtualMachineCondition{
				{
					Type:   v1.VirtualMachineSynchronized,
					Status: k8sv1.ConditionFalse,
				},
			}

			updatedVM := vm.DeepCopy()
			updatedVM.Status.Conditions = []v1.VirtualMachineCondition{}

			mockWatchdog.CreateFile(vm)
			vmFeeder.Add(vm)

			secrets := map[string]*k8sv1.Secret{}
			client.EXPECT().SyncVirtualMachine(vm, secrets)
			vmInterface.EXPECT().Update(updatedVM)

			controller.Execute()
		})

		table.DescribeTable("should leave the VM alone if it is in the final phase", func(phase v1.VMPhase) {
			vm := v1.NewMinimalVM("testvm")
			vm.Status.Phase = phase
			vmFeeder.Add(vm)
			controller.Execute()
			// expect no errors and no mock interactions
			Expect(mockQueue.NumRequeues("default/testvm")).To(Equal(0))
		},
			table.Entry("succeeded", v1.Succeeded),
			table.Entry("failed", v1.Failed),
		)
	})
})

type MockGracefulShutdown struct {
	baseDir string
}

func (m *MockGracefulShutdown) TriggerShutdown(vm *v1.VirtualMachine) {
	Expect(os.MkdirAll(virtlauncher.GracefulShutdownTriggerDir(m.baseDir), os.ModePerm)).To(Succeed())

	namespace := precond.MustNotBeEmpty(vm.GetObjectMeta().GetNamespace())
	domain := precond.MustNotBeEmpty(vm.GetObjectMeta().GetName())
	triggerFile := virtlauncher.GracefulShutdownTriggerFromNamespaceName(m.baseDir, namespace, domain)
	err := virtlauncher.GracefulShutdownTriggerInitiate(triggerFile)
	Expect(err).NotTo(HaveOccurred())
}

type MockWatchdog struct {
	baseDir string
}

func (m *MockWatchdog) CreateFile(vm *v1.VirtualMachine) {
	Expect(os.MkdirAll(watchdog.WatchdogFileDirectory(m.baseDir), os.ModePerm)).To(Succeed())
	err := watchdog.WatchdogFileUpdate(
		watchdog.WatchdogFileFromNamespaceName(m.baseDir, vm.ObjectMeta.Namespace, vm.ObjectMeta.Name),
	)
	Expect(err).NotTo(HaveOccurred())
}
