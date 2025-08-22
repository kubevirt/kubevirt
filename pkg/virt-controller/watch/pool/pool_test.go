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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package pool

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/client-go/testing"

	"kubevirt.io/kubevirt/pkg/libvmi"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	controllertesting "kubevirt.io/kubevirt/pkg/controller/testing"
	"kubevirt.io/kubevirt/pkg/pointer"
	testutils "kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
	watchtesting "kubevirt.io/kubevirt/pkg/virt-controller/watch/testing"
)

var _ = Describe("Pool", func() {

	DescribeTable("Calculate new VM names on scale out", func(existing []string, expected []string, count int) {

		namespace := "test"
		baseName := "my-pool"
		vmInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachine{})

		for _, name := range existing {
			vm, _ := watchtesting.DefaultVirtualMachine(true)
			vm.Name = name
			vm.Namespace = namespace
			vm.GenerateName = ""
			Expect(vmInformer.GetStore().Add(vm)).To(Succeed())
		}

		newNames := calculateNewVMNames(count, baseName, namespace, vmInformer.GetStore())

		Expect(newNames).To(HaveLen(len(expected)))
		Expect(newNames).To(Equal(expected))

	},
		Entry("should fill in name gaps",
			[]string{"my-pool-1", "my-pool-3"},
			[]string{"my-pool-0", "my-pool-2", "my-pool-4"},
			3),
		Entry("should append to end if no name gaps exist",
			[]string{"my-pool-0", "my-pool-1", "my-pool-2"},
			[]string{"my-pool-3", "my-pool-4"},
			2),
	)

	Context("One valid Pool controller given", func() {

		const (
			testNamespace = "default"
		)

		var controller *Controller
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue[string]
		var fakeVirtClient *kubevirtfake.Clientset
		var k8sClient *k8sfake.Clientset

		addCR := func(cr *appsv1.ControllerRevision) {
			controller.revisionIndexer.Add(cr)
			_, err := k8sClient.AppsV1().ControllerRevisions(cr.Namespace).Create(context.TODO(), cr, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		}

		addVM := func(vm *v1.VirtualMachine) {
			_, err := fakeVirtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			controller.vmIndexer.Add(vm)
		}

		addVMI := func(vm *v1.VirtualMachineInstance) {
			_, err := fakeVirtClient.KubevirtV1().VirtualMachineInstances(vm.Namespace).Create(context.TODO(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			controller.vmiStore.Add(vm)
		}

		BeforeEach(func() {
			virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))

			vmiInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmInformer, _ := testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			poolInformer, _ := testutils.NewFakeInformerFor(&poolv1.VirtualMachinePool{})
			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			crInformer, _ := testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
				"vmpool": func(obj interface{}) ([]string, error) {
					cr := obj.(*appsv1.ControllerRevision)
					for _, ref := range cr.OwnerReferences {
						if ref.Kind == "VirtualMachinePool" {
							return []string{string(ref.UID)}, nil
						}
					}
					return nil, nil
				},
			})

			controller, _ = NewController(virtClient,
				vmiInformer,
				vmInformer,
				poolInformer,
				crInformer,
				recorder,
				uint(10))
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.queue)
			controller.queue = mockQueue

			fakeVirtClient = kubevirtfake.NewSimpleClientset()

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(fakeVirtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

			virtClient.EXPECT().VirtualMachinePool(testNamespace).Return(fakeVirtClient.PoolV1alpha1().VirtualMachinePools(testNamespace)).AnyTimes()

			k8sClient = k8sfake.NewSimpleClientset()
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
		})

		addPool := func(pool *poolv1.VirtualMachinePool) {
			_, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Create(context.TODO(), pool, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			controller.poolIndexer.Add(pool)
			key, err := virtcontroller.KeyFunc(pool)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)
		}

		createPoolRevision := func(pool *poolv1.VirtualMachinePool) *appsv1.ControllerRevision {
			bytes, err := json.Marshal(&pool.Spec)
			Expect(err).ToNot(HaveOccurred())

			cr := &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:            getRevisionName(pool),
					Namespace:       pool.Namespace,
					OwnerReferences: []metav1.OwnerReference{poolOwnerRef(pool)},
				},
				Data:     runtime.RawExtension{Raw: bytes},
				Revision: pool.Generation,
			}
			return cr
		}

		createVMsWithOrdinal := func(pool *poolv1.VirtualMachinePool, count int, newPoolRevision *appsv1.ControllerRevision, oldPoolRevision *appsv1.ControllerRevision, vm *v1.VirtualMachine) {
			var updatedPoolSpec poolv1.VirtualMachinePoolSpec
			Expect(json.Unmarshal(newPoolRevision.Data.Raw, &updatedPoolSpec)).To(Succeed())

			for i := range count {
				vmCopy := vm.DeepCopy()
				vmCopy.Name = fmt.Sprintf("%s-%d", pool.Name, i)

				vmCopy.Labels = maps.Clone(updatedPoolSpec.VirtualMachineTemplate.ObjectMeta.Labels)
				vmCopy.Annotations = maps.Clone(updatedPoolSpec.VirtualMachineTemplate.ObjectMeta.Annotations)
				vmCopy.Spec = *indexVMSpec(&updatedPoolSpec, i)
				vmCopy = injectPoolRevisionLabelsIntoVM(vmCopy, newPoolRevision.Name)

				markVmAsReady(vmCopy)
				vmi := createReadyVMI(vmCopy, oldPoolRevision)
				addVM(vmCopy)
				addVMI(vmi)
			}
		}

		sanityExecute := func() {
			controllertesting.SanityExecute(controller, []cache.Store{
				controller.vmiStore, controller.vmIndexer, controller.poolIndexer, controller.revisionIndexer,
			}, Default)
		}

		It("should create missing VMs", func() {
			pool, _ := DefaultPool(3)

			addPool(pool)

			poolRevision := createPoolRevision(pool)

			sanityExecute()

			_, err := k8sClient.AppsV1().ControllerRevisions(pool.Namespace).Get(context.TODO(), poolRevision.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		})

		It("should update VM when VM template changes, but not VMI unless VMI template changes", func() {
			pool, vm := DefaultPool(1)
			pool.Status.Replicas = 1
			pool.Status.ReadyReplicas = 1
			poolRevision := createPoolRevision(pool)

			pool.Generation = 123
			addPool(pool)

			createVMsWithOrdinal(pool, 1, poolRevision, poolRevision, vm)
			addCR(poolRevision)

			sanityExecute()

			vm, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).Get(context.TODO(), fmt.Sprintf("%s-0", pool.Name), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.Labels).To(HaveKeyWithValue(v1.VirtualMachinePoolRevisionName, poolRevision.Name))

			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(BeEmpty())
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "update", "virtualmachineinstances")).To(BeEmpty())
		})

		It("should delete controller revisions when pool is being deleted", func() {
			pool, _ := DefaultPool(1)
			pool.Status.Replicas = 0
			pool.DeletionTimestamp = pointer.P(metav1.Now())

			poolRevision := createPoolRevision(pool)

			addPool(pool)
			addCR(poolRevision)

			sanityExecute()
			_, err := k8sClient.AppsV1().ControllerRevisions(pool.Namespace).Get(context.TODO(), poolRevision.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"))
		})

		It("should update VM and VMI with both VM and VMI template change", func() {
			pool, vm := DefaultPool(1)
			pool.Status.Replicas = 1
			pool.Status.ReadyReplicas = 1

			oldPoolRevision := createPoolRevision(pool)

			pool.Generation = 123

			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = map[string]string{}
			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels["newkey"] = "newval"
			newPoolRevision := createPoolRevision(pool)

			addPool(pool)

			addCR(oldPoolRevision)
			addCR(newPoolRevision)
			createVMsWithOrdinal(pool, 1, newPoolRevision, oldPoolRevision, vm)

			sanityExecute()

			testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(HaveLen(1))
		})

		It("should do nothing", func() {
			pool, vm := DefaultPool(1)
			vm.Name = fmt.Sprintf("%s-0", pool.Name)

			poolRevision := createPoolRevision(pool)
			pool.Status.Replicas = 1
			pool.Status.ReadyReplicas = 1
			addPool(pool)
			createVMsWithOrdinal(pool, 1, poolRevision, poolRevision, vm)

			sanityExecute()

			_, err := k8sClient.AppsV1().ControllerRevisions(pool.Namespace).Get(context.TODO(), poolRevision.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			currentPool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(pool).To(Equal(currentPool))

			Expect(testing.FilterActions(&fakeVirtClient.Fake, "update", "virtualmachinepools")).To(BeEmpty())
		})

		It("should prune unused controller revisions", func() {
			pool, vm := DefaultPool(1)
			vm.Name = fmt.Sprintf("%s-0", pool.Name)

			poolRevision := createPoolRevision(pool)
			vm = injectPoolRevisionLabelsIntoVM(vm, poolRevision.Name)
			markVmAsReady(vm)

			oldPoolRevision := poolRevision.DeepCopy()
			oldPoolRevision.Name = "madeup"
			pool.Status.Replicas = 1
			pool.Status.ReadyReplicas = 1
			addPool(pool)

			createVMsWithOrdinal(pool, 1, poolRevision, poolRevision, vm)

			addCR(poolRevision)
			addCR(oldPoolRevision)

			sanityExecute()

			_, err := k8sClient.AppsV1().ControllerRevisions(pool.Namespace).Get(context.TODO(), oldPoolRevision.Name, metav1.GetOptions{})
			Expect(err).To(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"))

			_, err = k8sClient.AppsV1().ControllerRevisions(pool.Namespace).Get(context.TODO(), poolRevision.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		It("should not create missing VMs when it is paused and add paused condition", func() {
			pool, _ := DefaultPool(3)
			pool.Spec.Paused = true

			addPool(pool)

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(0)))
			Expect(vmpool.Status.Conditions).To(HaveLen(1))
			Expect(vmpool.Status.Conditions[0].Type).To(Equal(poolv1.VirtualMachinePoolReplicaPaused))
			Expect(vmpool.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionTrue))

			testutils.ExpectEvent(recorder, SuccessfulPausedPoolReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(BeEmpty())
		})

		It("should set failed condition when reconcile error occurs", func() {
			pool, _ := DefaultPool(3)

			addPool(pool)

			fakeVirtClient.Fake.PrependReactor("create", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, &v1.VirtualMachine{}, fmt.Errorf("error")
			})

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(0)))
			Expect(vmpool.Status.Conditions).To(HaveLen(1))
			Expect(vmpool.Status.Conditions[0].Type).To(Equal(poolv1.VirtualMachinePoolReplicaFailure))
			Expect(vmpool.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionTrue))

			testutils.ExpectEvent(recorder, common.FailedCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, FailedScaleOutReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		})

		It("should remove failed condition when no reconcile error occurs", func() {
			pool, _ := DefaultPool(0)
			pool.Status.Conditions = append(pool.Status.Conditions,
				poolv1.VirtualMachinePoolCondition{
					Type:               poolv1.VirtualMachinePoolReplicaFailure,
					Reason:             "",
					Message:            "",
					LastTransitionTime: metav1.Now(),
					Status:             k8sv1.ConditionTrue,
				})

			addPool(pool)

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(0)))
			Expect(vmpool.Status.Conditions).To(Not(ContainElement(
				HaveField("Type", Equal(poolv1.VirtualMachinePoolReplicaFailure)),
			)))
		})

		It("should create missing VMs when pool is resumed", func() {
			pool, _ := DefaultPool(3)

			pool.Status.Conditions = append(pool.Status.Conditions,
				poolv1.VirtualMachinePoolCondition{
					Type:               poolv1.VirtualMachinePoolReplicaPaused,
					Reason:             "",
					Message:            "",
					LastTransitionTime: metav1.Now(),
					Status:             k8sv1.ConditionTrue,
				})

			addPool(pool)

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Conditions).To(Not(ContainElement(
				HaveField("Type", Equal(poolv1.VirtualMachinePoolReplicaPaused)),
			)))

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulResumePoolReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		})

		It("should create missing VMs in batches of a maximum of burst replicas VMs at once", func() {
			pool, _ := DefaultPool(15)

			addPool(pool)

			sanityExecute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			}
			// Check if only 10 are created
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(10))
		})

		It("should delete VMs in batches of a maximum of burst replicas", func() {
			pool, vm := DefaultPool(0)

			addPool(pool)

			poolRevision := createPoolRevision(pool)

			createVMsWithOrdinal(pool, 15, poolRevision, poolRevision, vm)

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(15)))

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			}
			// Check if only 10 are deleted
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachines")).To(HaveLen(10))
		})

		It("should not delete vms which are already marked deleted", func() {

			pool, vm := DefaultPool(0)

			addPool(pool)

			poolRevision := createPoolRevision(pool)
			createVMsWithOrdinal(pool, 10, poolRevision, poolRevision, vm)

			vms, err := controller.listVMsFromNamespace(pool.Namespace)
			Expect(err).ToNot(HaveOccurred())
			for i, vm := range vms {
				if i >= 5 {
					break
				}
				vm.DeletionTimestamp = pointer.P(metav1.Now())
				_, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(10)))

			newVMs, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMs.Items).To(HaveLen(5))

			for x := 0; x < 5; x++ {
				testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			}
			// Check if only 5 are deleted
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachines")).To(HaveLen(5))
		})

		It("should ignore and skip the name for non-matching VMs", func() {

			pool, vm := DefaultPool(3)

			nonMatchingVM := vm.DeepCopy()
			nonMatchingVM.Name = fmt.Sprintf("%s-1", pool.Name)
			nonMatchingVM.Labels = map[string]string{"madeup": "value"}
			nonMatchingVM.OwnerReferences = []metav1.OwnerReference{}
			controller.vmIndexer.Add(nonMatchingVM)
			_, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).Create(context.TODO(), nonMatchingVM, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			addPool(pool)

			sanityExecute()

			vms, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vms.Items).To(HaveLen(4))
			Expect(vms.Items[3].Name).To(Equal(fmt.Sprintf("%s-3", pool.Name)))

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(4))
		})

		It("should ignore orphaned VMs even when selector matches", func() {
			pool, vm := DefaultPool(3)
			vm.OwnerReferences = []metav1.OwnerReference{}

			// Orphaned VM won't cause key to enqueue
			controller.vmIndexer.Add(vm)
			addPool(pool)

			poolRevision := createPoolRevision(pool)

			createVMsWithOrdinal(pool, 3, poolRevision, poolRevision, vm)

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(0)))
		})

		It("should detect a VM is detached, then release and replace it", func() {
			pool, vm := DefaultPool(3)

			addPool(pool)
			poolRevision := createPoolRevision(pool)

			createVMsWithOrdinal(pool, 3, poolRevision, poolRevision, vm)

			vm, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).Get(context.TODO(), fmt.Sprintf("%s-1", pool.Name), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Remove OwnerReferences and change labels so the VM is truly detached
			vm.OwnerReferences = []metav1.OwnerReference{}
			_, err = fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).Update(context.TODO(), vm, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Update the controller's cache to reflect the changes
			controller.vmIndexer.Update(vm)

			sanityExecute()

			vms, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vms.Items).To(HaveLen(4))

			for _, vm := range vms.Items {
				if vm.Name == fmt.Sprintf("%s-1", pool.Name) {
					Expect(vm.OwnerReferences).To(BeEmpty())
					continue
				}
				Expect(vm.OwnerReferences[0].Kind).To(Equal("VirtualMachinePool"))
			}

			Expect(testing.FilterActions(&fakeVirtClient.Fake, "update", "virtualmachinepools")).To(HaveLen(1))
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(4))
		})

		It("should delete VMs with highest ordinals first when using DescendingOrder base scale-in strategy", func() {
			pool, vm := DefaultPool(2)

			basePolicy := poolv1.VirtualMachinePoolBasePolicyDescendingOrder
			pool.Spec.ScaleInStrategy = &poolv1.VirtualMachinePoolScaleInStrategy{
				Proactive: &poolv1.VirtualMachinePoolProactiveScaleInStrategy{
					SelectionPolicy: &poolv1.VirtualMachinePoolSelectionPolicy{
						BasePolicy: &basePolicy,
					},
				},
			}

			addPool(pool)

			poolRevision := createPoolRevision(pool)

			createVMsWithOrdinal(pool, 5, poolRevision, poolRevision, vm)

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Replicas).To(Equal(int32(5)))

			vms, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vms.Items).To(HaveLen(2))
			Expect(vms.Items).To(ContainElements(
				HaveField("Name", Equal(fmt.Sprintf("%s-0", pool.Name))),
				HaveField("Name", Equal(fmt.Sprintf("%s-1", pool.Name))),
			))

			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachines")).To(HaveLen(3))
		})

		DescribeTable("should respect name generation settings", func(appendIndex *bool) {
			const (
				cmName     = "configmap"
				secretName = "secret"
			)

			pool, _ := DefaultPool(3)
			pool.Spec.VirtualMachineTemplate.Spec.Template.Spec.Volumes = []v1.Volume{
				{
					Name: cmName,
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: k8sv1.LocalObjectReference{Name: cmName},
						},
					},
				},
				{
					Name: secretName,
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{SecretName: secretName},
					},
				},
			}
			pool.Spec.NameGeneration = &poolv1.VirtualMachinePoolNameGeneration{
				AppendIndexToConfigMapRefs: appendIndex,
				AppendIndexToSecretRefs:    appendIndex,
			}

			addPool(pool)

			poolRevision := createPoolRevision(pool)

			sanityExecute()

			cr, err := k8sClient.AppsV1().ControllerRevisions(pool.Namespace).Get(context.TODO(), poolRevision.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cr.Name).To(Equal(poolRevision.Name))

			vms, err := fakeVirtClient.KubevirtV1().VirtualMachines(pool.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vms.Items).To(HaveLen(3))

			for i, vm := range vms.Items {
				if appendIndex != nil && *appendIndex {
					Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).To(MatchRegexp(fmt.Sprintf("%s-%d", cmName, i)))
					Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName).To(MatchRegexp(fmt.Sprintf("%s-%d", secretName, i)))
				} else {
					Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).To(Equal(cmName))
					Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName).To(Equal(secretName))
				}
			}

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		},
			Entry("do not append index by default", nil),
			Entry("do not append index if set to false", pointer.P(false)),
			Entry("append index if set to true", pointer.P(true)),
		)

		It("should respect maxUnavailable limit during proactive updates", func() {
			pool, vm := DefaultPool(4)
			pool.Status.Replicas = 4
			pool.Status.ReadyReplicas = 4
			maxUnavailable := intstr.FromString("25%")
			pool.Spec.MaxUnavailable = &maxUnavailable

			oldPoolRevision := createPoolRevision(pool)

			pool.Generation = 123

			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = map[string]string{}
			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels["newkey"] = "newval"
			newPoolRevision := createPoolRevision(pool)

			addPool(pool)
			addCR(oldPoolRevision)
			addCR(newPoolRevision)

			createVMsWithOrdinal(pool, 4, newPoolRevision, oldPoolRevision, vm)

			sanityExecute()
			crs, err := k8sClient.AppsV1().ControllerRevisions(pool.Namespace).List(context.TODO(), metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(crs.Items).To(HaveLen(2))
			Expect(crs.Items[0].Name).To(Equal(oldPoolRevision.Name))
			Expect(crs.Items[1].Name).To(Equal(newPoolRevision.Name))

			testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(HaveLen(1))
		})

		It("should not trigger proactive update when VMs are not ready", func() {
			pool, vm := DefaultPool(4)
			pool.Status.Replicas = 4
			pool.Status.ReadyReplicas = 4
			maxUnavailable := intstr.FromString("25%")
			pool.Spec.MaxUnavailable = &maxUnavailable

			oldPoolRevision := createPoolRevision(pool)

			pool.Generation = 123

			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = map[string]string{}
			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels["newkey"] = "newval"
			newPoolRevision := createPoolRevision(pool)

			addPool(pool)
			addCR(oldPoolRevision)
			addCR(newPoolRevision)

			createVMsWithOrdinal(pool, 4, newPoolRevision, oldPoolRevision, vm)

			vmi, err := fakeVirtClient.KubevirtV1().VirtualMachineInstances(pool.Namespace).Patch(context.TODO(), fmt.Sprintf("%s-3", pool.Name), k8stypes.MergePatchType, []byte(`{"status":{"conditions":[{"type":"Ready","status":"False","message":"timeout occurred while waiting for the VMI to become healthy"}]}}`), metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			controller.vmiStore.Update(vmi)

			sanityExecute()

			// Should not see any VMI deletions since one VMI is not ready
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(BeEmpty())
		})

		It("should back off and respect maxUnavailable when updates fail", func() {
			pool, vm := DefaultPool(4)
			pool.Status.Replicas = 4
			pool.Status.ReadyReplicas = 4
			maxUnavailable := intstr.FromString("25%")
			pool.Spec.MaxUnavailable = &maxUnavailable

			oldPoolRevision := createPoolRevision(pool)

			pool.Generation = 123
			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = map[string]string{}
			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels["newkey"] = "newval"
			newPoolRevision := createPoolRevision(pool)

			addPool(pool)
			createVMsWithOrdinal(pool, 4, newPoolRevision, oldPoolRevision, vm)
			addCR(oldPoolRevision)
			addCR(newPoolRevision)

			// Simulate failed VMI deletion
			deleteCount := 0
			fakeVirtClient.Fake.PrependReactor("delete", "virtualmachineinstances", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				deleteCount++
				if deleteCount == 1 {
					// First deletion fails
					return true, obj, fmt.Errorf("simulated deletion failure")
				}
				return true, obj, nil
			})

			sanityExecute()

			vmpool, err := fakeVirtClient.PoolV1alpha1().VirtualMachinePools(pool.Namespace).Get(context.TODO(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmpool.Status.Conditions).To(HaveLen(1))
			Expect(vmpool.Status.Conditions[0].Type).To(Equal(poolv1.VirtualMachinePoolReplicaFailure))
			Expect(vmpool.Status.Conditions[0].Reason).To(Equal(FailedUpdateReason))

			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(HaveLen(1))
			testutils.ExpectEvent(recorder, common.FailedUpdateVirtualMachineReason)
		})
	})
})

