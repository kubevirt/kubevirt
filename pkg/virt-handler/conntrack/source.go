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

package conntrack

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	exportTimeout = 5 * time.Second
)

type SourceHandler struct {
	ciliumClient ConntrackClient
	mu           sync.Mutex
	sentVMIs     map[types.UID]struct{}
}

func NewSourceHandler(ciliumClient ConntrackClient) *SourceHandler {
	return &SourceHandler{
		ciliumClient: ciliumClient,
		sentVMIs:     make(map[types.UID]struct{}),
	}
}

func (h *SourceHandler) ExportAndSend(vmi *v1.VirtualMachineInstance, socketPath string) error {
	vmiUID := vmi.UID

	h.mu.Lock()
	if _, alreadySent := h.sentVMIs[vmiUID]; alreadySent {
		h.mu.Unlock()
		log.Log.V(3).Infof("Conntrack sync: CT already sent for VMI %s", vmiUID)
		return nil
	}
	h.sentVMIs[vmiUID] = struct{}{}
	h.mu.Unlock()

	log.Log.V(3).Infof("Conntrack sync: starting export for VMI %s", vmiUID)

	ips := extractVMIIPs(vmi)
	if len(ips) == 0 {
		log.Log.Warningf("Conntrack sync: no IPs found for VMI %s", vmiUID)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), exportTimeout)
	defer cancel()

	var allData []byte
	var version byte

	for _, ip := range ips {
		result, err := h.ciliumClient.ExportConntrack(ctx, ip)
		if err != nil {
			log.Log.Warningf("Conntrack sync: failed to export CT for IP %s: %v", ip, err)
			continue
		}
		if len(result.Data) > 0 {
			allData = append(allData, result.Data...)
			if version == 0 {
				version = result.Version
			}
		}
		log.Log.V(3).Infof("Conntrack sync: exported %d bytes for IP %s", len(result.Data), ip)
	}

	if len(allData) == 0 {
		log.Log.V(3).Infof("Conntrack sync: no CT entries to send for VMI %s", vmiUID)
		return nil
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to proxy socket: %w", err)
	}
	defer conn.Close()

	msg := &SyncMessage{
		Version: version,
		Data:    allData,
	}

	encoded := msg.Encode()
	if _, err := conn.Write(encoded); err != nil {
		return fmt.Errorf("failed to send CT data: %w", err)
	}

	log.Log.V(3).Infof("Conntrack sync: sent %d bytes for VMI %s", len(encoded), vmiUID)
	return nil
}

func (h *SourceHandler) HasSentCT(vmiUID types.UID) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	_, sent := h.sentVMIs[vmiUID]
	return sent
}

func (h *SourceHandler) Cleanup(vmiUID types.UID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.sentVMIs, vmiUID)

	log.Log.V(3).Infof("Conntrack sync: cleaned up source state for VMI %s", vmiUID)
}

func extractVMIIPs(vmi *v1.VirtualMachineInstance) []string {
	var ips []string
	for _, iface := range vmi.Status.Interfaces {
		for _, ip := range iface.IPs {
			if ip != "" {
				ips = append(ips, ip)
			}
		}
	}
	return ips
}
