/*
Copyright 2017 Red Hat Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at:

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file implements a tool that simplifies registration of API services.

package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"

	certv1 "k8s.io/api/certificates/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"kubevirt.io/kubevirt/pkg/logging"
)

// Values of the command line flags:
//
var (
	apiGroup           string
	apiGroupPriority   int
	apiVersion         string
	apiVersionPriority int
	approveTimeout     int64
	autoApprove        bool
	dnsDomain          string
	kubeConfig         string
	namespaceName      string
	secretName         string
	serviceName        string
	servicePort        int
	targetPort         int
	targetSelector     string
)

// Keys used to save data inside the secret:
//
const (
	caCertKey = "ca.crt"
)

// Names of PEM blocks:
//
const (
	csrBlockType           = "CERTIFICATE REQUEST"
	rsaPrivateKeyBlockType = "RSA PRIVATE KEY"
)

// The logger:
//
var log *logging.FilteredLogger

func main() {
	// Define the command line options:
	flag.StringVar(
		&kubeConfig,
		"kubeconfig",
		"",
		"Path to the kubeconfig file.",
	)
	flag.StringVar(
		&dnsDomain,
		"dns-domain",
		"cluster.local",
		"The DNS domain name of the cluster.",
	)
	flag.StringVar(
		&namespaceName,
		"namespace",
		"default",
		"The name of the namespace.",
	)
	flag.StringVar(
		&apiGroup,
		"api-group",
		"",
		"The API group.",
	)
	flag.StringVar(
		&apiVersion,
		"api-version",
		"",
		"The API version.",
	)
	flag.IntVar(
		&apiGroupPriority,
		"api-group-priority",
		2000,
		"The priority of the API group.",
	)
	flag.IntVar(
		&apiVersionPriority,
		"api-version-priority",
		10,
		"The priority of the API version.",
	)
	flag.StringVar(
		&serviceName,
		"service-name",
		"",
		"The name of the service that will handles the traffic to the API server. The default "+
			"is 'apiservice' followed by the API group and version.",
	)
	flag.StringVar(
		&secretName,
		"secret-name",
		"",
		"The name of the secret where the tool will save the CA, the private key and the certificate. "+
			"The default is 'apiservice' followed by the API group and version.",
	)
	flag.StringVar(
		&targetSelector,
		"target-selector",
		"",
		"The selector used to find the pod that contains the API server.",
	)
	flag.IntVar(
		&servicePort,
		"service-port",
		443,
		"The port where the service that handles the traffic to the API server will be listening. ",
	)
	flag.IntVar(
		&targetPort,
		"target-port",
		443,
		"The port where the where the pod that contains the API server is listening.",
	)
	flag.BoolVar(
		&autoApprove,
		"auto-approve",
		false,
		"Indicates if the tool should try to approve the certificate signing request that it will "+
			"generate. Note that this requires permissions that are not to be granted lightly, use "+
			"with care.",
	)
	flag.Int64Var(
		&approveTimeout,
		"approve-timeout",
		0,
		"The maximum time to wait for approval of the certificate, in seconds. The default value is "+
			"zero, which means wait for ever.",
	)

	// Initialize logging:
	logging.InitializeLogging("virt-apiservice-register")
	log = logging.DefaultLogger()

	// Parse the command line:
	flag.Parse()

	// Check that mandatory options have been provided:
	ok := true
	if apiGroup == "" {
		log.Error().Msgf("The -api-group option is mandatory")
		ok = false
	}
	if apiVersion == "" {
		log.Error().Msgf("The -api-version option is mandatory")
		ok = false
	}
	if targetSelector == "" {
		log.Error().Msgf("The -target-selector option is mandatory")
		ok = false
	}
	if !ok {
		os.Exit(1)
	}

	// Calculate default names for the services and the secret:
	dotReplacer := strings.NewReplacer(".", "-")
	defaultName := fmt.Sprintf(
		"apiservice-%s-%s",
		dotReplacer.Replace(apiGroup),
		dotReplacer.Replace(apiVersion),
	)
	if serviceName == "" {
		serviceName = defaultName
	}
	if secretName == "" {
		secretName = defaultName
	}

	// Run the tool and report unhandled errors:
	err := run()
	if err != nil {
		reportError(err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}

func run() error {
	// Load the configuration and create the API clients. Note that in addition to the typed client
	// we also need a dynamic client, because the 'client-go' package doesn't currently support the
	// API service registration API in the typed way.
	client, dyclient, err := createClients()
	if err != nil {
		return err
	}

	// Make sure that the secret exists:
	secret, err := ensureSecret(client)
	if err != nil {
		return err
	}

	// Make sure that the CA certificate has been saved to the secret:
	ca, err := ensureCA(client, secret)
	if err != nil {
		return err
	}

	// Make sure that the private key has been generated and saved to the secret:
	key, err := ensurePrivateKey(client, secret)
	if err != nil {
		return err
	}

	// Make sure that the certificate has been generated, approved and saved to the secret:
	_, err = ensureCertificate(client, secret, key, approveTimeout)
	if err != nil {
		return err
	}

	// Make sure that the service exists:
	err = ensureService(client)
	if err != nil {
		return err
	}

	// Make sure that the API service exists:
	err = ensureAPIService(dyclient, ca)
	if err != nil {
		return err
	}

	return nil
}

// createClients loads the Kubernetes API client configuration using the value of the kubeconfig
// command line option and creates the API clients.
//
func createClients() (client *kubernetes.Clientset, dyclient *dynamic.Client, err error) {
	// Create the typed client, which will be used for most of the interactions with the API:
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return
	}
	client, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return
	}

	// Create the dynamic client, which will be used for the API service registration API, as it
	// isn't yet supported by the typed part of the 'client-go' package.
	dycfg, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return
	}
	dycfg.APIPath = "/apis/"
	dycfg.GroupVersion = &schema.GroupVersion{
		Group:   "apiregistration.k8s.io",
		Version: "v1beta1",
	}
	dyclient, err = dynamic.NewClient(dycfg)
	if err != nil {
		return
	}

	return
}

// ensureSecret ensures that the secret that will contain the CA, the private key and certificate
// exists. If it doesn't exist it will be created.
//
func ensureSecret(client *kubernetes.Clientset) (secret *corev1.Secret, err error) {
	// Get the reference to the secrets resource:
	secrets := client.CoreV1().Secrets(namespaceName)

	// If the secret already exists, then we don't need to do anything else:
	secret, err = secrets.Get(secretName, metav1.GetOptions{})
	if err == nil {
		log.Info().Msgf("Secret '%s' already exists", secretName)
		return
	}
	if errors.IsNotFound(err) {
		log.Info().Msgf("Secret '%s' doesn't exist", secretName)
	} else {
		return
	}

	// Create the new secret. Note that when creating a TLS secret it is mandatory to set the
	// tls.key and tls.crt keys. But we don't have the values yet, so we set them to empty arrays of
	// bytes.
	log.Info().Msgf("Creating secret '%s'", secretName)
	secret, err = secrets.Create(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSPrivateKeyKey: []byte{},
			corev1.TLSCertKey:       []byte{},
		},
	})

	return
}

// ensureCA ensures that the CA certificate has been found and saved to the secret.
//
func ensureCA(client *kubernetes.Clientset, secret *corev1.Secret) (ca []byte, err error) {
	// Get the reference to the secrets resource:
	secrets := client.CoreV1().Secrets(namespaceName)

	// Update the secret with the CA if needed:
	ca = secret.Data[caCertKey]
	if ca == nil || len(ca) == 0 {
		ca, err = findCA(client)
		if err != nil {
			return
		}
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[caCertKey] = ca
		var updated *corev1.Secret
		updated, err = secrets.Update(secret)
		if err != nil {
			return
		}
		*secret = *updated
	} else {
		log.Info().Msgf("CA certificate already exists")
	}

	return
}

// findCA tries to find the certificate of the CA that signs the certificates for API servers.
// Returns an array of bytes containing the CA certificate in PEM format, or nil if the CA can't be
// found.
//
func findCA(client *kubernetes.Clientset) (ca []byte, err error) {
	// Get the reference to the configuration maps service:
	configMaps := client.CoreV1().ConfigMaps("kube-public")

	// Retrieve the details of the configuration map:
	clusterInfo, err := configMaps.Get("cluster-info", metav1.GetOptions{})
	if err != nil {
		return
	}

	// The CA certificate is stored inside a 'kubeconfig' file that is stored inside the
	// 'cluster-info' configuration map under the 'kubeconfig' key:
	text := clusterInfo.Data["kubeconfig"]
	if text == "" {
		err = fmt.Errorf("Can't find the 'kubeconfig' data")
		return
	}
	config, err := clientcmd.Load([]byte(text))
	if err != nil {
		return
	}
	cluster := config.Clusters[""]
	if cluster == nil {
		err = fmt.Errorf("Can't find the default cluster in the 'kubeconfig' data")
		return
	}
	ca = cluster.CertificateAuthorityData
	if ca == nil {
		err = fmt.Errorf("Can't find CA in the 'kubeconfig' data")
		return
	}
	log.Info().Msgf("Found CA")
	log.Debug().Msgf("%s", ca)

	return
}

// ensurePrivateKey ensures that the key pair has been generated and saved to the secret.
//
func ensurePrivateKey(client *kubernetes.Clientset, secret *corev1.Secret) (key []byte, err error) {
	// Get the reference to the secrets resource:
	secrets := client.CoreV1().Secrets(namespaceName)

	// Check if the secret already contains a private key. If it doesn't then generate a new one and
	// update the secret.
	key = secret.Data[corev1.TLSPrivateKeyKey]
	if key == nil || len(key) == 0 {
		key, err = makeKey()
		if err != nil {
			return
		}
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[corev1.TLSPrivateKeyKey] = key
		var updated *corev1.Secret
		updated, err = secrets.Update(secret)
		if err != nil {
			return
		}
		*secret = *updated
	} else {
		log.Info().Msgf("Private key already exists")
	}

	return
}

// ensureCertificate ensures that the certificate has been generated, signed, and saved to the
// secret.
//
func ensureCertificate(client *kubernetes.Clientset, secret *corev1.Secret, key []byte, timeout int64) (cert []byte, err error) {
	// Get the reference to the secrets resource:
	secrets := client.CoreV1().Secrets(namespaceName)

	// Check if the secret already contains the certificagte. If it doesn't then request a new one
	// and update the secret.
	cert = secret.Data[corev1.TLSCertKey]
	if cert == nil || len(cert) == 0 {
		cert, err = makeCertificate(client, key, timeout)
		if err != nil {
			return
		}
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[corev1.TLSCertKey] = key
		var updated *corev1.Secret
		updated, err = secrets.Update(secret)
		if err != nil {
			return
		}
		*secret = *updated
	} else {
		log.Info().Msgf("Certificate already exists")
	}

	return
}

// makeCertificate creates a certificate signing request for the given private key, sends it to the
// server and waits till it is approved or denied. If the request is approved it returns an array of
// bytes containing the certificate, in PEM format.
//
func makeCertificate(client *kubernetes.Clientset, key []byte, timeout int64) (cert []byte, err error) {
	// Calculate the CN:
	cn, hosts := findCN()
	log.Info().Msgf("The CN for the certificate is '%s'", cn)
	for _, host := range hosts {
		log.Info().Msgf("Additional host name for the certificate is '%s'", host)
	}

	// Try to retrieve an existing CSR:
	csr, err := findCSR(client, cn)
	if errors.IsNotFound(err) {
		err = nil
	}
	if err != nil {
		return
	}

	// If the CSR already exists then we need to check that it is compatible with the current
	// private key and CN. If it isn't then we will need to delete it and create a new one, as it
	// will be impossible to restore the private key that was used to create it.
	if csr != nil {
		log.Info().Msgf("The CSR '%s' already exists", cn)
		log.Debug().Msgf("%s", csr.Spec.Request)
		var ok bool
		ok, err = checkCSR(csr, key, cn)
		if err != nil {
			return
		}
		if !ok {
			log.Info().Msgf("Deleting CSR '%s'", cn)
			err = deleteCSR(client, cn)
			if err != nil {
				return
			}
			csr = nil
		}
	}

	// At this point the CSR may not exist, or may have been deleted due to incompatibility with the
	// current private key or CN, so we need to check and create it if needed:
	if csr == nil {
		log.Info().Msgf("Creating CSR '%s'", cn)
		csr, err = createCSR(client, key, cn, hosts)
		if err != nil {
			return
		}
		log.Info().Msgf("The CSR '%s' has been created", cn)
		log.Debug().Msgf("%s", cn, csr.Spec.Request)
	}

	// Try to approve the CSR:
	if autoApprove {
		log.Info().Msgf("Approving CSR '%s'", cn)
		csr, err = approveCSR(client, cn)
		if err != nil {
			return
		}
	}

	// Wait till the CSR is approved, denied or removed:
	if timeout > 0 {
		log.Info().Msgf("Waiting up to %d seconds for approval of CSR '%s'", timeout, cn)
	} else {
		log.Info().Msgf("Waiting indefinitely for approval of CSR '%s'", cn)
	}
	csr, err = waitCSR(client, cn, timeout)
	if err != nil {
		return
	}
	if csr == nil {
		err = fmt.Errorf("The CSR '%s' has been removed", cn)
	} else if isCSRDenied(csr) {
		err = fmt.Errorf("The CSR '%s' has been denied", cn)
	} else if isCSRApproved(csr) {
		cert = csr.Status.Certificate
		log.Info().Msgf("The CSR '%s' has been approved", cn)
		log.Debug().Msgf("%s", cert)
	} else {
		err = fmt.Errorf("The CSR '%s' hasn't been approved", cn)
	}

	return
}

// findCSR tries to find an existing certificate signing request with the given CN.
//
func findCSR(client *kubernetes.Clientset, cn string) (csr *certv1.CertificateSigningRequest, err error) {
	// Get the reference to the CSR resource:
	csrs := client.CertificatesV1beta1().CertificateSigningRequests()

	// Retrieve the CSR:
	csr, err = csrs.Get(cn, metav1.GetOptions{})
	return
}

// checkCSR checks if the given CSR is compatible with the given private key and CN.
//
func checkCSR(csr *certv1.CertificateSigningRequest, key []byte, cn string) (ok bool, err error) {
	// Decode the CSR:
	var req *x509.CertificateRequest
	req, err = decodeCSR(csr.Spec.Request)
	if err != nil {
		return
	}

	// Decode the private key:
	var pair *rsa.PrivateKey
	pair, err = decodeKey(key)
	if err != nil {
		return
	}

	// Check if the public key in the CSR is compatible with the public key that corresponds to the
	// private key:
	if !comparePublicKeys(req.PublicKey, &pair.PublicKey) {
		log.Error().Msgf("The public key in the CSR isn't compatible with the private key")
		return
	}

	// Check that the CN in the CSR is correct:
	if req.Subject.CommonName != cn {
		log.Error().Msgf("The CN in the CSR is '%s', but should be '%s'", req.Subject.CommonName, cn)
		return
	}

	// Everything matches:
	ok = true
	return
}

// deleteCSR deletes the CSR for the given CN.
//
func deleteCSR(client *kubernetes.Clientset, cn string) error {
	// Get the reference to the CSR resource:
	csrs := client.CertificatesV1beta1().CertificateSigningRequests()

	// Send the delete request:
	return csrs.Delete(cn, &metav1.DeleteOptions{})
}

// createCSR creates a key pair and a certificate singing request. It sends the request to the server
// and returns the CSR and the bytes of the private key, in PEM format.
//
func createCSR(client *kubernetes.Clientset, key []byte, cn string, addrs []string) (csr *certv1.CertificateSigningRequest, err error) {
	// Generate the CSR:
	bytes, err := makeCSR(key, cn, addrs)
	if err != nil {
		return
	}

	// Get the reference to the CSR resource:
	csrs := client.CertificatesV1beta1().CertificateSigningRequests()

	// Send the CSR to the server:
	csr, err = csrs.Create(
		&certv1.CertificateSigningRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: cn,
			},
			Spec: certv1.CertificateSigningRequestSpec{
				Groups: []string{
					"system:authenticated",
				},
				Request: bytes,
				Usages: []certv1.KeyUsage{
					certv1.UsageDigitalSignature,
					certv1.UsageKeyEncipherment,
					certv1.UsageServerAuth,
				},
			},
		},
	)

	return
}

// approveCSR approves the CSR with the given CN.
//
func approveCSR(client *kubernetes.Clientset, cn string) (csr *certv1.CertificateSigningRequest, err error) {
	// Get the reference to the CSR resource:
	csrs := client.CertificatesV1beta1().CertificateSigningRequests()

	// Retrieve the current data of the CSR:
	csr, err = findCSR(client, cn)
	if err != nil {
		return
	}

	// Add a new condition indicating that the CSR has been approved:
	approved := certv1.CertificateSigningRequestCondition{
		Type:           certv1.CertificateApproved,
		Reason:         "KubeVirtApprove",
		Message:        "This CSR was automatically approved by the KubeVirt API service registration tool.",
		LastUpdateTime: metav1.Now(),
	}
	csr.Status.Conditions = append(csr.Status.Conditions, approved)

	// Send the update to the server:
	csr, err = csrs.UpdateApproval(csr)
	if err != nil {
		return
	}

	return
}

// waitCSR waits till the CSR for the given CN has been approved, denied or removed. The complete
// CSR will be returned unless there is an error.
//
func waitCSR(client *kubernetes.Clientset, cn string, timeout int64) (csr *certv1.CertificateSigningRequest, err error) {
	var ok bool

	// Get the reference to the CSR resource:
	csrs := client.CertificatesV1beta1().CertificateSigningRequests()

	// Retrieve the latest state of the CSR and check if it is already approved or denied:
	csr, err = csrs.Get(cn, metav1.GetOptions{})
	if err != nil {
		return
	}
	if isCSRApproved(csr) || isCSRDenied(csr) {
		return
	}

	// Watch for changes to the CSR, till it is approved, denied or removed:
	csrWatch, err := csrs.Watch(
		metav1.ListOptions{
			FieldSelector:   "metadata.name=" + cn,
			ResourceVersion: csr.ResourceVersion,
			TimeoutSeconds:  &timeout,
		},
	)
	if err != nil {
		return
	}
	csrCh := csrWatch.ResultChan()
	for csrEvent := range csrCh {
		switch csrEvent.Type {
		case watch.Modified:
			if csr, ok = csrEvent.Object.(*certv1.CertificateSigningRequest); ok {
				if isCSRApproved(csr) || isCSRDenied(csr) {
					csrWatch.Stop()
				}
			}
		case watch.Deleted:
			csr = nil
			csrWatch.Stop()
		}
	}

	return
}

// isCSRApproved checks if the CSR has the been approved and the corresponding certificate has been
// generated.
//
func isCSRApproved(csr *certv1.CertificateSigningRequest) bool {
	return hasCSRCondition(csr, certv1.CertificateApproved) && csr.Status.Certificate != nil
}

// isCSRDenied checks if the CSR has been denied.
//
func isCSRDenied(csr *certv1.CertificateSigningRequest) bool {
	return hasCSRCondition(csr, certv1.CertificateDenied)
}

// hasCSRCondition checks if the CSR contains the given condition in the 'status.conditions' array.
//
func hasCSRCondition(csr *certv1.CertificateSigningRequest, condition certv1.RequestConditionType) bool {
	conditions := csr.Status.Conditions
	for _, current := range conditions {
		if current.Type == condition {
			return true
		}
	}
	return false
}

// findCN calculates the common name for the certificate. It returns the calculated CN as well as
// other names that should also be included in the certificate.
//
func findCN() (cn string, names []string) {
	// Currently the API aggregator creates HTTPS connections to the API services using incomplete
	// DNS names in the URL. If the name of the service is 'myservice' it uses 'myservice.default.svc'
	// when the correct DNS name should be 'myservice.default.svc.cluster.local'. This is probably a
	// bug in the API aggreator. To work it around we will add to the set of names that will be
	// included in the certificate both the correct and incorrect names.
	names = []string{
		fmt.Sprintf("%s.%s.svc.%s", serviceName, namespaceName, dnsDomain),
		fmt.Sprintf("%s.%s.svc", serviceName, namespaceName),
	}

	// Select the first name as the CN:
	cn = names[0]

	return
}

// ensureService ensures that the service that handles TCP traffic to the API service exists.
// If it doesn't exist it will be created.
//
func ensureService(client *kubernetes.Clientset) error {
	// Get the reference to the services resource:
	services := client.CoreV1().Services(namespaceName)

	// If the service already exists, then we don't need to do anything else:
	_, err := services.Get(serviceName, metav1.GetOptions{})
	if err == nil {
		log.Info().Msgf("Service '%s' already exists", serviceName)
		return nil
	}
	if errors.IsNotFound(err) {
		log.Info().Msgf("Service service '%s' doesn't exist", serviceName)
	} else {
		return err
	}

	// Parse the selector into a set of labels:
	labels, err := parseSelector(targetSelector)
	if err != nil {
		return err
	}

	// Create the new service:
	log.Info().Msgf("Creating service '%s'", serviceName)
	_, err = services.Create(&corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Protocol:   corev1.ProtocolTCP,
					Port:       int32(servicePort),
					TargetPort: intstr.FromInt(targetPort),
				},
			},
		},
	})
	return err
}

func parseSelector(selector string) (labels map[string]string, err error) {
	labels = map[string]string{}
	for _, spec := range strings.Split(selector, ",") {
		parts := strings.Split(spec, "=")
		if len(parts) != 2 {
			err = fmt.Errorf("Selector spec '%s' is incorrect, should have two parts", spec)
			return
		}
		name := parts[0]
		value := parts[1]
		if name == "" {
			err = fmt.Errorf("Selector spec '%s' is incorrect, the name is empty", spec)
			return
		}
		labels[name] = value
	}
	return
}

// ensureAPIService ensures that the API service exists.  If it doesn't exist it will be created.
//
func ensureAPIService(client *dynamic.Client, ca []byte) error {
	// Get the reference to the resource for API service registration:
	registration := client.Resource(
		&metav1.APIResource{
			Name: "apiservices",
		},
		"",
	)

	// If the API service already exists, then we don't need to do anything else:
	apiServiceName := fmt.Sprintf("%s.%s", apiVersion, apiGroup)
	_, err := registration.Get(apiServiceName, metav1.GetOptions{})
	if err == nil {
		log.Info().Msgf("API service '%s' already exists", apiServiceName)
		return nil
	}
	if errors.IsNotFound(err) {
		log.Info().Msgf("API service service '%s' doesn't exist", apiServiceName)
	} else {
		return err
	}

	// Create the new API service:
	log.Info().Msgf("Creating API service '%s'", apiServiceName)
	_, err = registration.Create(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"name": apiServiceName,
			},
			"spec": map[string]interface{}{
				"group":   apiGroup,
				"version": apiVersion,
				"service": map[string]interface{}{
					"name":      serviceName,
					"namespace": namespaceName,
				},
				"insecureSkipTLSVerify": false,
				"caBundle":              base64.StdEncoding.EncodeToString(ca),
				"groupPriorityMinimum":  apiGroupPriority,
				"versionPriority":       apiVersionPriority,
			},
		},
	})
	return err
}

// makeKey generates a new key pair, and returns the private key in PEM format.
//
func makeKey() (key []byte, err error) {
	// Generate a key pair:
	pair, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	// Wrap it using PEM format:
	key = x509.MarshalPKCS1PrivateKey(pair)
	key = pem.EncodeToMemory(&pem.Block{
		Type:  rsaPrivateKeyBlockType,
		Bytes: key,
	})

	return
}

// makeCSR generates a certificate signing request from the given private key. The CSR will contain
// the given common name in the subject, and the given names in the subject alternate names
// extension. It returns the CSR in PEM format.
//
func makeCSR(key []byte, cn string, names []string) (csr []byte, err error) {
	// Decode the private key:
	pair, err := decodeKey(key)
	if err != nil {
		return
	}

	// Create and serialize the CSR to an array of bytes:
	csr, err = x509.CreateCertificateRequest(
		rand.Reader,
		&x509.CertificateRequest{
			Subject: pkix.Name{
				CommonName: cn,
			},
			DNSNames: names,
		},
		pair,
	)
	if err != nil {
		return
	}

	// Wrap the CSR using PEM format:
	csr = pem.EncodeToMemory(&pem.Block{
		Type:  csrBlockType,
		Bytes: csr,
	})

	return
}

func decodeKey(data []byte) (key *rsa.PrivateKey, err error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != rsaPrivateKeyBlockType {
		err = fmt.Errorf("Can't decode private key")
		return
	}
	key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	return
}

func decodeCSR(data []byte) (req *x509.CertificateRequest, err error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != csrBlockType {
		err = fmt.Errorf("Can't decode CSR")
		return
	}
	req, err = x509.ParseCertificateRequest(block.Bytes)
	return
}

func comparePublicKeys(left, right crypto.PublicKey) bool {
	leftRsa, leftOk := left.(*rsa.PublicKey)
	rightRsa, rightOk := right.(*rsa.PublicKey)
	if leftOk && rightOk {
		return compareRSAPublicKeys(leftRsa, rightRsa)
	}
	return false
}

func compareRSAPublicKeys(left, right *rsa.PublicKey) bool {
	return left.N.Cmp(right.N) == 0 && left.E == right.E
}

// reportError reports the given error to the log, extracting additional for errors returned by the
// Kubernetes API.
//
func reportError(err error) {
	switch typed := err.(type) {
	case *errors.StatusError:
		status := typed.ErrStatus
		log.Error().Msgf("Request failed with status code %d", status.Code)
		if status.Status != "" {
			log.Error().Msgf("Status is '%s'", status.Status)
		}
		if status.Reason != "" {
			log.Error().Msgf("Reason is '%s'", status.Reason)
		}
		if status.Message != "" {
			log.Error().Msgf("Message is '%s'", status.Message)
		}
		details := status.Details
		if details != nil {
			if details.Name != "" {
				log.Error().Msgf("Detail name is '%s'", details.Name)
			}
			if details.Group != "" {
				log.Error().Msgf("Detail group is '%s'", details.Group)
			}
			if details.Kind != "" {
				log.Error().Msgf("Detail kind is '%s'", details.Kind)
			}
			if details.UID != "" {
				log.Error().Msgf("Detail UID is '%s'", details.UID)
			}
			causes := details.Causes
			if causes != nil {
				for i, cause := range causes {
					if cause.Type != "" {
						log.Error().Msgf("Cause %d type is '%s'", i, cause.Type)
					}
					if cause.Field != "" {
						log.Error().Msgf("Cause %d field is '%s'", i, cause.Field)
					}
				}
			}
		}
	default:
		log.Error().Msgf("%s", err)
	}
}
