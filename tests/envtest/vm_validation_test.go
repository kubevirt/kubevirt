package envtest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
)

var _ = Describe("CRD Validation", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New()
		f.Start()
	})

	AfterEach(func() {
		f.Stop()
	})

	It("should reject a VM with a relative containerPath via CEL validation", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
		)
		vm.TypeMeta = metav1.TypeMeta{
			APIVersion: virtv1.StorageGroupVersion.String(),
			Kind:       "VirtualMachine",
		}

		jsonBytes, err := json.Marshal(vm)
		Expect(err).NotTo(HaveOccurred())

		By("injecting a containerPath volume with a relative path via raw JSON")
		var raw map[string]interface{}
		Expect(json.Unmarshal(jsonBytes, &raw)).To(Succeed())
		spec := raw["spec"].(map[string]interface{})
		template := spec["template"].(map[string]interface{})
		templateSpec := template["spec"].(map[string]interface{})
		templateSpec["volumes"] = []interface{}{
			map[string]interface{}{
				"name": "bad-vol",
				"containerPath": map[string]interface{}{
					"path": "relative/path",
				},
			},
		}
		patchedJSON, err := json.Marshal(raw)
		Expect(err).NotTo(HaveOccurred())

		result := f.VirtClient().RestClient().Post().
			Resource("virtualmachines").
			Namespace("default").
			Body(patchedJSON).
			SetHeader("Content-Type", "application/json").
			Do(ctx)

		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

		body, _ := result.Raw()
		Expect(string(body)).To(ContainSubstring("path must be absolute"),
			"CEL rule should reject a relative containerPath")
	})

	It("should reject a VM with an invalid structural schema", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
		)
		vm.TypeMeta = metav1.TypeMeta{
			APIVersion: virtv1.StorageGroupVersion.String(),
			Kind:       "VirtualMachine",
		}

		jsonBytes, err := json.Marshal(vm)
		Expect(err).NotTo(HaveOccurred())

		By("renaming a required field to break the structural schema")
		jsonString := strings.Replace(string(jsonBytes), "\"domain\"", "\"not-a-domain\"", 1)

		result := f.VirtClient().RestClient().Post().
			Resource("virtualmachines").
			Namespace("default").
			Body([]byte(jsonString)).
			SetHeader("Content-Type", "application/json").
			Do(ctx)

		statusCode := 0
		result.StatusCode(&statusCode)
		Expect(statusCode).To(Equal(http.StatusUnprocessableEntity),
			"CRD structural schema should reject a VM missing the required domain field")
	})
})
