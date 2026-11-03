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

package domainpublisher

import (
	"errors"
	"fmt"

	"libvirt.org/go/libvirt"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type EventHandler interface {
	OnLifecycle(*libvirt.DomainEventLifecycle)
	OnJobCompleted(*libvirt.DomainEventJobCompleted)
	OnAgentLifecycle(*libvirt.DomainEventAgentLifecycle)
	Resync()
}

// RegisterLibvirtEvents delivers the libvirt events of the connection to the handler.
func RegisterLibvirtEvents(conn cli.Connection, handler EventHandler) error {
	reconnect := make(chan bool, 10)
	conn.SetReconnectChan(reconnect)
	go func() {
		for range reconnect {
			log.Log.Info("Libvirt reconnected, resyncing the domain")
			handler.Resync()
		}
	}()

	err := errors.Join(
		conn.DomainEventLifecycleRegister(deliver(handler.OnLifecycle)),
		conn.DomainEventJobCompletedRegister(deliver(handler.OnJobCompleted)),
		conn.AgentEventLifecycleRegister(deliver(handler.OnAgentLifecycle)),
		conn.DomainEventDeviceAddedRegister(deliver(resyncOn[*libvirt.DomainEventDeviceAdded](handler))),
		conn.DomainEventDeviceRemovedRegister(deliver(resyncOn[*libvirt.DomainEventDeviceRemoved](handler))),
		conn.DomainEventMemoryDeviceSizeChangeRegister(deliver(resyncOn[*libvirt.DomainEventMemoryDeviceSizeChange](handler))),
	)
	if err != nil {
		return fmt.Errorf("failed to register libvirt event callbacks: %w", err)
	}
	log.Log.Info("Registered libvirt event callbacks")
	return nil
}

func deliver[E any](handle func(E)) func(*libvirt.Connect, *libvirt.Domain, E) {
	return func(_ *libvirt.Connect, _ *libvirt.Domain, event E) {
		logLibvirtEvent(event)
		handle(event)
	}
}

func resyncOn[E any](handler EventHandler) func(E) {
	return func(E) { handler.Resync() }
}

func logLibvirtEvent(event any) {
	if job, ok := event.(*libvirt.DomainEventJobCompleted); ok {
		log.Log.Infof("Libvirt job completed event received: type %v, operation %v, success reported %t, succeeded %t",
			job.Info.Type, job.Info.Operation, job.Info.JobSuccessSet, job.Info.JobSuccess)
		return
	}
	log.Log.Infof("Libvirt event received: %T %v", event, event)
}
