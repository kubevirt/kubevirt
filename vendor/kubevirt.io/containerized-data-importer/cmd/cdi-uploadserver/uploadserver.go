/*
 * This file is part of the CDI project
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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"flag"
	"os"
	"strconv"

	"k8s.io/klog"
	"kubevirt.io/containerized-data-importer/pkg/common"
	"kubevirt.io/containerized-data-importer/pkg/uploadserver"
)

const (
	defaultListenPort    = 8443
	defaultListenAddress = "0.0.0.0"

	defaultDestination = common.ImporterWritePath
)

func init() {
	flag.Parse()
	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			f2.Value.Set(value)
		}
	})

}

func main() {
	defer klog.Flush()

	listenAddress, listenPort := getListenAddressAndPort()

	destination := getDestination()

	server := uploadserver.NewUploadServer(
		listenAddress,
		listenPort,
		destination,
		os.Getenv("TLS_KEY"),
		os.Getenv("TLS_CERT"),
		os.Getenv("CLIENT_CERT"),
		os.Getenv(common.UploadImageSize),
	)

	klog.Infof("Upload destination: %s", destination)

	klog.Infof("Running server on %s:%d", listenAddress, listenPort)

	err := server.Run()
	if err != nil {
		klog.Errorf("UploadServer failed: %s", err)
		os.Exit(1)
	}

	klog.Info("UploadServer successfully exited")
}

func getListenAddressAndPort() (string, int) {
	addr, port := defaultListenAddress, defaultListenPort

	// empty value okay here
	if val, exists := os.LookupEnv("LISTEN_ADDRESS"); exists {
		addr = val
	}

	// not okay here
	if val := os.Getenv("LISTEN_PORT"); len(val) > 0 {
		n, err := strconv.ParseUint(val, 10, 16)
		if err == nil {
			port = int(n)
		}
	}

	return addr, port
}

func getDestination() string {
	destination := defaultDestination

	if val := os.Getenv("DESTINATION"); len(val) > 0 {
		destination = val
	}

	return destination
}
