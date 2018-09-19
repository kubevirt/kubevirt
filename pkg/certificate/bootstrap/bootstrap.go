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
	"time"

	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	certificatesclient "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/certificate"
	"k8s.io/client-go/util/certificate/csr"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

const (
	keyBytesValue = "key-bytes"
)

// LoadClientCertForService requests a client cert for a component if the the certDir does not contain a certificate.
// The certificate and key file are stored in certDir.
func LoadClientCertForNode(bootstrapClient certificatesclient.CertificatesV1beta1Interface, certStore certificate.Store, name types.NodeName) error {

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

	certData, err := RequestKubeVirtCertificate(bootstrapClient.CertificateSigningRequests(), keyData, "nodes:"+string(name))
	if err != nil {
		return err
	}
	if _, err := certStore.Update(certData, keyData); err != nil {
		return err
	}

	return nil
}

// LoadClientCertForService fetches a client key from a secret or generates one and updates the secret.
// Afterwards it creates as certificate signing request and fetches the result after it got signed.
func LoadClientCertForService(client kubecli.KubevirtClient, certStore certificate.Store, serviceName string, namespace string) error {

	keyData, err := LoadKeyFromSecret(client, namespace, "kubevirt-certs-"+serviceName)
	if err != nil {
		return err
	}

	certData, err := RequestKubeVirtCertificate(client.CertificatesV1beta1().CertificateSigningRequests(), keyData, "service:"+serviceName)
	if err != nil {
		return err
	}
	if _, err := certStore.Update(certData, keyData); err != nil {
		return err
	}

	return nil
}

// LoadCertificateFromSecret create a certificate key and uploads it in to a secret if not preset.
// If the secret is already present, it returns the private certificate key
func LoadKeyFromSecret(client kubecli.KubevirtClient, secretNamespace, secretName string) ([]byte, error) {
	keyData, err := certutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		return nil, err
	}
	secret := &v12.Secret{
		ObjectMeta: v13.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
			Labels: map[string]string{
				v1.AppLabel: "",
			},
		},
		Type: "Opaque",
		Data: map[string][]byte{
			keyBytesValue: keyData,
		},
	}

	_, err = client.CoreV1().Secrets(secretNamespace).Create(secret)
	if errors.IsAlreadyExists(err) {
		secret, err := client.CoreV1().Secrets(secretNamespace).Get(secretName, v13.GetOptions{})
		if err != nil {
			return nil, err
		}
		if len(secret.Data[keyBytesValue]) > 0 {
			return secret.Data[keyBytesValue], nil
		} else {
			return nil, fmt.Errorf("found a secret but no key")
		}
	} else if err != nil {
		return nil, err
	}
	return keyData, nil
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

func RequestKubeVirtCertificate(client certificatesclient.CertificateSigningRequestInterface, privateKeyData []byte, name string) (certData []byte, err error) {
	subject := &pkix.Name{
		Organization: []string{"system:kubevirt"},
		CommonName:   "system:kubevirt:" + name,
	}

	privateKey, err := certutil.ParsePrivateKeyPEM(privateKeyData)
	if err != nil {
		return nil, fmt.Errorf("invalid private key for certificate request: %v", err)
	}
	csrData, err := certutil.MakeCSR(privateKey, subject, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to generate certificate request: %v", err)
	}

	usages := []certificates.KeyUsage{
		certificates.UsageDigitalSignature,
		certificates.UsageKeyEncipherment,
		certificates.UsageClientAuth,
	}
	req, err := csr.RequestCertificate(client, csrData, name, usages, privateKey)
	if err != nil {
		return nil, err
	}
	return csr.WaitForCertificate(client, req, 3600*time.Second)
}
