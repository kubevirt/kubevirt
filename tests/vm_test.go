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
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v13 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virtctl/vm"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("[rfe_id:1177][crit:medium][vendor:cnv-qe@redhat.com][level:component]VirtualMachine", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("An invalid VirtualMachine given", func() {

		It("[test_id:1518]should be rejected on POST", func() {
			vmiImage := tests.ContainerDiskFor(tests.ContainerDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			newVMI := NewRandomVirtualMachine(template, false)
			newVMI.TypeMeta = v12.TypeMeta{
				APIVersion: v1.GroupVersion.String(),
				Kind:       "VirtualMachine",
			}

			jsonBytes, err := json.Marshal(newVMI)
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
			vmiImage := tests.ContainerDiskFor(tests.ContainerDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")
			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			template.Spec.Domain.Devices.Disks = append(template.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			newVMI := NewRandomVirtualMachine(template, false)
			newVMI.TypeMeta = v12.TypeMeta{
				APIVersion: v1.GroupVersion.String(),
				Kind:       "VirtualMachine",
			}

			result := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(newVMI).Do()

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
	})

	Context("A valid VirtualMachine given", func() {

		newVirtualMachine := func(running bool) *v1.VirtualMachine {
			vmiImage := tests.ContainerDiskFor(tests.ContainerDiskCirros)
			template := tests.NewRandomVMIWithEphemeralDiskAndUserdata(vmiImage, "echo Hi\n")

			var newVMI *v1.VirtualMachine
			var err error

			newVMI = NewRandomVirtualMachine(template, running)

			newVMI, err = virtClient.VirtualMachine(tests.NamespaceTestDefault).Create(newVMI)
			Expect(err).ToNot(HaveOccurred())

			return newVMI
		}

		startVMI := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Starting the VirtualMachineInstance")

			Eventually(func() error {
				updatedVMI, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVMI.Spec.Running = true
				_, err = virtClient.VirtualMachine(updatedVMI.Namespace).Update(updatedVMI)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			updatedVMI, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Observe the VirtualMachineInstance created
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(updatedVMI.Namespace).Get(updatedVMI.Name, &v12.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(Succeed())

			By("VMI has the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVMI.Namespace).Get(updatedVMI.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			return updatedVMI
		}

		stopVMI := func(vm *v1.VirtualMachine) *v1.VirtualMachine {
			By("Stopping the VirtualMachineInstance")

			Eventually(func() error {
				updatedVMI, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				updatedVMI.Spec.Running = false
				_, err = virtClient.VirtualMachine(updatedVMI.Namespace).Update(updatedVMI)
				return err
			}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			updatedVMI, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Observe the VirtualMachineInstance deleted
			Eventually(func() bool {
				_, err = virtClient.VirtualMachineInstance(updatedVMI.Namespace).Get(updatedVMI.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 300*time.Second, 1*time.Second).Should(BeTrue(), "The vmi did not disappear")

			By("VMI has not the running condition")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(updatedVMI.Namespace).Get(updatedVMI.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeFalse())

			return updatedVMI
		}

		It("[test_id:1520]should update VirtualMachine once VMIs are up", func() {
			newVMI := newVirtualMachine(true)
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(tests.NamespaceTestDefault).Get(newVMI.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("[test_id:1521]should remove VirtualMachineInstance once the VMI is marked for deletion", func() {
			newVMI := newVirtualMachine(true)
			// Delete it
			Expect(virtClient.VirtualMachine(newVMI.Namespace).Delete(newVMI.Name, &v12.DeleteOptions{})).To(Succeed())
			// Wait until VMIs are gone
			Eventually(func() int {
				vmis, err := virtClient.VirtualMachineInstance(newVMI.Namespace).List(&v12.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(vmis.Items)
			}, 300*time.Second, 2*time.Second).Should(BeZero(), "The VirtualMachineInstance did not disappear")
		})

		It("[test_id:1522]should remove owner references on the VirtualMachineInstance if it is orphan deleted", func() {
			newVMI := newVirtualMachine(true)

			Eventually(func() []v12.OwnerReference {
				// Check for owner reference
				vmi, _ := virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
				return vmi.OwnerReferences
			}, 300*time.Second, 1*time.Second).ShouldNot(BeEmpty())

			// Delete it
			orphanPolicy := v12.DeletePropagationOrphan
			Expect(virtClient.VirtualMachine(newVMI.Namespace).
				Delete(newVMI.Name, &v12.DeleteOptions{PropagationPolicy: &orphanPolicy})).To(Succeed())
			// Wait until the virtual machine is deleted
			Eventually(func() bool {
				_, err := virtClient.VirtualMachine(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
				if errors.IsNotFound(err) {
					return true
				}
				return false
			}, 300*time.Second, 1*time.Second).Should(BeTrue())

			vmi, err := virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.OwnerReferences).To(BeEmpty())
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:1523]should recreate VirtualMachineInstance if it gets deleted", func() {
			newVMI := startVMI(newVirtualMachine(false))

			currentVMI, err := virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			Expect(virtClient.VirtualMachineInstance(newVMI.Namespace).Delete(newVMI.Name, &v12.DeleteOptions{})).To(Succeed())

			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
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

			By("Creating a new VMI")
			newVMI := newVirtualMachine(true)

			// wait for a running VirtualMachineInstance.
			By("Waiting for the VMI's VirtualMachineInstance to start")
			Eventually(func() error {
				firstVMI, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
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
			pods, err := virtClient.CoreV1().Pods(newVMI.Namespace).List(tests.UnfinishedVMIPodSelector(firstVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			firstPod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(newVMI.Namespace).Delete(firstPod.Name, &v12.DeleteOptions{})
			}, 120*time.Second, 1*time.Second).Should(Succeed())

			// Wait on the VMI controller to create a new VirtualMachineInstance
			By("Waiting for a new VirtualMachineInstance to spawn")
			Eventually(func() bool {
				curVMI, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})

				// verify a new VirtualMachineInstance gets created for the VMI after the Pod is deleted.
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
			pods, err = virtClient.CoreV1().Pods(newVMI.Namespace).List(tests.UnfinishedVMIPodSelector(curVMI))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			pod := pods.Items[0]
			Expect(pod.Name).ToNot(Equal(firstPod.Name))
		})

		It("[test_id:1525]should stop VirtualMachineInstance if running set to false", func() {

			currVMI := newVirtualMachine(false)
			currVMI = startVMI(currVMI)
			currVMI = stopVMI(currVMI)

		})

		It("[test_id:1526]should start and stop VirtualMachineInstance multiple times", func() {
			var currVMI *v1.VirtualMachine

			currVMI = newVirtualMachine(false)

			// Start and stop VirtualMachineInstance multiple times
			for i := 0; i < 5; i++ {
				By(fmt.Sprintf("Doing run: %d", i))
				startVMI(currVMI)
				stopVMI(currVMI)
			}
		})

		It("[test_id:1527]should not update the VirtualMachineInstance spec if Running", func() {
			newVMI := newVirtualMachine(true)

			Eventually(func() bool {
				newVMI, err = virtClient.VirtualMachine(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return newVMI.Status.Ready
			}, 360*time.Second, 1*time.Second).Should(BeTrue())

			By("Updating the VMI template spec")
			newVMI, err = virtClient.VirtualMachine(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			updatedVMI := newVMI.DeepCopy()
			updatedVMI.Spec.Template.Spec.Domain.Resources.Requests = v13.ResourceList{
				v13.ResourceMemory: resource.MustParse("4096Ki"),
			}
			updatedVMI, err := virtClient.VirtualMachine(updatedVMI.Namespace).Update(updatedVMI)
			Expect(err).ToNot(HaveOccurred())

			By("Expecting the old VirtualMachineInstance spec still running")
			vmi, err := virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory := vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory := newVMI.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))

			By("Restarting the VMI")
			newVMI = stopVMI(newVMI)
			newVMI = startVMI(newVMI)

			By("Expecting updated spec running")
			vmi, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmiMemory = vmi.Spec.Domain.Resources.Requests.Memory()
			vmMemory = updatedVMI.Spec.Template.Spec.Domain.Resources.Requests.Memory()
			Expect(vmiMemory.Cmp(*vmMemory)).To(Equal(0))
		})

		It("[test_id:1528]should survive guest shutdown, multiple times", func() {
			By("Creating new VMI, not running")
			newVMI := newVirtualMachine(false)
			newVMI = startVMI(newVMI)
			var vmi *v1.VirtualMachineInstance

			for i := 0; i < 3; i++ {
				currentVMI, err := virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Getting the running VirtualMachineInstance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
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
					vmi, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
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

		Context("Using virtctl interface", func() {
			It("[test_id:1529]should start a VirtualMachineInstance once", func() {
				var vmi *v1.VirtualMachineInstance
				var err error
				By("getting an VMI")
				newVMI := newVirtualMachine(false)

				By("Invoking virtctl start")
				virtctl := tests.NewRepeatableVirtctlCommand(vm.COMMAND_START, "--namespace", newVMI.Namespace, newVMI.Name)

				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Getting the status of the VMI")
				Eventually(func() bool {
					newVMI, err = virtClient.VirtualMachine(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return newVMI.Status.Ready
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Getting the running VirtualMachineInstance")
				Eventually(func() bool {
					vmi, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase == v1.Running
				}, 240*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring a second invocation should fail")
				err = virtctl()
				Expect(err).To(HaveOccurred())
			})

			It("[test_id:1530]should stop a VirtualMachineInstance once", func() {
				var err error
				By("getting an VMI")
				newVMI := newVirtualMachine(true)

				By("Invoking virtctl stop")
				virtctl := tests.NewRepeatableVirtctlCommand(vm.COMMAND_STOP, "--namespace", newVMI.Namespace, newVMI.Name)

				By("Ensuring VMI is running")
				Eventually(func() bool {
					newVMI, err = virtClient.VirtualMachine(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return newVMI.Status.Ready
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				err = virtctl()
				Expect(err).ToNot(HaveOccurred())

				By("Ensuring VMI is not running")
				Eventually(func() bool {
					newVMI, err = virtClient.VirtualMachine(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return !newVMI.Status.Ready && !newVMI.Status.Created
				}, 360*time.Second, 1*time.Second).Should(BeTrue())

				By("Ensuring the VirtualMachineInstance is removed")
				Eventually(func() error {
					_, err = virtClient.VirtualMachineInstance(newVMI.Namespace).Get(newVMI.Name, &v12.GetOptions{})
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
