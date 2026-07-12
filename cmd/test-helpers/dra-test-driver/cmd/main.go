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
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	drav1 "k8s.io/kubelet/pkg/apis/dra/v1"
	"kubevirt.io/kubevirt/cmd/test-helpers/dra-test-driver/pkg/driver"
)

const (
	driverName = "hostpath.dra.kubevirt.io"
	pluginPath = "/var/lib/kubelet/plugins/" + driverName
	socketPath = pluginPath + "/dra.sock"
	regPath    = "/var/lib/kubelet/plugins_registry/" + driverName + "-reg.sock"
)

func main() {
	os.RemoveAll(pluginPath)
	os.MkdirAll(pluginPath, 0755)

	go func() {
		reg := driver.NewRegistrationServer(driverName, socketPath)
		if err := reg.Serve(regPath); err != nil {
			log.Fatal(err)
		}
	}()

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	drav1.RegisterDRAPluginServer(grpcServer, driver.New())
	log.Println("Starting DRA plugin server")
	grpcServer.Serve(lis)
}
