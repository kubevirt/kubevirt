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

type vmStarter interface {
	StartVM(ctx context.Context, namespace, name string, opts *v1.StartOptions) *apierrors.StatusError
}

// StartREST serves the virtualmachines/start subresource PUT of the
// subresources.kubevirt.io API group as a rest.Connecter storage.
type StartREST struct {
	starter vmStarter
}

func NewStartREST(starter vmStarter) *StartREST {
	return &StartREST{starter: starter}
}

var (
	_ = rest.Storage(&StartREST{})
	_ = rest.Connecter(&StartREST{})
)

func (r *StartREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *StartREST) Destroy() {}

// start only answers to PUT, matching the legacy route and the generated client
func (r *StartREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

// NewConnectOptions returns nil because start takes no options from the
// URL/query string. The StartOptions payload travels in the request body and
// is decoded inside the Connect handler
func (r *StartREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *StartREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to start a VirtualMachine")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		startOptions := &v1.StartOptions{}
		if req.Body != nil {
			if err := yaml.NewYAMLOrJSONDecoder(req.Body, 1024).Decode(startOptions); err != nil && err != io.EOF {
				responder.Error(apierrors.NewBadRequest(fmt.Sprintf("unable to decode StartOptions: %v", err)))
				return
			}
		}

		if statusErr := r.starter.StartVM(req.Context(), namespace, name, startOptions); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}), nil
}
