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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	poolv1beta1 "kubevirt.io/api/pool/v1beta1"

	"kubevirt.io/kubevirt/pkg/crdvalidation"
)

var _ = Describe("VirtualMachinePool Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Minimum validations", func() {
		It("should reject negative replicas", func() {
			pool := &poolv1beta1.VirtualMachinePool{
				Spec: poolv1beta1.VirtualMachinePoolSpec{
					Replicas: ptr.To(int32(-1)),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					VirtualMachineTemplate: &poolv1beta1.VirtualMachineTemplateSpec{},
				},
			}

			errs := validator.Validate("virtualmachinepool", pool)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).ToNot(BeEmpty())
		})

		It("should accept zero replicas", func() {
			pool := &poolv1beta1.VirtualMachinePool{
				Spec: poolv1beta1.VirtualMachinePoolSpec{
					Replicas: ptr.To(int32(0)),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app": "test",
						},
					},
					VirtualMachineTemplate: &poolv1beta1.VirtualMachineTemplateSpec{},
				},
			}

			errs := validator.Validate("virtualmachinepool", pool)
			Expect(errs.ByType(crdvalidation.ErrorTypeMinimum)).To(BeEmpty())
		})
	})
})
