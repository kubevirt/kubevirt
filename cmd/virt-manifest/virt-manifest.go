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
	"strconv"
	"time"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap"
	"kubevirt.io/kubevirt/pkg/virt-manifest/rest"
)

func main() {
	logging.InitializeLogging("virt-manifest")
	libvirtUri := flag.String("libvirt-uri", "qemu:///system", "Libvirt connection string.")
	listen := flag.String("listen", "0.0.0.0", "Address where to listen on")
	port := flag.Int("port", 8186, "Port to listen on")
	flag.Parse()

	log := logging.DefaultLogger()
	log.Info().Msg("Starting virt-manifest server")

	log.Info().Msg("Connecting to libvirt")

	domainConn, err := virtwrap.NewConnection(*libvirtUri, "", "", 60*time.Second)
	if err != nil {
		log.Error().Reason(err).Msg("cannot connect to libvirt")
		panic(fmt.Sprintf("failed to connect to libvirt: %v", err))
	}
	defer domainConn.Close()

	log.Info().Msg("Connected to libvirt")

	ws, err := rest.ManifestService(domainConn)
	if err != nil {
		log.Error().Reason(err).Msg("Unable to create REST server.")
	}

	restful.DefaultContainer.Add(ws)
	server := &http.Server{Addr: *listen + ":" + strconv.Itoa(*port), Handler: restful.DefaultContainer}
	log.Info().Msg("Listening for client connections")

	if err := server.ListenAndServe(); err != nil {
		log.Error().Reason(err).Msg("Unable to start web server.")
	}
}
