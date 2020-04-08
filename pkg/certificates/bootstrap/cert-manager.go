package bootstrap

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

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
