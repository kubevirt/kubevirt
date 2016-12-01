package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	libvirtapi "github.com/rgbkrk/libvirt-go"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt/core/pkg/api/v1"
	"kubevirt/core/pkg/kubecli"
	"kubevirt/core/pkg/libvirt"
	virtcache "kubevirt/core/pkg/libvirt/cache"
	"kubevirt/core/pkg/util"
	"os"
	"time"
	"github.com/rmohr/go-model"
)

func main() {

	flag.Parse()
	libvirtapi.EventRegisterDefaultImpl()
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	libvirtUser := flag.String("user", "vdsm@ovirt", "Libvirt user")
	libvirtPass := flag.String("pass", "shibboleth", "Libvirt password")
	host := flag.String("hostname-override", "", "Kubernetes Pod to monitor for changes")

	if *host == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*host = defaultHostName
	}

	go func() {
		for {
			if res := libvirtapi.EventRunDefaultImpl(); res < 0 {
				// Report the error somehow or break the loop.
			}
		}
	}()
	c, err := libvirtapi.NewVirConnectionWithAuth(*libvirtUri, *libvirtUser, *libvirtPass)
	defer c.CloseConnection()
	if err != nil {
		panic(err)
	}

	vmCache, err := util.NewVMCache()
	if err != nil {
		panic(err)
	}

	domainCache, err := virtcache.NewDomainCache(c)
	if err != nil {
		panic(err)
	}

	domainListWatcher := virtcache.NewListWatchFromClient(c, libvirtapi.VIR_DOMAIN_EVENT_ID_LIFECYCLE)

	_, domainController := cache.NewInformer(domainListWatcher, &libvirt.Domain{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			j, _ := xml.Marshal(obj.(*libvirt.Domain).Spec)
			fmt.Printf("ADDED: %s, %s\n", obj.(*libvirt.Domain).Status.Status, string(j))
		},
		DeleteFunc: func(obj interface{}) {
			j, _ := xml.Marshal(obj.(*libvirt.Domain).Spec)
			fmt.Printf("DELETED: %s, %s\n", obj.(*libvirt.Domain).Status.Status, string(j))
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			j, _ := xml.Marshal(new.(*libvirt.Domain).Spec)
			fmt.Printf("UPDATED: %s, %s\n", new.(*libvirt.Domain).Status.Status, string(j))
		},
	})
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	l, err := labels.Parse(fmt.Sprintf("kubvirt.io/nodeName in (%s)", *host))
	if err != nil {
		panic(err)
	}
	vmListWatcher := kubecli.NewListWatchFromClient(restClient, "vms", api.NamespaceDefault, fields.Everything(), l)

	vmStore, vmController := kubecli.NewInformer(vmListWatcher, &v1.VM{}, 0, &vmResourceEventHandler{})

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)
	go vmCache.Run(stop)
	go domainCache.Run(stop)
	cache.WaitForCacheSync(stop, vmCache.HasSynced)
	cache.WaitForCacheSync(stop, domainCache.HasSynced)

	for domain := range domainCache.GetStore().List() {
		d := domain.(*libvirt.Domain)
		vmStore.Add(&v1.VM{
			ObjectMeta: api.ObjectMeta{Name: d.ObjectMeta.Name, Namespace: api.NamespaceDefault},
		})
	}

	// Watch for domain changes
	go domainController.Run(stop)
	// Watch for VM changes
	go vmController.Run(stop)

	for {
		fmt.Println("Sleeping")
		// Sleep for one minute
		time.Sleep(60000 * time.Millisecond)

	}
}

type vmResourceEventHandler struct {}

func (v *vmResourceEventHandler) OnAdd(obj interface{}) error {
	// define and start
	domainSpec := &libvirt.DomainSpec{}
	vm := obj.(*v1.VM)
	model.Copy(domainSpec, vm.Spec.Domain)
	return nil
}

func (v *vmResourceEventHandler) OnUpdate(oldObj, newObj interface{}) error {
	return nil
}

func (v *vmResourceEventHandler) OnDelete(obj interface{}) error {
	// stop and undefine
	return nil
}
