package bootstrap

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"k8s.io/client-go/util/certificate"

	"kubevirt.io/kubevirt/pkg/certificates/triple"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

const (
	CertBytesValue = "tls.crt"
	KeyBytesValue  = "tls.key"
)

type FileCertificateManager struct {
	stopCh             chan struct{}
	certAccessLock     sync.Mutex
	stopped            bool
	cert               *tls.Certificate
	certDir            string
	errorRetryInterval time.Duration
}

// NewFallbackCertificateManager returns a certificate manager which can fall back to a self signed certificate,
// if there is currently no kubevirt installation present on the cluster. This helps dealing with situations where e.g.
// readiness probes try to access an API which can't right now provide a fully managed certificate.
// virt-operator is the main recipient of this manager, since the certificate management infrastructure is not always
// already present when virt-operator gets created.
func NewFallbackCertificateManager(certManager certificate.Manager) *FallbackCertificateManager {
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour*24*7)
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		"fallback.certificate.kubevirt.io",
		"fallback",
		"fallback",
		"cluster.local",
		nil,
		nil,
		time.Hour*24*356*10,
	)
	crt, err := tls.X509KeyPair(cert.EncodeCertPEM(keyPair.Cert), cert.EncodePrivateKeyPEM(keyPair.Key))
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Failed to generate a fallback certificate.")
	}
	crt.Leaf = keyPair.Cert

	return &FallbackCertificateManager{
		certManager:         certManager,
		fallbackCertificate: &crt,
	}
}

type FallbackCertificateManager struct {
	certManager         certificate.Manager
	fallbackCertificate *tls.Certificate
}

func (f *FallbackCertificateManager) Start() {
	f.certManager.Start()
}

func (f *FallbackCertificateManager) Stop() {
	f.certManager.Stop()
}

func (f *FallbackCertificateManager) Current() *tls.Certificate {
	crt := f.certManager.Current()
	if crt != nil {
		return crt
	}
	return f.fallbackCertificate
}

func (f *FallbackCertificateManager) ServerHealthy() bool {
	return f.certManager.ServerHealthy()
}

func NewFileCertificateManager(certDir string) *FileCertificateManager {
	return &FileCertificateManager{certDir: certDir, stopCh: make(chan struct{}, 1), errorRetryInterval: 1 * time.Minute}
}

func (f *FileCertificateManager) Start() {
	objectUpdated := make(chan struct{}, 1)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Failed to create an inotify watcher")
	}
	defer watcher.Close()
	err = watcher.Add(f.certDir)
	if err != nil {
		log.DefaultLogger().Reason(err).Criticalf("Failed to establish a watch on %s", f.certDir)
	}

	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}
				select {
				case objectUpdated <- struct{}{}:
				default:
					log.DefaultLogger().V(5).Infof("Dropping redundant wakeup for cert reload")
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.DefaultLogger().Reason(err).Errorf("An error occured when watching %s", f.certDir)
			}
		}
	}()

	// ensure we load the certificates on startup
	objectUpdated <- struct{}{}

sync:
	for {
		select {
		case <-objectUpdated:
			if err := f.rotateCerts(); err != nil {
				go func() {
					time.Sleep(f.errorRetryInterval)
					select {
					case objectUpdated <- struct{}{}:
					default:
						log.DefaultLogger().V(5).Infof("Dropping redundant wakeup for cert reload")
					}
				}()
			}
		case <-f.stopCh:
			break sync
		}
	}
}

func (f *FileCertificateManager) Stop() {
	f.certAccessLock.Lock()
	defer f.certAccessLock.Unlock()
	if f.stopped {
		return
	}
	close(f.stopCh)
	f.stopped = true
}

func (f *FileCertificateManager) ServerHealthy() bool {
	panic("implement me")
}

func (s *FileCertificateManager) Current() *tls.Certificate {
	s.certAccessLock.Lock()
	defer s.certAccessLock.Unlock()
	return s.cert
}

func (f *FileCertificateManager) rotateCerts() error {

	crt, err := f.loadCertificates(f.certDir)
	if err != nil {
		log.DefaultLogger().Reason(err).Infof("failed to load the certificate in %s", f.certDir)
		return err
	}

	f.certAccessLock.Lock()
	defer f.certAccessLock.Unlock()
	// update after the callback, to ensure that the reconfiguration succeeded
	f.cert = crt

	log.DefaultLogger().Infof("certificate from %s with common name '%s' retrieved.", f.certDir, crt.Leaf.Subject.CommonName)
	return nil
}

func (s *FileCertificateManager) loadCertificates(certDir string) (serverCrt *tls.Certificate, err error) {

	certBytesPath := filepath.Join(certDir, CertBytesValue)
	keyBytesPath := filepath.Join(certDir, KeyBytesValue)

	certBytes, err := ioutil.ReadFile(certBytesPath)
	if err != nil {
		return nil, err
	}
	keyBytes, err := ioutil.ReadFile(keyBytesPath)
	if err != nil {
		return nil, err
	}

	crt, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %v\n", err)
	}
	leaf, err := cert.ParseCertsPEM(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to load leaf certificate: %v\n", err)
	}
	crt.Leaf = leaf[0]
	return &crt, nil
}
