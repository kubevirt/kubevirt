package watch

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	kvpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("RestartRequired", Serial, func() {
	var ctrl *gomock.Controller
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiSource *framework.FakeControllerSource
	var vmSource *framework.FakeControllerSource
	var vmiInformer cache.SharedIndexInformer
	var vmInformer cache.SharedIndexInformer
	var dataVolumeInformer cache.SharedIndexInformer
	var pvcInformer cache.SharedIndexInformer
	var crInformer cache.SharedIndexInformer
	var crSource *framework.FakeControllerSource
	var podInformer cache.SharedIndexInformer
	var instancetypeMethods *testutils.MockInstancetypeMethods
	var stop chan struct{}
	var controller *VMController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var vmiFeeder *testutils.VirtualMachineFeeder
	var k8sClient *k8sfake.Clientset
	var virtClient *kubecli.MockKubevirtClient
	var config *virtconfig.ClusterConfig
	var kvInformer cache.SharedIndexInformer

	syncCaches := func() {
		Expect(cache.WaitForCacheSync(stop,
			vmiInformer.HasSynced,
			vmInformer.HasSynced,
			dataVolumeInformer.HasSynced,
			crInformer.HasSynced,
		)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		generatedInterface := fake.NewSimpleClientset()

		dataVolumeInformer, _ = testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
		dataSourceInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataSource{})
		vmiInformer, vmiSource = testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachineInstance{}, virtcontroller.GetVMIInformerIndexers())
		vmInformer, vmSource = testutils.NewFakeInformerWithIndexersFor(&virtv1.VirtualMachine{}, virtcontroller.GetVirtualMachineInformerIndexers())
		pvcInformer, _ = testutils.NewFakeInformerFor(&k8sv1.PersistentVolumeClaim{})
		namespaceInformer, _ := testutils.NewFakeInformerFor(&k8sv1.Namespace{})
		crInformer, crSource = testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
			"vm": func(obj interface{}) ([]string, error) {
				cr := obj.(*appsv1.ControllerRevision)
				for _, ref := range cr.OwnerReferences {
					if ref.Kind == "VirtualMachine" {
						return []string{string(ref.UID)}, nil
					}
				}
				return nil, nil
			},
		})
		podInformer, _ = testutils.NewFakeInformerFor(&k8sv1.Pod{})

		instancetypeMethods = testutils.NewMockInstancetypeMethods()

		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		config, _, kvInformer = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})

		controller, _ = NewVMController(vmiInformer,
			vmInformer,
			dataVolumeInformer,
			dataSourceInformer,
			namespaceInformer.GetStore(),
			pvcInformer,
			crInformer,
			podInformer,
			instancetypeMethods,
			recorder,
			virtClient,
			config)

		// Wrap our workqueue to have a way to detect when we are done processing updates
		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue

		vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()
		virtClient.EXPECT().GeneratedKubeVirtClient().Return(generatedInterface).AnyTimes()

		k8sClient = k8sfake.NewSimpleClientset()
		virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
		virtClient.EXPECT().CoreV1().Return(k8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().AuthorizationV1().Return(k8sClient.AuthorizationV1()).AnyTimes()
		go vmiInformer.Run(stop)
		go vmInformer.Run(stop)
		go dataVolumeInformer.Run(stop)
		go crInformer.Run(stop)
	})

	AfterEach(func() {
		close(stop)
	})

	addVirtualMachine := func(vm *virtv1.VirtualMachine) {
		syncCaches()
		mockQueue.ExpectAdds(1)
		vmSource.Add(vm)
		mockQueue.Wait()
	}

	modifyVirtualMachine := func(vm *virtv1.VirtualMachine) {
		mockQueue.ExpectAdds(1)
		vmSource.Modify(vm)
		mockQueue.Wait()
	}

	Context("the condition", func() {
		var vm *virtv1.VirtualMachine
		var vmi *virtv1.VirtualMachineInstance
		var kv *virtv1.KubeVirt
		var crList appsv1.ControllerRevisionList
		var crListLock sync.Mutex

		restartRequired := false

		expectVMUpdate := func() {
			vmInterface.EXPECT().Update(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, vm *virtv1.VirtualMachine) (interface{}, error) {
				for _, condition := range vm.Status.Conditions {
					if condition.Type == virtv1.VirtualMachineRestartRequired {
						restartRequired = condition.Status == k8sv1.ConditionTrue
					}
				}
				return vm, nil
			}).AnyTimes()
		}

		expectVMStatusUpdate := func() {
			vmInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, vm *virtv1.VirtualMachine) (interface{}, error) {
				for _, condition := range vm.Status.Conditions {
					if condition.Type == virtv1.VirtualMachineRestartRequired {
						restartRequired = condition.Status == k8sv1.ConditionTrue
					}
				}
				return vm, nil
			}).AnyTimes()
		}

		expectVMICreation := func() {
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, arg interface{}) (interface{}, error) {
				return arg, nil
			}).AnyTimes()
		}

		expectVMIPatch := func() {
			vmiInterface.EXPECT().Patch(context.Background(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(vmi, nil).AnyTimes()
		}

		expectControllerRevisionList := func() {
			k8sClient.Fake.PrependReactor("list", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				crListLock.Lock()
				defer crListLock.Unlock()
				return true, crList.DeepCopy(), nil
			})
		}

		expectControllerRevisionDelete := func() {
			k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				deleted, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())

				crListLock.Lock()
				defer crListLock.Unlock()
				for i, obj := range crList.Items {
					if obj.Name == deleted.GetName() && obj.Namespace == deleted.GetNamespace() {
						crList.Items = append(crList.Items[:i], crList.Items[i+1:]...)
						return true, nil, nil
					}
				}
				return true, nil, fmt.Errorf("not found")
			})
		}

		expectControllerRevisionCreation := func() {
			k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				createObj, ok := created.GetObject().(*appsv1.ControllerRevision)
				Expect(ok).To(BeTrue())

				crListLock.Lock()
				defer crListLock.Unlock()
				crList.Items = append(crList.Items, *createObj)
				crSource.Add(createObj)

				return true, created.GetObject(), nil
			})
		}

		crFor := func(uid string) string {
			crListLock.Lock()
			defer crListLock.Unlock()
			for _, cr := range crList.Items {
				if strings.Contains(cr.Name, uid) {
					return cr.Name
				}
			}
			return ""
		}

		BeforeEach(func() {
			k8sClient.Fake.ClearActions()
			crList = appsv1.ControllerRevisionList{}
			vm, vmi = DefaultVirtualMachine(true)
			vm.ObjectMeta.UID = types.UID(uuid.NewString())
			vmi.ObjectMeta.UID = vm.ObjectMeta.UID
			vm.Generation = 1
			vm.Spec.Template.Spec.Domain.CPU = &v1.CPU{
				Cores: 2,
			}
			guest := resource.MustParse("128Mi")
			vm.Spec.Template.Spec.Domain.Memory = &v1.Memory{
				Guest: &guest,
			}
			kv = &v1.KubeVirt{
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						LiveUpdateConfiguration: &virtv1.LiveUpdateConfiguration{},
						VMRolloutStrategy:       &liveUpdate,
						DeveloperConfiguration: &v1.DeveloperConfiguration{
							FeatureGates: []string{virtconfig.VMLiveUpdateFeaturesGate},
						},
					},
				},
			}
			restartRequired = false
		})

		AfterEach(func() {
			k8sClient.Fake.ClearActions()
		})

		It("should appear when changing a non-live-updatable field", func() {
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

			By("Creating a VM with hostname a")
			vm.Spec.Template.Spec.Hostname = "a"
			addVirtualMachine(vm)

			By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
			vmi = controller.setupVMIFromVM(vm)
			expectVMICreation()
			expectVMStatusUpdate()
			expectControllerRevisionList()
			expectControllerRevisionCreation()
			controller.Execute()
			syncCaches()
			Expect(restartRequired).To(BeFalse(), "restart required")
			markAsReady(vmi)
			vmiFeeder.Add(vmi)

			By("Bumping the VM sockets above the cluster maximum")
			vm.Spec.Template.Spec.Hostname = "b"
			vm.Generation = 2
			modifyVirtualMachine(vm)

			By("Executing the controller again expecting the RestartRequired condition to appear")
			expectControllerRevisionDelete()
			expectVMUpdate()
			controller.Execute()
			syncCaches()
			Expect(restartRequired).To(BeTrue(), "restart required")
		})

		It("should appear when VM doesn't specify maxSockets and sockets go above cluster-wide maxSockets", func() {
			var maxSockets uint32 = 8

			By("Setting a cluster-wide CPU maxSockets value")
			kv.Spec.Configuration.LiveUpdateConfiguration.MaxCpuSockets = kvpointer.P(maxSockets)
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

			By("Creating a VM with CPU sockets set to the cluster maxiumum")
			vm.Spec.Template.Spec.Domain.CPU.Sockets = 8
			addVirtualMachine(vm)

			By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
			vmi = controller.setupVMIFromVM(vm)
			expectVMICreation()
			expectVMStatusUpdate()
			expectControllerRevisionList()
			expectControllerRevisionCreation()
			controller.Execute()
			syncCaches()
			Expect(restartRequired).To(BeFalse(), "restart required")
			markAsReady(vmi)
			vmiFeeder.Add(vmi)

			By("Bumping the VM sockets above the cluster maximum")
			vm.Spec.Template.Spec.Domain.CPU.Sockets = 10
			vm.Generation = 2
			modifyVirtualMachine(vm)

			By("Executing the controller again expecting the RestartRequired condition to appear")
			expectVMUpdate()
			expectControllerRevisionDelete()
			controller.Execute()
			syncCaches()
			Expect(restartRequired).To(BeTrue(), "restart required")
		})

		It("should appear when VM doesn't specify maxGuest and guest memory goes above cluster-wide maxGuest", func() {
			var maxGuest = resource.MustParse("256Mi")

			By("Setting a cluster-wide CPU maxGuest value")
			kv.Spec.Configuration.LiveUpdateConfiguration.MaxGuest = &maxGuest
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

			By("Creating a VM with guest memory set to the cluster maximum")
			vm.Spec.Template.Spec.Domain.Memory.Guest = &maxGuest
			addVirtualMachine(vm)

			By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
			vmi = controller.setupVMIFromVM(vm)
			expectVMICreation()
			expectVMStatusUpdate()
			expectControllerRevisionList()
			expectControllerRevisionCreation()
			controller.Execute()
			syncCaches()
			Expect(restartRequired).To(BeFalse(), "restart required")
			markAsReady(vmi)
			vmi.Status.Memory = &virtv1.MemoryStatus{
				GuestAtBoot:  &maxGuest,
				GuestCurrent: &maxGuest,
			}
			vmiFeeder.Add(vmi)

			By("Bumping the VM guest memory above the cluster maximum")
			bigGuest := resource.MustParse("257Mi")
			vm.Spec.Template.Spec.Domain.Memory.Guest = &bigGuest
			vm.Generation = 2
			modifyVirtualMachine(vm)

			By("Executing the controller again expecting the RestartRequired condition to appear")
			expectVMUpdate()
			expectControllerRevisionDelete()
			controller.Execute()
			syncCaches()
			Expect(restartRequired).To(BeTrue(), "restart required")
		})

		DescribeTable("when changing a live-updatable field", func(fgs []string, strat *virtv1.VMRolloutStrategy, expectCond bool) {
			kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = fgs
			kv.Spec.Configuration.VMRolloutStrategy = strat
			testutils.UpdateFakeKubeVirtClusterConfig(kvInformer, kv)

			By("Creating a VM with CPU sockets set to the cluster maximum")
			vm.Spec.Template.Spec.Domain.CPU.Sockets = 2
			addVirtualMachine(vm)

			By("Executing the controller expecting a VMI to get created and no RestartRequired condition")
			vmi = controller.setupVMIFromVM(vm)
			expectVMICreation()
			expectVMStatusUpdate()
			expectControllerRevisionList()
			expectControllerRevisionCreation()
			controller.Execute()
			syncCaches()
			Expect(crFor(string(vm.ObjectMeta.UID))).To(ContainSubstring(fmt.Sprintf("%s-%d", vm.ObjectMeta.UID, 1)))
			Expect(restartRequired).To(BeFalse(), "restart required")
			markAsReady(vmi)
			vmiFeeder.Add(vmi)

			By("Bumping the VM sockets to a reasonable value")
			vm.Spec.Template.Spec.Domain.CPU.Sockets = 4
			vm.Generation = 2
			modifyVirtualMachine(vm)

			By("Executing the controller again expecting the RestartRequired condition to appear")
			expectVMUpdate()
			expectControllerRevisionDelete()
			if !expectCond {
				expectVMIPatch()
			}
			controller.Execute()
			syncCaches()
			Expect(crFor(string(vm.ObjectMeta.UID))).To(ContainSubstring(fmt.Sprintf("%s-%d", vm.ObjectMeta.UID, 1)))
			Expect(restartRequired).To(Equal(expectCond), "restart required")
		},
			Entry("should appear if the feature gate is not set",
				[]string{}, &liveUpdate, true),
			Entry("should appear if the VM rollout strategy is not set",
				[]string{virtconfig.VMLiveUpdateFeaturesGate}, nil, true),
			Entry("should appear if the VM rollout strategy is set to Stage",
				[]string{virtconfig.VMLiveUpdateFeaturesGate}, &stage, true),
			Entry("should not appear if both the VM rollout strategy and feature gate are set",
				[]string{virtconfig.VMLiveUpdateFeaturesGate}, &liveUpdate, false),
		)
	})
})
