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
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	kubecorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/pkg/api"
	kubeapi "k8s.io/client-go/pkg/api"
	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/rest"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	virt_api "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
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
	coreClient, err := kubecli.Get()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&kubecorev1.EventSinkImpl{Interface: coreClient.Events(api.NamespaceDefault)})
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

	l, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", *host))
	if err != nil {
		panic(err)
	}

	// Wire VM controller
	vmListWatcher := kubecli.NewListWatchFromClient(restClient, "vms", api.NamespaceDefault, fields.Everything(), l)
	vmStore, vmQueue, vmController := virthandler.NewVMController(vmListWatcher, domainManager, recorder, *restClient, coreClient, *host)

	// Wire Domain controller
	domainSharedInformer, err := virtcache.NewSharedInformer(domainConn)
	if err != nil {
		panic(err)
	}
	domainStore, domainController := virthandler.NewDomainController(vmQueue, vmStore, domainSharedInformer, *restClient, recorder)

	if err != nil {
		panic(err)
	}

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)

	// Start domain controller and wait for Domain cache sync
	domainController.StartInformer(stop)
	domainController.WaitForSync(stop)

	// Poplulate the VM store with known Domains on the host, to get deletes since the last run
	for _, domain := range domainStore.List() {
		d := domain.(*virt_api.Domain)
		vmStore.Add(v1.NewVMReferenceFromName(d.ObjectMeta.Name))
	}

	// Watch for VM changes
	vmController.StartInformer(stop)
	vmController.WaitForSync(stop)

	go domainController.Run(1, stop)
	go vmController.Run(1, stop)

	// TODO add a http handler which provides health check

	// Add websocket route to access consoles remotely
	console := rest.NewConsoleResource(domainConn)
	ws := new(restful.WebService)
	ws.Route(ws.GET("/api/v1/console/{name}").To(console.Console))
	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: *listen + ":" + strconv.Itoa(*port), Handler: restful.DefaultContainer}
	server.ListenAndServe()
}
