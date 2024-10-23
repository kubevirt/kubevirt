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
 * Copyright The KubeVirt Authors
 *
 */

package compute

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("[rfe_id:1177][crit:medium] VirtualMachine", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("[test_id:3007][QUARANTINE] Should force restart a VM with terminationGracePeriodSeconds>0", decorators.Quarantine, func() {
		By("getting a VM with high TerminationGracePeriod")
		vm := libvmi.NewVirtualMachine(libvmifact.NewFedora(libvmi.WithTerminationGracePeriod(600)))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vm = libvmops.StartVirtualMachine(vm)

		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Force restarting the VM with grace period of 0")
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{GracePeriodSeconds: pointer.P(int64(0))})
		Expect(err).ToNot(HaveOccurred())

		// Checks if the old VMI Pod still exists after the force restart
		Eventually(func() error {
			_, err := libpod.GetRunningPodByLabel(string(vmi.UID), v1.CreatedByLabel, vm.Namespace, "")
			return err
		}, 120*time.Second, 1*time.Second).Should(MatchError(ContainSubstring("failed to find pod with the label")))

		Eventually(matcher.ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(matcher.BeRestarted(vmi.UID))

		By("Comparing the new UID and CreationTimeStamp with the old one")
		newVMI, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(newVMI.CreationTimestamp).ToNot(Equal(vmi.CreationTimestamp))
		Expect(newVMI.UID).ToNot(Equal(vmi.UID))
	})

	It("should force stop a VM with terminationGracePeriodSeconds>0", func() {
		By("getting a VM with high TerminationGracePeriod")
		vm := libvmi.NewVirtualMachine(libvmifact.NewFedora(libvmi.WithTerminationGracePeriod(600)))
		vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		vm = libvmops.StartVirtualMachine(vm)

		By("setting up a watch for vmi")
		lw, err := virtClient.VirtualMachineInstance(vm.Namespace).Watch(context.Background(), metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		terminationGracePeriodUpdated := func(done <-chan bool, events <-chan watch.Event, updated chan<- bool) {
			GinkgoRecover()
			for {
				select {
				case <-done:
					return
				case e := <-events:
					vmi, ok := e.Object.(*v1.VirtualMachineInstance)
					Expect(ok).To(BeTrue())
					if vmi.Name != vm.Name {
						continue
					}
					if *vmi.Spec.TerminationGracePeriodSeconds == 0 {
						updated <- true
					}
				}
			}
		}
		done := make(chan bool, 1)
		updated := make(chan bool, 1)
		go terminationGracePeriodUpdated(done, lw.ResultChan(), updated)

		By("Stopping the VM")
		err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{GracePeriod: pointer.P(int64(0))})
		Expect(err).ToNot(HaveOccurred())

		By("Ensuring the VirtualMachineInstance is removed")
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 240*time.Second, 1*time.Second).ShouldNot(matcher.Exist())

		Expect(updated).To(Receive(), "vmi should be updated")
		done <- true
	})
})
