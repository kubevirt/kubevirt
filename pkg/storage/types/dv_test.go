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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"

	storagev1 "k8s.io/api/storage/v1"
	virtv1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/testutils"
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

	Context("IsStorageClassCSI", func() {
		var (
			dataVolumeStore cache.Store
			scStore         cache.Store
			csiDriverStore  cache.Store

			dvCSI     *cdiv1.DataVolume
			dvNoSCSI  *cdiv1.DataVolume
			dvNoSc    *cdiv1.DataVolume
			csiSC     *storagev1.StorageClass
			noCSISc   *storagev1.StorageClass
			csiDriver *storagev1.CSIDriver
		)
		const (
			noCSIDVName   = "nocsi-dv"
			csiDVName     = "csi-dv"
			noSCDVName    = "no-sc-dv"
			noCSISCName   = "nocsi"
			csiSCName     = "csi-sc"
			scNoExist     = "noexist"
			csiDriverName = "csi-driver"
			ns            = "test"
		)
		BeforeEach(func() {
			dataVolumeInformer, _ := testutils.NewFakeInformerFor(&cdiv1.DataVolume{})
			scInformer, _ := testutils.NewFakeInformerFor(&storagev1.StorageClass{})
			csiDriverInformer, _ := testutils.NewFakeInformerFor(&storagev1.CSIDriver{})

			dataVolumeStore = dataVolumeInformer.GetStore()
			scStore = scInformer.GetStore()
			csiDriverStore = csiDriverInformer.GetStore()

			dvCSI = libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(csiDVName),
				libdv.WithStorage(libdv.StorageWithStorageClass(csiSCName)))
			dvNoSCSI = libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(noCSIDVName),
				libdv.WithStorage(libdv.StorageWithStorageClass(noCSISCName)))
			dvNoSc = libdv.NewDataVolume(libdv.WithNamespace(ns), libdv.WithName(noSCDVName),
				libdv.WithStorage(libdv.StorageWithStorageClass(scNoExist)))

			csiSC = &storagev1.StorageClass{
				ObjectMeta:  metav1.ObjectMeta{Name: csiSCName},
				Provisioner: csiDriverName,
			}
			noCSISc = &storagev1.StorageClass{
				ObjectMeta:  metav1.ObjectMeta{Name: noCSISCName},
				Provisioner: "kubernetes.io/no-provisioner",
			}
			csiDriver = &storagev1.CSIDriver{
				ObjectMeta: metav1.ObjectMeta{Name: csiDriverName},
			}

			Expect(dataVolumeStore.Add(dvCSI)).To(Succeed())
			Expect(dataVolumeStore.Add(dvNoSCSI)).To(Succeed())
			Expect(dataVolumeStore.Add(dvNoSc)).To(Succeed())
			Expect(scStore.Add(csiSC)).To(Succeed())
			Expect(scStore.Add(noCSISc)).To(Succeed())
			Expect(csiDriverStore.Add(csiDriver)).To(Succeed())
		})

		It("should return an error with not existing datavolumes", func() {
			res, err := IsStorageClassCSI(ns, "noexist", dataVolumeStore, scStore, csiDriverStore)
			Expect(err).To(MatchError(fmt.Sprintf("datavolume %s/noexist doesn't exist", ns)))
			Expect(res).To(BeFalse())
		})

		It("should return an error with not existing storage class", func() {
			res, err := IsStorageClassCSI(ns, noSCDVName, dataVolumeStore, scStore, csiDriverStore)
			Expect(err).To(MatchError(fmt.Sprintf("storage class %s for datavolume %s/%s doesn't exist", scNoExist, ns, noSCDVName)))
			Expect(res).To(BeFalse())
		})

		It("should return true if the datavolume storageclass is a csi driver", func() {
			res, err := IsStorageClassCSI(ns, csiDVName, dataVolumeStore, scStore, csiDriverStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(BeTrue())
		})

		It("should return false if the datavolume if the datavolume storageclass isn't a csi driver", func() {
			res, err := IsStorageClassCSI(ns, noCSIDVName, dataVolumeStore, scStore, csiDriverStore)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(BeFalse())
		})
	})
})
