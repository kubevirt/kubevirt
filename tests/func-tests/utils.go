package tests

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/onsi/ginkgo"
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

//GetJobTypeEnvVar returns "JOB_TYPE" enviroment varibale
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

func SkipIfNotOpenShift(cli kubecli.KubevirtClient) {
	isOpenShift, err := IsOpenShift(cli)
	kvtutil.PanicOnError(err)

	if !isOpenShift {
		ginkgo.Skip("Skipping Prometheus tests when the cluster is not OpenShift")
	}
}

func IsOpenShift(cli kubecli.KubevirtClient) (bool, error) {
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
		return true, nil
	} else if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
		return false, nil
	} else {
		return false, err
	}

}
