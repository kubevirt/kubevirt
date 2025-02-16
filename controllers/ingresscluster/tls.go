package ingresscluster

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"
)

func verifyCertificate(customCert []byte) error {
	block, _ := pem.Decode(customCert)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	now := time.Now()
	if now.After(certificate.NotAfter) {
		return fmt.Errorf("custom TLS certificate is expired")
	}
	if now.Before(certificate.NotBefore) {
		return fmt.Errorf("custom TLS certificate is not valid yet")
	}
	return nil
}

func verifyPrivateKey(customKey []byte) error {
	block, _ := pem.Decode(customKey)

	if block == nil {
		return fmt.Errorf("failed to decode key PEM")
	}
	if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		if _, err = x509.ParsePKCS1PrivateKey(block.Bytes); err != nil {
			if _, err = x509.ParseECPrivateKey(block.Bytes); err != nil {
				return fmt.Errorf("block %s is not valid key PEM", block.Type)
			}
		}
	}
	return nil
}
