package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func handleKubeClientConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		log.Printf("Using env kubeconfig %s", kubeconfig)
		kubeconfig = os.Getenv("KUBECONFIG")
	}

	var config *rest.Config
	var err error
	if kubeconfig != "" {
		log.Printf("Loading kube client config from path %q", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		log.Printf("Using in-cluster kube client config")
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("could not get the client: %v", err)
	}

	return config, nil
}

func generateSelfSignedCertificate(commonName string, dnsNames []string) (*bytes.Buffer, *bytes.Buffer, error) {
	var caPEM, serverCertPEM, serverPrivKeyPEM *bytes.Buffer

	// CA config
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(2020),
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

	// CA private key
	caPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// Self signed CA certificate
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// PEM encode CA cert
	caPEM = new(bytes.Buffer)
	_ = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// server cert config
	cert := &x509.Certificate{
		DNSNames:     dnsNames,
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"kubevirt.io"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// server private key
	serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, ca, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	// PEM encode the  server cert and key
	serverCertPEM = new(bytes.Buffer)
	_ = pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	serverPrivKeyPEM = new(bytes.Buffer)
	_ = pem.Encode(serverPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})

	return serverCertPEM, serverPrivKeyPEM, nil
}

func generate(namespace string) ([]byte, []byte, error) {
	commonName := fmt.Sprintf("operator-webhook-service.%s.svc", namespace)
	dnsnames := []string{"operator-webhook-service", fmt.Sprintf("operator-webhook-service.%s", namespace), commonName}
	certificate, key, err := generateSelfSignedCertificate(commonName, dnsnames)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate self-signed certificate: %v", err)
	}
	log.Printf("Self-Signed certificate created sucessfully for CN %s", commonName)

	return certificate.Bytes(), key.Bytes(), nil
}

func exportCertificateFile(data []byte, filePath string) error {
	certificateFileName := fmt.Sprintf("%s.cert", filePath)
	encodedData := []byte(base64.StdEncoding.EncodeToString(data))
	if err := ioutil.WriteFile(certificateFileName, encodedData, 0644); err != nil {
		return fmt.Errorf("failed to write content to file %s: %v", filePath, err)
	}
	log.Printf("certificate exported successfully to: %s", filePath)

	return nil
}

func createSecret(clusterApi kubernetes.Interface, namespace, secretName string, certificate, key []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"tls.crt": certificate,
			"tls.key": key,
		},
	}

	_, err := clusterApi.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		return fmt.Errorf("failed to create secret %v", err)
	}
	log.Printf("Secret '%s' at '%s' created sucessfully", secretName, namespace)

	return nil
}

func main() {
	namespace := flag.String("namespace", "", "The namespace of the webhook")
	kubeconfig := flag.String("kubeconfig", "", "The path of kubeconfig")
	hookName := flag.String("hook", "", "The name of the hook")
	secretName := flag.String("secret", "", "The name of the secret")
	flag.Parse()

	if *namespace == "" || *hookName == "" || *secretName == "" {
		flag.Usage()
		log.Fatal("Not enough arguments")
	}

	var err error
	config, err := handleKubeClientConfig(*kubeconfig)
	if err != nil {
		log.Fatalf("Failed to set kubernetes client config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to set up Kubernetes client: %v", err)
	}

	certificate, key, err := generate(*namespace)
	if err != nil {
		log.Fatalf("Failed to generate certificate: %v", err)
	}

	err = exportCertificateFile(certificate, *hookName)
	if err != nil {
		log.Fatalf("Failed to export certificate to file: %v", err)
	}

	err = createSecret(clientset, *namespace, *secretName, certificate, key)
	if err != nil {
		log.Fatalf("Failed to create Secret: %v", err)
	}
}
