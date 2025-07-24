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
package agentpoller

import (
	"math"
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
// Aliases are also used as keys to the store, it does not matter how the keys are named,
// only whether it relates to the right data
const (
	GetFilesystem     AgentCommand = "guest-get-fsinfo"
	GetAgent          AgentCommand = "guest-info"
	GetFSFreezeStatus AgentCommand = "guest-fsfreeze-status"

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
	const agentUpdatedChanBuffer = 10

	return AsyncAgentStore{
		store:        sync.Map{},
		AgentUpdated: make(chan AgentUpdatedEvent, agentUpdatedChanBuffer),
	}
}

// Store saves the value with a key to the storage, when there is a change in data
// it fires up updated event
func (s *AsyncAgentStore) Store(key, value any) {
	oldData, _ := s.store.Load(key)
	updated := (oldData == nil) || !equality.Semantic.DeepEqual(oldData, value)

	s.store.Store(key, value)

	if updated {
		domainInfo := api.DomainGuestInfo{}
		switch key {
		case libvirt.DOMAIN_GUEST_INFO_OS, libvirt.DOMAIN_GUEST_INFO_INTERFACES, GetFSFreezeStatus:
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
	data, ok := s.store.Load(libvirt.DOMAIN_GUEST_INFO_OS)
	osinfo := api.GuestOSInfo{}
	if ok {
		osinfo = data.(api.GuestOSInfo)
	}

	data, ok = s.store.Load(libvirt.DOMAIN_GUEST_INFO_HOSTNAME)
	hostname := ""
	if ok {
		hostname = data.(string)
	}

	data, ok = s.store.Load(libvirt.DOMAIN_GUEST_INFO_TIMEZONE)
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
	data, ok := s.store.Load(libvirt.DOMAIN_GUEST_INFO_INTERFACES)
	if ok {
		return data.([]api.InterfaceStatus)
	}

	return nil
}

// GetGuestOSInfo returns the Guest OS version and architecture
func (s *AsyncAgentStore) GetGuestOSInfo() *api.GuestOSInfo {
	data, ok := s.store.Load(libvirt.DOMAIN_GUEST_INFO_OS)
	if ok {
		osInfo := data.(api.GuestOSInfo)
		return &osInfo
	}

	return nil
}

// GetGA returns guest agent record with its version if present
func (s *AsyncAgentStore) GetGA() AgentInfo {
	data, ok := s.store.Load(GetAgent)
	agent := AgentInfo{}
	if !ok {
		return agent
	}

	agent = data.(AgentInfo)
	return agent
}

// GetFSFreezeStatus returns the Guest fsfreeze status
func (s *AsyncAgentStore) GetFSFreezeStatus() *api.FSFreeze {
	data, ok := s.store.Load(GetFSFreezeStatus)
	if !ok {
		return nil
	}

	fsfreezeStatus := data.(api.FSFreeze)
	return &fsfreezeStatus
}

// GetFS returns the filesystem list limited to the limit set
// set limit to -1 to return the whole list
func (s *AsyncAgentStore) GetFS(limit int) []api.Filesystem {
	data, ok := s.store.Load(GetFilesystem)
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
	data, ok := s.store.Load(libvirt.DOMAIN_GUEST_INFO_USERS)
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

	// InfoTypes defines the type of guest info to fetch (if applicable)
	InfoTypes libvirt.DomainGuestInfoTypes

	// CallTick is how often to call this set of commands
	CallTick time.Duration
}

// Poll is the call to the guestagent.
func (p *PollerWorker) Poll(execFunc func(), closeChan chan struct{}, initialInterval time.Duration) {
	// Do the first round to fill the cache immediately.
	execFunc()

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
			execFunc()
		}
		if pollInterval < pollMaxInterval {
			pollInterval = incrementPollInterval(pollInterval, pollMaxInterval)
			ticker.Reset(pollInterval)
		}
	}
}

func incrementPollInterval(interval, maxInterval time.Duration) time.Duration {
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
	connection cli.Connection,
	vmiUID types.UID,
	domainName string,
	store *AsyncAgentStore,
	qemuAgentSysInterval time.Duration,
	qemuAgentFileInterval time.Duration,
	qemuAgentUserInterval time.Duration,
	qemuAgentVersionInterval time.Duration,
	qemuAgentFSFreezeStatusInterval time.Duration,
) *AgentPoller {
	return &AgentPoller{
		Connection: connection,
		VmiUID:     vmiUID,
		domainName: domainName,
		agentStore: store,
		workers: []PollerWorker{
			// Polling for QEMU agent commands
			{
				CallTick:      qemuAgentVersionInterval,
				AgentCommands: []AgentCommand{GetAgent},
			},
			{
				CallTick:      qemuAgentFileInterval,
				AgentCommands: []AgentCommand{GetFilesystem},
			},
			{
				CallTick:      qemuAgentFSFreezeStatusInterval,
				AgentCommands: []AgentCommand{GetFSFreezeStatus},
			},
			// Polling for guest info API
			{
				CallTick: qemuAgentSysInterval,
				InfoTypes: libvirt.DOMAIN_GUEST_INFO_INTERFACES |
					libvirt.DOMAIN_GUEST_INFO_OS |
					libvirt.DOMAIN_GUEST_INFO_HOSTNAME |
					libvirt.DOMAIN_GUEST_INFO_TIMEZONE,
			},
			{
				CallTick:  qemuAgentUserInterval,
				InfoTypes: libvirt.DOMAIN_GUEST_INFO_USERS,
			},
		},
	}
}

// Start the poller workers and libvirt API operations
func (p *AgentPoller) Start() {
	if p.agentDone != nil {
		return
	}
	p.agentDone = make(chan struct{})

	for _, worker := range p.workers {
		if len(worker.AgentCommands) != 0 {
			log.Log.Infof("Starting agent poller with commands: %v", worker.AgentCommands)
		} else {
			log.Log.Infof("Starting agent poller with API operations: %v", worker.InfoTypes)
		}

		go worker.Poll(func() {
			if len(worker.AgentCommands) != 0 {
				executeAgentCommands(worker.AgentCommands, p)
			} else {
				fetchAndStoreGuestInfo(worker.InfoTypes, p)
			}
		}, p.agentDone, pollInitialInterval)
	}
}

// Stop all poller workers and libvirt API operations
func (p *AgentPoller) Stop() {
	if p.agentDone != nil {
		close(p.agentDone)
		p.agentDone = nil
	}
}

// TODO: Remove all commands with this function
//
// GET_FSFREEZE_STATUS - This is not implemented in libvirt API and won't be
// implemented (KubeVirt is expected to provide its own implementation for it).
//
// GET_FILESYSTEM - We are missing busType field in the response, which will
// be included in libvirt 11.2 upstream later (https://gitlab.com/libvirt/libvirt-go-module/-/issues/18).
//
// GET_AGENT - According to libvirt engineers this command shouldn't be used
// by KubeVirt, because it provides irrelevant information (version and supported commands).
func executeAgentCommands(commands []AgentCommand, agentPoller *AgentPoller) {
	log.Log.Infof("Polling command: %v", commands)

	for _, command := range commands {
		cmdResult, err := agentPoller.Connection.QemuAgentCommand(`{"execute":"`+string(command)+`"}`, agentPoller.domainName)
		if err != nil {
			// skip the command on error, it is not vital
			continue
		}

		switch command {
		case GetFSFreezeStatus:
			fsfreezeStatus, err := ParseFSFreezeStatus(cmdResult)
			if err != nil {
				log.Log.Errorf("Cannot parse guest agent fsfreeze status %s", err.Error())
				continue
			}
			agentPoller.agentStore.Store(GetFSFreezeStatus, fsfreezeStatus)
		case GetFilesystem:
			filesystems, err := parseFilesystem(cmdResult)
			if err != nil {
				log.Log.Errorf("Cannot parse guest agent filesystem %s", err.Error())
				continue
			}
			agentPoller.agentStore.Store(GetFilesystem, filesystems)
		case GetAgent:
			agent, err := parseAgent(cmdResult)
			if err != nil {
				log.Log.Errorf("Cannot parse guest agent information %s", err.Error())
				continue
			}
			agentPoller.agentStore.Store(GetAgent, agent)
		}
	}
}

func fetchAndStoreGuestInfo(infoTypes libvirt.DomainGuestInfoTypes, agentPoller *AgentPoller) {
	log.Log.Infof("Polling API operations: %v", infoTypes)

	domain, err := agentPoller.Connection.LookupDomainByName(agentPoller.domainName)
	if err != nil {
		log.Log.Errorf("Domain lookup failed: %v", err)
		return
	}

	// Ignoring errors from domain.Free() is safe because it
	// only fails if called multiple times or if the domain object
	// is invalid, neither of which is the case here.
	defer func() { _ = domain.Free() }()

	guestInfo, err := domain.GetGuestInfo(infoTypes, 0)
	if err != nil {
		log.Log.Errorf("Fetching guest info failed: %v", err)
		return
	}

	if infoTypes&libvirt.DOMAIN_GUEST_INFO_INTERFACES != 0 {
		agentPoller.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_INTERFACES, convertToInterfaces(guestInfo))
	}

	if infoTypes&libvirt.DOMAIN_GUEST_INFO_OS != 0 {
		agentPoller.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, convertToOSInfo(guestInfo))
	}

	if infoTypes&libvirt.DOMAIN_GUEST_INFO_HOSTNAME != 0 {
		agentPoller.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_HOSTNAME, guestInfo.Hostname)
	}

	if infoTypes&libvirt.DOMAIN_GUEST_INFO_TIMEZONE != 0 {
		agentPoller.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_TIMEZONE, convertToTimezone(guestInfo))
	}

	if infoTypes&libvirt.DOMAIN_GUEST_INFO_USERS != 0 {
		agentPoller.agentStore.Store(libvirt.DOMAIN_GUEST_INFO_USERS, convertToUsers(guestInfo))
	}
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

func convertToIPAddresses(ipAddresses []libvirt.DomainGuestInfoIPAddress) (primaryIP string, allIPs []string) {
	var interfaceIPs []string
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
				LoginTime: (time.Duration(safeConvertToInt64(user.LoginTime)) * time.Millisecond).Seconds(),
			})
		}
	}
	return users
}

func safeConvertToInt64(value uint64) int64 {
	if value > math.MaxInt64 {
		log.Log.Errorf("Conversion overflow detected: %v", value)
		return 0
	}
	return int64(value)
}
