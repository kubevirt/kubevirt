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

package virtualmachineinstance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

type sevOperator interface {
	SEVFetchCertChain(ctx context.Context, namespace, name string) (interface{}, *apierrors.StatusError)
	SEVQueryLaunchMeasurement(ctx context.Context, namespace, name string) (interface{}, *apierrors.StatusError)
	SEVSetupSession(ctx context.Context, namespace, name string, body io.ReadCloser) *apierrors.StatusError
	SEVInjectLaunchSecret(ctx context.Context, namespace, name string, body io.ReadCloser) *apierrors.StatusError
}

const (
	sevFetchCertChain     = "fetchcertchain"
	sevQueryLaunchMeasure = "querylaunchmeasurement"
	sevSetupSession       = "setupsession"
	sevInjectLaunchSecret = "injectlaunchsecret"
)

// SEVREST serves the virtualmachineinstances/sev/* subresources of the
// subresources.kubevirt.io API group as a rest.Connecter storage. The concrete
// action (fetchcertchain, querylaunchmeasurement, setupsession,
// injectlaunchsecret) is carried as an extra path segment after the sev
// subresource, so subpath is enabled and the segment is dispatched in Connect
type SEVREST struct {
	sev sevOperator
}

func NewSEVREST(sev sevOperator) *SEVREST {
	return &SEVREST{sev: sev}
}

var (
	_ = rest.Storage(&SEVREST{})
	_ = rest.Connecter(&SEVREST{})
)

func (r *SEVREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *SEVREST) Destroy() {}

func (r *SEVREST) ConnectMethods() []string {
	return []string{http.MethodGet, http.MethodPut}
}

func (r *SEVREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, true, "path"
}

func (r *SEVREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required for SEV subresources of a VirtualMachineInstance")
	}

	// [resource, name, subresource, extra...] i.e.
	// [virtualmachineinstances, <name>, sev, <action>].
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if !ok || len(requestInfo.Parts) <= 3 {
		return nil, apierrors.NewNotFound(v1.Resource("virtualmachineinstances"), name)
	}
	action := requestInfo.Parts[3]

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch action {
		case sevFetchCertChain:
			r.writeJSON(w, responder, func() (interface{}, *apierrors.StatusError) {
				return r.sev.SEVFetchCertChain(req.Context(), namespace, name)
			})
		case sevQueryLaunchMeasure:
			r.writeJSON(w, responder, func() (interface{}, *apierrors.StatusError) {
				return r.sev.SEVQueryLaunchMeasurement(req.Context(), namespace, name)
			})
		case sevSetupSession:
			if statusErr := r.sev.SEVSetupSession(req.Context(), namespace, name, req.Body); statusErr != nil {
				responder.Error(statusErr)
				return
			}
			w.WriteHeader(http.StatusAccepted)
		case sevInjectLaunchSecret:
			if statusErr := r.sev.SEVInjectLaunchSecret(req.Context(), namespace, name, req.Body); statusErr != nil {
				responder.Error(statusErr)
				return
			}
			w.WriteHeader(http.StatusOK)
		default:
			responder.Error(apierrors.NewNotFound(v1.Resource("virtualmachineinstances"), name))
		}
	}), nil
}

func (r *SEVREST) writeJSON(w http.ResponseWriter, responder rest.Responder, fetch func() (interface{}, *apierrors.StatusError)) {
	result, statusErr := fetch()
	if statusErr != nil {
		responder.Error(statusErr)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Log.Reason(err).Error("Failed to write SEV response")
	}
}
