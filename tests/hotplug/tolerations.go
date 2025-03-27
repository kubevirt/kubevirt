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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package hotplug

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute]VM Tolerations", decorators.SigCompute, decorators.VMLiveUpdateRolloutStrategy, func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Updating VMs tolerations", func() {
		It("should successfully live update tolerations", func() {
			By("Creating a running VM")
			vmi := libvmifact.NewGuestless()
			vm := libvmi.NewVirtualMachine(vmi, libvmi.WithRunStrategy(v1.RunStrategyAlways))

			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Create(context.Background(), vm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			vmi.Namespace = vm.Namespace

			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(matcher.BeReady())
			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Adding tolerations")
			tolerations := []k8sv1.Toleration{
				{
					Effect:            k8sv1.TaintEffectNoExecute,
					Key:               "testTaint",
					Operator:          k8sv1.TolerationOpExists,
					TolerationSeconds: pointer.P(int64(300)),
				},
			}
			patchSet := patch.New(patch.WithAdd("/spec/template/spec/tolerations", tolerations))
			vm, err = patchVM(virtClient, vm, patchSet)
			Expect(err).NotTo(HaveOccurred())

			By("Ensuring the VMI has added the tolerations")
			validateTolerations(virtClient, vmi, ContainElement(tolerations[0]))

			By("Adding additional tolerations")
			newToleration := k8sv1.Toleration{
				Effect:            k8sv1.TaintEffectNoExecute,
				Key:               "testTaint2",
				Operator:          k8sv1.TolerationOpExists,
				TolerationSeconds: pointer.P(int64(300)),
			}
			tolerations = append(tolerations, newToleration)

			patchSet = patch.New(patch.WithAdd("/spec/template/spec/tolerations/-", newToleration))
			vm, err = patchVM(virtClient, vm, patchSet)
			Expect(err).NotTo(HaveOccurred())

			By("Ensuring the VMI has added the additional toleration")
			validateTolerations(virtClient, vmi, Equal(tolerations))

			By("Removing a single toleration")
			patchSet = patch.New(patch.WithRemove("/spec/template/spec/tolerations/1"))
			vm, err = patchVM(virtClient, vm, patchSet)
			Expect(err).NotTo(HaveOccurred())

			By("Ensuring the VMI has removed the toleration")
			validateTolerations(virtClient, vmi, Not(ContainElement(tolerations[1])))

			By("Removing all tolerations")
			patchSet = patch.New(patch.WithRemove("/spec/template/spec/tolerations"))
			vm, err = patchVM(virtClient, vm, patchSet)
			Expect(err).NotTo(HaveOccurred())

			By("Ensuring the VMI has removed all tolerations")
			validateTolerations(virtClient, vmi, BeEmpty())
		})
	})
})

func patchVM(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, patchSet *patch.PatchSet) (*v1.VirtualMachine, error) {
	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return nil, err
	}

	vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return vm, err
}

func validateTolerations(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, matcher gomegatypes.GomegaMatcher) {
	EventuallyWithOffset(1, func() []k8sv1.Toleration {
		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		return vmi.Spec.Tolerations
	}, 60*time.Second, time.Second).Should(matcher)
}
