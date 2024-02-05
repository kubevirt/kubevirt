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
	"os"
	"path/filepath"

	srv "kubevirt.io/kubevirt/cmd/sidecars/network-passt-binding/server"

	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/cmd/sidecars/launcher"
)

const hookSocket = "passt.sock"

func main() {
	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, hookSocket)

	log.Log.Infof("Starting hook server exposing 'info' and '%s' services on socket %q", socketPath, "v1alpha2")

	registerCallbacks := func(server *grpc.Server, shutdownCh chan struct{}) {
		hooksInfo.RegisterInfoServer(server, srv.InfoServer{Version: "v1alpha2"})
		hooksV1alpha2.RegisterCallbacksServer(server, srv.V1alpha2Server{})
	}
	if err := launcher.Run(socketPath, registerCallbacks); err != nil {
		log.Log.Reason(err).Error("failed to launch sidecar")
		os.Exit(1)
	}
}
