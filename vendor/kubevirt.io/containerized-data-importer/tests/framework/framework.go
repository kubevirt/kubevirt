package framework

import (
	"flag"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	cdiClientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/tests/utils"
	"kubevirt.io/qe-tools/pkg/ginkgo-reporters"
)

const (
	NsCreateTime  = 30 * time.Second
	NsDeleteTime  = 5 * time.Minute
	NsPrefixLabel = "cdi-e2e"
	CdiPodPrefix  = "cdi-deployment"
)

// run-time flags
var (
	kubectlPath  *string
	ocPath       *string
	cdiInstallNs *string
	kubeConfig   *string
	master       *string
	goCLIPath    *string
)

type Config struct {
	// Whether to skip creating a namespace. Use this ONLY for tests that do not require
	// a namespace at all, like basic sanity or other global tests.
	SkipNamespaceCreation bool
	// Whether to skip looking up the name of the cdi controller pod.
	SkipControllerPodLookup bool
}

// Framework supports common operations used by functional/e2e tests. It holds the k8s and cdi clients,
// a generated unique namespace, run-time flags, and more fields will be added over time as cdi e2e
// evolves. Global BeforeEach and AfterEach are called in the Framework constructor.
type Framework struct {
	Config
	// prefix for generated namespace
	NsPrefix string
	//  k8s client
	K8sClient *kubernetes.Clientset
	// cdi client
	CdiClient *cdiClientset.Clientset
	// REST client config.
	RestConfig *rest.Config
	// generated/unique ns per test
	Namespace *v1.Namespace
	// generated/unique secondary ns for testing across namespaces (eg. clone tests)
	Namespace2 *v1.Namespace // note: not instantiated in NewFramework
	// list of ns to delete beyond the generated ns
	namespacesToDelete []*v1.Namespace

	ControllerPod *v1.Pod

	// test run-time flags
	KubectlPath  string
	OcPath       string
	CdiInstallNs string
	KubeConfig   string
	Master       string
	GoCLIPath    string
}

// TODO: look into k8s' SynchronizedBeforeSuite() and SynchronizedAfterSuite() code and their general
//       purpose test/e2e/framework/cleanup.go function support.

// initialize run-time flags
func init() {
	// By accessing something in the ginkgo_reporters package, we are ensuring that the init() is called
	// That init calls flag.StringVar, and makes sure the --junit-output flag is added before we call
	// flag.Parse in NewFramework. Without this, the flag is NOT added.
	fmt.Fprintf(GinkgoWriter, "Making sure junit flag is available %v\n", ginkgo_reporters.JunitOutput)
	kubectlPath = flag.String("kubectl-path", "kubectl", "The path to the kubectl binary")
	ocPath = flag.String("oc-path", "oc", "The path to the oc binary")
	cdiInstallNs = flag.String("cdi-namespace", "kube-system", "The namespace of the CDI controller")
	kubeConfig = flag.String("kubeconfig", "/var/run/kubernetes/admin.kubeconfig", "The absolute path to the kubeconfig file")
	master = flag.String("master", "", "master url:port")
	goCLIPath = flag.String("gocli-path", "cli.sh", "The path to cli script")
}

// NewFrameworkOrDie calls NewFramework and handles errors by calling Fail. Config is optional, but
// if passed there can only be one.
func NewFrameworkOrDie(prefix string, config ...Config) *Framework {
	cfg := Config{}
	if len(config) > 0 {
		cfg = config[0]
	}
	f, err := NewFramework(prefix, cfg)
	if err != nil {
		Fail(fmt.Sprintf("failed to create test framework with config %+v: %v", cfg, err))
	}
	return f
}

// NewFramework makes a new framework and sets up the global BeforeEach/AfterEach's.
// Test run-time flags are parsed and added to the Framework struct.
func NewFramework(prefix string, config Config) (*Framework, error) {
	f := &Framework{
		Config:   config,
		NsPrefix: prefix,
	}

	// handle run-time flags
	if !flag.Parsed() {
		flag.Parse()
		fmt.Fprintf(GinkgoWriter, "** Test flags:\n")
		flag.Visit(func(f *flag.Flag) {
			fmt.Fprintf(GinkgoWriter, "   %s = %q\n", f.Name, f.Value.String())
		})
		fmt.Fprintf(GinkgoWriter, "**\n")
	}

	f.KubectlPath = *kubectlPath
	f.OcPath = *ocPath
	f.CdiInstallNs = *cdiInstallNs
	f.KubeConfig = *kubeConfig
	f.Master = *master
	f.GoCLIPath = *goCLIPath

	restConfig, err := f.LoadConfig()
	if err != nil {
		// Can't use Expect here due this being called outside of an It block, and Expect
		// requires any calls to it to be inside an It block.
		err = errors.Wrap(err, "ERROR, unable to load RestConfig")
	} else {
		f.RestConfig = restConfig
	}

	// clients
	kcs, err := f.GetKubeClient()
	if err != nil {
		return nil, errors.Wrap(err, "ERROR, unable to create K8SClient")
	}
	f.K8sClient = kcs

	cs, err := f.GetCdiClient()
	if err != nil {
		return nil, errors.Wrap(err, "ERROR, unable to create CdiClient")
	}
	f.CdiClient = cs

	BeforeEach(f.BeforeEach)
	AfterEach(f.AfterEach)

	return f, err
}

