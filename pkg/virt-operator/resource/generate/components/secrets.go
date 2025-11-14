package components

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"reflect"
	"sort"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

// #nosec 101, false positives were caused by variables not holding any secret value.
const (
	KubeVirtCASecretName                              = "kubevirt-ca"
	ExternalKubeVirtCAConfigMapName                   = "kubevirt-external-ca"
	KubeVirtExportCASecretName                        = "kubevirt-export-ca"
	VirtHandlerCertSecretName                         = "kubevirt-virt-handler-certs"
	VirtHandlerServerCertSecretName                   = "kubevirt-virt-handler-server-certs"
	VirtHandlerMigrationClientCertSecretName          = "kubevirt-virt-handler-migration-client-certs"
	VirtHandlerVsockClientCertSecretName              = "kubevirt-virt-handler-vsock-client-certs"
	VirtOperatorCertSecretName                        = "kubevirt-operator-certs"
	VirtApiCertSecretName                             = "kubevirt-virt-api-certs"
	VirtControllerCertSecretName                      = "kubevirt-controller-certs"
	VirtExportProxyCertSecretName                     = "kubevirt-exportproxy-certs"
	VirtSynchronizationControllerCertSecretName       = "kubevirt-synchronization-controller-certs"
	VirtSynchronizationControllerServerCertSecretName = "kubevirt-synchronization-controller-server-certs"
	CABundleKey                                       = "ca-bundle"
	LocalPodDNStemplateString                         = "%s.%s.pod.cluster.local"
	CaClusterLocal                                    = "cluster.local"
	maxCertificatesInBundle                           = 50
)

type CertificateCreationCallback func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey)

var populationStrategy = map[string]CertificateCreationCallback{
	KubeVirtCASecretName: func(secret *k8sv1.Secret, _ *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair, _ := triple.NewCA("kubevirt.io", duration)
		return caKeyPair.Cert, caKeyPair.Key
	},
	KubeVirtExportCASecretName: func(secret *k8sv1.Secret, _ *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair, _ := triple.NewCA("export.kubevirt.io", duration)
		return caKeyPair.Cert, caKeyPair.Key
	},
	VirtOperatorCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			fmt.Sprintf(LocalPodDNStemplateString, VirtOperatorServiceName, secret.Namespace),
			VirtOperatorServiceName,
			secret.Namespace,
			CaClusterLocal,
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtApiCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			fmt.Sprintf(LocalPodDNStemplateString, VirtApiServiceName, secret.Namespace),
			VirtApiServiceName,
			secret.Namespace,
			CaClusterLocal,
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtControllerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			fmt.Sprintf(LocalPodDNStemplateString, VirtControllerServiceName, secret.Namespace),
			VirtControllerServiceName,
			secret.Namespace,
			CaClusterLocal,
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtHandlerServerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			"kubevirt.io:system:node:virt-handler",
			VirtHandlerServiceName,
			secret.Namespace,
			CaClusterLocal,
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtHandlerMigrationClientCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewClientKeyPair(
			caKeyPair,
			"kubevirt.io:system:client:migration",
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtHandlerVsockClientCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewClientKeyPair(
			caKeyPair,
			"kubevirt.io:system:client:vsock",
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtHandlerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
			"kubevirt.io:system:client:virt-handler",
			nil,
			duration,
		)
		return clientKeyPair.Cert, clientKeyPair.Key
	},
	VirtExportProxyCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			fmt.Sprintf(LocalPodDNStemplateString, VirtExportProxyServiceName, secret.Namespace),
			VirtExportProxyServiceName,
			secret.Namespace,
			CaClusterLocal,
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtSynchronizationControllerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
			"kubevirt.io:system:client:virt-synchronization-controller",
			nil,
			duration,
		)
		return clientKeyPair.Cert, clientKeyPair.Key
	},
	VirtSynchronizationControllerServerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *ecdsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*ecdsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			"kubevirt.io:system:node:virt-synchronization-controller",
			VirtSynchronizationControllerServiceName,
			secret.Namespace,
			CaClusterLocal,
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
}

func PopulateSecretWithCertificate(secret *k8sv1.Secret, caCert *tls.Certificate, duration *metav1.Duration) (err error) {
	strategy, ok := populationStrategy[secret.Name]
	if !ok {
		return fmt.Errorf("no certificate population strategy found for secret")
	}
	crt, certKey := strategy(secret, caCert, duration.Duration)

	secret.Data = map[string][]byte{
		bootstrap.CertBytesValue: cert.EncodeCertPEM(crt),
		bootstrap.KeyBytesValue:  cert.EncodePrivateKeyPEM(certKey),
	}

	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	secret.Annotations["kubevirt.io/duration"] = duration.String()
	return nil
}

func NewCACertSecrets(operatorNamespace string) []*k8sv1.Secret {
	return []*k8sv1.Secret{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      KubeVirtCASecretName,
				Namespace: operatorNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      KubeVirtExportCASecretName,
				Namespace: operatorNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
	}
}

func NewCAConfigMaps(operatorNamespace string) []*k8sv1.ConfigMap {
	return []*k8sv1.ConfigMap{
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      KubeVirtCASecretName,
				Namespace: operatorNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      KubeVirtExportCASecretName,
				Namespace: operatorNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ExternalKubeVirtCAConfigMapName,
				Namespace: operatorNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
		},
	}
}

