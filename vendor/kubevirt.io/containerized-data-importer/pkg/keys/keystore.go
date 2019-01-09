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

package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"io/ioutil"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/cert/triple"

	"kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	// KeyStoreTLSKeyFile is the key in a secret containing the TLS key
	KeyStoreTLSKeyFile = "tls.key"

	// KeyStoreTLSCertFile is the key in a secret containing the TLS cert
	KeyStoreTLSCertFile = "tls.crt"

	// KeyStoreTLSCAFile is the key in a secret containing a CA cert
	KeyStoreTLSCAFile = "ca.crt"

	// KeyStorePrivateKeyFile is the key in a secret containing an RSA private key
	KeyStorePrivateKeyFile = "id_rsa"

	// KeyStorePublicKeyFile is the key in a secret containing an RSA publis key
	KeyStorePublicKeyFile = "id_rsa.pub"
)

// KeyPairAndCert holds a KeyPair and optional CA
// In the case of a server key pair, the CA is the CA that signed client certs
// In the case of a client key pair, the CA is the CA that signed the server cert
type KeyPairAndCert struct {
	KeyPair triple.KeyPair
	CACert  *x509.Certificate
}

// KeyPairAndCertBytes contains the PEM encoded key data
type KeyPairAndCertBytes struct {
	PrivateKey []byte
	Cert       []byte
	CACert     []byte
}

// GetOrCreateCA will get the CA KeyPair, creating it if necessary
func GetOrCreateCA(client kubernetes.Interface, namespace, secretName, caName string) (*triple.KeyPair, error) {
	keyPairAndCert, err := GetKeyPairAndCert(client, namespace, secretName)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting CA")
	}

	if keyPairAndCert != nil {
		glog.Infof("Retrieved CA key/cert %s from kubernetes", caName)
		return &keyPairAndCert.KeyPair, nil
	}

	glog.Infof("Recreating CA %s", caName)

	keyPair, err := triple.NewCA(caName)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating CA")
	}

	exists, err := SaveKeyPairAndCert(client, namespace, secretName, &KeyPairAndCert{*keyPair, nil}, nil)
	if !exists && err != nil {
		return nil, errors.Wrap(err, "Error saving CA")
	}

	// do another get
	// this should be very unlikely to hit code path
	if exists {
		keyPairAndCert, err = GetKeyPairAndCert(client, namespace, secretName)
		if keyPairAndCert == nil || err != nil {
			return nil, errors.Wrap(err, "Error getting CA second time around")
		}
		keyPair = &keyPairAndCert.KeyPair
	}

	return keyPair, nil
}

// GetOrCreateServerKeyPairAndCert creates secret for an upload server
func GetOrCreateServerKeyPairAndCert(client kubernetes.Interface,
	namespace,
	secretName string,
	caKeyPair *triple.KeyPair,
	clientCACert *x509.Certificate,
	commonName string,
	serviceName string,
	owner *metav1.OwnerReference) (*KeyPairAndCert, error) {
	keyPairAndCert, err := GetKeyPairAndCert(client, namespace, secretName)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting server cert")
	}

	if keyPairAndCert != nil {
		glog.Infof("Retrieved server key/cert %s from kubernetes", commonName)
		return keyPairAndCert, nil
	}

	keyPair, err := triple.NewServerKeyPair(caKeyPair, commonName, serviceName, namespace, "cluster.local", []string{}, []string{})
	if err != nil {
		return nil, errors.Wrap(err, "Error creating server key pair")
	}

	keyPairAndCert = &KeyPairAndCert{*keyPair, clientCACert}

	exists, err := SaveKeyPairAndCert(client, namespace, secretName, keyPairAndCert, owner)
	if !exists && err != nil {
		return nil, errors.Wrap(err, "Error saving server key pair")
	}

	if exists {
		// race condition
		return GetKeyPairAndCert(client, namespace, secretName)
	}

	return keyPairAndCert, nil
}

