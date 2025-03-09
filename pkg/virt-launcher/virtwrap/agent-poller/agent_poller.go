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
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"libvirt.org/go/libvirt"

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
	GET_OSINFO          AgentCommand = "guest-get-osinfo"
	GET_HOSTNAME        AgentCommand = "guest-get-host-name"
	GET_INTERFACES      AgentCommand = "guest-network-get-interfaces"
	GET_TIMEZONE        AgentCommand = "guest-get-timezone"
	GET_USERS           AgentCommand = "guest-get-users"
	GET_FILESYSTEM      AgentCommand = "guest-get-fsinfo"
	GET_AGENT           AgentCommand = "guest-info"
	GET_FSFREEZE_STATUS AgentCommand = "guest-fsfreeze-status"

	pollInitialInterval = 10 * time.Second
)

// AgentUpdatedEvent fire up when data is changes in the store
type AgentUpdatedEvent struct {
	DomainInfo api.DomainGuestInfo
}

// AsyncAgentStore stores the agent data converted to api domain objects
// it offers methods to get the data and fire up an event when there
// is a change of the data
type AsyncAgentStore struct {
	store        sync.Map
	AgentUpdated chan AgentUpdatedEvent
}

// NewAsyncAgentStore creates new agent store
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
	updated := (oldData == nil) || !equality.Semantic.DeepEqual(oldData, value)

	s.store.Store(key, value)

	if updated {
		domainInfo := api.DomainGuestInfo{}
		switch key {
		case GET_OSINFO, GET_INTERFACES, GET_FSFREEZE_STATUS:
			domainInfo.OSInfo = s.GetGuestOSInfo()
			domainInfo.Interfaces = s.GetInterfaceStatus()
			domainInfo.FSFreezeStatus = s.GetFSFreezeStatus()
		}

		s.AgentUpdated <- AgentUpdatedEvent{
			DomainInfo: domainInfo,
		}
	}
}

// GetSysInfo returns the sysInfo information packed together.
// Sysinfo comprises of:
//   - Guest Hostname
//   - Guest OS version and architecture
//   - Guest Timezone
func (s *AsyncAgentStore) GetSysInfo() api.DomainSysInfo {
	data, ok := s.store.Load(GET_OSINFO)
	osinfo := api.GuestOSInfo{}
	if ok {
		osinfo = data.(api.GuestOSInfo)
	}

	data, ok = s.store.Load(GET_HOSTNAME)
	hostname := ""
	if ok {
		hostname = data.(string)
	}

	data, ok = s.store.Load(GET_TIMEZONE)
	timezone := api.Timezone{}
	if ok {
		timezone = data.(api.Timezone)
	}

	return api.DomainSysInfo{
		Hostname: hostname,
		OSInfo:   osinfo,
		Timezone: timezone,
	}
}

// GetInterfaceStatus returns the interfaces Guest Agent reported
func (s *AsyncAgentStore) GetInterfaceStatus() []api.InterfaceStatus {
	data, ok := s.store.Load(GET_INTERFACES)
	if ok {
		return data.([]api.InterfaceStatus)
	}

	return nil
}

// GetGuestOSInfo returns the Guest OS version and architecture
func (s *AsyncAgentStore) GetGuestOSInfo() *api.GuestOSInfo {
	data, ok := s.store.Load(GET_OSINFO)
	if ok {
		osInfo := data.(api.GuestOSInfo)
		return &osInfo
	}

	return nil
}

// GetGA returns guest agent record with its version if present
func (s *AsyncAgentStore) GetGA() AgentInfo {
	data, ok := s.store.Load(GET_AGENT)
	agent := AgentInfo{}
	if !ok {
		return agent
	}

	agent = data.(AgentInfo)
	return agent
}

// GetFSFreezeStatus returns the Guest fsfreeze status
func (s *AsyncAgentStore) GetFSFreezeStatus() *api.FSFreeze {
	data, ok := s.store.Load(GET_FSFREEZE_STATUS)
	if !ok {
		return nil
	}

	fsfreezeStatus := data.(api.FSFreeze)
	return &fsfreezeStatus
}

// GetFS returns the filesystem list limited to the limit set
// set limit to -1 to return the whole list
func (s *AsyncAgentStore) GetFS(limit int) []api.Filesystem {
	data, ok := s.store.Load(GET_FILESYSTEM)
	filesystems := []api.Filesystem{}
	if !ok {
		return filesystems
	}

	filesystems = data.([]api.Filesystem)
	if len(filesystems) <= limit || limit == -1 {
		return filesystems
	}

	limitedFilesystems := make([]api.Filesystem, limit)
	copy(limitedFilesystems, filesystems[:limit])
	return limitedFilesystems
}

// GetUsers return the use list limited to the limit set
// set limit to -1 to return all users
func (s *AsyncAgentStore) GetUsers(limit int) []api.User {
	data, ok := s.store.Load(GET_USERS)
	users := []api.User{}
	if !ok {
		return users
	}

	users = data.([]api.User)
	if len(users) <= limit || limit == -1 {
		return users
	}

	limitedUsers := make([]api.User, limit)
	copy(limitedUsers, users[:limit])
	return limitedUsers
}

