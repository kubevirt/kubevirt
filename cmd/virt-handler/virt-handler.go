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
	"fmt"
	"net/http"
	"os"

	"time"

	"github.com/emicklei/go-restful"
	"github.com/libvirt/libvirt-go"
	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	configdisk "kubevirt.io/kubevirt/pkg/config-disk"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	registrydisk "kubevirt.io/kubevirt/pkg/registry-disk"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/rest"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	virt_api "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cache"
	virtcli "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	watchdog "kubevirt.io/kubevirt/pkg/watchdog"
)

const (
	defaultWatchdogTimeout = 30 * time.Second

	// Default port that virt-handler listens on.
	defaultPort = 8185

	// Default address that virt-handler listens on.
	defaultHost = "0.0.0.0"

	// The URI connection string supplied to libvirt. By default, we connect to system-mode daemon of QEMU.
	libvirtUri = "qemu:///system"

	hostOverride = ""

	virtShareDir = "/var/run/kubevirt"

	ephemeralDiskDir = "/var/run/libvirt/kubevirt-ephemeral-disk"
)

type virtHandlerApp struct {
	service.ServiceListen
	service.ServiceLibvirt
	HostOverride            string
	VirtShareDir            string
	EphemeralDiskDir        string
	WatchdogTimeoutDuration time.Duration
}

var _ service.Service = &virtHandlerApp{}

func (app *virtHandlerApp) Run() {
	// HostOverride should default to os.Hostname(), to make sure we handle errors ensure it here.
	if app.HostOverride == "" {
		defaultHostName, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		app.HostOverride = defaultHostName
	}

	logger := log.Log
	logger.V(1).Level(log.INFO).Log("hostname", app.HostOverride)

	err := cloudinit.SetLocalDirectory(app.EphemeralDiskDir + "/cloud-init-data")
	if err != nil {
		panic(err)
	}
	err = registrydisk.SetLocalDirectory(app.EphemeralDiskDir + "/registry-disk-data")
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			if res := libvirt.EventRunDefaultImpl(); res != nil {
				// Report the error somehow or break the loop.
				logger.Reason(res).Error("Listening to libvirt events failed.")
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
		isolation.NewSocketBasedIsolationDetector(app.VirtShareDir),
	)
	if err != nil {
		panic(err)
	}

	l, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", app.HostOverride))
	if err != nil {
		panic(err)
	}

	configDiskClient := configdisk.NewConfigDiskClient(virtCli)

	// Wire VM controller

	// Wire Domain controller
	domainSharedInformer, err := virtcache.NewSharedInformer(domainConn)
	if err != nil {
		panic(err)
	}

	vmSharedInformer := cache.NewSharedIndexInformer(
		controller.NewListWatchFromClient(virtCli.RestClient(), "virtualmachines", k8sv1.NamespaceAll, fields.Everything(), l),
		&v1.VirtualMachine{},
		0,
		cache.Indexers{},
	)

	watchdogInformer := cache.NewSharedIndexInformer(
		watchdog.NewWatchdogListWatchFromClient(
			app.VirtShareDir,
			int(app.WatchdogTimeoutDuration.Seconds())),
		&virt_api.Domain{},
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	vmController := virthandler.NewController(
		domainManager,
		recorder,
		virtCli,
		app.HostOverride,
		configDiskClient,
		app.VirtShareDir,
		int(app.WatchdogTimeoutDuration.Seconds()),
		vmSharedInformer,
		domainSharedInformer,
		watchdogInformer,
	)

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)

	go vmController.Run(3, stop)

	// TODO add a http handler which provides health check

	// Add websocket route to access consoles remotely
	console := rest.NewConsoleResource(domainConn)
	migrationHostInfo := rest.NewMigrationHostInfo(isolation.NewSocketBasedIsolationDetector(app.VirtShareDir))
	ws := new(restful.WebService)
	ws.Route(ws.GET("/api/v1/namespaces/{namespace}/virtualmachines/{name}/console").To(console.Console))
	ws.Route(ws.GET("/api/v1/namespaces/{namespace}/virtualmachines/{name}/migrationHostInfo").To(migrationHostInfo.MigrationHostInfo))
	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: app.Address(), Handler: restful.DefaultContainer}
	server.ListenAndServe()
}

func (app *virtHandlerApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort
	app.LibvirtUri = libvirtUri

	app.AddCommonFlags()
	app.AddLibvirtFlags()

	flag.StringVar(&app.HostOverride, "hostname-override", hostOverride,
		"Name under which the node is registered in kubernetes, where this virt-handler instance is running on")

	flag.StringVar(&app.VirtShareDir, "kubevirt-share-dir", virtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.StringVar(&app.EphemeralDiskDir, "ephemeral-disk-dir", ephemeralDiskDir,
		"Base directory for ephemeral disk data")

	flag.DurationVar(&app.WatchdogTimeoutDuration, "watchdog-timeout", defaultWatchdogTimeout,
		"Watchdog file timeout")
}

func main() {
	log.InitializeLogging("virt-handler")
	libvirt.EventRegisterDefaultImpl()
	app := &virtHandlerApp{}
	service.Setup(app)
	app.Run()
}
