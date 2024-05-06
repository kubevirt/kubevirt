package watch

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	gomegaTypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/testing"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/pointer"
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
		var virtClientset *fake.Clientset
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
			virtClientset = fake.NewSimpleClientset()
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

			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
			virtClient.EXPECT().ReplicaSet(metav1.NamespaceDefault).Return(virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault)).AnyTimes()

			testing.PrependGenerateNameCreateReactor(&virtClientset.Fake, "virtualmachineinstances")
			syncCaches(stop)
		})

		addReplicaSet := func(rs *v1.VirtualMachineInstanceReplicaSet) {
			mockQueue.ExpectAdds(1)
			rsSource.Add(rs)
			mockQueue.Wait()
			_, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(rs.Namespace).Create(context.Background(), rs, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		addVMI := func(vmi *v1.VirtualMachineInstance) {
			Expect(vmiInformer.GetStore().Add(vmi)).To(Succeed())
			_, err := virtClientset.KubevirtV1().VirtualMachineInstances(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		expectVMIReplicas := func(rs *v1.VirtualMachineInstanceReplicaSet, matcher gomegaTypes.GomegaMatcher) {
			vmiList, err := virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			var rsVMIs []*v1.VirtualMachineInstance
			for _, vmi := range vmiList.Items {
				for _, or := range vmi.OwnerReferences {
					if equality.Semantic.DeepEqual(or, OwnerRef(rs)) {
						rsVMIs = append(rsVMIs, &vmi)
						break
					}
				}
			}
			Expect(rsVMIs).To(matcher)
		}

		It("should create missing VMIs", func() {
			rs, _ := DefaultReplicaSet(3)
			addReplicaSet(rs)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			expectVMIReplicas(rs, HaveLen(3))
		})

		It("should create missing VMIs when it gets unpaused", func() {
			rs, _ := DefaultReplicaSet(3)
			rs.Spec.Paused = false
			rs.Status.Conditions = []v1.VirtualMachineInstanceReplicaSetCondition{
				{
					Type:               v1.VirtualMachineInstanceReplicaSetReplicaPaused,
					LastTransitionTime: metav1.Now(),
				},
			}
			addReplicaSet(rs)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulResumedReplicaSetReason)
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.Conditions).To(BeNil())
			expectVMIReplicas(rs, HaveLen(3))
		})

		It("should not create missing VMIs when it is paused and add paused condition", func() {
			rs, _ := DefaultReplicaSet(3)
			rs.Spec.Paused = true
			// This will trigger a status update, since there are no replicas present
			rs.Status.Replicas = 1
			addReplicaSet(rs)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulPausedReplicaSetReason)
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.Conditions).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Type":   Equal(v1.VirtualMachineInstanceReplicaSetReplicaPaused),
					"Status": Equal(k8sv1.ConditionTrue)},
				),
			))
			expectVMIReplicas(rs, BeEmpty())
		})

		It("should create missing VMIs in batches of a maximum of 10 VMIs at once", func() {
			rs, _ := DefaultReplicaSet(15)
			addReplicaSet(rs)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			}
			expectVMIReplicas(rs, HaveLen(10))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))

			// TODO test for missing 5
		})

		It("should already create replacement VMIs once it discovers that a VMI is marked for deletion", func() {
			rs, _ := DefaultReplicaSet(10)
			addReplicaSet(rs)
			// Add 3 VMIs to the cache which are already running
			for x := 0; x < 3; x++ {
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				addVMI(vmi)
			}
			// Add 3 VMIs to the cache which are marked for deletion
			for x := 3; x < 6; x++ {
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi.DeletionTimestamp = pointer.P(metav1.Now())
				addVMI(vmi)
			}

			// Clear actions, so we can track what the controller does
			virtClientset.ClearActions()
			controller.Execute()

			for x := 0; x < 7; x++ {
				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			}
			// There will be 7 created vms, 3 are already there and 3 are there but marked for deletion
			expectVMIReplicas(rs, HaveLen(13))
		})

		It("should delete missing VMIs in batches of a maximum of 10 VMIs at once", func() {
			rs, _ := DefaultReplicaSet(0)
			addReplicaSet(rs)

			// Add 15 VMIs to the cache
			for x := 0; x < 15; x++ {
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				addVMI(vmi)
			}

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}
			expectVMIReplicas(rs, HaveLen(5))
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
				addVMI(vmi)
			}
			// Add 4 VMIs with deletion timestamp
			for x := 5; x < 9; x++ {
				vmi := api.NewMinimalVMI(fmt.Sprintf("testvmi%d", x))
				vmi.ObjectMeta.Labels = map[string]string{"test": "test"}
				vmi.OwnerReferences = []metav1.OwnerReference{OwnerRef(rs)}
				vmi.DeletionTimestamp = pointer.P(metav1.Now())
				addVMI(vmi)
			}

			// Clear actions, so we can track what the controller does
			virtClientset.ClearActions()
			controller.Execute()

			for x := 0; x < 2; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}
			// There will be 3 are already there and 4 are there but marked for deletion
			expectVMIReplicas(rs, HaveLen(7))
		})

		It("should ignore non-matching VMIs", func() {
			rs, _ := DefaultReplicaSet(3)
			addReplicaSet(rs)

			nonMatchingVMI := api.NewMinimalVMI("testvmi1")
			nonMatchingVMI.ObjectMeta.Labels = map[string]string{"test": "test1"}
			addVMI(nonMatchingVMI)

			controller.Execute()

			// We still expect three calls to create VMIs, since VirtualMachineInstance does not meet the requirements
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			expectVMIReplicas(rs, HaveLen(3))
		})

		It("should delete a VirtualMachineInstance and increase the replica count", func() {
			rs, vmi := DefaultReplicaSet(0)
			rs.Status.Replicas = 0
			addReplicaSet(rs)
			addVMI(vmi)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			expectVMIReplicas(rs, BeEmpty())
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(1)))
		})

		It("should detect that a VirtualMachineInstance already exists and adopt it", func() {
			rs, vmi := DefaultReplicaSet(1)
			vmi.OwnerReferences = []metav1.OwnerReference{}
			rs.Status.Replicas = 1
			addReplicaSet(rs)
			addVMI(vmi)

			controller.Execute()

			vmiList, err := virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmiList.Items).To(HaveLen(1))
			Expect(vmiList.Items[0].OwnerReferences).To(HaveLen(1))
			Expect(vmiList.Items[0].OwnerReferences[0].Kind).To(Equal("VirtualMachineInstanceReplicaSet"))
			Expect(vmiList.Items[0].OwnerReferences[0].Name).To(Equal("rs"))
			Expect(vmiList.Items[0].OwnerReferences[0].UID).To(BeEquivalentTo("rs"))
		})

		It("should detect that it has nothing to do", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1

			addReplicaSet(rs)
			addVMI(vmi)

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
			addVMI(vmi)

			controller.Execute()

			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.LabelSelector).To(Equal(s.String()))
			expectVMIReplicas(rs, HaveLen(1))
		})

		It("should be woken by a stopped VirtualMachineInstance and create a new one", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)

			addReplicaSet(rs)
			addVMI(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.Status.Phase = v1.Succeeded
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Run the controller again
			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulDeleteVirtualMachineReason, SuccessfulCreateVirtualMachineReason)
			_, err := virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			expectVMIReplicas(rs, HaveLen(1))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))
		})

		It("should be woken by a ready VirtualMachineInstance and update the readyReplicas counter", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 0
			addReplicaSet(rs)
			addVMI(vmi)

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

			expectVMIReplicas(rs, HaveLen(1))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(1)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(1)))
		})

		It("should be woken by a not ready but running VirtualMachineInstance and update the readyReplicas counter", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)
			addReplicaSet(rs)
			addVMI(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			markAsNonReady(modifiedVMI)
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Run the controller again
			controller.Execute()

			expectVMIReplicas(rs, HaveLen(1))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(1)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))
		})

		It("should be woken by a deleted VirtualMachineInstance and create a new one", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			addReplicaSet(rs)
			addVMI(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Delete one VirtualMachineInstance
			err := virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Delete(context.TODO(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmiFeeder.Delete(vmi)

			// Run the controller again
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			expectVMIReplicas(rs, HaveLen(1))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			// The count stays at zero, since we just experienced the delete when we create the new VirtualMachineInstance
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
		})

		It("should delete VirtualMachineInstance in the final state", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)
			addReplicaSet(rs)
			addVMI(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.Status.Phase = v1.Failed
			modifiedVMI.ResourceVersion = "1"
			vmiFeeder.Modify(modifiedVMI)

			// Run the cleanFinishedVmis method
			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			_, err := virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			expectVMIReplicas(rs, HaveLen(1))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))
		})

		It("should delete VirtualMachineIstance in unknown state", func() {
			rs, vmi := DefaultReplicaSet(1)
			rs.Status.Replicas = 1
			rs.Status.ReadyReplicas = 1
			vmi.Status.Phase = v1.Running
			markAsReady(vmi)
			addReplicaSet(rs)
			addVMI(vmi)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VirtualMachineInstance to a final state
			modifiedVMI := vmi.DeepCopy()
			modifiedVMI.ResourceVersion = "1"
			markAsPodTerminating(modifiedVMI)
			vmiFeeder.Modify(modifiedVMI)

			// Run the cleanFinishedVmis method
			controller.Execute()
			_, err := virtClientset.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault).Get(context.TODO(), vmi.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			expectVMIReplicas(rs, HaveLen(1))
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(0)))
			Expect(updatedRS.Status.ReadyReplicas).To(Equal(int32(0)))

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
		})

		It("should add a fail condition if scaling up fails", func() {
			rs, vmi := DefaultReplicaSet(3)
			addReplicaSet(rs)
			addVMI(vmi)

			// Let first one succeed
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any(), metav1.CreateOptions{}).Return(vmi, nil)
			// Let second one fail
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any(), metav1.CreateOptions{}).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
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
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(1)))
			Expect(updatedRS.Status.Conditions).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Type":    Equal(v1.VirtualMachineInstanceReplicaSetReplicaFailure),
					"Reason":  Equal("FailedCreate"),
					"Message": Equal("failure"),
					"Status":  Equal(k8sv1.ConditionTrue)},
				),
			))
		})

		It("should add a fail condition if scaling down fails", func() {
			rs, vmi := DefaultReplicaSet(0)
			vmi1 := vmi.DeepCopy()
			vmi1.ObjectMeta.Name = "test1"
			addReplicaSet(rs)
			addVMI(vmi)
			addVMI(vmi1)

			// Let first one succeed
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			// Let second one fail
			vmiInterface.EXPECT().Delete(context.Background(), vmi1.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 2
			rsInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
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
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			// We should see the failed condition, replicas should stay at 2
			Expect(updatedRS.Status.Replicas).To(Equal(int32(2)))
			Expect(updatedRS.Status.Conditions).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Type":    Equal(v1.VirtualMachineInstanceReplicaSetReplicaFailure),
					"Reason":  Equal("FailedDelete"),
					"Message": Equal("failure"),
					"Status":  Equal(k8sv1.ConditionTrue)},
				),
			))
		})

		It("should back off if a sync error occurs", func() {
			rs, vmi := DefaultReplicaSet(0)
			vmi1 := vmi.DeepCopy()
			vmi1.ObjectMeta.Name = "test1"
			addReplicaSet(rs)
			addVMI(vmi)
			addVMI(vmi1)

			// Let first one succeed
			vmiInterface.EXPECT().Delete(context.Background(), vmi.ObjectMeta.Name, gomock.Any()).Return(nil)
			// Let second one fail
			vmiInterface.EXPECT().Delete(context.Background(), vmi1.ObjectMeta.Name, gomock.Any()).Return(fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 2
			rsInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
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
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			// We should see the failed condition, replicas should stay at 2
			Expect(updatedRS.Status.Replicas).To(Equal(int32(2)))
			Expect(updatedRS.Status.Conditions).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Type":    Equal(v1.VirtualMachineInstanceReplicaSetReplicaFailure),
					"Reason":  Equal("FailedDelete"),
					"Message": Equal("failure"),
					"Status":  Equal(k8sv1.ConditionTrue)},
				),
			))
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
			addVMI(vmi)

			// Let first one succeed
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any(), metav1.CreateOptions{}).Return(vmi, nil)
			// Let second one fail
			vmiInterface.EXPECT().Create(context.Background(), gomock.Any(), metav1.CreateOptions{}).Return(nil, fmt.Errorf("failure"))

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().UpdateStatus(context.Background(), gomock.Any(), metav1.UpdateOptions{}).Do(func(ctx context.Context, obj interface{}, options metav1.UpdateOptions) {
				objRS := obj.(*v1.VirtualMachineInstanceReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(1)))
				Expect(objRS.Status.Conditions).To(HaveLen(1))
				cond := objRS.Status.Conditions[0]
				Expect(cond.Message).To(Equal(rs.Status.Conditions[0].Message))
				Expect(cond.LastTransitionTime).To(Equal(rs.Status.Conditions[0].LastTransitionTime))
			})

			controller.Execute()

			testutils.ExpectEvents(recorder, SuccessfulCreateVirtualMachineReason, FailedCreateVirtualMachineReason)
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(1)))
			Expect(updatedRS.Status.Conditions).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Type":               Equal(v1.VirtualMachineInstanceReplicaSetReplicaFailure),
					"Message":            Equal(rs.Status.Conditions[0].Message),
					"LastTransitionTime": Equal(rs.Status.Conditions[0].LastTransitionTime)},
				),
			))
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
			addVMI(vmi)

			controller.Execute()

			Expect(mockQueue.GetRateLimitedEnqueueCount()).To(Equal(0))
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			updatedRS, err := virtClientset.KubevirtV1().VirtualMachineInstanceReplicaSets(metav1.NamespaceDefault).Get(context.TODO(), rs.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(updatedRS).ToNot(BeNil())
			Expect(updatedRS.Status.Replicas).To(Equal(int32(1)))
			Expect(updatedRS.Status.Conditions).To(BeEmpty())
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
