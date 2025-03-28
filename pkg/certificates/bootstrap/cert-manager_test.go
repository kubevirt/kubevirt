package bootstrap

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

var _ = Describe("cert-manager", func() {
	Context("based on mounted files", func() {
		var certDir string
		var certFilePath string
		var keyFilePath string

		BeforeEach(func() {
			var err error
			certDir, err = os.MkdirTemp("", "certs")
			certFilePath = filepath.Join(certDir, "tls.crt")
			keyFilePath = filepath.Join(certDir, "tls.key")
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return nil if no certificate exists", func() {
			certManager := NewFileCertificateManager(certFilePath, keyFilePath)
			go certManager.Start()
			defer certManager.Stop()
			Consistently(func() *tls.Certificate {
				return certManager.Current()
			}, time.Second).Should(BeNil())
		})

		It("should load a certificate if it exists", func() {
			certManager := NewFileCertificateManager(certFilePath, keyFilePath)
			writeCertsToDir(certDir)
			go certManager.Start()
			defer certManager.Stop()
			Eventually(func() *tls.Certificate {
				return certManager.Current()
			}, time.Second).Should(Not(BeNil()))
		})

		It("should load a certificate even if cert and key file are in different directories", func() {
			writeCertsToDir(certDir)

			var err error
			newCertDir, err := os.MkdirTemp("", "certs")
			Expect(err).ToNot(HaveOccurred())
			newKeyDir, err := os.MkdirTemp("", "keys")
			Expect(err).ToNot(HaveOccurred())
			crt, err := os.ReadFile(certFilePath)
			Expect(err).ToNot(HaveOccurred())
			key, err := os.ReadFile(keyFilePath)
			Expect(err).ToNot(HaveOccurred())

			newCertFilePath := filepath.Join(newCertDir, "tls.crt")
			newKeyFilePath := filepath.Join(newKeyDir, "tls.key")
			Expect(os.WriteFile(newCertFilePath, crt, 0o777)).To(Succeed())
			Expect(os.WriteFile(newKeyFilePath, key, 0o777)).To(Succeed())

			certManager := NewFileCertificateManager(newCertFilePath, newKeyFilePath)
			go certManager.Start()
			defer certManager.Stop()
			Eventually(func() *tls.Certificate {
				return certManager.Current()
			}, time.Second).Should(Not(BeNil()))
		})

		It("should load a certificate if it appears after the start", func() {
			certManager := NewFileCertificateManager(certFilePath, keyFilePath)
			go certManager.Start()
			defer certManager.Stop()
			Consistently(func() *tls.Certificate {
				return certManager.Current()
			}, time.Second).Should(BeNil())
			writeCertsToDir(certDir)
			Eventually(func() *tls.Certificate {
				return certManager.Current()
			}, 3*time.Second).Should(Not(BeNil()))
		})

		It("should keep the latest certificate if it can't load new certs", func() {
			certManager := NewFileCertificateManager(certFilePath, keyFilePath)
			writeCertsToDir(certDir)
			go certManager.Start()
			defer certManager.Stop()
			Eventually(func() *tls.Certificate {
				return certManager.Current()
			}, time.Second).Should(Not(BeNil()))
			Expect(os.WriteFile(filepath.Join(certDir, CertBytesValue), []byte{}, 0o777)).To(Succeed())
			Consistently(func() *tls.Certificate {
				return certManager.Current()
			}, 2*time.Second).ShouldNot(BeNil())
		})

		Context("with fallback handling", func() {
			It("should return a fallback certificate if the is no certificate", func() {
				certManager := NewFallbackCertificateManager(NewFileCertificateManager(certFilePath, keyFilePath))
				go certManager.Start()
				defer certManager.Stop()
				Expect(certManager.Current().Leaf.Subject.CommonName).To(Equal("fallback.certificate.kubevirt.io"))
			})
			It("should return the real certificate if the is one", func() {
				kubevirtCache := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
				kubevirtCache.Add(&v1.KubeVirt{})
				certManager := NewFallbackCertificateManager(NewFileCertificateManager(certFilePath, keyFilePath))
				writeCertsToDir(certDir)
				go certManager.Start()
				defer certManager.Stop()
				Eventually(func() string {
					return certManager.Current().Leaf.Subject.CommonName
				}, time.Second).Should(Equal("loaded.certificate.kubevirt.io"))
			})
		})
		AfterEach(func() {
			os.RemoveAll(certDir)
		})
	})

	Context("based on a secret store", func() {
		var secretCache cache.Store

		BeforeEach(func() {
			secretCache = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		})
		It("should return nil if there is no certificate", func() {
			manager := NewSecretCertificateManager("name", "namespace", secretCache)
			Expect(manager.Current()).To(BeNil())
		})
		It("should load the certificate from a secret in the cache", func() {
			secretCache.Add(writeCertsToSecret("name", "namespace", "1"))
			manager := NewSecretCertificateManager("name", "namespace", secretCache)
			Expect(manager.Current()).ToNot(BeNil())
		})
		It("should update the certificate if the revision changes", func() {
			secretCache.Add(writeCertsToSecret("name", "namespace", "1"))
			manager := NewSecretCertificateManager("name", "namespace", secretCache)
			crt := manager.Current()
			Expect(crt).ToNot(BeNil())
			secretCache.Add(writeCertsToSecret("name", "namespace", "2"))
			newCrt := manager.Current()
			Expect(newCrt).ToNot(BeNil())
			Expect(newCrt).ToNot(Equal(crt))
		})
		It("should not update the certificate if the revision does not change", func() {
			secretCache.Add(writeCertsToSecret("name", "namespace", "1"))
			manager := NewSecretCertificateManager("name", "namespace", secretCache)
			crt := manager.Current()
			Expect(crt).ToNot(BeNil())
			secretCache.Add(writeCertsToSecret("name", "namespace", "1"))
			newCrt := manager.Current()
			Expect(newCrt).To(Equal(crt))
		})
	})
})

func writeCertsToDir(dir string) {
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour*24*7)
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		"loaded.certificate.kubevirt.io",
		"important",
		"this is",
		"cluster.local",
		nil,
		nil,
		time.Hour*24,
	)
	crt := cert.EncodeCertPEM(keyPair.Cert)
	key := cert.EncodePrivateKeyPEM(keyPair.Key)
	Expect(os.WriteFile(filepath.Join(dir, CertBytesValue), crt, 0o777)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(dir, KeyBytesValue), key, 0o777)).To(Succeed())
}

func writeCertsToSecret(name string, namespace string, revision string) *k8sv1.Secret {
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour*24*7)
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		"loaded.certificate.kubevirt.io",
		"important",
		"this is",
		"cluster.local",
		nil,
		nil,
		time.Hour*24,
	)
	crt := cert.EncodeCertPEM(keyPair.Cert)
	key := cert.EncodePrivateKeyPEM(keyPair.Key)
	return &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			ResourceVersion: revision,
		},
		Data: map[string][]byte{
			CertBytesValue: crt,
			KeyBytesValue:  key,
		},
	}
}
