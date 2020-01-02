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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package agentpoller

import (
	"reflect"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

// AgentCommand is a command executable on guest agent
type AgentCommand string

// Aliases for commands executed on guest agent
// TODO: when updated to libvirt 5.6.0 this can change to libvirt types
// Aliases are also used as keys to the store, it does not matter how the keys are named,
// only whether it relates to the right data
const (
	GET_OSINFO     AgentCommand = "guest-get-osinfo"
	GET_INTERFACES AgentCommand = "guest-network-get-interfaces"
)

// AgentUpdatedEvent fire up when data is changes in the store
type AgentUpdatedEvent struct {
	Type       AgentCommand
	DomainInfo api.DomainGuestInfo
}

// AgentStore stores the agent data converted to api domain objects
// it offers methods to get the data and fire up an event when there
// is a change of the data
type AsyncAgentStore struct {
	store        sync.Map
	AgentUpdated chan AgentUpdatedEvent
}

// NewAgentAstore creates new agent store
func NewAsyncAgentStore() AsyncAgentStore {
	return AsyncAgentStore{
		store:        sync.Map{},
		AgentUpdated: make(chan AgentUpdatedEvent, 10),
	}
}

// Store saves the value with a key to the storage, when there is a change in data
// it fires up updated event
func (s *AsyncAgentStore) Store(key AgentCommand, value interface{}) {

	oldData, _ := s.store.Load(key)
	updated := (oldData == nil) || !reflect.DeepEqual(oldData, value)

	s.store.Store(key, value)

	if updated {
		domainInfo := api.DomainGuestInfo{}
		// Fill only updated part of the domainInfo
		// not everything have to be watched for
		switch key {
		case GET_OSINFO:
			info := value.(api.GuestOSInfo)
			domainInfo.OSInfo = &info
		case GET_INTERFACES:
			domainInfo.Interfaces = value.([]api.InterfaceStatus)
		}

		s.AgentUpdated <- AgentUpdatedEvent{
			Type:       key,
			DomainInfo: domainInfo,
		}
	}
}

// PollerWorker collects the data from the guest agent
// only unique items are stored as configuration
type PollerWorker struct {
	// AgentCommands is a list of commands executed on the guestAgent
	AgentCommands []AgentCommand
	// CallTick is how often to call this set of commands
	CallTick time.Duration
}

// Poll is the call to the guestagent
// TODO: with libvirt 5.6.0 direct call to agent can be replaced with call to libvirt
// Domain.GetGuestInfo
func (p *PollerWorker) Poll(con cli.Connection, agentStore *AsyncAgentStore, domainName string, closeChan chan struct{}) {
	ticker := time.NewTicker(time.Second * p.CallTick)

	log.Log.Infof("Polling command: %v", p.AgentCommands)

	// poller, used as a workaround for golang ticker
	// ticker does not do first tick immediately, but after period
	poll := func(commands []AgentCommand) {
		for _, command := range p.AgentCommands {
			// replace with direct call to libvirt function when 5.6.0 is available
			cmdResult, err := con.QemuAgentCommand(`{"execute":"`+string(command)+`"}`, domainName)
			if err != nil {
				// skip the command on error, it is not vital
				continue
			}

			// parse the json data and convert to domain api
			// TODO: for libvirt 5.6.0 json conversion deprecated
			switch command {
			case GET_INTERFACES:
				interfaces, err := parseInterfaces(cmdResult)
				if err != nil {
					log.Log.Errorf("Cannot parse guest agent interface %s", err.Error())
				}
				agentStore.Store(GET_INTERFACES, interfaces)
			case GET_OSINFO:
				osInfo, err := parseGuestOSInfo(cmdResult)
				if err != nil {
					log.Log.Errorf("Cannot parse guest agent guestosinfo %s", err.Error())
				}
				agentStore.Store(GET_OSINFO, osInfo)

			}

		}
	}

	// do the first round to fill the cache immediately
	poll(p.AgentCommands)

	for {
		select {
		case <-closeChan:
			ticker.Stop()
			return
		case <-ticker.C:
			poll(p.AgentCommands)
		}
	}
}

type AgentPoller struct {
	Connection cli.Connection
	VmiUID     types.UID
	domainName string
	agentDone  chan struct{}
	workers    []PollerWorker
	agentStore *AsyncAgentStore
}

// CreatePoller creates the new structure that holds guest agent pollers
func CreatePoller(
	connecton cli.Connection,
	vmiUID types.UID,
	domainName string,
	store *AsyncAgentStore,
	qemuAgentPollerInterval time.Duration,
) *AgentPoller {
	p := &AgentPoller{
		Connection: connecton,
		VmiUID:     vmiUID,
		domainName: domainName,
		agentStore: store,
		workers:    []PollerWorker{},
	}

	// this have to be done via configuration passed in, for now this is OK
	p.workers = append(p.workers, PollerWorker{
		CallTick:      qemuAgentPollerInterval,
		AgentCommands: []AgentCommand{GET_INTERFACES, GET_OSINFO},
	})

	return p
}

// Start the poller workers
func (p *AgentPoller) Start() {
	if p.agentDone != nil {
		return
	}
	p.agentDone = make(chan struct{})

	for i := 0; i < len(p.workers); i++ {
		log.Log.Infof("Starting agent poller with commands: %v", p.workers[i].AgentCommands)
		go p.workers[i].Poll(p.Connection, p.agentStore, p.domainName, p.agentDone)
	}
}

// Stop all poller workers
func (p *AgentPoller) Stop() {
	if p.agentDone != nil {
		close(p.agentDone)
		p.agentDone = nil
	}
}
