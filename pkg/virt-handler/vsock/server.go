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
	"os"
	"sync"
	"time"

	"github.com/mdlayher/vsock"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util/tls"
	"kubevirt.io/kubevirt/pkg/virt-handler/vsock/system"
	v1 "kubevirt.io/kubevirt/pkg/vsock/system/v1"
)

type Hypervisor struct {
	running   bool
	lock      sync.Mutex
	doneChan  chan struct{}
	stopChan  chan struct{}
	port      uint32
	caManager tls.ClientCAManager
	server    *grpc.Server
}

func (h *Hypervisor) Stop() {
	h.lock.Lock()
	defer h.lock.Unlock()
	if !h.running {
		log.DefaultLogger().Infof("VSOCK server is already stopped")
		return
	}
	h.stop()
}

func (h *Hypervisor) Start() {
	h.lock.Lock()
	defer h.lock.Unlock()
	if h.running {
		log.DefaultLogger().Infof("VSOCK server is already running")
		return
	}
	h.running = true
	h.doneChan = make(chan struct{})
	h.stopChan = make(chan struct{})
	h.server = grpc.NewServer()
	go h.start()
}

func (h *Hypervisor) stop() {
	log.DefaultLogger().Infof("VSOCK server shutting down ...")
	close(h.stopChan)
	h.server.Stop()
	<-h.doneChan
	h.running = false
	log.DefaultLogger().Infof("VSOCK server shut down.")
}

func (h *Hypervisor) start() {
	log.DefaultLogger().Infof("Starting VSOCK server ...")
	defer close(h.doneChan)
	wait.Until(h.serve, 1*time.Second, h.stopChan)
}

func (h *Hypervisor) serve() {
	// Load the vhost_vsock module on demand.
	if fd, err := os.Open("/dev/vhost-vsock"); err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to open /dev/vhost-vsock.")
		return
	} else {
		fd.Close()
	}
	conn, err := vsock.ListenContextID(vsock.Host, h.port, &vsock.Config{})
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Failed to bind to VSOCK port %v.", h.port)
		return
	}
	defer conn.Close()
	v1.RegisterSystemServer(h.server, system.NewSystemService(h.caManager))
	err = h.server.Serve(conn)
	if err != nil {
		log.DefaultLogger().Reason(err).Error("Failed to listen for VSOCK connections.")
		return
	}
}

func NewVSOCKHypervisorService(port uint32, caManager tls.ClientCAManager) *Hypervisor {
	return &Hypervisor{
		port:      port,
		caManager: caManager,
	}
}
