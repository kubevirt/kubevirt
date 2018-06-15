package hooks_test

import (
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/hooks"
)

var _ = Describe("HooksAPI", func() {
	Context("test HookSidecarsList structure and helper functions", func() {
		It("by unmarshalling of VM annotations", func() {
			expectedHookSidecarList := hooks.HookSidecarList{
				hooks.HookSidecar{
					Image:           "some-image:v1",
					ImagePullPolicy: "IfNotPresent",
				},
				hooks.HookSidecar{
					Image:           "another-image:v1",
					ImagePullPolicy: "Always",
				},
			}
			hookSidecarListAnnotation := `
                [
                  {
                    "image": "some-image:v1",
                    "imagePullPolicy": "IfNotPresent"
                  },
                  {
                    "image": "another-image:v1",
                    "imagePullPolicy": "Always"
                  }
                ]
`
			hookSidecarList, err := hooks.UnmarshalHookSidecarList(hookSidecarListAnnotation)
			Expect(err).ToNot(HaveOccurred())
			Expect(reflect.DeepEqual(hookSidecarList, expectedHookSidecarList)).To(Equal(true))
		})
	})
})
