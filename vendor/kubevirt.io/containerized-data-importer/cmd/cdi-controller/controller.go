package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/golang/glog"
	. "kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/controller"

	clientset "kubevirt.io/containerized-data-importer/pkg/client/clientset/versioned"
	informers "kubevirt.io/containerized-data-importer/pkg/client/informers/externalversions"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	configPath    string
	masterURL     string
	importerImage string
	clonerImage   string
	pullPolicy    string
	verbose       string
)

// The importer and cloner images are obtained here along with the supported flags. IMPORTER_IMAGE and CLONER_IMAGE
// are required by the controller and will cause it to fail if not defined.
// Note: kubeconfig hierarchy is 1) -kubeconfig flag, 2) $KUBECONFIG exported var. If neither is
//   specified we do an in-cluster config. For testing it's easiest to export KUBECONFIG.
func init() {
	const IMPORTER_IMAGE = "IMPORTER_IMAGE"
	const CLONER_IMAGE = "CLONER_IMAGE"

	// flags
	flag.StringVar(&configPath, "kubeconfig", os.Getenv("KUBECONFIG"), "(Optional) Overrides $KUBECONFIG")
	flag.StringVar(&masterURL, "server", "", "(Optional) URL address of a remote api server.  Do not set for local clusters.")
	flag.Parse()

	// env variables
	importerImage = os.Getenv(IMPORTER_IMAGE)
	if importerImage == "" {
		glog.Fatalf("Environment Variable %q undefined\n", IMPORTER_IMAGE)
	}

	clonerImage = os.Getenv(CLONER_IMAGE)
	if clonerImage == "" {
		glog.Fatalf("Environment Variable %q undefined\n", CLONER_IMAGE)
	}

	pullPolicy = DEFAULT_PULL_POLICY
	if pp := os.Getenv(PULL_POLICY); len(pp) != 0 {
		pullPolicy = pp
	}

	// get the verbose level so it can be passed to the importer pod
	defVerbose := fmt.Sprintf("%d", DEFAULT_VERBOSE) // note flag values are strings
	verbose = defVerbose
	// visit actual flags passed in and if passed check -v and set verbose
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "v" {
			verbose = f.Value.String()
		}
	})
	if verbose == defVerbose {
		glog.V(Vuser).Infof("Note: increase the -v level in the controller deployment for more detailed logging, eg. -v=%d or -v=%d\n", Vadmin, Vdebug)
	}

	glog.V(Vdebug).Infof("init: complete: cdi controller will create importer using image %q\n", importerImage)
}

func main() {
	defer glog.Flush()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, configPath)
	if err != nil {
		glog.Fatalf("Unable to get kube config: %v\n", errors.WithStack(err))
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Unable to get kube client: %v\n", errors.WithStack(err))
	}

	cdiClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Error building example clientset: %s", err.Error())
	}

	cdiInformerFactory := informers.NewSharedInformerFactory(cdiClient, DEFAULT_RESYNC_PERIOD)
	pvcInformerFactory := k8sinformers.NewSharedInformerFactory(client, DEFAULT_RESYNC_PERIOD)
	podInformerFactory := k8sinformers.NewFilteredSharedInformerFactory(client, DEFAULT_RESYNC_PERIOD, "", func(options *v1.ListOptions) {
		options.LabelSelector = CDI_LABEL_SELECTOR
	})

	pvcInformer := pvcInformerFactory.Core().V1().PersistentVolumeClaims()
	podInformer := podInformerFactory.Core().V1().Pods()
	dataVolumeInformer := cdiInformerFactory.Cdi().V1alpha1().DataVolumes()

	dataVolumeController := controller.NewDataVolumeController(
		client,
		cdiClient,
		pvcInformer,
		dataVolumeInformer)

	importController := controller.NewImportController(client,
		pvcInformer.Informer(),
		podInformer.Informer(),
		importerImage,
		pullPolicy,
		verbose)

	cloneController := controller.NewCloneController(client,
		pvcInformer.Informer(),
		podInformer.Informer(),
		clonerImage,
		pullPolicy,
		verbose)

	glog.V(Vuser).Infoln("created cdi controllers")

	stopCh := handleSignals()

	go cdiInformerFactory.Start(stopCh)
	go pvcInformerFactory.Start(stopCh)
	go podInformerFactory.Start(stopCh)

	glog.V(Vuser).Infoln("started informers")

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

	<-stopCh
	glog.V(Vadmin).Infoln("cdi controller exited")
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
