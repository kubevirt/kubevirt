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
	"os"

	"time"

	flag "github.com/spf13/pflag"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes/scheme"
	k8coresv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/controller"
	inotifyinformer "kubevirt.io/kubevirt/pkg/inotify-informer"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-handler"
	virtcache "kubevirt.io/kubevirt/pkg/virt-handler/cache"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	virt_api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	defaultWatchdogTimeout = 15 * time.Second

	// Default port that virt-handler listens on.
	defaultPort = 8185

	// Default address that virt-handler listens on.
	defaultHost = "0.0.0.0"

	hostOverride = ""

	virtShareDir = "/var/run/kubevirt"
)

type virtHandlerApp struct {
	service.ServiceListen
	HostOverride            string
	VirtShareDir            string
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

	// Create event recorder
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&k8coresv1.EventSinkImpl{Interface: virtCli.CoreV1().Events(k8sv1.NamespaceAll)})
	// Scheme is used to create an ObjectReference from an Object (e.g. VirtualMachineInstance) during Event creation
	recorder := broadcaster.NewRecorder(scheme.Scheme, k8sv1.EventSource{Component: "virt-handler", Host: app.HostOverride})

	if err != nil {
		panic(err)
	}

	l, err := labels.Parse(fmt.Sprintf(v1.NodeNameLabel+" in (%s)", app.HostOverride))
	if err != nil {
		panic(err)
	}

	// Wire VirtualMachineInstance controller

	// Wire Domain controller
	domainSharedInformer, err := virtcache.NewSharedInformer(app.VirtShareDir, int(app.WatchdogTimeoutDuration.Seconds()))
	if err != nil {
		panic(err)
	}

	vmSharedInformer := cache.NewSharedIndexInformer(
		controller.NewListWatchFromClient(virtCli.RestClient(), "virtualmachineinstances", k8sv1.NamespaceAll, fields.Everything(), l),
		&v1.VirtualMachineInstance{},
		0,
		cache.Indexers{},
	)

	virtlauncher.InitializeSharedDirectories(app.VirtShareDir)

	gracefulShutdownInformer := cache.NewSharedIndexInformer(
		inotifyinformer.NewFileListWatchFromClient(
			virtlauncher.GracefulShutdownTriggerDir(app.VirtShareDir)),
		&virt_api.Domain{},
		0,
		cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})

	vmController := virthandler.NewController(
		recorder,
		virtCli,
		app.HostOverride,
		app.VirtShareDir,
		vmSharedInformer,
		domainSharedInformer,
		gracefulShutdownInformer,
		int(app.WatchdogTimeoutDuration.Seconds()),
	)

	// Bootstrapping. From here on the startup order matters
	stop := make(chan struct{})
	defer close(stop)

	vmController.Run(3, stop)
}

func (app *virtHandlerApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort

	app.AddCommonFlags()

	flag.StringVar(&app.HostOverride, "hostname-override", hostOverride,
		"Name under which the node is registered in kubernetes, where this virt-handler instance is running on")

	flag.StringVar(&app.VirtShareDir, "kubevirt-share-dir", virtShareDir,
		"Shared directory between virt-handler and virt-launcher")

	flag.DurationVar(&app.WatchdogTimeoutDuration, "watchdog-timeout", defaultWatchdogTimeout,
		"Watchdog file timeout")
}

func main() {
	app := &virtHandlerApp{}
	service.Setup(app)
	log.InitializeLogging("virt-handler")
	app.Run()
}
