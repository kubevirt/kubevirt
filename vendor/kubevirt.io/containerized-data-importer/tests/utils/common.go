package utils

import (
	"github.com/onsi/ginkgo"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// cdi-file-host pod/service relative values
const (
	//RegistryHostName provides a deploymnet and service name for registry
	RegistryHostName = "cdi-docker-registry-host"
	// FileHostName provides a deployment and service name for tests
	FileHostName = "cdi-file-host"
	// FileHostS3Bucket provides an S3 bucket name for tests (e.g. http://<serviceIP:port>/FileHostS3Bucket/image)
	FileHostS3Bucket = "images"
	// AccessKeyValue provides a username to use for http and S3 (see hack/build/docker/cdi-func-test-file-host-http/htpasswd)
	AccessKeyValue = "admin"
	// SecretKeyValue provides a password to use for http and S3 (see hack/build/docker/cdi-func-test-file-host-http/htpasswd)
	SecretKeyValue = "password"
	// HttpAuthPort provides a cdi-file-host service auth port for tests
	HTTPAuthPort = 81
	// HttpNoAuthPort provides a cdi-file-host service no-auth port for tests, requires AccessKeyValue and SecretKeyValue
	HTTPNoAuthPort = 80
	// HTTPRateLimitPort provides a cdi-file-host service rate limit port for tests, speed is limited to 25k/s to allow for testing slow connection behavior. No auth.
	HTTPRateLimitPort = 82
	// S3Port provides a cdi-file-host service S3 port, requires AccessKey and SecretKeyValue
	S3Port = 9000
	// HTTPSPort is the https port of cdi-file-host
	HTTPSNoAuthPort = 443
	// RegistryCertConfigMap is the ConfigMap where the cert for the docker registry is stored
	RegistryCertConfigMap = "cdi-docker-registry-host-certs"
	// FileHostCertConfigMap is the ConfigMap where the cert fir the file host is stored
	FileHostCertConfigMap = "cdi-file-host-certs"
)

var (
	// DefaultNodeName the default node to use in tests
	DefaultNodeName string
	// DefaultStorageClass the defauld storage class used in tests
	DefaultStorageClass *storagev1.StorageClass
)

func getDefaultNodeName(client *kubernetes.Clientset) string {
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("Unable to list nodes")
	}
	return nodes.Items[0].Name
}

func getDefaultStorageClass(client *kubernetes.Clientset) *storagev1.StorageClass {
	storageclasses, err := client.StorageV1().StorageClasses().List(metav1.ListOptions{})
	if err != nil {
		ginkgo.Fail("Unable to list storage classes")
		return nil
	}
	for _, storageClass := range storageclasses.Items {
		if storageClass.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return &storageClass
		}
	}
	ginkgo.Fail("Unable to find default storage classes")
	return nil
}

// IsHostpathProvisioner returns true if hostpath-provisioner is the default storage class
func IsHostpathProvisioner() bool {
	if DefaultStorageClass == nil {
		return false
	}
	return DefaultStorageClass.Provisioner == "kubevirt.io/hostpath-provisioner"
}

// AddProvisionOnNodeToAnn adds 'kubevirt.io/provisionOnNode' annotaion
func AddProvisionOnNodeToAnn(pvcAnn map[string]string) {
	pvcAnn["kubevirt.io/provisionOnNode"] = DefaultNodeName
}

// CacheTestsData fetch and cache data required for tests
func CacheTestsData(client *kubernetes.Clientset) {
	DefaultNodeName = getDefaultNodeName(client)
	DefaultStorageClass = getDefaultStorageClass(client)
}
