package v1beta1

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ResourceName             = "kubevirt-hyperconverged"
	ResourceInvalidNamespace = "an-arbitrary-namespace"
	HcoValidNamespace        = "kubevirt-hyperconverged"
)

var _ = Describe("Hyperconverged Webhooks", func() {
	Context("Check validating webhook", func() {
		BeforeEach(func() {
			os.Setenv("OPERATOR_NAMESPACE", HcoValidNamespace)
		})

		cr := &HyperConverged{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ResourceName,
				Namespace: HcoValidNamespace,
			},
			Spec: HyperConvergedSpec{},
		}

		It("should accept creation of a resource with a valid namespace", func() {
			err := cr.ValidateCreate()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should reject creation of a resource with an arbitrary namespace", func() {
			cr.ObjectMeta.Namespace = ResourceInvalidNamespace
			err := cr.ValidateCreate()
			Expect(err).To(HaveOccurred())
		})
	})
})
