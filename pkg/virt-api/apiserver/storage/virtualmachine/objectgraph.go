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
	"encoding/json"
	"io"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
)

type vmObjectGraphGetter interface {
	GetVMObjectGraph(namespace, name string, body io.Reader) (v1.ObjectGraphNode, *apierrors.StatusError)
}

// ObjectGraphREST serves the virtualmachines/objectgraph subresource of the
// subresources.kubevirt.io API group as a rest.Connecter storage
type ObjectGraphREST struct {
	getter vmObjectGraphGetter
}

func NewObjectGraphREST(getter vmObjectGraphGetter) *ObjectGraphREST {
	return &ObjectGraphREST{getter: getter}
}

var (
	_ = rest.Storage(&ObjectGraphREST{})
	_ = rest.Connecter(&ObjectGraphREST{})
	_ = rest.StorageMetadata(&ObjectGraphREST{})
)

func (r *ObjectGraphREST) ProducesMIMETypes(string) []string { return nil }

func (r *ObjectGraphREST) ProducesObject(verb string) interface{} {
	if verb == http.MethodGet {
		return v1.ObjectGraphNode{}
	}
	return nil
}

func (r *ObjectGraphREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *ObjectGraphREST) Destroy() {}

func (r *ObjectGraphREST) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (r *ObjectGraphREST) ConnectRequestBody(method string) interface{} {
	if method == http.MethodGet {
		return &v1.ObjectGraphOptions{}
	}
	return nil
}

func (r *ObjectGraphREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *ObjectGraphREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to get the object graph of a VirtualMachine")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		graph, statErr := r.getter.GetVMObjectGraph(namespace, name, req.Body)
		if statErr != nil {
			writeStatusError(w, statErr)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(graph)
	}), nil
}

// writeStatusError preserves the legacy behavior of the objectgraph
// handler (rest.writeError), it writes the metav1.Status carried by the
// StatusError as JSON with the status code the error dictates (e.g. 400 for a
// disabled feature gate or malformed body, 404 for a missing object, 500 for a
// build error) rather than mapping every failure to 500
func writeStatusError(w http.ResponseWriter, statErr *apierrors.StatusError) {
	status := statErr.Status()
	status.Kind = "Status"
	status.APIVersion = "v1"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(int(status.Code))
	_ = json.NewEncoder(w).Encode(status)
}
