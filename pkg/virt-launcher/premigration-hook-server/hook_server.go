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
 */

package premigrationhookserver

import (
	"encoding/xml"
	"fmt"
	"net"
	"os"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/compute"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/storage"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/types"
)

// PreMigrationHookServer handles libvirt premigration hook communication via unix socket
type PreMigrationHookServer struct {
	vmi   *v1.VirtualMachineInstance
	hooks []types.HookFunc
}

// NewPreMigrationHookServer creates a new premigration hook server with registered hooks
func NewPreMigrationHookServer() *PreMigrationHookServer {
	server := &PreMigrationHookServer{
		hooks: make([]types.HookFunc, 0),
	}

	server.registerAllHooks()

	return server
}

func (h *PreMigrationHookServer) Start(stopChan chan struct{}) (chan struct{}, error) {
	socketPath := "/var/run/kubevirt/migration-hook-socket"

	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on unix socket %s: %v", socketPath, err)
	}

	done := make(chan struct{})
	go func() {
		<-stopChan
		log.Log.Infof("Stopping premigration hook server")
		if socket != nil {
			err := socket.Close()
			if err != nil {
				log.Log.Errorf("failed to close premigration hook server socket:%v", err.Error())
			}
		}
		err := os.Remove(socketPath)
		if err != nil {
			log.Log.Errorf("failed to remove premigration hook server socket:%v", err.Error())
		}
		close(done)
	}()

	go func() {
		for {
			conn, err := socket.Accept()
			if err != nil {
				log.Log.Reason(err).Info("Premigration hook server stopped accepting connections")
				return
			}
			h.handleConnection(conn)
			err = conn.Close()
			if err != nil {
				log.Log.Errorf("Failed to close connection: %v", err.Error())
			}
		}
	}()

	log.Log.Infof("Started premigration hook server on %s", socketPath)
	return done, nil
}

func (h *PreMigrationHookServer) handleConnection(conn net.Conn) {
	var domain libvirtxml.Domain
	decoder := xml.NewDecoder(conn)
	if err := decoder.Decode(&domain); err != nil {
		log.Log.Errorf("Failed to decode XML from connection:%v", err.Error())
		return
	}

	for _, hook := range h.hooks {
		hook(h.vmi, &domain)
	}

	encoder := xml.NewEncoder(conn)
	encoder.Indent("", "  ")
	if err := encoder.Encode(&domain); err != nil {
		log.Log.Errorf("Failed to encode XML to connection:%v", err.Error())
		return
	}

	log.Log.Infof("Hook Server successfully processed and returned XML")
}

func (h *PreMigrationHookServer) SetVMI(fullVMI *v1.VirtualMachineInstance) {
	h.vmi = fullVMI
	log.Log.Object(fullVMI).Info("Hook server updated with full VMI including status")
}

func (h *PreMigrationHookServer) registerAllHooks() {
	for _, hook := range compute.GetComputeHooks() {
		h.hooks = append(h.hooks, hook)
	}

	for _, hook := range storage.GetStorageHooks() {
		h.hooks = append(h.hooks, hook)
	}

	for _, hook := range network.GetNetworkHooks() {
		h.hooks = append(h.hooks, hook)
	}
}
