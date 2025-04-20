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
 */

package openapi_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/util/openapi"
	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

var _ = Describe("Openapi", func() {

	var validator *openapi.Validator
	var vmi *v1.VirtualMachineInstance
	var obj *unstructured.Unstructured

	BeforeEach(func() {
		validator = openapi.CreateOpenAPIValidator(definitions.ComposeAPIDefinitions())
		vmi = v1.NewVMI("testvm", "")
		obj = &unstructured.Unstructured{}
	})

	var expectValidationsToSucceed = func() {
		Expect(validator.Validate(obj.GroupVersionKind(), obj.Object)).To(BeEmpty())
		Expect(validator.ValidateStatus(obj.GroupVersionKind(), obj.Object)).To(BeEmpty())
	}

	var expectValidationsToFail = func() {
		Expect(validator.Validate(obj.GroupVersionKind(), obj.Object)).ToNot(BeEmpty())
		Expect(validator.ValidateSpec(obj.GroupVersionKind(), obj.Object)).ToNot(BeEmpty())
	}

	It("should accept unknown fields in the status", func() {
		data, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, obj)).To(Succeed())
		Expect(unstructured.SetNestedField(obj.Object, "something", "status", "unknown")).To(Succeed())
		expectValidationsToSucceed()
	})

	It("should reject unknown fields in the spec", func() {
		data, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, obj)).To(Succeed())
		Expect(unstructured.SetNestedField(obj.Object, "something", "spec", "unknown")).To(Succeed())
		expectValidationsToFail()
	})

	It("should accept Machine with an empty Type", func() {
		// This is needed to provide backward compatibility since our example VMIs used to be defined in this way
		vmi.Spec.Domain.Machine = &v1.Machine{Type: ""}
		data, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		Expect(json.Unmarshal(data, obj)).To(Succeed())
		expectValidationsToSucceed()
	})

})
