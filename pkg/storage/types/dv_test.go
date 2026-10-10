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
 *
 */

package types

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"

	virtv1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
)

var _ = Describe("DataVolume utils test", func() {
	Context("with VM", func() {
		vm := &virtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "vmnamespace",
				Name:      "vm",
			},
		}

		createClient := func(cdiObjects ...runtime.Object) kubecli.KubevirtClient {
			ctrl := gomock.NewController(GinkgoT())
			virtClient := kubecli.NewMockKubevirtClient(ctrl)
			cdiClient := cdifake.NewSimpleClientset(cdiObjects...)
			virtClient.EXPECT().CdiClient().Return(cdiClient).AnyTimes()
			return virtClient
		}

		It("should ignore DataVolume with no clone operation", func() {
			dv := &cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{
					Blank: &cdiv1.DataVolumeBlankImage{},
				},
			}

			cs, err := GetResolvedCloneSource(context.TODO(), createClient(), vm.Namespace, dv)
			Expect(err).ToNot(HaveOccurred())
			Expect(cs).To(BeNil())
		})

		DescribeTable("should properly handle DataVolume clone source", func(sourceNamespace, expectedNamespace string) {
			sourceName := "name"
			dv := &cdiv1.DataVolumeSpec{
				Source: &cdiv1.DataVolumeSource{
					PVC: &cdiv1.DataVolumeSourcePVC{
						Namespace: sourceNamespace,
						Name:      sourceName,
					},
				},
			}

			cs, err := GetResolvedCloneSource(context.TODO(), createClient(), vm.Namespace, dv)
			Expect(err).ToNot(HaveOccurred())
			Expect(cs).ToNot(BeNil())
			Expect(cs.PVC.Namespace).To(Equal(expectedNamespace))
			Expect(cs.PVC.Name).To(Equal(sourceName))
		},
			Entry("source namespace not specified", "", vm.Namespace),
			Entry("source namespace is specified", "ns2", "ns2"),
		)

		It("should error if DataSource does not exist", func() {
			ns := "foo"
			dv := &cdiv1.DataVolumeSpec{
				SourceRef: &cdiv1.DataVolumeSourceRef{
					Kind:      "DataSource",
					Namespace: &ns,
					Name:      "bar",
				},
			}

			cs, err := GetResolvedCloneSource(context.TODO(), createClient(), vm.Namespace, dv)
			Expect(err).To(HaveOccurred())
			Expect(cs).To(BeNil())
		})

		DescribeTable("should properly handle DataVolume clone sourceRef", func(sourceRefNamespace, sourceNamespace, expectedNamespace string) {
			sourceRefName := "sourceRef"
			sourceName := "name"

			ref := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vm.Namespace,
					Name:      sourceRefName,
				},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: sourceNamespace,
							Name:      sourceName,
						},
					},
				},
			}

			dv := &cdiv1.DataVolumeSpec{
				SourceRef: &cdiv1.DataVolumeSourceRef{
					Kind: "DataSource",
					Name: sourceRefName,
				},
			}

			if sourceRefNamespace != "" {
				ref.Namespace = sourceRefNamespace
				dv.SourceRef.Namespace = &sourceRefNamespace
			}

			cs, err := GetResolvedCloneSource(context.TODO(), createClient(ref), vm.Namespace, dv)
			Expect(err).ToNot(HaveOccurred())
			Expect(cs).ToNot(BeNil())
			Expect(cs.PVC.Namespace).To(Equal(expectedNamespace))
			Expect(cs.PVC.Name).To(Equal(sourceName))
		},
			Entry("sourceRef namespace and source namespace not specified", "", "", vm.Namespace),
			Entry("source namespace not specified", "foo", "", "foo"),
			Entry("sourceRef namespace not specified", "", "bar", "bar"),
			Entry("everything specified", "foo", "bar", "bar"),
		)

		It("should properly handle DataVolume sourceRef pointer", func() {
			sourceRefName := "sourceRef"
			sourceRefPointerName := "sourceRefPointer"
			sourceName := "name"

			pointerRef := &cdiv1.DataSource{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: vm.Namespace,
					Name:      sourceRefPointerName,
				},
				Spec: cdiv1.DataSourceSpec{
					Source: cdiv1.DataSourceSource{
						DataSource: &cdiv1.DataSourceRefSourceDataSource{
							Namespace: vm.Namespace,
							Name:      sourceRefName,
						},
					},
				},
				Status: cdiv1.DataSourceStatus{
					Source: cdiv1.DataSourceSource{
						PVC: &cdiv1.DataVolumeSourcePVC{
							Namespace: vm.Namespace,
							Name:      sourceName,
						},
					},
				},
			}

			dv := &cdiv1.DataVolumeSpec{
				SourceRef: &cdiv1.DataVolumeSourceRef{
					Kind: "DataSource",
					Name: sourceRefPointerName,
				},
			}

			cs, err := GetResolvedCloneSource(context.TODO(), createClient(pointerRef), vm.Namespace, dv)
			Expect(err).ToNot(HaveOccurred())
			Expect(cs).ToNot(BeNil())
			Expect(cs.PVC.Namespace).To(Equal(vm.Namespace))
			Expect(cs.PVC.Name).To(Equal(sourceName))
		})
	})

	Context("ListDataVolumeClaimCandidates", func() {
		var (
			vm    *virtv1.VirtualMachine
			store cache.Store
		)

		BeforeEach(func() {
			vm = &virtv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-vm",
					UID:       types.UID("vm-uid"),
				},
			}
			store = cache.NewStore(controller.KeyFunc)
		})

		ownerRef := func(vm *virtv1.VirtualMachine) metav1.OwnerReference {
			return metav1.OwnerReference{
				APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
				Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
				Name:               vm.Name,
				UID:                vm.UID,
				Controller:         ptr.To(true),
				BlockOwnerDeletion: ptr.To(true),
			}
		}

		dvTemplatesFor := func(names ...string) []virtv1.DataVolumeTemplateSpec {
			templates := make([]virtv1.DataVolumeTemplateSpec, len(names))
			for i, name := range names {
				templates[i] = virtv1.DataVolumeTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Name: name},
				}
			}
			return templates
		}

		It("should return owned DV that is still in templates", func() {
			vm.Spec.DataVolumeTemplates = dvTemplatesFor("dv1")
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "dv1",
					Namespace:       vm.Namespace,
					OwnerReferences: []metav1.OwnerReference{ownerRef(vm)},
				},
			}
			Expect(store.Add(dv)).To(Succeed())

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf(HaveField("Name", "dv1")))
		})

		It("should return owned DV removed from templates for release", func() {
			vm.Spec.DataVolumeTemplates = nil
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "stale-dv",
					Namespace:       vm.Namespace,
					OwnerReferences: []metav1.OwnerReference{ownerRef(vm)},
				},
			}
			Expect(store.Add(dv)).To(Succeed())

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf(HaveField("Name", "stale-dv")))
		})

		It("should return orphan DV matching a template for adoption", func() {
			vm.Spec.DataVolumeTemplates = dvTemplatesFor("orphan-dv")
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "orphan-dv",
					Namespace: vm.Namespace,
				},
			}
			Expect(store.Add(dv)).To(Succeed())

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf(HaveField("Name", "orphan-dv")))
		})

		It("should not duplicate a DV that is both owned and in templates", func() {
			vm.Spec.DataVolumeTemplates = dvTemplatesFor("dv1")
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "dv1",
					Namespace:       vm.Namespace,
					OwnerReferences: []metav1.OwnerReference{ownerRef(vm)},
				},
			}
			Expect(store.Add(dv)).To(Succeed())

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf(HaveField("Name", "dv1")))
		})

		It("should exclude DVs from a different namespace", func() {
			vm.Spec.DataVolumeTemplates = nil
			dv := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "other-ns-dv",
					Namespace:       "other-ns",
					OwnerReferences: []metav1.OwnerReference{ownerRef(vm)},
				},
			}
			Expect(store.Add(dv)).To(Succeed())

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should skip templates with no existing DV", func() {
			vm.Spec.DataVolumeTemplates = dvTemplatesFor("missing-dv")

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("should return both owned stale and orphan template DVs", func() {
			vm.Spec.DataVolumeTemplates = dvTemplatesFor("new-dv")
			staleDV := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:            "stale-dv",
					Namespace:       vm.Namespace,
					OwnerReferences: []metav1.OwnerReference{ownerRef(vm)},
				},
			}
			orphanDV := &cdiv1.DataVolume{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-dv",
					Namespace: vm.Namespace,
				},
			}
			Expect(store.Add(staleDV)).To(Succeed())
			Expect(store.Add(orphanDV)).To(Succeed())

			result, err := ListDataVolumeClaimCandidates(vm, store)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ConsistOf(
				HaveField("Name", "stale-dv"),
				HaveField("Name", "new-dv"),
			))
		})
	})
})
