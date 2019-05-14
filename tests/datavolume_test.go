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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

const InvalidDataVolumeUrl = "http://127.0.0.1/invalid"

var _ = Describe("DataVolume Integration", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		if !tests.HasCDI() {
			Skip("Skip DataVolume tests when CDI is not present")
		}

	})

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		By("Checking that the DataVolume has succeeded")
		tests.WaitForSuccessfulDataVolumeImport(vmi, timeout)

		By("Starting a VirtualMachineInstance with DataVolume")
		var obj *v1.VirtualMachineInstance
		var err error
		Eventually(func() error {
			obj, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			return err
		}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

		By("Waiting until the VirtualMachineInstance will start")
		tests.WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
		return obj
	}

	Describe("Starting a VirtualMachineInstance with a DataVolume as a volume source", func() {
		Context("using Alpine import", func() {
			It("should be successfully started and stopped multiple times", func() {

				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.AlpineHttpUrl, tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(dataVolume)
				Expect(err).To(BeNil())

				num := 2
				By("Starting and stopping the VirtualMachineInstance a number of times")
				for i := 1; i <= num; i++ {
					vmi := runVMIAndExpectLaunch(vmi, 240)

					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).To(BeNil())
						expecter.Close()
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(dataVolume.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("Starting a VirtualMachine with an invalid DataVolume", func() {
		Context("using DataVolume with invalid URL", func() {
			It("should correctly handle invalid DataVolumes", func() {
				// Don't actually create the DataVolume since it's invalid.
				dataVolume := tests.NewRandomDataVolumeWithHttpImport(InvalidDataVolumeUrl, tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)

				By("Creating a VM with an invalid DataVolume")
				_, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for VMI to be created")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.GetName(), &metav1.GetOptions{})
					if err != nil {
						Expect(err.Error()).To(ContainSubstring("not found"),
							"A 404 while VMI is being created would be normal. All other errors are unexpected")
						return v1.VmPhaseUnset
					}
					return vmi.Status.Phase

				}, 100*time.Second, 5*time.Second).Should(Equal(v1.Pending), "VMI with invalid DataVolume should not be scheduled")
			})
		})
	})

	Describe("[rfe_id:896][crit:high][vendor:cnv-qe@redhat.com][level:system] with oc/kubectl", func() {
		var vm *v1.VirtualMachine
		var err error
		var workDir string
		var vmJson string
		var dataVolumeName string
		var pvcName string

		k8sClient := tests.GetK8sCmdClient()

		BeforeEach(func() {
			running := true

			vm = tests.NewRandomVMWithDataVolume(tests.AlpineHttpUrl, tests.NamespaceTestDefault)
			vm.Spec.Running = &running

			dataVolumeName = vm.Spec.DataVolumeTemplates[0].Name
			pvcName = dataVolumeName

			workDir, err := ioutil.TempDir("", tests.TempDirPrefix+"-")
			Expect(err).ToNot(HaveOccurred())
			vmJson, err = tests.GenerateVMJson(vm, workDir)
			Expect(err).ToNot(HaveOccurred())
		})

		deleteIfExistsVM := func(name string, namespace string) {
			vm, err := virtClient.VirtualMachine(namespace).Get(name, &metav1.GetOptions{})
			if err == nil && vm.DeletionTimestamp == nil {
				err := virtClient.VirtualMachine(namespace).Delete(name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}

		deleteIfExistsVMI := func(name string, namespace string) {
			vmi, err := virtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
			if err == nil && vmi.DeletionTimestamp == nil {
				err := virtClient.VirtualMachineInstance(namespace).Delete(name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}

		deleteIfExistsDataVolume := func(name string, namespace string) {
			dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(name, metav1.GetOptions{})
			if err == nil && dataVolume.DeletionTimestamp == nil {
				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Delete(name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}
		}

		vmiIsRunningAndOwned := func(name, namespace string) {
			Eventually(func() error {
				vmi, err := virtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
				if err != nil {
					return err
				}
				Expect(vmi.OwnerReferences).ToNot(BeEmpty())

				if !vmi.IsRunning() {
					return fmt.Errorf("Waiting on VMI to enter running phase")
				}
				return nil
			}, 160*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		vmiIsRunningAndNotOwned := func(name, namespace string) {
			Eventually(func() error {
				vmi, err := virtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
				if err != nil {
					return err
				}
				Expect(vmi.OwnerReferences).To(BeEmpty())

				if !vmi.IsRunning() {
					return fmt.Errorf("Waiting on VMI to enter running phase")
				}
				return nil
			}, 160*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		dataVolumeIsSuccessAndOwned := func(name, namespace string) {
			Eventually(func() error {
				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				Expect(dataVolume.OwnerReferences).ToNot(BeEmpty())

				if dataVolume.Status.Phase != cdiv1.Succeeded {
					return fmt.Errorf("Waiting on DataVolume to enter succeeded phase")
				}
				return nil
			}, 160*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		dataVolumeIsSuccessAndNotOwned := func(name, namespace string) {
			Eventually(func() error {
				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				Expect(dataVolume.OwnerReferences).To(BeEmpty())

				if dataVolume.Status.Phase != cdiv1.Succeeded {
					return fmt.Errorf("Waiting on DataVolume to enter succeeded phase")
				}
				return nil
			}, 160*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		pvcExists := func(name, namespace string) {
			Eventually(func() error {
				_, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
				if err != nil {
					return err
				}
				return nil
			}, 160*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		}

		waitForDeletionVM := func(name, namespace string) {
			Eventually(func() bool {
				_, err := virtClient.VirtualMachine(namespace).Get(name, &metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 100*time.Second, 1*time.Second).Should(BeTrue())
		}

		waitForDeletionVMI := func(name, namespace string) {
			Eventually(func() bool {
				_, err := virtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 100*time.Second, 1*time.Second).Should(BeTrue())
		}

		waitForDeletionDataVolume := func(name, namespace string) {
			Eventually(func() bool {
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 100*time.Second, 1*time.Second).Should(BeTrue())
		}

		waitForDeletionPVC := func(name, namespace string) {
			Eventually(func() bool {
				_, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 100*time.Second, 1*time.Second).Should(BeTrue())

		}

		AfterEach(func() {
			if workDir != "" {
				err = os.RemoveAll(workDir)
				Expect(err).ToNot(HaveOccurred())
				workDir = ""
			}

			deleteIfExistsVM(vm.Name, vm.Namespace)
			deleteIfExistsVMI(vm.Name, vm.Namespace)
			deleteIfExistsDataVolume(dataVolumeName, vm.Namespace)
		})

		It("[test_id:836] Creating a VM with DataVolumeTemplates should succeed.", func() {
			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying DataVolume succeeded and is created with VM owner reference")
			dataVolumeIsSuccessAndOwned(dataVolumeName, vm.Namespace)

			By("Verifying PVC is created")
			pvcExists(pvcName, vm.Namespace)

			By("Verifying VMI is created with VM owner reference")
			vmiIsRunningAndOwned(vm.Name, vm.Namespace)

			By("Delete VM")
			_, _, err = tests.RunCommand("kubectl", "delete", "vm", vm.Name)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:837]deleting VM with cascade=true should automatically delete DataVolumes and VMI owned by VM.", func() {
			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying DataVolume succeeded and is created with VM owner reference")
			dataVolumeIsSuccessAndOwned(dataVolumeName, vm.Namespace)

			By("Verifying PVC is created")
			pvcExists(pvcName, vm.Namespace)

			By("Verifying VMI is created with VM owner reference")
			vmiIsRunningAndOwned(vm.Name, vm.Namespace)

			By("Deleting VM with cascade=true")
			_, _, err = tests.RunCommand("kubectl", "delete", "vm", vm.Name, "--cascade=true")
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VM to be deleted")
			waitForDeletionVM(vm.Name, vm.Namespace)

			By("Waiting for the VMI to be deleted")
			waitForDeletionVMI(vm.Name, vm.Namespace)

			By("Waiting for the DataVolume to be deleted")
			waitForDeletionDataVolume(dataVolumeName, vm.Namespace)

			By("Waiting for the PVC to be deleted")
			waitForDeletionPVC(pvcName, vm.Namespace)
		})

		It("[test_id:838]deleting VM with cascade=false should orphan DataVolumes and VMI owned by VM.", func() {

			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying DataVolume succeeded and is created with VM owner reference")
			dataVolumeIsSuccessAndOwned(dataVolumeName, vm.Namespace)

			By("Verifying PVC is created")
			pvcExists(pvcName, vm.Namespace)

			By("Verifying VMI is created with VM owner reference")
			vmiIsRunningAndOwned(vm.Name, vm.Namespace)

			By("Deleting VM with cascade=false")
			_, _, err = tests.RunCommand("kubectl", "delete", "vm", vm.Name, "--cascade=false")
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VM to be deleted")
			waitForDeletionVM(vm.Name, vm.Namespace)

			By("Verifying DataVolume still exists with owner references removed")
			dataVolumeIsSuccessAndNotOwned(dataVolumeName, vm.Namespace)

			By("Verifying VMI still exists with owner references removed")
			vmiIsRunningAndNotOwned(vm.Name, vm.Namespace)

			By("Deleting the orphaned VMI")
			err = virtClient.VirtualMachineInstance(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the VMI to be deleted")
			waitForDeletionVMI(vm.Name, vm.Namespace)

			By("Deleting the orphaned DataVolume")
			err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Delete(dataVolumeName, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the DataVolume to be deleted")
			waitForDeletionDataVolume(dataVolumeName, vm.Namespace)

			By("Waiting for the PVC to be deleted")
			waitForDeletionPVC(pvcName, vm.Namespace)
		})

	})

	Describe("Starting a VirtualMachine with a DataVolume", func() {
		Context("using Alpine import", func() {
			It("should be successfully started and stopped multiple times", func() {
				vm := tests.NewRandomVMWithDataVolume(tests.AlpineHttpUrl, tests.NamespaceTestDefault)
				vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
				num := 2
				By("Starting and stopping the VirtualMachine number of times")
				for i := 0; i < num; i++ {
					By(fmt.Sprintf("Doing run: %d", i))
					vm = tests.StartVirtualMachine(vm)
					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						expecter, err := tests.LoggedInAlpineExpecter(vmi)
						Expect(err).To(BeNil())
						expecter.Close()
					}
					vm = tests.StopVirtualMachine(vm)
				}
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})).To(Succeed())
			})

			It("should remove owner references on DataVolume if VM is orphan deleted.", func() {
				vm := tests.NewRandomVMWithDataVolume(tests.AlpineHttpUrl, tests.NamespaceTestDefault)
				vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				// Check for owner reference
				Eventually(func() []metav1.OwnerReference {
					dataVolume, _ := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Get(vm.Spec.DataVolumeTemplates[0].Name, metav1.GetOptions{})
					return dataVolume.OwnerReferences
				}, 100*time.Second, 1*time.Second).ShouldNot(BeEmpty())

				// Delete the VM with orphan Propagation
				orphanPolicy := metav1.DeletePropagationOrphan
				Expect(virtClient.VirtualMachine(vm.Namespace).
					Delete(vm.Name, &metav1.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())

				// Wait until the virtual machine instance is deleted
				Eventually(func() bool {
					_, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					if errors.IsNotFound(err) {
						return true
					}
					return false
				}, 100*time.Second, 1*time.Second).Should(BeTrue())

				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Get(vm.Spec.DataVolumeTemplates[0].Name, metav1.GetOptions{})
				Expect(dataVolume.OwnerReferences).To(BeEmpty())
				Expect(err).ToNot(HaveOccurred())

				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Delete(dataVolume.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

})
