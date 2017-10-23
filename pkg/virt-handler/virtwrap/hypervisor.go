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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virtwrap

import (
	"time"
	// Currenlty only libvirt is supported and there is not desire
	// so-far to support anything else, but let's have a clear
	// design.
	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/api"
	isol "kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/libvirt"
)

// Defining the hypervisor interface.
//
// Each hypervisor drivers will have to implmement this interface. It
// will represents the only public solution for the other components
// to comunicate with internal hypervisor functionalities

type Hypervisor interface {
	Close() (int, error)
	MonitorConnection(interval time.Duration)

	// XXX: replace VirDomain by Guest interface when all
	// implemented

	// Find and retourn `Guest` from its name
	LookupGuestByName(name string) (libvirt.VirDomain, error)
	// Defining a domain spec/configuration
	DefineGuestSpec(vm *v1.VirtualMachine, spec interface{}) (libvirt.VirDomain, error)
	UpdateGuestSpec(vm *v1.VirtualMachine, isolation *isol.IsolationResult) (*api.DomainSpec, error)
	// Returns list of Guests
	ListAllGuests(actives bool, inactives bool) ([]libvirt.VirDomain, error)
	// Register guests events callback life cycle
	RegisterGuestEventLifecycle(callback interface{}) error
}

type Guest interface{}

// Returns a new Hypervisor Connection
//
// Initializes connection to hypervisor based on `uri`, `user`, `pass`.
func NewHypervisorConnection(uri string, user string, pass string) (libvirt.Connection, error) {
	// XXX: The return of this function will be at some point Hypervisor
	// interface

	// Currently only libvirt is supported, no need to add
	// complexity.
	return libvirt.NewConnection(uri, user, pass)
}

// Monitors the hypervisor connection to the daemon.
//
// The monitor will check by `interval` if the connection is still
// alive.
func MonitorHypervisorConnection(h libvirt.Connection, interval time.Duration) {
	// XXX: h should type ot Hypervisor interface

	// Currently only libvirt is supported, no need to add
	// complexity.
	h.MonitorConnection(interval)
}

// Define Guest in hypervisor.
//
// Based on `spec` request, definning `Guest` in hypervisor.
func DefineGuest(h libvirt.Connection, vm *v1.VirtualMachine, spec interface{}) (libvirt.VirDomain, error) {
	// XXX: h should type ot Hypervisor interface, should return a
	// Guest

	// Currently only libvirt is supported, no need to add
	// complexity.
	return h.DefineGuestSpec(vm, spec.(api.DomainSpec))
}

// Update Guest spec in hypervisor.
//
// Acording to the `isolation`, keep the guest spec updated.
func UpdateGuest(h libvirt.Connection, vm *v1.VirtualMachine, isolation *isol.IsolationResult) (*api.DomainSpec, error) {
	return h.UpdateGuestSpec(vm, isolation)
}
