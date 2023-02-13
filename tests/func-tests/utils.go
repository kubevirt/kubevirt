package tests

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"
	kvtutil "kubevirt.io/kubevirt/tests/util"
)

var KubeVirtStorageClassLocal string

func init() {
	flag.StringVar(&KubeVirtStorageClassLocal, "storage-class-local", "local", "Storage provider to use for tests which want local storage")
}

// GetJobTypeEnvVar returns "JOB_TYPE" enviroment varibale
func GetJobTypeEnvVar() string {
	return (os.Getenv("JOB_TYPE"))
}

func FlagParse() {
	flag.Parse()
}

func BeforeEach() {
	virtClient, err := kubecli.GetKubevirtClient()
	kvtutil.PanicOnError(err)

	kvtutil.PanicOnError(virtClient.RestClient().Delete().Namespace(kvtutil.NamespaceTestDefault).Resource("virtualmachines").Do(context.TODO()).Error())
	kvtutil.PanicOnError(virtClient.RestClient().Delete().Namespace(kvtutil.NamespaceTestDefault).Resource("virtualmachineinstances").Do(context.TODO()).Error())
	kvtutil.PanicOnError(virtClient.CoreV1().RESTClient().Delete().Namespace(kvtutil.NamespaceTestDefault).Resource("persistentvolumeclaims").Do(context.TODO()).Error())
}

func SkipIfNotOpenShift(cli kubecli.KubevirtClient, testName string) {
	isOpenShift, err := IsOpenShift(cli)
	kvtutil.PanicOnError(err)

	if !isOpenShift {
		ginkgo.Skip(fmt.Sprintf("Skipping %s tests when the cluster is not OpenShift", testName))
	}
}

type cacheIsOpenShift struct {
	isOpenShift bool
	hasSet      bool
	lock        sync.Mutex
}

func (c *cacheIsOpenShift) IsOpenShift(cli kubecli.KubevirtClient) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.hasSet {
		return c.isOpenShift, nil
	}

	s := scheme.Scheme
	_ = openshiftconfigv1.Install(s)
	s.AddKnownTypes(openshiftconfigv1.GroupVersion)

	clusterVersion := &openshiftconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}

	err := cli.RestClient().Get().
		Resource("clusterversions").
		Name("version").
		AbsPath("/apis", openshiftconfigv1.GroupVersion.Group, openshiftconfigv1.GroupVersion.Version).
		Timeout(10 * time.Second).
		Do(context.TODO()).Into(clusterVersion)

	if err == nil {
		c.isOpenShift = true
		c.hasSet = true
		return c.isOpenShift, nil
	}

	if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
		c.isOpenShift = false
		c.hasSet = true
		return c.isOpenShift, nil
	}

	return false, err
}

var isOpenShiftCache cacheIsOpenShift

func IsOpenShift(cli kubecli.KubevirtClient) (bool, error) {
	return isOpenShiftCache.IsOpenShift(cli)
}
