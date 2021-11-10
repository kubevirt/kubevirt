package tests

import (
	"context"
	"flag"
	"os"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
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
	tests.PanicOnError(err)

	tests.PanicOnError(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachines").Do(context.TODO()).Error())
	tests.PanicOnError(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachineinstances").Do(context.TODO()).Error())
	tests.PanicOnError(virtClient.CoreV1().RESTClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("persistentvolumeclaims").Do(context.TODO()).Error())
}
