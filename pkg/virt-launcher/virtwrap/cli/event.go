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

package cli

import (
	"fmt"

	"libvirt.org/go/libvirt"
)

type DomainEventDeviceRemoved struct {
	connection     Connection
	domain         VirDomain
	registrationID int
	callback       libvirt.DomainEventDeviceRemovedCallback
	eventChan      <-chan interface{}
}

func NewDomainEventDeviceRemoved(
	connection Connection,
	domain VirDomain,
	callback libvirt.DomainEventDeviceRemovedCallback, eventChan <-chan interface{}) *DomainEventDeviceRemoved {

	return &DomainEventDeviceRemoved{
		connection: connection,
		domain:     domain,
		callback:   callback,
		eventChan:  eventChan,
	}
}

func (c *DomainEventDeviceRemoved) Register() error {
	id, err := c.connection.VolatileDomainEventDeviceRemovedRegister(c.domain, c.callback)
	if err != nil {
		return fmt.Errorf("register callback failure: %v", err)
	}
	c.registrationID = id
	return nil
}

func (c *DomainEventDeviceRemoved) Deregister() error {
	if c.registrationID == 0 {
		return fmt.Errorf("deregister callback failure: no registration occur")
	}

	if err := c.connection.DomainEventDeregister(c.registrationID); err != nil {
		return fmt.Errorf("deregister callback failure: %v", err)
	}
	return nil
}

func (c *DomainEventDeviceRemoved) EventChannel() <-chan interface{} {
	return c.eventChan
}
