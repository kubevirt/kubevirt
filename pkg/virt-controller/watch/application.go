package watch

import (
	"flag"
	golog "log"
	"net/http"
	"os"
	"strconv"

	"github.com/emicklei/go-restful"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

type VirtControllerApp struct {
	clientSet       kubecli.KubevirtClient
	templateService services.TemplateService
	restClient      *clientrest.RESTClient
	vmService       services.VMService
	informerFactory controller.KubeInformerFactory
	podInformer     cache.SharedIndexInformer

	migrationCache      cache.Store
	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer
	migrationQueue      workqueue.RateLimitingInterface
	migrationRecorder   record.EventRecorder

	vmCache      cache.Store
	vmController *VMController
	vmInformer   cache.SharedIndexInformer
	vmQueue      workqueue.RateLimitingInterface

	rsController *VMReplicaSet
	rsInformer   cache.SharedIndexInformer

	LeaderElection leaderelectionconfig.Configuration

	host             string
	port             int
	launcherImage    string
	migratorImage    string
	virtShareDir     string
	ephemeralDiskDir string
	readyChan        chan bool
}

var _ service.Service = &VirtControllerApp{}

func Execute() {
	var err error
	var app VirtControllerApp = VirtControllerApp{}

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	app.DefineFlags()

	app.readyChan = make(chan bool, 1)

	log.InitializeLogging("virt-controller")

	app.clientSet, err = kubecli.GetKubevirtClient()

	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	webService := rest.WebService
	webService.Route(webService.GET("/leader").To(app.readinessProbe).Doc("Leader endpoint"))
	restful.Add(webService)

	// Bootstrapping. From here on the initialization order is important

	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet)

	app.vmInformer = app.informerFactory.VM()
	app.migrationInformer = app.informerFactory.Migration()
	app.podInformer = app.informerFactory.KubeVirtPod()

	app.vmQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	app.vmCache = app.vmInformer.GetStore()
	app.vmInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(app.vmQueue))
	app.podInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForFunc(vmLabelHandler(app.vmQueue)))

	app.migrationQueue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	app.migrationInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForWorkqueue(app.migrationQueue))
	app.podInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForFunc(migrationJobLabelHandler(app.migrationQueue)))
	app.podInformer.AddEventHandler(controller.NewResourceEventHandlerFuncsForFunc(migrationPodLabelHandler(app.migrationQueue)))
	app.migrationCache = app.migrationInformer.GetStore()

	app.migrationRecorder = app.getNewRecorder(k8sv1.NamespaceAll, "virt-migration-controller")

	app.rsInformer = app.informerFactory.VMReplicaSet()

	app.initCommon()
	app.initReplicaSet()
	app.Run()
}
func (vca *VirtControllerApp) Run() {
	logger := log.Log
	stop := make(chan struct{})
	defer close(stop)
	vca.informerFactory.Start(stop)
	go func() {
		httpLogger := logger.With("service", "http")
		httpLogger.Level(log.INFO).Log("action", "listening", "interface", vca.host, "port", vca.port)
		if err := http.ListenAndServe(vca.host+":"+strconv.Itoa(vca.port), nil); err != nil {
			golog.Fatal(err)
		}
	}()

	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, leaderelectionconfig.DefaultEndpointName)

	id, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}

	rl, err := resourcelock.New(vca.LeaderElection.ResourceLock,
		leaderelectionconfig.DefaultNamespace,
		leaderelectionconfig.DefaultEndpointName,
		vca.clientSet.CoreV1(),
		resourcelock.ResourceLockConfig{
			Identity:      id,
			EventRecorder: recorder,
		})
	if err != nil {
		golog.Fatal(err)
	}

	leaderElector, err := leaderelection.NewLeaderElector(
		leaderelection.LeaderElectionConfig{
			Lock:          rl,
			LeaseDuration: vca.LeaderElection.LeaseDuration.Duration,
			RenewDeadline: vca.LeaderElection.RenewDeadline.Duration,
			RetryPeriod:   vca.LeaderElection.RetryPeriod.Duration,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(stopCh <-chan struct{}) {
					go vca.vmController.Run(3, stop)
					go vca.migrationController.Run(3, stop)
					go vca.rsController.Run(3, stop)
					close(vca.readyChan)
				},
				OnStoppedLeading: func() {
					golog.Fatal("leaderelection lost")
				},
			},
		})
	if err != nil {
		golog.Fatal(err)
	}

	leaderElector.Run()
	panic("unreachable")
}

func (vca *VirtControllerApp) getNewRecorder(namespace string, componentName string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: vca.clientSet.CoreV1().Events(namespace)})
	return eventBroadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: componentName})
}

func (vca *VirtControllerApp) initCommon() {
	var err error

	err = registrydisk.SetLocalDirectory(vca.ephemeralDiskDir + "/registry-disk-data")
	if err != nil {
		golog.Fatal(err)
	}
	vca.templateService, err = services.NewTemplateService(vca.launcherImage, vca.migratorImage, vca.virtShareDir)
	if err != nil {
		golog.Fatal(err)
	}
	vca.vmService = services.NewVMService(vca.clientSet, vca.restClient, vca.templateService)
	vca.vmController = NewVMController(vca.restClient, vca.vmService, vca.vmQueue, vca.vmCache, vca.vmInformer, vca.podInformer, nil, vca.clientSet)
	vca.migrationController = NewMigrationController(vca.restClient, vca.vmService, vca.clientSet, vca.migrationQueue, vca.migrationInformer, vca.podInformer, vca.migrationCache, vca.migrationRecorder)
}

func (vca *VirtControllerApp) initReplicaSet() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "virtualmachinereplicaset-controller")
	vca.rsController = NewVMReplicaSet(vca.vmInformer, vca.rsInformer, recorder, vca.clientSet, controller.BurstReplicas)
}

func (vca *VirtControllerApp) readinessProbe(_ *restful.Request, response *restful.Response) {
	res := map[string]interface{}{}

	select {
	case _, opened := <-vca.readyChan:
		if !opened {
			res["apiserver"] = map[string]interface{}{"leader": "true"}
			response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
			return
		}
	default:
	}
	res["apiserver"] = map[string]interface{}{"leader": "false", "error": "current pod is not leader"}
	response.WriteHeaderAndJson(http.StatusServiceUnavailable, res, restful.MIME_JSON)
}

func (vca *VirtControllerApp) DefineFlags() {
	flag.StringVar(&vca.host, "listen", "0.0.0.0", "Address and port where to listen on")
	flag.IntVar(&vca.port, "port", 8182, "Port to listen on")
	flag.StringVar(&vca.launcherImage, "launcher-image", "virt-launcher", "Shim container for containerized VMs")
	flag.StringVar(&vca.migratorImage, "migrator-image", "virt-handler", "Container which orchestrates a VM migration")
	flag.StringVar(&vca.virtShareDir, "kubevirt-share-dir", "/var/run/kubevirt", "Shared directory between virt-handler and virt-launcher")
	flag.StringVar(&vca.ephemeralDiskDir, "ephemeral-disk-dir", "/var/run/libvirt/kubevirt-ephemeral-disk", "Base direcetory for ephemeral disk data")
	leaderelectionconfig.BindFlags(&vca.LeaderElection)
	flag.Parse()
}
