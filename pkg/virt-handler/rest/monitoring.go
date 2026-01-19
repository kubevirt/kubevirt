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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type MonitoringHandler struct {
	vmiStore cache.Store
}

func NewMonitoringHandler(vmiStore cache.Store) *MonitoringHandler {
	return &MonitoringHandler{
		vmiStore: vmiStore,
	}
}

func (mh *MonitoringHandler) QueryHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, mh.vmiStore)
	if err != nil {
		log.Log.Reason(err).Error(failedRetrieveVMI)
		response.WriteError(code, err)
		return
	}

	// Get command from query parameter
	command := request.QueryParameter("command")
	if command == "" {
		log.Log.Object(vmi).Error("Command query parameter is required")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("command query parameter is required"))
		return
	}

	// Get optional arguments from query parameter (JSON encoded)
	var arguments map[string]interface{}
	argsParam := request.QueryParameter("arguments")
	if argsParam != "" {
		if err := json.Unmarshal([]byte(argsParam), &arguments); err != nil {
			log.Log.Object(vmi).Reason(err).Error("Failed to parse arguments query parameter")
			response.WriteError(http.StatusBadRequest, fmt.Errorf("invalid arguments JSON: %v", err))
			return
		}
	}

	// Build the QEMU guest agent command JSON
	qemuCommand := map[string]interface{}{
		"execute": command,
	}
	if arguments != nil {
		qemuCommand["arguments"] = arguments
	}

	commandJSON, err := json.Marshal(qemuCommand)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to marshal QEMU command")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	// Get the launcher client
	sockFile, err := cmdclient.FindSocket(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedDetectCmdClient)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	client, err := cmdclient.NewClient(sockFile)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedConnectCmdClient)
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	defer client.Close()

	// Execute the monitoring query
	domainName := api.VMINamespaceKeyFunc(vmi)
	rawOutput, err := client.ExecuteMonitoringQuery(domainName, string(commandJSON))
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to execute monitoring query")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	log.Log.Object(vmi).Infof("Monitoring query executed successfully for command: %s", command)

	// Write the raw JSON output directly
	response.Header().Set("Content-Type", "application/json")
	if _, err := response.Write([]byte(rawOutput)); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to write monitoring query response")
	}
}
