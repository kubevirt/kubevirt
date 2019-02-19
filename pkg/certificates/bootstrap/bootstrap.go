/*
Copyright 2016 The Kubernetes Authors.

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

package bootstrap

import (
	"crypto/x509/pkix"
	"fmt"
	"net"
	"time"

	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	certificatesclient "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/certificate/csr"
)

const (
	keyBytesValue = "key-bytes"
)

// loadCert requests a client cert for a component if the the certDir does not contain a certificate.
// The certificate and key file are stored in certDir.
func loadCert(bootstrapClient certificatesclient.CertificatesV1beta1Interface, certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) error {

	var keyData []byte
	if cert, err := certStore.Current(); err == nil {
		if cert.PrivateKey != nil {
			keyData, err = certutil.MarshalPrivateKeyToPEM(cert.PrivateKey)
			if err != nil {
				keyData = nil
			}
		}
	}
	if !verifyKeyData(keyData) {
		var err error
		glog.V(2).Infof("No valid private key found for bootstrapping, creating a new one")
		keyData, err = certutil.MakeEllipticPrivateKeyPEM()
		if err != nil {
			return err
		}
	}

	certData, err := RequestKubeVirtCertificate(bootstrapClient.CertificateSigningRequests(), keyData, "kubevirt.io:system:"+name, dnsSANs, ipSANs)
	if err != nil {
		return err
	}
	if _, err := certStore.Update(certData, keyData); err != nil {
		return err
	}

	return nil
}

func LoadCertForNode(bootstrapClient certificatesclient.CertificatesV1beta1Interface, certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) error {
	return loadCert(bootstrapClient, certStore, "nodes:"+name, dnsSANs, ipSANs)
}

func LoadCertForService(bootstrapClient certificatesclient.CertificatesV1beta1Interface, certStore certificate.Store, name string, dnsSANs []string, ipSANs []net.IP) error {
	return loadCert(bootstrapClient, certStore, "services:"+name, dnsSANs, ipSANs)
}

// verifyKeyData returns true if the provided data appears to be a valid private key.
func verifyKeyData(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if _, err := certutil.ParsePrivateKeyPEM(data); err != nil {
		return false
	}
	return true
}

func RequestKubeVirtCertificate(client certificatesclient.CertificateSigningRequestInterface, privateKeyData []byte, name string, dnsSANs []string, ipSANs []net.IP) (certData []byte, err error) {
	subject := &pkix.Name{
		Organization: []string{"kubevirt.io:system"},
		CommonName:   name,
	}

	privateKey, err := certutil.ParsePrivateKeyPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("invalid private key for certificate request: %v", err)
	}
	csrData, err := certutil.MakeCSR(privateKey, subject, dnsSANs, ipSANs)
	if err != nil {
		return nil, fmt.Errorf("unable to generate certificate request: %v", err)
	}

	usages := []certificates.KeyUsage{
		certificates.UsageDigitalSignature,
		certificates.UsageKeyEncipherment,
		certificates.UsageClientAuth,
		certificates.UsageServerAuth,
	}
	req, err := csr.RequestCertificate(client, csrData, name, usages, privateKey)
	if err != nil {
		return nil, err
	}
	return csr.WaitForCertificate(client, req, 3600*time.Second)
}
