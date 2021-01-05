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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
)

const InvalidDataVolumeUrl = "http://127.0.0.1/invalid"
const DummyFilePath = "/usr/share/nginx/html/dummy.file"

var _ = Describe("[Serial]DataVolume Integration", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
		if !tests.HasCDI() {
			Skip("Skip DataVolume tests when CDI is not present")
		}

	})

	runVMIAndExpectLaunch := func(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
		tests.WaitForSuccessfulDataVolumeImportOfVMI(vmi, timeout)

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

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachineInstance with a DataVolume as a volume source", func() {
		Context("using Alpine import", func() {
			It("[test_id:3189]should be successfully started and stopped multiple times", func() {

				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
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

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with an invalid DataVolume", func() {
		Context("using DataVolume with invalid URL", func() {
			deleteDataVolume := func(dv *cdiv1.DataVolume) {
				By("Deleting the DataVolume")
				ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(dv.Name, &metav1.DeleteOptions{})).To(Succeed())
			}

			waitForDv := func(dataVolume *cdiv1.DataVolume) {
				Eventually(func() cdiv1.DataVolumePhase {
					dv, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Get(dataVolume.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return dv.Status.Phase
				}, 30*time.Second, 1*time.Second).
					Should(Equal(cdiv1.ImportInProgress), "Timed out waiting for DataVolume to enter ImportInProgress phase")
			}

			deleteDummyFile := func(fileName string) {
				httpPod, err := tests.GetRunningPodByLabel("cdi-http-import-server", "kubevirt.io", flags.KubeVirtInstallNamespace, "")
				Expect(err).ToNot(HaveOccurred())
				By("Deleting dummy file")
				_, err = tests.ExecuteCommandOnPod(
					virtClient,
					httpPod,
					httpPod.Spec.Containers[0].Name,
					[]string{"rm", fileName},
				)
				Expect(err).ToNot(HaveOccurred())
			}

			createDummyFile := func(fileName string, sizeInMB string) {
				httpPod, err := tests.GetRunningPodByLabel("cdi-http-import-server", "kubevirt.io", flags.KubeVirtInstallNamespace, "")
				Expect(err).ToNot(HaveOccurred())
				_, _, err = tests.ExecuteCommandOnPodV2(
					virtClient,
					httpPod,
					httpPod.Spec.Containers[0].Name,
					[]string{"dd", "if=/dev/urandom", "of=" + fileName, "bs=1M", "count=" + sizeInMB},
				)
				Expect(err).ToNot(HaveOccurred())
			}

			waitForVM := func(vm *v1.VirtualMachine, phase v1.VirtualMachineInstancePhase, message string) {
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.GetName(), &metav1.GetOptions{})
					if err != nil {
						Expect(err.Error()).To(ContainSubstring("not found"),
							"A 404 while VMI is being created would be normal. All other errors are unexpected")
						return v1.VmPhaseUnset
					}
					return vmi.Status.Phase
				}, 100*time.Second, 1*time.Second).Should(Equal(phase), message)
			}

			It("[test_id:3190]should correctly handle invalid DataVolumes", func() {
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
				waitForVM(vm, v1.Pending, "VMI with invalid DataVolume should not be scheduled")
			})
			It("[test_id:3190]should correctly handle eventually consistent DataVolumes", func() {
				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.DummyFileHttpUrl),
					tests.NamespaceTestDefault,
					k8sv1.ReadWriteOnce,
				)
				defer deleteDataVolume(dataVolume)

				By("Creating DataVolume with invalid URL")
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(dataVolume)
				Expect(err).To(BeNil())
				waitForDv(dataVolume)

				By("Creating a VM with an invalid DataVolume")
				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)
				_, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				vmi, _ = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.GetName(), &metav1.GetOptions{})
				waitForVM(vm, v1.Pending, "VMI with inconsistent DV should be created")

				By("Fix DataVolume URL")
				createDummyFile(DummyFilePath, "1")
				defer deleteDummyFile(DummyFilePath)

				By("Wait for DataVolume to complete")
				Eventually(func() cdiv1.DataVolumePhase {
					dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Get(dataVolume.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					return dataVolume.Status.Phase
				}, 160*time.Second, 1*time.Second).Should(Equal(cdiv1.Succeeded))

				By("Waiting for VMI to be created")
				waitForVM(vm, v1.Running, "VMI with eventually consistent DataVolume should have been started")
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

			vm = tests.NewRandomVMWithDataVolume(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault)
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
				// In some tests, OwnerReferences in k8s can cause this to be deleted already
				// just ignore 404's to avoid that race.
				if err != nil && !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			}
		}

		deleteIfExistsDataVolume := func(name string, namespace string) {
			dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(name, metav1.GetOptions{})
			if err == nil && dataVolume.DeletionTimestamp == nil {
				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Delete(name, &metav1.DeleteOptions{})
				// In some tests, OwnerReferences in k8s can cause this to be deleted already
				// just ignore 404's to avoid that race.
				if err != nil && !errors.IsNotFound(err) {
					Expect(err).ToNot(HaveOccurred())
				}
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

			// Cascade=false delete fails in ocp 3.11 with CRDs that contain multiple versions.
			tests.SkipIfOpenShiftAndBelowOrEqualVersion("cascade=false delete does not work with CRD multi version support in ocp 3.11", "1.11.0")

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

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with a DataVolume", func() {
		Context("using Alpine import", func() {
			It("[test_id:3191]should be successfully started and stopped multiple times", func() {
				vm := tests.NewRandomVMWithDataVolume(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault)
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

			It("[test_id:3192]should remove owner references on DataVolume if VM is orphan deleted.", func() {
				// Cascade=false delete fails in ocp 3.11 with CRDs that contain multiple versions.
				tests.SkipIfOpenShiftAndBelowOrEqualVersion("cascade=false delete does not work with CRD multi version support in ocp 3.11", "1.11.0")

				vm := tests.NewRandomVMWithDataVolume(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault)
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

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] DataVolume clone permission checking", func() {
		Context("using Alpine import/clone", func() {
			var dataVolume *cdiv1.DataVolume
			var createdVirtualMachine *v1.VirtualMachine
			var cloneRole *rbacv1.Role
			var cloneRoleBinding *rbacv1.RoleBinding

			BeforeEach(func() {
				var err error
				dv := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestAlternative, k8sv1.ReadWriteOnce)
				dataVolume, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					dataVolume, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Get(dataVolume.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(dataVolume.Status.Phase).ToNot(Equal(cdiv1.Failed))
					return dataVolume.Status.Phase == cdiv1.Succeeded
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			})

			AfterEach(func() {
				if cloneRole != nil {
					err := virtClient.RbacV1().Roles(cloneRole.Namespace).Delete(cloneRole.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if cloneRoleBinding != nil {
					err := virtClient.RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(cloneRoleBinding.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if createdVirtualMachine != nil {
					err := virtClient.VirtualMachine(createdVirtualMachine.Namespace).Delete(createdVirtualMachine.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if dataVolume != nil {
					err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(dataVolume.Name, &metav1.DeleteOptions{})
					if err != nil && !errors.IsNotFound(err) {
						Expect(err).ToNot(HaveOccurred())
					}
				}
			})

			table.DescribeTable("deny then allow clone request", func(role *rbacv1.Role) {
				vm := tests.NewRandomVMWithCloneDataVolume(dataVolume.Namespace, dataVolume.Name, tests.NamespaceTestDefault)
				saVol := v1.Volume{
					Name: "sa",
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: tests.AdminServiceAccountName,
						},
					},
				}
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, saVol)

				vmBytes, err := json.Marshal(vm)
				Expect(err).ToNot(HaveOccurred())
				byteReader := bytes.NewReader(vmBytes)

				// this should fail because don't have permission
				stdOut, stdErr, err := tests.RunCommandWithNSAndInput(vm.Namespace, byteReader, "kubectl", "create", "-f", "-")
				if err == nil {
					fmt.Printf("command should have failed\nstdOut\n%s\nstdErr\n%s\n", stdOut, stdErr)
					Expect(err).To(HaveOccurred())
				}
				Expect(stdErr).Should(ContainSubstring("Authorization failed, message is:"))

				// add permission
				cloneRole, cloneRoleBinding = addClonePermission(virtClient, role, tests.AdminServiceAccountName, tests.NamespaceTestDefault, tests.NamespaceTestAlternative)

				// sometimes it takes a bit for permission to actually be applied so eventually
				Eventually(func() bool {
					byteReader = bytes.NewReader(vmBytes)
					stdOut, stdErr, err = tests.RunCommandWithNSAndInput(vm.Namespace, byteReader, "kubectl", "create", "-f", "-")
					if err != nil {
						fmt.Printf("command should have succeeded maybe new permissions not applied yet\nstdOut\n%s\nstdErr\n%s\n", stdOut, stdErr)
						return false
					}
					return true
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				createdVirtualMachine = vm

				// wait for clone to complete
				targetDVName := vm.Spec.DataVolumeTemplates[0].Name
				Eventually(func() bool {
					dv, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(createdVirtualMachine.Namespace).Get(targetDVName, metav1.GetOptions{})
					if err != nil && errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					return dv.Status.Phase == cdiv1.Succeeded
				}, 90*time.Second, 1*time.Second).Should(BeTrue())

				// start/stop vm
				createdVirtualMachine = tests.StartVirtualMachine(createdVirtualMachine)
				createdVirtualMachine = tests.StopVirtualMachine(createdVirtualMachine)
			},
				table.Entry("[test_id:3193]with explicit role", explicitCloneRole),
				table.Entry("[test_id:3194]with implicit role", implicitCloneRole),
			)
		})
	})
})

var explicitCloneRole = &rbacv1.Role{
	ObjectMeta: metav1.ObjectMeta{
		Name: "explicit-clone-role",
	},
	Rules: []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"cdi.kubevirt.io",
			},
			Resources: []string{
				"datavolumes/source",
			},
			Verbs: []string{
				"create",
			},
		},
	},
}

var implicitCloneRole = &rbacv1.Role{
	ObjectMeta: metav1.ObjectMeta{
		Name: "implicit-clone-role",
	},
	Rules: []rbacv1.PolicyRule{
		{
			APIGroups: []string{
				"",
			},
			Resources: []string{
				"pods",
			},
			Verbs: []string{
				"create",
			},
		},
	},
}

func addClonePermission(client kubecli.KubevirtClient, role *rbacv1.Role, sa, saNamespace, targetNamesace string) (*rbacv1.Role, *rbacv1.RoleBinding) {
	role, err := client.RbacV1().Roles(targetNamesace).Create(role)
	Expect(err).ToNot(HaveOccurred())

	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: role.Name,
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     role.Name,
			APIGroup: "rbac.authorization.k8s.io",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: saNamespace,
			},
		},
	}

	rb, err = client.RbacV1().RoleBindings(targetNamesace).Create(rb)
	Expect(err).ToNot(HaveOccurred())

	return role, rb
}
