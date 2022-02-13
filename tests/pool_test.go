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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[sig-compute]VirtualMachinePool", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	waitForVMIs := func(namespace string, expectedCount int) {
		Eventually(func() error {
			vmis, err := virtClient.VirtualMachineInstance(namespace).List(&v12.ListOptions{})
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

		By("Checking the number of replicas")
		Eventually(func() int32 {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pool.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(scale)))

		vms, err := virtClient.VirtualMachine(util.NamespaceTestDefault).List(&v12.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tests.NotDeletedVMs(vms)).To(HaveLen(int(scale)))
	}
	newVirtualMachinePoolWithTemplate := func(template *v1.VirtualMachineInstance, running bool) *poolv1.VirtualMachinePool {
		newPool := tests.NewRandomPoolFromVMI(template, int32(0), running)
		newPool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return newPool
	}

	newPersistentStorageVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool with persistent storage")

		vm := tests.NewRandomVMWithDataVolume(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), util.NamespaceTestDefault)

		newPool := tests.NewRandomPoolFromVMI(&v1.VirtualMachineInstance{
			ObjectMeta: vm.Spec.Template.ObjectMeta,
			Spec:       vm.Spec.Template.Spec,
		}, int32(0), true)
		newPool.Spec.VirtualMachineTemplate.Spec.DataVolumeTemplates = vm.Spec.DataVolumeTemplates
		newPool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		return newPool
	}

	newVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool")
		template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		return newVirtualMachinePoolWithTemplate(template, true)
	}
	newOfflineVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool")
		template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
		return newVirtualMachinePoolWithTemplate(template, false)
	}

	DescribeTable("[Serial]pool should scale", func(startScale int, stopScale int) {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, int32(startScale))
		doScale(newPool.ObjectMeta.Name, int32(stopScale))
		doScale(newPool.ObjectMeta.Name, int32(0))

	},
		Entry("[QUARANTINE]to three, to two and then to zero replicas", 3, 2),
		Entry("[QUARANTINE]to five, to six and then to zero replicas", 5, 6),
	)

	It("should be rejected on POST if spec is invalid", func() {
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

	It("should reject POST if vmi spec is invalid", func() {
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

	It("should remove VMs once they are marked for deletion", func() {
		newPool := newVirtualMachinePool()
		// Create a pool with two replicas
		doScale(newPool.ObjectMeta.Name, 2)
		// Delete it
		By("Deleting the VirtualMachinePool")
		Expect(virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Delete(context.Background(), newPool.ObjectMeta.Name, metav1.DeleteOptions{})).To(Succeed())
		// Wait until VMs are gone
		By("Waiting until all VMs are gone")
		Eventually(func() int {
			vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 120*time.Second, 1*time.Second).Should(BeZero())
	})

	It("should handle pool with dataVolumeTemplates", func() {
		newPool := newPersistentStorageVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 2)

		var err error
		var vms *v1.VirtualMachineList

		By("Waiting until all VMs are created")
		Eventually(func() int {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 60*time.Second, 1*time.Second).Should(Equal(2))

		By("Waiting until all VMIs are created and online")
		waitForVMIs(newPool.Namespace, 2)

		By("Ensure DataVolumes are created")
		dvs, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(newPool.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(dvs.Items).To(HaveLen(2))

		// Select a VM to delete, and record the VM and DV UIDs associated with the VM.
		origUID := vms.Items[0].UID
		name := vms.Items[0].Name
		dvName := vms.Items[0].Spec.DataVolumeTemplates[0].ObjectMeta.Name
		var dvOrigUID types.UID
		for _, dv := range dvs.Items {
			if dv.Name == dvName {
				dvOrigUID = dv.UID
			}
		}
		Expect(string(dvOrigUID)).ToNot(Equal(""))

		By("deleting a VM")
		foreGround := metav1.DeletePropagationForeground
		Expect(virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).Delete(name, &k8smetav1.DeleteOptions{PropagationPolicy: &foreGround})).To(Succeed())

		By("Waiting for deleted VM to be replaced")
		Eventually(func() error {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
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

		By("Verify datavolume count after VM replacement")
		dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(newPool.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(dvs.Items).To(HaveLen(2))

		By("Verify datavolume for deleted VM is replaced")
		for _, dv := range dvs.Items {
			Expect(dv.UID).ToNot(Equal(dvOrigUID))
		}
	})

	It("should replace deleted VM and get replacement", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 3)

		var err error
		var vms *v1.VirtualMachineList

		By("Waiting until all VMs are created")
		Eventually(func() int {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 120*time.Second, 1*time.Second).Should(Equal(3))

		origUID := vms.Items[1].UID
		name := vms.Items[1].Name

		By("deleting a VM")
		Expect(virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).Delete(name, &k8smetav1.DeleteOptions{})).To(Succeed())

		By("Waiting for deleted VM to be replaced")
		Eventually(func() error {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
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

	It("should roll out VM template changes without impacting VMI", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 1)
		waitForVMIs(newPool.Namespace, 1)

		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})

		Expect(err).To(BeNil())
		Expect(len(vms.Items)).To(Equal(1))

		name := vms.Items[0].Name
		vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(name, &metav1.GetOptions{})
		Expect(err).To(BeNil())

		vmiUID := vmi.UID

		By("Rolling Out VM template change")
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, v12.GetOptions{})
		Expect(err).To(BeNil())

		newPool.Spec.VirtualMachineTemplate.ObjectMeta.Labels["newlabel"] = "newvalue"
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Update(context.Background(), newPool, metav1.UpdateOptions{})
		Expect(err).To(BeNil())

		By("Ensuring VM picks up label")
		Eventually(func() error {

			vm, err := virtClient.VirtualMachine(newPool.Namespace).Get(name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, ok := vm.Labels["newlabel"]
			if !ok {
				return fmt.Errorf("Expected vm pool update to roll out to VMs")
			}

			return nil
		}, 30*time.Second, 1*time.Second).Should(BeNil())

		By("Ensuring VMI remains consistent and isn't restarted")
		Consistently(func() error {
			vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(name, &metav1.GetOptions{})
			if err != nil {
				return nil
			}

			Expect(vmi.UID).To(Equal(vmiUID))
			Expect(vmi.DeletionTimestamp).To(BeNil())
			return nil
		}, 5*time.Second, 1*time.Second).Should(BeNil())
	})

	It("should roll out VMI template changes and proactively roll out new VMIs", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 1)
		waitForVMIs(newPool.Namespace, 1)

		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})

		Expect(err).To(BeNil())
		Expect(len(vms.Items)).To(Equal(1))

		name := vms.Items[0].Name
		vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(name, &metav1.GetOptions{})
		Expect(err).To(BeNil())

		vmiUID := vmi.UID

		By("Rolling Out VM template change")
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, v12.GetOptions{})
		Expect(err).To(BeNil())

		// Make a VMI template change
		newPool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels["newlabel"] = "newvalue"
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Update(context.Background(), newPool, metav1.UpdateOptions{})
		Expect(err).To(BeNil())

		By("Ensuring VM picks up label")
		Eventually(func() error {

			vm, err := virtClient.VirtualMachine(newPool.Namespace).Get(name, &metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, ok := vm.Spec.Template.ObjectMeta.Labels["newlabel"]
			if !ok {
				return fmt.Errorf("Expected vm pool update to roll out to VMs")
			}

			return nil
		}, 30*time.Second, 1*time.Second).Should(BeNil())

		By("Ensuring VMI is re-created to pick up new label")
		Eventually(func() error {
			vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(name, &metav1.GetOptions{})
			if err != nil {
				return nil
			}

			if vmi.UID == vmiUID {
				return fmt.Errorf("Waiting on VMI to get deleted and recreated")
			}
			_, ok := vmi.ObjectMeta.Labels["newlabel"]
			if !ok {
				return fmt.Errorf("Expected vmi to pick up the new updated label")
			}
			return nil
		}, 60*time.Second, 1*time.Second).Should(BeNil())
	})

	It("should remove owner references on the VirtualMachine if it is orphan deleted", func() {
		newPool := newOfflineVirtualMachinePool()
		doScale(newPool.ObjectMeta.Name, 2)

		// Check for owner reference
		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
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
		vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(&v12.ListOptions{})
		Expect(vms.Items).To(HaveLen(2))

		By("Checking a VirtualMachine owner references")
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).To(BeEmpty())
		}
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not scale when paused and scale when resume", func() {
		pool := newOfflineVirtualMachinePool()
		// pause controller
		By("Pausing the pool")
		_, err := virtClient.VirtualMachinePool(pool.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, []byte("[{ \"op\": \"add\", \"path\": \"/spec/paused\", \"value\": true }]"), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() poolv1.VirtualMachinePoolConditionType {
			pool, err = virtClient.VirtualMachinePool(util.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(pool.Status.Conditions) > 0 {
				return pool.Status.Conditions[0].Type
			}
			return ""
		}, 10*time.Second, 1*time.Second).Should(Equal(poolv1.VirtualMachinePoolReplicaPaused))

		// set new replica count while still being paused
		By("Updating the number of replicas")
		pool.Spec.Replicas = tests.NewInt32(1)
		_, err = virtClient.VirtualMachinePool(pool.ObjectMeta.Namespace).Update(context.Background(), pool, metav1.UpdateOptions{})
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
