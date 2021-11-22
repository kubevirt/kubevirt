package tests

import (
	"context"
	"flag"
	"os"

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
