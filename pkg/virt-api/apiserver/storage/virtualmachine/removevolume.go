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

type vmVolumeRemover interface {
	RemoveVolume(ctx context.Context, namespace, name string, opts *v1.RemoveVolumeOptions, ephemeral bool) *apierrors.StatusError
}

// RemoveVolumeREST serves the virtualmachines/removevolume subresource PUT of the
// subresources.kubevirt.io API group as a rest.Connecter storage.
type RemoveVolumeREST struct {
	remover vmVolumeRemover
}

func NewRemoveVolumeREST(remover vmVolumeRemover) *RemoveVolumeREST {
	return &RemoveVolumeREST{remover: remover}
}

var (
	_ = rest.Storage(&RemoveVolumeREST{})
	_ = rest.Connecter(&RemoveVolumeREST{})
)

func (r *RemoveVolumeREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *RemoveVolumeREST) Destroy() {}

func (r *RemoveVolumeREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *RemoveVolumeREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *RemoveVolumeREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to remove a volume from a VirtualMachine")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Body == nil {
			responder.Error(apierrors.NewBadRequest("Request with no body, a new name is expected as the request body"))
			return
		}

		opts := &v1.RemoveVolumeOptions{}
		if err := yaml.NewYAMLOrJSONDecoder(req.Body, 1024).Decode(opts); err != nil && err != io.EOF {
			responder.Error(apierrors.NewBadRequest(fmt.Sprintf("unable to decode RemoveVolumeOptions: %v", err)))
			return
		}

		// ephemeral is false: this is the VM subresource, which persists the
		// volume request on the VM (the VMI subresource uses ephemeral=true).
		if statusErr := r.remover.RemoveVolume(req.Context(), namespace, name, opts, false); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}), nil
}
