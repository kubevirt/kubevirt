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

package virtualmachine

import (
	"context"
	"fmt"
	"io"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
)

type vmRestarter interface {
	RestartVM(ctx context.Context, namespace, name string, opts *v1.RestartOptions) *apierrors.StatusError
}

type RestartREST struct {
	restarter vmRestarter
}

func NewRestartREST(restarter vmRestarter) *RestartREST {
	return &RestartREST{restarter: restarter}
}

var (
	_ = rest.Storage(&RestartREST{})
	_ = rest.Connecter(&RestartREST{})
)

func (r *RestartREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *RestartREST) Destroy() {}

func (r *RestartREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *RestartREST) ConnectRequestBody(method string) interface{} {
	if method == http.MethodPut {
		return &v1.RestartOptions{}
	}
	return nil
}

func (r *RestartREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *RestartREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to restart a VirtualMachine")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		restartOptions := &v1.RestartOptions{}
		if req.Body != nil {
			if err := yaml.NewYAMLOrJSONDecoder(req.Body, 1024).Decode(restartOptions); err != nil && err != io.EOF {
				responder.Error(apierrors.NewBadRequest(fmt.Sprintf("unable to decode RestartOptions: %v", err)))
				return
			}
		}

		if statusErr := r.restarter.RestartVM(req.Context(), namespace, name, restartOptions); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}), nil
}
