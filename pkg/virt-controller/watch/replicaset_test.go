package watch

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
		var rsInterface *kubecli.MockReplicaSetInterface
		var vmiSource *framework.FakeControllerSource
		var rsSource *framework.FakeControllerSource
		var vmiInformer cache.SharedIndexInformer
		var rsInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *VMIReplicaSet
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var vmiFeeder *testutils.VirtualMachineFeeder

		syncCaches := func(stop chan struct{}) {
			go vmiInformer.Run(stop)
			go rsInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, rsInformer.HasSynced)).To(BeTrue())
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			rsInterface = kubecli.NewMockReplicaSetInterface(ctrl)

			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			rsInformer, rsSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstanceReplicaSet{})
			recorder = record.NewFakeRecorder(100)

			controller = NewVMIReplicaSet(vmiInformer, rsInformer, recorder, virtClient, uint(10))
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.Queue)
			controller.Queue = mockQueue
			vmiFeeder = testutils.NewVirtualMachineFeeder(mockQueue, vmiSource)

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
			virtClient.EXPECT().ReplicaSet(metav1.NamespaceDefault).Return(rsInterface).AnyTimes()
			syncCaches(stop)
		})

		addReplicaSet := func(rs *v1.VirtualMachineInstanceReplicaSet) {
			mockQueue.ExpectAdds(1)
			rsSource.Add(rs)
			mockQueue.Wait()
		}

		It("should create missing VMIs", func() {
			rs, vmi := DefaultReplicaSet(3)

			addReplicaSet(rs)

			vmiInterface.EXPECT().Create(gomock.Any()).Times(3).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal("testvmi"))
			}).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should create missing VMIs when it gets unpaused", func() {
			rs, vmi := DefaultReplicaSet(3)
			rs.Spec.Paused = false
			rs.Status.Conditions = []v1.VirtualMachineInstanceReplicaSetCondition{
				{
					Type:               v1.VirtualMachineInstanceReplicaSetReplicaPaused,
					LastTransitionTime: metav1.Now(),
				},
			}

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 0
			expectedRS.Status.ReadyReplicas = 0
			expectedRS.Status.Conditions = nil

			addReplicaSet(rs)

			vmiInterface.EXPECT().Create(gomock.Any()).Times(3).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal("testvmi"))
			}).Return(vmi, nil)
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulResumedReplicaSetReason)
		})

		It("should not create missing VMIs when it is paused and add paused condition", func() {
			rs, _ := DefaultReplicaSet(3)
			rs.Spec.Paused = true

			// This will trigger a status update, since there are no replicas present
			rs.Status.Replicas = 1
			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 0
			expectedRS.Status.ReadyReplicas = 0

			addReplicaSet(rs)

			// No invocations expected
			vmiInterface.EXPECT().Create(gomock.Any()).Times(0)

			// Synchronizing the state is expected
			rsInterface.EXPECT().Update(gomock.Any()).Times(1).Do(func(obj *v1.VirtualMachineInstanceReplicaSet) {
				Expect(obj.Status.Replicas).To(Equal(int32(0)))
				Expect(obj.Status.ReadyReplicas).To(Equal(int32(0)))
				Expect(obj.Status.Conditions[0].Type).To(Equal(v1.VirtualMachineInstanceReplicaSetReplicaPaused))
			})

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulPausedReplicaSetReason)
		})

		It("should create missing VMIs in batches of a maximum of 10 VMIs at once", func() {
			rs, vmi := DefaultReplicaSet(15)

			addReplicaSet(rs)

			rsInterface.EXPECT().Update(gomock.Any()).AnyTimes()

			// Check if only 10 are created
			vmiInterface.EXPECT().Create(gomock.Any()).Times(10).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal("testvmi"))
			}).Return(vmi, nil)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			}

			// TODO test for missing 5
		})

		It("should already create replacement VMIs once it discovers that a VMI is marked for deletion", func() {
			rs, vmi := DefaultReplicaSet(10)

			addReplicaSet(rs)

			// Add 3 VMIs to the cache which are already running
			for x := 0; x < 3; x++ {
				vmi := v1.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmiFeeder.Add(vmi)
			}

			// Add 3 VMIs to the cache which are marked for deletion
			for x := 3; x < 6; x++ {
				vmi := v1.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi.DeletionTimestamp = now()
				vmiFeeder.Add(vmi)
			}

			rsInterface.EXPECT().Update(gomock.Any()).AnyTimes()

			// Should create 7 vms, 3 are already there and 3 are there but marked for deletion
			vmiInterface.EXPECT().Create(gomock.Any()).Times(7).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal("testvmi"))
			}).Return(vmi, nil)

			controller.Execute()

			for x := 0; x < 7; x++ {
				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			}
		})

		It("should delete missing VMIs in batches of a maximum of 10 VMIs at once", func() {
			rs, _ := DefaultReplicaSet(0)

			addReplicaSet(rs)

			// Add 15 VMIs to the cache
			for x := 0; x < 15; x++ {
				vmi := v1.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmiFeeder.Add(vmi)
			}

			rsInterface.EXPECT().Update(gomock.Any()).AnyTimes()

			// Check if only 10 are deleted
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).
				Times(10).Return(nil)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}

			// TODO test for missing 5
		})

		It("should not delete vmis which are already marked deleted", func() {
			rs, _ := DefaultReplicaSet(3)

			addReplicaSet(rs)

			// Add 5 VMIs without deletion timestamp
			for x := 0; x < 5; x++ {
				vmi := v1.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmiFeeder.Add(vmi)
			}
			// Add 5 VMIs with deletion timestamp
			for x := 5; x < 9; x++ {
				vmi := v1.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi.DeletionTimestamp = now()
				vmiFeeder.Add(vmi)
			}

			rsInterface.EXPECT().Update(gomock.Any()).AnyTimes()

			// Check if only two vms get deleted
			vmiInterface.EXPECT().Delete(gomock.Any(), gomock.Any()).
				Times(2).Return(nil)

			controller.Execute()

			for x := 0; x < 2; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}
		})

		It("should ignore non-matching VMIs", func() {
			rs, vmi := DefaultReplicaSet(3)

			nonMatchingVMI := v1.NewMinimalVMI("testvmi1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addReplicaSet(rs)

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			vmiSource.Add(nonMatchingVMI)

			vmiInterface.EXPECT().Create(gomock.Any()).Times(3).Return(vmi, nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should delete a VirtualMachineInstance and increase the replica count", func() {
			rs, vmi := DefaultReplicaSet(0)

			rs.Status.Replicas = 0

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 1

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any())
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should detect that it is orphan deleted and remove the owner reference on the remaining VirtualMachineInstance", func() {
			rs, vmi := DefaultReplicaSet(1)

			rs.Status.Replicas = 1

			// Mark it as orphan deleted
			now := metav1.Now()
			rs.ObjectMeta.DeletionTimestamp = &now
			rs.ObjectMeta.Finalizers = []string{metav1.FinalizerOrphanDependents}

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			rs, vmi := DefaultReplicaSet(1)
			vmi.OwnerReferences = []metav1.OwnerReference{}

			rs.Status.Replicas = 1

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			rsInterface.EXPECT().Get(rs.ObjectMeta.Name, gomock.Any()).Return(rs, nil)
			vmiInterface.EXPECT().Patch(vmi.ObjectMeta.Name, gomock.Any(), gomock.Any())

			controller.Execute()
		})

		It("should detect that it has nothing to do", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			controller.Execute()
		})

		It("should be woken by a stopped VirtualMachineInstance and create a new one", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running

			rsCopy := rs.DeepCopy()
			rsCopy.Status.Replicas = 0
			rsCopy.Status.ReadyReplicas = 0

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.Status.Phase = v1.Succeeded
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Expect the re-crate of the VirtualMachineInstance
			rsInterface.EXPECT().Update(rsCopy).Times(1)
			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)
			// Run the controller again
			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulDeleteVirtualMachineReason, SuccessfulCreateVirtualMachineReason)
		})

		It("should be woken by a ready VirtualMachineInstance and update the readyReplicas counter", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 0

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 1
			expectedRS.Status.ReadyReplicas = 1

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			rsInterface.EXPECT().Update(expectedRS).Times(1)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.Status.Phase = v1.Running
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Run the controller again
			controller.Execute()
		})

		It("should be woken by a deleted VirtualMachineInstance and create a new one", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			rsCopy := rs.DeepCopy()
			// The count stays at zero, since we just experienced the delete when we create the new VirtualMachineInstance
			rsCopy.Status.Replicas = 0

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Delete one VirtualMachineInstance
			vmiFeeder.Delete(vmi)

			// Expect the update from 1 to zero replicas
			rsInterface.EXPECT().Update(rsCopy).Times(1)

			// Expect the recrate of the VirtualMachineInstance
			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)

			// Run the controller again
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should delete VirtualMachineIstance in the final state", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.Status.Phase = v1.Failed
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Expect the re-crate of the VirtualMachineInstance
			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(nil)

			// Run the cleanFinishedVmis method
			controller.cleanFinishedVmis(rs, []*v1.VirtualMachineInstance{modifiedVMI})

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should add a fail condition if scaling up fails", func() {
			rs, vmi := DefaultReplicaSet(3)

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// Let first one succeed
			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)
			// Let second one fail
			vmiInterface.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(1)))
				Expect(objRS.Status.Conditions).To(HaveLen(1))
				cond := objRS.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineInstanceReplicaSetReplicaFailure))
				Expect(cond.Reason).To(Equal("FailedCreate"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulCreateVirtualMachineReason, FailedCreateVirtualMachineReason)
		})

		It("should add a fail condition if scaling down fails", func() {
			rs, vmi := DefaultReplicaSet(0)
			vmi1 := vmi.DeepCopy()
			vmi1.ObjectMeta.Name = "test1"

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)
			vmiFeeder.Add(vmi1)

			// Let first one succeed
			vmiInterface.EXPECT().Delete(vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			// Let second one fail
			vmiInterface.EXPECT().Delete(vmi1.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 2
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(2)))
				Expect(objRS.Status.Conditions).To(HaveLen(1))
				cond := objRS.Status.Conditions[0]
				Expect(cond.Type).To(Equal(v1.VirtualMachineInstanceReplicaSetReplicaFailure))
				Expect(cond.Reason).To(Equal("FailedDelete"))
				Expect(cond.Message).To(Equal("failure"))
				Expect(cond.Status).To(Equal(k8sv1.ConditionTrue))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulDeleteVirtualMachineReason, FailedDeleteVirtualMachineReason)
		})

		It("should update the replica count but keep the failed state", func() {
			rs, vmi := DefaultReplicaSet(3)
			rs.Status.Conditions = []v1.VirtualMachineInstanceReplicaSetCondition{
				{
					Type:               v1.VirtualMachineInstanceReplicaSetReplicaFailure,
					LastTransitionTime: metav1.Now(),
					Message:            "test",
				},
			}

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// Let first one succeed
			vmiInterface.EXPECT().Create(gomock.Any()).Return(vmi, nil)
			// Let second one fail
			vmiInterface.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
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
			rs, vmi := DefaultReplicaSet(3)
			rs.Status.Conditions = []v1.VirtualMachineInstanceReplicaSetCondition{
				{
					Type:               v1.VirtualMachineInstanceReplicaSetReplicaFailure,
					LastTransitionTime: metav1.Now(),
					Message:            "test",
				},
			}

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			vmiInterface.EXPECT().Create(gomock.Any()).Times(2).Return(vmi, nil)

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
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

func ReplicaSetFromVMI(name string, vmi *v1.VirtualMachineInstance, replicas int32) *v1.VirtualMachineInstanceReplicaSet {
	rs := &v1.VirtualMachineInstanceReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VirtualMachineInstanceReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: vmi.ObjectMeta.Labels,
			},
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return rs
}

func DefaultReplicaSet(replicas int32) (*v1.VirtualMachineInstanceReplicaSet, *v1.VirtualMachineInstance) {
	vmi := v1.NewMinimalVMI("testvmi")
	vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
	rs := ReplicaSetFromVMI("rs", vmi, replicas)
	vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
	return rs, vmi
}
