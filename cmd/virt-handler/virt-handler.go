package main

import (
	"encoding/xml"
	"fmt"
	libvirtapi "github.com/rgbkrk/libvirt-go"
	"k8s.io/client-go/1.5/tools/cache"
	"kubevirt/core/pkg/libvirt"
	"time"
)

func main() {
	libvirtapi.EventRegisterDefaultImpl()
	go func() {
		for {
			if res := libvirtapi.EventRunDefaultImpl(); res < 0 {
				// Report the error somehow or break the loop.
			}
		}
	}()
	c, err := libvirtapi.NewVirConnection("")
	defer c.CloseConnection()
	if err != nil {
		panic(err)
	}
	lw := libvirt.NewListWatchFromClient(c, libvirtapi.VIR_DOMAIN_EVENT_ID_LIFECYCLE)

	dur, err := time.ParseDuration("15s")
	if err != nil {
		panic(err)
	}
	_, ctl := cache.NewInformer(lw, &libvirt.Domain{}, dur, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			j, _ := xml.Marshal(obj.(*libvirt.Domain).Spec)
			fmt.Printf("ADDED: %s\n", string(j))
		},
		DeleteFunc: func(obj interface{}) {
			j, _ := xml.Marshal(obj.(*libvirt.Domain).Spec)
			fmt.Printf("DELETED: %s\n", string(j))
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			j, _ := xml.Marshal(new.(*libvirt.Domain).Spec)
			fmt.Printf("UPDATED: %s\n", string(j))
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
