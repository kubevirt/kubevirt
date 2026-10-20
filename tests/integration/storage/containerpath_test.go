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

package storage_test

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

var _ = Describe("ContainerPath CEL validation", func() {
	var f *framework.Framework

	BeforeEach(func() {
		f = framework.New(framework.WithoutWebhooks())
		f.Start(context.Background())
	})

	It("should reject a VM with a relative containerPath via CEL validation", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
		)
		vm.Spec.Template.Spec.Volumes = []virtv1.Volume{{
			Name: "bad-vol",
			VolumeSource: virtv1.VolumeSource{
				ContainerPath: &virtv1.ContainerPathVolumeSource{Path: "relative/path"},
			},
		}}

		By("creating a VM with a relative containerPath")
		_, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).To(MatchError(errors.IsInvalid, "IsInvalid"))
		Expect(err).To(MatchError(ContainSubstring("path must be absolute")),
			"CEL rule should reject a relative containerPath")
	})
})
