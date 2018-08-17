package tests

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"

	k8sv1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	"kubevirt.io/containerized-data-importer/pkg/util"
)

var KubectlPath = ""
var OcPath = ""
var CDIInstallNamespace = "kube-system"

const (
	defaultTimeout      = 30 * time.Second
	testNamespacePrefix = "cdi-test-"
)

var (
	kubeconfig string
	master     string
)

func init() {
	flag.StringVar(&KubectlPath, "kubectl-path", "", "Set path to kubectl binary")
	flag.StringVar(&OcPath, "oc-path", "", "Set path to oc binary")
	flag.StringVar(&CDIInstallNamespace, "installed-namespace", "kube-system", "Set the namespace CDI is installed in")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
}

// CDIFailHandler call ginkgo.Fail with printing the additional information
func CDIFailHandler(message string, callerSkip ...int) {
	if len(callerSkip) > 0 {
		callerSkip[0]++
	}
	Fail(message, callerSkip...)
}

func RunKubectlCommand(args ...string) (string, error) {
	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		return "", fmt.Errorf("can not find kubeconfig")
	}

	master := flag.Lookup("master").Value
	if master != nil && master.String() != "" {
		args = append(args, "--server", master.String())
	}

	cmd := exec.Command(KubectlPath, args...)
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())
	cmd.Env = append(os.Environ(), kubeconfEnv)

	stdOutBytes, err := cmd.Output()
	if err != nil {
		return string(stdOutBytes), err
	}
	return string(stdOutBytes), nil
}

func SkipIfNoKubectl() {
	if KubectlPath == "" {
		Skip("Skip test that requires kubectl binary")
	}
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}

// Gets an instance of a kubernetes client that includes all the CDI extensions.
func GetCDIClientOrDie() *clientset.Clientset {

	cfg, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	PanicOnError(err)
	cdiClient, err := clientset.NewForConfig(cfg)
	PanicOnError(err)

	return cdiClient
}

func GetKubeClient() (*kubernetes.Clientset, error) {
	return GetKubeClientFromFlags(master, kubeconfig)
}

func GetKubeClientFromFlags(master string, kubeconfig string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}
	return GetKubeClientFromRESTConfig(config)
}

func GetKubeClientFromRESTConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	return kubernetes.NewForConfig(config)
}

// Creates a new namespace with a randomly generated name that starts with cdi-test-
// and a base name, the base name has to conform to kubernetes namespace standards.
// for instance a base name of test-basic, will generate a name cdi-test-test-basic-sdlea4fsde
// but test_basic will cause a failure as it doesn't match the namespace standards.
func GenerateNamespace(client *kubernetes.Clientset, baseName string) *k8sv1.Namespace {
	var namespace *k8sv1.Namespace
	var err error
	nsDef := generateRandomNamespaceName(baseName)
	if wait.PollImmediate(2*time.Second, defaultTimeout, func() (bool, error) {
		namespace, err = client.CoreV1().Namespaces().Create(nsDef)
		if err != nil {
			if apierrs.IsAlreadyExists(err) {
				nsDef = generateRandomNamespaceName(baseName)
			}
			return false, nil
		}
		return true, nil
	}) != nil {
		Fail("Unable to create namespace: " + nsDef.GetName())
	}
	return namespace
}

func generateRandomNamespaceName(baseName string) *k8sv1.Namespace {
	namespaceName := fmt.Sprintf(testNamespacePrefix+"%s-%s", baseName, strings.ToLower(util.RandAlphaNum(10)))
	return &k8sv1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
}

// Destroys the passed in name space, be sure to clean any resources used before destroying the namespace.
func DestroyNamespace(client *kubernetes.Clientset, namespace *k8sv1.Namespace) {
	if wait.PollImmediate(2*time.Second, defaultTimeout, func() (bool, error) {
		err := client.CoreV1().Namespaces().Delete(namespace.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				return true, nil
			}
			return false, nil
		}
		return true, nil
	}) != nil {
		Fail("Unable to remove namespace: " + namespace.GetName())
	}
}

func DestroyAllTestNamespaces(client *kubernetes.Clientset) {
	var namespaces *k8sv1.NamespaceList
	var err error
	if wait.PollImmediate(2*time.Second, defaultTimeout, func() (bool, error) {
		namespaces, err = client.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	}) != nil {
		Fail("Unable to list namespaces")
	}

	for _, namespace := range namespaces.Items {
		if strings.HasPrefix(namespace.GetName(), testNamespacePrefix) {
			DestroyNamespace(client, &namespace)
		}
	}
}
