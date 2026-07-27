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
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
)

type vsockStreamer interface {
	StreamVSOCK(ctx context.Context, namespace, name, port, tls string, w http.ResponseWriter, req *http.Request) *apierrors.StatusError
}

// VSOCKREST serves the virtualmachineinstances/vsock endpoint (GET) of the
// subresources.kubevirt.io API group as a streaming rest.Connecter. The
// websocket is upgraded on the native http.ResponseWriter and proxied to
// virt-handler
type VSOCKREST struct {
	streamer vsockStreamer
}

func NewVSOCKREST(streamer vsockStreamer) *VSOCKREST {
	return &VSOCKREST{streamer: streamer}
}

var (
	_ = rest.Storage(&VSOCKREST{})
	_ = rest.Connecter(&VSOCKREST{})
)

func (r *VSOCKREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *VSOCKREST) Destroy() {}

func (r *VSOCKREST) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (r *VSOCKREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *VSOCKREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to open a VSOCK connection")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tls := "true"
		if val := req.URL.Query().Get("tls"); val != "" {
			tls = val
		}
		port := req.URL.Query().Get("port")

		if statusErr := r.streamer.StreamVSOCK(req.Context(), namespace, name, port, tls, w, req); statusErr != nil {
			responder.Error(statusErr)
		}
	}), nil
}
