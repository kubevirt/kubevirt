package bootstrap

import (
	"crypto/tls"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

var _ = Describe("cert-manager", func() {

	var certDir string

	BeforeEach(func() {
		var err error
		certDir, err = ioutil.TempDir("", "certs")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should return nil if no certificate exists", func() {
		certManager := NewFileCertificateManager(certDir)
		go certManager.Start()
		defer certManager.Stop()
		Consistently(func() *tls.Certificate {
			return certManager.Current()
		}, time.Second).Should(BeNil())
	})

	It("should load a certificate if it exists", func() {
		certManager := NewFileCertificateManager(certDir)
		writeCertsToDir(certDir)
		go certManager.Start()
		defer certManager.Stop()
		Eventually(func() *tls.Certificate {
			return certManager.Current()
		}, time.Second).Should(Not(BeNil()))
	})

	It("should load a certificate if it appears after the start", func() {
		certManager := NewFileCertificateManager(certDir)
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
		certManager := NewFileCertificateManager(certDir)
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

	AfterEach(func() {
		os.RemoveAll(certDir)
	})
})

func writeCertsToDir(dir string) {
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour*24*7)
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		"not",
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
