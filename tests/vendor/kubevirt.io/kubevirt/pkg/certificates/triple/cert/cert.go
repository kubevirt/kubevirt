/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cert

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"math"
	"math/big"
	mathrand "math/rand"
	"net"
	"time"
)

const (
	rsaKeySize = 2048
)

// Config contains the basic fields required for creating a certificate
type Config struct {
	CommonName          string
	Organization        []string
	AltNames            AltNames
	Usages              []x509.ExtKeyUsage
	NotBefore, NotAfter *time.Time
}

// AltNames contains the domain names and IP addresses that will be added
// to the API Server's x509 certificate SubAltNames field. The values will
// be passed directly to the x509.Certificate object.
type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

// NewRSAPrivateKey creates an RSA private key
func NewRSAPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(cryptorand.Reader, rsaKeySize)
}

// NewECDSAPrivateKey creates an ECDSA private key
func NewECDSAPrivateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), cryptorand.Reader)
}

// NewSelfSignedCACert creates a CA certificate
func NewSelfSignedCACert(cfg Config, key crypto.Signer, duration time.Duration) (*x509.Certificate, error) {
	now := time.Now()
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(randomSerialNumber()),
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		NotBefore:             now.UTC(),
		NotAfter:              now.Add(duration).UTC(),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              cfg.AltNames.DNSNames,
	}
	if cfg.NotBefore != nil {
		tmpl.NotBefore = *cfg.NotBefore
	}
	if cfg.NotAfter != nil {
		tmpl.NotAfter = *cfg.NotAfter
	}

	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, duration time.Duration) (*x509.Certificate, error) {
	serial, err := cryptorand.Int(cryptorand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}
	if len(cfg.Usages) == 0 {
		return nil, errors.New("must specify at least one ExtKeyUsage")
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:     cfg.AltNames.DNSNames,
		IPAddresses:  cfg.AltNames.IPs,
		SerialNumber: serial,
		NotBefore:    caCert.NotBefore,
		NotAfter:     time.Now().Add(duration).UTC(),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  cfg.Usages,
	}
	if cfg.NotBefore != nil {
		certTmpl.NotBefore = *cfg.NotBefore
	}
	if cfg.NotAfter != nil {
		certTmpl.NotAfter = *cfg.NotAfter
	}
	certDERBytes, err := x509.CreateCertificate(cryptorand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// randomSerialNumber returns a random int64 serial number based on
// time.Now. It is defined separately from the generator interface so
// that the caller doesn't have to worry about an input template or
// error - these are unnecessary when creating a random serial.
func randomSerialNumber() int64 {
	r := mathrand.New(mathrand.NewSource(time.Now().UTC().UnixNano()))
	return r.Int63()
}
