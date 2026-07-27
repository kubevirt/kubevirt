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
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-api/definitions"
)

type vncStreamer interface {
	StreamVNC(ctx context.Context, namespace, name string, preserveSession bool, w http.ResponseWriter, req *http.Request) *apierrors.StatusError
}

// VNCREST serves the virtualmachineinstances/vnc endpoint (GET) of the
// subresources.kubevirt.io API group as a streaming rest.Connecter. The
// websocket is upgraded on the native http.ResponseWriter and proxied to
// virt-handler
type VNCREST struct {
	streamer vncStreamer
}

func NewVNCREST(streamer vncStreamer) *VNCREST {
	return &VNCREST{streamer: streamer}
}

var (
	_ = rest.Storage(&VNCREST{})
	_ = rest.Connecter(&VNCREST{})
)

func (r *VNCREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *VNCREST) Destroy() {}

func (r *VNCREST) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (r *VNCREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *VNCREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to open a VNC connection")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		preserveSession := false
		if raw := req.URL.Query().Get(definitions.PreserveSessionParamName); raw != "" {
			val, err := strconv.ParseBool(raw)
			if err != nil {
				log.DefaultLogger().Reason(err).Warningf("Failed to parse VNC's query parameter: %s", definitions.PreserveSessionParamName)
			}
			preserveSession = val
		}

		if statusErr := r.streamer.StreamVNC(req.Context(), namespace, name, preserveSession, w, req); statusErr != nil {
			responder.Error(statusErr)
		}
	}), nil
}
