package watch

import (
	"reflect"

	"github.com/onsi/gomega/ghttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/dependencies"
	"kubevirt.io/kubevirt/pkg/kubecli"
)

func CreateTestServer(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return ghttp.NewServer(), nil
}

func GetTestServer(cc dependencies.ComponentCache) *ghttp.Server {
	return cc.Fetch(reflect.TypeOf((*ghttp.Server)(nil))).(*ghttp.Server)
}

func CreateTestClientSet(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	server := GetTestServer(cc)
	config := rest.Config{}
	config.Host = server.URL()
	return kubernetes.NewForConfig(&config)
}

func CreateTestRestClient(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	server := GetTestServer(cc)
	return kubecli.GetRESTClientFromFlags(server.URL(), "")
}

func createTestMigrationController(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	return &MigrationController{
		restClient: GetRestClient(cc),
		vmService:  *GetVMService(cc),
		clientset:  GetClientSet(cc),
		queue:      GetQueue(cc, "migration").RateLimitingInterface,
		store:      GetCache(cc, "migration").Indexer,
		informer:   nil,
	}, nil
}

func createTestVMController(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return &VMController{
		restClient: GetRestClient(cc),
		vmService:  *GetVMService(cc),
		queue:      GetQueue(cc, "vm").RateLimitingInterface,
		store:      GetCache(cc, "vm").Indexer,
	}, nil
}

func RegisterTestObjects() {
	CC.Register(reflect.TypeOf((*ghttp.Server)(nil)), CreateTestServer)
}
