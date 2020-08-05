package openapi_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/util/openapi"
	"kubevirt.io/kubevirt/pkg/virt-api/rest"
)

var _ = Describe("Openapi", func() {

	It("should accept unknown fields in the status", func() {
		validator := openapi.CreateOpenAPIValidator(rest.ComposeAPIDefinitions())
		vmi := v1.NewVMI("testvm", "")
		data, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		obj := &unstructured.Unstructured{}
		Expect(json.Unmarshal(data, obj)).To(Succeed())
		Expect(unstructured.SetNestedField(obj.Object, "something", "status", "unknown")).To(Succeed())
		Expect(validator.Validate(obj.GroupVersionKind(), obj.Object)).To(BeEmpty())
		Expect(validator.ValidateStatus(obj.GroupVersionKind(), obj.Object)).To(BeEmpty())
	})

	It("should reject unknown fields in the spec", func() {
		validator := openapi.CreateOpenAPIValidator(rest.ComposeAPIDefinitions())
		vmi := v1.NewVMI("testvm", "")
		data, err := json.Marshal(vmi)
		Expect(err).ToNot(HaveOccurred())
		obj := &unstructured.Unstructured{}
		Expect(json.Unmarshal(data, obj)).To(Succeed())
		Expect(unstructured.SetNestedField(obj.Object, "something", "spec", "unknown")).To(Succeed())
		Expect(validator.Validate(obj.GroupVersionKind(), obj.Object)).ToNot(BeEmpty())
		Expect(validator.ValidateSpec(obj.GroupVersionKind(), obj.Object)).ToNot(BeEmpty())
	})

})
