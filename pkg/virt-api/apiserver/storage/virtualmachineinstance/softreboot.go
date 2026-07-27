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

type vmiSoftRebooter interface {
	SoftRebootVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *apierrors.StatusError
}

// SoftRebootREST serves the virtualmachineinstances/softreboot subresource PUT of the
// subresources.kubevirt.io API group as a rest.Connecter storage
type SoftRebootREST struct {
	rebooter vmiSoftRebooter
}

func NewSoftRebootREST(rebooter vmiSoftRebooter) *SoftRebootREST {
	return &SoftRebootREST{rebooter: rebooter}
}

var (
	_ = rest.Storage(&SoftRebootREST{})
	_ = rest.Connecter(&SoftRebootREST{})
)

func (r *SoftRebootREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *SoftRebootREST) Destroy() {}

func (r *SoftRebootREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *SoftRebootREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *SoftRebootREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to soft reboot a VirtualMachineInstance")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if statusErr := r.rebooter.SoftRebootVMI(req.Context(), namespace, name, req.Body); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusOK)
	}), nil
}
