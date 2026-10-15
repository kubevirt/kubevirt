/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package compute_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

var _ = Describe("VM Admission", Ordered, ContinueOnFailure, func() {
	var f *framework.Framework

	BeforeAll(func() {
		f = framework.New(framework.WithWebhooks())
		f.Start(context.Background())
	})

	It("should mutate a VM with defaults on creation", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
		)
		Expect(vm.Spec.Template.Spec.Domain.Firmware).To(BeNil(),
			"firmware should not be set before admission")

		var err error
		vm, err = f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("verifying the mutating webhook set firmware defaults")
		Expect(vm.Spec.Template.Spec.Domain.Firmware).NotTo(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Firmware.UUID).NotTo(BeEmpty(),
			"firmware UUID should be defaulted by the mutating webhook")
	})

	It("should reject a VM with an invalid disk", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
		)
		vm.Spec.Template.Spec.Domain.Devices.Disks = []virtv1.Disk{
			{
				Name: "missing-volume",
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{Bus: virtv1.DiskBusVirtio},
				},
			},
		}

		_, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).To(MatchError(errors.IsInvalid, "IsInvalid"))
		Expect(err).To(MatchError(ContainSubstring("missing-volume")))
		_, err = f.VirtClient().VirtualMachine(ns).Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
	})
})
