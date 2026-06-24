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

package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/monitoring/metrics/virt-handler/collector"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

type VMStatsResult struct {
	Stats *stats.VMStats `json:"stats,omitempty"`
	Error string         `json:"error,omitempty"`
}

var _ collector.MetricsScraper = &VMStatsScraper{}

type VMStatsScraper struct {
	ch        chan *vmStatsChannelResult
	newClient func(string) (cmdclient.LauncherClient, error)
	request   *cmdv1.VMStatsRequest
}

type vmStatsChannelResult struct {
	key    string
	stats  *stats.VMStats
	errMsg string
}

func NewVMStatsScraper(channelLength int, newClient func(string) (cmdclient.LauncherClient, error), request *cmdv1.VMStatsRequest) *VMStatsScraper {
	return &VMStatsScraper{
		ch:        make(chan *vmStatsChannelResult, channelLength),
		newClient: newClient,
		request:   request,
	}
}

func (s *VMStatsScraper) Scrape(socketFile string, vmi *v1.VirtualMachineInstance) {
	ts := time.Now()
	key := fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)

	cli, err := s.newClient(socketFile)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to connect to cmd client socket")
		s.ch <- &vmStatsChannelResult{key: key, errMsg: err.Error()}
		return
	}
	defer cli.Close()

	vmStats, err := cli.GetVMStats(s.request)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get VM stats")
		s.ch <- &vmStatsChannelResult{key: key, errMsg: err.Error()}
		return
	}

	elapsed := time.Since(ts)
	if elapsed > collector.StatsMaxAge {
		log.Log.Infof("took too long (%v) to collect stats from %s: ignored", elapsed, socketFile)
		s.ch <- &vmStatsChannelResult{key: key, errMsg: fmt.Sprintf("stats collection took too long (%v), exceeded max age", elapsed)}
		return
	}

	s.ch <- &vmStatsChannelResult{key: key, stats: vmStats}
}

func (s *VMStatsScraper) Complete() {
	close(s.ch)
}

func (s *VMStatsScraper) GetValues() map[string]*VMStatsResult {
	results := make(map[string]*VMStatsResult)
	for r := range s.ch {
		results[r.key] = &VMStatsResult{
			Stats: r.stats,
			Error: r.errMsg,
		}
	}
	return results
}

type VMStatsHandler struct {
	vmiStore      cache.Store
	clusterConfig *virtconfig.ClusterConfig
	collector     collector.Collector
}

func NewVMStatsHandler(
	vmiStore cache.Store,
	clusterConfig *virtconfig.ClusterConfig,
	collector collector.Collector,
) *VMStatsHandler {
	return &VMStatsHandler{
		vmiStore:      vmiStore,
		clusterConfig: clusterConfig,
		collector:     collector,
	}
}

func (h *VMStatsHandler) GetVMStats(request *restful.Request, response *restful.Response) {
	if !h.clusterConfig.VMStatsCollectorEnabled() {
		response.WriteError(http.StatusForbidden, fmt.Errorf("VMStatsCollector feature gate is not enabled"))
		return
	}

	statsRequest := buildVMStatsRequestFromQuery(request)
	if *statsRequest == (cmdv1.VMStatsRequest{}) {
		response.WriteError(http.StatusBadRequest, fmt.Errorf("at least one stats category must be requested via query parameters"))
		return
	}

	items := h.vmiStore.List()
	vmis := make([]*v1.VirtualMachineInstance, 0, len(items))
	for _, obj := range items {
		if vmi, ok := obj.(*v1.VirtualMachineInstance); ok {
			vmis = append(vmis, vmi)
		}
	}

	scraper := NewVMStatsScraper(len(vmis), cmdclient.NewClient, statsRequest)
	h.collector.Collect(vmis, scraper, collector.CollectionTimeout)

	results := scraper.GetValues()

	for _, vmi := range vmis {
		key := fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)
		if _, exists := results[key]; !exists {
			results[key] = &VMStatsResult{Error: "stats not available: VMI socket not found or busy"}
		}
	}

	response.WriteEntity(results)
}

func buildVMStatsRequestFromQuery(request *restful.Request) *cmdv1.VMStatsRequest {
	query := request.Request.URL.Query()
	req := &cmdv1.VMStatsRequest{}

	if query.Get("domainStats") == "true" {
		req.DomainStats = &cmdv1.DomainStatsRequest{}
	}
	if query.Get("dirtyRate") == "true" {
		req.DirtyRate = &cmdv1.DirtyRateRequest{}
	}
	if query.Get("guestGetLoad") == "true" {
		req.GuestGetLoad = &cmdv1.AgentLoadRequest{}
	}
	if query.Get("guestGetCpuStats") == "true" {
		req.GuestGetCpuStats = &cmdv1.AgentCpuStatsRequest{}
	}
	if query.Get("guestGetDiskStats") == "true" {
		req.GuestGetDiskStats = &cmdv1.AgentDiskStatsRequest{}
	}
	if query.Get("guestGetTime") == "true" {
		req.GuestGetTime = &cmdv1.AgentTimeRequest{}
	}
	if query.Get("guestGetVcpus") == "true" {
		req.GuestGetVcpus = &cmdv1.AgentVcpusRequest{}
	}
	if query.Get("guestGetMemoryBlockInfo") == "true" {
		req.GuestGetMemoryBlockInfo = &cmdv1.AgentMemoryBlockInfoRequest{}
	}
	if query.Get("guestGetUsers") == "true" {
		req.GuestGetUsers = &cmdv1.AgentUsersRequest{}
	}
	if query.Get("guestGetOsInfo") == "true" {
		req.GuestGetOsInfo = &cmdv1.AgentOsInfoRequest{}
	}
	if query.Get("guestGetDisks") == "true" {
		req.GuestGetDisks = &cmdv1.AgentDisksRequest{}
	}
	if query.Get("guestGetHostName") == "true" {
		req.GuestGetHostName = &cmdv1.AgentHostNameRequest{}
	}
	if query.Get("guestGetTimezone") == "true" {
		req.GuestGetTimezone = &cmdv1.AgentTimezoneRequest{}
	}
	if query.Get("guestNetworkGetRoute") == "true" {
		req.GuestNetworkGetRoute = &cmdv1.AgentNetworkRouteRequest{}
	}
	if query.Get("guestNetworkGetInterfaces") == "true" {
		req.GuestNetworkGetInterfaces = &cmdv1.AgentNetworkInterfacesRequest{}
	}
	if query.Get("guestGetMemoryBlocks") == "true" {
		req.GuestGetMemoryBlocks = &cmdv1.AgentMemoryBlocksRequest{}
	}

	return req
}