func (f *Framework) BeforeEach() {
	if !f.SkipControllerPodLookup {
		if f.ControllerPod == nil {
			pod, err := utils.FindPodByPrefix(f.K8sClient, f.CdiInstallNs, CdiPodPrefix, common.CDILabelSelector)
			Expect(err).NotTo(HaveOccurred())
			fmt.Fprintf(GinkgoWriter, "INFO: Located cdi-controller-pod: %q\n", pod.Name)
			f.ControllerPod = pod
		}
	}

	if !f.SkipNamespaceCreation {
		// generate unique primary ns (ns2 not created here)
		By(fmt.Sprintf("Building a %q namespace api object", f.NsPrefix))
		ns, err := f.CreateNamespace(f.NsPrefix, map[string]string{
			NsPrefixLabel: f.NsPrefix,
		})
		Expect(err).NotTo(HaveOccurred())
		f.Namespace = ns
		f.AddNamespaceToDelete(ns)
	}
}

func (f *Framework) AfterEach() {
	// delete the namespace(s) in a defer in case future code added here could generate
	// an exception. For now there is only a defer.
	defer func() {
		for _, ns := range f.namespacesToDelete {
			defer func() { f.namespacesToDelete = nil }()
			if ns == nil || len(ns.Name) == 0 {
				continue
			}
			By(fmt.Sprintf("Destroying namespace %q for this suite.", ns.Name))
			err := DeleteNS(f.K8sClient, ns.Name)
			Expect(err).NotTo(HaveOccurred())
		}
	}()
	return
}

// Instantiate a new namespace object with a unique name and the passed-in label(s).
func (f *Framework) CreateNamespace(prefix string, labels map[string]string) (*v1.Namespace, error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("cdi-e2e-tests-%s-", prefix),
			Namespace:    "",
			Labels:       labels,
		},
		Status: v1.NamespaceStatus{},
	}

	var nsObj *v1.Namespace
	c := f.K8sClient
	err := wait.PollImmediate(2*time.Second, NsCreateTime, func() (bool, error) {
		var err error
		nsObj, err = c.CoreV1().Namespaces().Create(ns)
		if err == nil || apierrs.IsAlreadyExists(err) {
			return true, nil // done
		}
		glog.Warningf("Unexpected error while creating %q namespace: %v", ns.GenerateName, err)
		return false, err // keep trying
	})
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(GinkgoWriter, "INFO: Created new namespace %q\n", nsObj.Name)
	return nsObj, nil
}

func (f *Framework) AddNamespaceToDelete(ns *v1.Namespace) {
	f.namespacesToDelete = append(f.namespacesToDelete, ns)
}

func DeleteNS(c *kubernetes.Clientset, ns string) error {
	return wait.PollImmediate(2*time.Second, NsDeleteTime, func() (bool, error) {
		err := c.CoreV1().Namespaces().Delete(ns, nil)
		if err != nil && !apierrs.IsNotFound(err) {
			glog.Warningf("namespace %q Delete api err: %v", ns, err)
			return false, nil // keep trying
		}
		// see if ns is really deleted
		_, err = c.CoreV1().Namespaces().Get(ns, metav1.GetOptions{})
		if apierrs.IsNotFound(err) {
			return true, nil // deleted, done
		}
		if err != nil {
			glog.Warningf("namespace %q Get api error: %v", ns, err)
		}
		return false, nil // keep trying
	})
}

// Gets an instance of a kubernetes client that includes all the CDI extensions.
func (f *Framework) GetCdiClient() (*cdiClientset.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(f.Master, f.KubeConfig)
	if err != nil {
		return nil, err
	}
	cdiClient, err := cdiClientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return cdiClient, nil
}

func (f *Framework) GetKubeClient() (*kubernetes.Clientset, error) {
	return GetKubeClientFromRESTConfig(f.RestConfig)
}

func (f *Framework) LoadConfig() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags(f.Master, f.KubeConfig)
}

func GetKubeClientFromRESTConfig(config *rest.Config) (*kubernetes.Clientset, error) {
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	return kubernetes.NewForConfig(config)
}
