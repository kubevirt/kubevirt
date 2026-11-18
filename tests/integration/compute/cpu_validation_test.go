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
 * Copyright The KubeVirt Authors.
 *
 */

package compute_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

type cpuCreateFunc func(ctx context.Context, cpu *virtv1.CPU) error

// These specs exercise the +kubebuilder:validation:XValidation (CEL) rules on
// the CPU type (staging/src/kubevirt.io/api/core/v1/schema.go) against the
// generated CRD schemas via a real, in-process API server. No kubevirt webhook
// runs in this environment, so a rejection here can only come from the
// CRD-embedded CEL rule itself.
var _ = Describe("CPU CEL validation", Ordered, ContinueOnFailure, func() {
	var (
		f  *framework.Framework
		ns string
	)

	BeforeAll(func() {
		f = framework.New()
		f.Start(context.Background())
	})

	BeforeEach(func(ctx SpecContext) {
		ns = f.CreateNamespace(ctx)
	})

	newVMI := func(cpu *virtv1.CPU) *virtv1.VirtualMachineInstance {
		vmi := libvmi.New()
		vmi.Spec.Domain.CPU = cpu
		return vmi
	}

	createVMI := func(ctx context.Context, cpu *virtv1.CPU) error {
		_, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, newVMI(cpu), metav1.CreateOptions{})
		return err
	}

	createVM := func(ctx context.Context, cpu *virtv1.CPU) error {
		_, err := f.VirtClient().VirtualMachine(ns).Create(ctx, libvmi.NewVirtualMachine(newVMI(cpu)), metav1.CreateOptions{})
		return err
	}

	Context("threads with dedicated CPU placement", func() {
		const message = "Not more than two threads must be provided when DedicatedCPUPlacement is true"

		DescribeTable("should accept", func(ctx SpecContext, cpu *virtv1.CPU) {
			Expect(createVMI(ctx, cpu)).To(Succeed())
		},
			Entry("two threads with dedicated placement", &virtv1.CPU{DedicatedCPUPlacement: true, Threads: 2}),
			Entry("unset threads with dedicated placement", &virtv1.CPU{DedicatedCPUPlacement: true}),
			Entry("more than two threads without dedicated placement", &virtv1.CPU{Threads: 4}),
		)

		DescribeTable("should reject more than two threads with dedicated placement", func(ctx SpecContext, create cpuCreateFunc) {
			Expect(create(ctx, &virtv1.CPU{DedicatedCPUPlacement: true, Threads: 3})).To(MatchError(ContainSubstring(message)))
		},
			Entry("on a VirtualMachineInstance", createVMI),
			Entry("on a VirtualMachine template", createVM),
		)
	})

	Context("sockets with maxSockets", func() {
		const message = "Number of sockets in CPU topology is greater than the maximum sockets allowed"

		patchVMISockets := func(ctx context.Context, name string, sockets uint32) error {
			GinkgoHelper()

			payload, err := patch.New(patch.WithReplace("/spec/domain/cpu/sockets", sockets)).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())

			_, err = f.VirtClient().VirtualMachineInstance(ns).Patch(ctx, name, types.JSONPatchType, payload, metav1.PatchOptions{})
			return err
		}

		DescribeTable("should accept", func(ctx SpecContext, cpu *virtv1.CPU) {
			Expect(createVMI(ctx, cpu)).To(Succeed())
		},
			Entry("sockets equal to maxSockets", &virtv1.CPU{Sockets: 4, MaxSockets: 4}),
			Entry("sockets lower than maxSockets", &virtv1.CPU{Sockets: 2, MaxSockets: 4}),
			Entry("unset sockets with maxSockets", &virtv1.CPU{MaxSockets: 4}),
			Entry("sockets without maxSockets", &virtv1.CPU{Sockets: 4}),
		)

		It("should validate sockets against maxSockets on update", func(ctx SpecContext) {
			vmi, err := f.VirtClient().VirtualMachineInstance(ns).Create(ctx, newVMI(&virtv1.CPU{Sockets: 2, MaxSockets: 4}), metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("raising sockets up to maxSockets")
			Expect(patchVMISockets(ctx, vmi.Name, 4)).To(Succeed())

			By("raising sockets above maxSockets")
			Expect(patchVMISockets(ctx, vmi.Name, 8)).To(MatchError(ContainSubstring(message)))
		})

		DescribeTable("should reject sockets greater than maxSockets", func(ctx SpecContext, create cpuCreateFunc) {
			Expect(create(ctx, &virtv1.CPU{Sockets: 16, MaxSockets: 8})).To(MatchError(ContainSubstring(message)))
		},
			Entry("on a VirtualMachineInstance", createVMI),
			Entry("on a VirtualMachine template", createVM),
		)
	})
})
