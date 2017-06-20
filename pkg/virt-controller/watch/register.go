package watch

import (
	"flag"
	"net/http"
	"reflect"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"strconv"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/dependencies"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var (
	CC            dependencies.ComponentCache
	launcherImage string = ""
	migratorImage string = ""
	Host          string
	Port          int
	registered    bool = false
)

type RateLimitingInterfaceStruct struct {
	workqueue.RateLimitingInterface
}
type IndexerStruct struct {
	cache.Indexer
}

func Register() {
	if registered {
		return
	}
	registered = true
	CC = dependencies.NewComponentCache()

	CC.Register(reflect.TypeOf((*http.Server)(nil)), createHttpServer)
	CC.Register(reflect.TypeOf((*StoreAndInformer)(nil)), createStoreAndInformer)
	CC.Register(reflect.TypeOf((*kubernetes.Clientset)(nil)), createClientSet)
	CC.Register(reflect.TypeOf((*MigrationController)(nil)), createMigrationController)
	CC.Register(reflect.TypeOf((*VMController)(nil)), createVMController)
	CC.Register(reflect.TypeOf((*rest.RESTClient)(nil)), createRestClient)
	CC.Register(reflect.TypeOf((*VMServiceStruct)(nil)), createVMService)
	CC.Register(reflect.TypeOf((*TemplateServiceStruct)(nil)), createTemplateService)
	CC.RegisterFactory(reflect.TypeOf((*IndexerStruct)(nil)), "migration", createCache)
	CC.RegisterFactory(reflect.TypeOf((*RateLimitingInterfaceStruct)(nil)), "migration", createQueue)
	CC.RegisterFactory(reflect.TypeOf((*cache.ListWatch)(nil)), "migration", createListWatch)

	CC.RegisterFactory(reflect.TypeOf((*IndexerStruct)(nil)), "vm", createCache)
	CC.RegisterFactory(reflect.TypeOf((*RateLimitingInterfaceStruct)(nil)), "vm", createQueue)
	CC.RegisterFactory(reflect.TypeOf((*cache.ListWatch)(nil)), "vm", createListWatch)

	flag.StringVar(&migratorImage, "migrator-image", "virt-handler", "Container which orchestrates a VM migration")
	flag.StringVar(&launcherImage, "launcher-image", "virt-launcher", "Shim container for containerized VMs")
	flag.StringVar(&Host, "listen", "0.0.0.0", "Address and Port where to listen on")
	flag.IntVar(&Port, "port", 8182, "Port to listen on")

	flag.Parse()
}

type VMServiceStruct struct {
	services.VMService
}

//TODO Wrap with a structure
func createVMService(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	return &VMServiceStruct{
		services.NewVMService(
			GetClientSet(CC),
			GetRestClient(CC),
			*GetTemplateService(CC))}, nil
}

func createRestClient(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return kubecli.GetRESTClient()
}

func createClientSet(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return kubecli.Get()
}

type TemplateServiceStruct struct {
	services.TemplateService
}

func createTemplateService(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	ts, err := services.NewTemplateService(launcherImage, migratorImage)
	return &TemplateServiceStruct{
		ts,
	}, err
}

func createVMController(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return NewVMController(*GetVMService(cc), nil, GetRestClient(cc), GetClientSet(cc)), nil
}

func createMigrationController(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return NewMigrationController(GetVMService(cc), GetRestClient(cc), GetClientSet(cc)), nil
}

func createCache(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	migrationCache := &IndexerStruct{cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)}
	return migrationCache, nil
}

func createQueue(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	migrationQueue := RateLimitingInterfaceStruct{workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())}
	return &migrationQueue, nil
}

func createListWatch(cc dependencies.ComponentCache, which string) (interface{}, error) {
	return cache.NewListWatchFromClient(GetRestClient(cc), which, k8sv1.NamespaceDefault, fields.Everything()), nil
}

type StoreAndInformer struct {
	Store    cache.Indexer
	Informer cache.Controller
}

func createStoreAndInformer(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	lw := GetListWatch(cc, "migration")
	queue := GetQueue(cc, "migration")
	store, informer := cache.NewIndexerInformer(lw, &kubev1.Migration{}, 0, kubecli.NewResourceEventHandlerFuncsForWorkqueue(queue), cache.Indexers{})
	return StoreAndInformer{
		store,
		informer,
	}, nil

}

func createHttpServer(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	logger := logging.DefaultLogger()
	httpLogger := logger.With("service", "http")
	httpLogger.Info().Log("action", "listening", "interface", Host, "port", Port)
	Address := Host + ":" + strconv.Itoa(Port)
	server := &http.Server{Addr: Address, Handler: nil}
	return server, nil
}

// Accessor functions below

func GetStoreAndInformer(cc dependencies.ComponentCache) *StoreAndInformer {
	val, ok := cc.FetchComponent(reflect.TypeOf((*StoreAndInformer)(nil)), "migration").(*StoreAndInformer)
	if !ok {
		panic(val)
	}
	return val
}

func GetListWatch(cc dependencies.ComponentCache, which string) *cache.ListWatch {
	val, ok := cc.FetchComponent(reflect.TypeOf((*cache.ListWatch)(nil)), which).(*cache.ListWatch)
	if !ok {
		panic(val)
	}
	return val

}

func GetQueue(cc dependencies.ComponentCache, which string) *RateLimitingInterfaceStruct {
	val, ok := cc.FetchComponent(reflect.TypeOf((*RateLimitingInterfaceStruct)(nil)), which).(*RateLimitingInterfaceStruct)
	if !ok {
		panic(val)
	}
	return val
}

func GetCache(cc dependencies.ComponentCache, which string) *IndexerStruct {
	val, ok := cc.FetchComponent(reflect.TypeOf((*IndexerStruct)(nil)), which).(*IndexerStruct)
	if !ok {
		panic(val)
	}
	return val
}

func GetClientSet(cc dependencies.ComponentCache) *kubernetes.Clientset {
	val, ok := cc.Fetch(reflect.TypeOf((*kubernetes.Clientset)(nil))).(*kubernetes.Clientset)
	if !ok {
		panic(val)
	}
	return val
}

func GetRestClient(cc dependencies.ComponentCache) *rest.RESTClient {
	t, ok := cc.Fetch(reflect.TypeOf((*rest.RESTClient)(nil))).(*rest.RESTClient)
	if !ok {
		panic(t)
	}
	return t
}

func GetTemplateService(cc dependencies.ComponentCache) *TemplateServiceStruct {
	return CC.Fetch(reflect.TypeOf((*TemplateServiceStruct)(nil))).(*TemplateServiceStruct)
}

func GetVMService(cc dependencies.ComponentCache) *VMServiceStruct {
	return cc.Fetch(reflect.TypeOf((*VMServiceStruct)(nil))).(*VMServiceStruct)
}

func GetVMController(cc dependencies.ComponentCache) *VMController {
	return (cc.Fetch(reflect.TypeOf((*VMController)(nil)))).(*VMController)
}

func GetMigrationController(cc dependencies.ComponentCache) *MigrationController {
	return cc.Fetch(reflect.TypeOf((*MigrationController)(nil))).(*MigrationController)
}

func GetHttpServer(cc dependencies.ComponentCache) *http.Server {
	return cc.Fetch(reflect.TypeOf((*http.Server)(nil))).(*http.Server)
}
