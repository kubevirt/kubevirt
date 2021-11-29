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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package tests_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/intstr"

	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

var _ = Describe("[rfe_id:3064][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Pausing", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Context("A valid VMI", func() {

		var vmi *v1.VirtualMachineInstance

		runVMI := func() {
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			tests.RunVMIAndExpectLaunch(vmi, 90)
		}

		When("paused via API", func() {
			It("[test_id:4597]should signal paused state with condition", func() {
				runVMI()

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Pause(vmi.Name, &v1.PauseOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(vmi.Name, &v1.UnpauseOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			})
		})

		When("paused via virtctl", func() {
			It("[test_id:3079]should signal paused state with condition", func() {
				runVMI()
				command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			})

			It("[test_id:3080]should signal unpaused state with removed condition", func() {
				runVMI()
				command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			})
		})

		When("paused via virtctl multiple times", func() {
			It("[test_id:3225]should signal unpaused state with removed condition at the end", func() {
				runVMI()

				for i := 0; i < 3; i++ {
					By("Pausing VMI")
					command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
					Expect(command()).To(Succeed())
					tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

					By("Unpausing VMI")
					command = tests.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
					Expect(command()).To(Succeed())
					tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
				}
			})
		})

		Context("with a LivenessProbe configured", func() {
			When("paused via virtctl", func() {
				It("[test_id:3224]should not be paused", func() {
					By("Launching a VMI with LivenessProbe")
					vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
					// a random probe which will not fail immediately
					vmi.Spec.LivenessProbe = &v1.Probe{
						Handler: v1.Handler{
							HTTPGet: &k8sv1.HTTPGetAction{
								Path: "/something",
								Port: intstr.FromInt(8080),
							},
						},
						InitialDelaySeconds: 120,
						TimeoutSeconds:      120,
						PeriodSeconds:       120,
						SuccessThreshold:    1,
						FailureThreshold:    1,
					}
					tests.RunVMIAndExpectLaunch(vmi, 90)

					By("Pausing it")
					command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
					err := command()
					Expect(err.Error()).To(ContainSubstring("Pausing VMIs with LivenessProbe is currently not supported"))
				})
			})
		})

		When("paused via virtctl with --dry-run flag", func() {
			It("should not paused", func() {
				runVMI()
				command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--dry-run", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				By(fmt.Sprintf("Checking that VMI remains running"))
				Consistently(func() bool {
					updatedVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstancePaused && condition.Status == k8sv1.ConditionTrue {
							return false
						}
					}
					return true
				}, time.Duration(5)*time.Second).Should(BeTrue())
			})
		})

		When("unpaused via virtctl with --dry-run flag", func() {
			It("should not unpaused", func() {
				runVMI()
				command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vmi", "--dry-run", "--namespace", util.NamespaceTestDefault, vmi.Name)
				Expect(command()).To(Succeed())

				By(fmt.Sprintf("Checking that VMI remains paused"))
				Consistently(func() bool {
					updatedVmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstancePaused && condition.Status == k8sv1.ConditionTrue {
							return true
						}
					}
					return false
				}, time.Duration(5)*time.Second).Should(BeTrue())
			})
		})

		It("should not appear as ready when paused", func() {
			runVMI()

			tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceReady, 30)

			By("Pausing the VMI and expecting to become unready")
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Pause(vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstanceReady, 30)

			By("Unpausing the VMI and expecting to become ready")
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceReady, 30)
		})
	})

	Context("A valid VM", func() {

		var vm *v1.VirtualMachine

		runVM := func() {
			vm = tests.NewRandomVMWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(vm)
			Expect(err).ToNot(HaveOccurred())
			vm = tests.StartVirtualMachine(vm)
		}

		When("paused via API", func() {
			It("[test_id:4598]should signal paused state with condition", func() {

				runVM()

				err = virtClient.VirtualMachineInstance(vm.Namespace).Pause(vm.Name, &v1.PauseOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				err = virtClient.VirtualMachineInstance(vm.Namespace).Unpause(vm.Name, &v1.UnpauseOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)
			})

		})

		When("paused via virtctl", func() {

			It("[test_id:3059]should signal paused state with condition", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)
			})

			It("[test_id:3081]should gracefully handle pausing the VM again", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				err := command()
				Expect(err.Error()).To(ContainSubstring("VMI is already paused"))
			})

			It("[test_id:3088]should gracefully handle pausing the VMI again", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vm.Name)
				err := command()
				Expect(err.Error()).To(ContainSubstring("VMI is already paused"))
			})

			It("[test_id:3060]should signal unpaused state with removed condition", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)
			})

			It("[test_id:3082]should gracefully handle unpausing again", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				err := command()
				Expect(err.Error()).To(ContainSubstring("VMI is not paused"))
			})

			It("[test_id:3085]should be stopped successfully", func() {

				runVM()

				By("Pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("Stopping the VM")
				command = tests.NewRepeatableVirtctlCommand("stop", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())

				By("Checking deletion of VMI")
				Eventually(func() bool {
					_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
					if errors.IsNotFound(err) {
						return true
					}
					Expect(err).ToNot(HaveOccurred())
					return false
				}, 300*time.Second, 1*time.Second).Should(BeTrue(), "The VMI did not disappear")

				By("Checking status of VM")
				Eventually(func() bool {
					vm, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vm.Status.Ready
				}, 300*time.Second, 1*time.Second).Should(BeFalse())

			})

			It("[test_id:3229]should gracefully handle being started again", func() {

				runVM()

				By("Pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("Starting the VM")
				command = tests.NewRepeatableVirtctlCommand("start", "--namespace", util.NamespaceTestDefault, vm.Name)
				err = command()
				Expect(err.Error()).To(ContainSubstring("VM is already running"))

			})

			It("[test_id:3226]should be restarted successfully into unpaused state", func() {

				runVM()

				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				oldId := vmi.UID

				By("Pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("Restarting the VM")
				command = tests.NewRepeatableVirtctlCommand("restart", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())

				By("Checking deletion of VMI")
				Eventually(func() bool {
					newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
					if errors.IsNotFound(err) || (err == nil && newVMI.UID != oldId) {
						return true
					}
					Expect(err).ToNot(HaveOccurred())
					return false
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), "The VMI did not disappear")

				By("Waiting for for new VMI to start")
				Eventually(func() error {
					_, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
					return err
				}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred(), "No new VMI appeared")

				newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &v12.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(newVMI, 300)

				By("Ensuring unpaused state")
				tests.WaitForVMConditionRemovedOrFalse(virtClient, vm, v1.VirtualMachinePaused, 30)
				tests.WaitForVMIConditionRemovedOrFalse(virtClient, newVMI, v1.VirtualMachineInstancePaused, 30)

			})

			It("[test_id:3086]should not be migrated", func() {

				runVM()

				By("Pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("Trying to migrate the VM")
				command = tests.NewRepeatableVirtctlCommand("migrate", "--namespace", util.NamespaceTestDefault, vm.Name)
				err = command()
				Expect(err.Error()).To(ContainSubstring("VM is paused"))

			})

			It("[test_id:3083]should connect to serial console", func() {

				runVM()

				By("Pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("Trying to console into the VM")
				_, err = virtClient.VirtualMachineInstance(vm.ObjectMeta.Namespace).SerialConsole(vm.ObjectMeta.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:3084]should connect to vnc console", func() {

				runVM()

				By("Pausing the VM")
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				By("Trying to vnc into the VM")
				_, err = virtClient.VirtualMachineInstance(vm.ObjectMeta.Namespace).VNC(vm.ObjectMeta.Name)
				Expect(err).ToNot(HaveOccurred())

			})
		})

		When("paused via virtctl with --dry-run flag", func() {
			It("should not paused", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--dry-run", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				By(fmt.Sprintf("Checking that VM remains running"))
				Consistently(func() bool {
					updatedVm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVm.Status.Conditions {
						if condition.Type == v1.VirtualMachinePaused && condition.Status == k8sv1.ConditionTrue {
							return false
						}
					}
					return true
				}, time.Duration(5)*time.Second).Should(BeTrue())
			})
		})

		When("unpaused via virtctl with --dry-run flag", func() {
			It("should not unpaused", func() {
				runVM()
				command := tests.NewRepeatableVirtctlCommand("pause", "vm", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())
				tests.WaitForVMCondition(virtClient, vm, v1.VirtualMachinePaused, 30)

				command = tests.NewRepeatableVirtctlCommand("unpause", "vm", "--dry-run", "--namespace", util.NamespaceTestDefault, vm.Name)
				Expect(command()).To(Succeed())

				By(fmt.Sprintf("Checking that VM remains paused"))
				Consistently(func() bool {
					updatedVm, err := virtClient.VirtualMachine(util.NamespaceTestDefault).Get(vm.Name, &v12.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range updatedVm.Status.Conditions {
						if condition.Type == v1.VirtualMachinePaused && condition.Status == k8sv1.ConditionTrue {
							return true
						}
					}
					return false
				}, time.Duration(5)*time.Second).Should(BeTrue())
			})
		})
	})

	Context("Guest and Host uptime difference before pause", func() {
		startTime := time.Now()
		var (
			vmi                     *v1.VirtualMachineInstance
			uptimeDiffBeforePausing float64
		)

		grepGuestUptime := func(vmi *v1.VirtualMachineInstance) float64 {
			res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
				&expect.BSnd{S: `cat /proc/uptime | awk '{print $1;}'` + "\n"},
				&expect.BExp{R: console.RetValue("[0-9\\.]+")}, // guest uptime
			}, 15)
			Expect(err).ToNot(HaveOccurred())
			re := regexp.MustCompile("\r\n[0-9\\.]+\r\n")
			guestUptime, err := strconv.ParseFloat(strings.TrimSpace(re.FindString(res[0].Match[0])), 64)
			Expect(err).ToNot(HaveOccurred(), "should be able to parse uptime to float")
			return guestUptime
		}

		hostUptime := func() float64 {
			return time.Since(startTime).Seconds()
		}

		BeforeEach(func() {
			By("Starting a Cirros VMI")
			vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskCirros))
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			vmi = tests.RunVMIAndExpectLaunchWithIgnoreWarningArg(vmi, 240, false)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

			By("checking uptime difference between guest and host")
			uptimeDiffBeforePausing = hostUptime() - grepGuestUptime(vmi)
		})

		It("[test_id:3090]should be less than uptime difference after pause", func() {
			By("Pausing the VMI")
			command := tests.NewRepeatableVirtctlCommand("pause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
			Expect(command()).To(Succeed(), "should successfully pause the vmi")
			tests.WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)
			time.Sleep(10 * time.Second) // sleep to increase uptime diff

			By("Unpausing the VMI")
			command = tests.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", util.NamespaceTestDefault, vmi.Name)
			Expect(command()).To(Succeed(), "should successfully unpause tthe vmi")
			tests.WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstancePaused, 30)

			By("Verifying VMI was indeed Paused")
			uptimeDiffAfterPausing := hostUptime() - grepGuestUptime(vmi)
			Expect(uptimeDiffAfterPausing).To(BeNumerically(">", uptimeDiffBeforePausing+10), "uptime diff after pausing should be greater by at least 10 than before pausing")
		})
	})
})
