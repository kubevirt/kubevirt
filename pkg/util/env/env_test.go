package env_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

	It("should parse typed values when set", func() {
		Expect(os.Setenv(key, " 1.5 ")).To(Succeed())
		value, ok := env.Float(key)
		Expect(ok).To(BeTrue())
		Expect(value).To(Equal(1.5))
	})

	It("should treat invalid values as absent", func() {
		Expect(os.Setenv(key, "not-a-number")).To(Succeed())
		_, ok := env.Float(key)
		Expect(ok).To(BeFalse())
	})

	It("should treat invalid uint64 values as absent", func() {
		Expect(os.Setenv(key, "not-a-number")).To(Succeed())
		_, ok := env.Uint64(key)
		Expect(ok).To(BeFalse())
	})

	It("should treat invalid bool values as absent", func() {
		Expect(os.Setenv(key, "not-bool")).To(Succeed())
		_, ok := env.Bool(key)
		Expect(ok).To(BeFalse())
	})
})
