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

	"github.com/cloudflare/cfssl/csr"
	"github.com/pkg/errors"

	"k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset kubernetes.Interface
	namespace string
	prefix    string
)

func generateCSR() ([]byte, []byte, error) {
	serviceName := strings.Join([]string{prefix, "service"}, "-")
	certRequest := csr.New()
	certRequest.KeyRequest = &csr.KeyRequest{"rsa", 2048}
	certRequest.CN = strings.Join([]string{serviceName, namespace, "svc"}, ".")
	certRequest.Hosts = []string{
		serviceName,
		strings.Join([]string{serviceName, namespace}, "."),
		strings.Join([]string{serviceName, namespace, "svc"}, "."),
	}

	log.Printf("generating Certificate Signing Request %v", certRequest)

	return csr.ParseRequest(certRequest)
}

func getSignedCertificate(request []byte) ([]byte, error) {
	csrName := strings.Join([]string{prefix, "csr"}, "-")
	log.Printf("before")
	csr, err := clientset.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
	log.Printf("after")
	if csr != nil || err == nil {
		log.Printf("CSR %s already exists, removing it first", csrName)
		clientset.CertificatesV1beta1().CertificateSigningRequests().Delete(csrName, &metav1.DeleteOptions{})
	}

	log.Printf("creating new CSR %s", csrName)
	/* build Kubernetes CSR object */
	csr = &v1beta1.CertificateSigningRequest{}
	csr.ObjectMeta.Name = csrName
	csr.ObjectMeta.Namespace = namespace
	csr.Spec.Request = request
	csr.Spec.Groups = []string{"system:authenticated"}
	csr.Spec.Usages = []v1beta1.KeyUsage{v1beta1.UsageDigitalSignature, v1beta1.UsageServerAuth, v1beta1.UsageKeyEncipherment}

	/* push CSR to Kubernetes API server */
	csr, err = clientset.CertificatesV1beta1().CertificateSigningRequests().Create(csr)
	if err != nil {
		return nil, errors.Wrap(err, "error creating CSR in Kubernetes API: %s")
	}
	log.Printf("CSR pushed to the Kubernetes API")

	if csr.Status.Certificate != nil {
		log.Printf("using already issued certificate for CSR %s", csrName)
		return csr.Status.Certificate, nil
	}
	/* approve certificate in K8s API */
	csr.ObjectMeta.Name = csrName
	csr.ObjectMeta.Namespace = namespace
	csr.Status.Conditions = append(csr.Status.Conditions, v1beta1.CertificateSigningRequestCondition{
		Type:           v1beta1.CertificateApproved,
		Reason:         "Approved by net-attach-def admission controller installer",
		Message:        "This CSR was approved by net-attach-def admission controller installer.",
		LastUpdateTime: metav1.Now(),
	})
	csr, err = clientset.CertificatesV1beta1().CertificateSigningRequests().UpdateApproval(csr)
	log.Printf("certificate approval sent")
	if err != nil {
		return nil, errors.Wrap(err, "error approving CSR in Kubernetes API")
	}

	/* wait for the cert to be issued */
	log.Printf("waiting for the signed certificate to be issued...")
	start := time.Now()
	for range time.Tick(time.Second) {
		csr, err = clientset.CertificatesV1beta1().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "error getting signed ceritificate from the API server")
		}
		if csr.Status.Certificate != nil {
			return csr.Status.Certificate, nil
		}
		if time.Since(start) > 60*time.Second {
			break
		}
	}

	return nil, errors.New("error getting certificate from the API server: request timed out - verify that Kubernetes certificate signer is setup, more at https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/#a-note-to-cluster-administrators")
}

// Install creates resources required by mutating admission webhook
func generate(config *rest.Config, k8sNamespace, namePrefix, secretName string) {
	var err error
	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("error setting up Kubernetes client: %s", err)
	}

	namespace = k8sNamespace
	prefix = namePrefix

	/* generate CSR and private key */
	csr, key, err := generateCSR()
	if err != nil {
		log.Fatalf("error generating CSR and private key: %s", err)
	}
	log.Printf("raw CSR and private key successfully created")

	/* obtain signed certificate */
	certificate, err := getSignedCertificate(csr)
	if err != nil {
		log.Fatalf("error getting signed certificate: %s", err)
	}
	log.Printf("signed certificate successfully obtained")

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

	encoded := base64.StdEncoding.EncodeToString(certificate)
	if err := ioutil.WriteFile(namePrefix+".cert", []byte(encoded), 0644); err != nil {
		log.Fatalf("Failed to create file %s", namePrefix)
	}

	_, err = clientset.CoreV1().Secrets(namespace).Create(secret)
	if err != nil {
		log.Fatal("Failed to create secret", err)
	}
	log.Printf("Secret %s %s created", namespace, secretName)

}

func main() {
	namespace := flag.String("namespace", "", "the namespace of the webhook")
	kubeconfig := flag.String("kubeconfig", "", "the path of kubeconfig")
	hookName := flag.String("hook", "", "the name of the hook")
	secretName := flag.String("secret", "", "the name of the secret")
	flag.Parse()

	if *namespace == "" || *hookName == "" || *secretName == "" {
		flag.Usage()
		log.Fatal("Not enough arguments")
	}

	var config *rest.Config
	var err error
	if *kubeconfig == "" {
		*kubeconfig = os.Getenv("KUBECONFIG")
		fmt.Printf("Using env kubeconfig %s", *kubeconfig)
	}

	if *kubeconfig != "" {
		log.Printf("Loading kube client config from path %q", *kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatal("could not get the client", err)
		}
	} else {
		log.Printf("Using in-cluster kube client config")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatal("could not get the client")
		}
	}
	generate(config, *namespace, *hookName, *secretName)
}
