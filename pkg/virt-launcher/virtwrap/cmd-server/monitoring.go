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
	"time"

	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/client-go/log"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

func (l *Launcher) GetMonitoringData(_ context.Context, request *cmdv1.MonitoringRequest) (*cmdv1.MonitoringResponse, error) {
	response := &cmdv1.MonitoringResponse{
		Response: &cmdv1.Response{
			Success: true,
		},
	}

	if request.DomainStats {
		domainStats, err := l.domainManager.GetDomainStats()
		if err != nil {
			log.Log.Reason(err).Error("Failed to get domain stats")
			return monitoringError(response, err), nil
		}

		data, err := json.Marshal(domainStats)
		if err != nil {
			log.Log.Reason(err).Error("Failed to marshal monitoring data field")
			return monitoringError(response, err), nil
		}

		response.DomainStats = string(data)
	}

	if request.DirtyRateStats {
		const dirtyRateCalculationTime = time.Second
		dirtyRate, err := l.domainManager.GetDomainDirtyRateStats(dirtyRateCalculationTime)
		if err != nil {
			log.Log.Reason(err).Error("Failed to get domain dirty rate stats")
			return monitoringError(response, err), nil
		}

		data, err := json.Marshal(dirtyRate)
		if err != nil {
			log.Log.Reason(err).Error("Failed to marshal monitoring data field")
			return monitoringError(response, err), nil
		}

		response.DirtyRateStats = string(data)
	}

	if request.AgentData != nil {
		response.GuestAgentVersion = l.domainManager.GetGuestInfo().GAVersion

		response.AgentData = make(map[string]string, len(request.AgentData))
		for _, dataKey := range request.AgentData {
			response.AgentData[dataKey] = l.domainManager.GetAgentData(dataKey)
		}
	}

	return response, nil
}

func monitoringError(response *cmdv1.MonitoringResponse, err error) *cmdv1.MonitoringResponse {
	response.Response.Success = false
	response.Response.Message = getErrorMessage(err)
	return response
}
