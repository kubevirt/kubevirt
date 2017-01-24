package main

import (
	"flag"
	"fmt"
	"github.com/libvirt/libvirt-go"
	kubecorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/labels"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
	"os"
	"time"
)

func main() {

	logging.InitializeLogging("virt-handler")
	libvirt.EventRegisterDefaultImpl()
	libvirtUri := flag.String("virtwrap-uri", "qemu:///system", "Libvirt connection string.")
	libvirtUser := flag.String("user", "", "Libvirt user")
	libvirtPass := flag.String("pass", "", "Libvirt password")
	host := flag.String("hostname-override", "", "Kubernetes Pod to monitor for changes")
	flag.Parse()

	if *host == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*host = defaultHostName
	}
	log := logging.DefaultLogger()
	log.Info().V(1).Log("hostname", *host)

	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				// Report the error somehow or break the loop.
				log.Warning().Log("msg", "No results from virtwrap")
			}
		}
	}()
	// TODO we need to handle disconnects
	domainConn, err := virtwrap.NewConnection(*libvirtUri, *libvirtUser, *libvirtPass)
	if err != nil {
		panic(err)
	}
	defer domainConn.Close()

	// Create event recorder
	coreClient, err := kubecli.Get()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&kubecorev1.EventSinkImpl{Interface: coreClient.Events(api.NamespaceDefault)})
	recorder := broadcaster.NewRecorder(kubev1.EventSource{Component: "virt-handler", Host: *host})

	domainManager, err := virtwrap.NewLibvirtDomainManager(domainConn, recorder)
	if err != nil {
		panic(err)
	}

	domainListWatcher := virtcache.NewListWatchFromClient(domainConn)

	domainController := virthandler.NewDomainController(domainListWatcher)

	domainCache, err := virtcache.NewDomainCache(domainConn)
	if err != nil {
		panic(err)
	}

	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	l, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", *host))
	if err != nil {
		panic(err)
	}

	vmListWatcher := kubecli.NewListWatchFromClient(restClient, "vms", api.NamespaceDefault, fields.Everything(), l)

	vmStore, vmController := virthandler.NewVMController(vmListWatcher, domainManager, recorder, *restClient)

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)

	stopWarmup := make(chan struct{})
	go domainCache.Run(stopWarmup)
	cache.WaitForCacheSync(stop, domainCache.HasSynced)
	close(stopWarmup)

	// Poplulate the VM store with known Domains on the host, to get deletes since the last run
	for _, domain := range domainCache.GetStore().List() {
		d := domain.(*virtwrap.Domain)
		vmStore.Add(util.NewVMReferenceFromName(d.ObjectMeta.Name))
	}

	// Watch for domain changes
	go domainController.Run(stop)
	// Watch for VM changes
	go vmController.Run(1, stop)

	// Sleep forever
	// TODO add a http handler which provides health check
	for {
		time.Sleep(60000 * time.Millisecond)

	}
}
