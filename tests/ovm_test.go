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
	"flag"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"time"

	"github.com/google/goexpect"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/offlinevm"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("OfflineVirtualMachine", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("An invalid OfflineVirtualMachine given", func() {

		It("should be rejected on POST", func() {
			vmImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMWithEphemeralDiskAndUserdata(vmImage, "echo Hi\n")
			newOVM := NewRandomOfflineVirtualMachine(template, false)
			newOVM.TypeMeta = v12.TypeMeta{
				APIVersion: v1.GroupVersion.String(),
				Kind:       "OfflineVirtualMachine",
			}

			jsonBytes, err := json.Marshal(newOVM)
			Expect(err).To(BeNil())

			// change the name of a required field (like domain) so validation will fail
			jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

			result := virtClient.RestClient().Post().Resource("offlinevirtualmachines").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		})
		It("should reject POST if validation webhoook deems the spec is invalid", func() {
			vmImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMWithEphemeralDiskAndUserdata(vmImage, "echo Hi\n")
			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			template.Spec.Domain.Devices.Disks = append(template.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			newOVM := NewRandomOfflineVirtualMachine(template, false)
			newOVM.TypeMeta = v12.TypeMeta{
				APIVersion: v1.GroupVersion.String(),
				Kind:       "OfflineVirtualMachine",
			}

			result := virtClient.RestClient().Post().Resource("offlinevirtualmachines").Namespace(tests.NamespaceTestDefault).Body(newOVM).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

			reviewResponse := &v12.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).To(BeNil())

			Expect(len(reviewResponse.Details.Causes)).To(Equal(1))
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[2].volumeName"))
		})
	})

	Context("A valid OfflineVirtualMachine given", func() {

		newOfflineVirtualMachine := func(running bool) *v1.OfflineVirtualMachine {
			vmImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMWithEphemeralDiskAndUserdata(vmImage, "echo Hi\n")

			var newOVM *v1.OfflineVirtualMachine
			var err error

			newOVM = NewRandomOfflineVirtualMachine(template, running)
			Eventually(func() int {
				ovms, err := virtClient.OfflineVirtualMachine(newOVM.Namespace).List(&v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(ovms.Items)
			}, 300*time.Second, 2*time.Second).Should(BeZero())

			Eventually(func() error {
				newOVM, err = virtClient.OfflineVirtualMachine(tests.NamespaceTestDefault).Create(newOVM)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			returnedOVM, err := virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return returnedOVM
		}

		startOVM := func(ovm *v1.OfflineVirtualMachine) *v1.OfflineVirtualMachine {
			By("Starting the VM")
			var err error

			updatedOVM, err := virtClient.OfflineVirtualMachine(ovm.Namespace).Get(ovm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedOVM = updatedOVM.DeepCopy()
			updatedOVM.Spec.Running = true
			Eventually(func() error {
				updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Update(updatedOVM)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			// Observe the VM created
			Eventually(func() error {
				_, err = virtClient.VM(updatedOVM.Namespace).Get(updatedOVM.Name, v12.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("OVM has the running condition")
			Eventually(func() bool {
				updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Get(updatedOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedOVM.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			return updatedOVM
		}

		stopOVM := func(ovm *v1.OfflineVirtualMachine) *v1.OfflineVirtualMachine {
			By("Stopping the VM")
			var err error

			updatedOVM, err := virtClient.OfflineVirtualMachine(ovm.Namespace).Get(ovm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedOVM = updatedOVM.DeepCopy()
			updatedOVM.Spec.Running = false
			Eventually(func() error {
				updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Update(updatedOVM)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			// Observe the VM deleted
			Eventually(func() bool {
				_, err = virtClient.VM(updatedOVM.Namespace).Get(updatedOVM.Name, v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			By("OVM has not the running condition")
			Eventually(func() bool {
				updatedOVM, err = virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Get(updatedOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return updatedOVM.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeFalse())

			return updatedOVM
		}

		It("should update OfflineVirtualMachine once VMs are up", func() {
			newOVM := newOfflineVirtualMachine(true)
			Eventually(func() bool {
				ovm, err := virtClient.OfflineVirtualMachine(tests.NamespaceTestDefault).Get(newOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return ovm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("should remove VM once the OVM is marked for deletion", func() {
			newOVM := newOfflineVirtualMachine(true)
			// Create a offlinevm with vm
			// Delete it
			Expect(virtClient.OfflineVirtualMachine(newOVM.Namespace).Delete(newOVM.Name, &v12.DeleteOptions{})).To(Succeed())
			// Wait until VMs are gone
			Eventually(func() int {
				vms, err := virtClient.VM(newOVM.Namespace).List(v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(vms.Items)
			}, 300*time.Second, 2*time.Second).Should(BeZero())
		})

		It("should remove owner references on the VM if it is orphan deleted", func() {
			newOVM := newOfflineVirtualMachine(true)

			Eventually(func() []v12.OwnerReference {
				// Check for owner reference
				vm, _ := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
				return vm.OwnerReferences
			}, 300*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			// Delete it
			orphanPolicy := v12.DeletePropagationOrphan
			Expect(virtClient.OfflineVirtualMachine(newOVM.Namespace).
				Delete(newOVM.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
			// Wait until the offlinevm is deleted
			Eventually(func() bool {
				_, err := virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			vm, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vm.OwnerReferences).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("should recreate VM if it gets deleted", func() {
			newOVM := newOfflineVirtualMachine(true)
			// Delete the VM
			Eventually(func() error {
				return virtClient.VM(newOVM.Namespace).Delete(newOVM.Name, &v12.DeleteOptions{})
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			Eventually(func() bool {
				_, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
				if errors.IsNotFound(err) {
					return false
				}
				return true
			}, 120*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("should recreate VM if the VM's pod gets deleted", func() {
			var firstVM *v1.VirtualMachine
			var curVM *v1.VirtualMachine
			var err error

			By("Creating a new OVM")
			newOVM := newOfflineVirtualMachine(true)

			// wait for a running VM.
			By("Waiting for the OVM's VM to start")
			Eventually(func() error {
				firstVM, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
				if err != nil {
					return err
				}
				if !firstVM.IsRunning() {
					return fmt.Errorf("vm still isn't running")
				}
				return nil
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			// get the pod backing the VM
			By("Getting the pod backing the VM")
			pods, err := virtClient.CoreV1().Pods(newOVM.Namespace).List(tests.UnfinishedVMPodSelector(firstVM))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			firstPod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VM's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(newOVM.Namespace).Delete(firstPod.Name, &v12.DeleteOptions{})
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			// Wait on the OVM controller to create a new VM
			By("Waiting for a new VM to spawn")
			Eventually(func() bool {
				curVM, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})

				// verify a new VM gets created for the OVM after the Pod is deleted.
				if errors.IsNotFound(err) {
					return false
				} else if string(curVM.UID) == string(firstVM.UID) {
					return false
				} else if !curVM.IsRunning() {
					return false
				}
				return true
			}, 120*time.Second, 1*time.Second).Should(BeTrue())

			// sanity check that the test ran correctly by
			// verifying a different Pod backs the OVM as well.
			By("Verifying a new pod backs the OVM")
			pods, err = virtClient.CoreV1().Pods(newOVM.Namespace).List(tests.UnfinishedVMPodSelector(curVM))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			pod := pods.Items[0]
			Expect(pod.Name).ToNot(Equal(firstPod.Name))
		})

		It("should stop VM if running set to false", func() {

			currOVM := newOfflineVirtualMachine(false)
			currOVM = startOVM(currOVM)
			currOVM = stopOVM(currOVM)

		})

		It("should start and stop VM multiple times", func() {
			var currOVM *v1.OfflineVirtualMachine

			currOVM = newOfflineVirtualMachine(false)

			// Start and stop VM multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				startOVM(currOVM)
				stopOVM(currOVM)
			}
		})

		It("should not update the VM spec if Running", func() {
			newOVM := newOfflineVirtualMachine(true)

			Eventually(func() bool {
				newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return newOVM.Status.Ready
			}, 360*time.Second, 1*time.Second).Should(BeTrue())

			By("Updating the OVM template spec")
			newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedOVM := newOVM.DeepCopy()
			updatedOVM.Spec.Template.Spec.Domain.Resources.Requests = v13.ResourceList{
				v13.ResourceMemory: resource.MustParse("4096Ki"),
			}
			updatedOVM, err := virtClient.OfflineVirtualMachine(updatedOVM.Namespace).Update(updatedOVM)
			Expect(err).ToNot(HaveOccurred())

			By("Expecting the old VM spec still running")
			vm, err := virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmMemory := vm.Spec.Domain.Resources.Requests.Memory()
			ovmMemory := newOVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmMemory.Cmp(*ovmMemory)).To(Equal(0))

			By("Restarting the OVM")
			newOVM = stopOVM(newOVM)
			newOVM = startOVM(newOVM)

			By("Expecting updated spec running")
			vm, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmMemory = vm.Spec.Domain.Resources.Requests.Memory()
			ovmMemory = updatedOVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmMemory.Cmp(*ovmMemory)).To(Equal(0))
		})

		It("should survive guest shutdown, multiple times", func() {
			By("Creating new OVM, not running")
			newOVM := newOfflineVirtualMachine(false)
			newOVM = startOVM(newOVM)
			var vm *v1.VirtualMachine
			var err error

			for i := 0; i < 3; i++ {
				By("Getting the running VM")
				Eventually(func() bool {
					vm, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase == v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())

				By("Obtaining the serial console")
				expecter, err := tests.LoggedInCirrosExpecter(vm)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				By("Guest shutdown")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: "The system is going down NOW!"},
				}, 240*time.Second)
				Expect(err).ToNot(HaveOccurred())

				By("Testing the VM is not running")
				Eventually(func() bool {
					vm, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase != v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())

				By("OVM should run the VM again")
			}
		})

		Context("Using virtctl interface", func() {
			It("should start a VM once", func() {
				var vm *v1.VirtualMachine
				var err error
				By("getting an OVM")
				newOVM := newOfflineVirtualMachine(false)

				By("Invoking virtctl start")
				virtctl := tests.NewRepeatableVirtctlCommand(offlinevm.COMMAND_START, "--namespace", newOVM.Namespace, newOVM.Name)

				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting the status of the OVM")
				Eventually(func() bool {
					newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return newOVM.Status.Ready
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Getting the running VM")
				Eventually(func() bool {
					vm, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Phase == v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring a second invocation should fail")
				err = virtctl()
				Expect(err).To(HaveOccurred())
			})

			It("should stop a VM once", func() {
				var err error
				By("getting an OVM")
				newOVM := newOfflineVirtualMachine(true)

				By("Invoking virtctl stop")
				virtctl := tests.NewRepeatableVirtctlCommand(offlinevm.COMMAND_STOP, "--namespace", newOVM.Namespace, newOVM.Name)

				By("Ensuring OVM is running")
				Eventually(func() bool {
					newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return newOVM.Status.Ready
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring OVM is not running")
				Eventually(func() bool {
					newOVM, err = virtClient.OfflineVirtualMachine(newOVM.Namespace).Get(newOVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return !newOVM.Status.Ready && !newOVM.Status.Created
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring the VM is removed")
				Eventually(func() error {
					_, err = virtClient.VM(newOVM.Namespace).Get(newOVM.Name, v12.GetOptions{})
					// Expect a 404 error
					return err
				}, 240*time.Second, 1*time.Second).Should(HaveOccurred())

				By("Ensuring a second invocation should fail")
				err = virtctl()
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

// NewRandomOfflineVirtualMachine creates new OfflineVirtualMachine
func NewRandomOfflineVirtualMachine(vm *v1.VirtualMachine, running bool) *v1.OfflineVirtualMachine {
	name := vm.Name
	namespace := vm.Namespace
	ovm := &v1.OfflineVirtualMachine{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.OfflineVirtualMachineSpec{
			Running: running,
			Template: &v1.VMTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Labels:    map[string]string{"name": name},
					Name:      name,
					Namespace: namespace,
				},
				Spec: vm.Spec,
			},
		},
	}
	return ovm
}
