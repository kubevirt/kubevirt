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

package cmdserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

type agentDataField struct {
	commandKey string
	requested  func(*cmdv1.VMStatsRequest) bool
	setResult  func(*cmdv1.VMStatsResponse, *cmdv1.Response)
}

var agentDataFields = []agentDataField{
	{"guest-get-load", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetLoad != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetLoad = resp }},
	{"guest-get-cpustats", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetCpuStats != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetCpuStats = resp }},
	{"guest-get-diskstats", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetDiskStats != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetDiskStats = resp }},
	{"guest-get-time", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetTime != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetTime = resp }},
	{"guest-get-vcpus", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetVcpus != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetVcpus = resp }},
	{"guest-get-memory-block-info", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetMemoryBlockInfo != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetMemoryBlockInfo = resp }},
	{"guest-get-users", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetUsers != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetUsers = resp }},
	{"guest-get-osinfo", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetOsInfo != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetOsInfo = resp }},
	{"guest-get-disks", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetDisks != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetDisks = resp }},
	{"guest-get-host-name", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetHostName != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetHostName = resp }},
	{"guest-get-timezone", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetTimezone != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetTimezone = resp }},
	{"guest-network-get-route", func(r *cmdv1.VMStatsRequest) bool { return r.GuestNetworkGetRoute != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestNetworkGetRoute = resp }},
	{"guest-network-get-interfaces", func(r *cmdv1.VMStatsRequest) bool { return r.GuestNetworkGetInterfaces != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestNetworkGetInterfaces = resp }},
	{"guest-get-memory-blocks", func(r *cmdv1.VMStatsRequest) bool { return r.GuestGetMemoryBlocks != nil }, func(r *cmdv1.VMStatsResponse, resp *cmdv1.Response) { r.GuestGetMemoryBlocks = resp }},
}

func AgentDataCommandKeys() []string {
	keys := make([]string, 0, len(agentDataFields))
	for _, f := range agentDataFields {
		keys = append(keys, f.commandKey)
	}
	return keys
}

func (l *Launcher) GetVMStats(ctx context.Context, request *cmdv1.VMStatsRequest) (*cmdv1.VMStatsResponse, error) {
	if !l.vmStatsCollectorEnabled {
		return nil, fmt.Errorf("VMStatsCollector feature gate is not enabled")
	}

	start := time.Now()

	hasAgentRequests := false
	requestedAgentCommands := make([]string, 0)
	for _, f := range agentDataFields {
		if f.requested(request) {
			hasAgentRequests = true
			requestedAgentCommands = append(requestedAgentCommands, f.commandKey)
		}
	}

	log.Log.V(2).Infof("GetVMStats called: domainStats=%t, dirtyRate=%t, agentCommands=%v",
		request.GetDomainStats() != nil, request.GetDirtyRate() != nil, requestedAgentCommands)

	response := &cmdv1.VMStatsResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	var errs []string

	if request.GetDomainStats() != nil {
		response.DomainStats, _ = l.GetDomainStats(ctx, &cmdv1.EmptyRequest{})
		if !response.GetDomainStats().GetResponse().GetSuccess() {
			errs = append(errs, fmt.Sprintf("domain stats: %s", response.GetDomainStats().GetResponse().GetMessage()))
		}
	}

	if request.GetDirtyRate() != nil {
		response.DirtyRateStats, _ = l.GetDomainDirtyRateStats(ctx, &cmdv1.EmptyRequest{})
		if !response.GetDirtyRateStats().GetResponse().GetSuccess() {
			errs = append(errs, fmt.Sprintf("dirty rate stats: %s", response.GetDirtyRateStats().GetResponse().GetMessage()))
		}
	}

	if hasAgentRequests {
		response.GuestAgentVersion = &cmdv1.Response{
			Success: true,
			Message: l.domainManager.GetGuestAgentVersion(),
		}

		for _, f := range agentDataFields {
			if !f.requested(request) {
				continue
			}

			cmdStart := time.Now()
			resp := l.fetchAgentData(f.commandKey)
			f.setResult(response, resp)

			if !resp.GetSuccess() {
				errs = append(errs, fmt.Sprintf("agent data %s: %s", f.commandKey, resp.GetMessage()))
			}
			log.Log.V(2).Infof("GetVMStats agent command %s completed in %s", f.commandKey, time.Since(cmdStart))
		}
	}

	if len(errs) > 0 {
		response.Response.Success = false
		response.Response.Message = strings.Join(errs, "; ")
	}

	log.Log.V(2).Infof("GetVMStats completed: total=%s, errors=%d", time.Since(start), len(errs))

	return response, nil
}

func (l *Launcher) fetchAgentData(commandKey string) *cmdv1.Response {
	data, err := l.domainManager.GetAgentData(commandKey)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to get agent data for %s", commandKey)
		return &cmdv1.Response{Success: false, Message: getErrorMessage(err)}
	}
	return &cmdv1.Response{Success: true, Message: data}
}
