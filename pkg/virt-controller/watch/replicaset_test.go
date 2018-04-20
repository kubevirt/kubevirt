package watch

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v13 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Replicaset", func() {

	table.DescribeTable("Different replica diffs given", func(diff int, burstReplicas int, result int) {
		Expect(limit(diff, uint(burstReplicas))).To(Equal(result))
	},
		table.Entry("should limit negative diff to negative burst maximum", -10, 5, -5),
		table.Entry("should return negative diff if bigger than negative burst maximum", -4, 5, -4),
		table.Entry("should limit positive diff to positive burst maximum", 10, 5, 5),
		table.Entry("should return positive diff if less than positive burst maximum", 4, 5, 4),
		table.Entry("should return 0 for zero diff", 0, 5, 0),
	)

	Context("One valid ReplicaSet controller given", func() {

		var ctrl *gomock.Controller
		var vmInterface *kubecli.MockVMInterface
		var rsInterface *kubecli.MockReplicaSetInterface
		var vmSource *framework.FakeControllerSource
		var rsSource *framework.FakeControllerSource
		var vmInformer cache.SharedIndexInformer
		var rsInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *VMReplicaSet
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmFeeder *testutils.VirtualMachineFeeder

		syncCaches := func(stop chan struct{}) {
			go vmInformer.Run(stop)
			go rsInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, rsInformer.HasSynced)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmInterface = kubecli.NewMockVMInterface(ctrl)
			rsInterface = kubecli.NewMockReplicaSetInterface(ctrl)

			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			rsInformer, rsSource = testutils.NewFakeInformerFor(&v1.VirtualMachineReplicaSet{})
			recorder = record.NewFakeRecorder(100)

			controller = NewVMReplicaSet(vmInformer, rsInformer, recorder, virtClient, uint(10))
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			vmFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmSource)

			// Set up mock client
			virtClient.EXPECT().VM(v12.NamespaceDefault).Return(vmInterface).AnyTimes()
			virtClient.EXPECT().ReplicaSet(v12.NamespaceDefault).Return(rsInterface).AnyTimes()
			syncCaches(stop)
		})

		addReplicaSet := func(rs *v1.VirtualMachineReplicaSet) {
			mockQueue.ExpectAdds(1)
			rsSource.Add(rs)
			mockQueue.Wait()
		}

		It("should create missing VMs", func() {
			rs, vm := DefaultReplicaSet(3)

			addReplicaSet(rs)

			vmInterface.EXPECT().Create(gomock.Any()).Times(3).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal("testvm"))
			}).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should create missing VMs when it gets unpaused", func() {
			rs, vm := DefaultReplicaSet(3)
			rs.Spec.Paused = false
			rs.Status.Conditions = []v1.VMReplicaSetCondition{
				{
					Type:               v1.VMReplicaSetReplicaPaused,
					LastTransitionTime: v12.Now(),
				},
			}

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 0
			expectedRS.Status.ReadyReplicas = 0
			expectedRS.Status.Conditions = nil

			addReplicaSet(rs)

			vmInterface.EXPECT().Create(gomock.Any()).Times(3).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal("testvm"))
			}).Return(vm, nil)
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulResumedReplicaSetReason)
		})

		It("should not create missing VMs when it is paused and add paused condition", func() {
			rs, _ := DefaultReplicaSet(3)
			rs.Spec.Paused = true

			// This will trigger a status update, since there are no replicas present
			rs.Status.Replicas = 1
			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 0
			expectedRS.Status.ReadyReplicas = 0

			addReplicaSet(rs)

			// No invocations expected
			vmInterface.EXPECT().Create(gomock.Any()).Times(0)

			// Synchronizing the state is expected
			rsInterface.EXPECT().Update(gomock.Any()).Times(1).Do(func(obj *v1.VirtualMachineReplicaSet) {
				Expect(obj.Status.Replicas).To(Equal(int32(0)))
				Expect(obj.Status.ReadyReplicas).To(Equal(int32(0)))
				Expect(obj.Status.Conditions[0].Type).To(Equal(v1.VMReplicaSetReplicaPaused))
			})

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulPausedReplicaSetReason)
		})

		It("should create missing VMs in batches of a maximum of 10 VMs at once", func() {
			rs, vm := DefaultReplicaSet(15)

			addReplicaSet(rs)

			rsInterface.EXPECT().Update(gomock.Any()).AnyTimes()

			// Check if only 10 are created
			vmInterface.EXPECT().Create(gomock.Any()).Times(10).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal("testvm"))
			}).Return(vm, nil)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			}

			// TODO test for missing 5
		})

		It("should delete missing VMs in batches of a maximum of 10 VMs at once", func() {
			rs, _ := DefaultReplicaSet(0)

			addReplicaSet(rs)

			// Add 15 VMs to the cache
			for x := 0; x < 15; x++ {
				vm := v1.NewMinimalVM(fmt.Sprintf("testvm%d", x))
				vm.ObjectMeta.Labels = map[string]string{"test": "test"}
				vm.OwnerReferences = []v12.OwnerReference{OwnerRef(rs)}
				vmFeeder.Add(vm)
			}

			rsInterface.EXPECT().Update(gomock.Any()).AnyTimes()

			// Check if only 10 are deleted
			vmInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).
				Times(10).Return(nil)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}

			// TODO test for missing 5
		})

		It("should ignore non-matching VMs", func() {
			rs, vm := DefaultReplicaSet(3)

			nonMatchingVM := v1.NewMinimalVM("testvm1")
			nonMatchingVM.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addReplicaSet(rs)

			// We still expect three calls to create VMs, since VM does not meet the requirements
			vmSource.Add(nonMatchingVM)

			vmInterface.EXPECT().Create(gomock.Any()).Times(3).Return(vm, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should delete a VM and increase the replica count", func() {
			rs, vm := DefaultReplicaSet(0)

			rs.Status.Replicas = 0

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 1

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Delete(vm.ObjectMeta.Name, gomock.Any())
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should detect that it is orphan deleted and remove the owner reference on the remaining VM", func() {
			rs, vm := DefaultReplicaSet(1)

			rs.Status.Replicas = 1

			// Mark it as orphan deleted
			now := v12.Now()
			rs.ObjectMeta.DeletionTimestamp = &now
			rs.ObjectMeta.Finalizers = []string{v12.FinalizerOrphanDependents}

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a VM already exists and adopt it", func() {
			rs, vm := DefaultReplicaSet(1)
			vm.OwnerReferences = []v12.OwnerReference{}

			rs.Status.Replicas = 1

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			rsInterface.EXPECT().Get(rs.ObjectMeta.Name, gomock.Any()).Return(rs, nil)
			vmInterface.EXPECT().Patch(vm.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that it has nothing to do", func() {
			rs, vm := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			controller.Execute()
		})

		It("should be woken by a stopped VM and create a new one", func() {
			rs, vm := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			rsCopy := rs.DeepCopy()
			rsCopy.Status.Replicas = 0

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			rsInterface.EXPECT().Update(rsCopy).Times(1)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VM to a final state
			modifiedVM := vm.DeepCopy()
			modifiedVM.Status.Phase = v1.Succeeded
			modifiedVM.ResourceVersion = "1"
			vmFeeder.Modify(modifiedVM)

			// Expect the re-crate of the VM
			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)

			// Run the controller again
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should be woken by a ready VM and update the readyReplicas counter", func() {
			rs, vm := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 0

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 1
			expectedRS.Status.ReadyReplicas = 1

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			rsInterface.EXPECT().Update(expectedRS).Times(1)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VM to a final state
			modifiedVM := vm.DeepCopy()
			modifiedVM.Status.Phase = v1.Running
			modifiedVM.ResourceVersion = "1"
			vmFeeder.Modify(modifiedVM)

			// Run the controller again
			controller.Execute()
		})

		It("should be woken by a deleted VM and create a new one", func() {
			rs, vm := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			rsCopy := rs.DeepCopy()
			// The count stays at zero, since we just experienced the delete when we create the new VM
			rsCopy.Status.Replicas = 0

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Delete one VM
			vmFeeder.Delete(vm)

			// Expect the update from 1 to zero replicas
			rsInterface.EXPECT().Update(rsCopy).Times(1)

			// Expect the recrate of the VM
			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)

			// Run the controller again
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should add a fail condition if scaling up fails", func() {
			rs, vm := DefaultReplicaSet(3)

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			// Let first one succeed
			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)
			// Let second one fail
			vmInterface.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(1)))
				Expect(objRS.Status.Conditions).To(HaveLen(1))
				cond := objRS.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VMReplicaSetReplicaFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(v13.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulCreateVirtualMachineReason, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if scaling down fails", func() {
			rs, vm := DefaultReplicaSet(0)
			vm1 := vm.DeepCopy()
			vm1.ObjectMeta.Name = "test1"

			addReplicaSet(rs)
			vmFeeder.Add(vm)
			vmFeeder.Add(vm1)

			// Let first one succeed
			vmInterface.EXPECT().Delete(vm.ObjectMeta.Name, gomock.Any()).Return(nil)
			// Let second one fail
			vmInterface.EXPECT().Delete(vm1.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 2
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(2)))
				Expect(objRS.Status.Conditions).To(HaveLen(1))
				cond := objRS.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VMReplicaSetReplicaFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(v13.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulDeleteVirtualMachineReason, FailedDeleteVirtualMachineReason)
		})

		It("should update the replica count but keep the failed state", func() {
			rs, vm := DefaultReplicaSet(3)
			rs.Status.Conditions = []v1.VMReplicaSetCondition{
				{
					Type:               v1.VMReplicaSetReplicaFailure,
					LastTransitionTime: v12.Now(),
					Message:            "test",
				},
			}

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			// Let first one succeed
			vmInterface.EXPECT().Create(gomock.Any()).Return(vm, nil)
			// Let second one fail
			vmInterface.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(1)))
				Expect(objRS.Status.Conditions).To(HaveLen(1))
				cond := objRS.Status.Conditions[0]
				Expect(cond.Message).To(Equal(rs.Status.Conditions[0].Message))
				Expect(cond.LastTransitionTime).To(Equal(rs.Status.Conditions[0].LastTransitionTime))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulCreateVirtualMachineReason, FailedCreateVirtualMachineReason)
		})
		It("should update the replica count and remove the failed condition", func() {
			rs, vm := DefaultReplicaSet(3)
			rs.Status.Conditions = []v1.VMReplicaSetCondition{
				{
					Type:               v1.VMReplicaSetReplicaFailure,
					LastTransitionTime: v12.Now(),
					Message:            "test",
				},
			}

			addReplicaSet(rs)
			vmFeeder.Add(vm)

			vmInterface.EXPECT().Create(gomock.Any()).Times(2).Return(vm, nil)

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(1)))
				Expect(objRS.Status.Conditions).To(HaveLen(0))
			})

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		AfterEach(func() {
			close(stop)
			// Ensure that we add checks for expected events to every test
			Expect(recorder.Events).To(BeEmpty())
			ctrl.Finish()
		})
	})
})

func ReplicaSetFromVM(name string, vm *v1.VirtualMachine, replicas int32) *v1.VirtualMachineReplicaSet {
	rs := &v1.VirtualMachineReplicaSet{
		ObjectMeta: v12.ObjectMeta{Name: name, Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VMReplicaSetSpec{
			Replicas: &replicas,
			Selector: &v12.LabelSelector{
				MatchLabels: vm.ObjectMeta.Labels,
			},
			Template: &v1.VMTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Name:   vm.ObjectMeta.Name,
					Labels: vm.ObjectMeta.Labels,
				},
				Spec: vm.Spec,
			},
		},
	}
	return rs
}

func DefaultReplicaSet(replicas int32) (*v1.VirtualMachineReplicaSet, *v1.VirtualMachine) {
	vm := v1.NewMinimalVM("testvm")
	vm.ObjectMeta.Labels = map[string]string{"test": "test"}
	rs := ReplicaSetFromVM("rs", vm, replicas)
	vm.OwnerReferences = []v12.OwnerReference{OwnerRef(rs)}
	return rs, vm
}
