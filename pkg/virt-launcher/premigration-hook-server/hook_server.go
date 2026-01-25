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
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

type HookFunc func(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error

// PreMigrationHookServer handles libvirt premigration hook communication via unix socket
type PreMigrationHookServer struct {
	vmi       *v1.VirtualMachineInstance
	hooks     []HookFunc
	stopChan  chan struct{}
	done      chan struct{}
	startOnce sync.Once
	socket    net.Listener
}

// NewPreMigrationHookServer creates a new premigration hook server with registered hooks
func NewPreMigrationHookServer(stopChan chan struct{}, hooks ...HookFunc) *PreMigrationHookServer {
	server := &PreMigrationHookServer{
		hooks:    hooks,
		stopChan: stopChan,
	}

	return server
}

func (h *PreMigrationHookServer) Start(vmi *v1.VirtualMachineInstance) error {
	// Always update the VMI
	h.vmi = vmi
	log.Log.Object(vmi).Info("Hook server updated with VMI")

	var startErr error
	h.startOnce.Do(func() {
		const socketPath = "/var/run/kubevirt/migration-hook-socket"

		socket, err := net.Listen("unix", socketPath)
		if err != nil {
			startErr = fmt.Errorf("failed to listen on unix socket %s: %v", socketPath, err)
			return
		}
		h.socket = socket
		h.done = make(chan struct{})

		connectionHandled := make(chan struct{})

		go func() {
			select {
			case <-h.stopChan:
				log.Log.Infof("Stopping premigration hook server")
			case <-connectionHandled:
				log.Log.Infof("Premigration hook server completed connection handling")
			}
			if h.socket != nil {
				err := h.socket.Close()
				if err != nil {
					log.Log.Errorf("failed to close premigration hook server socket:%v", err.Error())
				}
			}
			if err := os.Remove(socketPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				log.Log.Reason(err).Warning("Failed to remove migration hook socket")
			}
			close(h.done)
		}()

		go func() {
			defer close(connectionHandled)
			conn, err := h.socket.Accept()
			if err != nil {
				log.Log.Reason(err).Info("Premigration hook server stopped accepting connections")
				return
			}
			h.handleConnection(conn)
			err = conn.Close()
			if err != nil {
				log.Log.Errorf("Failed to close connection: %v", err.Error())
			}
			h.vmi = nil // Release VMI memory after processing
		}()

		log.Log.Infof("Started premigration hook server on %s", socketPath)
	})
	return startErr
}

// Done returns a channel that is closed when the server has stopped.
// If Start was never called (source pod), returns a pre-closed channel to not block.
func (h *PreMigrationHookServer) Done() <-chan struct{} {
	if h.done == nil {
		// If Start was never called, return a closed channel to not block
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return h.done
}

func (h *PreMigrationHookServer) handleConnection(conn net.Conn) {
	if err := h.processHook(conn); err != nil {
		log.Log.Errorf("Hook processing failed: %v", err)
		// Just close the connection without sending anything on error
		// The client will detect the closed connection and exit with error
	}
}

func (h *PreMigrationHookServer) processHook(conn net.Conn) error {
	var domain libvirtxml.Domain
	decoder := xml.NewDecoder(conn)
	if err := decoder.Decode(&domain); err != nil {
		return fmt.Errorf("failed to decode XML: %w", err)
	}

	for _, hook := range h.hooks {
		if err := hook(h.vmi, &domain); err != nil {
			return err
		}
	}

	encoder := xml.NewEncoder(conn)
	encoder.Indent("", "  ")
	if err := encoder.Encode(&domain); err != nil {
		return fmt.Errorf("failed to encode XML: %w", err)
	}

	log.Log.Infof("Hook Server successfully processed and returned XML")
	return nil
}
