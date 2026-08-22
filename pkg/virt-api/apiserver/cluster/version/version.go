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

// version is a GET to the collection path without a resource name
// (GET /apis/subresources.kubevirt.io/{version}/version) so it cannot be
// expressed as a rest.Storage. To keep existing clients working, it is served
// as a plain http.Handler
package version

import (
	"encoding/json"
	"net/http"

	"kubevirt.io/client-go/log"
	virtversion "kubevirt.io/client-go/version"
)

// Handler serves the version subresource as a plain http.Handler
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(virtversion.Get()); err != nil {
		log.Log.Reason(err).Error("Failed to write version response.")
	}
}
