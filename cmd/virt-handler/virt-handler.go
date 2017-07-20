/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"time"

	"github.com/emicklei/go-restful"
	"github.com/libvirt/libvirt-go"
	kubecorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api"
	kubeapi "k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	kubeinformers "kubevirt.io/kubevirt/pkg/informers"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/rest"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	virtapi "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
)

func main() {

	logging.InitializeLogging("virt-handler")
	libvirt.EventRegisterDefaultImpl()
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	listen := flag.String("listen", "0.0.0.0", "Address where to listen on")
	port := flag.Int("port", 8185, "Port to listen on")
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
				log.Error().Reason(res).Msg("Listening to libvirt events failed.")
			}
		}
	}()
	domainConn, err := virtwrap.NewConnection(*libvirtUri, "", "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to libvirtd: %v", err))
	}
	defer domainConn.Close()

	// Create event recorder
	clientSet, err := kubecli.Get()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&kubecorev1.EventSinkImpl{Interface: clientSet.Events(api.NamespaceAll)})
	// TODO what is scheme used for in Recorder?
	recorder := broadcaster.NewRecorder(kubeapi.Scheme, kubev1.EventSource{Component: "virt-handler", Host: *host})

	domainManager, err := virtwrap.NewLibvirtDomainManager(domainConn, recorder)
	if err != nil {
		panic(err)
	}

	restClient, err := kubecli.GetRESTClient()
	if err != nil {
		panic(err)
	}

	// Create all the informers used by virt-handler here
	informerFactory := kubeinformers.NewKubeInformerFactory(restClient, clientSet)
	vmOnHostInformer := informerFactory.VmOnHost(*host)
	domainSharedInformer := informerFactory.CustomInformer("domainInformer", func() cache.SharedIndexInformer {
		lw := virtcache.NewListWatchFromClient(domainConn)
		return cache.NewSharedIndexInformer(lw, &virtapi.Domain{}, 0, cache.Indexers{})
	})

	// Wire VM controller
	vmStore, vmQueue, vmController := virthandler.NewVMController(domainManager, recorder, *restClient, clientSet, *host, vmOnHostInformer)

	// Wire Domain controller
	domainStore, domainController := virthandler.NewDomainController(vmQueue, vmStore, domainSharedInformer, *restClient, recorder)

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)

	// Start informers
	informerFactory.Start(stop)

	//wait for Domain cache sync
	domainController.WaitForSync(stop)

	// Poplulate the VM store with known Domains on the host, to get deletes since the last run
	for _, domain := range domainStore.List() {
		d := domain.(*virtapi.Domain)
		vmStore.Add(v1.NewVMReferenceFromNameWithNS(d.ObjectMeta.Namespace, d.ObjectMeta.Name))
	}

	vmController.WaitForSync(stop)

	go domainController.Run(1, stop)
	go vmController.Run(1, stop)

	// TODO add a http handler which provides health check

	// Add websocket route to access consoles remotely
	console := rest.NewConsoleResource(domainConn)
	ws := new(restful.WebService)
	ws.Route(ws.GET("/api/v1/namespaces/{namespace}/vms/{name}/console").To(console.Console))
	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: *listen + ":" + strconv.Itoa(*port), Handler: restful.DefaultContainer}
	server.ListenAndServe()
}
