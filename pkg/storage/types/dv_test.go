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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package types

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	virtv1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
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
	})
})
