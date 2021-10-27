package certlib

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"math/rand"
	"time"
)

type SelfSignedCertificate struct {
	DNSNames    []string
	CommonName  string
	Certificate *bytes.Buffer
	PrivateKey  *bytes.Buffer
}

func (s *SelfSignedCertificate) Generate() error {
	var caPEM *bytes.Buffer

	randomSource := rand.New(rand.NewSource(time.Now().Unix()))
	caCertificateConfig := &x509.Certificate{
		SerialNumber: big.NewInt(randomSource.Int63()),
		Subject: pkix.Name{
			Organization: []string{"kubvirt.io"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caPrivateKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %v", err)
	}

	caSelfSignedCertificateBytes, err := x509.CreateCertificate(
		cryptorand.Reader,
		caCertificateConfig,
		caCertificateConfig,
		&caPrivateKey.PublicKey,
		caPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to generate CA certificate: %v", err)
	}

	// PEM encode CA cert
	caPEM = new(bytes.Buffer)
	err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caSelfSignedCertificateBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to encode CA certificate bytes to PEM: %v", err)
	}

	serverCertificateConfig := &x509.Certificate{
		DNSNames:     s.DNSNames,
		SerialNumber: big.NewInt(randomSource.Int63()),
		Subject: pkix.Name{
			CommonName:   s.CommonName,
			Organization: []string{"kubevirt.io"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	serverPrivateKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate server private key: %v", err)
	}

	// Signing server certificate
	serverCertificateBytes, err := x509.CreateCertificate(
		cryptorand.Reader,
		serverCertificateConfig,
		caCertificateConfig,
		&serverPrivateKey.PublicKey,
		caPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to sign server certificate: %v", err)
	}

	// PEM encode the  server cert and key
	s.Certificate = new(bytes.Buffer)
	err = pem.Encode(s.Certificate, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertificateBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to encode server certificate bytes to PEM: %v", err)
	}

	s.PrivateKey = new(bytes.Buffer)
	err = pem.Encode(s.PrivateKey, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivateKey),
	})
	if err != nil {
		return fmt.Errorf("failed to encode server private key bytes to PEM: %v", err)
	}

	return nil
}
