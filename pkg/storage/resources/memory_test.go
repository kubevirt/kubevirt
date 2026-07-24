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

package resources_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/storage/resources"
	"kubevirt.io/kubevirt/pkg/testutils"
)

var _ = Describe("Storage memory overhead", func() {
	var (
		pvcStore        cache.Store
		trackerInformer cache.SharedIndexInformer
		calc            resources.MemoryCalculator
	)

	BeforeEach(func() {
		pvcStore = cache.NewStore(cache.MetaNamespaceKeyFunc)
		trackerInformer, _ = testutils.NewFakeInformerWithIndexersFor(
			&backupv1.VirtualMachineBackupTracker{},
			controller.GetVirtualMachineBackupTrackerInformerIndexers(),
		)
		calc = resources.NewMemoryCalculator(pvcStore, trackerInformer)
	})

	addPVC := func(name, namespace string, capacity resource.Quantity) {
		pvc := &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Status: k8sv1.PersistentVolumeClaimStatus{
				Capacity: k8sv1.ResourceList{k8sv1.ResourceStorage: capacity},
			},
		}
		Expect(pvcStore.Add(pvc)).To(Succeed())
	}

	addTracker := func(name, vmiName, namespace string) {
		tracker := &backupv1.VirtualMachineBackupTracker{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: backupv1.VirtualMachineBackupTrackerSpec{
				Source: k8sv1.TypedLocalObjectReference{
					APIGroup: new("kubevirt.io"),
					Kind:     "VirtualMachine",
					Name:     vmiName,
				},
			},
		}
		Expect(trackerInformer.GetStore().Add(tracker)).To(Succeed())
	}

	cbtEnabled := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
		vmi.Status.ChangedBlockTracking = &v1.ChangedBlockTrackingStatus{
			State: v1.ChangedBlockTrackingEnabled,
		}
		return vmi
	}

	It("should return zero when CBT is not enabled", func() {
		addPVC("test-pvc", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addTracker("tracker1", "test-vmi", metav1.NamespaceDefault)
		testVMI := libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "test-pvc"),
		)
		overhead := calc.Calculate(testVMI)
		Expect(overhead.IsZero()).To(BeTrue())
	})

	It("should return only buffer overhead with no eligible volumes", func() {
		testVMI := cbtEnabled(libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault), libvmi.WithName("test-vmi")))
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("20Mi"))
	})

	It("should calculate bitmap overhead for a 1TiB disk with 1 tracker", func() {
		addPVC("test-pvc", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addTracker("tracker1", "test-vmi", metav1.NamespaceDefault)
		testVMI := cbtEnabled(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "test-pvc"),
		))
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("22Mi"))
	})

	It("should multiply bitmap overhead by tracker count", func() {
		addPVC("test-pvc", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addTracker("tracker1", "test-vmi", metav1.NamespaceDefault)
		addTracker("tracker2", "test-vmi", metav1.NamespaceDefault)
		testVMI := cbtEnabled(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "test-pvc"),
		))
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("24Mi"))
	})

	It("should sum overhead across multiple disks", func() {
		addPVC("pvc1", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addPVC("pvc2", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addTracker("tracker1", "test-vmi", metav1.NamespaceDefault)
		testVMI := cbtEnabled(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "pvc1"),
			libvmi.WithPersistentVolumeClaim("disk1", "pvc2"),
		))
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("24Mi"))
	})

	It("should skip non-eligible volumes", func() {
		addPVC("test-pvc", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addTracker("tracker1", "test-vmi", metav1.NamespaceDefault)
		testVMI := cbtEnabled(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "test-pvc"),
		))
		testVMI.Spec.Volumes = append(testVMI.Spec.Volumes, v1.Volume{
			Name:         "cloudinit",
			VolumeSource: v1.VolumeSource{CloudInitNoCloud: &v1.CloudInitNoCloudSource{UserData: "test"}},
		})
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("22Mi"))
	})

	It("should scale bitmap overhead proportionally to disk size", func() {
		addPVC("small-pvc", metav1.NamespaceDefault, resource.MustParse("512Gi"))
		addPVC("large-pvc", metav1.NamespaceDefault, resource.MustParse("2Ti"))
		addTracker("tracker1", "test-vmi", metav1.NamespaceDefault)
		testVMI := cbtEnabled(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "small-pvc"),
			libvmi.WithPersistentVolumeClaim("disk1", "large-pvc"),
		))
		// 512Gi / 64Ki / 8 = 1Mi bitmap, 2Ti / 64Ki / 8 = 4Mi bitmap
		// Total: 1Mi + 4Mi + 20Mi buffer = 25Mi
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("25Mi"))
	})

	It("should not count trackers from a different namespace", func() {
		addPVC("test-pvc", metav1.NamespaceDefault, resource.MustParse("1Ti"))
		addTracker("tracker1", "test-vmi", "other-namespace")
		testVMI := cbtEnabled(libvmi.New(
			libvmi.WithNamespace(metav1.NamespaceDefault),
			libvmi.WithName("test-vmi"),
			libvmi.WithPersistentVolumeClaim("disk0", "test-pvc"),
		))
		overhead := calc.Calculate(testVMI)
		Expect(overhead.String()).To(Equal("20Mi"))
	})
})
