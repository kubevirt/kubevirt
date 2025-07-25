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

package tests_test

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1beta1"
	"kubevirt.io/client-go/kubecli"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
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

	waitForVMIs := func(namespace string, labelSelector *metav1.LabelSelector, expectedCount int) {
		Eventually(func() error {
			vmis, err := virtClient.VirtualMachineInstance(namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(labelSelector),
			})
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
		}, 120*time.Second, 1*time.Second).Should(Succeed())

	}

	scalePool := func(name string, scale int32) *poolv1.VirtualMachinePool {
		patchSet := patch.New()
		patchSet.AddOption(patch.WithReplace("/spec/replicas", pointer.P(scale)))
		patchData, err := patchSet.GeneratePayload()
		Expect(err).ToNot(HaveOccurred())

		pool, err := virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Patch(context.Background(), name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pool
	}

	doScale := func(name string, scale int32) {
		By(fmt.Sprintf("Scaling to %d", scale))
		pool := scalePool(name, scale)

		runStrategy := pool.Spec.VirtualMachineTemplate.Spec.RunStrategy
		running := runStrategy != nil && *runStrategy == v1.RunStrategyAlways
		By("Checking the number of replicas")
		Eventually(func() int32 {
			pool, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Get(context.Background(), name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if running {
				return pool.Status.ReadyReplicas
			}
			return pool.Status.Replicas
		}, 90*time.Second, time.Second).Should(Equal(int32(scale)))

		vms, err := virtClient.VirtualMachine(testsuite.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(pool.Spec.Selector),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(filterNotDeletedVMsOwnedByPool(pool.Name, vms)).To(HaveLen(int(scale)))
	}
	createVirtualMachinePool := func(pool *poolv1.VirtualMachinePool) *poolv1.VirtualMachinePool {
		pool, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Create(context.Background(), pool, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pool
	}

	newPersistentStorageVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool with persistent storage")

		sc, exists := libstorage.GetRWOFileSystemStorageClass()
		if !exists {
			Fail("Filesystem storage (RWO) is not present")
		}

		dataVolume := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
			libdv.WithStorage(libdv.StorageWithStorageClass(sc)),
		)

		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithDataVolume("disk0", dataVolume.Name),
				libvmi.WithMemoryRequest("100M"),
			),
			libvmi.WithDataVolumeTemplate(dataVolume),
		)

		newPool := newPoolFromVMI(&v1.VirtualMachineInstance{
			ObjectMeta: vm.Spec.Template.ObjectMeta,
			Spec:       vm.Spec.Template.Spec,
		})
		newPool.Spec.VirtualMachineTemplate.Spec.DataVolumeTemplates = vm.Spec.DataVolumeTemplates
		newPool.Spec.VirtualMachineTemplate.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)

		return newPool
	}

	newVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool")
		pool := newPoolFromVMI(libvmi.New(libvmi.WithMemoryRequest("2Mi")))
		pool.Spec.VirtualMachineTemplate.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
		return createVirtualMachinePool(pool)
	}

	newOfflineVirtualMachinePool := func() *poolv1.VirtualMachinePool {
		By("Create a new VirtualMachinePool")
		return createVirtualMachinePool(newPoolFromVMI(libvmifact.NewAlpine()))
	}

	DescribeTable("pool should scale", func(startScale int, stopScale int) {
		newPool := newVirtualMachinePool()
		doScale(newPool.Name, int32(startScale))
		doScale(newPool.Name, int32(stopScale))
		doScale(newPool.Name, int32(0))
	},
		Entry("to three, to two and then to zero replicas", 3, 2),
		Entry("to five, to six and then to zero replicas", 5, 6),
	)

	It("should be rejected on POST if spec is invalid", func() {
		newPool := newOfflineVirtualMachinePool()
		newPool.TypeMeta = metav1.TypeMeta{
			APIVersion: poolv1.SchemeGroupVersion.String(),
			Kind:       poolv1.VirtualMachinePoolKind,
		}

		newPool.Spec.VirtualMachineTemplate.Spec.RunStrategy = nil
		newPool.Spec.VirtualMachineTemplate.ObjectMeta.Labels = map[string]string{}
		_, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err.Error()).To(ContainSubstring("selector does not match labels"))
	})

	It("should reject POST if vmi spec is invalid", func() {
		newPool := newOfflineVirtualMachinePool()
		newPool.TypeMeta = metav1.TypeMeta{
			APIVersion: poolv1.SchemeGroupVersion.String(),
			Kind:       poolv1.VirtualMachinePoolKind,
		}

		// Add a disk that doesn't map to a volume.
		// This should get rejected which tells us the webhook validator is working.
		newPool.Spec.VirtualMachineTemplate.Spec.Template.Spec.Domain.Devices.Disks = append(newPool.Spec.VirtualMachineTemplate.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name: "testdisk",
		})

		_, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Create(context.Background(), newPool, metav1.CreateOptions{})
		Expect(err.Error()).To(ContainSubstring("admission webhook \"virtualmachinepool-validator.kubevirt.io\" denied the request: spec.virtualMachineTemplate.spec.template.spec.domain.devices.disks[1].Name 'testdisk' not found"))
	})

	It("should remove VMs once they are marked for deletion", func() {
		newPool := newVirtualMachinePool()
		// Create a pool with two replicas
		doScale(newPool.Name, 2)
		// Delete it
		By("Deleting the VirtualMachinePool")
		Expect(virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Delete(context.Background(), newPool.ObjectMeta.Name, metav1.DeleteOptions{})).To(Succeed())
		// Wait until VMs are gone
		By("Waiting until all VMs are gone")
		Eventually(func() int {
			vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 120*time.Second, 1*time.Second).Should(BeZero())
	})

	It("should handle pool with dataVolumeTemplates", func() {
		pool := newPersistentStorageVirtualMachinePool()
		newPool := createVirtualMachinePool(pool)
		doScale(newPool.ObjectMeta.Name, 2)

		var (
			err       error
			vms       *v1.VirtualMachineList
			dvs       *cdiv1.DataVolumeList
			dvOrigUID types.UID
		)

		By("Waiting until all VMs are created")
		Eventually(func() int {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 60*time.Second, 1*time.Second).Should(Equal(2))

		By("Waiting until all VMIs are created and online")
		waitForVMIs(newPool.Namespace, newPool.Spec.Selector, 2)

		// Select a VM to delete, and record the VM and DV/PVC UIDs associated with the VM.
		origUID := vms.Items[0].UID
		name := vms.Items[0].Name
		dvName := vms.Items[0].Spec.DataVolumeTemplates[0].ObjectMeta.Name
		dvName1 := vms.Items[1].Spec.DataVolumeTemplates[0].ObjectMeta.Name
		Expect(dvName).ToNot(Equal(dvName1))

		By("Ensure DataVolumes are created")
		dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(newPool.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(labelSelectorFromVMs(vms)),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(dvs.Items).To(HaveLen(2))
		for _, dv := range dvs.Items {
			if dv.Name == dvName {
				dvOrigUID = dv.UID
			}
		}

		Expect(string(dvOrigUID)).ToNot(Equal(""))

		By("deleting a VM")
		foreGround := metav1.DeletePropagationForeground
		Expect(virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).Delete(context.Background(), name, metav1.DeleteOptions{PropagationPolicy: &foreGround})).To(Succeed())

		By("Waiting for deleted VM to be replaced")
		Eventually(func() error {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
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

		}, 120*time.Second, 1*time.Second).Should(Succeed())

		By("Waiting until all VMIs are created and online again")
		waitForVMIs(newPool.Namespace, newPool.Spec.Selector, 2)

		By("Verify datavolume count after VM replacement")
		dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(newPool.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(labelSelectorFromVMs(vms)),
		})
		Expect(dvs.Items).To(HaveLen(2))

		By("Verify datavolume for deleted VM is replaced")
		for _, dv := range dvs.Items {
			Expect(dv.UID).ToNot(Equal(dvOrigUID))
		}
	})

	It("should replace deleted VM and get replacement", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.Name, 3)

		var err error
		var vms *v1.VirtualMachineList

		By("Waiting until all VMs are created")
		Eventually(func() int {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			return len(vms.Items)
		}, 120*time.Second, 1*time.Second).Should(Equal(3))

		origUID := vms.Items[1].UID
		name := vms.Items[1].Name

		By("deleting a VM")
		Expect(virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).Delete(context.Background(), name, metav1.DeleteOptions{})).To(Succeed())

		By("Waiting for deleted VM to be replaced")
		Eventually(func() error {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
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

		}, 120*time.Second, 1*time.Second).Should(Succeed())

	})

	It("should roll out VM template changes without impacting VMI", func() {
		newPool := newVirtualMachinePool()
		doScale(newPool.Name, 1)
		waitForVMIs(newPool.Namespace, newPool.Spec.Selector, 1)

		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(newPool.Spec.Selector),
		})

		Expect(err).ToNot(HaveOccurred())
		Expect(vms.Items).To(HaveLen(1))

		name := vms.Items[0].Name
		vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		vmiUID := vmi.UID

		By("Rolling Out VM template change")
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		patchData, err := patch.New(patch.WithAdd(
			fmt.Sprintf("/spec/virtualMachineTemplate/metadata/labels/%s", newLabelKey), newLabelValue),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		newPool, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Patch(context.Background(), newPool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring VM picks up label")
		Eventually(func() error {

			vm, err := virtClient.VirtualMachine(newPool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				return err
			}

			_, ok := vm.Labels[newLabelKey]
			if !ok {
				return fmt.Errorf("Expected vm pool update to roll out to VMs")
			}

			return nil
		}, 30*time.Second, 1*time.Second).Should(Succeed())

		By("Ensuring VMI remains consistent and isn't restarted")
		Consistently(func() error {
			vmi, err := virtClient.VirtualMachineInstance(newPool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			if err != nil {
				return nil
			}

			Expect(vmi.UID).To(Equal(vmiUID))
			Expect(vmi.DeletionTimestamp).To(BeNil())
			return nil
		}, 5*time.Second, 1*time.Second).Should(Succeed())
	})

	type updateStrategyTest struct {
		description        string
		updateStrategy     *poolv1.VirtualMachinePoolUpdateStrategy
		initialVMLabels    map[string]string
		expectedVMUpdate   bool
		expectedVmiRestart bool
	}

	DescribeTable("should handle VMI template changes based on update strategy",
		func(test updateStrategyTest) {
			pool := newPoolFromVMI(libvmi.New(libvmi.WithMemoryRequest("2Mi")))
			pool.Spec.VirtualMachineTemplate.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
			pool.Spec.UpdateStrategy = test.updateStrategy

			if test.initialVMLabels != nil {
				if pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels == nil {
					pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels = make(map[string]string)
				}
				for k, v := range test.initialVMLabels {
					pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels[k] = v
				}
			}

			pool = createVirtualMachinePool(pool)
			doScale(pool.Name, 1)
			waitForVMIs(pool.Namespace, pool.Spec.Selector, 1)

			vmis, err := virtClient.VirtualMachineInstance(pool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(pool.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmis.Items).To(HaveLen(1))

			name := vmis.Items[0].Name
			vmiUID := vmis.Items[0].UID
			vmiGeneration := vmis.Items[0].Generation

			By("Rolling Out VMI template change")
			pool, err = virtClient.VirtualMachinePool(pool.ObjectMeta.Namespace).Get(context.Background(), pool.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			patchData, err := patch.New(patch.WithAdd(
				fmt.Sprintf("/spec/virtualMachineTemplate/spec/template/metadata/labels/%s", newLabelKey), newLabelValue),
			).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			pool, err = virtClient.VirtualMachinePool(pool.ObjectMeta.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking VM update behavior")
			if test.expectedVMUpdate {
				Eventually(func(g Gomega) {
					vm, err := virtClient.VirtualMachine(pool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(vm.Spec.Template.ObjectMeta.Labels[newLabelKey]).To(Equal(newLabelValue))
				}, 30*time.Second, 1*time.Second).Should(Succeed())
			} else {
				Consistently(func(g Gomega) {
					vm, err := virtClient.VirtualMachine(pool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(vm.Spec.Template.ObjectMeta.Labels[newLabelKey]).ToNot(Equal(newLabelValue))
				}, 30*time.Second, 2*time.Second).Should(Succeed())
			}

			By("Checking VMI update behavior")
			if test.expectedVmiRestart {
				Eventually(func(g Gomega) {
					vmi, err := virtClient.VirtualMachineInstance(pool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(vmi.UID).NotTo(Equal(vmiUID))
					g.Expect(vmi.Labels[newLabelKey]).To(Equal(newLabelValue))
				}, 60*time.Second, 2*time.Second).Should(Succeed())
			} else {
				Consistently(func(g Gomega) {
					vmi, err := virtClient.VirtualMachineInstance(pool.Namespace).Get(context.Background(), name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(vmi.UID).To(Equal(vmiUID))
					g.Expect(vmi.Generation).To(Equal(vmiGeneration))
					g.Expect(vmi.Labels[newLabelKey]).ToNot(Equal(newLabelValue))
				}, 30*time.Second, 2*time.Second).Should(Succeed())
			}
		},
		Entry("should roll out VMI template changes only to VMs when opportunistic update strategy is set",
			updateStrategyTest{
				updateStrategy: &poolv1.VirtualMachinePoolUpdateStrategy{
					Opportunistic: &poolv1.VirtualMachineOpportunisticUpdateStrategy{},
				},
				expectedVMUpdate:   true,
				expectedVmiRestart: false,
			},
		),
		Entry("should roll out VMI template changes and proactively roll out new VMIs when proactive update strategy is nil",
			updateStrategyTest{
				updateStrategy:     nil,
				expectedVMUpdate:   true,
				expectedVmiRestart: true,
			},
		),
		Entry("should not roll out VMI template changes and proactively roll out new VMIs when proactive update strategy is set and label selector does not match",
			updateStrategyTest{
				updateStrategy: &poolv1.VirtualMachinePoolUpdateStrategy{
					Proactive: &poolv1.VirtualMachinePoolProactiveUpdateStrategy{
						SelectionPolicy: &poolv1.VirtualMachinePoolSelectionPolicy{
							Selectors: &poolv1.VirtualMachinePoolSelectors{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"app": "test"},
								},
							},
						},
					},
				},
				expectedVMUpdate:   false,
				expectedVmiRestart: false,
			},
		),
		Entry("should roll out VMI template changes and proactively roll out new VMIs when proactive update strategy is set and label selector matches",
			updateStrategyTest{
				updateStrategy: &poolv1.VirtualMachinePoolUpdateStrategy{
					Proactive: &poolv1.VirtualMachinePoolProactiveUpdateStrategy{
						SelectionPolicy: &poolv1.VirtualMachinePoolSelectionPolicy{
							Selectors: &poolv1.VirtualMachinePoolSelectors{
								LabelSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{"app": "test"},
								},
							},
						},
					},
				},
				initialVMLabels:    map[string]string{"app": "test"},
				expectedVMUpdate:   true,
				expectedVmiRestart: true,
			},
		),
	)

	It("should auto-heal consistently failing VMs when autohealing is enabled", func() {
		By("Creating a pool with autohealing enabled and VMs that will crash loop")
		pool := newPoolFromVMI(libvmi.New(libvmi.WithMemoryRequest("2Mi")))
		pool.Spec.VirtualMachineTemplate.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
		pool.Spec.Autohealing = &poolv1.VirtualMachinePoolAutohealingStrategy{}

		// Add the fail-fast annotation to the VMI template so VMIs will crash loop
		if pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Annotations == nil {
			pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
		}
		pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Annotations[v1.FuncTestLauncherFailFastAnnotation] = ""

		pool = createVirtualMachinePool(pool)

		By("Scaling pool to 1 replica")
		scalePool(pool.Name, 1)

		By("Waiting for VM to be created")
		var name string
		Eventually(func(g Gomega) {
			vms, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(pool.Spec.Selector),
			})
			g.Expect(err).ToNot(HaveOccurred())
			activeVMs := filterNotDeletedVMsOwnedByPool(pool.Name, vms)
			g.Expect(activeVMs).To(HaveLen(1))
			name = activeVMs[0].Name
		}, 60*time.Second, 1*time.Second).Should(Succeed())

		By("Waiting for VM to accumulate consecutive start failures (>= 3)")
		var vmUID types.UID
		Eventually(func(g Gomega) {
			vm, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			vmUID = vm.UID

			g.Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusCrashLoopBackOff))
			g.Expect(vm.Status.StartFailure).ToNot(BeNil())
			g.Expect(vm.Status.StartFailure.ConsecutiveFailCount).To(BeNumerically(">=", 3))
		}, 5*time.Minute, 1*time.Second).Should(Succeed())

		By("Checking that autohealing deleted and replaced the failing VMs")
		Eventually(func(g Gomega) {
			vm, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			if vm.Status.StartFailure != nil {
				g.Expect(vm.Status.StartFailure.ConsecutiveFailCount).To(BeNumerically("<", 3))
			}

			g.Expect(vm.UID).ToNot(Equal(vmUID))
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		By("Checking the startFailure status of the VM after replacement as the annotation is still present")
		Eventually(func(g Gomega) {
			vm, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vm.Status.StartFailure).NotTo(BeNil())
		}, 30*time.Second, 5*time.Second).Should(Succeed())

		By("Removing the annotation that causes the VM to crash loop back off")
		patchData, err := patch.New(patch.WithRemove(fmt.Sprintf("/spec/virtualMachineTemplate/spec/template/metadata/annotations/%s", patch.EscapeJSONPointer(v1.FuncTestLauncherFailFastAnnotation)))).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		pool, err = virtClient.VirtualMachinePool(pool.ObjectMeta.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Checking the startFailure status of the VM to be nil and the VM to be running")
		Eventually(func(g Gomega) {
			vm, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vm.Status.StartFailure).To(BeNil())
			g.Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusRunning))
		}, 30*time.Second, 5*time.Second).Should(Succeed())
	})

	It("should auto-heal VMs when VM is in not ready state for too long when pvc is not found", func() {
		pool := newPoolFromVMI(libvmi.New(libvmi.WithMemoryRequest("2Mi")))
		pool.Spec.VirtualMachineTemplate.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
		pool.Spec.VirtualMachineTemplate.Spec.Template.Spec.Volumes = []v1.Volume{
			{
				Name: "non-existent-pvc",
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "non-existent-pvc",
						},
					},
				},
			},
		}

		pool.Spec.Autohealing = &poolv1.VirtualMachinePoolAutohealingStrategy{
			MinFailingToStartDuration: &metav1.Duration{
				Duration: 2 * time.Minute,
			},
		}
		pool = createVirtualMachinePool(pool)

		By("Scaling pool to 1 replica")
		scalePool(pool.Name, 1)

		By("Waiting for VM to be created")
		var name string
		Eventually(func(g Gomega) {
			vms, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(pool.Spec.Selector),
			})
			g.Expect(err).ToNot(HaveOccurred())
			activeVMs := filterNotDeletedVMsOwnedByPool(pool.Name, vms)
			g.Expect(activeVMs).To(HaveLen(1))
			name = activeVMs[0].Name
		}, 60*time.Second, 1*time.Second).Should(Succeed())

		By("Checking status of the VM to be PvcNotFound")
		var vm *v1.VirtualMachine
		Eventually(func(g Gomega) {
			vm, err = virtClient.VirtualMachine(pool.ObjectMeta.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vm.Status.PrintableStatus).To(Equal(v1.VirtualMachineStatusPvcNotFound))
		}, 30*time.Second, 5*time.Second).Should(Succeed())

		vmUID := vm.UID

		By("checking that the VM is deleted and replaced after the threshold(i.e. 5 minutes)")
		Eventually(func(g Gomega) {
			vm, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).Get(context.Background(), name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vm.UID).ToNot(Equal(vmUID))
		}, 3*time.Minute, 30*time.Second).Should(Succeed())
	})

	It("should remove owner references on the VirtualMachine if it is orphan deleted", func() {
		newPool := newOfflineVirtualMachinePool()
		doScale(newPool.Name, 2)

		// Check for owner reference
		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(newPool.Spec.Selector),
		})
		Expect(vms.Items).To(HaveLen(2))
		Expect(err).ToNot(HaveOccurred())
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).ToNot(BeEmpty())
		}

		// Delete it
		By("Deleting the VirtualMachine pool with the 'orphan' deletion strategy")
		orphanPolicy := metav1.DeletePropagationOrphan
		Expect(virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Delete(context.Background(), newPool.ObjectMeta.Name, metav1.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
		// Wait until the pool is deleted
		By("Waiting until the pool got deleted")
		Eventually(func() error {
			_, err := virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Get(context.Background(), newPool.ObjectMeta.Name, metav1.GetOptions{})
			return err
		}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Checking if two VMs are orphaned and still exist")
		vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(newPool.Spec.Selector),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(vms.Items).To(HaveLen(2))

		By("Checking a VirtualMachine owner references")
		for _, vm := range vms.Items {
			Expect(vm.OwnerReferences).To(BeEmpty())
		}

		By("Checking VirtualMachines are not blocked by pool finalizer")
		err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).DeleteCollection(context.Background(), metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelectorToString(newPool.Spec.Selector),
		})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() ([]v1.VirtualMachine, error) {
			vms, err = virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
			return vms.Items, err
		}, 60*time.Second, 1*time.Second).Should(BeEmpty())
	})

	It("should not scale when paused and scale when resume", func() {
		pool := newOfflineVirtualMachinePool()
		// pause controller
		By("Pausing the pool")
		_, err := virtClient.VirtualMachinePool(pool.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, []byte("[{ \"op\": \"add\", \"path\": \"/spec/paused\", \"value\": true }]"), metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() *poolv1.VirtualMachinePool {
			pool, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pool
		}, 10*time.Second, 1*time.Second).Should(matcher.HaveConditionTrue(poolv1.VirtualMachinePoolReplicaPaused))

		// set new replica count while still being paused
		By("Updating the number of replicas")
		patchData, err := patch.GenerateTestReplacePatch("/spec/replicas", pool.Spec.Replicas, pointer.P(1))
		Expect(err).ToNot(HaveOccurred())
		pool, err = virtClient.VirtualMachinePool(pool.ObjectMeta.Namespace).Patch(context.Background(), pool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		// make sure that we don't scale up
		By("Checking that the pool do not scale while it is paused")
		Consistently(func() int32 {
			pool, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, metav1.GetOptions{})
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
			pool, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(pool.Status.Conditions)
		}, 10*time.Second, 1*time.Second).Should(Equal(0))

		// Replicas should be created
		By("Checking that the missing replicas are now created")
		Eventually(func() int32 {
			pool, err = virtClient.VirtualMachinePool(testsuite.NamespaceTestDefault).Get(context.Background(), pool.ObjectMeta.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pool.Status.Replicas
		}, 10*time.Second, 1*time.Second).Should(Equal(int32(1)))
	})

	It("should use DescendingOrder scale-in strategy when specified", func() {
		By("Create a new VirtualMachinePool with DescendingOrder scale-in policy")
		pool := newPoolFromVMI(libvmifact.NewAlpine())

		// Set up DescendingOrder scale-in strategy
		pool.Spec.ScaleInStrategy = &poolv1.VirtualMachinePoolScaleInStrategy{
			Proactive: &poolv1.VirtualMachinePoolProactiveScaleInStrategy{
				SelectionPolicy: &poolv1.VirtualMachinePoolSelectionPolicy{
					SortPolicy: pointer.P(poolv1.VirtualMachinePoolSortPolicyDescendingOrder),
				},
			},
		}
		pool.Spec.VirtualMachineTemplate.Spec.RunStrategy = pointer.P(v1.RunStrategyAlways)
		pool = createVirtualMachinePool(pool)

		By("Scaling pool to 5 replicas")
		doScale(pool.ObjectMeta.Name, 5)

		By("Waiting until all VMs are created and running")
		waitForVMIs(pool.Namespace, pool.Spec.Selector, 5)

		By("Getting all VMs to verify their names/ordinals")
		vms, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(filterNotDeletedVMsOwnedByPool(pool.Name, vms)).To(HaveLen(5))

		// Store original VM names sorted by ordinal for comparison
		var vmNames []string
		for _, vm := range vms.Items {
			if vm.DeletionTimestamp == nil {
				for _, ref := range vm.OwnerReferences {
					if ref.Name == pool.Name {
						vmNames = append(vmNames, vm.Name)
						break
					}
				}
			}
		}

		// Sort by ordinal to know which VMs should be deleted first (highest ordinals)
		sort.Slice(vmNames, func(i, j int) bool {
			// Extract ordinal from VM name (assuming format like "poolXXXXX-0", "poolXXXXX-1", etc.)
			// This assumes indexFromName function extracts the ordinal correctly
			ordinalI := extractOrdinalFromName(vmNames[i])
			ordinalJ := extractOrdinalFromName(vmNames[j])
			return ordinalI > ordinalJ // Sort descending (highest first)
		})

		By("Scaling down to 2 replicas (should remove 3 VMs with highest ordinals)")
		doScale(pool.ObjectMeta.Name, 2)

		By("Waiting for scale down to complete")
		Eventually(func() int {
			vms, err := virtClient.VirtualMachine(pool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			return len(filterNotDeletedVMsOwnedByPool(pool.Name, vms))
		}, 120*time.Second, 1*time.Second).Should(Equal(2))

		By("Verifying that VMs with highest ordinals were deleted")
		vms, err = virtClient.VirtualMachine(pool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		remainingVMs := filterNotDeletedVMsOwnedByPool(pool.Name, vms)
		Expect(remainingVMs).To(HaveLen(2))

		// The remaining VMs should be the ones with the lowest ordinals
		expectedRemainingVMs := vmNames[3:] // Last 2 VMs (lowest ordinals)
		var actualRemainingVMNames []string
		for _, vm := range remainingVMs {
			actualRemainingVMNames = append(actualRemainingVMNames, vm.Name)
		}

		// Sort both slices for comparison
		sort.Strings(expectedRemainingVMs)
		sort.Strings(actualRemainingVMNames)

		Expect(actualRemainingVMNames).To(Equal(expectedRemainingVMs))

		By("Verifying VMIs are still running for remaining VMs")
		waitForVMIs(pool.Namespace, pool.Spec.Selector, 2)
	})

	It("should scale in VMs based on the scale-in strategy and preserve state is set to offline", func() {
		pool := newPersistentStorageVirtualMachinePool()
		pool.Spec.VirtualMachineTemplate.Spec.Template.ObjectMeta.Labels["app"] = "test"
		pool.Spec.ScaleInStrategy = &poolv1.VirtualMachinePoolScaleInStrategy{
			Proactive: &poolv1.VirtualMachinePoolProactiveScaleInStrategy{
				StatePreservation: pointer.P(poolv1.StatePreservationOffline),
				SelectionPolicy: &poolv1.VirtualMachinePoolSelectionPolicy{
					SortPolicy: pointer.P(poolv1.VirtualMachinePoolSortPolicyAscendingOrder),
				},
			},
		}
		pool = createVirtualMachinePool(pool)

		By("Scaling pool to 3 replicas")
		doScale(pool.Name, 3)

		By("Waiting until all VMs are created and running")
		waitForVMIs(pool.Namespace, pool.Spec.Selector, 3)

		vms, err := virtClient.VirtualMachine(pool.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(pool.Spec.Selector),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(vms.Items).To(HaveLen(3))

		By("Verify that the DataVolumes are created and owned by the VMs")
		dvs, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(pool.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(labelSelectorFromVMs(vms)),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(dvs.Items).To(HaveLen(3))

		Eventually(func() error {
			for _, dv := range dvs.Items {
				fmt.Println("dv", dv.Name, dv.OwnerReferences)
				if dv.OwnerReferences == nil || len(dv.OwnerReferences) == 0 {
					return fmt.Errorf("DataVolume %s has no owner references", dv.Name)
				}
				Expect(dv.OwnerReferences).To(HaveLen(1))
				Expect(dv.OwnerReferences[0].Name).To(ContainSubstring(pool.Name))
			}

			return nil
		}, 120*time.Second, 1*time.Second).Should(Succeed())

		By("Scaling pool to 2 replicas")
		doScale(pool.Name, 2)

		By("Waiting until the pool is scaled down to 2 replicas")
		Eventually(func() int32 {
			pool, err = virtClient.VirtualMachinePool(pool.Namespace).Get(context.Background(), pool.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return pool.Status.Replicas
		}, 120*time.Second, 1*time.Second).Should(Equal(int32(2)))

		By("Verify that the DataVolume of the deleted VM is not owned by the pool but is present in the namespace")
		dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(pool.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(labelSelectorFromVMs(vms)),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(dvs.Items).To(HaveLen(3))
		var dvUID string
		for i, dv := range dvs.Items {
			if i == 0 {
				Expect(dv.OwnerReferences).To(BeEmpty())
				dvUID = string(dv.UID)
				continue
			}
			Expect(dv.OwnerReferences).To(HaveLen(1))
			Expect(dv.OwnerReferences[0].Name).To(ContainSubstring(pool.Name))
		}

		By("Scale up the pool to 3 replicas and verify that the DataVolume which was orphaned is re-adopted by the pool")
		doScale(pool.Name, 3)
		waitForVMIs(pool.Namespace, pool.Spec.Selector, 3)

		dvs, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(pool.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(labelSelectorFromVMs(vms)),
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(dvs.Items).To(HaveLen(3))
		for i, dv := range dvs.Items {
			if i == 0 {
				Expect(dv.UID).To(Equal(types.UID(dvUID)))
			}
			Expect(dv.OwnerReferences).To(HaveLen(1))
			Expect(dv.OwnerReferences[0].Name).To(ContainSubstring(pool.Name))
		}
	})

	DescribeTable("should respect name generation settings", func(appendIndex *bool) {
		const (
			cmName     = "configmap"
			secretName = "secret"
		)

		newPool := newPoolFromVMI(libvmi.New(
			libvmi.WithConfigMapDisk(cmName, cmName),
			libvmi.WithSecretDisk(secretName, secretName),
		))
		newPool.Spec.NameGeneration = &poolv1.VirtualMachinePoolNameGeneration{
			AppendIndexToConfigMapRefs: appendIndex,
			AppendIndexToSecretRefs:    appendIndex,
		}
		newPool = createVirtualMachinePool(newPool)
		doScale(newPool.Name, 2)

		vms, err := virtClient.VirtualMachine(newPool.ObjectMeta.Namespace).List(context.Background(), metav1.ListOptions{
			LabelSelector: labelSelectorToString(newPool.Spec.Selector),
		})
		Expect(vms.Items).To(HaveLen(2))
		Expect(err).ToNot(HaveOccurred())
		for _, vm := range vms.Items {
			Expect(vm.Spec.Template.Spec.Volumes).To(HaveLen(2))
			Expect(vm.Spec.Template.Spec.Volumes[0].Name).To(Equal(cmName))
			Expect(vm.Spec.Template.Spec.Volumes[1].Name).To(Equal(secretName))
			if appendIndex != nil && *appendIndex {
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).To(MatchRegexp(fmt.Sprintf("%s-\\d", cmName)))
				Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName).To(MatchRegexp(fmt.Sprintf("%s-\\d", secretName)))
			} else {
				Expect(vm.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.Name).To(Equal(cmName))
				Expect(vm.Spec.Template.Spec.Volumes[1].VolumeSource.Secret.SecretName).To(Equal(secretName))
			}
		}
	},
		Entry("do not append index by default", nil),
		Entry("do not append index if set to false", pointer.P(false)),
		Entry("append index if set to true", pointer.P(true)),
	)

	It("should respect maxUnavailable strategy during updates", func() {
		newPool := newVirtualMachinePool()

		replicas := 4
		maxUnavailable := 1

		doScale(newPool.Name, int32(replicas))
		waitForVMIs(newPool.Namespace, newPool.Spec.Selector, replicas)

		By(fmt.Sprintf("Setting maxUnavailable to %d", maxUnavailable))
		patchData, err := patch.New(patch.WithReplace("/spec/maxUnavailable", intstr.FromInt(maxUnavailable))).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Patch(context.Background(), newPool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Making a VMI template change")
		patchData, err = patch.New(patch.WithAdd(
			fmt.Sprintf("/spec/virtualMachineTemplate/spec/template/metadata/labels/%s", newLabelKey), newLabelValue),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.VirtualMachinePool(newPool.ObjectMeta.Namespace).Patch(context.Background(), newPool.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying maxUnavailable constraint throughout the update process")
		Consistently(func() error {
			vmis, err := virtClient.VirtualMachineInstance(newPool.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelectorToString(newPool.Spec.Selector),
			})
			Expect(err).ToNot(HaveOccurred())

			unavailableCount := 0

			for _, vmi := range vmis.Items {
				if vmi.DeletionTimestamp != nil || vmi.Status.Phase != v1.Running {
					unavailableCount++
				}
			}

			if unavailableCount > maxUnavailable {
				return fmt.Errorf("maxUnavailable constraint violated: %d unavailable VMIs (max allowed: %d)",
					unavailableCount, maxUnavailable)
			}
			return nil
		}, 120*time.Second, 1*time.Second).Should(Succeed(), "maxUnavailable constraint was violated during the update process")

		By("Waiting for all VMIs to be updated")
		updatedSelector := newPool.Spec.Selector.DeepCopy()
		if updatedSelector.MatchLabels == nil {
			updatedSelector.MatchLabels = make(map[string]string)
		}
		updatedSelector.MatchLabels[newLabelKey] = newLabelValue
		labelSelector := labelSelectorToString(updatedSelector)
		Eventually(func(g Gomega) {
			vmis, err := virtClient.VirtualMachineInstance(newPool.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(vmis.Items).To(HaveLen(replicas))
		}, 120*time.Second, 1*time.Second).Should(Succeed(), "VMIs not all updated")
	})
})

func newPoolFromVMI(vmi *v1.VirtualMachineInstance) *poolv1.VirtualMachinePool {
	selector := "pool" + rand.String(5)
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
					RunStrategy: pointer.P(v1.RunStrategyManual),
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

func filterNotDeletedVMsOwnedByPool(poolName string, vms *v1.VirtualMachineList) []v1.VirtualMachine {
	var result []v1.VirtualMachine
	for _, vm := range vms.Items {
		if vm.DeletionTimestamp != nil {
			continue
		}
		for _, ref := range vm.OwnerReferences {
			if ref.Name == poolName {
				result = append(result, vm)
			}
		}
	}
	return result
}

func labelSelectorToString(labelSelector *metav1.LabelSelector) string {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	Expect(err).ToNot(HaveOccurred())
	return selector.String()
}

func labelSelectorFromVMs(vms *v1.VirtualMachineList) *metav1.LabelSelector {
	var values []string
	for _, vm := range vms.Items {
		values = append(values, string(vm.UID))
	}
	return &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      v1.CreatedByLabel,
				Operator: metav1.LabelSelectorOpIn,
				Values:   values,
			},
		},
	}
}

// Helper function to extract ordinal from VM name
// This assumes VM names follow a pattern where the ordinal is at the end
func extractOrdinalFromName(name string) int {
	parts := strings.Split(name, "-")
	if len(parts) == 0 {
		return 0
	}

	ordinalStr := parts[len(parts)-1]
	ordinal, err := strconv.Atoi(ordinalStr)
	if err != nil {
		return 0
	}
	return ordinal
}
