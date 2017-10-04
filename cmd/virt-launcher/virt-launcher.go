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
	"log"
	"net"
	"os"
	"time"

	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
)

func markReady(readinessFile string) {
	f, err := os.OpenFile(readinessFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	f.Close()
	log.Printf("Marked as ready")
}

func createSocket(socketDir string, namespace string, name string) net.Listener {
	socket, err := net.Listen("unix", isolation.SocketFromNamespaceName(socketDir, namespace, name))

	if err != nil {
		log.Fatal("Could not create socket for cgroup detection.", err)
	}

	return socket
}

func main() {
	startTimeout := 0 * time.Second

	logging.InitializeLogging("virt-launcher")
	qemuTimeout := flag.Duration("qemu-timeout", startTimeout, "Amount of time to wait for qemu")
	debugMode := flag.Bool("debug", false, "Enable debug messages")
	socketDir := flag.String("socket-dir", "/var/run/kubevirt", "Directory where to place a socket for cgroup detection")
	name := flag.String("name", "", "Name of the VM")
	namespace := flag.String("namespace", "", "Namespace of the VM")
	readinessFile := flag.String("readiness-file", "/tmp/health", "Pod looks for tihs file to determine when virt-launcher is initialized")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	socket := createSocket(*socketDir, *namespace, *name)
	defer socket.Close()

	mon := virtlauncher.NewProcessMonitor("qemu", *debugMode)

	markReady(*readinessFile)
	mon.RunForever(*qemuTimeout)
}
