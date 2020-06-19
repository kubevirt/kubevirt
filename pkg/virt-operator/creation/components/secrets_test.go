package components

import (
	cryptorand "crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

var _ = Describe("Certificate Management", func() {
	Context("CA certificate bundle", func() {

		It("should drop expired CAs", func() {
			now := time.Now()
			current := NewSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				NewSelfSignedCert(now.Add(-20*time.Minute), now.Add(-10*time.Minute)),
			}
			givenBundle := CACertsToBundle(given)
			expectBundle := CACertsToBundle([]*tls.Certificate{current})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should be properly appended when within the overlap period", func() {
			now := time.Now()
			current := NewSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				NewSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := CACertsToBundle(given)
			expectBundle := CACertsToBundle([]*tls.Certificate{current, given[0]})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should kick out the first CA cert if it is outside of the overlap period", func() {
			now := time.Now()
			current := NewSelfSignedCert(now.Add(-3*time.Minute), now.Add(1*time.Hour))
			given := []*tls.Certificate{
				NewSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := CACertsToBundle(given)
			expectBundle := CACertsToBundle([]*tls.Certificate{current})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should kick out a CA certs which are outside of the overlap period", func() {
			now := time.Now()
			current := NewSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				NewSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
				NewSelfSignedCert(now.Add(-10*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := CACertsToBundle(given)
			expectBundle := CACertsToBundle([]*tls.Certificate{current, given[1]})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should kick out multiple CA certs which are outside of the overlap period", func() {
			now := time.Now()
			current := NewSelfSignedCert(now.Add(-5*time.Minute), now.Add(-5*time.Minute).Add(1*time.Hour))
			given := []*tls.Certificate{
				NewSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
				NewSelfSignedCert(now.Add(-10*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := CACertsToBundle(given)
			expectBundle := CACertsToBundle([]*tls.Certificate{current})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should ensure that the current CA is not added over and over again", func() {
			now := time.Now()
			current := NewSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				NewSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
				current,
				current,
			}
			givenBundle := CACertsToBundle(given)
			expectBundle := CACertsToBundle([]*tls.Certificate{current, given[0]})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should be protected against misuse by cropping big arrays", func() {
			now := time.Now()
			current := NewSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{}
			for i := 1; i < 20; i++ {
				given = append(given, NewSelfSignedCert(now.Add(-1*time.Minute), now.Add(1*time.Hour)))
			}
			givenBundle := CACertsToBundle(given)
			_, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(11))
		})

	})

	It("should set the right namespaces on the certificate secrets", func() {
		secrets := NewCertSecrets("install_namespace", "operator_namespace")
		for _, secret := range secrets[:len(secrets)-1] {
			Expect(secret.Namespace).To(Equal("install_namespace"))
			Expect(secret.Name).ToNot(Equal(VirtOperatorCertSecretName))
		}
		Expect(secrets[len(secrets)-1].Namespace).To(Equal("operator_namespace"))
	})

	It("should create the kubevirt-ca configmap for the right namespace", func() {
		configMap := NewKubeVirtCAConfigMap("namespace")
		Expect(configMap.Namespace).To(Equal("namespace"))
	})

	It("should populate secrets with certificates", func() {
		secrets := NewCertSecrets("install_namespace", "operator_namespace")
		caSecret := NewCACertSecret("operator_namespace")
		Expect(PopulateSecretWithCertificate(caSecret, nil, &v1.Duration{Duration: 1 * time.Hour})).To(Succeed())
		Expect(caSecret.Data).To(HaveKey(bootstrap.CertBytesValue))
		Expect(caSecret.Data).To(HaveKey(bootstrap.KeyBytesValue))

		caCert, err := LoadCertificates(caSecret)
		Expect(err).ToNot(HaveOccurred())

		for _, secret := range secrets {
			Expect(PopulateSecretWithCertificate(secret, caCert, &v1.Duration{Duration: 1 * time.Hour})).To(Succeed())
			Expect(secret.Data).To(HaveKey(bootstrap.CertBytesValue))
			Expect(secret.Data).To(HaveKey(bootstrap.KeyBytesValue))
			_, err = LoadCertificates(secret)
			Expect(err).ToNot(HaveOccurred())
		}
	})
})

// NewSelfSignedCert creates a CA certificate
func NewSelfSignedCert(notBefore time.Time, notAfter time.Time) *tls.Certificate {
	key, err := certutil.NewPrivateKey()
	Expect(err).ToNot(HaveOccurred())
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   "who",
			Organization: []string{"cares"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, key.Public(), key)
	Expect(err).ToNot(HaveOccurred())
	leaf, err := x509.ParseCertificate(certDERBytes)
	Expect(err).ToNot(HaveOccurred())
	keyBytes := certutil.EncodePrivateKeyPEM(key)
	Expect(err).ToNot(HaveOccurred())

	crtBytes := certutil.EncodeCertPEM(leaf)
	crt, err := tls.X509KeyPair(crtBytes, keyBytes)
	Expect(err).ToNot(HaveOccurred())
	crt.Leaf = leaf
	return &crt
}

func CACertsToBundle(crts []*tls.Certificate) []byte {
	var caBundle []byte
	for _, crt := range crts {
		caBundle = append(caBundle, certutil.EncodeCertPEM(crt.Leaf)...)
	}
	return caBundle
}
