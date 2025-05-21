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
 * Copyright The KubeVirt Authors.
 */

//nolint:dupl
package revision_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	fakeclientset "kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
)

const (
	nonExistingResourceName           = "non-existing-resource"
	resourceUID             types.UID = "9160e5de-2540-476a-86d9-af0081aee68a"
	resourceGeneration      int64     = 1
)

type handler interface {
	Store(*virtv1.VirtualMachine) error
}

var _ = Describe("Instancetype and Preferences revision handler", func() {
	var (
		storeHandler handler
		vm           *virtv1.VirtualMachine

		virtClient *kubecli.MockKubevirtClient

		instancetypeInformerStore        cache.Store
		clusterInstancetypeInformerStore cache.Store
		preferenceInformerStore          cache.Store
		clusterPreferenceInformerStore   cache.Store
	)

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().AppsV1().Return(k8sfake.NewSimpleClientset().AppsV1()).AnyTimes()

		virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
			fakeclientset.NewSimpleClientset().KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(
			fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterInstancetypes()).AnyTimes()

		virtClient.EXPECT().VirtualMachineInstancetype(metav1.NamespaceDefault).Return(
			fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineInstancetypes(metav1.NamespaceDefault)).AnyTimes()

		virtClient.EXPECT().VirtualMachineClusterPreference().Return(
			fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachineClusterPreferences()).AnyTimes()

		virtClient.EXPECT().VirtualMachinePreference(metav1.NamespaceDefault).Return(
			fakeclientset.NewSimpleClientset().InstancetypeV1beta1().VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()

		instancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineInstancetype{})
		instancetypeInformerStore = instancetypeInformer.GetStore()

		clusterInstancetypeInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterInstancetype{})
		clusterInstancetypeInformerStore = clusterInstancetypeInformer.GetStore()

		preferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachinePreference{})
		preferenceInformerStore = preferenceInformer.GetStore()

		clusterPreferenceInformer, _ := testutils.NewFakeInformerFor(&instancetypev1beta1.VirtualMachineClusterPreference{})
		clusterPreferenceInformerStore = clusterPreferenceInformer.GetStore()

		storeHandler = revision.New(
			instancetypeInformerStore,
			clusterInstancetypeInformerStore,
			preferenceInformerStore,
			clusterInstancetypeInformerStore,
			virtClient)

		vm = kubecli.NewMinimalVM("testvm")
		vm.Spec.Template = &virtv1.VirtualMachineInstanceTemplateSpec{
			Spec: virtv1.VirtualMachineInstanceSpec{
				Domain: virtv1.DomainSpec{},
			},
		}
		vm.Namespace = k8sv1.NamespaceDefault

		_, err := virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	Context("store instancetype", func() {
		It("store returns error when instancetypeMatcher kind is invalid", func() {
			vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
				Kind: "foobar",
			}
			Expect(storeHandler.Store(vm)).To(MatchError(ContainSubstring("got unexpected kind in InstancetypeMatcher")))
		})

		It("store returns nil when no instancetypeMatcher is specified", func() {
			vm.Spec.Instancetype = nil
			Expect(storeHandler.Store(vm)).To(Succeed())
		})

		Context("using global ClusterInstancetype", func() {
			var clusterInstancetype *instancetypev1beta1.VirtualMachineClusterInstancetype

			BeforeEach(func() {
				clusterInstancetype = &instancetypev1beta1.VirtualMachineClusterInstancetype{
					TypeMeta: metav1.TypeMeta{
						Kind:       apiinstancetype.ClusterSingularResourceName,
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-cluster-instancetype",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}

				_, err := virtClient.VirtualMachineClusterInstancetype().Create(
					context.Background(), clusterInstancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = clusterInstancetypeInformerStore.Add(clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name: clusterInstancetype.Name,
					Kind: apiinstancetype.ClusterSingularResourceName,
				}

				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("store VirtualMachineClusterInstancetype ControllerRevision", func() {
				Expect(storeHandler.Store(vm)).To(Succeed())

				clusterInstancetypeControllerRevision, err := revision.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				Expect(vm.Spec.Instancetype.RevisionName).To(BeEmpty())
				Expect(vm.Status.InstancetypeRef.Name).To(Equal(clusterInstancetype.Name))
				Expect(vm.Status.InstancetypeRef.Kind).To(Equal(clusterInstancetype.Kind))
				Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(clusterInstancetypeControllerRevision.Name))

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedVM.Spec.Instancetype.RevisionName).To(BeEmpty())
				Expect(updatedVM.Status.InstancetypeRef.Name).To(Equal(clusterInstancetype.Name))
				Expect(updatedVM.Status.InstancetypeRef.Kind).To(Equal(clusterInstancetype.Kind))
				Expect(updatedVM.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(clusterInstancetypeControllerRevision.Name))

				createdCR, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(createdCR).To(Equal(clusterInstancetypeControllerRevision))
			})

			It("store succeeds when RevisionName already populated", func() {
				clusterInstancetypeControllerRevision, err := revision.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), clusterInstancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name:         clusterInstancetype.Name,
					Kind:         clusterInstancetype.Kind,
					RevisionName: clusterInstancetypeControllerRevision.Name,
				}

				vm, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())

				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(clusterInstancetypeControllerRevision.Name))
				Expect(vm.Status.InstancetypeRef.Name).To(Equal(clusterInstancetype.Name))
				Expect(vm.Status.InstancetypeRef.Kind).To(Equal(clusterInstancetype.Kind))
				Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(clusterInstancetypeControllerRevision.Name))

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedVM.Spec.Instancetype.RevisionName).To(Equal(clusterInstancetypeControllerRevision.Name))
				Expect(updatedVM.Status.InstancetypeRef.Name).To(Equal(clusterInstancetype.Name))
				Expect(updatedVM.Status.InstancetypeRef.Kind).To(Equal(clusterInstancetype.Kind))
				Expect(updatedVM.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(clusterInstancetypeControllerRevision.Name))
			})

			It("store fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = nonExistingResourceName
				Expect(storeHandler.Store(vm)).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {
				instancetypeControllerRevision, err := revision.CreateControllerRevision(vm, clusterInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())

				Expect(vm.Spec.Instancetype.RevisionName).To(BeEmpty())
				Expect(vm.Status.InstancetypeRef.Name).To(Equal(clusterInstancetype.Name))
				Expect(vm.Status.InstancetypeRef.Kind).To(Equal(clusterInstancetype.Kind))
				Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(instancetypeControllerRevision.Name))

				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(updatedVM.Spec.Instancetype.RevisionName).To(BeEmpty())
				Expect(updatedVM.Status.InstancetypeRef.Name).To(Equal(clusterInstancetype.Name))
				Expect(updatedVM.Status.InstancetypeRef.Kind).To(Equal(clusterInstancetype.Kind))
				Expect(updatedVM.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedInstancetype := clusterInstancetype.DeepCopy()
				unexpectedInstancetype.Spec.CPU.Guest = 15

				instancetypeControllerRevision, err := revision.CreateControllerRevision(vm, unexpectedInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})

			It("store ControllerRevision fails if instancetype conflicts with vm", func() {
				vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{
					Cores: 1,
				}
				Expect(storeHandler.Store(vm)).To(MatchError(
					conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu", "cores")}))
			})
			It("store InferFromVolumeFailurePolicy when missing from InstancetypeRef", func() {
				vm.Spec.Instancetype.InferFromVolumeFailurePolicy = pointer.P(virtv1.IgnoreInferFromVolumeFailure)
				vm.Status.InstancetypeRef = nil
				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.InstancetypeRef.InferFromVolumeFailurePolicy).To(HaveValue(Equal(virtv1.IgnoreInferFromVolumeFailure)))
			})

			It("store InferFromVolumeFailurePolicy when different to PreferenceRef", func() {
				vm.Spec.Instancetype.InferFromVolumeFailurePolicy = pointer.P(virtv1.IgnoreInferFromVolumeFailure)
				vm.Status.InstancetypeRef = &virtv1.InstancetypeStatusRef{
					InferFromVolumeFailurePolicy: pointer.P(virtv1.RejectInferFromVolumeFailure),
				}
				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.InstancetypeRef.InferFromVolumeFailurePolicy).To(HaveValue(Equal(virtv1.IgnoreInferFromVolumeFailure)))
			})
		})

		Context("using namespaced Instancetype", func() {
			var fakeInstancetype *instancetypev1beta1.VirtualMachineInstancetype

			BeforeEach(func() {
				fakeInstancetype = &instancetypev1beta1.VirtualMachineInstancetype{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineInstancetype",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-instancetype",
						Namespace:  vm.Namespace,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
						CPU: instancetypev1beta1.CPUInstancetype{
							Guest: uint32(2),
						},
						Memory: instancetypev1beta1.MemoryInstancetype{
							Guest: resource.MustParse("128Mi"),
						},
					},
				}

				_, err := virtClient.VirtualMachineInstancetype(vm.Namespace).Create(
					context.Background(), fakeInstancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = instancetypeInformerStore.Add(fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name: fakeInstancetype.Name,
					Kind: apiinstancetype.SingularResourceName,
				}

				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("store VirtualMachineInstancetype ControllerRevision", func() {
				instancetypeControllerRevision, err := revision.CreateControllerRevision(vm, fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(instancetypeControllerRevision.Name))

				createdCR, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(createdCR).To(Equal(instancetypeControllerRevision))
			})

			It("store fails when instancetype does not exist", func() {
				vm.Spec.Instancetype.Name = nonExistingResourceName
				Expect(storeHandler.Store(vm)).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			})

			It("store succeeds when RevisionName already populated", func() {
				instancetypeControllerRevision, err := revision.CreateControllerRevision(vm, fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Instancetype = &virtv1.InstancetypeMatcher{
					Name:         fakeInstancetype.Name,
					RevisionName: instancetypeControllerRevision.Name,
					Kind:         apiinstancetype.SingularResourceName,
				}

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Spec.Instancetype.RevisionName).To(Equal(instancetypeControllerRevision.Name))
				Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {
				instancetypeControllerRevision, err := revision.CreateControllerRevision(vm, fakeInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.InstancetypeRef.ControllerRevisionRef.Name).To(Equal(instancetypeControllerRevision.Name))
			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedInstancetype := fakeInstancetype.DeepCopy()
				unexpectedInstancetype.Spec.CPU.Guest = 15

				instancetypeControllerRevision, err := revision.CreateControllerRevision(vm, unexpectedInstancetype)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), instancetypeControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})

			It("store ControllerRevision fails if instancetype conflicts with vm", func() {
				vm.Spec.Template.Spec.Domain.CPU = &virtv1.CPU{
					Cores: 1,
				}
				Expect(storeHandler.Store(vm)).To(MatchError(conflict.Conflicts{conflict.New("spec", "template", "spec", "domain", "cpu", "cores")}))
			})
		})
	})
	Context("store preference", func() {
		It("store returns error when preferenceMatcher kind is invalid", func() {
			vm.Spec.Preference = &virtv1.PreferenceMatcher{
				Kind: "foobar",
			}
			Expect(storeHandler.Store(vm)).To(MatchError(ContainSubstring("got unexpected kind in PreferenceMatcher")))
		})

		It("store returns nil when no preference is specified", func() {
			vm.Spec.Preference = nil
			Expect(storeHandler.Store(vm)).To(Succeed())
		})
		Context("using global ClusterPreference", func() {
			var clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference

			BeforeEach(func() {
				preferredCPUTopology := instancetypev1beta1.Cores
				clusterPreference = &instancetypev1beta1.VirtualMachineClusterPreference{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachineClusterPreference",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-cluster-preference",
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
							PreferredCPUTopology: &preferredCPUTopology,
						},
					},
				}

				_, err := virtClient.VirtualMachineClusterPreference().Create(context.Background(), clusterPreference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = clusterPreferenceInformerStore.Add(clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &virtv1.PreferenceMatcher{
					Name: clusterPreference.Name,
					Kind: apiinstancetype.ClusterSingularPreferenceResourceName,
				}

				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("store VirtualMachineClusterPreference ControllerRevision", func() {
				clusterPreferenceControllerRevision, err := revision.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(clusterPreferenceControllerRevision.Name))

				createdCR, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.Background(), vm.Status.PreferenceRef.ControllerRevisionRef.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(createdCR).To(Equal(clusterPreferenceControllerRevision))
			})

			It("store fails when VirtualMachineClusterPreference doesn't exist", func() {
				vm.Spec.Preference.Name = nonExistingResourceName
				Expect(storeHandler.Store(vm)).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			})

			It("store succeeds when RevisionName already populated", func() {
				clusterPreferenceControllerRevision, err := revision.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference.RevisionName = clusterPreferenceControllerRevision.Name

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(clusterPreferenceControllerRevision.Name))
				Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(clusterPreferenceControllerRevision.Name))
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {
				clusterPreferenceControllerRevision, err := revision.CreateControllerRevision(vm, clusterPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(clusterPreferenceControllerRevision.Name))
			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedPreference := clusterPreference.DeepCopy()
				preferredCPUTopology := instancetypev1beta1.Threads
				unexpectedPreference.Spec.CPU.PreferredCPUTopology = &preferredCPUTopology

				clusterPreferenceControllerRevision, err := revision.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), clusterPreferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})
			It("store InferFromVolumeFailurePolicy when missing from PreferenceRef", func() {
				vm.Spec.Preference.InferFromVolumeFailurePolicy = pointer.P(virtv1.IgnoreInferFromVolumeFailure)
				vm.Status.PreferenceRef = nil
				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.PreferenceRef.InferFromVolumeFailurePolicy).To(HaveValue(Equal(virtv1.IgnoreInferFromVolumeFailure)))
			})

			It("store InferFromVolumeFailurePolicy when different to PreferenceRef", func() {
				vm.Spec.Preference.InferFromVolumeFailurePolicy = pointer.P(virtv1.IgnoreInferFromVolumeFailure)
				vm.Status.PreferenceRef = &virtv1.InstancetypeStatusRef{
					InferFromVolumeFailurePolicy: pointer.P(virtv1.RejectInferFromVolumeFailure),
				}
				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.PreferenceRef.InferFromVolumeFailurePolicy).To(HaveValue(Equal(virtv1.IgnoreInferFromVolumeFailure)))
			})
		})

		Context("using namespaced Preference", func() {
			var preference *instancetypev1beta1.VirtualMachinePreference

			BeforeEach(func() {
				preferredCPUTopology := instancetypev1beta1.Cores
				preference = &instancetypev1beta1.VirtualMachinePreference{
					TypeMeta: metav1.TypeMeta{
						Kind:       "VirtualMachinePreference",
						APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:       "test-preference",
						Namespace:  vm.Namespace,
						UID:        resourceUID,
						Generation: resourceGeneration,
					},
					Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{
						CPU: &instancetypev1beta1.CPUPreferences{
							PreferredCPUTopology: &preferredCPUTopology,
						},
					},
				}

				_, err := virtClient.VirtualMachinePreference(vm.Namespace).Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = preferenceInformerStore.Add(preference)
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference = &virtv1.PreferenceMatcher{
					Name: preference.Name,
					Kind: apiinstancetype.SingularPreferenceResourceName,
				}

				_, err = virtClient.VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("store VirtualMachinePreference ControllerRevision", func() {
				preferenceControllerRevision, err := revision.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(preferenceControllerRevision.Name))

				createdCR, err := virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
					context.Background(), vm.Status.PreferenceRef.ControllerRevisionRef.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(createdCR).To(Equal(preferenceControllerRevision))
			})

			It("store fails when VirtualMachinePreference doesn't exist", func() {
				vm.Spec.Preference.Name = nonExistingResourceName
				Expect(storeHandler.Store(vm)).To(MatchError(k8serrors.IsNotFound, "IsNotFound"))
			})

			It("store succeeds when RevisionName already populated", func() {
				preferenceControllerRevision, err := revision.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), preferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vm.Spec.Preference.RevisionName = preferenceControllerRevision.Name

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Spec.Preference.RevisionName).To(Equal(preferenceControllerRevision.Name))
				Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(preferenceControllerRevision.Name))
			})

			It("store ControllerRevision succeeds if a revision exists with expected data", func() {
				preferenceControllerRevision, err := revision.CreateControllerRevision(vm, preference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), preferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(Succeed())
				Expect(vm.Status.PreferenceRef.ControllerRevisionRef.Name).To(Equal(preferenceControllerRevision.Name))
			})

			It("store ControllerRevision fails if a revision exists with unexpected data", func() {
				unexpectedPreference := preference.DeepCopy()
				preferredCPUTopology := instancetypev1beta1.Threads
				unexpectedPreference.Spec.CPU.PreferredCPUTopology = &preferredCPUTopology

				preferenceControllerRevision, err := revision.CreateControllerRevision(vm, unexpectedPreference)
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(
					context.Background(), preferenceControllerRevision, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(storeHandler.Store(vm)).To(MatchError(ContainSubstring("found existing ControllerRevision with unexpected data")))
			})
		})
	})
})
