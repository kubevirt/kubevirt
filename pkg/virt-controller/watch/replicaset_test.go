package watch_test

import (
	. "kubevirt.io/kubevirt/pkg/virt-controller/watch"

	"github.com/golang/mock/gomock"
	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/cache/testing"

	"fmt"

	v13 "k8s.io/api/core/v1"

	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

var _ = Describe("Replicaset", func() {

	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var vmInterface *kubecli.MockVMInterface
	var rsInterface *kubecli.MockReplicaSetInterface
	var vmSource *framework.FakeControllerSource
	var rsSource *framework.FakeControllerSource
	var vmInformer cache.SharedIndexInformer
	var rsInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *VMReplicaSet
	var recorder *record.FakeRecorder

	sync := func(stop chan struct{}) {
		go vmInformer.Run(stop)
		go rsInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, vmInformer.HasSynced, rsInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVMInterface(ctrl)
		rsInterface = kubecli.NewMockReplicaSetInterface(ctrl)

		vmSource = framework.NewFakeControllerSource()
		rsSource = framework.NewFakeControllerSource()
		vmInformer = cache.NewSharedIndexInformer(vmSource, &v1.VirtualMachine{}, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		rsInformer = cache.NewSharedIndexInformer(rsSource, &v1.VirtualMachineReplicaSet{}, 0, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		recorder = record.NewFakeRecorder(100)

		controller = NewVMReplicaSet(vmInformer, rsInformer, recorder, virtClient)
	})

	Context("One valid ReplicaSet controller given", func() {

		It("should create missing VMs and increase the replica count", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 3)

			expectedRS := clone(rs)
			expectedRS.Status.Replicas = 3

			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface).AnyTimes()
			vmInterface.EXPECT().Create(gomock.Any()).Times(3).Do(func(arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal("testvm"))
			})
			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()
		})

		It("should ignore non-matching VMs", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 3)

			expectedRS := clone(rs)
			expectedRS.Status.Replicas = 3

			// We still expect three calls to create VMs, since VM does not meet the requirements
			vm.ObjectMeta.Labels = map[string]string{"test": "test1"}
			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface).AnyTimes()
			vmInterface.EXPECT().Create(gomock.Any()).Times(3)
			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()
		})

		It("should delete a VM and decrease the replica count", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 0)
			rs.Status.Replicas = 1

			expectedRS := clone(rs)
			expectedRS.Status.Replicas = 0

			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface)
			vmInterface.EXPECT().Delete(vm.ObjectMeta.Name, gomock.Any())
			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)
			rsInterface.EXPECT().Update(expectedRS)

			controller.Execute()
		})

		It("should detect that it has nothing to do", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 1)
			rs.Status.Replicas = 1

			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)
			rsInterface.EXPECT().Update(rs)
			controller.Execute()
		})

		It("should be woken by a stopped VM and create a new one", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 1)
			rs.Status.Replicas = 1

			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface).Times(2)
			rsInterface.EXPECT().Update(rs).Times(2)

			// First make sure that we don't have to do anything
			controller.Execute()

			// Move one VM to a final state
			vm.Status.Phase = v1.Succeeded
			vmSource.Modify(vm)

			// Expect the recrate of the VM
			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface)
			vmInterface.EXPECT().Create(gomock.Any())

			// Run the controller again
			controller.Execute()
		})

		It("should add a fail condition if scaling fails", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 3)

			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface).Times(3)
			// Let first one succeed
			vmInterface.EXPECT().Create(gomock.Any())
			// Let second one fail
			vmInterface.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("failure"))

			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)

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
		})

		It("should update the replica count but keep the failed state", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 3)
			rs.Status.Conditions = []v1.VMReplicaSetCondition{
				{
					Type:               v1.VMReplicaSetReplicaFailure,
					LastTransitionTime: v12.Now(),
					Message:            "test",
				},
			}

			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface).Times(3)
			// Let first one succeed
			vmInterface.EXPECT().Create(gomock.Any())
			// Let second one fail
			vmInterface.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("failure"))

			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)

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
		})
		It("should update the replica count and remove the failed condition", func() {
			vm := v1.NewMinimalVM("testvm")
			vm.ObjectMeta.Labels = map[string]string{"test": "test"}
			rs := ReplicaSetFromVM("rs", vm, 3)
			rs.Status.Conditions = []v1.VMReplicaSetCondition{
				{
					Type:               v1.VMReplicaSetReplicaFailure,
					LastTransitionTime: v12.Now(),
					Message:            "test",
				},
			}

			vmSource.Add(vm)
			rsSource.Add(rs)

			sync(stop)

			virtClient.EXPECT().VM(vm.ObjectMeta.Namespace).Return(vmInterface).Times(3)
			vmInterface.EXPECT().Create(gomock.Any()).Times(2)

			virtClient.EXPECT().ReplicaSet(vm.ObjectMeta.Namespace).Return(rsInterface)

			// We should see the failed condition, replicas should stay at 0
			rsInterface.EXPECT().Update(gomock.Any()).Do(func(obj interface{}) {
				objRS := obj.(*v1.VirtualMachineReplicaSet)
				Expect(objRS.Status.Replicas).To(Equal(int32(3)))
				Expect(objRS.Status.Conditions).To(HaveLen(0))
			})

			controller.Execute()
		})

	})
})

func clone(rs *v1.VirtualMachineReplicaSet) *v1.VirtualMachineReplicaSet {
	c, err := model.Clone(rs)
	Expect(err).ToNot(HaveOccurred())
	return c.(*v1.VirtualMachineReplicaSet)
}

func ReplicaSetFromVM(name string, vm *v1.VirtualMachine, replicas int32) *v1.VirtualMachineReplicaSet {
	rs := &v1.VirtualMachineReplicaSet{
		ObjectMeta: v12.ObjectMeta{Name: name, Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: v1.VMReplicaSetSpec{
			Replicas: &replicas,
			Selector: &v12.LabelSelector{
				MatchLabels: vm.ObjectMeta.Labels,
			},
			Template: &v1.VMTemplateSpec{Spec: vm.Spec},
		},
	}
	return rs
}
