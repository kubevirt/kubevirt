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

// native storage should reuses the proxy logic without using any of the legacy code
type consoleStreamer interface {
	StreamConsole(ctx context.Context, namespace, name string, w http.ResponseWriter, req *http.Request) *apierrors.StatusError
}

// ConsoleREST serves the virtualmachineinstances/console endpoint (GET) of
// the subresources.kubevirt.io API group as streaming rest.Connecter.
// The websocket is upgraded on the native http.ResponseWriter
// and proxied to virt-handler
type ConsoleREST struct {
	streamer consoleStreamer
}

func NewConsoleREST(streamer consoleStreamer) *ConsoleREST {
	return &ConsoleREST{streamer: streamer}
}

var (
	_ = rest.Storage(&ConsoleREST{})
	_ = rest.Connecter(&ConsoleREST{})
)

func (r *ConsoleREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *ConsoleREST) Destroy() {}

// console answers to GET and is upgraded to a websocket connection.
func (r *ConsoleREST) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (r *ConsoleREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *ConsoleREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to open a console")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if statusErr := r.streamer.StreamConsole(req.Context(), namespace, name, w, req); statusErr != nil {
			responder.Error(statusErr)
		}
	}), nil
}
