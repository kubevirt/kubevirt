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

package tests_test

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

const (
	newLabelKey   = "newlabel"
	newLabelValue = "newvalue"
)

var _ = Describe("[sig-compute]VirtualMachinePool", decorators.SigCompute, func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	waitForVMIs := func(namespace string, expectedCount int) {
		Eventually(func() error {
			vmis, err := virtClient.VirtualMachineInstance(namespace).List(context.Background(), &v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(vmis.Items) != expectedCount {
				return fmt.Errorf("Only %d vmis exist, expected %d", len(vmis.Items), expectedCount)
			}

			for _, vmi := range vmis.Items {
				if vmi.Status.Phase != v1.Running {
					return fmt.Errorf("Waiting on vmi with phase %s to be Running", vmi.Status.Phase)
				}
			}

			return nil
		}, 120*time.Second, 1*time.Second).Should(BeNil())

	}

	doScale := func(name string, scale int32) {

		By(fmt.Sprintf("Scaling to %d", scale))
		pool, err := virtClient.VirtualMachinePool(util.NamespaceTestDefault).Patch(context.Background(), name, types.JSONPatchType, []byte(fmt.Sprintf("[{ \"op\": \"replace\", \"path\": \"/spec/replicas\", \"value\": %v }]", scale)), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		running := *pool.Spec.VirtualMachineTemplate.Spec.Running
		By("Checking the number of replicas")
		Eventually(func() int32 {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if running {
				return pool.Status.ReadyReplicas
			}
			return pool.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(scale)))

		vms, err := virtClient.VirtualMachine(util.NamespaceTestDefault).List(context.Background(), &v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(notDeletedVMs(pool.Name, vms)).To(HaveLen(int(scale)))
	}
	createVirtualMachinePool := func(pool *poolv1.VirtualMachinePool) *poolv1.VirtualMachinePool {
		pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Create(context.Background(), pool, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pool
	}

	newPersistentStorageVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool with persistent storage")

		vm, foundSC := tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault)
		if !foundSC {
			Skip("Skip test when Filesystem storage is not present")
		}

		newPool := newPoolFromVMI(&v1.VirtualMachineInstance{
			ObjectMeta: vm.Spec.Template.ObjectMeta,
			Spec:       vm.Spec.Template.Spec,
		})
		newPool.Spec.VirtualMachineTemplate.Spec.DataVolumeTemplates = vm.Spec.DataVolumeTemplates
		running := true
		newPool.Spec.VirtualMachineTemplate.Spec.Running = &running
		newPool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		return newPool
	}

	newVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool")
		pool := newPoolFromVMI(libvmi.New(libvmi.WithResourceMemory("2Mi")))
		running := true
		pool.Spec.VirtualMachineTemplate.Spec.Running = &running
		return createVirtualMachinePool(pool)
	}

	newOfflineVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool")
		return createVirtualMachinePool(newPoolFromVMI(libvmi.NewCirros()))
	}

	DescribeTable("[Serial]pool should scale", Serial, func(startScale int, stopScale int) {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, int32(startScale))
		doScale(newPool.ObjectMeta.Name, int32(stopScale))
		doScale(newPool.ObjectMeta.Name, int32(0))
	},
		Entry("[test_cid:31535]to three, to two and then to zero replicas", 3, 2),
		Entry("[test_cid:27840]to five, to six and then to zero replicas", 5, 6),
	)

	It("[test_cid:21476]should be rejected on POST if spec is invalid", func() {
		newPool := newOfflineVirtualMachinePool()
		newPool.TypeMeta = v12.TypeMeta{
			APIVersion: poolv1.SchemeGroupVersion.String(),
			Kind:       poolv1.VirtualMachinePoolKind,
		}

		newPool.Spec.VirtualMachineTemplate.Spec.RunStrategy = nil
		newPool.Spec.VirtualMachineTemplate.Spec.Running = nil
		newPool.Spec.VirtualMachineTemplate.ObjectMeta.Labels = map[string]string{}
		_, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err.Error()).To(ContainSubstring("selector does not match labels"))
	})

	It("[test_cid:18605]should reject POST if vmi spec is invalid", func() {
		newPool := newOfflineVirtualMachinePool()
		newPool.TypeMeta = v12.TypeMeta{
			APIVersion: poolv1.SchemeGroupVersion.String(),
			Kind:       poolv1.VirtualMachinePoolKind,
		}

		// Add a disk that doesn't map to a volume.
		// This should get rejected which tells us the webhook validator is working.
		newPool.Spec.VirtualMachineTemplate.Spec.Template.Spec.Domain.Devices.Disks = append(newPool.Spec.VirtualMachineTemplate.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})

		_, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err.Error()).To(ContainSubstring("admission webhook \"virtualmachinepool-validator.kubevirt.io\" denied the request: spec.virtualMachineTemplate.spec.template.spec.domain.devices.disks[2].Name 'testdisk' not found"))
	})

	It("[test_cid:42278]should remove VMs once they are marked for deletion", func() {
		newPool := newVirtualMachinePool()
		// Create a pool with two replicas
		doScale(newPool.ObjectMeta.Name, 2)
		// Delete it
		By("Deleting the VirtualMachinePool")
		Expect(virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Delete(context.Background(), newPool.ObjectMeta.Name, metav1.DeleteOptions{})).To(Succeed())
		// Wait until VMs are gone
		By("Waiting until all VMs are gone")
		Eventually(func() int {
			vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 120*time.Second, 1*time.Second).Should(BeZero())
	})

	It("[test_cid:20556]should handle pool with dataVolumeTemplates", func() {
		newPool := newPersistentStorageVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 2)

		var (
			err       error
			vms       *v1.VirtualMachineList
			dvs       *cdiv1.DataVolumeList
			pvcs      *corev1.PersistentVolumeClaimList
			dvOrigUID types.UID
		)

		By("Waiting until all VMs are created")
		Eventually(func() int {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 60*time.Second, 1*time.Second).Should(Equal(2))

		By("Waiting until all VMIs are created and online")
		waitForVMIs(newPool.Namespace, 2)

		// Select a VM to delete, and record the VM and DV/PVC UIDs associated with the VM.
		origUID := vms.Items[0].UID
		name := vms.Items[0].Name
		dvName := vms.Items[0].Spec.DataVolumeTemplates[0].ObjectMeta.Name
		dvName1 := vms.Items[1].Spec.DataVolumeTemplates[0].ObjectMeta.Name
		Expect(dvName).ToNot(Equal(dvName1))

		isGC := libstorage.IsDataVolumeGC(virtClient)
		if isGC {
			By("Ensure PVCs are created")
			pvcs, err = virtClient.CoreV1().PersistentVolumeClaims(newPool.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			pvcCount := 0
			for _, pvc := range pvcs.Items {
				if pvc.Name == dvName || pvc.Name == dvName1 {
					pvcCount++
					if pvc.Name == dvName {
						dvOrigUID = pvc.UID
					}
				}
			}
			Expect(pvcCount).To(Equal(2))
		} else {
			By("Ensure DataVolumes are created")
			dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(newPool.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(dvs.Items).To(HaveLen(2))
			for _, dv := range dvs.Items {
				if dv.Name == dvName {
					dvOrigUID = dv.UID
				}
			}
		}

		Expect(string(dvOrigUID)).ToNot(Equal(""))

		By("deleting a VM")
		foreGround := metav1.DeletePropagationForeground
		Expect(virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).Delete(context.Background(), name, &k8smetav1.DeleteOptions{PropagationPolicy: &foreGround})).To(Succeed())

		By("Waiting for deleted VM to be replaced")
		Eventually(func() error {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
			if err != nil {
				return err
			}

			if len(vms.Items) != 2 {
				return fmt.Errorf("Only %d vms exist, expected 2", len(vms.Items))
			}

			found := false
			for _, vm := range vms.Items {
				if vm.Name == name && vm.UID != origUID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Waiting on VM named %s with new UID to appear", name)
			}
			return nil

		}, 120*time.Second, 1*time.Second).Should(BeNil())

		By("Waiting until all VMIs are created and online again")
		waitForVMIs(newPool.Namespace, 2)

		if isGC {
			pvcs, err = virtClient.CoreV1().PersistentVolumeClaims(newPool.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verify pvc for deleted VM is replaced")
			pvcCount := 0
			for _, pvc := range pvcs.Items {
				if pvc.Name == dvName || pvc.Name == dvName1 {
					Expect(pvc.UID).ToNot(Equal(dvOrigUID))
					pvcCount++
				}
			}
			By("Verify pvc count after VM replacement")
			Expect(pvcCount).To(Equal(2))
		} else {
			By("Verify datavolume count after VM replacement")
			dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(newPool.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(dvs.Items).To(HaveLen(2))

			By("Verify datavolume for deleted VM is replaced")
			for _, dv := range dvs.Items {
				Expect(dv.UID).ToNot(Equal(dvOrigUID))
			}
		}
	})

	It("[test_cid:30524]should replace deleted VM and get replacement", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 3)

		var err error
		var vms *v1.VirtualMachineList

		By("Waiting until all VMs are created")
		Eventually(func() int {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 120*time.Second, 1*time.Second).Should(Equal(3))

		origUID := vms.Items[1].UID
		name := vms.Items[1].Name

		By("deleting a VM")
		Expect(virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).Delete(context.Background(), name, &k8smetav1.DeleteOptions{})).To(Succeed())

		By("Waiting for deleted VM to be replaced")
		Eventually(func() error {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
			if err != nil {
				return err
			}

			if len(vms.Items) != 3 {
				return fmt.Errorf("Only %d vms exist, expected 3", len(vms.Items))
			}

			found := false
			for _, vm := range vms.Items {
				if vm.Name == name && vm.UID != origUID {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("Waiting on VM named %s with new UID to appear", name)
			}
			return nil

		}, 120*time.Second, 1*time.Second).Should(BeNil())

	})

	It("[test_cid:32931]should roll out VM template changes without impacting VMI", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 1)
		waitForVMIs(newPool.Namespace, 1)

		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(vms.Items).To(HaveLen(1))

		name := vms.Items[0].Name
		vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(context.Background(), name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmiUID := vmi.UID

		By("Rolling Out VM template change")
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		patchData, err := patch.GeneratePatchPayload(patch.PatchOperation{
			Op:    patch.PatchAddOp,
			Path:  fmt.Sprintf("/spec/virtualMachineTemplate/metadata/labels/%s", newLabelKey),
			Value: newLabelValue,
		})
		Expect(err).ToNot(HaveOccurred())
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Patch(context.Background(), newPool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring VM picks up label")
		Eventually(func() error {

			vm, err := virtClient.VirtualMachine(newPool.Namespace).Get(context.Background(), name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, ok := vm.Labels[newLabelKey]
			if !ok {
				return fmt.Errorf("Expected vm pool update to roll out to VMs")
			}

			return nil
		}, 30*time.Second, 1*time.Second).Should(BeNil())

		By("Ensuring VMI remains consistent and isn't restarted")
		Consistently(func() error {
			vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(context.Background(), name, &metav1.GetOptions{})
			if err != nil {
				return nil
			}

			Expect(vmi.UID).To(Equal(vmiUID))
			Expect(vmi.DeletionTimestamp).To(BeNil())
			return nil
		}, 5*time.Second, 1*time.Second).Should(BeNil())
	})

	It("[test_cid:34072]should roll out VMI template changes and proactively roll out new VMIs", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 1)
		waitForVMIs(newPool.Namespace, 1)

		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})

		Expect(err).ToNot(HaveOccurred())
		Expect(vms.Items).To(HaveLen(1))

		name := vms.Items[0].Name
		vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(context.Background(), name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmiUID := vmi.UID

		By("Rolling Out VM template change")
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Make a VMI template change
		patchData, err := patch.GeneratePatchPayload(patch.PatchOperation{
			Op:    patch.PatchAddOp,
			Path:  fmt.Sprintf("/spec/virtualMachineTemplate/spec/template/metadata/labels/%s", newLabelKey),
			Value: newLabelValue,
		})
		Expect(err).ToNot(HaveOccurred())
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Patch(context.Background(), newPool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring VM picks up label")
		Eventually(func() error {

			vm, err := virtClient.VirtualMachine(newPool.Namespace).Get(context.Background(), name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, ok := vm.Spec.Template.ObjectMeta.Labels[newLabelKey]
			if !ok {
				return fmt.Errorf("Expected vm pool update to roll out to VMs")
			}

			return nil
		}, 30*time.Second, 1*time.Second).Should(BeNil())

		By("Ensuring VMI is re-created to pick up new label")
		Eventually(func() error {
			vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(context.Background(), name, &metav1.GetOptions{})
			if err != nil {
				return nil
			}

			if vmi.UID == vmiUID {
				return fmt.Errorf("Waiting on VMI to get deleted and recreated")
			}
			_, ok := vmi.ObjectMeta.Labels[newLabelKey]
			if !ok {
				return fmt.Errorf("Expected vmi to pick up the new updated label")
			}
			return nil
		}, 60*time.Second, 1*time.Second).Should(BeNil())
	})

	It("[test_cid:16120]should remove owner references on the VirtualMachine if it is orphan deleted", func() {
		newPool := newOfflineVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 2)

		// Check for owner reference
		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
		Expect(vms.Items).To(HaveLen(2))
		Expect(err).ToNot(HaveOccurred())
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).ToNot(BeEmpty())
		}

		// Delete it
		By("Deleting the VirtualMachine pool with the 'orphan' deletion strategy")
		orphanPolicy := v12.DeletePropagationOrphan
		Expect(virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Delete(context.Background(), newPool.ObjectMeta.Name, v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
		// Wait until the pool is deleted
		By("Waiting until the pool got deleted")
		Eventually(func() bool {
			_, err := virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, v12.GetOptions{})
			if errors.IsNotFound(err) {
				return true
			}
			return false
		}, 60*time.Second, 1*time.Second).Should(BeTrue())

		By("Checking if two VMs are orphaned and still exist")
		vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), &v12.ListOptions{})
		Expect(vms.Items).To(HaveLen(2))

		By("Checking a VirtualMachine owner references")
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).To(BeEmpty())
		}
		Expect(err).ToNot(HaveOccurred())
	})

	It("[test_cid:19990]should not scale when paused and scale when resume", func() {
		pool := newOfflineVirtualMachinePool()
		// pause controller
		By("Pausing the pool")
		_, err := virtClient.VirtualMachinePool(pool.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, []byte("[{ \"op\": \"add\", \"path\": \"/spec/paused\", \"value\": true }]"), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() *poolv1.VirtualMachinePool {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pool
		}, 10*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(poolv1.VirtualMachinePoolReplicaPaused))

		// set new replica count while still being paused
		By("Updating the number of replicas")
		patchData, err := patch.GenerateTestReplacePatch("/spec/replicas", pool.Spec.Replicas, tests.NewInt32(1))
		Expect(err).ToNot(HaveOccurred())
		pool, err = virtClient.VirtualMachinePool(pool.ObjectMeta.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		// make sure that we don't scale up
		By("Checking that the pool do not scale while it is paused")
		Consistently(func() int32 {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			// Make sure that no failure happened, so that ensure that we don't scale because we are paused
			Expect(pool.Status.Conditions).To(HaveLen(1))
			return pool.Status.Replicas
		}, 5*time.Second, 1*time.Second).Should(Equal(int32(0)))

		// resume controller
		By("Resuming the pool")
		_, err = virtClient.VirtualMachinePool(pool.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, []byte("[{ \"op\": \"replace\", \"path\": \"/spec/paused\", \"value\": false }]"), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		// Paused condition should disappear
		By("Checking that the pause condition disappeared from the pool")
		Eventually(func() int {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(pool.Status.Conditions)
		}, 10*time.Second, 1*time.Second).Should(Equal(0))

		// Replicas should be created
		By("Checking that the missing replicas are now created")
		Eventually(func() int32 {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pool.Status.Replicas
		}, 10*time.Second, 1*time.Second).Should(Equal(int32(1)))
	})
})

func newPoolFromVMI(vmi *v1.VirtualMachineInstance) *poolv1.VirtualMachinePool {
	selector := "pool" + rand.String(5)
	running := false
	replicas := int32(0)
	pool := &poolv1.VirtualMachinePool{
		ObjectMeta: metav1.ObjectMeta{Name: "pool" + rand.String(5)},
		Spec: poolv1.VirtualMachinePoolSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"select": selector},
			},
			VirtualMachineTemplate: &poolv1.VirtualMachineTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"select": selector},
				},
				Spec: v1.VirtualMachineSpec{
					Running: &running,
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"select": selector},
						},
						Spec: vmi.Spec,
					},
				},
			},
		},
	}
	return pool
}

func notDeletedVMs(poolName string, vms *v1.VirtualMachineList) (notDeleted []v1.VirtualMachine) {
	nonDeletedVms := tests.NotDeletedVMs(vms)
	for _, vm := range nonDeletedVms {
		for _, ref := range vm.OwnerReferences {
			if ref.Name == poolName {
				notDeleted = append(notDeleted, vm)
			}
		}
	}
	return
}
