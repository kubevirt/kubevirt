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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

var _ = Describe("VM schema validation", func() {
	var f *framework.Framework

	BeforeEach(func() {
		f = framework.New(framework.WithoutWebhooks())
		f.Start(context.Background())
	})

	It("should reject a VM missing the required domain field", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
		)
		vm.TypeMeta = metav1.TypeMeta{
			APIVersion: virtv1.StorageGroupVersion.String(),
			Kind:       "VirtualMachine",
		}

		jsonBytes, err := json.Marshal(vm)
		Expect(err).NotTo(HaveOccurred())

		By("removing the required domain field")
		var object map[string]interface{}
		Expect(json.Unmarshal(jsonBytes, &object)).To(Succeed())
		unstructured.RemoveNestedField(object, "spec", "template", "spec", "domain")
		jsonBytes, err = json.Marshal(object)
		Expect(err).NotTo(HaveOccurred())

		err = f.VirtClient().RestClient().Post().
			Resource("virtualmachines").
			Namespace(ns).
			Body(jsonBytes).
			SetHeader("Content-Type", "application/json").
			Do(ctx).Error()

		Expect(err).To(MatchError(errors.IsInvalid, "IsInvalid"))
		Expect(err).To(MatchError(ContainSubstring("spec.template.spec.domain: Required value")),
			"CRD structural schema should reject a VM missing the required domain field")
	})
})
