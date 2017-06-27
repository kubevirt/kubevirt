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
	golog "log"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-controller/rest"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"
)

func main() {

	logging.InitializeLogging("virt-controller")

	watch.Register()

	restful.Add(rest.WebService)

	// Bootstrapping. From here on the initialization order is important
	stop := make(chan struct{})
	defer close(stop)

	// Start watching vms
	vmController := watch.GetVMController(watch.CC)
	go vmController.Run(1, stop)

	//FIXME when we have more than one worker, we need a lock on the VM
	migrationController := watch.GetMigrationController(watch.CC)
	go migrationController.Run(1, stop)

	server := watch.GetHttpServer(watch.CC)

	if err := server.ListenAndServe(); err != nil {
		golog.Fatal(err)
	}
}
