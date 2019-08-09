package main

import (
	"context"
	"crypto/rsa"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	crdv1alpha1 "github.com/kubernetes-csi/external-snapshotter/pkg/apis/volumesnapshot/v1alpha1"
	route1client "github.com/openshift/client-go/route/clientset/versioned"
	routeinformers "github.com/openshift/client-go/route/informers/externalversions"
	"github.com/pkg/errors"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	crdinformers "k8s.io/apiextensions-apiserver/pkg/client/informers/externalversions"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"
	csiclientset "kubevirt.io/containerized-data-importer/pkg/snapshot-client/clientset/versioned"
	csiinformers "kubevirt.io/containerized-data-importer/pkg/snapshot-client/informers/externalversions"
)

const (
	readyFile = "/tmp/ready"
)

var (
	configPath             string
	masterURL              string
	importerImage          string
	clonerImage            string
	uploadServerImage      string
	uploadProxyServiceName string
	configName             string
	pullPolicy             string
	verbose                string
)

// The importer and cloner images are obtained here along with the supported flags. IMPORTER_IMAGE, CLONER_IMAGE, and UPLOADSERVICE_IMAGE
// are required by the controller and will cause it to fail if not defined.
// Note: kubeconfig hierarchy is 1) -kubeconfig flag, 2) $KUBECONFIG exported var. If neither is
//   specified we do an in-cluster config. For testing it's easiest to export KUBECONFIG.
func init() {
	// flags
	flag.StringVar(&configPath, "kubeconfig", os.Getenv("KUBECONFIG"), "(Optional) Overrides $KUBECONFIG")
	flag.StringVar(&masterURL, "server", "", "(Optional) URL address of a remote api server.  Do not set for local clusters.")
	klog.InitFlags(nil)
	flag.Parse()

	importerImage = getRequiredEnvVar("IMPORTER_IMAGE")
	clonerImage = getRequiredEnvVar("CLONER_IMAGE")
	uploadServerImage = getRequiredEnvVar("UPLOADSERVER_IMAGE")
	uploadProxyServiceName = getRequiredEnvVar("UPLOADPROXY_SERVICE")

	pullPolicy = common.DefaultPullPolicy
	if pp := os.Getenv(common.PullPolicy); len(pp) != 0 {
		pullPolicy = pp
	}
	configName = common.ConfigName

	// NOTE we used to have a constant here and we're now just passing in the level directly
	// that should be fine since it was a constant and not a mutable variable
	defVerbose := fmt.Sprintf("%d", 1) // note flag values are strings
	verbose = defVerbose
	// visit actual flags passed in and if passed check -v and set verbose
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "v" {
			verbose = f.Value.String()
		}
	})
	if verbose == defVerbose {
		klog.V(1).Infof("Note: increase the -v level in the controller deployment for more detailed logging, eg. -v=%d or -v=%d\n", 2, 3)
	}

	klog.V(3).Infof("init: complete: cdi controller will create importer using image %q\n", importerImage)
}

func getRequiredEnvVar(name string) string {
	val := os.Getenv(name)
	if val == "" {
		klog.Fatalf("Environment Variable %q undefined\n", name)
	}
	return val
}

