/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package watch

import (
	"io/ioutil"
	golog "log"
	"net/http"
	"os"

	restful "github.com/emicklei/go-restful"
	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clientrest "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/certificates"
	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
)

const (
	defaultPort = 8182

	defaultHost = "0.0.0.0"

	launcherImage = "virt-launcher"

	imagePullSecret = ""

	virtShareDir = "/var/run/kubevirt"

	ephemeralDiskDir = virtShareDir + "-ephemeral-disks"

	controllerThreads = 3
)

type VirtControllerApp struct {
	service.ServiceListen

	clientSet       kubecli.KubevirtClient
	templateService services.TemplateService
	restClient      *clientrest.RESTClient
	informerFactory controller.KubeInformerFactory
	podInformer     cache.SharedIndexInformer

	nodeInformer   cache.SharedIndexInformer
	nodeController *NodeController

	vmiCache      cache.Store
	vmiController *VMIController
	vmiInformer   cache.SharedIndexInformer
	vmiRecorder   record.EventRecorder

	configMapCache    cache.Store
	configMapInformer cache.SharedIndexInformer

	persistentVolumeClaimCache    cache.Store
	persistentVolumeClaimInformer cache.SharedIndexInformer

	rsController *VMIReplicaSet
	rsInformer   cache.SharedIndexInformer

	vmController *VMController
	vmInformer   cache.SharedIndexInformer

	dataVolumeInformer cache.SharedIndexInformer

	migrationController *MigrationController
	migrationInformer   cache.SharedIndexInformer

	LeaderElection leaderelectionconfig.Configuration

	launcherImage     string
	imagePullSecret   string
	virtShareDir      string
	ephemeralDiskDir  string
	readyChan         chan bool
	kubevirtNamespace string
}

var _ service.Service = &VirtControllerApp{}

func Execute() {
	var err error
	var app VirtControllerApp = VirtControllerApp{}

	app.LeaderElection = leaderelectionconfig.DefaultLeaderElectionConfiguration()

	service.Setup(&app)

	virtconfig.Init()

	app.readyChan = make(chan bool, 1)

	log.InitializeLogging("virt-controller")

	app.clientSet, err = kubecli.GetKubevirtClient()

	if err != nil {
		golog.Fatal(err)
	}

	app.restClient = app.clientSet.RestClient()

	webService := rest.WebService
	webService.Route(webService.GET("/leader").To(app.leaderProbe).Doc("Leader endpoint"))
	restful.Add(webService)

	// Bootstrapping. From here on the initialization order is important
	app.kubevirtNamespace, err = util.GetNamespace()
	if err != nil {
		golog.Fatalf("Error searching for namespace: %v", err)
	}
	app.informerFactory = controller.NewKubeInformerFactory(app.restClient, app.clientSet, app.kubevirtNamespace)

	app.vmiInformer = app.informerFactory.VMI()
	app.podInformer = app.informerFactory.KubeVirtPod()
	app.nodeInformer = app.informerFactory.KubeVirtNode()

	app.vmiCache = app.vmiInformer.GetStore()
	app.vmiRecorder = app.getNewRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	app.rsInformer = app.informerFactory.VMIReplicaSet()

	app.configMapInformer = app.informerFactory.ConfigMap()
	app.configMapCache = app.configMapInformer.GetStore()

	app.persistentVolumeClaimInformer = app.informerFactory.PersistentVolumeClaim()
	app.persistentVolumeClaimCache = app.persistentVolumeClaimInformer.GetStore()

	app.vmInformer = app.informerFactory.VirtualMachine()

	app.migrationInformer = app.informerFactory.VirtualMachineInstanceMigration()

	if virtconfig.DataVolumesEnabled() {
		app.dataVolumeInformer = app.informerFactory.DataVolume()
		log.Log.Infof("DataVolume integration enabled")
	} else {
		// Add a dummy DataVolume informer in the event datavolume support
		// is disabled. This lets the controller continue to work without
		// requiring a separate branching code path.
		app.dataVolumeInformer = app.informerFactory.DummyDataVolume()
		log.Log.Infof("DataVolume integration disabled")
	}

	app.initCommon()
	app.initReplicaSet()
	app.initVirtualMachines()
	app.Run()
}

