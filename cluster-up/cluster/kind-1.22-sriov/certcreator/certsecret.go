package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirtci/cluster-up/cluster/kind-k8s-sriov-1.17.0/certcreator/certlib"
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

func generate(hookName, namespace string) ([]byte, []byte, error) {
	serviceName := strings.Join([]string{hookName, "service"}, "-")

	certConfig := certlib.SelfSignedCertificate{
		CommonName: strings.Join([]string{serviceName, namespace, "svc"}, "."),
		DNSNames: []string{
			serviceName,
			strings.Join([]string{serviceName, namespace}, "."),
			strings.Join([]string{serviceName, namespace, "svc"}, ".")},
	}
	err := certConfig.Generate()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate self-signed certificate: %v", err)
	}
	log.Printf("Self-Signed certificate created successfully for CN %s", certConfig.CommonName)

	return certConfig.Certificate.Bytes(), certConfig.PrivateKey.Bytes(), nil
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

	err := wait.Poll(time.Second*5, time.Minute*3, func() (bool, error) {
		_, err := clusterApi.CoreV1().Secrets(namespace).Get(secret.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		return false, fmt.Errorf("secret %s already exists", secret.Name)
	})

	if err != nil {
		return err
	}

	err = wait.Poll(time.Second*5, time.Minute*3, func() (bool, error) {
		_, err := clusterApi.CoreV1().Secrets(namespace).Create(secret)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				return true, nil
			}
			log.Printf("failed to create secret '%s': %v", secret.Name, err)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("timeout waiting for secret '%s' to create secret: %v", secret.Name, err)
	}
	log.Printf("Secret '%s' at '%s' created successfully", secret.Name, namespace)

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

	certificate, key, err := generate(*hookName, *namespace)
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
