package env_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/util/env"
)

var _ = Describe("env", func() {
	const key = "KUBEVIRT_TEST_ENV_KEY"

	AfterEach(func() {
		Expect(os.Unsetenv(key)).To(Succeed())
	})

	It("should report unset keys as absent", func() {
		_, ok := env.Lookup(key)
		Expect(ok).To(BeFalse())
	})

	It("should parse raw strings into typed values", func() {
		parsedBool, err := env.Parse[bool]("true")
		Expect(err).ToNot(HaveOccurred())
		Expect(parsedBool).To(BeTrue())

		parsedInt, err := env.Parse[int64]("7")
		Expect(err).ToNot(HaveOccurred())
		Expect(parsedInt).To(Equal(int64(7)))

		parsedUint, err := env.Parse[uint64]("9")
		Expect(err).ToNot(HaveOccurred())
		Expect(parsedUint).To(Equal(uint64(9)))

		parsedQuantity, err := env.Parse[resource.Quantity]("1.5")
		Expect(err).ToNot(HaveOccurred())
		Expect(parsedQuantity).To(Equal(resource.MustParse("1.5")))
	})

	It("should parse typed values from the environment", func() {
		Expect(os.Setenv(key, " 1.5 ")).To(Succeed())
		value, err := env.Var[resource.Quantity]{Name: key}.LoadAndParse()
		Expect(err).ToNot(HaveOccurred())
		Expect(value.AsApproximateFloat64()).To(BeNumerically("~", 1.5))
	})

	It("should error when a Var is unset", func() {
		_, err := env.Var[bool]{Name: key}.LoadAndParse()
		Expect(err).To(HaveOccurred())
	})

	DescribeTable("should error when a Var value does not parse", func(raw string, parse func() error) {
		Expect(os.Setenv(key, raw)).To(Succeed())
		Expect(parse()).To(HaveOccurred())
	},
		Entry("quantity", "not-a-number", func() error {
			_, err := env.Var[resource.Quantity]{Name: key}.LoadAndParse()
			return err
		}),
		Entry("uint64", "not-a-number", func() error {
			_, err := env.Var[uint64]{Name: key}.LoadAndParse()
			return err
		}),
		Entry("bool", "not-bool", func() error {
			_, err := env.Var[bool]{Name: key}.LoadAndParse()
			return err
		}),
		Entry("int64", "0.07", func() error {
			_, err := env.Var[int64]{Name: key}.LoadAndParse()
			return err
		}),
	)

	It("should reject bindings that do not parse as their declared type", func() {
		_, err := env.Var[int64]{Name: key}.Binding("0.07").Parse()
		Expect(err).To(HaveOccurred())
	})

	It("should return the parsed value from a binding", func() {
		parsed, err := env.Var[int64]{Name: key}.Binding("7").Parse()
		Expect(err).ToNot(HaveOccurred())
		Expect(parsed).To(Equal(int64(7)))
	})
})
