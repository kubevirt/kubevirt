package main

import (
	"flag"
	"fmt"
	libvirtapi "github.com/rgbkrk/libvirt-go"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/libvirt"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/libvirt/cache"
	"os"
	"time"
)

func main() {

	libvirtapi.EventRegisterDefaultImpl()
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	libvirtUser := flag.String("user", "vdsm@ovirt", "Libvirt user")
	libvirtPass := flag.String("pass", "shibboleth", "Libvirt password")
	host := flag.String("hostname-override", "", "Kubernetes Pod to monitor for changes")
	flag.Parse()

	if *host == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*host = defaultHostName
	}
	fmt.Printf("Hostname: %s\n", *host)

	go func() {
		for {
			if res := libvirtapi.EventRunDefaultImpl(); res < 0 {
				// Report the error somehow or break the loop.
			}
		}
	}()
	// TODO we need to handle disconnects
	domainConn, err := libvirt.NewConnection(*libvirtUri, *libvirtUser, *libvirtPass)
	defer domainConn.CloseConnection()
	if err != nil {
		panic(err)
	}

	domainManager, err := libvirt.NewLibvirtDomainManager(domainConn)
	if err != nil {
		panic(err)
	}

	domainListWatcher := virtcache.NewListWatchFromClient(domainConn, libvirtapi.VIR_DOMAIN_EVENT_ID_LIFECYCLE)

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

	vmStore, vmController := virthandler.NewVMController(vmListWatcher, domainManager)

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)
	go domainCache.Run(stop)
	cache.WaitForCacheSync(stop, domainCache.HasSynced)

	// Poplulate the VM store with known Domains on the host, to get deletes since the last run
	for _, domain := range domainCache.GetStore().List() {
		d := domain.(*libvirt.Domain)
		vmStore.Add(&v1.VM{
			ObjectMeta: api.ObjectMeta{Name: d.ObjectMeta.Name, Namespace: api.NamespaceDefault},
		})
	}

	// Watch for domain changes
	go domainController.Run(stop)
	// Watch for VM changes
	go vmController.Run(stop)

	// Sleep forever
	// TODO add a http handler which provides health check
	for {
		time.Sleep(60000 * time.Millisecond)

	}
}
