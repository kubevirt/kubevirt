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
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type AgentUpdateEvent struct {
	InterfaceStatuses *[]api.InterfaceStatus
	DomainName        string
}

type AgentPoller struct {
	Connection      cli.Connection
	VmiUID          types.UID
	domainData      *DomainData
	agentDone       chan struct{}
	domainUpdate    chan *api.Domain
	pollTime        time.Duration
	agentUpdateChan chan AgentUpdateEvent
}

type DomainData struct {
	name       string
	aliasByMac map[string]string
	interfaces []api.InterfaceStatus
}

// Result for json unmarshalling
type Result struct {
	Interfaces []Interface `json:"return"`
}

// Interface for json unmarshalling
type Interface struct {
	MAC  string `json:"hardware-address"`
	IPs  []IP   `json:"ip-addresses"`
	Name string `json:"name"`
}

// IP for json unmarshalling
type IP struct {
	IP     string `json:"ip-address"`
	Type   string `json:"ip-address-type"`
	Prefix int    `json:"prefix"`
}

func CreatePoller(connecton cli.Connection, vmiUID types.UID, agentUpdateChan chan AgentUpdateEvent, qemuAgentPollerInterval *time.Duration) *AgentPoller {
	p := &AgentPoller{
		Connection:      connecton,
		VmiUID:          vmiUID,
		pollTime:        *qemuAgentPollerInterval,
		agentUpdateChan: agentUpdateChan,
		domainUpdate:    make(chan *api.Domain, 10),
	}
	return p
}

func (p *AgentPoller) Start() {
	if p.agentDone != nil {
		return
	}
	p.agentDone = make(chan struct{})
	go func() {
		for {
			log.Log.Info("Qemu agent poller started")
			select {
			case <-p.agentDone:
				log.Log.Info("Qemu agent poller stopped")
				return // stop polling
			case domain := <-p.domainUpdate:
				p.domainData = p.createDomainData(domain)
			case <-time.After(time.Duration(p.pollTime) * time.Second):
				cmdResult, err := p.pollQemuAgent(p.domainData.name)
				if err != nil {
					log.Log.Reason(err).Error("Qemu agent poller error")
					continue
				}
				interfaceStatuses := p.GetInterfaceStatuses(cmdResult)
				if !reflect.DeepEqual(p.domainData.interfaces, interfaceStatuses) {
					p.domainData.interfaces = interfaceStatuses

					agentUpdateEvent := AgentUpdateEvent{
						InterfaceStatuses: &interfaceStatuses,
						DomainName:        p.domainData.name,
					}
					p.agentUpdateChan <- agentUpdateEvent
				}
			}
		}
	}()
}

func (p *AgentPoller) Stop() {
	if p.agentDone != nil {
		close(p.agentDone)
		p.agentDone = nil
	}
}

func (p *AgentPoller) GetInterfaceStatuses(cmdResult string) []api.InterfaceStatus {
	parsedResult := parseAgentReplyToJson(cmdResult)
	interfaceStatuses := calculateInterfaceStatusesFromAgentJson(parsedResult)
	return p.mergeAgentStatusesWithDomainData(interfaceStatuses)
}

func (p *AgentPoller) UpdateDomain(domain *api.Domain) {
	if domain != nil {
		select {
		case p.domainUpdate <- domain:
		default:
			log.Log.Error("Failed to upate agent poller domain info")
		}
	}
}

func (p *AgentPoller) createDomainData(domain *api.Domain) *DomainData {
	aliasByMac := map[string]string{}
	for _, ifc := range domain.Spec.Devices.Interfaces {
		mac := ifc.MAC.MAC
		alias := ifc.Alias.Name
		aliasByMac[mac] = alias
	}
	return &DomainData{
		name:       domain.Spec.Name,
		aliasByMac: aliasByMac,
	}
}

func (p *AgentPoller) pollQemuAgent(domainName string) (string, error) {
	cmdResult, err := p.Connection.QemuAgentCommand("{\"execute\":\"guest-network-get-interfaces\"}", domainName)
	return cmdResult, err
}

func parseAgentReplyToJson(agentReply string) *Result {
	result := Result{}
	json.Unmarshal([]byte(agentReply), &result)
	return &result
}

func (p *AgentPoller) mergeAgentStatusesWithDomainData(interfaceStatuses []api.InterfaceStatus) []api.InterfaceStatus {
	aliasesCoveredByAgent := []string{}
	// Add alias from domain to interfaceStatus
	for i, interfaceStatus := range interfaceStatuses {
		if alias, exists := p.domainData.aliasByMac[interfaceStatus.Mac]; exists {
			interfaceStatuses[i].Name = alias
			aliasesCoveredByAgent = append(aliasesCoveredByAgent, alias)
		}
	}

	// If interface present in domain was not found in interfaceStatuses, add it
	for mac, alias := range p.domainData.aliasByMac {
		isCoveredByAgentData := false
		for _, coveredAlias := range aliasesCoveredByAgent {
			if alias == coveredAlias {
				isCoveredByAgentData = true
				break
			}
		}
		if !isCoveredByAgentData {
			interfaceStatuses = append(interfaceStatuses,
				api.InterfaceStatus{
					Mac:  mac,
					Name: alias,
				},
			)
		}
	}
	return interfaceStatuses
}

func calculateInterfaceStatusesFromAgentJson(agentResult *Result) []api.InterfaceStatus {
	interfaceStatuses := []api.InterfaceStatus{}
	for _, ifc := range agentResult.Interfaces {
		if ifc.Name == "lo" {
			continue
		}
		interfaceIP, interfaceIPs := extractIps(ifc.IPs)
		interfaceStatuses = append(interfaceStatuses, api.InterfaceStatus{
			Mac:           ifc.MAC,
			Ip:            interfaceIP,
			IPs:           interfaceIPs,
			InterfaceName: ifc.Name,
		})
	}
	return interfaceStatuses
}

func extractIps(ipAddresses []IP) (string, []string) {
	interfaceIPs := []string{}
	var interfaceIP string
	for _, ipAddr := range ipAddresses {
		ip := fmt.Sprintf("%s/%d", ipAddr.IP, ipAddr.Prefix)
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
