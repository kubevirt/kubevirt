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

// guestfs is a GET to the collection path without a resource name
// (GET /apis/subresources.kubevirt.io/{version}/guestfs) so it cannot be
// expressed as a rest.Storage. To keep existing clients (virtctl guestfs)
// working, it is served as a plain http.Handler
package guestfs

import (
	"encoding/json"
	"fmt"
	"net/http"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virtoperatorutils "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

// Handler serves the guestfs subresource as a plain http.Handler
type Handler struct {
	clusterConfig *virtconfig.ClusterConfig
}

func NewHandler(clusterConfig *virtconfig.ClusterConfig) *Handler {
	return &Handler{clusterConfig: clusterConfig}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	var kvConfig virtoperatorutils.KubeVirtDeploymentConfig
	kv := h.clusterConfig.GetConfigFromKubeVirtCR()
	if kv == nil {
		writeError(w, fmt.Errorf("failed getting KubeVirt config"))
		return
	}
	if err := json.Unmarshal([]byte(kv.Status.ObservedDeploymentConfig), &kvConfig); err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(kubecli.GuestfsInfo{
		Registry:    kv.Status.ObservedKubeVirtRegistry,
		Tag:         kv.Status.ObservedKubeVirtVersion,
		ImagePrefix: kvConfig.GetImagePrefix(),
		GsImage:     kvConfig.GsImage,
	}); err != nil {
		log.Log.Reason(err).Error("Failed to write guestfs response.")
	}
}

func writeError(w http.ResponseWriter, err error) {
	res := map[string]interface{}{
		"guestfs": map[string]interface{}{"status": "failed", "error": fmt.Sprintf("%v", err)},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	if encErr := json.NewEncoder(w).Encode(res); encErr != nil {
		log.Log.Reason(encErr).Error("Failed to write guestfs error response.")
	}
}
