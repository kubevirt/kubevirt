package operands

import (
	"errors"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("HyperConverged Ensure Result", func() {

	Context("HyperConverged Ensure Result", func() {
		kv, err := NewKubeVirt(&hcov1beta1.HyperConverged{})
		Expect(err).ToNot(HaveOccurred())

		It("should create new EnsureResult with default values", func() {
			er := NewEnsureResult(kv)

			Expect(er.Type).To(Equal("KubeVirt"))
			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).To(BeNil())
		})

		It("Should update Name", func() {
			er := NewEnsureResult(kv)
			er.SetName("a name")

			Expect(er.Name).To(Equal("a name"))
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).To(BeNil())
		})

		It("Should update UpgradeDone", func() {
			er := NewEnsureResult(kv)
			er.SetUpgradeDone(true)

			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeTrue())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).To(BeNil())

			er.SetUpgradeDone(false)

			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).To(BeNil())
		})

		It("Should set created", func() {
			er := NewEnsureResult(kv)
			er.SetCreated()

			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeTrue())
			Expect(er.Err).To(BeNil())
		})

		It("Should set updated", func() {
			er := NewEnsureResult(kv)
			er.SetUpdated()
			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeTrue())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).To(BeNil())
		})

		It("Should set overwritten", func() {
			er := NewEnsureResult(kv)
			er.SetOverwritten()
			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeTrue())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).To(BeNil())
		})

		It("Should set Error", func() {
			er := NewEnsureResult(kv)
			er.Error(errors.New("a test error"))

			Expect(er.Name).To(BeEmpty())
			Expect(er.UpgradeDone).To(BeFalse())
			Expect(er.Updated).To(BeFalse())
			Expect(er.Overwritten).To(BeFalse())
			Expect(er.Created).To(BeFalse())
			Expect(er.Err).ToNot(BeNil())
		})

		It("Should use the builder pattern", func() {
			er := NewEnsureResult(kv).
				Error(errors.New("a test error")).
				SetUpdated().
				SetOverwritten().
				SetCreated().
				SetUpgradeDone(true).
				SetName("a name")

			Expect(er.Name).To(Equal("a name"))
			Expect(er.UpgradeDone).To(BeTrue())
			Expect(er.Updated).To(BeTrue())
			Expect(er.Overwritten).To(BeTrue())
			Expect(er.Created).To(BeTrue())
			Expect(er.Err).ToNot(BeNil())
		})
	})
})
