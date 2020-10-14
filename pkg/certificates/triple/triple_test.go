package triple

import (
	"crypto/x509"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cert library", func() {
	Context("when NewCA is called", func() {
		var (
			name     string
			duration time.Duration
			now      time.Time
		)
		BeforeEach(func() {
			now = time.Now()
			name = "foo-bar-name"
			duration = time.Minute
		})
		It("should generate key and CA cert with expected fields", func() {

			keyAndCert, err := NewCA(name, duration)
			Expect(err).ToNot(HaveOccurred(), "should succeed generating CA")

			privateKey := keyAndCert.Key
			caCert := keyAndCert.Cert

			Expect(privateKey).ToNot(BeNil(), "should generate a private key")
			Expect(caCert).ToNot(BeNil(), "should generate a CA certificate")
			Expect(caCert.SerialNumber.Int64()).To(Equal(int64(0)), "should have zero as serial number")
			Expect(caCert.Subject.CommonName).To(Equal(name), "should take CommonName from name field")
			Expect(caCert.NotBefore).To(BeTemporally("~", now.UTC(), time.Second), "should set NotBefore to now")
			Expect(caCert.NotAfter).To(BeTemporally("~", now.Add(duration).UTC(), time.Second), "should  set NotAfter to now + duration")
			Expect(caCert.KeyUsage).To(Equal(x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature|x509.KeyUsageCertSign), "should set proper KeyUsage")
			Expect(caCert.BasicConstraintsValid).To(BeTrue(), "should mark it as BasicConstraintsValid")
			Expect(caCert.IsCA).To(BeTrue(), "should mark it as CA")
			Expect(caCert.SubjectKeyId).ToNot(BeEmpty(), "should include a SKI")
		})

	})
})
