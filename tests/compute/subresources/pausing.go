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

package compute

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = compute.SIGDescribe("[rfe_id:3064][crit:medium][vendor:cnv-qe@redhat.com][level:component] Pausing", func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("A valid VMI", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			const timeout = 90
			vmi = libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), timeout)
		})

		It("[test_id:4597]should signal paused state with condition", func() {
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

			By("Pausing VMI")
			err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

			By("Unpausing VMI")
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		})

		It("[test_id:3224]should not be paused with a LivenessProbe configured", func() {
			By("Launching a VMI with LivenessProbe")
			vmi = libvmifact.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
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
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

			By("Pausing it")
			err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).To(MatchError(ContainSubstring("Pausing VMIs with LivenessProbe is currently not supported")))
		})

		It("[test_id:7671]should not pause with dry run", func() {
			err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{DryRun: []string{metav1.DryRunAll}})
			Expect(err).ToNot(HaveOccurred())
			By("Checking that VMI remains running")
			Consistently(matcher.ThisVMI(vmi), 5*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
		})

		It("[test_id:7672]should not unpause with dry run", func() {
			err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

			err = virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{DryRun: []string{metav1.DryRunAll}})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that VMI remains paused")
			Consistently(matcher.ThisVMI(vmi), 5*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
		})
	})

	Context("A valid VM", func() {
		var vm *v1.VirtualMachine

		BeforeEach(func() {
			vm = libvmi.NewVirtualMachine(libvmifact.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork())),
				libvmi.WithRunStrategy(v1.RunStrategyAlways))
			var err error
			vm, err = virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(matcher.BeReady())
		})

		It("[test_id:4598]should signal paused state with condition", func() {
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			err = virtClient.VirtualMachineInstance(vm.Namespace).Unpause(context.Background(), vm.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachinePaused))
		})

		It("[test_id:3081]should gracefully handle pausing the VM again", func() {
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			err = virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).To(MatchError(ContainSubstring("VMI is already paused")))
		})

		It("[test_id:3060]should signal unpaused state with removed condition", func() {
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			err = virtClient.VirtualMachineInstance(vm.Namespace).Unpause(context.Background(), vm.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachinePaused))
		})

		It("[test_id:3082]should gracefully handle unpausing again", func() {
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			err = virtClient.VirtualMachineInstance(vm.Namespace).Unpause(context.Background(), vm.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachinePaused))

			err = virtClient.VirtualMachineInstance(vm.Namespace).Unpause(context.Background(), vm.Name, &v1.UnpauseOptions{})
			Expect(err).To(MatchError(ContainSubstring("VMI is not paused")))
		})

		It("[test_id:3085]should be stopped successfully", func() {
			By("Pausing the VM")
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			By("Stopping the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking deletion of VMI")
			Eventually(func() error {
				_, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return err
			}, 300*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "The VMI did not disappear")

			By("Checking status of VM")
			Eventually(func() bool {
				vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vm.Status.Ready
			}, 300*time.Second, 1*time.Second).Should(BeFalse())
		})

		It("[test_id:3229]should gracefully handle being started again", func() {
			By("Pausing the VM")
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			By("Starting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
			Expect(err).To(MatchError(ContainSubstring("VM is already running")))
		})

		It("[test_id:3226]should be restarted successfully into unpaused state", func() {
			vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			oldId := vmi.UID

			By("Pausing the VM")
			err = virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			By("Restarting the VM")
			err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Checking deletion of VMI")
			Eventually(func() bool {
				newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				if errors.IsNotFound(err) || (err == nil && newVMI.UID != oldId) {
					return true
				}
				Expect(err).ToNot(HaveOccurred())
				return false
			}, 60*time.Second, 1*time.Second).Should(BeTrue(), "The VMI did not disappear")

			By("Waiting for for new VMI to start")
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				return err
			}, 60*time.Second, 1*time.Second).ShouldNot(HaveOccurred(), "No new VMI appeared")

			newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(newVMI,
				libwait.WithTimeout(300),
			)

			By("Ensuring unpaused state")
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachinePaused))
			Eventually(matcher.ThisVMI(newVMI), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
		})

		It("[test_id:3083]should connect to serial console", func() {
			By("Pausing the VM")
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			By("Trying to console into the VM")
			_, err = virtClient.VirtualMachineInstance(vm.ObjectMeta.Namespace).SerialConsole(vm.ObjectMeta.Name, &kvcorev1.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:3084]should connect to vnc console", func() {
			By("Pausing the VM")
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			By("Trying to vnc into the VM")
			_, err = virtClient.VirtualMachineInstance(vm.ObjectMeta.Namespace).VNC(vm.ObjectMeta.Name)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:7673]should not pause with dry run", func() {
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{DryRun: []string{metav1.DryRunAll}})
			Expect(err).ToNot(HaveOccurred())
			By("Checking that VM remains running")
			Consistently(matcher.ThisVM(vm), 5*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachinePaused))
		})

		It("[test_id:7674]should not unpause with dry run", func() {
			err := virtClient.VirtualMachineInstance(vm.Namespace).Pause(context.Background(), vm.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))

			err = virtClient.VirtualMachineInstance(vm.Namespace).Unpause(context.Background(), vm.Name, &v1.UnpauseOptions{DryRun: []string{metav1.DryRunAll}})
			Expect(err).ToNot(HaveOccurred())

			By("Checking that VM remains paused")
			Consistently(matcher.ThisVM(vm), 5*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachinePaused))
		})
	})

	Context("Guest and Host uptime difference before pause", func() {
		startTime := time.Now()
		const (
			sleepTimeSeconds = 10
			deviation        = 4
		)

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
			const timeout = 90
			vmi = libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), timeout)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			By("checking uptime difference between guest and host")
			uptimeDiffBeforePausing = hostUptime() - grepGuestUptime(vmi)
		})

		It("[test_id:3090]should be less than uptime difference after pause", func() {
			By("Pausing the VMI")
			err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
			time.Sleep(sleepTimeSeconds * time.Second) // sleep to increase uptime diff

			By("Unpausing the VMI")
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

			By("Verifying VMI was indeed Paused")
			uptimeDiffAfterPausing := hostUptime() - grepGuestUptime(vmi)

			// We subtract from the sleep time the deviation due to the low resolution of `uptime` (seconds).
			// If you capture the uptime when it is at the beginning of that second or at the end of that second,
			// the value comes out the same even though in fact a whole second has almost passed.
			// In extreme cases, as we take 4 readings (2 initially and 2 after the unpause), the deviation could be up to just under 4 seconds.
			// This fact does not invalidate the purpose of the test, which is to prove that during the pause the vmi is actually paused.
			Expect(uptimeDiffAfterPausing-uptimeDiffBeforePausing).To(BeNumerically(">=", sleepTimeSeconds-deviation), fmt.Sprintf("guest should be paused for at least %d seconds", sleepTimeSeconds-deviation))
		})
	})
})
