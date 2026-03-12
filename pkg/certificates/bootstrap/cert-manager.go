package bootstrap

import (
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
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
	certBytesPath      string
	keyBytesPath       string
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

func NewFileCertificateManager(certBytesPath string, keyBytesPath string) *FileCertificateManager {
	return &FileCertificateManager{
		certBytesPath:      certBytesPath,
		keyBytesPath:       keyBytesPath,
		stopCh:             make(chan struct{}, 1),
		errorRetryInterval: 1 * time.Minute,
	}
}

func (f *FileCertificateManager) Start() {
	objectUpdated := make(chan struct{}, 1)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.DefaultLogger().Reason(err).Critical("Failed to create an inotify watcher")
	}
	defer watcher.Close()

	certDir := filepath.Dir(f.certBytesPath)
	err = watcher.Add(certDir)
	if err != nil {
		log.DefaultLogger().Reason(err).Criticalf("Failed to establish a watch on %s", f.certBytesPath)
	}
	keyDir := filepath.Dir(f.keyBytesPath)
	if keyDir != certDir {
		err = watcher.Add(keyDir)
		if err != nil {
			log.DefaultLogger().Reason(err).Criticalf("Failed to establish a watch on %s", f.keyBytesPath)
		}
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
				log.DefaultLogger().Reason(err).Errorf("An error occurred when watching certificates files %s and %s", f.certBytesPath, f.keyBytesPath)
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
	crt, err := f.loadCertificates()
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to load the certificate %s and %s", f.certBytesPath, f.keyBytesPath)
		return err
	}

	f.certAccessLock.Lock()
	defer f.certAccessLock.Unlock()
	// update after the callback, to ensure that the reconfiguration succeeded
	f.cert = crt

	log.DefaultLogger().Infof("certificate with common name '%s' retrieved.", crt.Leaf.Subject.CommonName)
	return nil
}

func (f *FileCertificateManager) loadCertificates() (serverCrt *tls.Certificate, err error) {
	// #nosec No risk for path injection. Used for specific cert file for key rotation
	certBytes, err := os.ReadFile(f.certBytesPath)
	if err != nil {
		return nil, err
	}
	// #nosec No risk for path injection. Used for specific cert file for key rotation
	keyBytes, err := os.ReadFile(f.keyBytesPath)
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

type SecretCertificateManager struct {
	store     cache.Store
	secretKey string
	tlsCrt    string
	tlsKey    string
	crtLock   *sync.Mutex
	revision  string
	crt       *tls.Certificate
}

func (s *SecretCertificateManager) Start() {
}

func (s *SecretCertificateManager) Stop() {
}

func (s *SecretCertificateManager) Current() *tls.Certificate {
	s.crtLock.Lock()
	defer s.crtLock.Unlock()
	rawSecret, exists, err := s.store.GetByKey(s.secretKey)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Secret %s can't be retrieved from the cache", s.secretKey)
		return s.crt
	} else if !exists {
		return s.crt
	}
	secret := rawSecret.(*v1.Secret)
	if secret.ObjectMeta.ResourceVersion == s.revision {
		return s.crt
	}
	crt, err := tls.X509KeyPair(secret.Data[s.tlsCrt], secret.Data[s.tlsKey])
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to load certificate from secret %s", s.secretKey)
		return s.crt
	}
	leaf, err := cert.ParseCertsPEM(secret.Data[s.tlsCrt])
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("failed to load leaf certificate from secret %s", s.secretKey)
		return s.crt
	}
	crt.Leaf = leaf[0]
	s.revision = secret.ResourceVersion
	s.crt = &crt
	return s.crt
}

func (s *SecretCertificateManager) ServerHealthy() bool {
	panic("implement me")
}

// NewSecretCertificateManager takes a secret store and the name and the  namespace of a secret. If there is a newer
// version of the secret in the cache, the next Current() call will immediately wield it. It takes resource versions
// into account to be efficient.
func NewSecretCertificateManager(name string, namespace string, store cache.Store) *SecretCertificateManager {
	return &SecretCertificateManager{
		store:     store,
		secretKey: fmt.Sprintf("%s/%s", namespace, name),
		tlsCrt:    CertBytesValue,
		tlsKey:    KeyBytesValue,
		crtLock:   &sync.Mutex{},
	}
}
