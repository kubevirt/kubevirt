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

package watch

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/api"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	virtcontroller "kubevirt.io/kubevirt/pkg/controller"
	testutils "kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Pool", func() {

	DescribeTable("Calculate new VM names on scale out", func(existing []string, expected []string, count int) {

		namespace := "test"
		baseName := "my-pool"
		vmInformer, _ := testutils.NewFakeInformerFor(&virtv1.VirtualMachine{})

		for _, name := range existing {
			vm, _ := DefaultVirtualMachine(true)
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

		var ctrl *gomock.Controller

		var crInformer cache.SharedIndexInformer
		var crSource *framework.FakeControllerSource

		var vmInterface *kubecli.MockVirtualMachineInterface
		var vmiInterface *kubecli.MockVirtualMachineInstanceInterface

		var vmiSource *framework.FakeControllerSource
		var vmSource *framework.FakeControllerSource
		var poolSource *framework.FakeControllerSource
		var vmiInformer cache.SharedIndexInformer
		var vmInformer cache.SharedIndexInformer
		var poolInformer cache.SharedIndexInformer
		var stop chan struct{}
		var controller *PoolController
		var recorder *record.FakeRecorder
		var mockQueue *testutils.MockWorkQueue
		var client *kubevirtfake.Clientset
		var k8sClient *k8sfake.Clientset

		syncCaches := func(stop chan struct{}) {
			go vmiInformer.Run(stop)
			go vmInformer.Run(stop)
			go poolInformer.Run(stop)
			go crInformer.Run(stop)
			Expect(cache.WaitForCacheSync(stop, vmiInformer.HasSynced, vmInformer.HasSynced, poolInformer.HasSynced, crInformer.HasSynced)).To(BeTrue())
		}

		addCR := func(cr *appsv1.ControllerRevision) {
			mockQueue.ExpectAdds(1)
			crSource.Add(cr)
			mockQueue.Wait()
		}

		addVM := func(vm *virtv1.VirtualMachine) {
			mockQueue.ExpectAdds(1)
			vmSource.Add(vm)
			mockQueue.Wait()
		}

		addVMI := func(vm *virtv1.VirtualMachineInstance, expectQueue bool) {
			if expectQueue {
				mockQueue.ExpectAdds(1)
			}
			vmiSource.Add(vm)
			if expectQueue {
				mockQueue.Wait()
			} else {
				time.Sleep(1 * time.Second)
			}
		}

		BeforeEach(func() {
			stop = make(chan struct{})
			ctrl = gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)

			vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
			vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
			vmiInformer, vmiSource = testutils.NewFakeInformerFor(&v1.VirtualMachineInstance{})
			vmInformer, vmSource = testutils.NewFakeInformerFor(&v1.VirtualMachine{})
			poolInformer, poolSource = testutils.NewFakeInformerFor(&poolv1.VirtualMachinePool{})
			recorder = record.NewFakeRecorder(100)
			recorder.IncludeObject = true

			crInformer, crSource = testutils.NewFakeInformerWithIndexersFor(&appsv1.ControllerRevision{}, cache.Indexers{
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

			controller, _ = NewPoolController(virtClient,
				vmiInformer,
				vmInformer,
				poolInformer,
				crInformer,
				recorder,
				uint(10))
			// Wrap our workqueue to have a way to detect when we are done processing updates
			mockQueue = testutils.NewMockWorkQueue(controller.queue)
			controller.queue = mockQueue

			client = kubevirtfake.NewSimpleClientset()

			// Set up mock client
			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(vmiInterface).AnyTimes()
			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(vmInterface).AnyTimes()

			virtClient.EXPECT().VirtualMachinePool(testNamespace).Return(client.PoolV1alpha1().VirtualMachinePools(testNamespace)).AnyTimes()

			client.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})

			k8sClient = k8sfake.NewSimpleClientset()
			k8sClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				Expect(action).To(BeNil())
				return true, nil, nil
			})
			virtClient.EXPECT().AppsV1().Return(k8sClient.AppsV1()).AnyTimes()

			syncCaches(stop)
		})

		addPool := func(pool *poolv1.VirtualMachinePool) {
			mockQueue.ExpectAdds(1)
			poolSource.Add(pool)
			mockQueue.Wait()
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
			k8sClient.Fake.PrependReactor("delete", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				deleted, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())

				delName := deleted.GetName()
				Expect(delName).To(Equal(poolRevision.Name))

				return true, nil, nil
			})
		}

		expectControllerRevisionCreation := func(poolRevision *appsv1.ControllerRevision) {
			k8sClient.Fake.PrependReactor("create", "controllerrevisions", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				created, ok := action.(testing.CreateAction)
				Expect(ok).To(BeTrue())

				createObj := created.GetObject().(*appsv1.ControllerRevision)
				Expect(createObj.Name).To(Equal(poolRevision.Name))

				return true, created.GetObject(), nil
			})
		}

		It("should create missing VMs", func() {
			pool, vm := DefaultPool(3)

			addPool(pool)

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(HavePrefix(fmt.Sprintf("%s-", pool.Name)))
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal(""))
			}).Return(vm, nil)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
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
				APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
				Name:               vm.ObjectMeta.Name,
				UID:                vm.ObjectMeta.UID,
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}}

			markVmAsReady(vm)
			markAsReady(vmi)
			addPool(pool)
			addVM(vm)
			addCR(poolRevision)
			// not expecting vmi to cause enqueue of pool because VMI and VM use the same pool revision
			addVMI(vmi, false)

			expectControllerRevisionCreation(newPoolRevision)

			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Times(0)
			vmInterface.EXPECT().Update(context.Background(), gomock.Any()).MaxTimes(1).Do(func(ctx context.Context, arg interface{}) {
				newVM := arg.(*v1.VirtualMachine)
				revisionName := newVM.Labels[virtv1.VirtualMachinePoolRevisionName]
				Expect(revisionName).To(Equal(newPoolRevision.Name))
			}).Return(vm, nil)

			controller.Execute()
		})

		It("should delete controller revisions when pool is being deleted", func() {
			pool, _ := DefaultPool(1)
			pool.Status.Replicas = 0
			pool.DeletionTimestamp = now()

			poolRevision := createPoolRevision(pool)

			addPool(pool)
			addCR(poolRevision)

			expectControllerRevisionDeletion(poolRevision)
			controller.Execute()
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
			vmi.Labels = mapCopy(vm.Spec.Template.ObjectMeta.Labels)
			vmi.OwnerReferences = []metav1.OwnerReference{{
				APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
				Name:               vm.ObjectMeta.Name,
				UID:                vm.ObjectMeta.UID,
				Controller:         &t,
				BlockOwnerDeletion: &t,
			}}

			vmi.Labels[virtv1.VirtualMachinePoolRevisionName] = oldPoolRevision.Name

			addPool(pool)
			addVM(vm)
			addVMI(vmi, true)
			addCR(oldPoolRevision)
			addCR(newPoolRevision)

			expectControllerRevisionCreation(newPoolRevision)

			vmiInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Times(1).Return(nil)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
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

			controller.Execute()
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
			controller.Execute()
		})

		It("should not create missing VMs when it is paused and add paused condition", func() {
			pool, _ := DefaultPool(3)
			pool.Spec.Paused = true

			addPool(pool)

			expectedPool := pool.DeepCopy()
			expectedPool.Status.Replicas = 0

			// No invocations expected
			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(0)

			// Expect pool to be updated with paused condition
			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(0)))
				Expect(updateObj.Status.Conditions).To(HaveLen(1))
				Expect(updateObj.Status.Conditions[0].Type).To(Equal(poolv1.VirtualMachinePoolReplicaPaused))
				Expect(updateObj.Status.Conditions[0].Status).To(Equal(k8sv1.ConditionTrue))
				return true, update.GetObject(), nil
			})

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulPausedPoolReason)
		})

		It("should set failed condition when reconcile error occurs", func() {
			pool, _ := DefaultPool(3)

			addPool(pool)

			expectedPool := pool.DeepCopy()
			expectedPool.Status.Replicas = 0

			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Return(nil, fmt.Errorf("error"))

			// Expect pool to be updated with paused condition
			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
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

			controller.Execute()

			testutils.ExpectEvent(recorder, FailedCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, FailedScaleOutReason)
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
			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Conditions).To(BeEmpty())
				return true, update.GetObject(), nil
			})

			controller.Execute()
		})

		It("should create missing VMs when pool is resumed", func() {
			pool, vm := DefaultPool(3)

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

			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(HavePrefix(fmt.Sprintf("%s-", pool.Name)))
			}).Return(vm, nil)

			// Expect pool to be updated with paused condition
			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Conditions).To(BeEmpty())
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			controller.Execute()

			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulResumePoolReason)
		})

		It("should create missing VMs in batches of a maximum of burst replicas VMs at once", func() {
			pool, vm := DefaultPool(15)

			addPool(pool)

			// Check if only 10 are created
			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(10).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(HavePrefix(fmt.Sprintf("%s-", pool.Name)))
			}).Return(vm, nil)

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			}
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

			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(15)))
				return true, update.GetObject(), nil
			})

			// Check if only 10 are deleted
			vmInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Times(10).Return(nil)

			controller.Execute()

			for x := 0; x < 10; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}
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
				newVM.DeletionTimestamp = now()
				addVM(newVM)
			}

			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(10)))
				return true, update.GetObject(), nil
			})

			// Check if only 5 are deleted
			vmInterface.EXPECT().Delete(context.Background(), gomock.Any(), gomock.Any()).Times(5).Return(nil)

			controller.Execute()

			for x := 0; x < 5; x++ {
				testutils.ExpectEvent(recorder, SuccessfulDeleteVirtualMachineReason)
			}
		})

		It("should ignore and skip the name for non-matching VMs", func() {

			pool, vm := DefaultPool(3)

			nonMatchingVM := vm.DeepCopy()
			nonMatchingVM.Name = fmt.Sprintf("%s-1", pool.Name)
			nonMatchingVM.Labels = map[string]string{"madeup": "value"}
			nonMatchingVM.OwnerReferences = []metav1.OwnerReference{}
			vmSource.Add(nonMatchingVM)

			// allow for cache to catch up. This non matching vm won't cause key to be queued
			time.Sleep(1 * time.Second)

			addPool(pool)

			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(HavePrefix(fmt.Sprintf("%s-", pool.Name)))
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).ToNot(Equal(nonMatchingVM.Name))
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal(""))
			}).Return(vm, nil)

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
		})

		It("should ignore orphaned VMs even when selector matches", func() {
			pool, vm := DefaultPool(3)
			vm.OwnerReferences = []metav1.OwnerReference{}

			// Orphaned VM won't cause key to enqueue
			vmSource.Add(vm)
			time.Sleep(1)

			addPool(pool)

			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Do(func(ctx context.Context, arg interface{}) {
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.Name).To(HavePrefix(fmt.Sprintf("%s-", pool.Name)))
				Expect(arg.(*v1.VirtualMachine).ObjectMeta.GenerateName).To(Equal(""))
			}).Return(vm, nil)

			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				updateObj := update.GetObject().(*poolv1.VirtualMachinePool)
				Expect(updateObj.Status.Replicas).To(Equal(int32(0)))
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

		})

		It("should detect a VM is detached, then release and replace it", func() {
			pool, vm := DefaultPool(3)
			vm.Labels = map[string]string{}
			vm.Name = fmt.Sprintf("%s-1", pool.Name)

			addPool(pool)
			addVM(vm)

			vmInterface.EXPECT().Patch(context.Background(), vm.ObjectMeta.Name, gomock.Any(), gomock.Any(), gomock.Any())
			vmInterface.EXPECT().Create(context.Background(), gomock.Any()).Times(3).Return(vm, nil)

			client.Fake.PrependReactor("update", "virtualmachinepools", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				update, ok := action.(testing.UpdateAction)
				Expect(ok).To(BeTrue())
				return true, update.GetObject(), nil
			})

			poolRevision := createPoolRevision(pool)
			expectControllerRevisionCreation(poolRevision)

			controller.Execute()
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)
			testutils.ExpectEvent(recorder, SuccessfulCreateVirtualMachineReason)

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
	vmi := api.NewMinimalVMI("testvmi")
	vmi.Labels = map[string]string{}
	vm := VirtualMachineFromVMI(vmi.Name, vmi, true)
	vm.Labels = map[string]string{}
	vm.Labels["selector"] = "value"

	pool := PoolFromVM("my-pool", vm, replicas)
	pool.Labels = map[string]string{}
	vm.OwnerReferences = []metav1.OwnerReference{poolOwnerRef(pool)}
	virtcontroller.SetLatestApiVersionAnnotation(vm)
	virtcontroller.SetLatestApiVersionAnnotation(pool)
	return pool, vm.DeepCopy()
}
