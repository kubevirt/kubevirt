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
