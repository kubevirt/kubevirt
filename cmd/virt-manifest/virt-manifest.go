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
	"time"

	"github.com/emicklei/go-restful"
	flag "github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-manifest/rest"
)

const (
	// Default port that virt-manifest listens on.
	defaultPort = 8186

	// Default address that virt-manifest listens on.
	defaultHost = "0.0.0.0"

	libvirtUri = "qemu:///system"
)

type virtManifestApp struct {
	service.ServiceListen
	service.ServiceLibvirt
	LibvirtUri string
}

var _ service.Service = &virtManifestApp{}

func (app *virtManifestApp) Run() {
	logger := log.Log
	logger.Info("Starting virt-manifest server")

	logger.Info("Connecting to libvirt")

	domainConn, err := cli.NewConnection(app.LibvirtUri, "", "", 60*time.Second)
	if err != nil {
		logger.Reason(err).Error("cannot connect to libvirt")
		panic(fmt.Sprintf("failed to connect to libvirt: %v", err))
	}
	defer domainConn.Close()

	logger.Info("Connected to libvirt")

	ws, err := rest.ManifestService(domainConn)
	if err != nil {
		logger.Reason(err).Error("Unable to create REST server.")
	}

	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: app.Address(), Handler: restful.DefaultContainer}
	logger.Info("Listening for client connections")

	if err := server.ListenAndServe(); err != nil {
		logger.Reason(err).Error("Unable to start web server.")
	}
}

func (app *virtManifestApp) AddFlags() {
	app.InitFlags()

	app.BindAddress = defaultHost
	app.Port = defaultPort
	app.LibvirtUri = libvirtUri

	app.AddCommonFlags()
	app.AddLibvirtFlags()

	flag.Parse()
}

func main() {
	log.InitializeLogging("virt-manifest")
	app := virtManifestApp{}
	app.AddFlags()
	app.Run()
}
