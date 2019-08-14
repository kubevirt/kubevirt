package operatorclient

import (
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClientInterface interface {
	KubernetesInterface() kubernetes.Interface
	ApiextensionsV1beta1Interface() apiextensions.Interface
	CustomResourceClient
	ServiceAccountClient
	DeploymentClient
}

// CustomResourceClient contains methods for the Custom Resource.
type CustomResourceClient interface {
	GetCustomResource(apiGroup, version, namespace, resourceKind, resourceName string) (*unstructured.Unstructured, error)
	GetCustomResourceRaw(apiGroup, version, namespace, resourceKind, resourceName string) ([]byte, error)
	CreateCustomResource(item *unstructured.Unstructured) error
	CreateCustomResourceRaw(apiGroup, version, namespace, kind string, data []byte) error
	CreateCustomResourceRawIfNotFound(apiGroup, version, namespace, kind, name string, data []byte) (bool, error)
	UpdateCustomResource(item *unstructured.Unstructured) error
	UpdateCustomResourceRaw(apiGroup, version, namespace, resourceKind, resourceName string, data []byte) error
	CreateOrUpdateCustomeResourceRaw(apiGroup, version, namespace, resourceKind, resourceName string, data []byte) error
	DeleteCustomResource(apiGroup, version, namespace, resourceKind, resourceName string) error
	AtomicModifyCustomResource(apiGroup, version, namespace, resourceKind, resourceName string, f CustomResourceModifier, data interface{}) error
	ListCustomResource(apiGroup, version, namespace, resourceKind string) (*CustomResourceList, error)
}

// ServiceAccountClient contains methods for manipulating ServiceAccount.
type ServiceAccountClient interface {
	CreateServiceAccount(*v1.ServiceAccount) (*v1.ServiceAccount, error)
	GetServiceAccount(namespace, name string) (*v1.ServiceAccount, error)
	UpdateServiceAccount(modified *v1.ServiceAccount) (*v1.ServiceAccount, error)
	DeleteServiceAccount(namespace, name string, options *metav1.DeleteOptions) error
}

// DeploymentClient contains methods for the Deployment resource.
type DeploymentClient interface {
	GetDeployment(namespace, name string) (*appsv1.Deployment, error)
	CreateDeployment(*appsv1.Deployment) (*appsv1.Deployment, error)
	DeleteDeployment(namespace, name string, options *metav1.DeleteOptions) error
	UpdateDeployment(*appsv1.Deployment) (*appsv1.Deployment, bool, error)
	PatchDeployment(*appsv1.Deployment, *appsv1.Deployment) (*appsv1.Deployment, bool, error)
	RollingUpdateDeployment(*appsv1.Deployment) (*appsv1.Deployment, bool, error)
	RollingPatchDeployment(*appsv1.Deployment, *appsv1.Deployment) (*appsv1.Deployment, bool, error)
	RollingUpdateDeploymentMigrations(namespace, name string, f UpdateFunction) (*appsv1.Deployment, bool, error)
	RollingPatchDeploymentMigrations(namespace, name string, f PatchFunction) (*appsv1.Deployment, bool, error)
	CreateOrRollingUpdateDeployment(*appsv1.Deployment) (*appsv1.Deployment, bool, error)
	ListDeploymentsWithLabels(namespace string, labels labels.Set) (*appsv1.DeploymentList, error)
}

// Interface assertion.
var _ ClientInterface = &Client{}

// Client is a kubernetes client that can talk to the API server.
type Client struct {
	kubernetes.Interface
	extInterface apiextensions.Interface
}

// NewClient creates a kubernetes client or bails out on on failures.
func NewClientFromConfig(kubeconfig string) ClientInterface {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		log.Infof("Loading kube client config from path %q", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		log.Infof("Using in-cluster kube client config")
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		log.Fatalf("Cannot load config for REST client: %v", err)
	}

	return &Client{kubernetes.NewForConfigOrDie(config), apiextensions.NewForConfigOrDie(config)}
}

// NewClient creates a kubernetes client
func NewClient(k8sClient kubernetes.Interface, extclient apiextensions.Interface) ClientInterface {
	return &Client{k8sClient, extclient}
}

// KubernetesInterface returns the Kubernetes interface.
func (c *Client) KubernetesInterface() kubernetes.Interface {
	return c.Interface
}

// ApiextensionsV1beta1Interface returns the API extention interface.
func (c *Client) ApiextensionsV1beta1Interface() apiextensions.Interface {
	return c.extInterface
}
