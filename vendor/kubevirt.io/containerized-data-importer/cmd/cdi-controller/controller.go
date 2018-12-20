package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/golang/glog"

	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"

	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	configPath             string
	masterURL              string
	importerImage          string
	clonerImage            string
	uploadServerImage      string
	uploadProxyServiceName string
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
	flag.Parse()

	importerImage = getRequiredEnvVar("IMPORTER_IMAGE")
	clonerImage = getRequiredEnvVar("CLONER_IMAGE")
	uploadServerImage = getRequiredEnvVar("UPLOADSERVER_IMAGE")
	uploadProxyServiceName = getRequiredEnvVar("UPLOADPROXY_SERVICE")

	pullPolicy = common.DefaultPullPolicy
	if pp := os.Getenv(common.PullPolicy); len(pp) != 0 {
		pullPolicy = pp
	}

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
		glog.V(1).Infof("Note: increase the -v level in the controller deployment for more detailed logging, eg. -v=%d or -v=%d\n", 2, 3)
	}

	glog.V(3).Infof("init: complete: cdi controller will create importer using image %q\n", importerImage)
}

func getRequiredEnvVar(name string) string {
	val := os.Getenv(name)
	if val == "" {
		glog.Fatalf("Environment Variable %q undefined\n", name)
	}
	return val
}

func start(cfg *rest.Config, stopCh <-chan struct{}) {
	glog.Info("Starting CDI controller components")

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to get kube client: %v\n", errors.WithStack(err))
	}

	cdiClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	cdiInformerFactory := informers.NewSharedInformerFactory(cdiClient, common.DefaultResyncPeriod)
	pvcInformerFactory := k8sinformers.NewSharedInformerFactory(client, common.DefaultResyncPeriod)
	podInformerFactory := k8sinformers.NewFilteredSharedInformerFactory(client, common.DefaultResyncPeriod, "", func(options *v1.ListOptions) {
		options.LabelSelector = common.CDILabelSelector
	})
	serviceInformerFactory := k8sinformers.NewFilteredSharedInformerFactory(client, common.DefaultResyncPeriod, "", func(options *v1.ListOptions) {
		options.LabelSelector = common.CDILabelSelector
	})

	pvcInformer := pvcInformerFactory.Core().V1().PersistentVolumeClaims()
	podInformer := podInformerFactory.Core().V1().Pods()
	serviceInformer := serviceInformerFactory.Core().V1().Services()
	dataVolumeInformer := cdiInformerFactory.Cdi().V1alpha1().DataVolumes()

	dataVolumeController := controller.NewDataVolumeController(
		client,
		cdiClient,
		pvcInformer,
		dataVolumeInformer)

	importController := controller.NewImportController(client,
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
		verbose)

	uploadController := controller.NewUploadController(client,
		pvcInformer,
		podInformer,
		serviceInformer,
		uploadServerImage,
		uploadProxyServiceName,
		pullPolicy,
		verbose)

	glog.V(1).Infoln("created cdi controllers")

	go cdiInformerFactory.Start(stopCh)
	go pvcInformerFactory.Start(stopCh)
	go podInformerFactory.Start(stopCh)
	go serviceInformerFactory.Start(stopCh)

	glog.V(1).Infoln("started informers")

	go func() {
		err = dataVolumeController.Run(3, stopCh)
		if err != nil {
			glog.Fatalln("Error running dataVolume controller: %+v", err)
		}
	}()

	go func() {
		err = importController.Run(1, stopCh)
		if err != nil {
			glog.Fatalln("Error running import controller: %+v", err)
		}
	}()

	go func() {
		err = cloneController.Run(1, stopCh)
		if err != nil {
			glog.Fatalln("Error running clone controller: %+v", err)
		}
	}()

	go func() {
		err = uploadController.Run(1, stopCh)
		if err != nil {
			glog.Fatalln("Error running upload controller: %+v", err)
		}
	}()
}

func main() {
	defer glog.Flush()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, configPath)
	if err != nil {
		glog.Fatalf("Unable to get kube config: %v\n", errors.WithStack(err))
	}

	stopCh := handleSignals()

	err = startLeaderElection(cfg, func() {
		start(cfg, stopCh)
	})

	if err != nil {
		glog.Fatalf("Unable to start leader election: %v\n", errors.WithStack(err))
	}

	<-stopCh
	glog.V(2).Infoln("cdi controller exited")
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