func start(cfg *rest.Config, stopCh <-chan struct{}) {
	klog.Info("Starting CDI controller components")

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Unable to get kube client: %v\n", errors.WithStack(err))
	}

	// Create an OpenShift route/v1 client.

	openshiftClient, err := route1client.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Unable to get openshift client: %v\n", errors.WithStack(err))
	}

	cdiClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
	}

	csiClient, err := csiclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building csi clientset: %s", err.Error())
	}

	extClient, err := extclientset.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building extClient: %s", err.Error())
	}

	cdiInformerFactory := informers.NewSharedInformerFactory(cdiClient, common.DefaultResyncPeriod)
	csiInformerFactory := csiinformers.NewFilteredSharedInformerFactory(csiClient, common.DefaultResyncPeriod, "", func(options *v1.ListOptions) {
		options.LabelSelector = common.CDILabelSelector
	})
	pvcInformerFactory := k8sinformers.NewSharedInformerFactory(client, common.DefaultResyncPeriod)
	podInformerFactory := k8sinformers.NewFilteredSharedInformerFactory(client, common.DefaultResyncPeriod, "", func(options *v1.ListOptions) {
		options.LabelSelector = common.CDILabelSelector
	})
	serviceInformerFactory := k8sinformers.NewFilteredSharedInformerFactory(client, common.DefaultResyncPeriod, "", func(options *v1.ListOptions) {
		options.LabelSelector = common.CDILabelSelector
	})
	ingressInformerFactory := k8sinformers.NewSharedInformerFactory(client, common.DefaultResyncPeriod)
	routeInformerFactory := routeinformers.NewSharedInformerFactory(openshiftClient, common.DefaultResyncPeriod)
	crdInformerFactory := crdinformers.NewSharedInformerFactory(extClient, common.DefaultResyncPeriod)

	pvcInformer := pvcInformerFactory.Core().V1().PersistentVolumeClaims()
	podInformer := podInformerFactory.Core().V1().Pods()
	serviceInformer := serviceInformerFactory.Core().V1().Services()
	ingressInformer := ingressInformerFactory.Extensions().V1beta1().Ingresses()
	routeInformer := routeInformerFactory.Route().V1().Routes()
	dataVolumeInformer := cdiInformerFactory.Cdi().V1alpha1().DataVolumes()
	configInformer := cdiInformerFactory.Cdi().V1alpha1().CDIConfigs()
	snapshotInformer := csiInformerFactory.Snapshot().V1alpha1().VolumeSnapshots()
	crdInformer := crdInformerFactory.Apiextensions().V1beta1().CustomResourceDefinitions().Informer()

	dataVolumeController := controller.NewDataVolumeController(
		client,
		cdiClient,
		csiClient,
		extClient,
		pvcInformer,
		dataVolumeInformer)

	importController := controller.NewImportController(
		client,
		cdiClient,
		pvcInformer,
		podInformer,
		importerImage,
		pullPolicy,
		verbose)

	cloneController := controller.NewCloneController(client,
		pvcInformer,
		podInformer,
		clonerImage,
		pullPolicy,
		verbose,
		getAPIServerPublicKey())

	smartCloneController := controller.NewSmartCloneController(client,
		cdiClient,
		csiClient,
		pvcInformer,
		snapshotInformer,
		dataVolumeInformer)

	uploadController := controller.NewUploadController(
		client,
		cdiClient,
		pvcInformer,
		podInformer,
		serviceInformer,
		uploadServerImage,
		uploadProxyServiceName,
		pullPolicy,
		verbose)

	configController := controller.NewConfigController(client,
		cdiClient,
		ingressInformer,
		routeInformer,
		configInformer,
		uploadProxyServiceName,
		configName,
		pullPolicy,
		verbose)

	klog.V(1).Infoln("created cdi controllers")

	err = uploadController.Init()
	if err != nil {
		klog.Fatalf("Error initializing upload controller: %+v", err)
	}

	err = configController.Init()
	if err != nil {
		klog.Fatalf("Error initializing config controller: %+v", err)
	}

	go cdiInformerFactory.Start(stopCh)
	go pvcInformerFactory.Start(stopCh)
	go podInformerFactory.Start(stopCh)
	go serviceInformerFactory.Start(stopCh)
	go ingressInformerFactory.Start(stopCh)
	go crdInformerFactory.Start(stopCh)
	if isOpenshift := controller.IsOpenshift(client); isOpenshift {
		go routeInformerFactory.Start(stopCh)
	}

	addCrdInformerEventHandlers(crdInformer, extClient, csiInformerFactory, smartCloneController, stopCh)

	klog.V(1).Infoln("started informers")

	go func() {
		err = dataVolumeController.Run(3, stopCh)
		if err != nil {
			klog.Fatalf("Error running dataVolume controller: %+v", err)
		}
	}()

	go func() {
		err = importController.Run(1, stopCh)
		if err != nil {
			klog.Fatalf("Error running import controller: %+v", err)
		}
	}()

	go func() {
		err = cloneController.Run(1, stopCh)
		if err != nil {
			klog.Fatalf("Error running clone controller: %+v", err)
		}
	}()

	go func() {
		err = uploadController.Run(1, stopCh)
		if err != nil {
			klog.Fatalf("Error running upload controller: %+v", err)
		}
	}()

	go func() {
		err = configController.Run(1, stopCh)
		if err != nil {
			klog.Fatalf("Error running config controller: %+v", err)
		}
	}()

	startSmartController(extClient, csiInformerFactory, smartCloneController, stopCh)
}

