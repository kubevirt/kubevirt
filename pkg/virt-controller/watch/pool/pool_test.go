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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package pool

import (
	"encoding/json"
	"fmt"
	"maps"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	"kubevirt.io/client-go/testing"

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
			key, err := virtcontroller.KeyFunc(cr)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)
		}

		addVM := func(vm *v1.VirtualMachine) {
			controller.vmIndexer.Add(vm)
			key, err := virtcontroller.KeyFunc(vm)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)
		}

		addVMI := func(vm *v1.VirtualMachineInstance) {
			controller.vmiStore.Add(vm)
			key, err := virtcontroller.KeyFunc(vm)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)
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

			fakeVirtClient.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			k8sClient = k8sfake.NewSimpleClientset()
			k8sClient.Fake.PrependReactor("*", "*", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()
		})

		addPool := func(pool *poolv1.VirtualMachinePool) {
			controller.poolIndexer.Add(pool)
			key, err := virtcontroller.KeyFunc(pool)
			Expect(err).To(Not(HaveOccurred()))
			mockQueue.Add(key)
		}

		createPoolRevision := func(pool *poolv1.VirtualMachinePool) *appsv1.ControllerRevision {
			bytes, err := json.Marshal(&pool.Spec)
			Expect(err).ToNot(HaveOccurred())

			return &appsv1.ControllerRevision{
				ObjectMeta: metav1.ObjectMeta{
					Name:            getRevisionName(pool),
					Namespace:       pool.Namespace,
					OwnerReferences: []metav1.OwnerReference{poolOwnerRef(pool)},
				},
				Data:     runtime.RawExtension{Raw: bytes},
				Revision: pool.Generation,
			}
		}

		expectControllerRevisionDeletion := func(poolRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				deleted, ok := action.(k8stesting.DeleteAction)
				Expect(ok).To(BeTrue())

				delName := deleted.GetName()
				Expect(delName).To(Equal(poolRevision.Name))

				return true, nil, nil
			})
		}

		expectControllerRevisionCreation := func(poolRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(k8stesting.CreateAction)
				Expect(ok).To(BeTrue())

				createObj := created.GetObject().(*appsv1.ControllerRevision)
				Expect(createObj.Name).To(Equal(poolRevision.Name))

				return true, created.GetObject(), nil
			})
		}

		expectVMCreationWithValidation := func(nameMatcher types.GomegaMatcher, validateFn func(machine *v1.VirtualMachine)) {
			fakeVirtClient.Fake.PrependReactor("create", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(k8stesting.CreateAction)
				Expect(ok).To(BeTrue())
				createObj := created.GetObject().(*v1.VirtualMachine)
				Expect(createObj.Name).To(nameMatcher)
				Expect(createObj.GenerateName).To(Equal(""))
				validateFn(createObj)
				return true, created.GetObject(), nil
			})
		}

		expectVMCreation := func(nameMatcher types.GomegaMatcher) {
			expectVMCreationWithValidation(nameMatcher, func(_ *v1.VirtualMachine) {})
		}

		expectVMUpdate := func(revisionName string) {
			fakeVirtClient.Fake.PrependReactor("update", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := created.GetObject().(*v1.VirtualMachine)
				Expect(updateObj.Labels).To(HaveKeyWithValue(v1.VirtualMachinePoolRevisionName, revisionName))
				return true, created.GetObject(), nil
			})
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
			expectControllerRevisionCreation(poolRevision)
			expectVMCreation(HavePrefix(fmt.Sprintf("%s-", pool.Name)))

			sanityExecute()
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
			newPoolRevision := createPoolRevision(pool)

			vm.Name = fmt.Sprintf("%s-0", pool.Name)

			vm = injectPoolRevisionLabelsIntoVM(vm, poolRevision.Name)

			vmi := api.NewMinimalVMI(vm.Name)
			vmi.Spec = vm.Spec.Template.Spec
			vmi.Name = vm.Name
			vmi.Namespace = vm.Namespace
			vmi.Labels = vm.Spec.Template.ObjectMeta.Labels
			vmi.OwnerReferences = []metav1.OwnerReference{{
				APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Kind:               v1.VirtualMachineGroupVersionKind.Kind,
				Name:               vm.ObjectMeta.Name,
				UID:                vm.ObjectMeta.UID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			}}

			markVmAsReady(vm)
			watchtesting.MarkAsReady(vmi)
			addPool(pool)
			addVM(vm)
			addCR(poolRevision)
			// not expecting vmi to cause enqueue of pool because VMI and VM use the same pool revision
			controller.vmiStore.Add(vmi)

			expectControllerRevisionCreation(newPoolRevision)
			expectVMUpdate(newPoolRevision.Name)

			sanityExecute()

			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(BeEmpty())
		})

		It("should delete controller revisions when pool is being deleted", func() {
			pool, _ := DefaultPool(1)
			pool.Status.Replicas = 0
			pool.DeletionTimestamp = pointer.P(metav1.Now())

			poolRevision := createPoolRevision(pool)

			addPool(pool)
			addCR(poolRevision)

			expectControllerRevisionDeletion(poolRevision)
			sanityExecute()
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

			vm = injectPoolRevisionLabelsIntoVM(vm, newPoolRevision.Name)
			vm.Name = fmt.Sprintf("%s-0", pool.Name)
			markVmAsReady(vm)

			vmi := api.NewMinimalVMI(vm.Name)
			vmi.Spec = vm.Spec.Template.Spec
			vmi.Name = vm.Name
			vmi.Namespace = vm.Namespace
			vmi.Labels = maps.Clone(vm.Spec.Template.ObjectMeta.Labels)
			vmi.OwnerReferences = []metav1.OwnerReference{{
				APIVersion:         v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Kind:               v1.VirtualMachineGroupVersionKind.Kind,
				Name:               vm.ObjectMeta.Name,
				UID:                vm.ObjectMeta.UID,
				Controller:         pointer.P(true),
				BlockOwnerDeletion: pointer.P(true),
			}}

			vmi.Labels[v1.VirtualMachinePoolRevisionName] = oldPoolRevision.Name

			addPool(pool)
			addVM(vm)
			addVMI(vmi)
			addCR(oldPoolRevision)
			addCR(newPoolRevision)

			expectControllerRevisionCreation(newPoolRevision)
			fakeVirtClient.Fake.PrependReactor("delete", "virtualmachineinstances", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, nil
			})

			sanityExecute()

			testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachineinstances")).To(HaveLen(1))
		})

		It("should do nothing", func() {
			pool, vm := DefaultPool(1)
			vm.Name = fmt.Sprintf("%s-0", pool.Name)

			poolRevision := createPoolRevision(pool)
			vm = injectPoolRevisionLabelsIntoVM(vm, poolRevision.Name)
			markVmAsReady(vm)

			pool.Status.Replicas = 1
			pool.Status.ReadyReplicas = 1
			addPool(pool)
			addVM(vm)
			addCR(poolRevision)

			sanityExecute()
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
			addVM(vm)
			addCR(poolRevision)
			addCR(oldPoolRevision)

			expectControllerRevisionDeletion(oldPoolRevision)
			sanityExecute()
		})

		It("should not create missing VMs when it is paused and add paused condition", func() {
			pool, _ := DefaultPool(3)
			pool.Spec.Paused = true

			addPool(pool)

			expectedPool := pool.DeepCopy()
			expectedPool.Status.Replicas = 0

			// Expect pool to be updated with paused condition
			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(0)))
				Expect(updateObj.Status.Conditions).To(HaveLen(1))
				Expect(updateObj.Status.Conditions[0].Type).To(Equal(poolv1.VirtualMachinePoolReplicaPaused))
				Expect(updateObj.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionTrue))
				return true, update.GetObject(), nil
			})

			sanityExecute()

			testutils.ExpectEvent(recorder, SuccessfulPausedPoolReason)
			// Expect pool to be updated with paused condition
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(BeEmpty())
		})

		It("should set failed condition when reconcile error occurs", func() {
			pool, _ := DefaultPool(3)

			addPool(pool)

			expectedPool := pool.DeepCopy()
			expectedPool.Status.Replicas = 0

			fakeVirtClient.Fake.PrependReactor("create", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, &v1.VirtualMachine{}, fmt.Errorf("error")
			})

			// Expect pool to be updated with paused condition
			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(0)))
				Expect(updateObj.Status.Conditions).To(HaveLen(1))
				Expect(updateObj.Status.Conditions[0].Type).To(Equal(poolv1.VirtualMachinePoolReplicaFailure))
				Expect(updateObj.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionTrue))
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			sanityExecute()

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

			expectedPool := pool.DeepCopy()
			expectedPool.Status.Replicas = 0

			// Expect pool to be updated with paused condition
			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Conditions).To(BeEmpty())
				return true, update.GetObject(), nil
			})

			sanityExecute()
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

			expectedPool := pool.DeepCopy()
			expectedPool.Status.Replicas = 0

			expectVMCreation(HavePrefix(fmt.Sprintf("%s-", pool.Name)))

			// Expect pool to be updated with paused condition
			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Conditions).To(BeEmpty())
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			sanityExecute()

			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulResumePoolReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		})

		It("should create missing VMs in batches of a maximum of burst replicas VMs at once", func() {
			pool, _ := DefaultPool(15)

			addPool(pool)

			expectVMCreation(HavePrefix(fmt.Sprintf("%s-", pool.Name)))

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

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

			// Add 15 VMs to the informer cache
			for x := 0; x < 15; x++ {
				newVM := vm.DeepCopy()
				newVM.Name = fmt.Sprintf("%s-%d", pool.Name, x)
				addVM(newVM)
			}

			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(15)))
				return true, update.GetObject(), nil
			})

			fakeVirtClient.Fake.PrependReactor("delete", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, nil
			})

			sanityExecute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, common.SuccessfulDeleteVirtualMachineReason)
			}
			// Check if only 10 are deleted
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "delete", "virtualmachines")).To(HaveLen(10))
		})

		It("should not delete vms which are already marked deleted", func() {
			pool, vm := DefaultPool(0)

			addPool(pool)

			// Add 5 VMs to the informer cache
			for x := 0; x < 5; x++ {
				newVM := vm.DeepCopy()
				newVM.Name = fmt.Sprintf("%s-%d", pool.Name, x)
				addVM(newVM)
			}
			for x := 5; x < 10; x++ {
				newVM := vm.DeepCopy()
				newVM.Name = fmt.Sprintf("%s-%d", pool.Name, x)
				newVM.DeletionTimestamp = pointer.P(metav1.Now())
				addVM(newVM)
			}

			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(10)))
				return true, update.GetObject(), nil
			})
			fakeVirtClient.Fake.PrependReactor("delete", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, nil
			})
			sanityExecute()

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

			addPool(pool)

			expectVMCreation(And(HavePrefix(fmt.Sprintf("%s-", pool.Name)), Not(Equal(nonMatchingVM.Name))))

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			sanityExecute()
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		})

		It("should ignore orphaned VMs even when selector matches", func() {
			pool, vm := DefaultPool(3)
			vm.OwnerReferences = []metav1.OwnerReference{}

			// Orphaned VM won't cause key to enqueue
			controller.vmIndexer.Add(vm)
			addPool(pool)

			expectVMCreation(HavePrefix(fmt.Sprintf("%s-", pool.Name)))

			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(0)))
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			sanityExecute()
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
		})

		It("should detect a VM is detached, then release and replace it", func() {
			pool, vm := DefaultPool(3)
			vm.Labels = map[string]string{}
			vm.Name = fmt.Sprintf("%s-1", pool.Name)

			addPool(pool)
			addVM(vm)

			fakeVirtClient.Fake.PrependReactor("patch", "virtualmachines", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				patchAction, ok := action.(k8stesting.PatchAction)
				Expect(ok).To(BeTrue())
				Expect(patchAction.GetName()).To(Equal(vm.Name))
				return true, vm, nil
			})
			expectVMCreation(HavePrefix(fmt.Sprintf("%s-", pool.Name)))

			fakeVirtClient.Fake.PrependReactor("update", "virtualmachinepools", func(action k8stesting.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(k8stesting.UpdateAction)
				Expect(ok).To(BeTrue())
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			sanityExecute()
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
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
			expectControllerRevisionCreation(poolRevision)
			expectVMCreationWithValidation(
				HavePrefix(fmt.Sprintf("%s-", pool.Name)),
				func(vm *v1.VirtualMachine) {
					defer GinkgoRecover()
					Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cmName))
					Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(secretName))
					if appendIndex != nil && *appendIndex {
						Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).To(MatchRegexp(fmt.Sprintf("%s-\\d", cmName)))
						Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName).To(MatchRegexp(fmt.Sprintf("%s-\\d", secretName)))
					} else {
						Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).To(Equal(cmName))
						Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName).To(Equal(secretName))
					}
				},
			)

			sanityExecute()
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, common.SuccessfulCreateVirtualMachineReason)
			Expect(testing.FilterActions(&fakeVirtClient.Fake, "create", "virtualmachines")).To(HaveLen(3))
		},
			Entry("do not append index by default", nil),
			Entry("do not append index if set to false", pointer.P(false)),
			Entry("append index if set to true", pointer.P(true)),
		)
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
	vmi := api.NewMinimalVMI("testvmi")
	vmi.Labels = map[string]string{}
	vm := watchtesting.VirtualMachineFromVMI(vmi.Name, vmi, true)
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