func (vca *VirtControllerApp) Run() {
	logger := log.Log

	certsDirectory, err := ioutil.TempDir("", "certsdir")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(certsDirectory)

	certStore, err := certificates.GenerateSelfSignedCert(certsDirectory, "virt-controller", vca.kubevirtNamespace)
	if err != nil {
		glog.Fatalf("unable to generate certificates: %v", err)
	}
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		httpLogger := logger.With("service", "http")
		httpLogger.Level(log.INFO).Log("action", "listening", "interface", vca.BindAddress, "port", vca.Port)
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServeTLS(vca.Address(), certStore.CurrentPath(), certStore.CurrentPath(), nil); err != nil {
			golog.Fatal(err)
		}
	}()

	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, leaderelectionconfig.DefaultEndpointName)

	id, err := os.Hostname()
	if err != nil {
		golog.Fatalf("unable to get hostname: %v", err)
	}

	rl, err := resourcelock.New(vca.LeaderElection.ResourceLock,
		vca.kubevirtNamespace,
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
					vca.informerFactory.Start(stop)
					go vca.nodeController.Run(controllerThreads, stop)
					go vca.vmiController.Run(controllerThreads, stop)
					go vca.rsController.Run(controllerThreads, stop)
					go vca.vmController.Run(controllerThreads, stop)
					go vca.migrationController.Run(controllerThreads, stop)
					cache.WaitForCacheSync(stopCh, vca.persistentVolumeClaimInformer.HasSynced)
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

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		golog.Fatal(err)
	}

	containerdisk.SetLocalDirectory(vca.ephemeralDiskDir + "/container-disk-data")
	vca.templateService = services.NewTemplateService(vca.launcherImage,
		vca.virtShareDir,
		vca.ephemeralDiskDir,
		vca.imagePullSecret,
		vca.configMapCache,
		vca.persistentVolumeClaimCache,
		virtClient)

	vca.vmiController = NewVMIController(vca.templateService, vca.vmiInformer, vca.podInformer, vca.vmiRecorder, vca.clientSet, vca.configMapInformer, vca.dataVolumeInformer)
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "node-controller")
	vca.nodeController = NewNodeController(vca.clientSet, vca.nodeInformer, vca.vmiInformer, recorder)
	vca.migrationController = NewMigrationController(vca.templateService, vca.vmiInformer, vca.podInformer, vca.migrationInformer, vca.vmiRecorder, vca.clientSet)
}

func (vca *VirtControllerApp) initReplicaSet() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "virtualmachinereplicaset-controller")
	vca.rsController = NewVMIReplicaSet(vca.vmiInformer, vca.rsInformer, recorder, vca.clientSet, controller.BurstReplicas)
}

func (vca *VirtControllerApp) initVirtualMachines() {
	recorder := vca.getNewRecorder(k8sv1.NamespaceAll, "virtualmachine-controller")

	vca.vmController = NewVMController(
		vca.vmiInformer,
		vca.vmInformer,
		vca.dataVolumeInformer,
		recorder,
		vca.clientSet)
}

func (vca *VirtControllerApp) leaderProbe(_ *restful.Request, response *restful.Response) {
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
	res["apiserver"] = map[string]interface{}{"leader": "false"}
	response.WriteHeaderAndJson(http.StatusOK, res, restful.MIME_JSON)
}

func (vca *VirtControllerApp) AddFlags() {
	vca.InitFlags()

	leaderelectionconfig.BindFlags(&vca.LeaderElection)

	vca.BindAddress = defaultHost
	vca.Port = defaultPort

	vca.AddCommonFlags()

	flag.StringVar(&vca.launcherImage, "launcher-image", launcherImage,
		"Shim container for containerized VMIs")

	flag.StringVar(&vca.imagePullSecret, "image-pull-secret", imagePullSecret,
		"Secret to use for pulling virt-launcher and/or registry disks")

	flag.StringVar(&vca.virtShareDir, "kubevirt-share-dir", virtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.StringVar(&vca.ephemeralDiskDir, "ephemeral-disk-dir", ephemeralDiskDir,
		"Base directory for ephemeral disk data")
}
