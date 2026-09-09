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

type vmMemoryDumper interface {
	MemoryDump(ctx context.Context, namespace, name string, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) *apierrors.StatusError
}

// MemoryDumpREST serves the virtualmachines/memorydump subresource PUT of the
// subresources.kubevirt.io API group as a rest.Connecter storage.
type MemoryDumpREST struct {
	dumper vmMemoryDumper
}

func NewMemoryDumpREST(dumper vmMemoryDumper) *MemoryDumpREST {
	return &MemoryDumpREST{dumper: dumper}
}

var (
	_ = rest.Storage(&MemoryDumpREST{})
	_ = rest.Connecter(&MemoryDumpREST{})
)

func (r *MemoryDumpREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *MemoryDumpREST) Destroy() {}

func (r *MemoryDumpREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *MemoryDumpREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *MemoryDumpREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to request a memory dump of a VirtualMachine")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Body == nil {
			responder.Error(apierrors.NewBadRequest("Request with no body"))
			return
		}

		memoryDumpReq := &v1.VirtualMachineMemoryDumpRequest{}
		if err := yaml.NewYAMLOrJSONDecoder(req.Body, 1024).Decode(memoryDumpReq); err != nil && err != io.EOF {
			responder.Error(apierrors.NewBadRequest(fmt.Sprintf("unable to decode VirtualMachineMemoryDumpRequest: %v", err)))
			return
		}

		if statusErr := r.dumper.MemoryDump(req.Context(), namespace, name, memoryDumpReq); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}), nil
}
