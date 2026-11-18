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

package vsock

import (
	"fmt"

	"github.com/mdlayher/vsock"
	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"
	"kubevirt.io/kubevirt/pkg/vsock/system"
	systemv1 "kubevirt.io/kubevirt/pkg/vsock/system/v1"
)

type ServerCache struct {
	caManager kvtls.ClientCAManager
	servers   *RefCounter[int, *grpc.Server]
}

func (s *ServerCache) StartCAServer(pid int) (func(), error) {
	_, release, err := s.servers.Get(pid, func() (*grpc.Server, func(), error) {
		return s.startServer(pid)
	})
	if err != nil {
		return nil, err
	}
	return release, nil
}

func (s *ServerCache) startServer(pid int) (*grpc.Server, func(), error) {
	const vsockPort = 1
	listener, lisErr := vsock.ListenContextID(vsock.Host, vsockPort, &vsock.Config{})
	if lisErr != nil {
		return nil, nil, fmt.Errorf("failed to listen on VSOCK CID %d port %d in namespace of PID %d: %w", vsock.Host, vsockPort, pid, lisErr)
	}

	server := grpc.NewServer()
	systemv1.RegisterSystemServer(server, system.NewSystemService(s.caManager))
	go func() {
		serveErr := server.Serve(listener)
		if serveErr != nil {
			log.DefaultLogger().Errorf("VSOCK ns listener for PID %d on port %d failed: %v", pid, vsockPort, serveErr)
		}
	}()

	log.DefaultLogger().Infof("VSOCK ns listener created for PID %d on port %d", pid, vsockPort)

	destroyFn := func() {
		server.Stop()
		log.DefaultLogger().Infof("VSOCK ns listener removed for PID %d", pid)
	}

	return server, destroyFn, nil
}

func NewServerCache(caManager kvtls.ClientCAManager) *ServerCache {
	return &ServerCache{
		caManager: caManager,
		servers:   NewRefCounter[int, *grpc.Server](),
	}
}
