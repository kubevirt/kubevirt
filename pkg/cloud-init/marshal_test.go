package cloudinit

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cloud-Init Marshaling", func() {
	Context("NoCloudMetadata", func() {
		It("should marshal standard and custom metadata correctly", func() {
			ncm := &NoCloudMetadata{
				InstanceID:   "vmi-123",
				InstanceType: "small",
				CustomMetadata: map[string]string{
					"app_name":     "my-app",
					"complex_json": `{"foo": "bar"}`,
					"complex_yaml": "foo: bar\nbaz: qux",
				},
			}

			bytes, err := json.Marshal(ncm)
			Expect(err).ToNot(HaveOccurred())

			var result map[string]any
			err = json.Unmarshal(bytes, &result)
			Expect(err).ToNot(HaveOccurred())

			Expect(result["instance-id"]).To(Equal("vmi-123"))
			Expect(result["instance-type"]).To(Equal("small"))
			Expect(result["app_name"]).To(Equal("my-app"))

			Expect(result["complex_json"]).To(Equal(`{"foo": "bar"}`))
			Expect(result["complex_yaml"]).To(Equal("foo: bar\nbaz: qux"))
		})

		It("should fail if custom metadata key conflicts with reserved field", func() {
			ncm := &NoCloudMetadata{
				InstanceID: "vmi-123",
				CustomMetadata: map[string]string{
					"instance-id": "conflict",
				},
			}

			_, err := json.Marshal(ncm)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("custom metadata key 'instance-id' conflicts with reserved field"))
		})
	})

	Context("ConfigDriveMetadata", func() {
		It("should marshal standard and custom metadata correctly", func() {
			cdm := &ConfigDriveMetadata{
				InstanceID: "vmi-456",
				CustomMetadata: map[string]string{
					"project": "tester",
				},
			}

			bytes, err := json.Marshal(cdm)
			Expect(err).ToNot(HaveOccurred())

			var result map[string]interface{}
			err = json.Unmarshal(bytes, &result)
			Expect(err).ToNot(HaveOccurred())

			Expect(result["instance_id"]).To(Equal("vmi-456"))
			Expect(result["project"]).To(Equal("tester"))
		})
	})
})
