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
	"path/filepath"
	"time"

	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
	watchdog "kubevirt.io/kubevirt/pkg/watchdog"
)

func markReady(readinessFile string) {
	f, err := os.OpenFile(readinessFile, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	f.Close()
	log.Printf("Marked as ready\n")
}

func createSocket(virtShareDir string, namespace string, name string) net.Listener {
	sockFile := isolation.SocketFromNamespaceName(virtShareDir, namespace, name)

	err := os.MkdirAll(filepath.Dir(sockFile), 0755)
	if err != nil {
		log.Fatal("Could not create directory for socket.", err)
	}

	if err := os.RemoveAll(sockFile); err != nil {
		log.Fatal("Could not clean up old socket for cgroup detection", err)
	}
	socket, err := net.Listen("unix", sockFile)

	if err != nil {
		log.Fatal("Could not create socket for cgroup detection.", err)
	}

	return socket
}

func main() {
	startTimeout := 0 * time.Second
	defaultInterval := 10 * time.Second

	logging.InitializeLogging("virt-launcher")
	qemuTimeout := flag.Duration("qemu-timeout", startTimeout, "Amount of time to wait for qemu")
	debugMode := flag.Bool("debug", false, "Enable debug messages")
	virtShareDir := flag.String("kubevirt-share-dir", "/var/run/kubevirt", "Shared directory between virt-handler and virt-launcher")
	name := flag.String("name", "", "Name of the VM")
	namespace := flag.String("namespace", "", "Namespace of the VM")
	watchdogInterval := flag.Duration("watchdog-update-interval", defaultInterval, "Interval at which watchdog file should be updated")
	readinessFile := flag.String("readiness-file", "/tmp/health", "Pod looks for tihs file to determine when virt-launcher is initialized")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	socket := createSocket(*virtShareDir, *namespace, *name)
	defer socket.Close()

	err := virtlauncher.InitializeSharedDirectories(*virtShareDir)
	if err != nil {
		panic(err)
	}

	watchdogFile := watchdog.WatchdogFileFromNamespaceName(*virtShareDir, *namespace, *name)
	err = watchdog.WatchdogFileUpdate(watchdogFile)
	if err != nil {
		panic(err)
	}

	log.Printf("Watchdog file created at %s\n", watchdogFile)

	stopChan := make(chan struct{})
	defer close(stopChan)
	go func() {

		ticker := time.NewTicker(*watchdogInterval).C
		for {
			select {
			case <-stopChan:
				return
			case <-ticker:
				err := watchdog.WatchdogFileUpdate(watchdogFile)
				if err != nil {
					panic(err)
				}
			}
		}
	}()

	pidFile := virtlauncher.QemuPidfileFromNamespaceName(*virtShareDir, *namespace, *name)
	mon := virtlauncher.NewProcessMonitor(pidFile, *debugMode)

	markReady(*readinessFile)
	mon.RunForever(*qemuTimeout)
}
