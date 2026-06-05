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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package main

import (
	"net"
	"os"
	"path/filepath"

	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/cmd/sidecars/network-slirp-binding/dns"
	srv "kubevirt.io/kubevirt/cmd/sidecars/network-slirp-binding/server"
)

func main() {
	searchDomains, err := dns.ReadResolvConfSearchDomains()
	if err != nil {
		log.Log.Errorf("failed to read resolv.conf search domains: %v", err)
		os.Exit(1)
	}

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, "slirp.sock")
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		os.Exit(1)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, srv.InfoServer{Version: "v1alpha2"})
	hooksV1alpha2.RegisterCallbacksServer(server, srv.V1alpha2Server{SearchDomains: searchDomains})

	log.Log.Infof("Starting hook server exposing 'info' and '%s' services on socket %q", socketPath, "v1alpha2")
	server.Serve(socket)
}
