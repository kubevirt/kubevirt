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
	requests  map[string]*cmdv1.VMStatsRequest
}

type vmStatsChannelResult struct {
	key    string
	stats  *stats.VMStats
	errMsg string
}

func NewVMStatsScraper(channelLength int, newClient func(string) (cmdclient.LauncherClient, error), requests map[string]*cmdv1.VMStatsRequest) *VMStatsScraper {
	return &VMStatsScraper{
		ch:        make(chan *vmStatsChannelResult, channelLength),
		newClient: newClient,
		requests:  requests,
	}
}

func (s *VMStatsScraper) Scrape(socketFile string, vmi *v1.VirtualMachineInstance) {
	ts := time.Now()
	key := fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)

	req, ok := s.requests[key]
	if !ok {
		s.ch <- &vmStatsChannelResult{key: key, errMsg: "no stats request configured for VMI"}
		return
	}

	cli, err := s.newClient(socketFile)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to connect to cmd client socket")
		s.ch <- &vmStatsChannelResult{key: key, errMsg: err.Error()}
		return
	}
	defer cli.Close()

	vmStats, err := cli.GetVMStats(req)
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

	requests, err := parseVMStatsRequests(request)
	if err != nil {
		response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to parse request body: %v", err))
		return
	}
	if len(requests) == 0 {
		response.WriteError(http.StatusBadRequest, fmt.Errorf("at least one VMI must be requested"))
		return
	}
	for key, req := range requests {
		if req == nil || *req == (cmdv1.VMStatsRequest{}) {
			response.WriteError(http.StatusBadRequest, fmt.Errorf("at least one stats category must be requested for %q", key))
			return
		}
	}

	onNode := make(map[string]*v1.VirtualMachineInstance)
	for _, obj := range h.vmiStore.List() {
		if vmi, ok := obj.(*v1.VirtualMachineInstance); ok {
			onNode[fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)] = vmi
		}
	}

	results := make(map[string]*VMStatsResult)
	vmis := make([]*v1.VirtualMachineInstance, 0, len(requests))
	for key := range requests {
		if vmi, ok := onNode[key]; ok {
			vmis = append(vmis, vmi)
		} else {
			results[key] = &VMStatsResult{Error: "VMI not found on node"}
		}
	}

	scraper := NewVMStatsScraper(len(vmis), cmdclient.NewClient, requests)
	h.collector.Collect(vmis, scraper, collector.CollectionTimeout)

	for key, result := range scraper.GetValues() {
		results[key] = result
	}

	for _, vmi := range vmis {
		key := fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name)
		if _, exists := results[key]; !exists {
			results[key] = &VMStatsResult{Error: "stats not available: VMI socket not found or busy"}
		}
	}

	response.WriteEntity(results)
}

func parseVMStatsRequests(request *restful.Request) (map[string]*cmdv1.VMStatsRequest, error) {
	body := struct {
		VMIs map[string]*cmdv1.VMStatsRequest `json:"vmis"`
	}{}
	if err := request.ReadEntity(&body); err != nil {
		return nil, err
	}
	return body.VMIs, nil
}
