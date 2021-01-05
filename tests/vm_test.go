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

package tests_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = Describe("[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachine", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	runStrategyAlways := v1.RunStrategyAlways
	runStrategyHalted := v1.RunStrategyHalted

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Context("An invalid VirtualMachine given", func() {

		It("[test_id:1518]should be rejected on POST", func() {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			newVM := tests.NewRandomVirtualMachine(template, false)

			jsonBytes, err := json.Marshal(newVM)
			Expect(err).To(BeNil())

			// change the name of a required field (like domain) so validation will fail
			jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()
			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		})
		It("[test_id:1519]should reject POST if validation webhoook deems the spec is invalid", func() {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			template.Spec.Domain.Devices.Disks = append(template.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			newVM := tests.NewRandomVirtualMachine(template, false)

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(newVM).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

			reviewResponse := &v12.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).To(BeNil())

			Expect(len(reviewResponse.Details.Causes)).To(Equal(1))
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[2].name"))
		})
		It("[test_id:4643]should be rejected when VM template lists a DataVolume, but VM lists PVC VolumeSource", func() {
			dv := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
			_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
			Expect(err).To(BeNil())

			defer func(dv *cdiv1.DataVolume) {
				By("Deleting the DataVolume")
				ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(dv.Name, &metav1.DeleteOptions{})).To(Succeed())
			}(dv)
			tests.WaitForSuccessfulDataVolumeImport(dv, 240)

			vmi := tests.NewRandomVMI()

			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")

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
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dv)
			_, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).Should(HaveOccurred())
		})
		It("[Serial][test_id:4644]should fail to start when a volume is backed by PVC created by DataVolume instead of the DataVolume itself", func() {
			dv := tests.NewRandomDataVolumeWithHttpImport(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, k8sv1.ReadWriteOnce)
			_, err := virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Create(dv)
			Expect(err).To(BeNil())

			defer func(dv *cdiv1.DataVolume) {
				By("Deleting the DataVolume")
				ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(dv.Name, &metav1.DeleteOptions{})).To(Succeed())
			}(dv)
			tests.WaitForSuccessfulDataVolumeImport(dv, 240)

			vmi := tests.NewRandomVMI()

			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("64M")

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
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return vm.Status.Created
			}, 30*time.Second, 1*time.Second).Should(Equal(false))
		})
	})

	Context("[Serial]A mutated VirtualMachine given", func() {

		var testingMachineType string = "pc-q35-2.7"

		BeforeEach(func() {
			_, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get("kubevirt-config", metav1.GetOptions{})
			if err != nil && !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			if errors.IsNotFound(err) {
				// create an empty kubevirt-config configmap if none exists.
				cfgMap := &k8sv1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: "kubevirt-config"},
					Data: map[string]string{
						"machine-type": testingMachineType,
					},
				}

				_, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Create(cfgMap)
				Expect(err).ToNot(HaveOccurred())
			} else if err == nil {
				tests.UpdateClusterConfigValueAndWait("machine-type", testingMachineType)
			}
		})

		newVirtualMachineInstanceWithContainerDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			return tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n"), nil
		}

		createVirtualMachine := func(running bool, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
			By("Creating VirtualMachine")
			vm := tests.NewRandomVirtualMachine(template, running)
			newVM, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			return newVM
		}

		It("[test_id:3312]should set the default MachineType when created without explicit value", func() {
			By("Creating VirtualMachine")
			template, _ := newVirtualMachineInstanceWithContainerDisk()
			template.Spec.Domain.Machine.Type = ""
			vm := createVirtualMachine(false, template)

			createdVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(createdVM.Spec.Template.Spec.Domain.Machine.Type).To(Equal(testingMachineType))
		})

		It("[test_id:3311]should keep the supplied MachineType when created", func() {
			By("Creating VirtualMachine")
			explicitMachineType := "pc-q35-3.0"
			template, _ := newVirtualMachineInstanceWithContainerDisk()
			template.Spec.Domain.Machine.Type = explicitMachineType
			vm := createVirtualMachine(false, template)

			createdVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(createdVM.Spec.Template.Spec.Domain.Machine.Type).To(Equal(explicitMachineType))
		})
	})

	Context("A valid VirtualMachine given", func() {
		type vmiBuilder func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume)

		newVirtualMachineInstanceWithContainerDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			return tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n"), nil
		}

		newVirtualMachineInstanceWithOCSFileDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			return tests.NewRandomVirtualMachineInstanceWithOCSDisk(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, v13.ReadWriteOnce, v13.PersistentVolumeFilesystem)
		}

		newVirtualMachineInstanceWithOCSBlockDisk := func() (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
			return tests.NewRandomVirtualMachineInstanceWithOCSDisk(tests.GetUrl(tests.AlpineHttpUrl), tests.NamespaceTestDefault, v13.ReadWriteOnce, v13.PersistentVolumeBlock)
		}

		deleteDataVolume := func(dv *cdiv1.DataVolume) {
			if dv != nil {
				By("Deleting the DataVolume")
				ExpectWithOffset(1, virtClient.CdiClient().CdiV1alpha1().DataVolumes(dv.Namespace).Delete(dv.Name, &metav1.DeleteOptions{})).To(Succeed())
			}
		}

		createVirtualMachine := func(running bool, template *v1.VirtualMachineInstance) *v1.VirtualMachine {
			By("Creating VirtualMachine")
			vm := tests.NewRandomVirtualMachine(template, running)
			newVM, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			return newVM
		}

		newVirtualMachine := func(running bool) *v1.VirtualMachine {
			template, _ := newVirtualMachineInstanceWithContainerDisk()
			return createVirtualMachine(running, template)
		}

		newVirtualMachineWithRunStrategy := func(runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
			vmiImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")

			var newVM *v1.VirtualMachine
			var err error

			newVM = NewRandomVirtualMachineWithRunStrategy(template, runStrategy)

			newVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(newVM)
			Expect(err).ToNot(HaveOccurred())

			return newVM
		}

		startVM := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Starting the VirtualMachine")

			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVM.Spec.Running = nil
				updatedVM.Spec.RunStrategy = &runStrategyAlways
				_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Observe the VirtualMachineInstance created
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("VMI has the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			return updatedVM
		}

		stopVM := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Stopping the VirtualMachine")

			err = tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta v12.ObjectMeta) error {
				updatedVM, err := virtClient.VirtualMachine(meta.Namespace).Get(meta.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVM.Spec.Running = nil
				updatedVM.Spec.RunStrategy = &runStrategyHalted
				_, err = virtClient.VirtualMachine(meta.Namespace).Update(updatedVM)
				return err
			})
			Expect(err).ToNot(HaveOccurred())

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Observe the VirtualMachineInstance deleted
			Eventually(func() bool {
				_, err = virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 300*time.Second, 1*time.Second).Should(BeTrue(), "The vmi did not disappear")

			By("VMI has not the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeFalse())

			return updatedVM
		}

		startVMIDontWait := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Starting the VirtualMachineInstance")

			err := tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta v12.ObjectMeta) error {
				updatedVM, err := virtClient.VirtualMachine(meta.Namespace).Get(meta.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVM.Spec.Running = nil
				updatedVM.Spec.RunStrategy = &runStrategyAlways
				_, err = virtClient.VirtualMachine(meta.Namespace).Update(updatedVM)
				return err
			})
			Expect(err).ToNot(HaveOccurred())

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			return updatedVM
		}

		It("[test_id:3161]should carry annotations to VMI", func() {
			annotations := map[string]string{
				"testannotation": "test",
			}

			vm := newVirtualMachine(false)

			err = tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta v12.ObjectMeta) error {
				vm, err = virtClient.VirtualMachine(meta.Namespace).Get(meta.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Spec.Template.ObjectMeta.Annotations = annotations
				vm, err = virtClient.VirtualMachine(meta.Namespace).Update(vm)
				return err
			})
			Expect(err).ToNot(HaveOccurred())

			startVMIDontWait(vm)

			By("checking for annotations to be present")
			Eventually(func() map[string]string {
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				if err != nil {
					return map[string]string{}
				}
				return vmi.Annotations
			}, 300*time.Second, 1*time.Second).Should(HaveKeyWithValue("testannotation", "test"), "VM should start normaly.")
		})

		It("[test_id:3162]should ignore kubernetes and kubevirt annotations to VMI", func() {
			annotations := map[string]string{
				"kubevirt.io/test":   "test",
				"kubernetes.io/test": "test",
			}

			vm := newVirtualMachine(false)

			err = tests.RetryWithMetadataIfModified(vm.ObjectMeta, func(meta v12.ObjectMeta) error {
				vm, err = virtClient.VirtualMachine(meta.Namespace).Get(meta.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vm.Annotations = annotations
				vm, err = virtClient.VirtualMachine(meta.Namespace).Update(vm)
				return err
			})
			Expect(err).ToNot(HaveOccurred())

			startVMIDontWait(vm)

			By("checking for annotations to not be present")
			vmi := &v1.VirtualMachineInstance{}

			Eventually(func() error {
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred(), "VMI should be created normaly.")

			Expect(vmi.Annotations).ShouldNot(HaveKey("kubevirt.io/test"), "kubevirt internal annotations should be ignored")
			Expect(vmi.Annotations).ShouldNot(HaveKey("kubernetes.io/test"), "kubernetes internal annotations should be ignored")
		})

		table.DescribeTable("[test_id:1520]should update VirtualMachine once VMIs are up", func(createTemplate vmiBuilder) {
			template, dv := createTemplate()
			defer deleteDataVolume(dv)
			newVM := createVirtualMachine(true, template)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		},
			table.Entry("with ContainerDisk", newVirtualMachineInstanceWithContainerDisk),
			table.Entry("[Serial]with OCS Filesystem Disk", newVirtualMachineInstanceWithOCSFileDisk),
			table.Entry("[Serial]with OCS Block Disk", newVirtualMachineInstanceWithOCSBlockDisk),
		)

		table.DescribeTable("[test_id:1521]should remove VirtualMachineInstance once the VM is marked for deletion", func(createTemplate vmiBuilder) {
			template, dv := createTemplate()
			defer deleteDataVolume(dv)
			newVM := createVirtualMachine(true, template)
			// Delete it
			Expect(virtClient.VirtualMachine(newVM.Namespace).Delete(newVM.Name, &v12.DeleteOptions{})).To(Succeed())
			// Wait until VMI is gone
			Eventually(func() int {
				vmis, err := virtClient.VirtualMachineInstance(newVM.Namespace).List(&v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(vmis.Items)
			}, 300*time.Second, 2*time.Second).Should(BeZero(), "The VirtualMachineInstance did not disappear")
		},
			table.Entry("with ContainerDisk", newVirtualMachineInstanceWithContainerDisk),
			table.Entry("[Serial]with OCS Filesystem Disk", newVirtualMachineInstanceWithOCSFileDisk),
			table.Entry("[Serial]with OCS Block Disk", newVirtualMachineInstanceWithOCSBlockDisk),
		)

		It("[test_id:1522]should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {

			// Cascade=false delete fails in ocp 3.11 with CRDs that contain multiple versions.
			tests.SkipIfOpenShiftAndBelowOrEqualVersion("cascade=false delete does not work with CRD multi version support in ocp 3.11", "1.11.0")

			newVM := newVirtualMachine(true)

			By("Getting owner references")
			Eventually(func() []v12.OwnerReference {
				// Check for owner reference
				vmi, _ := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				return vmi.OwnerReferences
			}, 300*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			// Delete it
			orphanPolicy := v12.DeletePropagationOrphan
			By("Deleting VM")
			Expect(virtClient.VirtualMachine(newVM.Namespace).
				Delete(newVM.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
			// Wait until the virtual machine is deleted
			By("Waiting for VM to delete")
			Eventually(func() bool {
				_, err := virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			By("Verifying orphaned VMI still exists")
			vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1523]should recreate VirtualMachineInstance if it gets deleted", func() {
			newVM := startVM(newVirtualMachine(false))

			currentVMI, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(virtClient.VirtualMachineInstance(newVM.Namespace).Delete(newVM.Name, &v12.DeleteOptions{})).To(Succeed())

			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return false
				}
				if vmi.UID != currentVMI.UID {
					return true
				}
				return false
			}, 240*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("[test_id:1524]should recreate VirtualMachineInstance if the VirtualMachineInstance's pod gets deleted", func() {
			var firstVMI *v1.VirtualMachineInstance
			var curVMI *v1.VirtualMachineInstance
			var err error

			By("Start a new VM")
			newVM := newVirtualMachine(true)

			// wait for a running VirtualMachineInstance.
			By("Waiting for the VM's VirtualMachineInstance to start")
			Eventually(func() error {
				firstVMI, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				if err != nil {
					return err
				}
				if !firstVMI.IsRunning() {
					return fmt.Errorf("vmi still isn't running")
				}
				return nil
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			// get the pod backing the VirtualMachineInstance
			By("Getting the pod backing the VirtualMachineInstance")
			pods, err := virtClient.CoreV1().Pods(newVM.Namespace).List(tests.UnfinishedVMIPodSelector(firstVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			firstPod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(newVM.Namespace).Delete(firstPod.Name, &v12.DeleteOptions{})
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			// Wait on the VMI controller to create a new VirtualMachineInstance
			By("Waiting for a new VirtualMachineInstance to spawn")
			Eventually(func() bool {
				curVMI, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})

				// verify a new VirtualMachineInstance gets created for the VM after the Pod is deleted.
				if errors.IsNotFound(err) {
					return false
				} else if string(curVMI.UID) == string(firstVMI.UID) {
					return false
				} else if !curVMI.IsRunning() {
					return false
				}
				return true
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			// sanity check that the test ran correctly by
			// verifying a different Pod backs the VMI as well.
			By("Verifying a new pod backs the VMI")
			pods, err = virtClient.CoreV1().Pods(newVM.Namespace).List(tests.UnfinishedVMIPodSelector(curVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			pod := pods.Items[0]
			Expect(pod.Name).ToNot(Equal(firstPod.Name))
		})

		table.DescribeTable("[test_id:1525]should stop VirtualMachineInstance if running set to false", func(createTemplate vmiBuilder) {
			template, dv := createTemplate()
			defer deleteDataVolume(dv)
			vm := createVirtualMachine(false, template)
			vm = startVM(vm)
			vm = stopVM(vm)
		},
			table.Entry("with ContainerDisk", newVirtualMachineInstanceWithContainerDisk),
			table.Entry("[Serial]with OCS Filesystem Disk", newVirtualMachineInstanceWithOCSFileDisk),
			table.Entry("[Serial]with OCS Block Disk", newVirtualMachineInstanceWithOCSBlockDisk),
		)

		It("[test_id:1526]should start and stop VirtualMachineInstance multiple times", func() {
			vm := newVirtualMachine(false)
			// Start and stop VirtualMachineInstance multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				startVM(vm)
				stopVM(vm)
			}
		})

		It("[test_id:1527]should not update the VirtualMachineInstance spec if Running", func() {
			newVM := newVirtualMachine(true)

			Eventually(func() bool {
				newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return newVM.Status.Ready
			}, 360*time.Second, 1*time.Second).Should(BeTrue())

			By("Updating the VM template spec")
			newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedVM := newVM.DeepCopy()
			updatedVM.Spec.Template.Spec.Domain.Resources.Requests = v13.ResourceList{
				v13.ResourceMemory: resource.MustParse("4096Ki"),
			}
			updatedVM, err := virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
			Expect(err).ToNot(HaveOccurred())

			By("Expecting the old VirtualMachineInstance spec still running")
			vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory := newVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))

			By("Restarting the VM")
			newVM = stopVM(newVM)
			newVM = startVM(newVM)

			By("Expecting updated spec running")
			vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory = vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory = updatedVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))
		})

		It("[test_id:1528]should survive guest shutdown, multiple times", func() {
			By("Creating new VM, not running")
			newVM := newVirtualMachine(false)
			newVM = startVM(newVM)
			var vmi *v1.VirtualMachineInstance

			for i := 0; i < 3; i++ {
				currentVMI, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Getting the running VirtualMachineInstance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())

				By("Obtaining the serial console")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Guest shutdown")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: "The system is going down NOW!"},
				}, 240*time.Second)
				Expect(err).ToNot(HaveOccurred())

				By("waiting for the controller to replace the shut-down vmi with a new instance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					// Almost there, a new instance should be spawned soon
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred())
					// If the UID of the vmi changed we see the new vmi
					if vmi.UID != currentVMI.UID {
						return true
					}
					return false
				}, 240*time.Second, 1*time.Second).Should(BeTrue(), "No new VirtualMachineInstance instance showed up")

				By("VMI should run the VirtualMachineInstance again")
			}
		})

		It("[test_id:4645]should set the Ready condition on VM", func() {
			vm := newVirtualMachine(false)

			vmReadyConditionStatus := func() k8sv1.ConditionStatus {
				updatedVm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				cond := controller.NewVirtualMachineConditionManager().
					GetCondition(updatedVm, v1.VirtualMachineReady)
				if cond == nil {
					return ""
				}
				return cond.Status
			}

			Expect(vmReadyConditionStatus()).To(BeEmpty())

			startVM(vm)

			Eventually(vmReadyConditionStatus, 300*time.Second, 1*time.Second).
				Should(Equal(k8sv1.ConditionTrue))

			stopVM(vm)

			Eventually(vmReadyConditionStatus, 300*time.Second, 1*time.Second).
				Should(BeEmpty())
		})

		Context("Using virtctl interface", func() {
			It("[test_id:1529]should start a VirtualMachineInstance once", func() {
				By("getting a VM")
				newVM := newVirtualMachine(false)

				By("Invoking virtctl start")
				startCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", newVM.Namespace, newVM.Name)
				Expect(startCommand()).To(Succeed())

				By("Getting the status of the VM")
				Eventually(func() bool {
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return newVM.Status.Ready
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Getting the running VirtualMachineInstance")
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring a second invocation should fail")
				err = startCommand()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf(`Error starting VirtualMachine Operation cannot be fulfilled on virtualmachine.kubevirt.io "%s": VM is already running`, newVM.Name)))
			})

			It("[test_id:1530]should stop a VirtualMachineInstance once", func() {
				By("getting a VM")
				newVM := newVirtualMachine(true)

				By("Ensuring VM is running")
				Eventually(func() bool {
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return newVM.Status.Ready
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Invoking virtctl stop")
				stopCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", newVM.Namespace, newVM.Name)
				Expect(stopCommand()).To(Succeed())

				By("Ensuring VM is not running")
				Eventually(func() bool {
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return !newVM.Status.Ready && !newVM.Status.Created
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring the VirtualMachineInstance is removed")
				Eventually(func() error {
					_, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					// Expect a 404 error
					return err
				}, 240*time.Second, 1*time.Second).Should(HaveOccurred())

				By("Ensuring a second invocation should fail")
				err = stopCommand()
				Expect(err).ToNot(Succeed())
				Expect(err.Error()).To(Equal(fmt.Sprintf(`Error stopping VirtualMachine Operation cannot be fulfilled on virtualmachine.kubevirt.io "%s": VM is not running`, newVM.Name)))
			})

			It("[Serial][test_id:3007]Should force restart a VM with terminationGracePeriodSeconds>0", func() {

				By("getting a VM with high TerminationGracePeriod")
				newVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskFedora))
				gracePeriod := int64(600)
				newVMI.Spec.TerminationGracePeriodSeconds = &gracePeriod
				newVM := tests.NewRandomVirtualMachine(newVMI, true)
				_, err := virtClient.VirtualMachine(newVM.Namespace).Create(newVM)
				Expect(err).ToNot(HaveOccurred())
				waitForVMIStart(virtClient, newVMI)

				oldCreationTime := newVMI.ObjectMeta.CreationTimestamp
				oldVMIUuid := newVM.ObjectMeta.UID

				By("Invoking virtctl --force restart")
				forceRestart := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RESTART, "--namespace", newVM.Namespace, "--force", newVM.Name, "--grace-period=0")
				err = forceRestart()
				Expect(err).ToNot(HaveOccurred())

				zeroGracePeriod := int64(0)
				// Checks if the old VMI Pod still exists after force-restart command
				Eventually(func() string {
					pod, err := tests.GetRunningPodByLabel(string(oldVMIUuid), v1.CreatedByLabel, newVM.Namespace, "")
					if err != nil {
						return err.Error()
					}
					if pod.GetDeletionGracePeriodSeconds() == &zeroGracePeriod && pod.GetDeletionTimestamp() != nil {
						return "old VMI Pod still not deleted"
					}
					return ""
				}, 120*time.Second, 1*time.Second).Should(ContainSubstring("failed to find pod"))

				waitForNewVMI(virtClient, newVMI)

				By("Comparing the new CreationTimeStamp with the old one")
				newVMI, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(oldCreationTime).ToNot(Equal(newVMI.ObjectMeta.CreationTimestamp))
				Expect(oldVMIUuid).ToNot(Equal(newVMI.ObjectMeta.UID))
			})

			Context("Using RunStrategyAlways", func() {
				It("[test_id:3163]should stop a running VM", func() {
					By("creating a VM with RunStrategyAlways")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					By("Invoking virtctl stop")
					stopCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					Expect(stopCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is removed")
					Eventually(func() error {
						_, err = virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						// Expect a 404 error
						return err
					}, 240*time.Second, 1*time.Second).Should(HaveOccurred())

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyHalted))
					Expect(len(newVM.Status.StateChangeRequests)).To(Equal(0))
				})

				It("[test_id:3164]should restart a running VM", func() {
					By("creating a VM with RunStrategyAlways")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					By("Getting VM's UUID")
					virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					currentUUID := virtualMachine.UID

					By("Invoking virtctl restart")
					restartCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RESTART, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					Expect(restartCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(func() types.UID {
						nextVMI, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						if err != nil {
							// a 404 could happen normally while the VMI transitions
							if !errors.IsNotFound(err) {
								Expect(err).ToNot(HaveOccurred())
							}
							// If there's no VMI, just return the last known UUID
							return currentUUID
						}
						return nextVMI.UID
					}, 240*time.Second, 1*time.Second).ShouldNot(Equal(currentUUID))

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))

					// StateChangeRequest might still exist until the new VMI is created
					// But it must eventually be cleared
					Eventually(func() int {
						newVM, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(newVM.Status.StateChangeRequests)
					}, 240*time.Second, 1*time.Second).Should(Equal(0),
						"New VMI was created, but StateChangeRequest was never cleared")
				})

				It("[test_id:3165]should restart a succeeded VMI", func() {
					By("creating a VM with RunStategyRunning")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					vmi, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})

					expecter, err := tests.LoggedInCirrosExpecter(vmi)
					Expect(err).ToNot(HaveOccurred())
					defer expecter.Close()

					By("Issuing a poweroff command from inside VM")
					_, err = expecter.ExpectBatch([]expect.Batcher{
						&expect.BSnd{S: "sudo poweroff\n"},
					}, 10*time.Second)
					Expect(err).ToNot(HaveOccurred())

					By("Getting VM's UUID")
					virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					currentUUID := virtualMachine.UID

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(func() types.UID {
						nextVMI, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						if err != nil {
							// a 404 could happen normally while the VMI transitions
							if !errors.IsNotFound(err) {
								Expect(err).ToNot(HaveOccurred())
							}
							// If there's no VMI, just return the last known UUID
							return currentUUID
						}
						return nextVMI.UID
					}, 240*time.Second, 1*time.Second).ShouldNot(Equal(currentUUID))

				})

				It("[test_id:4119]should migrate a running VM", func() {
					nodes := tests.GetAllSchedulableNodes(virtClient)
					if len(nodes.Items) < 2 {
						Skip("Migration tests require at least 2 nodes")
					}
					By("creating a VM with RunStrategyAlways")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyAlways)

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					By("Invoking virtctl migrate")
					migrateCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_MIGRATE, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					Expect(migrateCommand()).To(Succeed())

					By("Ensuring the VirtualMachineInstance is migrated")
					Eventually(func() bool {
						nextVMI, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return nextVMI.Status.MigrationState != nil && nextVMI.Status.MigrationState.Completed
					}, 240*time.Second, 1*time.Second).Should(BeTrue())
				})
			})

			Context("Using RunStrategyRerunOnFailure", func() {
				It("[test_id:2186] should stop a running VM", func() {
					By("creating a VM with RunStrategyRerunOnFailure")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyRerunOnFailure)

					stopCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", virtualMachine.Namespace, virtualMachine.Name)

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					By("Invoking virtctl stop")
					err = stopCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance is removed")
					Eventually(func() error {
						_, err = virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						// Expect a 404 error
						return err
					}, 240*time.Second, 1*time.Second).Should(HaveOccurred())

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyHalted))
					By("Ensuring stateChangeRequests list is cleared")
					Expect(len(newVM.Status.StateChangeRequests)).To(Equal(0))
				})

				It("[test_id:2187] should restart a running VM", func() {
					By("creating a VM with RunStrategyRerunOnFailure")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyRerunOnFailure)

					restartCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RESTART, "--namespace", virtualMachine.Namespace, virtualMachine.Name)

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					By("Getting VM's UUID")
					virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					currentUUID := virtualMachine.UID

					By("Invoking virtctl restart")
					err = restartCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(func() types.UID {
						nextVMI, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						if err != nil {
							// a 404 could happen normally while the VMI transitions
							if !errors.IsNotFound(err) {
								Expect(err).ToNot(HaveOccurred())
							}
							// If there's no VMI, just return the last known UUID
							return currentUUID
						}
						return nextVMI.UID
					}, 240*time.Second, 1*time.Second).ShouldNot(Equal(currentUUID))

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyRerunOnFailure))

					By("Ensuring stateChangeRequests list gets cleared")
					// StateChangeRequest might still exist until the new VMI is created
					// But it must eventually be cleared
					Eventually(func() int {
						newVM, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(newVM.Status.StateChangeRequests)
					}, 240*time.Second, 1*time.Second).Should(Equal(0),
						"New VMI was created, but StateChangeRequest was never cleared")
				})

				It("[test_id:2188] should not remove a succeeded VMI", func() {
					By("creating a VM with RunStrategyRerunOnFailure")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyRerunOnFailure)

					By("Waiting for VMI to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					vmi, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})

					expecter, err := tests.LoggedInCirrosExpecter(vmi)
					Expect(err).ToNot(HaveOccurred())
					defer expecter.Close()

					By("Issuing a poweroff command from inside VM")
					_, err = expecter.ExpectBatch([]expect.Batcher{
						&expect.BSnd{S: "sudo poweroff\n"},
					}, 10*time.Second)
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance enters Succeeded phase")
					Eventually(func() v1.VirtualMachineInstancePhase {
						vmi, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})

						Expect(err).ToNot(HaveOccurred())
						return vmi.Status.Phase
					}, 240*time.Second, 1*time.Second).Should(Equal(v1.Succeeded))

					// At this point, explicitly test that a start command will delete an existing
					// VMI in the Succeeded phase.
					By("Invoking virtctl start")
					restartCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = restartCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for StartRequest to be cleared")
					Eventually(func() int {
						newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(newVM.Status.StateChangeRequests)
					}, 240*time.Second, 1*time.Second).Should(Equal(0), "StateChangeRequest was never cleared")

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())
				})
			})

			Context("Using RunStrategyHalted", func() {
				It("[test_id:2037] should start a stopped VM", func() {
					By("creating a VM with RunStrategyHalted")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyHalted)

					startCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = startCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyAlways))
					By("Ensuring stateChangeRequests list is cleared")
					Expect(len(newVM.Status.StateChangeRequests)).To(Equal(0))
				})
			})

			Context("Using RunStrategyManual", func() {
				It("[test_id:2036] should start", func() {
					By("creating a VM with RunStrategyManual")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = startCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyManual))
					By("Ensuring stateChangeRequests list is cleared")
					Expect(len(newVM.Status.StateChangeRequests)).To(Equal(0))
				})

				It("[test_id:2189] should stop", func() {
					By("creating a VM with RunStrategyManual")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = startCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					stopCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = stopCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance is removed")
					Eventually(func() bool {
						_, err = virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						return errors.IsNotFound(err)
					}, 240*time.Second, 1*time.Second).Should(BeTrue())

					By("Ensuring stateChangeRequests list is cleared")
					Eventually(func() bool {
						newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						if err != nil {
							return false
						}

						if newVM.Spec.RunStrategy == nil || *newVM.Spec.RunStrategy != v1.RunStrategyManual {
							return false
						}
						return len(newVM.Status.StateChangeRequests) == 0
					}, 30*time.Second, time.Second).Should(BeTrue())
				})

				It("[test_id:2035] should restart", func() {
					By("creating a VM with RunStrategyManual")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					stopCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					restartCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RESTART, "--namespace", virtualMachine.Namespace, virtualMachine.Name)

					By("Invoking virtctl restart should fail")
					err = restartCommand()
					Expect(err).To(HaveOccurred())

					By("Invoking virtctl start")
					err = startCommand()
					Expect(err).NotTo(HaveOccurred())

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 240*time.Second, 1*time.Second).Should(BeTrue())

					By("Invoking virtctl stop")
					err = stopCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance is stopped")
					Eventually(func() bool {
						vm, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						if err != nil {
							Expect(err).ToNot(HaveOccurred())
						}
						return vm.Status.Created
					}, 240*time.Second, 1*time.Second).Should(BeFalse())

					By("Waiting state change request to clear for stopped VM")
					Eventually(func() int {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(virtualMachine.Status.StateChangeRequests)
					}, 240*time.Second, 1*time.Second).Should(Equal(0))

					By("Invoking virtctl start")
					err = startCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					By("Getting VM's UUID")
					virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					currentUUID := virtualMachine.UID

					By("Invoking virtctl restart")
					err = restartCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance is restarted")
					Eventually(func() types.UID {
						nextVMI, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						if err != nil {
							// a 404 could happen normally while the VMI transitions
							if !errors.IsNotFound(err) {
								Expect(err).ToNot(HaveOccurred())
							}
							// If there's no VMI, just return the last known UUID
							return currentUUID
						}
						return nextVMI.UID
					}, 240*time.Second, 1*time.Second).ShouldNot(Equal(currentUUID))

					newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(newVM.Spec.RunStrategy).ToNot(BeNil())
					Expect(*newVM.Spec.RunStrategy).To(Equal(v1.RunStrategyManual))

					By("Ensuring stateChangeRequests list gets cleared")
					// StateChangeRequest might still exist until the new VMI is created
					// But it must eventually be cleared
					Eventually(func() int {
						newVM, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(newVM.Status.StateChangeRequests)
					}, 240*time.Second, 1*time.Second).Should(Equal(0),
						"New VMI was created, but StateChangeRequest was never cleared")
				})

				It("[test_id:2190] should not remove a succeeded VMI", func() {
					By("creating a VM with RunStrategyManual")
					virtualMachine := newVirtualMachineWithRunStrategy(v1.RunStrategyManual)

					startCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = startCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())

					vmi, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					expecter, err := tests.LoggedInCirrosExpecter(vmi)
					Expect(err).ToNot(HaveOccurred())
					defer expecter.Close()

					By("Issuing a poweroff command from inside VM")
					_, err = expecter.ExpectBatch([]expect.Batcher{
						&expect.BSnd{S: "sudo poweroff\n"},
					}, 10*time.Second)
					Expect(err).ToNot(HaveOccurred())

					By("Ensuring the VirtualMachineInstance enters Succeeded phase")
					Eventually(func() v1.VirtualMachineInstancePhase {
						vmi, err := virtClient.VirtualMachineInstance(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})

						Expect(err).ToNot(HaveOccurred())
						return vmi.Status.Phase
					}, 240*time.Second, 1*time.Second).Should(Equal(v1.Succeeded))

					// At this point, explicitly test that a start command will delete an existing
					// VMI in the Succeeded phase.
					By("Invoking virtctl start")
					restartCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", virtualMachine.Namespace, virtualMachine.Name)
					err = restartCommand()
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for StartRequest to be cleared")
					Eventually(func() int {
						newVM, err := virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return len(newVM.Status.StateChangeRequests)
					}, 240*time.Second, 1*time.Second).Should(Equal(0), "StateChangeRequest was never cleared")

					By("Waiting for VM to be ready")
					Eventually(func() bool {
						virtualMachine, err = virtClient.VirtualMachine(virtualMachine.Namespace).Get(virtualMachine.Name, &v12.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return virtualMachine.Status.Ready
					}, 360*time.Second, 1*time.Second).Should(BeTrue())
				})
			})
		})

		Context("VM rename", func() {
			var vm1 *v1.VirtualMachine

			BeforeEach(func() {
				vm1 = newVirtualMachine(false)
			})

			It("[test_id:4646]should rename a stopped VM only once", func() {
				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm1.Name, vm1.Name+"new",
					"--namespace", vm1.Namespace)
				Expect(renameCommand()).To(Succeed())
				Expect(renameCommand()).ToNot(Succeed())
			})

			It("[test_id:4647]should rename a stopped VM", func() {
				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm1.Name, vm1.Name+"new",
					"--namespace", vm1.Namespace)
				Expect(renameCommand()).To(Succeed())
			})

			It("[test_id:4648]should reject renaming a running VM", func() {
				vm2 := newVirtualMachine(true)

				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm2.Name, vm2.Name+"new",
					"--namespace", vm2.Namespace)
				Expect(renameCommand()).ToNot(Succeed())
			})

			It("[test_id:4649]should reject renaming a VM to the same name", func() {
				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm1.Name, vm1.Name,
					"--namespace", vm1.Namespace)
				Expect(renameCommand()).ToNot(Succeed())
			})

			It("[test_id:4650]should reject renaming a VM with an empty name", func() {
				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm1.Name, "",
					"--namespace", vm1.Namespace)
				Expect(renameCommand()).ToNot(Succeed())
			})

			It("[test_id:4651]should reject renaming a VM with invalid name", func() {
				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm1.Name, "invalid name <>?:;",
					"--namespace", vm1.Namespace)
				Expect(renameCommand()).ToNot(Succeed())
			})

			It("[test_id:4652]should reject renaming a VM if the new name is taken", func() {
				vm2 := newVirtualMachine(true)

				renameCommand := tests.NewRepeatableVirtctlCommand(vm.COMMAND_RENAME, vm1.Name, vm2.Name,
					"--namespace", vm1.Namespace)
				Expect(renameCommand()).ToNot(Succeed())
			})
		})
	})

	Context("[rfe_id:273]with oc/kubectl", func() {
		var vmi *v1.VirtualMachineInstance
		var err error
		var vmJson string

		var k8sClient string
		var workDir string

		var vmRunningRe *regexp.Regexp

		BeforeEach(func() {
			k8sClient = tests.GetK8sCmdClient()
			tests.SkipIfNoCmd(k8sClient)
			workDir, err = ioutil.TempDir("", tests.TempDirPrefix+"-")
			Expect(err).ToNot(HaveOccurred())

			// By default "." does not match newline: "Phase" and "Running" only match if on same line.
			vmRunningRe = regexp.MustCompile("Phase.*Running")
		})

		AfterEach(func() {
			if workDir != "" {
				err = os.RemoveAll(workDir)
				Expect(err).ToNot(HaveOccurred())
				workDir = ""
			}
		})

		It("[test_id:243][posneg:negative]should create VM only once", func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vm := tests.NewRandomVirtualMachine(vmi, true)

			vmJson, err = tests.GenerateVMJson(vm, workDir)
			Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Verifying VM is created")
			newVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "New VM was not created")
			Expect(newVM.Name).To(Equal(vm.Name), "New VM was not created")

			By("Creating the VM again")
			_, stdErr, err := tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).To(HaveOccurred())

			Expect(strings.HasPrefix(stdErr, "Error from server (AlreadyExists): error when creating")).To(BeTrue(), "command should error when creating VM second time")
		})

		It("[test_id:299]should create VM via command line", func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			vm := tests.NewRandomVirtualMachine(vmi, true)

			vmJson, err = tests.GenerateVMJson(vm, workDir)
			Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

			By("Creating VM using k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			waitForVMIStart(virtClient, vmi)

			By("Listing running pods")
			stdout, _, err := tests.RunCommand(k8sClient, "get", "pods")
			Expect(err).ToNot(HaveOccurred())

			By("Ensuring pod is running")
			expectedPodName := getExpectedPodName(vm)
			podRunningRe, err := regexp.Compile(fmt.Sprintf("%s.*Running", expectedPodName))
			Expect(err).ToNot(HaveOccurred())

			Expect(podRunningRe.FindString(stdout)).ToNot(Equal(""), "Pod is not Running")

			By("Checking that VM is running")
			stdout, _, err = tests.RunCommand(k8sClient, "describe", "vmis", vm.GetName())
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")
		})

		It("[test_id:264]should create and delete via command line", func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			thisVm := tests.NewRandomVirtualMachine(vmi, false)

			vmJson, err = tests.GenerateVMJson(thisVm, workDir)
			Expect(err).ToNot(HaveOccurred(), "Cannot generate VM's manifest")

			By("Creating VM using k8s client binary")
			_, _, err := tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Invoking virtctl start")
			virtctl := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", thisVm.Namespace, thisVm.Name)
			err = virtctl()
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			waitForVMIStart(virtClient, vmi)

			By("Checking that VM is running")
			stdout, _, err := tests.RunCommand(k8sClient, "describe", "vmis", thisVm.GetName())
			Expect(err).ToNot(HaveOccurred())

			Expect(vmRunningRe.FindString(stdout)).ToNot(Equal(""), "VMI is not Running")

			By("Deleting VM using k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "delete", "vm", thisVm.GetName())
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the VM gets deleted")
			waitForResourceDeletion(k8sClient, "vms", thisVm.GetName())

			By("Verifying pod gets deleted")
			expectedPodName := getExpectedPodName(thisVm)
			waitForResourceDeletion(k8sClient, "pods", expectedPodName)
		})

		It("[test_id:232]should create same manifest twice via command line", func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			thisVm := tests.NewRandomVirtualMachine(vmi, true)

			vmJson, err = tests.GenerateVMJson(thisVm, workDir)
			Expect(err).ToNot(HaveOccurred(), "Cannot generate VM's manifest")

			By("Creating VM using k8s client binary")
			_, _, err := tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			waitForVMIStart(virtClient, vmi)

			By("Deleting VM using k8s client binary")
			_, _, err = tests.RunCommand(k8sClient, "delete", "vm", thisVm.GetName())
			Expect(err).ToNot(HaveOccurred())

			By("Verifying the VM gets deleted")
			waitForResourceDeletion(k8sClient, "vms", thisVm.GetName())

			By("Creating same VM using k8s client binary and same manifest")
			_, _, err = tests.RunCommand(k8sClient, "create", "-f", vmJson)
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to start")
			waitForVMIStart(virtClient, vmi)
		})

		It("[test_id:233][posneg:negative]should fail when deleting nonexistent VM", func() {
			vmi := tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

			By("Creating VM with DataVolumeTemplate entry with k8s client binary")
			_, stdErr, err := tests.RunCommand(k8sClient, "delete", "vm", vmi.Name)
			Expect(err).To(HaveOccurred())
			Expect(strings.HasPrefix(stdErr, "Error from server (NotFound): virtualmachines.kubevirt.io")).To(BeTrue(), "should fail when deleting non existent VM")
		})

		Context("as ordinary OCP user trough test service account", func() {
			var testUser string

			BeforeEach(func() {
				testUser = "testuser-" + uuid.NewRandom().String()
			})

			Context("should succeed with right rights", func() {
				BeforeEach(func() {
					// kubectl doesn't have "adm" subcommand -- only oc does
					tests.SkipIfNoCmd("oc")
					By("Ensuring the cluster has new test serviceaccount")
					stdOut, stdErr, err := tests.RunCommand(k8sClient, "create", "serviceaccount", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

					By("Ensuring user has the admin rights for the test namespace project")
					// This simulates the ordinary user as an admin in this project
					stdOut, stdErr, err = tests.RunCommand(k8sClient, "adm", "policy", "add-role-to-user", "admin", fmt.Sprintf("system:serviceaccount:%s:%s", tests.NamespaceTestDefault, testUser))
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				AfterEach(func() {
					stdOut, stdErr, err := tests.RunCommand(k8sClient, "adm", "policy", "remove-role-from-user", "admin", fmt.Sprintf("system:serviceaccount:%s:%s", tests.NamespaceTestDefault, testUser))
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)

					stdOut, stdErr, err = tests.RunCommand(k8sClient, "delete", "serviceaccount", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				It("[test_id:2839]should create VM via command line", func() {
					vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
					vm := tests.NewRandomVirtualMachine(vmi, true)

					vmJson, err = tests.GenerateVMJson(vm, workDir)
					Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

					By("Checking VM creation permission using k8s client binary")
					// It might take time for the role to propagate
					Eventually(func() string {
						stdOut, _, _ := tests.RunCommand(k8sClient, "auth", "can-i", "create", "vms", "--as", testUser)
						return strings.TrimSpace(stdOut)
					}, 10*time.Second, 1*time.Second).Should(Equal("yes"), fmt.Sprintf("test account '%s' was never granted permission to create a VM", testUser))
				})
			})

			Context("should fail without right rights", func() {
				BeforeEach(func() {
					By("Ensuring the cluster has new test serviceaccount")
					stdOut, stdErr, err := tests.RunCommandWithNS(tests.NamespaceTestDefault, k8sClient, "create", "serviceaccount", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				AfterEach(func() {
					stdOut, stdErr, err := tests.RunCommandWithNS(tests.NamespaceTestDefault, k8sClient, "delete", "serviceaccount", testUser)
					Expect(err).ToNot(HaveOccurred(), "ERR: %s", stdOut+stdErr)
				})

				It("[test_id:2914]should create VM via command line", func() {
					vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
					vm := tests.NewRandomVirtualMachine(vmi, true)

					vmJson, err = tests.GenerateVMJson(vm, workDir)
					Expect(err).ToNot(HaveOccurred(), "Cannot generate VMs manifest")

					By("Checking VM creation permission using k8s client binary")
					stdOut, _, err := tests.RunCommand(k8sClient, "auth", "can-i", "create", "vms", "--as", testUser)
					// non-zero exit code
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("exit status 1"))
					Expect(strings.TrimSpace(stdOut)).To(Equal("no"))
				})
			})
		})

	})

	Context("VM rename", func() {
		var (
			cli kubecli.VirtualMachineInterface
		)

		BeforeEach(func() {
			cli = virtClient.VirtualMachine(tests.NamespaceTestDefault)
		})

		Context("VM update", func() {
			var (
				vm1 *v1.VirtualMachine
			)

			BeforeEach(func() {
				vm1 = tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				cli.Create(vm1)
			})

			It("[test_id:4654]should fail if the new name is already taken", func() {
				vm2 := tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
				cli.Create(vm2)

				err := cli.Rename(vm1.Name, &v1.RenameOptions{NewName: vm2.Name})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("name already exists"))
			})

			It("[test_id:4655]should fail if the new name is empty", func() {
				err := cli.Rename(vm1.Name, &v1.RenameOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Please provide a new name for the VM"))
			})

			It("[test_id:4656]should fail if the new name is invalid", func() {
				err := cli.Rename(vm1.Name, &v1.RenameOptions{NewName: "invalid name <>?:;"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("The VM's new name is not valid"))
			})

			It("[test_id:4657]should fail if the new name is identical to the current name", func() {
				err := cli.Rename(vm1.Name, &v1.RenameOptions{NewName: vm1.Name})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("identical"))
			})

			It("[test_id:4658]should fail if the VM is running", func() {
				err := cli.Start(vm1.Name)
				Expect(err).ToNot(HaveOccurred())

				err = cli.Rename(vm1.Name, &v1.RenameOptions{NewName: vm1.Name + "new"})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("running"))
			})

			It("[test_id:4659]should succeed", func() {
				err := cli.Rename(vm1.Name, &v1.RenameOptions{NewName: vm1.Name + "new"})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() error {
					_, err := cli.Get(vm1.Name+"new", &v12.GetOptions{})

					return err
				}, 10*time.Second, 1*time.Second).Should(BeNil())

				_, err = cli.Get(vm1.Name, &v12.GetOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("not found"))
			})
		})
	})
})

func getExpectedPodName(vm *v1.VirtualMachine) string {
	maxNameLength := 63
	podNamePrefix := "virt-launcher-"
	podGeneratedSuffixLen := 5
	charCountFromName := maxNameLength - len(podNamePrefix) - podGeneratedSuffixLen
	expectedPodName := fmt.Sprintf(fmt.Sprintf("virt-launcher-%%.%ds", charCountFromName), vm.GetName())
	return expectedPodName
}

func NewRandomVirtualMachineWithRunStrategy(vmi *v1.VirtualMachineInstance, runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	vm := tests.NewRandomVirtualMachine(vmi, false)
	vm.Spec.Running = nil
	vm.Spec.RunStrategy = &runStrategy
	return vm
}

func waitForVMIStart(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	Eventually(func() v1.VirtualMachineInstancePhase {
		newVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.GetName(), &v12.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			return v1.Unknown
		}
		return newVMI.Status.Phase
	}, 120*time.Second, 1*time.Second).Should(Equal(v1.Running), "New VMI was not created")
}

func waitForNewVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	Eventually(func() bool {
		newVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.GetName(), &v12.GetOptions{})
		if err != nil {
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			return false
		}
		return (newVMI.Status.Phase == v1.Scheduling) || (newVMI.Status.Phase == v1.Running)
	}, 120*time.Second, 1*time.Second).Should(BeTrue(), "New VMI was not created")
}

func waitForResourceDeletion(k8sClient string, resourceType string, resourceName string) {
	Eventually(func() bool {
		stdout, _, err := tests.RunCommand(k8sClient, "get", resourceType)
		Expect(err).ToNot(HaveOccurred())
		return strings.Contains(stdout, resourceName)
	}, 120*time.Second, 1*time.Second).Should(BeFalse(), "VM was not deleted")
}