// GetOrCreateClientKeyPairAndCert creates a secret for upload proxy
func GetOrCreateClientKeyPairAndCert(client kubernetes.Interface,
	namespace, secretName string,
	caKeyPair *triple.KeyPair,
	caCert *x509.Certificate,
	commonName string,
	organizations []string,
	owner *metav1.OwnerReference) (*KeyPairAndCert, error) {
	keyPairAndCert, err := GetKeyPairAndCert(client, namespace, secretName)
	if err != nil {
		return nil, errors.Wrap(err, "Error getting client cert")
	}

	if keyPairAndCert != nil {
		glog.Infof("Retrieved client key/cert %s from kubernetes", commonName)
		return keyPairAndCert, nil
	}

	keyPair, err := triple.NewClientKeyPair(caKeyPair, commonName, organizations)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating client key pair")
	}

	keyPairAndCert = &KeyPairAndCert{*keyPair, caCert}

	exists, err := SaveKeyPairAndCert(client, namespace, secretName, keyPairAndCert, owner)
	if !exists && err != nil {
		return nil, errors.Wrap(err, "Error saving server key pair")
	}

	if exists {
		// race condition
		return GetKeyPairAndCert(client, namespace, secretName)
	}

	return keyPairAndCert, nil
}

// GetKeyPairAndCert will return the secret data if it exists
func GetKeyPairAndCert(client kubernetes.Interface, namespace, secretName string) (*KeyPairAndCert, error) {
	var keyPairAndCert KeyPairAndCert

	keyPairAndCertBytes, err := GetKeyPairAndCertBytes(client, namespace, secretName)
	if err != nil {
		return nil, errors.Wrap(err, "Error retrieving key bytes")
	}

	if keyPairAndCertBytes == nil {
		return nil, nil
	}

	key, err := parsePrivateKey(keyPairAndCertBytes.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing private key")
	}

	keyPairAndCert.KeyPair.Key = key

	certs, err := cert.ParseCertsPEM(keyPairAndCertBytes.Cert)
	if err != nil || len(certs) != 1 {
		return nil, errors.Errorf("Cert parse error %s, %d", err, len(certs))
	}

	keyPairAndCert.KeyPair.Cert = certs[0]

	if keyPairAndCertBytes.CACert != nil {
		certs, err := cert.ParseCertsPEM(keyPairAndCertBytes.CACert)
		if err != nil || len(certs) != 1 {
			return nil, errors.Errorf("CA cert parse error %s, %d", err, len(certs))
		}
		keyPairAndCert.CACert = certs[0]
	}

	return &keyPairAndCert, nil
}

// GetKeyPairAndCertBytes returns the raw bytes stored in the secret
func GetKeyPairAndCertBytes(client kubernetes.Interface, namespace, secretName string) (*KeyPairAndCertBytes, error) {
	var keyPairAndCertBytes KeyPairAndCertBytes

	secret, err := client.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "Error getting secret")
	}

	bytes, ok := secret.Data[KeyStoreTLSKeyFile]
	if !ok {
		return nil, errors.Errorf("Private key missing from secret")
	}

	keyPairAndCertBytes.PrivateKey = bytes

	if bytes, ok = secret.Data[KeyStoreTLSCertFile]; !ok {
		return nil, errors.Errorf("Cert missing from secret")
	}

	keyPairAndCertBytes.Cert = bytes

	// okay if this doesn't exist
	if bytes, ok = secret.Data[KeyStoreTLSCAFile]; ok {
		keyPairAndCertBytes.CACert = bytes
	}

	return &keyPairAndCertBytes, nil
}

// SaveKeyPairAndCert saves a private key, cert, and maybe a ca cert to kubernetes
func SaveKeyPairAndCert(client kubernetes.Interface, namespace, secretName string, keyPairAndCA *KeyPairAndCert, owner *metav1.OwnerReference) (bool, error) {
	secret := newTLSSecret(namespace, secretName, keyPairAndCA, owner)

	_, err := client.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		return k8serrors.IsAlreadyExists(err), errors.Wrap(err, "Error creating cert")
	}

	return false, nil
}

