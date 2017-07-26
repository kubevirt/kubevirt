/*
 * This file is part of the kubevirt project
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

package designer_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	fake2 "k8s.io/client-go/kubernetes/fake"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/designer"
	"kubevirt.io/kubevirt/pkg/logging"

	/* XXX without this we get unregistered PVC type.
	 * THis must be triggering some other init method.
	 * Figure it out & import it directly */
	_ "kubevirt.io/kubevirt/pkg/virt-handler"
)

var _ = Describe("DomainMap", func() {
	RegisterFailHandler(Fail)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	var (
		k8sClient kubernetes.Interface
	)

	BeforeEach(func() {
		expectedPVC := &k8sv1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-claim",
				Namespace: k8sv1.NamespaceDefault,
			},
			Spec: k8sv1.PersistentVolumeClaimSpec{
				VolumeName: "disk-01",
			},
			Status: k8sv1.PersistentVolumeClaimStatus{
				Phase: k8sv1.ClaimBound,
			},
		}

		source := k8sv1.ISCSIVolumeSource{
			IQN:          "iqn.2009-02.com.test:for.all",
			Lun:          1,
			TargetPortal: "127.0.0.1:6543",
		}

		expectedPV := &k8sv1.PersistentVolume{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolume",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "disk-01",
			},
			Spec: k8sv1.PersistentVolumeSpec{
				PersistentVolumeSource: k8sv1.PersistentVolumeSource{
					ISCSI: &source,
				},
			},
		}

		k8sClient = fake2.NewSimpleClientset(expectedPV, expectedPVC)
	})

	Context("Map Source Disks", func() {
		It("looks up and applies PVC", func() {
			vm := v1.VM{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: k8sv1.NamespaceDefault,
				},
			}

			disk1 := v1.Disk{
				Type:   "PersistentVolumeClaim",
				Device: "disk",
				Source: v1.DiskSource{
					Name: "test-claim",
				},
				Target: v1.DiskTarget{
					Device: "vda",
				},
			}

			disk2 := v1.Disk{
				Type:   "network",
				Device: "cdrom",
				Source: v1.DiskSource{
					Name:     "iqn.2009-02.com.test:for.me/1",
					Protocol: "iscsi",
					Host: &v1.DiskSourceHost{
						Name: "127.0.0.2",
						Port: "3260",
					},
				},
				Target: v1.DiskTarget{
					Device: "hda",
				},
			}

			domain := v1.DomainSpec{}
			domain.Devices.Disks = []v1.Disk{disk1, disk2}
			vm.Spec.Domain = &domain

			domDesign, err := designer.DomainDesignFromAPISpec(&vm, k8sClient)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(domDesign.Domain.Devices.Disks)).To(Equal(2))
			newDisk := domDesign.Domain.Devices.Disks[0]
			Expect(newDisk.Type).To(Equal("network"))
			Expect(newDisk.Driver.Type).To(Equal("raw"))
			Expect(newDisk.Driver.Name).To(Equal("qemu"))
			Expect(newDisk.Device).To(Equal("disk"))
			Expect(newDisk.Source.Protocol).To(Equal("iscsi"))
			Expect(newDisk.Source.Name).To(Equal("iqn.2009-02.com.test:for.all/1"))
			Expect(newDisk.Source.Host.Name).To(Equal("127.0.0.1"))
			Expect(newDisk.Source.Host.Port).To(Equal("6543"))

			newDisk = domDesign.Domain.Devices.Disks[1]
			Expect(newDisk.Type).To(Equal("network"))
			Expect(newDisk.Device).To(Equal("cdrom"))
			Expect(newDisk.Source.Protocol).To(Equal("iscsi"))
			Expect(newDisk.Source.Name).To(Equal("iqn.2009-02.com.test:for.me/1"))
			Expect(newDisk.Source.Host.Name).To(Equal("127.0.0.2"))
			Expect(newDisk.Source.Host.Port).To(Equal("3260"))
		})
	})
})

func TestVMs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "DomainMap")
}
