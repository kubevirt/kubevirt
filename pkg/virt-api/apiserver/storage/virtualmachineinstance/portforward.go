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

type portForwardStreamer interface {
	StreamPortForward(ctx context.Context, namespace, name, port, protocol string, w http.ResponseWriter, req *http.Request) *apierrors.StatusError
}

// PortForwardREST serves the virtualmachineinstances/portforward endpoint (GET)
// of the subresources.kubevirt.io API group as a streaming rest.Connecter.
//
// The target port are carried as extra path segments
// after the subresource: .../portforward/{port} and .../portforward/{port}/{protocol}
// NewConnectOptions therefore enables the subpath so the aggregated apiserver
// routes those segments to this Connecter; they are then read back from the
// request info. The websocket is upgraded on the native http.ResponseWriter and
// the connection is tunneled directly to the VM
type PortForwardREST struct {
	streamer portForwardStreamer
}

func NewPortForwardREST(streamer portForwardStreamer) *PortForwardREST {
	return &PortForwardREST{streamer: streamer}
}

var (
	_ = rest.Storage(&PortForwardREST{})
	_ = rest.Connecter(&PortForwardREST{})
)

func (r *PortForwardREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *PortForwardREST) Destroy() {}

func (r *PortForwardREST) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (r *PortForwardREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, true, "path"
}

func (r *PortForwardREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to port-forward to a VirtualMachineInstance")
	}

	// [resource, name, subresource, extra...] i.e.
	// [virtualmachineinstances, <name>, portforward, <port>, <protocol>].
	var port, protocol string
	if requestInfo, ok := request.RequestInfoFrom(ctx); ok {
		if len(requestInfo.Parts) > 3 {
			port = requestInfo.Parts[3]
		}
		if len(requestInfo.Parts) > 4 {
			protocol = requestInfo.Parts[4]
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if statusErr := r.streamer.StreamPortForward(req.Context(), namespace, name, port, protocol, w, req); statusErr != nil {
			responder.Error(statusErr)
		}
	}), nil
}
