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

package watch

import (
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Launcher pod watcher", func() {

	var ctrl *gomock.Controller
	var migrationInterface *kubecli.MockVirtualMachineInstanceMigrationInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var vmInterface *kubecli.MockVirtualMachineInterface
	var podSource *framework.FakeControllerSource
	var podInformer cache.SharedIndexInformer
	var stop chan struct{}
	var controller *LauncherEvictionController
	var recorder *record.FakeRecorder
	var mockQueue *testutils.MockWorkQueue
	var podFeeder *testutils.PodFeeder
	var virtClient *kubecli.MockKubevirtClient
	var kubeClient *fake.Clientset

	syncCaches := func(stop chan struct{}) {
		go podInformer.Run(stop)

		Expect(cache.WaitForCacheSync(stop, podInformer.HasSynced)).To(BeTrue())
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		migrationInterface = kubecli.NewMockVirtualMachineInstanceMigrationInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)

		podInformer, podSource = testutils.NewFakeInformerFor(&k8sv1.Pod{})
		recorder = record.NewFakeRecorder(100)

		controller = NewLauncherEvictionController(virtClient, podInformer, recorder)

		mockQueue = testutils.NewMockWorkQueue(controller.Queue)
		controller.Queue = mockQueue
		podFeeder = testutils.NewPodFeeder(mockQueue, podSource)

		// Set up mock client
		virtClient.EXPECT().VirtualMachineInstanceMigration(k8sv1.NamespaceDefault).Return(migrationInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(k8sv1.NamespaceDefault).Return(vmiInterface).AnyTimes()
		virtClient.EXPECT().VirtualMachine(k8sv1.NamespaceDefault).Return(vmInterface).AnyTimes()
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
		syncCaches(stop)
	})

	AfterEach(func() {
		close(stop)
		// Ensure that we add checks for expected events to every test
		Expect(recorder.Events).To(BeEmpty())
		ctrl.Finish()
	})

	Context("Pod update event", func() {

		It("Should do nothing when the pod is not a was not marked for eviction", func() {
			podFeeder.Modify(&k8sv1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "foo",
					Namespace: k8sv1.NamespaceDefault,
					Annotations: map[string]string{
						v1.DomainAnnotation: "vmi-name",
					},
				},
			})

			controller.Execute()
		})

		It("Should create a migration object when launcher is marked for eviction", func() {
			podFeeder.Modify(&k8sv1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "foo",
					Namespace: k8sv1.NamespaceDefault,
					Annotations: map[string]string{
						v1.DomainAnnotation: "vmi-name",
					},
					Labels: map[string]string{
						v1.AppLabel: "virt-launcher",
					},
				},
				Status: k8sv1.PodStatus{
					Conditions: []k8sv1.PodCondition{
						{
							Type: v1.LauncherMarkedForEviction,
						},
					},
				},
			})

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-1",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}

			migrationInterface.EXPECT().Create(&v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}).MaxTimes(1).Return(&migration, nil)

			liveMigrateStrategy := v1.EvictionStrategyLiveMigrate
			vmiInterface.EXPECT().Get("vmi-name", &v12.GetOptions{}).Return(&v1.VirtualMachineInstance{
				ObjectMeta: v12.ObjectMeta{
					Name: "vmi-name",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					EvictionStrategy: &liveMigrateStrategy,
				},
				Status: v1.VirtualMachineInstanceStatus{
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionTrue,
						},
					},
				},
			}, nil).MaxTimes(1)

			// return an error to indicate that a migration was not yet created
			migrationInterface.EXPECT().
				List(&v12.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.MigrationForEvictedPodLabel, "foo")}).
				Return(nil, errors.NewNotFound(v1.Resource(""), "")).
				MaxTimes(1)

			controller.Execute()
			Expect(<-recorder.Events).To(Equal("Normal MigrationCreatedSuccessfully Created migration "))
		})

		It("Should not create a migration if one already exists", func() {
			podFeeder.Modify(&k8sv1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "foo",
					Namespace: k8sv1.NamespaceDefault,
					Annotations: map[string]string{
						v1.DomainAnnotation: "vmi-name",
					},
					Labels: map[string]string{
						v1.AppLabel: "virt-launcher",
					},
				},
				Status: k8sv1.PodStatus{
					Conditions: []k8sv1.PodCondition{
						{
							Type: v1.LauncherMarkedForEviction,
						},
					},
				},
			})

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-1",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}

			liveMigrateStrategy := v1.EvictionStrategyLiveMigrate
			vmiInterface.EXPECT().Get("vmi-name", &v12.GetOptions{}).Return(&v1.VirtualMachineInstance{
				ObjectMeta: v12.ObjectMeta{
					Name: "vmi-name",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					EvictionStrategy: &liveMigrateStrategy,
				},
				Status: v1.VirtualMachineInstanceStatus{
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionTrue,
						},
					},
				},
			}, nil).MaxTimes(1)

			migrationInterface.EXPECT().List(&v12.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", v1.MigrationForEvictedPodLabel, "foo")}).Return(&v1.VirtualMachineInstanceMigrationList{
				Items: []v1.VirtualMachineInstanceMigration{migration},
			}, nil).MaxTimes(1)

			controller.Execute()
		})

		It("Should record an error when failing to create the migration object", func() {
			podFeeder.Modify(&k8sv1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "foo",
					Namespace: k8sv1.NamespaceDefault,
					Annotations: map[string]string{
						v1.DomainAnnotation: "vmi-name",
					},
					Labels: map[string]string{
						v1.AppLabel: "virt-launcher",
					},
				},
				Status: k8sv1.PodStatus{
					Conditions: []k8sv1.PodCondition{
						{
							Type: v1.LauncherMarkedForEviction,
						},
					},
				},
			})

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-1",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}

			migrationInterface.EXPECT().
				Create(migration).
				MaxTimes(1).
				Return(nil, fmt.Errorf("something went wrong"))

			liveMigrateStrategy := v1.EvictionStrategyLiveMigrate
			vmiInterface.EXPECT().Get("vmi-name", &v12.GetOptions{}).Return(&v1.VirtualMachineInstance{
				ObjectMeta: v12.ObjectMeta{
					Name: "vmi-name",
				},
				Spec: v1.VirtualMachineInstanceSpec{EvictionStrategy: &liveMigrateStrategy},
				Status: v1.VirtualMachineInstanceStatus{
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionTrue,
						},
					},
				},
			}, nil).MaxTimes(1)

			migrationInterface.EXPECT().
				List(&v12.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.MigrationForEvictedPodLabel, "foo")}).
				Return(nil, fmt.Errorf("error")).
				MaxTimes(1)

			controller.Execute()
			Expect(<-recorder.Events).To(Equal("Warning FailureCreatingMigration error"))
		})

		It("Should delete launcehr when VMI is not migratable", func() {
			pod := k8sv1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "foo",
					Namespace: k8sv1.NamespaceDefault,
					Annotations: map[string]string{
						v1.DomainAnnotation: "vmi-name",
					},
					Labels: map[string]string{
						v1.AppLabel: "virt-launcher",
					},
				},
				Status: k8sv1.PodStatus{
					Conditions: []k8sv1.PodCondition{
						{
							Type: v1.LauncherMarkedForEviction,
						},
					},
				},
			}
			podFeeder.Modify(&pod)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-1",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}

			migrationInterface.EXPECT().Create(&v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}).MaxTimes(1).Return(&migration, nil)

			liveMigrateStrategy := v1.EvictionStrategyLiveMigrate
			vmiInterface.EXPECT().Get("vmi-name", &v12.GetOptions{}).Return(&v1.VirtualMachineInstance{
				ObjectMeta: v12.ObjectMeta{
					Name:      "vmi-name",
					Namespace: k8sv1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineInstanceSpec{EvictionStrategy: &liveMigrateStrategy},
				Status: v1.VirtualMachineInstanceStatus{
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionFalse,
						},
					},
				},
			}, nil).MaxTimes(1)

			kubeClient.Fake.PrependReactor("delete", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				delete, ok := action.(testing.DeleteAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(delete.GetNamespace()))
				Expect(pod.Name).To(Equal(delete.GetName()))
				return true, nil, nil
			})

			vmInterface.EXPECT().Stop("vmi-name").Return(errors.NewNotFound(v1.Resource(""), ""))
			controller.Execute()

			Expect(kubeClient.Fake.Actions()).Should(HaveLen(1))
		})

		It("Should stop the VM when possible", func() {
			pod := k8sv1.Pod{
				ObjectMeta: v12.ObjectMeta{
					Name:      "foo",
					Namespace: k8sv1.NamespaceDefault,
					Annotations: map[string]string{
						v1.DomainAnnotation: "vmi-name",
					},
					Labels: map[string]string{
						v1.AppLabel: "virt-launcher",
					},
				},
				Status: k8sv1.PodStatus{
					Conditions: []k8sv1.PodCondition{
						{
							Type: v1.LauncherMarkedForEviction,
						},
					},
				},
			}
			podFeeder.Modify(&pod)

			migration := v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-1",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}

			migrationInterface.EXPECT().Create(&v1.VirtualMachineInstanceMigration{
				ObjectMeta: v12.ObjectMeta{
					GenerateName: "kubevirt-eviction-",
					Labels: map[string]string{
						v1.MigrationForEvictedPodLabel: "foo",
					},
				},
				Spec: v1.VirtualMachineInstanceMigrationSpec{
					VMIName: "vmi-name",
				},
			}).MaxTimes(1).Return(&migration, nil)

			liveMigrateStrategy := v1.EvictionStrategyLiveMigrate
			vmiInterface.EXPECT().Get("vmi-name", &v12.GetOptions{}).Return(&v1.VirtualMachineInstance{
				ObjectMeta: v12.ObjectMeta{
					Name:      "vmi-name",
					Namespace: k8sv1.NamespaceDefault,
				},
				Spec: v1.VirtualMachineInstanceSpec{EvictionStrategy: &liveMigrateStrategy},
				Status: v1.VirtualMachineInstanceStatus{
					Conditions: []v1.VirtualMachineInstanceCondition{
						{
							Type:   v1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionFalse,
						},
					},
				},
			}, nil).MaxTimes(1)

			vmInterface.EXPECT().Stop("vmi-name").Return(nil)

			controller.Execute()

			Expect(kubeClient.Fake.Actions()).Should(HaveLen(0))
		})

	})

})
