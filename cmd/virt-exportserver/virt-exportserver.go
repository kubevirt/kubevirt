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
	"os"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/service"

	"kubevirt.io/kubevirt/pkg/storage/export/export"
	exportServer "kubevirt.io/kubevirt/pkg/storage/export/virt-exportserver"
)

const (
	listenAddr = ":8443"
)

func main() {
	log.InitializeLogging("virt-exportserver-" + os.Getenv("POD_NAME"))
	log.Log.Info("Starting export server")

	certFile, keyFile := getCert()
	config := exportServer.ExportServerConfig{
		CertFile:   certFile,
		KeyFile:    keyFile,
		Deadline:   getDeadline(),
		ListenAddr: getListenAddr(),
		TokenFile:  getTokenFile(),
		Paths:      export.CreateServerPaths(export.EnvironToMap()),
	}
	server := exportServer.NewExportServer(config)
	service.Setup(server)
	server.Run()
}

func getTokenFile() string {
	tokenFile := os.Getenv("TOKEN_FILE")
	if tokenFile == "" {
		panic("no token file set")
	}
	return tokenFile
}

func getCert() (certFile, keyFile string) {
	certFile = os.Getenv("CERT_FILE")
	keyFile = os.Getenv("KEY_FILE")
	if certFile == "" || keyFile == "" {
		panic("TLS config incomplete")
	}
	return
}

func getListenAddr() string {
	addr := os.Getenv("LISTEN_ADDR")
	if addr != "" {
		return addr
	}
	return listenAddr
}

func getDeadline() (result time.Time) {
	dl := os.Getenv("DEADLINE")
	if dl != "" {
		var err error
		result, err = time.Parse(time.RFC3339, dl)
		if err != nil {
			panic("Invalid Deadline")
		}
	}
	return
}
