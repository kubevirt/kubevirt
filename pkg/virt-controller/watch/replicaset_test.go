package watch

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Replicaset", func() {

	DescribeTable("Different replica diffs given", func(diff int, burstReplicas int, result int) {
		Expect(limit(diff, uint(burstReplicas))).To(Equal(result))
	},
		Entry("should limit negative diff to negative burst maximum", -10, 5, -5),
		Entry("should return negative diff if bigger than negative burst maximum", -4, 5, -4),
		Entry("should limit positive diff to positive burst maximum", 10, 5, 5),
		Entry("should return positive diff if less than positive burst maximum", 4, 5, 4),
		Entry("should return 0 for zero diff", 0, 5, 0),
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
			recorder.IncludeObject = true

			controller, _ = NewVMIReplicaSet(vmiInformer, rsInformer, recorder, virtClient, uint(10))
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

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Do(func(ctx context.Context, arg interface{}) {
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

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachineInstance).ObjectMeta.GenerateName).To(Equal("testvmi"))
			}).Return(vmi, nil)
			rsInterface.EXPECT().UpdateStatus(expectedRS)

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
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(0)

			// Synchronizing the state is expected
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Times(1).Do(func(obj *v1.VirtualMachineInstanceReplicaSet) {
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

			rsInterface.EXPECT().UpdateStatus(gomock.Any()).AnyTimes()

			// Check if only 10 are created
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(10).Do(func(ctx context.Context, arg interface{}) {
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
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmiFeeder.Add(vmi)
			}

			// Add 3 VMIs to the cache which are marked for deletion
			for x := 3; x < 6; x++ {
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi.DeletionTimestamp = now()
				vmiFeeder.Add(vmi)
			}

			rsInterface.EXPECT().UpdateStatus(gomock.Any()).AnyTimes()

			// Should create 7 vms, 3 are already there and 3 are there but marked for deletion
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(7).Do(func(ctx context.Context, arg interface{}) {
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
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmiFeeder.Add(vmi)
			}

			rsInterface.EXPECT().UpdateStatus(gomock.Any()).AnyTimes()

			// Check if only 10 are deleted
			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).
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
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmiFeeder.Add(vmi)
			}
			// Add 5 VMIs with deletion timestamp
			for x := 5; x < 9; x++ {
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi.DeletionTimestamp = now()
				vmiFeeder.Add(vmi)
			}

			rsInterface.EXPECT().UpdateStatus(gomock.Any()).AnyTimes()

			// Check if only two vms get deleted
			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).
				Times(2).Return(nil)

			controller.Execute()

			for x := 0; x < 2; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}
		})

		It("should ignore non-matching VMIs", func() {
			rs, vmi := DefaultReplicaSet(3)

			nonMatchingVMI := api.NewMinimalVMI("testvmi1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}

			addReplicaSet(rs)

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			vmiSource.Add(nonMatchingVMI)

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Return(vmi, nil)

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

			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any())
			rsInterface.EXPECT().UpdateStatus(expectedRS)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			rs, vmi := DefaultReplicaSet(1)
			vmi.OwnerReferences = []metav1.OwnerReference{}

			rs.Status.Replicas = 1

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			rsInterface.EXPECT().Get(rs.ObjectMeta.Name, gomock.Any()).Return(rs, nil)
			vmiInterface.EXPECT().Patch(context.Background(), vmi.ObjectMeta.Name, gomock.Any(), gomock.Any(), &metav1.PatchOptions{})

			controller.Execute()
		})

		It("should detect that it has nothing to do", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			controller.Execute()
		})

		It("should detect that it has to update the labelSelector in the status", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.LabelSelector = ""
			s, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
				MatchLabels: vmi.ObjectMeta.Labels,
			})
			Expect(err).ToNot(HaveOccurred())
			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
				Expect(objRS.Status.LabelSelector).To(Equal(s.String()))
			})
			controller.Execute()
		})

		It("should be woken by a stopped VirtualMachineInstance and create a new one", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)

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
			rsInterface.EXPECT().UpdateStatus(rsCopy).Times(1)
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)
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

			rsInterface.EXPECT().UpdateStatus(expectedRS).Times(1)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.Status.Phase = v1.Running
			markAsReady(modifiedVMI)
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Run the controller again
			controller.Execute()
		})

		It("should be woken by a not ready but running VirtualMachineInstance and update the readyReplicas counter", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)

			expectedRS := rs.DeepCopy()
			expectedRS.Status.Replicas = 1
			expectedRS.Status.ReadyReplicas = 0

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			rsInterface.EXPECT().UpdateStatus(expectedRS).Times(1)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			markAsNonReady(modifiedVMI)
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
			rsInterface.EXPECT().UpdateStatus(rsCopy).Times(1)

			// Expect the recrate of the VirtualMachineInstance
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)

			// Run the controller again
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should delete VirtualMachineIstance in the final state", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)

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
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
				return vmi, nil
			})
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			rsInterface.EXPECT().UpdateStatus(gomock.Any())

			// Run the cleanFinishedVmis method
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should delete VirtualMachineIstance in unknown state", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.ResourceVersion = "1"
			markAsPodTerminating(modifiedVMI)
			vmiFeeder.Modify(modifiedVMI)

			// Expect the re-crate of the VirtualMachineInstance
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).DoAndReturn(func(ctx context.Context, vmi *v1.VirtualMachineInstance) (*v1.VirtualMachineInstance, error) {
				return vmi, nil
			})
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			rsInterface.EXPECT().UpdateStatus(gomock.Any())

			// Run the cleanFinishedVmis method
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should add a fail condition if scaling up fails", func() {
			rs, vmi := DefaultReplicaSet(3)

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)

			// Let first one succeed
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)
			// Let second one fail
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			// Let second one fail
			vmiInterface.EXPECT().Delete(context.Background(), vmi1.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 2
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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
		It("should back off if a sync error occurs", func() {
			rs, vmi := DefaultReplicaSet(0)
			vmi1 := vmi.DeepCopy()
			vmi1.ObjectMeta.Name = "test1"

			addReplicaSet(rs)
			vmiFeeder.Add(vmi)
			vmiFeeder.Add(vmi1)

			// Let first one succeed
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			// Let second one fail
			vmiInterface.EXPECT().Delete(context.Background(), vmi1.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 2
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(1))
			Expect(mockQueue.Len()).To(Equal(0))

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
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(vmi, nil)
			// Let second one fail
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
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

			vmiInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(2).Return(vmi, nil)

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().UpdateStatus(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(1)))
				Expect(objRS.Status.Conditions).To(BeEmpty())
			})

			controller.Execute()
			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		AfterEach(func() {
			close(stop)
			// Ensure that we add checks for expected events to every test
			Expect(recorder.Events).To(BeEmpty())
		})
	})
})

func ReplicaSetFromVMI(name string, vmi *v1.VirtualMachineInstance, replicas int32) *v1.VirtualMachineInstanceReplicaSet {
	s, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: vmi.ObjectMeta.Labels,
	})
	Expect(err).ToNot(HaveOccurred())
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
		Status: v1.VirtualMachineInstanceReplicaSetStatus{LabelSelector: s.String()},
	}
	return rs
}

func DefaultReplicaSet(replicas int32) (*v1.VirtualMachineInstanceReplicaSet, *v1.VirtualMachineInstance) {
	vmi := api.NewMinimalVMI("testvmi")
	vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
	rs := ReplicaSetFromVMI("rs", vmi, replicas)
	vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
	virtcontroller.SetLatestApiVersionAnnotation(vmi)
	virtcontroller.SetLatestApiVersionAnnotation(rs)
	return rs, vmi
}
