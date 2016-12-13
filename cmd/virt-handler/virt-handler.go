package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	libvirtapi "github.com/rgbkrk/libvirt-go"
	"github.com/rmohr/go-model"
	"k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/fields"
	"k8s.io/client-go/1.5/pkg/labels"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/libvirt"
	virtcache "kubevirt.io/kubevirt/pkg/libvirt/cache"
	"kubevirt.io/kubevirt/pkg/util"
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

	// The right nodename is vital for us. Resolution order is:
	// 1) hostname-override
	// 2) HOSTNAME_OVERRIDE env variable
	// 3) hostname reported by the system (not very useful when in a container)
	if *host == "" {
		*host = os.Getenv("HOSTNAME_OVERRIDE")
		if *host == "" {
			defaultHostName, err := os.Hostname()
			if err != nil {
				panic(err)
			}
			*host = defaultHostName
		}
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
	domainCacheConn, err := libvirtapi.NewVirConnectionWithAuth(*libvirtUri, *libvirtUser, *libvirtPass)
	defer domainCacheConn.CloseConnection()
	if err != nil {
		panic(err)
	}

	// TODO we need to handle disconnects
	domainLWConn, err := libvirtapi.NewVirConnectionWithAuth(*libvirtUri, *libvirtUser, *libvirtPass)
	defer domainLWConn.CloseConnection()
	if err != nil {
		panic(err)
	}

	vmCache, err := util.NewVMCache()
	if err != nil {
		panic(err)
	}

	domainCache, err := virtcache.NewDomainCache(domainCacheConn)
	if err != nil {
		panic(err)
	}

	domainListWatcher := virtcache.NewListWatchFromClient(domainLWConn, libvirtapi.VIR_DOMAIN_EVENT_ID_LIFECYCLE)

	_, domainController := kubecli.NewInformer(domainListWatcher, &libvirt.Domain{}, 0, kubecli.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) error {
			fmt.Printf("Domain ADDED: %s: %s\n", obj.(*libvirt.Domain).GetObjectMeta().GetName(), obj.(*libvirt.Domain).Status.Status)
			return nil
		},
		DeleteFunc: func(obj interface{}) error {
			fmt.Printf("Domain DELETED: %s: %s\n", obj.(*libvirt.Domain).GetObjectMeta().GetName(), obj.(*libvirt.Domain).Status.Status)
			return nil
		},
		UpdateFunc: func(old interface{}, new interface{}) error {
			fmt.Printf("Domain UPDATED: %s: %s\n", new.(*libvirt.Domain).GetObjectMeta().GetName(), new.(*libvirt.Domain).Status.Status)
			return nil
		},
	})
	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	l, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", *host))
	if err != nil {
		panic(err)
	}
	vmListWatcher := kubecli.NewListWatchFromClient(restClient, "vms", api.NamespaceDefault, fields.Everything(), l)

	vmStore, vmController := kubecli.NewInformer(vmListWatcher, &v1.VM{}, 0, kubecli.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) error {
			// define and start
			fmt.Printf("VM ADD\n")
			domainSpec := &libvirt.DomainSpec{}
			vm := obj.(*v1.VM)
			errs := model.Copy(domainSpec, vm.Spec.Domain)
			if len(errs) > 0 {
				fmt.Println(errs)
				return nil
			}
			domXML, _ := xml.Marshal(domainSpec)
			dom, err := domainCacheConn.DomainDefineXML(string(domXML))
			if err != nil {
				fmt.Println(err)
				if code := err.(libvirtapi.VirError).Code; code != libvirtapi.VIR_ERR_DOM_EXIST {
					time.Sleep(1 * time.Second)
					return cache.ErrRequeue{Err: err}
				}
				// TODO more fine grained checks, backoff, ...
			}

			if err := dom.Create(); err != nil {
				fmt.Println(err)
				if code := err.(libvirtapi.VirError).Code; code != libvirtapi.VIR_ERR_OPERATION_INVALID {
					time.Sleep(1 * time.Second)
					return cache.ErrRequeue{Err: err}
				}
				// TODO check if vm is already runnin, backoff, ...
				// For now we assume the VM is running when the operation was invalid
			}

			return nil
		},
		DeleteFunc: func(obj interface{}) error {
			// stop and undefine
			fmt.Printf("VM DELETE\n")
			vm, ok := obj.(*v1.VM)
			if !ok {
				vm = obj.(cache.DeletedFinalStateUnknown).Obj.(*v1.VM)
			}
			domain, err := domainCacheConn.LookupDomainByName(vm.ObjectMeta.Name)
			if err != nil {
				fmt.Println(err)
				return nil
			}

			err = domain.Destroy()
			if err != nil {
				fmt.Println(err)
				if code := err.(libvirtapi.VirError).Code; code != libvirtapi.VIR_ERR_OPERATION_INVALID {
					// TODO more fine grained checks, backoff, ...
					time.Sleep(1 * time.Second)
					return cache.ErrRequeue{Err: err}
				}
			}
			domain.Undefine()
			if err != nil {
				fmt.Println(err)
				if code := err.(libvirtapi.VirError).Code; code != libvirtapi.VIR_ERR_OPERATION_INVALID {
					// TODO more fine grained checks, backoff, ...
					time.Sleep(1 * time.Second)
					return cache.ErrRequeue{Err: err}
				}
			}
			return nil
		},
		UpdateFunc: func(old interface{}, new interface{}) error {
			fmt.Printf("VM UPDATE\n")
			return nil
		},
	})

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)
	go vmCache.Run(stop)
	go domainCache.Run(stop)
	cache.WaitForCacheSync(stop, vmCache.HasSynced)
	cache.WaitForCacheSync(stop, domainCache.HasSynced)

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
