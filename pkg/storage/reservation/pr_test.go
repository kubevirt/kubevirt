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

package reservation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/reservation"
)

func newFakePVCStore(pvcs ...*k8sv1.PersistentVolumeClaim) cache.Store {
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	for _, pvc := range pvcs {
		Expect(store.Add(pvc)).To(Succeed())
	}
	return store
}

func newPVC(namespace, name string, uid types.UID) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       uid,
		},
	}
}

func newVMIWithPRPVC(namespace, diskName, pvcName string) *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: v1.VirtualMachineInstanceSpec{
			Domain: v1.DomainSpec{
				Devices: v1.Devices{
					Disks: []v1.Disk{
						{
							Name:       diskName,
							DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: true}},
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
							PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				},
			},
		},
	}
}

var _ = Describe("PersistentReservation", func() {
	Context("PersistentReservationPVCLabels", func() {
		It("should return no labels when there are no PR disks", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "disk0",
									DiskDevice: v1.DiskDevice{Disk: &v1.DiskTarget{Bus: v1.DiskBusVirtio}},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "disk0",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
							},
						},
					},
				},
			}

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore())
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(BeEmpty())
		})

		It("should return no labels when LUN has reservation disabled", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "lun0",
									DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: false}},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "lun0",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{},
							},
						},
					},
				},
			}

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore())
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(BeEmpty())
		})

		It("should return a label with PVC UID as key for a single PR PVC", func() {
			pvc := newPVC("default", "my-shared-pvc", "uid-1234")
			vmi := newVMIWithPRPVC("default", "lun0", "my-shared-pvc")

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore(pvc))
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(HaveLen(1))
			Expect(labels).To(HaveKeyWithValue(v1.PersistentReservationLabelPrefix+"uid-1234", ""))
		})

		It("should handle DataVolume sources", func() {
			pvc := newPVC("default", "my-dv", "uid-dv-1")

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "lun0",
									DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: true}},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "lun0",
							VolumeSource: v1.VolumeSource{
								DataVolume: &v1.DataVolumeSource{Name: "my-dv"},
							},
						},
					},
				},
			}

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore(pvc))
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(HaveLen(1))
			Expect(labels).To(HaveKey(v1.PersistentReservationLabelPrefix + "uid-dv-1"))
		})

		It("should return labels for multiple PR PVCs", func() {
			pvcA := newPVC("default", "pvc-a", "uid-a")
			pvcB := newPVC("default", "pvc-b", "uid-b")

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "lun0",
									DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: true}},
								},
								{
									Name:       "lun1",
									DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: true}},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "lun0",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc-a"},
								},
							},
						},
						{
							Name: "lun1",
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc-b"},
								},
							},
						},
					},
				},
			}

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore(pvcA, pvcB))
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(HaveLen(2))
			Expect(labels).To(HaveKey(v1.PersistentReservationLabelPrefix + "uid-a"))
			Expect(labels).To(HaveKey(v1.PersistentReservationLabelPrefix + "uid-b"))
		})

		It("should return no labels when disk has no matching volume", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "lun0",
									DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: true}},
								},
							},
						},
					},
					Volumes: []v1.Volume{},
				},
			}

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore())
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(BeEmpty())
		})

		It("should return no labels for non-PVC volumes", func() {
			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name:       "lun0",
									DiskDevice: v1.DiskDevice{LUN: &v1.LunTarget{Bus: v1.DiskBusSCSI, Reservation: true}},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "lun0",
							VolumeSource: v1.VolumeSource{
								ContainerDisk: &v1.ContainerDiskSource{Image: "test"},
							},
						},
					},
				},
			}

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore())
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(BeEmpty())
		})

		It("should skip PVCs not found in cache", func() {
			vmi := newVMIWithPRPVC("default", "lun0", "missing-pvc")

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore())
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(BeEmpty())
		})

		It("should use PVC UID in label key", func() {
			pvc := newPVC("default", "test-pvc", "550e8400-e29b-41d4-a716-446655440000")
			vmi := newVMIWithPRPVC("default", "lun0", "test-pvc")

			labels, err := reservation.PersistentReservationPVCLabels(vmi, newFakePVCStore(pvc))
			Expect(err).ToNot(HaveOccurred())
			Expect(labels).To(HaveLen(1))
			Expect(labels).To(HaveKeyWithValue(
				v1.PersistentReservationLabelPrefix+"550e8400-e29b-41d4-a716-446655440000",
				"",
			))
		})
	})

	Context("PersistentReservationPodAntiAffinityTerms", func() {
		It("should return no terms for empty labels", func() {
			terms := reservation.PersistentReservationPodAntiAffinityTerms(map[string]string{})
			Expect(terms).To(BeEmpty())
		})

		It("should return a term with correct topology key and Exists operator", func() {
			labels := map[string]string{
				"pr.kubevirt.io/uid-1234": "my-pvc",
			}
			terms := reservation.PersistentReservationPodAntiAffinityTerms(labels)
			Expect(terms).To(HaveLen(1))
			Expect(terms[0].TopologyKey).To(Equal("kubernetes.io/hostname"))
			Expect(terms[0].LabelSelector).ToNot(BeNil())
			Expect(terms[0].LabelSelector.MatchExpressions).To(HaveLen(1))
			Expect(terms[0].LabelSelector.MatchExpressions[0].Key).To(Equal("pr.kubevirt.io/uid-1234"))
			Expect(terms[0].LabelSelector.MatchExpressions[0].Operator).To(Equal(metav1.LabelSelectorOpExists))
		})

		It("should return one term per label", func() {
			labels := map[string]string{
				"pr.kubevirt.io/uid-aaa": "pvc-1",
				"pr.kubevirt.io/uid-bbb": "pvc-2",
			}
			terms := reservation.PersistentReservationPodAntiAffinityTerms(labels)
			Expect(terms).To(HaveLen(2))
			for _, term := range terms {
				Expect(term.TopologyKey).To(Equal("kubernetes.io/hostname"))
			}
		})
	})
})
