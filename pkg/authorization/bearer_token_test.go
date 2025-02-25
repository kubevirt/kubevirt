package authorization_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/authorization"
)

var _ = Describe("BearerToken", func() {
	const tokenPath = "/tmp/fake-token"

	var tmpFile *os.File

	BeforeEach(func() {
		var err error

		tmpFile, err = os.Create(tokenPath)
		Expect(err).ToNot(HaveOccurred())

		_, err = tmpFile.Write([]byte("test-secret-key"))
		Expect(err).ToNot(HaveOccurred())

		os.Setenv(authorization.TokenPathEnvVar, tokenPath)
	})

	AfterEach(func() {
		os.Remove(tokenPath)
		os.Unsetenv(authorization.TokenPathEnvVar)
		authorization.RefreshSecretKey()
	})

	Context("with a valid ServiceAccount token", func() {
		It("should create and validate a JWT token", func() {
			token, err := authorization.CreateToken()
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			valid, err := authorization.ValidateToken(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(valid).To(BeTrue())
		})
	})

	Context("with a valid in-memory token", func() {
		It("should create and validate a JWT token", func() {
			os.Remove(tokenPath)
			os.Setenv(authorization.TokenPathEnvVar, "random-path")

			token, err := authorization.CreateToken()
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			valid, err := authorization.ValidateToken(token)
			Expect(err).ToNot(HaveOccurred())
			Expect(valid).To(BeTrue())
		})
	})

	Context("with an invalid token", func() {
		It("should fail validation for malformed token", func() {
			valid, err := authorization.ValidateToken("invalid-token")
			Expect(err).To(HaveOccurred())
			Expect(valid).To(BeFalse())
		})

		It("should fail validation for empty token", func() {
			valid, err := authorization.ValidateToken("")
			Expect(err).To(HaveOccurred())
			Expect(valid).To(BeFalse())
		})

		It("should fail validation for old token", func() {
			token, err := authorization.CreateToken()
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			authorization.RefreshSecretKey()

			_, err = tmpFile.Write([]byte("new-test-secret-key"))
			Expect(err).ToNot(HaveOccurred())

			valid, err := authorization.ValidateToken(token)
			Expect(err).To(HaveOccurred())
			Expect(valid).To(BeFalse())
		})

		It("should fail validation for old token using in-memory token", func() {
			os.Remove(tokenPath)
			os.Setenv(authorization.TokenPathEnvVar, "random-path")

			token, err := authorization.CreateToken()
			Expect(err).ToNot(HaveOccurred())
			Expect(token).ToNot(BeEmpty())

			authorization.RefreshSecretKey()

			valid, err := authorization.ValidateToken(token)
			Expect(err).To(HaveOccurred())
			Expect(valid).To(BeFalse())
		})
	})
})
