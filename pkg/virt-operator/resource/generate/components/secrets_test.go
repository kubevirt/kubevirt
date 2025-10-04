package components

import (
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	certutil "kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

var _ = Describe("Certificate Management", func() {
	Context("CA certificate bundle", func() {

		It("should drop expired CAs", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				newSelfSignedCert(now.Add(-20*time.Minute), now.Add(-10*time.Minute)),
			}
			givenBundle := caCertsToBundle(given)
			expectBundle := caCertsToBundle([]*tls.Certificate{current})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should be properly appended when within the overlap period", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				newSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := caCertsToBundle(given)
			expectBundle := caCertsToBundle([]*tls.Certificate{current, given[0]})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should kick out the first CA cert if it is outside of the overlap period", func() {
			now := time.Now()
			current := newSelfSignedCert(now.Add(-3*time.Minute), now.Add(1*time.Hour))
			given := []*tls.Certificate{
				newSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := caCertsToBundle(given)
			expectBundle := caCertsToBundle([]*tls.Certificate{current})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should kick out a CA cert which are outside of the overlap period", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				newSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
				newSelfSignedCert(now.Add(-10*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := caCertsToBundle(given)
			expectBundle := caCertsToBundle([]*tls.Certificate{current, given[1]})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should kick out multiple CA certs which are outside of the overlap period", func() {
			now := time.Now()
			current := newSelfSignedCert(now.Add(-5*time.Minute), now.Add(-5*time.Minute).Add(1*time.Hour))
			given := []*tls.Certificate{
				newSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
				newSelfSignedCert(now.Add(-10*time.Minute), now.Add(1*time.Hour)),
				newSelfSignedCert(now.Add(-5*time.Minute), now.Add(1*time.Hour)),
			}
			givenBundle := caCertsToBundle(given)
			expectBundle := caCertsToBundle([]*tls.Certificate{current})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(1))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should ensure that the current CA is not added over and over again", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{
				newSelfSignedCert(now.Add(-20*time.Minute), now.Add(1*time.Hour)),
				current,
				current,
			}
			givenBundle := caCertsToBundle(given)
			expectBundle := caCertsToBundle([]*tls.Certificate{current, given[0]})
			bundle, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))
			Expect(bundle).To(Equal(expectBundle))
		})

		It("should be protected against misuse by cropping big arrays", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))
			given := []*tls.Certificate{}
			for i := 1; i < 60; i++ {
				given = append(given, newSelfSignedCert(now.Add(-1*time.Minute), now.Add(1*time.Hour)))
			}
			givenBundle := caCertsToBundle(given)
			_, count, err := MergeCABundle(current, givenBundle, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(51))
		})

		It("should keep newest certificates when truncating", func() {
			now := time.Now()
			current := newSelfSignedCert(now.Add(-10*time.Second), now.Add(1*time.Hour))
			certificates := make([]*tls.Certificate, 0, 60)

			// Create 60 certificates (half valid, half expired)
			for i := range 60 {
				if i%2 == 0 {
					certTime := now.Add(-time.Duration(i+1) * 30 * time.Second)
					certificates = append(certificates, newSelfSignedCert(certTime, now.Add(1*time.Hour)))
				} else {
					certificates = append(certificates, newSelfSignedCert(now.Add(-2*time.Hour), now.Add(-30*time.Minute)))
				}
			}

			givenBundle := caCertsToBundle(certificates)
			bundle, count, err := MergeCABundle(current, givenBundle, 1*time.Hour) // Long overlap duration
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(31)) // 30 valid certs from bundle + 1 current cert

			certs, parseErr := certutil.ParseCertsPEM(bundle)
			Expect(parseErr).ToNot(HaveOccurred())
			Expect(certs).To(HaveLen(31))

			// Verify all certificates are still valid
			for _, cert := range certs {
				Expect(cert.NotAfter.Before(now)).To(BeFalse(), "Certificate should not be expired")
			}

			// First certificate should be the current one
			Expect(certs[0]).To(Equal(current.Leaf))

			// Verify certificates are sorted by NotBefore in descending order (newest first)
			// Skip the first certificate (current) and check the remaining ones
			for i := 1; i < len(certs)-1; i++ {
				Expect(certs[i].NotBefore.After(certs[i+1].NotBefore) || certs[i].NotBefore.Equal(certs[i+1].NotBefore)).
					To(BeTrue(), "Certificates should be sorted by NotBefore in descending order (newest first)")
			}
		})

		It("should handle malformed CA bundle gracefully", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))

			// Test with invalid PEM data
			invalidBundle := []byte("-----BEGIN CERTIFICATE-----\nINVALID DATA\n-----END CERTIFICATE-----")

			bundle, count, err := MergeCABundle(current, invalidBundle, 2*time.Minute)
			Expect(bundle).To(BeEmpty())
			Expect(count).To(BeZero())
			Expect(err).To(HaveOccurred())
		})

		It("should immediately suggest a rotation if the cert is not signed by the provided CA", func() {
			now := time.Now()
			current := newSelfSignedCert(now, now.Add(1*time.Hour))
			ca := newSelfSignedCert(now, now.Add(1*time.Hour))
			renewal := &v1.Duration{Duration: 4 * time.Hour}
			deadline := NextRotationDeadline(current, ca, renewal, nil)
			Expect(deadline.Before(time.Now())).To(BeTrue())
		})

		It("should set notBefore on the certificate to notBefore value of the CA certificate ", func() {
			duration := &v1.Duration{Duration: 5 * time.Hour}
			caSecrets := NewCACertSecrets("test")
			var caSecret *v12.Secret
			for _, ca := range caSecrets {
				if ca.Name == KubeVirtCASecretName {
					caSecret = ca
				}
			}
			Expect(PopulateSecretWithCertificate(caSecret, nil, duration)).To(Succeed())
			caCrt, err := LoadCertificates(caSecret)
			Expect(err).NotTo(HaveOccurred())
			crtSecret := NewCertSecrets("test", "test")[0]
			Expect(PopulateSecretWithCertificate(crtSecret, caCrt, duration)).To(Succeed())
			crt, err := LoadCertificates(crtSecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(crt.Leaf.NotBefore).To(Equal(caCrt.Leaf.NotBefore))
		})

		DescribeTable("should set the notAfter on the certificate according to the supplied duration", func(caDuration time.Duration) {
			crtDuration := &v1.Duration{Duration: 2 * time.Hour}
			caSecrets := NewCACertSecrets("test")
			var caSecret *v12.Secret
			for _, ca := range caSecrets {
				if ca.Name == KubeVirtCASecretName {
					caSecret = ca
				}
			}
			now := time.Now()
			Expect(PopulateSecretWithCertificate(caSecret, nil, &v1.Duration{Duration: caDuration})).To(Succeed())
			caCrt, err := LoadCertificates(caSecret)
			Expect(err).NotTo(HaveOccurred())
			crtSecret := NewCertSecrets("test", "test")[0]
			Expect(PopulateSecretWithCertificate(crtSecret, caCrt, crtDuration)).To(Succeed())
			crt, err := LoadCertificates(crtSecret)
			Expect(err).NotTo(HaveOccurred())

			Expect(crt.Leaf.NotAfter.Unix()).To(BeNumerically("==", now.Add(crtDuration.Duration).Unix(), 10))
		},
			Entry("with a long valid CA", 24*time.Hour),
			Entry("with a CA which expires before the certificate rotation", 1*time.Hour),
		)

		DescribeTable("should suggest a rotation on the certificate according to its expiration", func(caDuration time.Duration) {
			crtDuration := &v1.Duration{Duration: 2 * time.Hour}
			crtRenewBefore := &v1.Duration{Duration: 1 * time.Hour}
			caSecrets := NewCACertSecrets("test")
			var caSecret *v12.Secret
			for _, ca := range caSecrets {
				if ca.Name == KubeVirtCASecretName {
					caSecret = ca
				}
			}
			Expect(PopulateSecretWithCertificate(caSecret, nil, &v1.Duration{Duration: caDuration})).To(Succeed())
			caCrt, err := LoadCertificates(caSecret)
			now := time.Now()
			Expect(err).NotTo(HaveOccurred())
			crtSecret := NewCertSecrets("test", "test")[0]
			Expect(PopulateSecretWithCertificate(crtSecret, caCrt, crtDuration)).To(Succeed())
			crt, err := LoadCertificates(crtSecret)
			Expect(err).NotTo(HaveOccurred())

			deadline := now.Add(time.Hour)
			// Generating certificates may take a little bit of time to execute (entropy, ...). Since we can't
			// inject a fake time into the foreign code which generates the certificates, allow a generous diff of three
			// seconds.
			Expect(NextRotationDeadline(crt, caCrt, crtRenewBefore, nil).Unix()).To(BeNumerically("==", deadline.Unix(), 3))
		},
			Entry("with a long valid CA", 24*time.Hour),
			Entry("with a CA which expires before the certificate rotation", 1*time.Hour),
		)

		DescribeTable("should successfully sign with the current CA the certificate for", func(scretName string) {
			duration := &v1.Duration{Duration: 5 * time.Hour}
			caSecrets := NewCACertSecrets("test")
			var caSecret *v12.Secret
			for _, ca := range caSecrets {
				if ca.Name == KubeVirtCASecretName {
					caSecret = ca
				}
			}
			Expect(PopulateSecretWithCertificate(caSecret, nil, duration)).To(Succeed())
			caCrt, err := LoadCertificates(caSecret)
			Expect(err).NotTo(HaveOccurred())
			var crtSecret *v12.Secret
			for _, s := range NewCertSecrets("test", "test") {
				if s.Name == scretName {
					crtSecret = s
					break
				}
			}
			Expect(crtSecret).ToNot(BeNil())
			Expect(PopulateSecretWithCertificate(crtSecret, caCrt, duration)).To(Succeed())
			crt, err := LoadCertificates(crtSecret)
			Expect(err).ToNot(HaveOccurred())
			Expect(crt).ToNot(BeNil())
		},
			Entry("virt-handler", VirtHandlerCertSecretName),
			Entry("virt-controller", VirtControllerCertSecretName),
			Entry("virt-api", VirtApiCertSecretName),
			Entry("virt-operator", VirtOperatorCertSecretName),
			Entry("virt-exportproxy", VirtExportProxyCertSecretName),
		)

		It("should suggest earlier rotation if CA expires before cert", func() {
			caDuration := 6 * time.Hour
			crtDuration := &v1.Duration{Duration: 24 * time.Hour}
			crtRenewBefore := &v1.Duration{Duration: 18 * time.Hour}
			caSecrets := NewCACertSecrets("test")
			var caSecret *v12.Secret
			for _, ca := range caSecrets {
				if ca.Name == KubeVirtCASecretName {
					caSecret = ca
				}
			}
			Expect(PopulateSecretWithCertificate(caSecret, nil, &v1.Duration{Duration: caDuration})).To(Succeed())
			caCrt, err := LoadCertificates(caSecret)
			now := time.Now()
			Expect(err).NotTo(HaveOccurred())
			crtSecret := NewCertSecrets("test", "test")[0]
			Expect(PopulateSecretWithCertificate(crtSecret, caCrt, crtDuration)).To(Succeed())
			crt, err := LoadCertificates(crtSecret)
			Expect(err).NotTo(HaveOccurred())

			deadline := now.Add(6 * time.Hour)
			// Generating certificates may take a little bit of time to execute (entropy, ...). Since we can't
			// inject a fake time into the foreign code which generates the certificates, allow a generous diff of three
			// seconds.
			Expect(NextRotationDeadline(crt, caCrt, crtRenewBefore, nil).Unix()).To(BeNumerically("==", deadline.Unix(), 3))
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
		configMaps := NewCAConfigMaps("namespace")
		var configMap *v12.ConfigMap
		for _, cm := range configMaps {
			if cm.Name == KubeVirtCASecretName {
				configMap = cm
			}
		}
		Expect(configMap.Namespace).To(Equal("namespace"))
	})

	It("should populate secrets with certificates", func() {
		secrets := NewCertSecrets("install_namespace", "operator_namespace")
		caSecrets := NewCACertSecrets("test")
		var caSecret *v12.Secret
		for _, ca := range caSecrets {
			if ca.Name == KubeVirtCASecretName {
				caSecret = ca
			}
		}
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

	Context("filterValidCertificates function", func() {
		It("should filter out expired certificates", func() {
			now := time.Now()
			validCert1 := newSelfSignedCert(now.Add(-1*time.Hour), now.Add(1*time.Hour))
			expiredCert := newSelfSignedCert(now.Add(-2*time.Hour), now.Add(-30*time.Minute))
			validCert2 := newSelfSignedCert(now.Add(-30*time.Minute), now.Add(30*time.Minute))

			certs := []*x509.Certificate{validCert1.Leaf, expiredCert.Leaf, validCert2.Leaf}
			filtered := filterValidCertificates(certs, now, 10)

			Expect(filtered).To(HaveLen(2))
			// Should be sorted by NotBefore descending (newest first)
			Expect(filtered[0]).To(Equal(validCert2.Leaf))
			Expect(filtered[1]).To(Equal(validCert1.Leaf))
		})

		It("should truncate to maxCount when limit is exceeded", func() {
			now := time.Now()
			certs := make([]*x509.Certificate, 0, 15)

			// Create 15 valid certificates
			for i := range 15 {
				cert := newSelfSignedCert(now.Add(-time.Duration(i)*time.Minute), now.Add(1*time.Hour))
				certs = append(certs, cert.Leaf)
			}

			// Filter with maxCount = 10
			filtered := filterValidCertificates(certs, now, 10)

			Expect(filtered).To(HaveLen(10))
			// Should keep the 10 newest certificates
			for i := range 10 {
				expectedTime := now.Add(-time.Duration(i) * time.Minute)
				Expect(filtered[i].NotBefore.Unix()).To(BeNumerically("==", expectedTime.Unix(), 1))
			}
		})

		It("should return all certificates when under maxCount limit", func() {
			now := time.Now()
			certs := make([]*x509.Certificate, 0, 5)

			// Create 5 valid certificates
			for i := range 5 {
				cert := newSelfSignedCert(now.Add(-time.Duration(i)*time.Minute), now.Add(1*time.Hour))
				certs = append(certs, cert.Leaf)
			}

			// Filter with maxCount = 10
			filtered := filterValidCertificates(certs, now, 10)

			Expect(filtered).To(HaveLen(5))
		})

		It("should handle empty certificate slice", func() {
			now := time.Now()
			certs := []*x509.Certificate{}

			filtered := filterValidCertificates(certs, now, 10)

			Expect(filtered).To(BeEmpty())
		})
	})
})

// newSelfSignedCert creates a CA certificate
func newSelfSignedCert(notBefore time.Time, notAfter time.Time) *tls.Certificate {
	key, err := certutil.NewECDSAPrivateKey()
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

	r := rand.New(rand.NewSource(time.Now().Unix()))
	certDERBytes, err := x509.CreateCertificate(r, &tmpl, &tmpl, key.Public(), key)
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

func caCertsToBundle(crts []*tls.Certificate) []byte {
	var caBundle []byte
	for _, crt := range crts {
		caBundle = append(caBundle, certutil.EncodeCertPEM(crt.Leaf)...)
	}
	return caBundle
}
