/*
 * This file is part of the KubeVirt project
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

	"time"

	"github.com/emicklei/go-restful"
	"github.com/libvirt/libvirt-go"
	"github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	configdisk "kubevirt.io/kubevirt/pkg/config-disk"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/rest"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	virt_api "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
	virtcli "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
)

type virtHandlerApp struct {
	Service      *service.Service
	HostOverride string
	LibvirtUri   string
	SocketDir    string
	CloudInitDir string
}

func newVirtHandlerApp(host *string, port *int, hostOverride *string, libvirtUri *string, socketDir *string, cloudInitDir *string) *virtHandlerApp {
	if *hostOverride == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*hostOverride = defaultHostName
	}

	return &virtHandlerApp{
		Service:      service.NewService("virt-handler", host, port),
		HostOverride: *hostOverride,
		LibvirtUri:   *libvirtUri,
		SocketDir:    *socketDir,
		CloudInitDir: *cloudInitDir,
	}
}

func main() {
	logging.InitializeLogging("virt-handler")
	libvirt.EventRegisterDefaultImpl()
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	host := flag.String("listen", "0.0.0.0", "Address where to listen on")
	port := flag.Int("port", 8185, "Port to listen on")
	hostOverride := flag.String("hostname-override", "", "Kubernetes Pod to monitor for changes")
	socketDir := flag.String("socket-dir", "/var/run/kubevirt", "Directory where to look for sockets for cgroup detection")
	cloudInitDir := flag.String("cloud-init-dir", "/var/run/libvirt/cloud-init-dir", "Base directory for ephemeral cloud init data")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	app := newVirtHandlerApp(host, port, hostOverride, libvirtUri, socketDir, cloudInitDir)

	log := logging.DefaultLogger()
	log.Info().V(1).Log("hostname", app.HostOverride)

	err := cloudinit.SetLocalDirectory(app.CloudInitDir)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				// Report the error somehow or break the loop.
				log.Error().Reason(res).Msg("Listening to libvirt events failed.")
			}
		}
	}()
	domainConn, err := virtcli.NewConnection(app.LibvirtUri, "", "", 10*time.Second)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to libvirtd: %v", err))
	}
	defer domainConn.Close()

	// Create event recorder
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: virtCli.CoreV1().Events(k8sv1.NamespaceAll)})
	// TODO what is scheme used for in Recorder?
	recorder := broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "virt-handler", Host: app.HostOverride})

	domainManager, err := virtwrap.NewLibvirtDomainManager(domainConn,
		recorder,
		isolation.NewSocketBasedIsolationDetector(app.SocketDir),
	)
	if err != nil {
		panic(err)
	}

	l, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", app.HostOverride))
	if err != nil {
		panic(err)
	}

	configDiskClient := configdisk.NewConfigDiskClient()

	// Wire VM controller
	vmListWatcher := kubecli.NewListWatchFromClient(virtCli.RestClient(), "vms", k8sv1.NamespaceAll, fields.Everything(), l)
	vmStore, vmQueue, vmController := virthandler.NewVMController(vmListWatcher, domainManager, recorder, *virtCli.RestClient(), virtCli, app.HostOverride, configDiskClient)

	// Wire Domain controller
	domainSharedInformer, err := virtcache.NewSharedInformer(domainConn)
	if err != nil {
		panic(err)
	}
	domainStore, domainController := virthandler.NewDomainController(vmQueue, vmStore, domainSharedInformer, *virtCli.RestClient(), recorder)

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
		vmStore.Add(v1.NewVMReferenceFromNameWithNS(d.ObjectMeta.Namespace, d.ObjectMeta.Name))
	}

	// Watch for VM changes
	vmController.StartInformer(stop)
	vmController.WaitForSync(stop)

	err = configDiskClient.UndefineUnseen(vmStore)
	if err != nil {
		panic(err)
	}

	go domainController.Run(3, stop)
	go vmController.Run(3, stop)

	// TODO add a http handler which provides health check

	// Add websocket route to access consoles remotely
	console := rest.NewConsoleResource(domainConn)
	migrationHostInfo := rest.NewMigrationHostInfo(isolation.NewSocketBasedIsolationDetector(app.SocketDir))
	ws := new(restful.WebService)
	ws.Route(ws.GET("/api/v1/namespaces/{namespace}/vms/{name}/console").To(console.Console))
	ws.Route(ws.GET("/api/v1/namespaces/{namespace}/vms/{name}/migrationHostInfo").To(migrationHostInfo.MigrationHostInfo))
	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: app.Service.Address(), Handler: restful.DefaultContainer}
	server.ListenAndServe()
}
