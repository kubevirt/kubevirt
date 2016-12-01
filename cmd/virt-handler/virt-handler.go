package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	libvirtapi "github.com/rgbkrk/libvirt-go"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt/core/pkg/libvirt"
	virtcache "kubevirt/core/pkg/libvirt/cache"
	"time"
)

func main() {
	flag.Parse()
	libvirtapi.EventRegisterDefaultImpl()
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	libvirtUser := flag.String("user", "vdsm@ovirt", "Libvirt user")
	libvirtPass := flag.String("pass", "shibboleth", "Libvirt password")
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
	lw := virtcache.NewListWatchFromClient(c, libvirtapi.VIR_DOMAIN_EVENT_ID_LIFECYCLE)

	_, ctl := cache.NewInformer(lw, &libvirt.Domain{}, 0, cache.ResourceEventHandlerFuncs{
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
	stop := make(chan struct{})
	defer close(stop)
	go ctl.Run(stop)

	for {
		fmt.Println("Sleeping")
		// Sleep for one minute
		time.Sleep(60000 * time.Millisecond)

	}
}
