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
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VirtualMachine", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("An invalid VirtualMachine given", func() {

		It("should be rejected on POST", func() {
			vmiImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			newVM := NewRandomVirtualMachine(template, false)
			newVM.TypeMeta = v12.TypeMeta{
				APIVersion: v1.GroupVersion.String(),
				Kind:       "VirtualMachine",
			}

			jsonBytes, err := json.Marshal(newVM)
			Expect(err).To(BeNil(), "VM should be able to marshal to json")

			// change the name of a required field (like domain) so validation will fail
			jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "Reject should happen by status code 422")

		})
		It("should reject POST if validation webhoook deems the spec is invalid", func() {
			vmiImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			template.Spec.Domain.Devices.Disks = append(template.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			newVM := NewRandomVirtualMachine(template, false)
			newVM.TypeMeta = v12.TypeMeta{
				APIVersion: v1.GroupVersion.String(),
				Kind:       "VirtualMachine",
			}

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(newVM).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "Reject should happen by status code 422")

			reviewResponse := &v12.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).To(BeNil(), "Response should be able to unmarshal from JSON")

			Expect(len(reviewResponse.Details.Causes)).To(Equal(1), "There should be only one reason to rejection")
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.template.spec.domain.devices.disks[2].volumeName"))
		})
	})

	Context("A valid VirtualMachine given", func() {

		newVirtualMachine := func(running bool) *v1.VirtualMachine {
			vmiImage := tests.RegistryDiskFor(tests.RegistryDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")

			var newVM *v1.VirtualMachine
			var err error

			newVM = NewRandomVirtualMachine(template, running)

			newVM, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(newVM)
			Expect(err).ToNot(HaveOccurred(), "Should create VM with client")

			return newVM
		}

		startVM := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Starting the VirtualMachineInstance")

			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should be able to retrieve the VM")
				updatedVM.Spec.Running = true
				_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				return err
			}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred(), "Should be able to update VM")

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should be able to retrieve VM")

			// Observe the VirtualMachineInstance created
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				return err
			}, 30*time.Second, 2*time.Second).Should(Succeed(), "VMI should come up")

			By("VM has the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve the VM")
				return vm.Status.Ready
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "VM should become ready, usually it takes less then 120s")

			return updatedVM
		}

		stopVM := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Stopping the VirtualMachineInstance")

			Eventually(func() error {
				updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
				updatedVM.Spec.Running = false
				_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
				return err
			}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred(), "Should update VM successfully")

			updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")

			// Observe the VirtualMachineInstance deleted
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
				if vmi.GetObjectMeta().GetDeletionTimestamp() != nil {
					return true
				}
				return false
			}, 30*time.Second, 2*time.Second).Should(BeTrue(), "The vmi did not disappear")

			By("VM has not the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
				return vm.Status.Ready
			}, 30*time.Second, 2*time.Second).Should(BeFalse(), "VM should became not ready")

			return updatedVM
		}

		It("should update VirtualMachine once VMIs are up", func() {
			newVM := newVirtualMachine(true)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
				return vm.Status.Ready
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "VM should become ready in less than 120s")
		})

		It("should remove VirtualMachineInstance once the VM is marked for deletion", func() {
			newVM := newVirtualMachine(true)
			// Create a offlinevmi with vmi
			// Delete it
			Expect(virtClient.VirtualMachine(newVM.Namespace).Delete(newVM.Name, &v12.DeleteOptions{})).To(Succeed(), "Should be able to delete VM")
			// Wait until VMIs are gone
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get the VMI")
				return vmi.GetObjectMeta().GetDeletionTimestamp() != nil
			}, 30*time.Second, 2*time.Second).Should(BeTrue(), "The VirtualMachineInstance is marked for deletion")
		})

		It("should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {
			newVM := newVirtualMachine(true)

			Eventually(func() []v12.OwnerReference {
				// Check for owner reference
				vmi, _ := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				return vmi.OwnerReferences
			}, 30*time.Second, 2*time.Second).ShouldNot(BeEmpty(), "Should have the ownerReference")

			// Delete it
			orphanPolicy := v12.DeletePropagationOrphan
			Expect(virtClient.VirtualMachine(newVM.Namespace).
				Delete(newVM.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed(), "Should be able to delete VM")
			// Wait until the offlinevmi is deleted
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get the VM")
				return vm.GetObjectMeta().GetDeletionTimestamp() != nil
			}, 30*time.Second, 2*time.Second).Should(BeTrue(), "VM should disappear")

			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
				return len(vmi.OwnerReferences) == 0
			}, 30*time.Second, 2*time.Second).Should(BeTrue(), "VM should not have ownerReference")

		})

		It("should recreate VirtualMachineInstance if it gets deleted", func() {
			newVM := startVM(newVirtualMachine(false))

			currentVMI, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should retrieve VMI")

			Expect(virtClient.VirtualMachineInstance(newVM.Namespace).Delete(newVM.Name, &v12.DeleteOptions{})).To(Succeed(), "Should delete VM")

			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return false
				}
				if vmi.UID != currentVMI.UID {
					return true
				}
				return false
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "VMI should appear again")
		})

		It("should recreate VirtualMachineInstance if the VirtualMachineInstance's pod gets deleted", func() {
			var firstVMI *v1.VirtualMachineInstance
			var curVMI *v1.VirtualMachineInstance
			var err error

			By("Creating a new VMI")
			newVM := newVirtualMachine(true)

			// wait for a running VirtualMachineInstance.
			By("Waiting for the VMI's VirtualMachineInstance to start")
			Eventually(func() error {
				firstVMI, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				if err != nil {
					return err
				}
				if !firstVMI.IsRunning() {
					return fmt.Errorf("vmi still isn't running")
				}
				return nil
			}, 120*time.Second, 5*time.Second).Should(Succeed(), "VMI should have started")

			// get the pod backing the VirtualMachineInstance
			By("Getting the pod backing the VirtualMachineInstance")
			pods, err := virtClient.CoreV1().Pods(newVM.Namespace).List(tests.UnfinishedVMIPodSelector(firstVMI))
			Expect(err).ToNot(HaveOccurred(), "Should list the pods")
			Expect(len(pods.Items)).To(Equal(1), "There should be only one VMI pod")
			firstPod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(newVM.Namespace).Delete(firstPod.Name, &v12.DeleteOptions{})
			}, 120*time.Second, 5*time.Second).Should(Succeed(), "Should delete the VMI pod")

			// Wait on the VMI controller to create a new VirtualMachineInstance
			By("Waiting for a new VirtualMachineInstance to spawn")
			Eventually(func() bool {
				curVMI, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})

				// verify a new VirtualMachineInstance gets created for the VMI after the Pod is deleted.
				if errors.IsNotFound(err) {
					return false
				} else if string(curVMI.UID) == string(firstVMI.UID) {
					return false
				} else if !curVMI.IsRunning() {
					return false
				}
				return true
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "New VMI should start")

			// sanity check that the test ran correctly by
			// verifying a different Pod backs the VMI as well.
			By("Verifying a new pod backs the VMI")
			pods, err = virtClient.CoreV1().Pods(newVM.Namespace).List(tests.UnfinishedVMIPodSelector(curVMI))
			Expect(err).ToNot(HaveOccurred(), "Should list the pods")
			Expect(len(pods.Items)).To(Equal(1), "There should be only one pod")
			pod := pods.Items[0]
			Expect(pod.Name).ToNot(Equal(firstPod.Name), "The pod should be different than previous VMI pod")
		})

		It("should stop VirtualMachineInstance if running set to false", func() {

			currVM := newVirtualMachine(false)
			currVM = startVM(currVM)
			currVM = stopVM(currVM)

		})

		It("should start and stop VirtualMachineInstance multiple times", func() {
			var currVM *v1.VirtualMachine

			currVM = newVirtualMachine(false)

			// Start and stop VirtualMachineInstance multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				startVM(currVM)
				stopVM(currVM)
			}
		})

		It("should not update the VirtualMachineInstance spec if Running", func() {
			newVM := newVirtualMachine(true)

			Eventually(func() bool {
				newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
				return newVM.Status.Ready
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "VM should be ready")

			By("Updating the VM template spec")
			newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")

			updatedVM := newVM.DeepCopy()
			updatedVM.Spec.Template.Spec.Domain.Resources.Requests = v13.ResourceList{
				v13.ResourceMemory: resource.MustParse("4096Ki"),
			}
			updatedVM, err := virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
			Expect(err).ToNot(HaveOccurred(), "Should update VM with new spec")

			By("Expecting the old VirtualMachineInstance spec still running")
			vmi, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should retrieve VMI")

			vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory := newVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0), "The retrieved VMI should have the same spec as updated")

			By("Restarting the VMI")
			newVM = stopVM(newVM)
			newVM = startVM(newVM)

			By("Expecting updated spec running")
			vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should retrieve VMI after restart")

			vmiMemory = vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory = updatedVM.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0), "Retrieved VMI should have the updated spec")
		})

		It("should survive guest shutdown, multiple times", func() {
			By("Creating new VM, not running")
			newVM := newVirtualMachine(false)
			newVM = startVM(newVM)
			var vmi *v1.VirtualMachineInstance

			for i := 0; i < 3; i++ {
				currentVMI, err := virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should retrieve VMI")

				By("Getting the running VirtualMachineInstance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 120*time.Second, 5*time.Second).Should(BeTrue(), "Retrieved VMI should be running")

				By("Obtaining the serial console")
				expecter, err := tests.LoggedInCirrosExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should connect to console")
				defer expecter.Close()

				By("Guest shutdown")
				_, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "sudo poweroff\n"},
					&expect.BExp{R: "The system is going down NOW!"},
				}, 120*time.Second)
				Expect(err).ToNot(HaveOccurred(), "Should poweroff the VMI from inside")

				By("waiting for the controller to replace the shut-down vmi with a new instance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					// Almost there, a new instance should be spawned soon
					if errors.IsNotFound(err) {
						return false
					}
					Expect(err).ToNot(HaveOccurred(), "Should retrieve new VMI")
					// If the UID of the vmi changed we see the new vmi
					if vmi.UID != currentVMI.UID {
						return true
					}
					return false
				}, 120*time.Second, 5*time.Second).Should(BeTrue(), "New VirtualMachineInstance instance showed up")

				By("VMI should run the VirtualMachineInstance again")
			}
		})

		Context("Using virtctl interface", func() {
			It("should start a VirtualMachineInstance once", func() {
				var vmi *v1.VirtualMachineInstance
				var err error
				By("getting an VM")
				newVM := newVirtualMachine(false)

				By("Invoking virtctl start")
				virtctl := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", newVM.Namespace, newVM.Name)

				err = virtctl()
				Expect(err).ToNot(HaveOccurred(), "Should execute virtctl correctly")

				By("Getting the status of the VM")
				Eventually(func() bool {
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should retrieve new VM")
					return newVM.Status.Ready
				}, 120*time.Second, 5*time.Second).Should(BeTrue(), "Retrieved VM should be ready")

				By("Getting the running VirtualMachineInstance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should retrieve VMI")
					return vmi.Status.Phase == v1.Running
				}, 30*time.Second, 5*time.Second).Should(BeTrue(), "Retrieved VMI should be running")

				By("Ensuring a second invocation should fail")
				err = virtctl()
				Expect(err).To(HaveOccurred(), "Calling virtctl second time should fail")
			})

			It("should stop a VirtualMachineInstance once", func() {
				var err error
				By("getting an VM")
				newVM := newVirtualMachine(true)

				By("Invoking virtctl stop")
				virtctl := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", newVM.Namespace, newVM.Name)

				By("Ensuring VM is running")
				Eventually(func() error {
					_, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					return err
				}, 10*time.Second, 2*time.Second).ShouldNot(HaveOccurred(), "Running VM should be running")

				err = virtctl()
				Expect(err).ToNot(HaveOccurred(), "Should call virtctl successfuly")

				By("Ensuring VM is not running")
				Eventually(func() bool {
					newVM, err = virtClient.VirtualMachine(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should retrieve VM")
					return !newVM.Status.Ready && !newVM.Status.Created
				}, 120*time.Second, 5*time.Second).Should(BeTrue(), "VM should not be running")

				By("Ensuring the VirtualMachineInstance is removed")
				Eventually(func() error {
					_, err = virtClient.VirtualMachineInstance(newVM.Namespace).Get(newVM.Name, &v12.GetOptions{})
					// Expect a 404 error
					return err
				}, 75*time.Second, 5*time.Second).Should(HaveOccurred(), "VMI should be removed")

				By("Ensuring a second invocation should fail")
				err = virtctl()
				Expect(err).To(HaveOccurred(), "Calling virtctl second time should fail")
			})
		})
	})
})

// NewRandomVirtualMachine creates new VirtualMachine
func NewRandomVirtualMachine(vmi *v1.VirtualMachineInstance, running bool) *v1.VirtualMachine {
	name := vmi.Name
	namespace := vmi.Namespace
	vm := &v1.VirtualMachine{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: running,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: v12.ObjectMeta{
					Labels:    map[string]string{"name": dns.SanitizeHostname(vmi)},
					Name:      name,
					Namespace: namespace,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return vm
}
