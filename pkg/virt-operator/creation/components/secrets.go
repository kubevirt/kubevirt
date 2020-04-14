package components

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"reflect"
	"sort"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
)

const (
	KubeVirtCASecretName            = "kubevirt-ca"
	VirtHandlerCertSecretName       = "kubevirt-virt-handler-certs"
	VirtHandlerServerCertSecretName = "kubevirt-virt-handler-server-certs"
	VirtOperatorCertSecretName      = "kubevirt-operator-certs"
	VirtApiCertSecretName           = "kubevirt-virt-api-certs"
	VirtControllerCertSecretName    = "kubevirt-controller-certs"
	CABundleKey                     = "ca-bundle"
)

type CertificateCreationCallback func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey)

var populationStrategy = map[string]CertificateCreationCallback{
	KubeVirtCASecretName: func(secret *k8sv1.Secret, _ *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey) {
		caKeyPair, _ := triple.NewCA("kubevirt.io", duration)
		return caKeyPair.Cert, caKeyPair.Key
	},
	VirtOperatorCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*rsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			VirtOperatorServiceName+"."+secret.Namespace+".pod.cluster.local",
			VirtOperatorServiceName,
			secret.Namespace,
			"cluster.local",
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtApiCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*rsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			VirtApiServiceName+"."+secret.Namespace+".pod.cluster.local",
			VirtApiServiceName,
			secret.Namespace,
			"cluster.local",
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtControllerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*rsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			VirtControllerServiceName+"."+secret.Namespace+".pod.cluster.local",
			VirtControllerServiceName,
			secret.Namespace,
			"cluster.local",
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtHandlerServerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*rsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		keyPair, _ := triple.NewServerKeyPair(
			caKeyPair,
			"kubevirt.io:system:node:virt-handler",
			VirtHandlerServiceName,
			secret.Namespace,
			"cluster.local",
			nil,
			nil,
			duration,
		)
		return keyPair.Cert, keyPair.Key
	},
	VirtHandlerCertSecretName: func(secret *k8sv1.Secret, caCert *tls.Certificate, duration time.Duration) (cert *x509.Certificate, key *rsa.PrivateKey) {
		caKeyPair := &triple.KeyPair{
			Key:  caCert.PrivateKey.(*rsa.PrivateKey),
			Cert: caCert.Leaf,
		}
		clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
			"kubevirt.io:system:client:virt-handler",
			nil,
			duration,
		)
		return clientKeyPair.Cert, clientKeyPair.Key
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

func NewCACertSecret(operatorNamespace string) *k8sv1.Secret {
	return &k8sv1.Secret{
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
		Type: "Opaque",
	}
}

func NewKubeVirtCAConfigMap(operatorNamespace string) *k8sv1.ConfigMap {
	return &k8sv1.ConfigMap{
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
			Type: "Opaque",
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
			Type: "Opaque",
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
			Type: "Opaque",
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
			Type: "Opaque",
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
			Type: "Opaque",
		},
	}
	return secrets
}

// nextRotationDeadline returns a value for the threshold at which the
// current certificate should be rotated, 80% of the expiration of the
// certificate.
func NextRotationDeadline(cert *tls.Certificate, ca *tls.Certificate) time.Time {

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

	notAfter := cert.Leaf.NotAfter
	totalDuration := float64(notAfter.Sub(cert.Leaf.NotBefore))
	deadline := cert.Leaf.NotBefore.Add(time.Duration(totalDuration - totalDuration*0.2))

	log.DefaultLogger().V(4).Infof("Certificate with common name '%s' expiration is %v, rotation deadline is %v", cert.Leaf.Subject.CommonName, notAfter, deadline)
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

func MergeCABundle(currentCert *tls.Certificate, currentBundle []byte, overlapDuration time.Duration) ([]byte, int, error) {
	current := cert.EncodeCertPEM(currentCert.Leaf)
	certs, err := cert.ParseCertsPEM(currentBundle)
	if err != nil {
		return nil, 0, err
	}

	// ensure that no one does something nasty and adds thousands of certs
	if len(certs) > 10 {
		certs = certs[:10]
	}

	now := time.Now()
	var newBundle []byte

	// To ensure that no one messes this up, sort
	// We want the newest certs first
	sort.SliceStable(certs, func(i, j int) bool {
		return certs[i].NotBefore.Unix() > certs[j].NotBefore.Unix()
	})

	certCount := 0
	// we check for every cert i > 0, if in context to the certificate i-1 it existed already longer than the overlap
	// duration. We check the certificate i = 0 against the current certificate.
	for i, crt := range certs {
		if i == 0 {
			if currentCert.Leaf.NotBefore.Add(overlapDuration).Before(now) {
				log.DefaultLogger().Infof("Kept old CA certificates for a duration of at least %v, dropping them now.", overlapDuration)
				break
			}
		} else {
			if certs[i-1].NotBefore.Add(overlapDuration).Before(now) {
				log.DefaultLogger().Infof("Kept old CA certificates for a duration of at least %v, dropping them now.", overlapDuration)
				break
			}
		}

		// drop expired CAs
		if crt.NotAfter.Before(now) {
			continue
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
