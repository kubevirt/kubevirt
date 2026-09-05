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
	"encoding/json"
	"net/http"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	v1 "kubevirt.io/api/core/v1"
)

type vmiUserListGetter interface {
	GetUserList(ctx context.Context, namespace, name string) (interface{}, error)
}

// UserListREST serves the virtualmachineinstances/userlist subresource GET of
// the subresources.kubevirt.io API group as a rest.Connecter storage. The
// request is proxied to virt-handler and the guest agent response is returned as JSON
type UserListREST struct {
	getter vmiUserListGetter
}

func NewUserListREST(getter vmiUserListGetter) *UserListREST {
	return &UserListREST{getter: getter}
}

var (
	_ = rest.Storage(&UserListREST{})
	_ = rest.Connecter(&UserListREST{})
	_ = rest.StorageMetadata(&UserListREST{})
)

func (r *UserListREST) ProducesMIMETypes(string) []string { return nil }

func (r *UserListREST) ProducesObject(verb string) interface{} {
	if verb == http.MethodGet {
		return v1.VirtualMachineInstanceGuestOSUserList{}
	}
	return nil
}

func (r *UserListREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *UserListREST) Destroy() {}

func (r *UserListREST) ConnectMethods() []string {
	return []string{http.MethodGet}
}

func (r *UserListREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *UserListREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to get the user list of a VirtualMachineInstance")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		result, err := r.getter.GetUserList(req.Context(), namespace, name)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}), nil
}