func main() {
	defer klog.Flush()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, configPath)
	if err != nil {
		klog.Fatalf("Unable to get kube config: %v\n", errors.WithStack(err))
	}

	stopCh := handleSignals()

	err = startLeaderElection(context.TODO(), cfg, func() {
		start(cfg, stopCh)
	})

	if err != nil {
		klog.Fatalf("Unable to start leader election: %v\n", errors.WithStack(err))
	}

	if err = createReadyFile(); err != nil {
		klog.Fatalf("Error creating ready file: %+v", err)
	}

	<-stopCh

	deleteReadyFile()

	klog.V(2).Infoln("cdi controller exited")
}

func createReadyFile() error {
	f, err := os.Create(readyFile)
	if err != nil {
		return err
	}
	defer f.Close()
	return nil
}

func deleteReadyFile() {
	os.Remove(readyFile)
}

// Shutdown gracefully on system signals
func handleSignals() <-chan struct{} {
	sigCh := make(chan os.Signal)
	stopCh := make(chan struct{})
	go func() {
		signal.Notify(sigCh)
		<-sigCh
		close(stopCh)
		os.Exit(1)
	}()
	return stopCh
}

func addCrdInformerEventHandlers(crdInformer cache.SharedIndexInformer, extClient extclientset.Interface,
	csiInformerFactory csiinformers.SharedInformerFactory, smartCloneController *controller.SmartCloneController,
	stopCh <-chan struct{}) {
	crdInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			crd := obj.(*v1beta1.CustomResourceDefinition)
			crdName := crd.Name

			vsClass := crdv1alpha1.VolumeSnapshotClassResourcePlural + "." + crdv1alpha1.GroupName
			vsContent := crdv1alpha1.VolumeSnapshotContentResourcePlural + "." + crdv1alpha1.GroupName
			vs := crdv1alpha1.VolumeSnapshotResourcePlural + "." + crdv1alpha1.GroupName

			switch crdName {
			case vsClass:
				fallthrough
			case vsContent:
				fallthrough
			case vs:
				startSmartController(extClient, csiInformerFactory, smartCloneController, stopCh)
			}
		},
	})
}

func startSmartController(extclient extclientset.Interface, csiInformerFactory csiinformers.SharedInformerFactory,
	smartCloneController *controller.SmartCloneController, stopCh <-chan struct{}) {
	if controller.IsCsiCrdsDeployed(extclient) {
		go csiInformerFactory.Start(stopCh)
		go func() {
			err := smartCloneController.Run(1, stopCh)
			if err != nil {
				klog.Fatalf("Error running smart clone controller: %+v", err)
			}
		}()
	}
}

func getAPIServerPublicKey() *rsa.PublicKey {
	keyBytes, err := ioutil.ReadFile(controller.APIServerPublicKeyPath)
	if err != nil {
		klog.Fatalf("Error reading apiserver public key")
	}

	key, err := controller.DecodePublicKey(keyBytes)
	if err != nil {
		klog.Fatalf("Error decoding public key")
	}

	return key
}
