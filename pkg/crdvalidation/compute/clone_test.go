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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	clonev1beta1 "kubevirt.io/api/clone/v1beta1"

	"kubevirt.io/kubevirt/pkg/crdvalidation"
)

var _ = Describe("VirtualMachineClone Validations", func() {
	var validator *crdvalidation.Validator

	BeforeEach(func() {
		var err error
		validator, err = crdvalidation.NewValidator()
		Expect(err).ToNot(HaveOccurred())
	})

	Context("Enum validations", func() {
		It("should reject invalid volumeNamePolicy enum value", func() {
			invalidPolicy := clonev1beta1.VolumeNamePolicy("InvalidPolicy")
			clone := &clonev1beta1.VirtualMachineClone{
				Spec: clonev1beta1.VirtualMachineCloneSpec{
					Source: &corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "source-vm",
					},
					VolumeNamePolicy: &invalidPolicy,
				},
			}

			errs := validator.Validate("virtualmachineclone", clone)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).ToNot(BeEmpty())
		})

		It("should accept valid volumeNamePolicy enum value - RandomizeNames", func() {
			policy := clonev1beta1.VolumeNamePolicyRandomizeNames
			clone := &clonev1beta1.VirtualMachineClone{
				Spec: clonev1beta1.VirtualMachineCloneSpec{
					Source: &corev1.TypedLocalObjectReference{
						APIGroup: ptr.To("kubevirt.io"),
						Kind:     "VirtualMachine",
						Name:     "source-vm",
					},
					VolumeNamePolicy: &policy,
				},
			}

			errs := validator.Validate("virtualmachineclone", clone)
			Expect(errs.ByType(crdvalidation.ErrorTypeEnum)).To(BeEmpty())
		})
	})
})
