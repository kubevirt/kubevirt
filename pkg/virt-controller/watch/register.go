package watch

import (
	"flag"
	"reflect"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	k8sv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/dependencies"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

var (
	CC            dependencies.ComponentCache
	launcherImage string = ""
	migratorImage string = ""
	registered    bool   = false
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
	CC.Register(reflect.TypeOf((*StoreAndInformer)(nil)), createStoreAndInformer)
	CC.Register(reflect.TypeOf((*kubernetes.Clientset)(nil)), createClientSet)
	CC.Register(reflect.TypeOf((*MigrationController)(nil)), createMigrationController)
	CC.Register(reflect.TypeOf((*VMController)(nil)), createVMController)
	CC.Register(reflect.TypeOf((*rest.RESTClient)(nil)), createRestClient)
	CC.Register(reflect.TypeOf((*services.VMService)(nil)), createVMService)
	CC.Register(reflect.TypeOf((*services.TemplateService)(nil)), createTemplateService)
	CC.RegisterFactory(reflect.TypeOf((*IndexerStruct)(nil)), "migration", createCache)
	CC.RegisterFactory(reflect.TypeOf((*RateLimitingInterfaceStruct)(nil)), "migration", createQueue)
	CC.RegisterFactory(reflect.TypeOf((*cache.ListWatch)(nil)), "migration", createListWatch)

	CC.RegisterFactory(reflect.TypeOf((*IndexerStruct)(nil)), "vm", createCache)
	CC.RegisterFactory(reflect.TypeOf((*RateLimitingInterfaceStruct)(nil)), "vm", createQueue)
	CC.RegisterFactory(reflect.TypeOf((*cache.ListWatch)(nil)), "vm", createListWatch)

	flag.StringVar(&migratorImage, "migrator-image", "virt-handler", "Container which orchestrates a VM migration")
	flag.StringVar(&launcherImage, "launcher-image", "virt-launcher", "Shim container for containerized VMs")
	flag.Parse()
}

//TODO Wrap with a structure
func createVMService(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	return services.NewVMService(
		GetClientSet(CC),
		GetRestClient(CC),
		*GetTemplateService(CC)), nil
}

func createRestClient(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return kubecli.GetRESTClient()
}

func createClientSet(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return kubecli.Get()
}

//TODO Wrap with a structure
func createTemplateService(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return services.NewTemplateService(launcherImage, migratorImage)
}

func createVMController(cc dependencies.ComponentCache, _ string) (interface{}, error) {
	return NewVMController(*GetVMService(cc), nil, GetRestClient(cc), GetClientSet(cc)), nil
}

func createMigrationController(cc dependencies.ComponentCache, _ string) (interface{}, error) {

	sni := GetStoreAndInformer(CC)

	return &MigrationController{
		restClient: GetRestClient(cc),
		vmService:  *GetVMService(cc),
		clientset:  GetClientSet(cc),
		queue:      GetQueue(cc, "migration").RateLimitingInterface,
		store:      sni.Store,
		informer:   sni.Informer,
	}, nil
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

func GetTemplateService(cc dependencies.ComponentCache) *services.TemplateService {
	return CC.Fetch(reflect.TypeOf((*services.TemplateService)(nil))).(*services.TemplateService)
}

func GetVMService(cc dependencies.ComponentCache) *services.VMService {
	return cc.Fetch(reflect.TypeOf((*services.VMService)(nil))).(*services.VMService)
}

func GetVMController(cc dependencies.ComponentCache) *VMController {
	return (cc.Fetch(reflect.TypeOf((*VMController)(nil)))).(*VMController)
}

func GetMigrationController(cc dependencies.ComponentCache) *MigrationController {
	return cc.Fetch(reflect.TypeOf((*MigrationController)(nil))).(*MigrationController)
}
