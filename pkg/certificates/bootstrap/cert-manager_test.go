package bootstrap

import (
	"crypto/tls"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

var _ = Describe("cert-manager", func() {

	var certDir string
	var certFilePath string
	var keyFilePath string

	BeforeEach(func() {
		var err error
		certDir, err = ioutil.TempDir("", "certs")
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
		newCertDir, err := ioutil.TempDir("", "certs")
		Expect(err).ToNot(HaveOccurred())
		newKeyDir, err := ioutil.TempDir("", "keys")
		Expect(err).ToNot(HaveOccurred())
		crt, err := ioutil.ReadFile(certFilePath)
		Expect(err).ToNot(HaveOccurred())
		key, err := ioutil.ReadFile(keyFilePath)
		Expect(err).ToNot(HaveOccurred())

		newCertFilePath := filepath.Join(newCertDir, "tls.crt")
		newKeyFilePath := filepath.Join(newKeyDir, "tls.key")
		Expect(ioutil.WriteFile(newCertFilePath, crt, 0777)).To(Succeed())
		Expect(ioutil.WriteFile(newKeyFilePath, key, 0777)).To(Succeed())

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
		Expect(ioutil.WriteFile(filepath.Join(certDir, CertBytesValue), []byte{}, 0777)).To(Succeed())
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
	Expect(ioutil.WriteFile(filepath.Join(dir, CertBytesValue), crt, 0777)).To(Succeed())
	Expect(ioutil.WriteFile(filepath.Join(dir, KeyBytesValue), key, 0777)).To(Succeed())
}
