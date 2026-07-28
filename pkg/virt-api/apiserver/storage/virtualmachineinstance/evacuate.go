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
	"io"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
)

type vmiEvacuationCanceler interface {
	EvacuateCancelVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *apierrors.StatusError
}

const cancelSubPath = "cancel"

// EvacuateCancelREST serves the virtualmachineinstances/evacuate/cancel
// subresource PUT of the subresources.kubevirt.io API group as a rest.Connecter
// storage. cancel is carried as an extra path segment after the evacuate
// subresource, so subpath is enabled and the segment is validated in Connect
type EvacuateCancelREST struct {
	canceler vmiEvacuationCanceler
}

func NewEvacuateCancelREST(canceler vmiEvacuationCanceler) *EvacuateCancelREST {
	return &EvacuateCancelREST{canceler: canceler}
}

var (
	_ = rest.Storage(&EvacuateCancelREST{})
	_ = rest.Connecter(&EvacuateCancelREST{})
)

func (r *EvacuateCancelREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *EvacuateCancelREST) Destroy() {}

func (r *EvacuateCancelREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *EvacuateCancelREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, true, "path"
}

func (r *EvacuateCancelREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to cancel evacuation of a VirtualMachineInstance")
	}

	// [resource, name, subresource, extra...] i.e.
	// [virtualmachineinstances, <name>, evacuate, cancel]
	requestInfo, ok := request.RequestInfoFrom(ctx)
	if !ok || len(requestInfo.Parts) <= 3 || requestInfo.Parts[3] != cancelSubPath {
		return nil, apierrors.NewNotFound(v1.Resource("virtualmachineinstances"), name)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if statusErr := r.canceler.EvacuateCancelVMI(req.Context(), namespace, name, req.Body); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusOK)
	}), nil
}
