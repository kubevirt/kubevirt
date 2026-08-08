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

// Package expand serves the cluster level expand-vm-spec subresource.
//
// Unlike the subresources that hang under a VM/VMI such as expand-spec, start,
// console, expand-vm-spec is a PUT to the collection path without a resource
// name (PUT /apis/subresources.kubevirt.io/{version}/namespaces/{ns}/expand-vm-spec).
// The aggregated API server's standard REST routing can only register PUT on
// the item path ({resource}/{name}), so there is no rest.Storage interface that
// reproduces this shape without changing the client. To keep existing virtctl
// working, this is served as a plain mux handler, similar to the webhooks.
package expand

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apiserver/pkg/endpoints/request"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	instancetypeexpand "kubevirt.io/kubevirt/pkg/instancetype/expand"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const unmarshalRequestErrFmt = "Can not unmarshal Request body to struct, error: %s"

// vmExpander applies a VM's instancetype and preference, returning the expanded
// VM. It is implemented by the shared instancetype expander.
type vmExpander interface {
	Expand(vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
}

// Handler serves the expand-vm-spec subresource as a plain http.Handler.
type Handler struct {
	expander vmExpander
}

func NewHandler(clusterConfig *virtconfig.ClusterConfig, virtClient kubecli.KubevirtClient) *Handler {
	return &Handler{
		expander: instancetypeexpand.New(
			clusterConfig,
			find.NewSpecFinder(nil, nil, nil, virtClient),
			preferenceFind.NewSpecFinder(nil, nil, nil, virtClient),
		),
	}
}

// ServeHTTP decodes the VirtualMachine from the request body, validates it,
// expands its instancetype/preference and writes back the expanded VM. The
// response is encoded as a kubevirt.io/v1 VirtualMachine (not the serving
// subresources.kubevirt.io group) so existing clients keep decoding it with
// their kubevirt.io/v1 scheme
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Body == nil {
		writeError(w, errors.NewBadRequest("empty request body"))
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, errors.NewBadRequest(err.Error()))
		return
	}

	rawObj := map[string]interface{}{}
	if err := json.Unmarshal(bodyBytes, &rawObj); err != nil {
		writeError(w, errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)))
		return
	}

	if validationErrors := definitions.Validator.Validate(v1.VirtualMachineGroupVersionKind, rawObj); len(validationErrors) > 0 {
		writeValidationErrors(w, validationErrors)
		return
	}

	vm := &v1.VirtualMachine{}
	if err := json.Unmarshal(bodyBytes, vm); err != nil {
		writeError(w, errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)))
		return
	}

	// The namespace is taken from the request's RequestInfo
	var namespace string
	if info, ok := request.RequestInfoFrom(r.Context()); ok {
		namespace = info.Namespace
	}
	if namespace == "" {
		writeError(w, errors.NewBadRequest("The request namespace must not be empty"))
		return
	}
	if vm.Namespace != "" && vm.Namespace != namespace {
		writeError(w, errors.NewBadRequest(fmt.Sprintf("VM namespace must be empty or %s", namespace)))
		return
	}
	vm.Namespace = namespace

	expandedVM, err := h.expander.Expand(vm)
	if err != nil {
		writeError(w, errors.NewBadRequest(err.Error()))
		return
	}

	writeVirtualMachine(w, expandedVM)
}

func writeVirtualMachine(w http.ResponseWriter, vm *v1.VirtualMachine) {
	vm.APIVersion = v1.VirtualMachineGroupVersionKind.GroupVersion().String()
	vm.Kind = v1.VirtualMachineGroupVersionKind.Kind

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(vm); err != nil {
		log.Log.Reason(err).Error("Failed to write expand-vm-spec response.")
	}
}

func writeValidationErrors(w http.ResponseWriter, validationErrors []error) {
	causes := make([]metav1.StatusCause, 0, len(validationErrors))
	for _, err := range validationErrors {
		causes = append(causes, metav1.StatusCause{
			Message: err.Error(),
		})
	}

	statusError := errors.NewBadRequest("Object is not a valid VirtualMachine")
	statusError.ErrStatus.Details = &metav1.StatusDetails{Causes: causes}
	writeError(w, statusError)
}

func writeError(w http.ResponseWriter, statusErr *errors.StatusError) {
	status := statusErr.Status()
	status.Kind = "Status"
	status.APIVersion = "v1"

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(status.Code))
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Log.Reason(err).Error("Failed to write error response.")
	}
}