func PoolFromVM(name string, vm *v1.VirtualMachine, replicas int32) *poolv1.VirtualMachinePool {
	s, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: vm.ObjectMeta.Labels,
	})
	Expect(err).ToNot(HaveOccurred())
	pool := &poolv1.VirtualMachinePool{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vm.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: poolv1.VirtualMachinePoolSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: vm.ObjectMeta.Labels,
			},
			VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vm.ObjectMeta.Name,
					Labels: vm.ObjectMeta.Labels,
				},
				Spec: vm.Spec,
			},
		},
		Status: poolv1.VirtualMachinePoolStatus{LabelSelector: s.String()},
	}
	return pool
}

func DefaultPool(replicas int32) (*poolv1.VirtualMachinePool, *v1.VirtualMachine) {
	vmi := libvmi.New(
		libvmi.WithNamespace(k8sv1.NamespaceDefault),
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
	)
	vm := libvmi.NewVirtualMachine(vmi)
	vm.Labels = map[string]string{}
	vm.Labels["selector"] = "value"

	pool := PoolFromVM("my-pool", vm, replicas)
	pool.Labels = map[string]string{}
	vm.OwnerReferences = []metav1.OwnerReference{poolOwnerRef(pool)}
	virtcontroller.SetLatestApiVersionAnnotation(vm)
	virtcontroller.SetLatestApiVersionAnnotation(pool)
	return pool, vm.DeepCopy()
}

func markVmAsReady(vm *v1.VirtualMachine) {
	virtcontroller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{Type: v1.VirtualMachineReady, Status: k8sv1.ConditionTrue})
}

func createReadyVMI(vm *v1.VirtualMachine, poolRevision *appsv1.ControllerRevision) *v1.VirtualMachineInstance {
	vmi := libvmi.New(
		libvmi.WithNamespace(vm.Namespace),
		libvmi.WithName(vm.Name),
	)
	vmi.Spec = vm.Spec.Template.Spec
	vmi.Labels = make(map[string]string)
	for k, v := range vm.Spec.Template.ObjectMeta.Labels {
		vmi.Labels[k] = v
	}
	vmi.Labels[v1.VirtualMachinePoolRevisionName] = poolRevision.Name
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               v1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         pointer.P(true),
		BlockOwnerDeletion: pointer.P(true),
	}}
	vmi.Status.Phase = v1.Running
	vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{
		{
			Type:   v1.VirtualMachineInstanceReady,
			Status: k8sv1.ConditionTrue,
		},
	}
	return vmi
}
