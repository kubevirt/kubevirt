package instancetype

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/api/instancetype/v1alpha1"
)

var _ = Describe("instancetype compatibility", func() {
	Context("reading old ControllerRevision", func() {
		DescribeTable("should decode v1alpha1 instancetype from ControllerRevision", func(apiVersion string) {
			instancetypeSpec := v1alpha1.VirtualMachineInstancetypeSpec{
				CPU: v1alpha1.CPUInstancetype{
					Guest: 4,
				},
			}

			specBytes, err := json.Marshal(&instancetypeSpec)
			Expect(err).ToNot(HaveOccurred())

			revision := v1alpha1.VirtualMachineInstancetypeSpecRevision{
				APIVersion: apiVersion,
				Spec:       specBytes,
			}

			revisionBytes, err := json.Marshal(revision)
			Expect(err).ToNot(HaveOccurred())

			decoded, err := decodeOldInstancetypeRevisionObject(revisionBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).ToNot(BeNil())
			Expect(decoded.Spec.CPU).To(Equal(instancetypeSpec.CPU))
		},
			Entry("with APIVersion", v1alpha1.SchemeGroupVersion.String()),
			Entry("without APIVersion", ""),
		)

		DescribeTable("should decode v1alpha1 preference from ControllerRevision", func(apiVersion string) {
			preferredTopology := v1alpha1.PreferCores
			preferenceSpec := v1alpha1.VirtualMachinePreferenceSpec{
				CPU: &v1alpha1.CPUPreferences{
					PreferredCPUTopology: preferredTopology,
				},
			}

			specBytes, err := json.Marshal(&preferenceSpec)
			Expect(err).ToNot(HaveOccurred())

			revision := v1alpha1.VirtualMachinePreferenceSpecRevision{
				APIVersion: apiVersion,
				Spec:       specBytes,
			}

			revisionBytes, err := json.Marshal(revision)
			Expect(err).ToNot(HaveOccurred())

			decoded, err := decodeOldPreferenceRevisionObject(revisionBytes)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).ToNot(BeNil())
			Expect(decoded.Spec).To(Equal(preferenceSpec))

		},
			Entry("with APIVersion", v1alpha1.SchemeGroupVersion.String()),
			Entry("without APIVersion", ""),
		)
	})
})