// newTLSSecret returns a new TLS secret from objects
func newTLSSecret(namespace, secretName string, keyPairAndCA *KeyPairAndCert, owner *metav1.OwnerReference) *v1.Secret {
	var privateKeyBytes, certBytes, caCertBytes []byte
	privateKeyBytes = cert.EncodePrivateKeyPEM(keyPairAndCA.KeyPair.Key)
	certBytes = cert.EncodeCertPEM(keyPairAndCA.KeyPair.Cert)

	if keyPairAndCA.CACert != nil {
		caCertBytes = cert.EncodeCertPEM(keyPairAndCA.CACert)
	}

	return newTLSSecretFromBytes(namespace, secretName, privateKeyBytes, certBytes, caCertBytes, owner)
}

// newTLSSecretFromBytes returns a new TLS secret from bytes
func newTLSSecretFromBytes(namespace, secretName string, privateKeyBytes, certBytes, caCertBytes []byte, owner *metav1.OwnerReference) *v1.Secret {
	data := map[string][]byte{
		KeyStoreTLSKeyFile:  privateKeyBytes,
		KeyStoreTLSCertFile: certBytes,
	}

	if caCertBytes != nil {
		data[KeyStoreTLSCAFile] = caCertBytes
	}

	return newSecret(namespace, secretName, data, owner)
}

// GetOrCreatePrivateKey gets or creates a private key secret
func GetOrCreatePrivateKey(client kubernetes.Interface, namespace, secretName string) (*rsa.PrivateKey, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "Error getting secret")
		}

		// let's create the secret
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, errors.Wrap(err, "Error generating key")
		}

		secret, err = newPrivateKeySecret(namespace, secretName, privateKey)
		if err != nil {
			return nil, errors.Wrap(err, "Error creating prvate key secret")
		}

		secret, err = client.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return nil, errors.Wrap(err, "Error creating secret")
			}

			secret, err = client.CoreV1().Secrets(namespace).Get(secretName, metav1.GetOptions{})
			if err != nil {
				return nil, errors.Wrap(err, "Error getting secret, second time")
			}
		}
	}

	bytes, ok := secret.Data[KeyStorePrivateKeyFile]
	if !ok {
		return nil, errors.Wrap(err, "Secret missing private key")
	}

	return parsePrivateKey(bytes)
}

// newPrivateKeySecret returns a new private key secret
func newPrivateKeySecret(namespace, secretName string, privateKey *rsa.PrivateKey) (*v1.Secret, error) {
	privateKeyBytes := cert.EncodePrivateKeyPEM(privateKey)
	publicKeyBytes, err := cert.EncodePublicKeyPEM(&privateKey.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "Error encoding public key")
	}

	data := map[string][]byte{
		KeyStorePrivateKeyFile: privateKeyBytes,
		KeyStorePublicKeyFile:  publicKeyBytes,
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

func parsePrivateKey(bytes []byte) (*rsa.PrivateKey, error) {
	obj, err := cert.ParsePrivateKeyPEM(bytes)
	if err != nil {
		return nil, errors.Wrap(err, "Error parsing secret")
	}

	key, ok := obj.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("Invalid pem format")
	}

	return key, nil
}

// GenerateSelfSignedCert generates a self signed certificate keyFile, certFile pair to be passed to http.ListenAndServeTLS
// The first return value is the keyFile name, the second the certFile name
// The caller is responsible for creating a writeable directory and cleaning up the generated files afterwards.
func GenerateSelfSignedCert(certsDirectory string, name string, namespace string) (string, string, error) {
	// Generic self signed CA.
	caKeyPair, _ := triple.NewCA("cdi.kubevirt.io")
	keyPair, _ := triple.NewServerKeyPair(
		caKeyPair,
		name+"."+namespace+".pod.cluster.local",
		name,
		namespace,
		"cluster.local",
		[]string{},
		[]string{},
	)

	keyFile := filepath.Join(certsDirectory, "key.pem")
	certFile := filepath.Join(certsDirectory, "cert.pem")

	err := ioutil.WriteFile(keyFile, cert.EncodePrivateKeyPEM(keyPair.Key), 0600)
	if err != nil {
		return "", "", err
	}
	err = ioutil.WriteFile(certFile, cert.EncodeCertPEM(keyPair.Cert), 0600)
	if err != nil {
		return "", "", err
	}

	return keyFile, certFile, nil
}
