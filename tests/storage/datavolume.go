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

package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	expect "github.com/google/goexpect"
	storagev1 "k8s.io/api/storage/v1"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
)

const InvalidDataVolumeUrl = "http://127.0.0.1/invalid"
const DummyFilePath = "/usr/share/nginx/html/dummy.file"

var _ = SIGDescribe("[Serial]DataVolume Integration", func() {

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

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachineInstance with a DataVolume as a volume source", func() {

		Context("Alpine import", func() {
			BeforeEach(func() {
				cdis, err := virtClient.CdiClient().CdiV1beta1().CDIs().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(cdis.Items).To(HaveLen(1))
				hasWaitForCustomerGate := false
				for _, feature := range cdis.Items[0].Spec.Config.FeatureGates {
					if feature == "HonorWaitForFirstConsumer" {
						hasWaitForCustomerGate = true
						break
					}
				}
				if !hasWaitForCustomerGate {
					Skip("HonorWaitForFirstConsumer is disabled in CDI, skipping tests relying on it")
				}
			})
			It("[test_id:3189]should be successfully started and stopped multiple times", func() {

				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				// This will only work on storage with binding mode WaitForFirstConsumer,
				if tests.HasBindingModeWaitForFirstConsumer() {
					tests.WaitForDataVolumePhaseWFFC(dataVolume.Namespace, dataVolume.Name, 30)
				}
				num := 2
				By("Starting and stopping the VirtualMachineInstance a number of times")
				for i := 1; i <= num; i++ {
					tests.WaitForDataVolumeReadyToStartVMI(vmi, 140)
					vmi := tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)
					// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
					// after being restarted multiple times
					if i == num {
						By("Checking that the VirtualMachineInstance console has expected output")
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}

					err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
					Expect(err).To(BeNil())
					tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)
				}
				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})

			It("[test_id:5252]should be successfully started when using a PVC volume owned by a DataVolume", func() {
				dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).To(BeNil())
				// This will only work on storage with binding mode WaitForFirstConsumer,
				if tests.HasBindingModeWaitForFirstConsumer() {
					tests.WaitForDataVolumePhaseWFFC(dataVolume.Namespace, dataVolume.Name, 30)
				}
				// with WFFC the run actually starts the import and then runs VM, so the timeout has to include both
				// import and start
				vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
				Expect(err).To(BeNil())
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
				Expect(err).To(BeNil())
			})
		})

		Context("with a PVC from a Datavolume", func() {
			var storageClass *storagev1.StorageClass
			BeforeEach(func() {
				// ensure that we always use a storage class which binds immediately,
				// otherwise we will never see a PVC appear for the datavolume
				bindMode := storagev1.VolumeBindingImmediate
				storageClass = &storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "fake",
					},
					Provisioner:       "afakeone",
					VolumeBindingMode: &bindMode,
				}
				storageClass, err = virtClient.StorageV1().StorageClasses().Create(context.Background(), storageClass, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				if storageClass != nil && storageClass.Name != "" {
					err := virtClient.StorageV1().StorageClasses().Delete(context.Background(), storageClass.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("[test_id:4643]should NOT be rejected when VM template lists a DataVolume, but VM lists PVC VolumeSource", func() {

				dv := tests.NewRandomDataVolumeWithHttpImportInStorageClass(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, storageClass.Name, k8sv1.ReadWriteOnce)
				_, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				defer func(dv *cdiv1.DataVolume) {
					By("Deleting the DataVolume")
					ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
				}(dv)

				Eventually(func() (*k8sv1.PersistentVolumeClaim, error) {
					return virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
				}, 30).Should(Not(BeNil()))

				vmi := tests.NewRandomVMI()

				diskName := "disk0"
				bus := "virtio"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: bus,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: dv.ObjectMeta.Name,
						},
					},
				})

				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")

				vm := tests.NewRandomVirtualMachine(vmi, true)
				dvt := &v1.DataVolumeTemplateSpec{
					ObjectMeta: dv.ObjectMeta,
					Spec:       dv.Spec,
				}
				vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
				_, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())
			})
			It("[Serial][test_id:4644]should fail to start when a volume is backed by PVC created by DataVolume instead of the DataVolume itself", func() {
				dv := tests.NewRandomDataVolumeWithHttpImportInStorageClass(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, storageClass.Name, k8sv1.ReadWriteOnce)
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				defer func(dv *cdiv1.DataVolume) {
					By("Deleting the DataVolume")
					ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
				}(dv)
				Eventually(func() error {
					_, err := virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
					return err
				}, 30*time.Second, 1*time.Second).Should(BeNil())

				vmi := tests.NewRandomVMI()

				diskName := "disk0"
				bus := "virtio"
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
					Name: diskName,
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: bus,
						},
					},
				})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
					Name: diskName,
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: dv.ObjectMeta.Name,
						},
					},
				})

				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")

				vm := tests.NewRandomVirtualMachine(vmi, true)
				_, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ShouldNot(HaveOccurred())

				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return vm.Status.Created
				}, 30*time.Second, 1*time.Second).Should(Equal(false))
			})
		})
	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with an invalid DataVolume", func() {
		Context("using DataVolume with invalid URL", func() {
			deleteDataVolume := func(dv *cdiv1.DataVolume) {
				By("Deleting the DataVolume")
				ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(context.Background(), dv.Name, metav1.DeleteOptions{})).To(Succeed(), metav1.DeleteOptions{})
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
			It("shold be possible to stop VM if datavolume is crashing", func() {
				dataVolume := tests.NewRandomDataVolumeWithHttpImport(InvalidDataVolumeUrl, tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
				vm := tests.NewRandomVirtualMachine(tests.NewRandomVMIWithDataVolume(dataVolume.Name), true)
				vm.Spec.DataVolumeTemplates = []v1.DataVolumeTemplateSpec{
					{
						ObjectMeta: dataVolume.ObjectMeta,
						Spec:       dataVolume.Spec,
					},
				}

				By("Creating a VM with an invalid DataVolume")
				vm, err := virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for DV to start crashing")
				tests.WaitForDataVolumeImportInProgress(vm.Namespace, dataVolume.Name, 30)

				By("Stop VM")
				tests.StopVirtualMachineWithTimeout(vm, time.Second*30)
			})

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
				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).To(BeNil())

				By("Creating a VM with an invalid DataVolume")
				//  Add the invalid DataVolume to a VMI
				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
				// Create a VM for this VMI
				vm := tests.NewRandomVirtualMachine(vmi, true)
				_, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				waitForVM(vm, v1.Pending, "VMI with inconsistent DV should be created")

				By("Fix DataVolume URL")
				createDummyFile(DummyFilePath, "1")
				defer deleteDummyFile(DummyFilePath)

				By("Wait for DataVolume to complete")
				Eventually(func() cdiv1.DataVolumePhase {
					dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Get(context.Background(), dataVolume.Name, metav1.GetOptions{})
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
			dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})
			if err == nil && dataVolume.DeletionTimestamp == nil {
				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
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
				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})
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
				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})
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
				_, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
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
				_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(namespace).Get(context.Background(), name, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 100*time.Second, 1*time.Second).Should(BeTrue())
		}

		waitForDeletionPVC := func(name, namespace string) {
			Eventually(func() bool {
				_, err = virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), name, metav1.GetOptions{})
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
			err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Delete(context.Background(), dataVolumeName, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for the DataVolume to be deleted")
			waitForDeletionDataVolume(dataVolumeName, vm.Namespace)

			By("Waiting for the PVC to be deleted")
			waitForDeletionPVC(pvcName, vm.Namespace)
		})

	})

	Describe("[rfe_id:3188][crit:high][vendor:cnv-qe@redhat.com][level:system] Starting a VirtualMachine with a DataVolume", func() {
		Context("using Alpine http import", func() {
			It("a DataVolume with preallocation shouldn't have discard=unmap", func() {
				var vm *v1.VirtualMachine
				vm = tests.NewRandomVMWithDataVolume(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault)
				preallocation := true
				vm.Spec.DataVolumeTemplates[0].Spec.Preallocation = &preallocation

				vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				vm = tests.StartVirtualMachine(vm)
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domXml).ToNot(ContainSubstring("discard='unmap'"))
				vm = tests.StopVirtualMachine(vm)
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})).To(Succeed())
			})

			table.DescribeTable("[test_id:3191]should be successfully started and stopped multiple times", func(isHTTP bool) {
				var vm *v1.VirtualMachine
				if isHTTP {
					vm = tests.NewRandomVMWithDataVolume(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault)
				} else {
					url := "docker://" + cd.ContainerDiskFor(cd.ContainerDiskAlpine)
					vm = tests.NewRandomVMWithRegistryDataVolume(url, tests.NamespaceTestDefault)
				}
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
						Expect(console.LoginToAlpine(vmi)).To(Succeed())
					}
					vm = tests.StopVirtualMachine(vm)
				}
				Expect(virtClient.VirtualMachine(vm.Namespace).Delete(vm.Name, &metav1.DeleteOptions{})).To(Succeed())
			},

				table.Entry("with http import", true),
				table.Entry("with registry import", false),
			)

			It("[test_id:3192]should remove owner references on DataVolume if VM is orphan deleted.", func() {
				// Cascade=false delete fails in ocp 3.11 with CRDs that contain multiple versions.
				tests.SkipIfOpenShiftAndBelowOrEqualVersion("cascade=false delete does not work with CRD multi version support in ocp 3.11", "1.11.0")

				vm := tests.NewRandomVMWithDataVolume(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault)
				vm, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
				Expect(err).ToNot(HaveOccurred())

				// Check for owner reference
				Eventually(func() []metav1.OwnerReference {
					dataVolume, _ := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Get(context.Background(), vm.Spec.DataVolumeTemplates[0].Name, metav1.GetOptions{})
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

				dataVolume, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Get(context.Background(), vm.Spec.DataVolumeTemplates[0].Name, metav1.GetOptions{})
				Expect(dataVolume.OwnerReferences).To(BeEmpty())
				Expect(err).ToNot(HaveOccurred())

				err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(vm.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
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
			var storageClass string

			BeforeEach(func() {
				var exists bool
				storageClass, exists = tests.GetCephStorageClass()
				if !exists {
					Skip("Skip OCS tests when Ceph is not present")
				}
				var err error
				dv := tests.NewRandomDataVolumeWithHttpImportInStorageClass(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestAlternative, storageClass, k8sv1.ReadWriteOnce)
				dataVolume, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					dataVolume, err = virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Get(context.Background(), dataVolume.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(dataVolume.Status.Phase).ToNot(Equal(cdiv1.Failed))
					return dataVolume.Status.Phase == cdiv1.Succeeded
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			})

			AfterEach(func() {
				if cloneRole != nil {
					err := virtClient.RbacV1().Roles(cloneRole.Namespace).Delete(context.Background(), cloneRole.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if cloneRoleBinding != nil {
					err := virtClient.RbacV1().RoleBindings(cloneRoleBinding.Namespace).Delete(context.Background(), cloneRoleBinding.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if createdVirtualMachine != nil {
					err := virtClient.VirtualMachine(createdVirtualMachine.Namespace).Delete(createdVirtualMachine.Name, &metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				if dataVolume != nil {
					err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Delete(context.Background(), dataVolume.Name, metav1.DeleteOptions{})
					if err != nil && !errors.IsNotFound(err) {
						Expect(err).ToNot(HaveOccurred())
					}
				}
			})

			table.DescribeTable("deny then allow clone request on rook-ceph", func(role *rbacv1.Role, allServiceAccounts, allServiceAccountsInNamespace bool) {
				vm := tests.NewRandomVMWithCloneDataVolume(dataVolume.Namespace, dataVolume.Name, tests.NamespaceTestDefault)
				const volumeName = "sa"
				saVol := v1.Volume{
					Name: volumeName,
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: tests.AdminServiceAccountName,
						},
					},
				}
				vm.Spec.DataVolumeTemplates[0].Spec.PVC.StorageClassName = pointer.StringPtr(storageClass)
				vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, saVol)
				vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{Name: volumeName})

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

				saName := tests.AdminServiceAccountName
				saNamespace := tests.NamespaceTestDefault

				if allServiceAccounts {
					saName = ""
					saNamespace = ""
				} else if allServiceAccountsInNamespace {
					saName = ""
				}

				// add permission
				cloneRole, cloneRoleBinding = addClonePermission(virtClient, role, saName, saNamespace, tests.NamespaceTestAlternative)

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
					dv, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(createdVirtualMachine.Namespace).Get(context.Background(), targetDVName, metav1.GetOptions{})
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
				table.Entry("[test_id:3193]with explicit role", explicitCloneRole, false, false),
				table.Entry("[test_id:3194]with implicit role", implicitCloneRole, false, false),
				table.Entry("[test_id:5253]with explicit role (all namespaces)", explicitCloneRole, true, false),
				table.Entry("[test_id:5254]with explicit role (one namespace)", explicitCloneRole, false, true),
			)
		})
	})

	Context("Fedora VMI tests", func() {
		getImageSize := func(vmi *v1.VirtualMachineInstance, dv *cdiv1.DataVolume, withOCS bool) int64 {
			var imageSize int64
			var unused string
			if withOCS {
				var matchingPv *k8sv1.PersistentVolume
				pvs, err := virtClient.CoreV1().PersistentVolumes().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, pv := range pvs.Items {
					if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Name == dv.Name {
						matchingPv = &pv
						break
					}
				}
				Expect(matchingPv).ToNot(BeNil())
				rbdCmd := fmt.Sprintf("rbd diff %s/%s | awk '{ SUM += $2 } END { print SUM }'",
					matchingPv.Spec.CSI.VolumeAttributes["pool"],
					matchingPv.Spec.CSI.VolumeAttributes["imageName"])
				dfOutput, err := tests.ExecuteCommandOnCephToolbox(virtClient, []string{"sh", "-c", rbdCmd})
				Expect(err).ToNot(HaveOccurred())
				fmt.Sscanf(dfOutput, "%d\n", &imageSize, &unused)
			} else {
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				lsOutput, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					"compute",
					[]string{"ls", "-s", "/var/run/kubevirt-private/vmi-disks/disk0/disk.img"},
				)
				Expect(err).ToNot(HaveOccurred())
				fmt.Sscanf(lsOutput, "%d %s", &imageSize, &unused)
			}
			return imageSize
		}

		noop := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			return dv
		}
		addPreallocationTrue := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			preallocation := true
			dv.Spec.Preallocation = &preallocation
			return dv
		}
		addPreallocationFalse := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			preallocation := false
			dv.Spec.Preallocation = &preallocation
			return dv
		}
		addThickProvisionedTrueAnnotation := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			dv.Annotations = map[string]string{"user.custom.annotation/storage.thick-provisioned": "true"}
			return dv
		}
		addThickProvisionedFalseAnnotation := func(dv *cdiv1.DataVolume) *cdiv1.DataVolume {
			dv.Annotations = map[string]string{"user.custom.annotation/storage.thick-provisioned": "false"}
			return dv
		}
		table.DescribeTable("[rfe_id:5070][crit:medium][vendor:cnv-qe@redhat.com][level:component]fstrim from the VM influences disk.img", func(dvChange func(*cdiv1.DataVolume) *cdiv1.DataVolume, expectSmaller, withOCS bool) {
			dataVolume := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.FedoraHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
			dataVolume.Spec.PVC.Resources.Requests[k8sv1.ResourceStorage] = resource.MustParse("5Gi")
			dataVolume = dvChange(dataVolume)
			preallocated := dataVolume.Spec.Preallocation != nil && *dataVolume.Spec.Preallocation

			if withOCS {
				volumeMode := k8sv1.PersistentVolumeBlock
				dataVolume.Spec.PVC.VolumeMode = &volumeMode
				sc, exists := tests.GetCephStorageClass()
				if !exists {
					Skip("Skip OCS tests when Ceph is not present")
				}
				dataVolume.Spec.PVC.StorageClassName = &sc
			}

			vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
			vmi.Spec.Domain.Devices.Disks[0].DiskDevice.Disk.Bus = "scsi"
			tests.AddUserData(vmi, "cloud-init", tests.GetFedoraToolsGuestAgentUserData())

			_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dataVolume.Namespace).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForDataVolumeReadyToStartVMI(vmi, 140)
			vmi = tests.RunVMIAndExpectLaunchWithDataVolume(vmi, dataVolume, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			imageSizeAfterBoot := getImageSize(vmi, dataVolume, withOCS)
			By(fmt.Sprintf("image size after boot is %d", imageSizeAfterBoot))

			By("Filling out disk space")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "dd if=/dev/urandom of=largefile bs=1M count=500 2> /dev/null\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 360)).To(Succeed(), "should write a large file")

			if preallocated {
				// Preallocation means no changes to disk size
				Eventually(getImageSize(vmi, dataVolume, withOCS), 120*time.Second).Should(Equal(imageSizeAfterBoot))
			} else {
				Eventually(getImageSize(vmi, dataVolume, withOCS), 120*time.Second).Should(BeNumerically(">", imageSizeAfterBoot))
			}

			imageSizeBeforeTrim := getImageSize(vmi, dataVolume, withOCS)
			By(fmt.Sprintf("image size before trim is %d", imageSizeBeforeTrim))

			By("Writing a small file so that we detect a disk space usage change.")
			By("Deleting large file and trimming disk")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// Write a small file so that we'll have an increase in image size if trim is unsupported.
				&expect.BSnd{S: "dd if=/dev/urandom of=smallfile bs=1M count=100 2> /dev/null\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "rm -f largefile\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 60)).To(Succeed(), "should trim within the VM")

			Eventually(func() bool {
				By("Running trim")
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo fstrim -v /\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "sync\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 60)
				Expect(err).ToNot(HaveOccurred())

				currentImageSize := getImageSize(vmi, dataVolume, withOCS)
				if expectSmaller {
					// Trim should make the space usage go down
					By(fmt.Sprintf("We expect disk usage to go down from the use of trim.\nIt is currently %d and was previously %d", currentImageSize, imageSizeBeforeTrim))
					return currentImageSize < imageSizeBeforeTrim
				} else if preallocated {
					By(fmt.Sprintf("Trim shouldn't do anything, and preallocation should mean no change to disk usage.\nIt is currently %d and was previously %d", currentImageSize, imageSizeBeforeTrim))
					return currentImageSize == imageSizeBeforeTrim

				} else {
					By(fmt.Sprintf("Trim shouldn't do anything, but we expect size usage to go up, because we wrote another small file.\nIt is currently %d and was previously %d", currentImageSize, imageSizeBeforeTrim))
					return currentImageSize > imageSizeBeforeTrim
				}
			}, 120*time.Second).Should(BeTrue())

			err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Delete(vmi.Name, &metav1.DeleteOptions{})
			Expect(err).To(BeNil())
		},
			table.Entry("[test_id:5894]by default, fstrim will make the image smaller", noop, true, false),
			table.Entry("[QUARANTINE][test_id:5898]with preallocation true, fstrim has no effect", addPreallocationTrue, false, false),
			table.Entry("[test_id:5897]with preallocation false, fstrim will make the image smaller", addPreallocationFalse, true, false),
			table.Entry("[test_id:5899]with thick provision true, fstrim has no effect", addThickProvisionedTrueAnnotation, false, false),
			table.Entry("[test_id:5896]with thick provision false, fstrim will make the image smaller", addThickProvisionedFalseAnnotation, true, false),
			table.Entry("[test_id:5894]with OCS, by default, fstrim will make the ceph space usage go down", noop, true, true),
			table.Entry("[test_id:5898]with OCS, with preallocation true, fstrim has no effect", addPreallocationTrue, false, true),
			table.Entry("[test_id:5897]with OCS, with preallocation false, fstrim will the ceph space usage go down", addPreallocationFalse, true, true),
			table.Entry("[test_id:5899]with OCS, with thick provision true, fstrim has no effect", addThickProvisionedTrueAnnotation, false, true),
			table.Entry("[test_id:5896]with OCS, with thick provision false, fstrim will make the ceph space usage go down", addThickProvisionedFalseAnnotation, true, true),
		)
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
	role, err := client.RbacV1().Roles(targetNamesace).Create(context.Background(), role, metav1.CreateOptions{})
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
	}

	if sa != "" {
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: saNamespace,
			},
		}
	} else {
		g := "system:serviceaccounts"
		if saNamespace != "" {
			g += ":" + saNamespace
		}
		rb.Subjects = []rbacv1.Subject{
			{
				Kind:     "Group",
				Name:     g,
				APIGroup: "rbac.authorization.k8s.io",
			},
		}
	}

	rb, err = client.RbacV1().RoleBindings(targetNamesace).Create(context.Background(), rb, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return role, rb
}
