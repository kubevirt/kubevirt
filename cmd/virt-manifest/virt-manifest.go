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
	"time"

	"github.com/emicklei/go-restful"
	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/service"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-manifest/rest"
)

type virtManifestApp struct {
	Service    *service.Service
	LibvirtUri string
}

func newVirtManifestApp(host *string, port *int, libvirtUri *string) *virtManifestApp {
	return &virtManifestApp{
		Service:    service.NewService("virt-manifest", *host, *port),
		LibvirtUri: *libvirtUri,
	}
}

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
	server := &http.Server{Addr: app.Service.Address(), Handler: restful.DefaultContainer}
	logger.Info("Listening for client connections")

	if err := server.ListenAndServe(); err != nil {
		logger.Reason(err).Error("Unable to start web server.")
	}
}

func main() {
	log.InitializeLogging("virt-manifest")
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	listen := flag.String("listen", "0.0.0.0", "Address where to listen on")
	port := flag.Int("port", 8186, "Port to listen on")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	app := newVirtManifestApp(listen, port, libvirtUri)
	app.Run()
}
