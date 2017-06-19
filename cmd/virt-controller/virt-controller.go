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
	golog "log"
	"net/http"
	"strconv"

	"github.com/emicklei/go-restful"
	clientrest "k8s.io/client-go/rest"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"
)

func main() {

	logging.InitializeLogging("virt-controller")
	host := flag.String("listen", "0.0.0.0", "Address and port where to listen on")
	port := flag.Int("port", 8182, "Port to listen on")

	watch.Register()

	logger := logging.DefaultLogger()
	var restClient *clientrest.RESTClient

	vmService := watch.GetVMService(watch.CC)

	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important
	stop := make(chan struct{})
	defer close(stop)

	// Start wachting vms
	restClient = watch.GetRestClient(watch.CC)
	clientSet := watch.GetClientSet(watch.CC)


	vmController := watch.GetVMController(watch.CC) //watch.NewVMController(vmService, nil, restClient, clientSet)
	go vmController.Run(1, stop)

	//FIXME when we have more than one worker, we need a lock on the VM
	migrationController := watch.NewMigrationController(vmService, restClient, clientSet)
	go migrationController.Run(1, stop)

	httpLogger := logger.With("service", "http")

	httpLogger.Info().Log("action", "listening", "interface", *host, "port", *port)
	if err := http.ListenAndServe(*host+":"+strconv.Itoa(*port), nil); err != nil {
		golog.Fatal(err)
	}
}