func NewCertSecrets(installNamespace string, operatorNamespace string) []*k8sv1.Secret {
	secrets := []*k8sv1.Secret{

		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtApiCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtControllerCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtExportProxyCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtHandlerServerCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtHandlerMigrationClientCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtHandlerVsockClientCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtHandlerCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtSynchronizationControllerCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtSynchronizationControllerServerCertSecretName,
				Namespace: installNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
		{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      VirtOperatorCertSecretName,
				Namespace: operatorNamespace,
				Labels: map[string]string{
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
			},
			Type: k8sv1.SecretTypeTLS,
		},
	}
	return secrets
}

// nextRotationDeadline returns a value for the threshold at which the
// current certificate should be rotated, 80% of the expiration of the
// certificate.
func NextRotationDeadline(cert *tls.Certificate, ca *tls.Certificate, renewBefore *metav1.Duration, caRenewBefore *metav1.Duration) time.Time {

	if cert == nil {
		return time.Now()
	}

	if ca != nil {
		certPool := x509.NewCertPool()
		certPool.AddCert(ca.Leaf)

		_, err := cert.Leaf.Verify(x509.VerifyOptions{
			Roots:     certPool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		})

		if err != nil {
			log.DefaultLogger().Reason(err).Infof("The certificate with common name '%s' is not signed with the supplied CA. Triggering a rotation.", cert.Leaf.Subject.CommonName)
			return time.Now()
		}
	}

	certNotAfter := cert.Leaf.NotAfter
	deadline := cert.Leaf.NotAfter.Add(-renewBefore.Duration)

	if ca != nil {
		caNotAfter := ca.Leaf.NotAfter
		if caNotAfter.Before(certNotAfter) {
			log.DefaultLogger().Infof("The certificate with common name '%s' expires after the supplied CA does. Scheduling rotation based on CA's lifetime.", cert.Leaf.Subject.CommonName)
			deadline = caNotAfter
			if caRenewBefore != nil {
				// Set cert rotation for the middle of the period of time when CA's overlap
				deadline = caNotAfter.Add(-time.Duration(float64(caRenewBefore.Duration) * 0.5))
			}
		}
	}

	log.DefaultLogger().V(4).Infof("Certificate with common name '%s' expiration is %v, rotation deadline is %v", cert.Leaf.Subject.CommonName, certNotAfter, deadline)
	return deadline
}

func ValidateSecret(secret *k8sv1.Secret) error {
	if _, ok := secret.Data[bootstrap.CertBytesValue]; !ok {
		return fmt.Errorf("%s value not found in %s secret\n", bootstrap.CertBytesValue, secret.Name)
	}
	if _, ok := secret.Data[bootstrap.KeyBytesValue]; !ok {
		return fmt.Errorf("%s value not found in %s secret\n", bootstrap.KeyBytesValue, secret.Name)
	}
	return nil
}

func LoadCertificates(secret *k8sv1.Secret) (serverCrt *tls.Certificate, err error) {

	if err := ValidateSecret(secret); err != nil {
		return nil, err
	}

	crt, err := tls.X509KeyPair(secret.Data[bootstrap.CertBytesValue], secret.Data[bootstrap.KeyBytesValue])
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %v\n", err)
	}
	leaf, err := cert.ParseCertsPEM(secret.Data[bootstrap.CertBytesValue])
	if err != nil {
		return nil, fmt.Errorf("failed to load leaf certificate: %v\n", err)
	}
	crt.Leaf = leaf[0]
	return &crt, nil
}

// filterValidCertificates filters out certificates that are not valid and sorts them by age.
// If there are more than maxCount, it truncates the list to maxCount
func filterValidCertificates(certs []*x509.Certificate, now time.Time, maxCount int) []*x509.Certificate {
	validCerts := make([]*x509.Certificate, 0, len(certs))
	for _, crt := range certs {
		if !crt.NotAfter.Before(now) {
			validCerts = append(validCerts, crt)
		}
	}

	sort.SliceStable(validCerts, func(i, j int) bool {
		return validCerts[i].NotBefore.Unix() > validCerts[j].NotBefore.Unix()
	})

	if len(validCerts) > maxCount {
		log.Log.Warningf("more than %d CA certificates found in the CA bundle, truncating to %d", maxCount, maxCount)
		return validCerts[:maxCount]
	}

	return validCerts
}

func MergeCABundle(currentCert *tls.Certificate, currentBundle []byte, overlapDuration time.Duration) ([]byte, int, error) {
	current := cert.EncodeCertPEM(currentCert.Leaf)
	certs, err := cert.ParseCertsPEM(currentBundle)
	if err != nil {
		return nil, 0, err
	}

	now := time.Now()
	validCerts := filterValidCertificates(certs, now, maxCertificatesInBundle)

	var newBundle []byte
	certCount := 0
	// we check for every cert i > 0, if in context to the certificate i-1 it existed already longer than the overlap
	// duration. We check the certificate i = 0 against the current certificate.
	for i, crt := range validCerts {
		if i == 0 {
			if currentCert.Leaf.NotBefore.Add(overlapDuration).Before(now) {
				log.DefaultLogger().Infof("Kept old CA certificates for a duration of at least %v, dropping them now.", overlapDuration)
				break
			}
		} else {
			if validCerts[i-1].NotBefore.Add(overlapDuration).Before(now) {
				log.DefaultLogger().Infof("Kept old CA certificates for a duration of at least %v, dropping them now.", overlapDuration)
				break
			}
		}

		certBytes := cert.EncodeCertPEM(crt)

		// don't add the current CA multiple times
		if reflect.DeepEqual(certBytes, current) {
			continue
		}
		newBundle = append(newBundle, certBytes...)
		certCount++
	}

	newBundle = append(current, newBundle...)
	certCount++
	return newBundle, certCount, nil
}
