/*
Copyright 2018 The CDI Authors.

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

package keystest

import (
	"crypto/rsa"
	"crypto/x509"

	"k8s.io/client-go/util/cert/triple"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/cert"
	"kubevirt.io/containerized-data-importer/pkg/common"
)

// NewTLSSecret returns a new TLS secret from objects
func NewTLSSecret(namespace, secretName string, keyPair *triple.KeyPair, caCert *x509.Certificate, owner *metav1.OwnerReference) *v1.Secret {
	var privateKeyBytes, certBytes, caCertBytes []byte
	privateKeyBytes = cert.EncodePrivateKeyPEM(keyPair.Key)
	certBytes = cert.EncodeCertPEM(keyPair.Cert)

	if caCert != nil {
		caCertBytes = cert.EncodeCertPEM(caCert)
	}

	return NewTLSSecretFromBytes(namespace, secretName, privateKeyBytes, certBytes, caCertBytes, owner)
}

// NewTLSSecretFromBytes returns a new TLS secret from bytes
func NewTLSSecretFromBytes(namespace, secretName string, privateKeyBytes, certBytes, caCertBytes []byte, owner *metav1.OwnerReference) *v1.Secret {
	data := map[string][]byte{
		"tls.key": privateKeyBytes,
		"tls.crt": certBytes,
	}

	if caCertBytes != nil {
		data["ca.crt"] = caCertBytes
	}

	return newSecret(namespace, secretName, data, owner)
}

// NewPrivateKeySecret returns a new private key secret
func NewPrivateKeySecret(namespace, secretName string, privateKey *rsa.PrivateKey) (*v1.Secret, error) {
	privateKeyBytes := cert.EncodePrivateKeyPEM(privateKey)
	publicKeyBytes, err := cert.EncodePublicKeyPEM(&privateKey.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "Error encoding public key")
	}

	data := map[string][]byte{
		"id_rsa":     privateKeyBytes,
		"id_rsa.pub": publicKeyBytes,
	}

	return newSecret(namespace, secretName, data, nil), nil
}

func newSecret(namespace, secretName string, data map[string][]byte, owner *metav1.OwnerReference) *v1.Secret {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels: map[string]string{
				common.CDIComponentLabel: "keystore",
			},
		},
		Type: "Opaque",
		Data: data,
	}

	if owner != nil {
		secret.OwnerReferences = []metav1.OwnerReference{*owner}
	}

	return secret
}