// PollerWorker collects the data from the guest agent
// only unique items are stored as configuration
type PollerWorker struct {
	// AgentCommands is a list of commands executed on the guestAgent
	AgentCommands []AgentCommand
	// CallTick is how often to call this set of commands
	CallTick time.Duration
}

type agentCommandsExecutor func(commands []AgentCommand)

// Poll is the call to the guestagent.
func (p *PollerWorker) Poll(execAgentCommands agentCommandsExecutor, closeChan chan struct{}, initialInterval time.Duration) {
	log.Log.Infof("Polling command: %v", p.AgentCommands)

	// Do the first round to fill the cache immediately.
	execAgentCommands(p.AgentCommands)

	pollMaxInterval := p.CallTick
	pollInterval := pollMaxInterval
	if initialInterval < pollMaxInterval {
		pollInterval = initialInterval
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-closeChan:
			return
		case <-ticker.C:
			execAgentCommands(p.AgentCommands)
		}
		if pollInterval < pollMaxInterval {
			pollInterval = incrementPollInterval(pollInterval, pollMaxInterval)
			ticker.Reset(pollInterval)
		}
	}
}

func incrementPollInterval(interval time.Duration, maxInterval time.Duration) time.Duration {
	interval *= 2
	if interval > maxInterval {
		interval = maxInterval
	}
	return interval
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
	qemuAgentSysInterval time.Duration,
	qemuAgentFileInterval time.Duration,
	qemuAgentUserInterval time.Duration,
	qemuAgentVersionInterval time.Duration,
	qemuAgentFSFreezeStatusInterval time.Duration,
) *AgentPoller {
	p := &AgentPoller{
		Connection: connecton,
		VmiUID:     vmiUID,
		domainName: domainName,
		agentStore: store,
		workers:    []PollerWorker{},
	}

	// version command group
	p.workers = append(p.workers, PollerWorker{
		CallTick:      qemuAgentVersionInterval,
		AgentCommands: []AgentCommand{GET_AGENT},
	})
	// sys command group
	p.workers = append(p.workers, PollerWorker{
		CallTick:      qemuAgentSysInterval,
		AgentCommands: []AgentCommand{GET_INTERFACES, GET_OSINFO, GET_TIMEZONE, GET_HOSTNAME},
	})
	// filesystem command group
	p.workers = append(p.workers, PollerWorker{
		CallTick:      qemuAgentFileInterval,
		AgentCommands: []AgentCommand{GET_FILESYSTEM},
	})
	// user command group
	p.workers = append(p.workers, PollerWorker{
		CallTick:      qemuAgentUserInterval,
		AgentCommands: []AgentCommand{GET_USERS},
	})
	// fsfreeze command group
	p.workers = append(p.workers, PollerWorker{
		CallTick:      qemuAgentFSFreezeStatusInterval,
		AgentCommands: []AgentCommand{GET_FSFREEZE_STATUS},
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
		go p.workers[i].Poll(func(commands []AgentCommand) {
			for _, command := range commands {
				switch command {
				case GET_FSFREEZE_STATUS, GET_FILESYSTEM, GET_AGENT:
					executeAgentCommands(commands, p.Connection, p.agentStore, p.domainName)
				case GET_INTERFACES, GET_OSINFO, GET_HOSTNAME, GET_TIMEZONE, GET_USERS:
					executeApiOperations(commands, p.Connection, p.agentStore, p.domainName)
				}
			}
		}, p.agentDone, pollInitialInterval)
	}
}

// Stop all poller workers
func (p *AgentPoller) Stop() {
	if p.agentDone != nil {
		close(p.agentDone)
		p.agentDone = nil
	}
}

func executeAgentCommands(commands []AgentCommand, con cli.Connection, agentStore *AsyncAgentStore, domainName string) {
	for _, command := range commands {
		cmdResult, err := con.QemuAgentCommand(`{"execute":"`+string(command)+`"}`, domainName)
		if err != nil {
			// skip the command on error, it is not vital
			continue
		}

		switch command {
		case GET_FSFREEZE_STATUS:
			fsfreezeStatus, err := ParseFSFreezeStatus(cmdResult)
			if err != nil {
				log.Log.Errorf("Cannot parse guest agent fsfreeze status %s", err.Error())
				continue
			}
			agentStore.Store(GET_FSFREEZE_STATUS, fsfreezeStatus)
		case GET_FILESYSTEM:
			filesystems, err := parseFilesystem(cmdResult)
			if err != nil {
				log.Log.Errorf("Cannot parse guest agent filesystem %s", err.Error())
				continue
			}
			agentStore.Store(GET_FILESYSTEM, filesystems)
		case GET_AGENT:
			agent, err := parseAgent(cmdResult)
			if err != nil {
				log.Log.Errorf("Cannot parse guest agent information %s", err.Error())
				continue
			}
			agentStore.Store(GET_AGENT, agent)
		}
	}
}

func executeApiOperations(commands []AgentCommand, con cli.Connection, agentStore *AsyncAgentStore, domainName string) {
	for _, command := range commands {
		switch command {
		case GET_INTERFACES:
			if guestInfo, err := getGuestInfo(libvirt.DOMAIN_GUEST_INFO_INTERFACES, con, domainName); err == nil {
				agentStore.Store(GET_INTERFACES, convertToInterfaces(guestInfo))
			}
		case GET_OSINFO:
			if guestInfo, err := getGuestInfo(libvirt.DOMAIN_GUEST_INFO_OS, con, domainName); err == nil {
				agentStore.Store(GET_OSINFO, convertToOSInfo(guestInfo))
			}
		case GET_HOSTNAME:
			if guestInfo, err := getGuestInfo(libvirt.DOMAIN_GUEST_INFO_HOSTNAME, con, domainName); err == nil {
				agentStore.Store(GET_HOSTNAME, guestInfo.Hostname)
			}
		case GET_TIMEZONE:
			if guestInfo, err := getGuestInfo(libvirt.DOMAIN_GUEST_INFO_TIMEZONE, con, domainName); err == nil {
				agentStore.Store(GET_TIMEZONE, convertToTimezone(guestInfo))
			}
		case GET_USERS:
			if guestInfo, err := getGuestInfo(libvirt.DOMAIN_GUEST_INFO_USERS, con, domainName); err == nil {
				agentStore.Store(GET_USERS, convertToUsers(guestInfo))
			}
		}
	}
}

func getGuestInfo(infoType libvirt.DomainGuestInfoTypes, con cli.Connection, domainName string) (*libvirt.DomainGuestInfo, error) {
	domain, err := con.LookupDomainByName(domainName)
	if err != nil {
		log.Log.Errorf("Failed to lookup domain (%s): %v", domainName, err)
		return nil, err
	}

	guestInfo, err := domain.GetGuestInfo(infoType, 0)
	if err != nil {
		log.Log.Errorf("Failed to get guest info (%d): %v", infoType, err)
		return nil, err
	}
	return guestInfo, nil
}

func convertToInterfaces(guestInfo *libvirt.DomainGuestInfo) []api.InterfaceStatus {
	var interfaceStatuses []api.InterfaceStatus
	if guestInfo.Interfaces != nil {
		for _, netInterface := range guestInfo.Interfaces {
			if netInterface.Name == "lo" {
				continue
			}

			interfaceIP, interfaceIPs := convertToIPAddresses(netInterface.Addrs)
			interfaceStatuses = append(interfaceStatuses, api.InterfaceStatus{
				Mac:           netInterface.Hwaddr,
				Ip:            interfaceIP,
				IPs:           interfaceIPs,
				InterfaceName: netInterface.Name,
			})
		}
	}
	return interfaceStatuses
}

func convertToIPAddresses(ipAddresses []libvirt.DomainGuestInfoIPAddress) (string, []string) {
	interfaceIPs := []string{}
	var interfaceIP string

	for _, ipAddr := range ipAddresses {
		ip := ipAddr.Addr

		// Prefer ipv4 as the main interface IP
		if ipAddr.Type == "ipv4" && interfaceIP == "" {
			interfaceIP = ip
		}

		interfaceIPs = append(interfaceIPs, ip)
	}

	// If no ipv4 interface was found, set any IP as the main IP of interface
	if interfaceIP == "" && len(interfaceIPs) > 0 {
		interfaceIP = interfaceIPs[0]
	}
	return interfaceIP, interfaceIPs
}

func convertToOSInfo(guestInfo *libvirt.DomainGuestInfo) api.GuestOSInfo {
	guestInfoOS := api.GuestOSInfo{}
	if guestInfo.OS != nil {
		guestInfoOS = api.GuestOSInfo{
			Name:          guestInfo.OS.Name,
			KernelRelease: guestInfo.OS.KernelRelease,
			Version:       guestInfo.OS.Version,
			PrettyName:    guestInfo.OS.PrettyName,
			VersionId:     guestInfo.OS.VersionID,
			KernelVersion: guestInfo.OS.KernelVersion,
			Machine:       guestInfo.OS.Machine,
			Id:            guestInfo.OS.ID,
		}
	}
	return guestInfoOS
}

func convertToTimezone(guestInfo *libvirt.DomainGuestInfo) api.Timezone {
	timezone := api.Timezone{}
	if guestInfo.TimeZone != nil {
		timezone = api.Timezone{
			Zone:   guestInfo.TimeZone.Name,
			Offset: guestInfo.TimeZone.Offset,
		}
	}
	return timezone
}

func convertToUsers(guestInfo *libvirt.DomainGuestInfo) []api.User {
	var users []api.User
	if guestInfo.Users != nil {
		for _, user := range guestInfo.Users {
			users = append(users, api.User{
				Name:      user.Name,
				Domain:    user.Domain,
				LoginTime: float64(user.LoginTime),
			})
		}
	}
	return users
}
