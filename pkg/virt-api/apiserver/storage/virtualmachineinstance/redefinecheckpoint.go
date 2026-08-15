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

	backupv1alpha1 "kubevirt.io/api/backup/v1alpha1"
	v1 "kubevirt.io/api/core/v1"
)

type vmiCheckpointRedefiner interface {
	RedefineCheckpointVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *apierrors.StatusError
}

// RedefineCheckpointREST serves the virtualmachineinstances/redefine-checkpoint
// subresource PUT of the subresources.kubevirt.io API group as a rest.Connecter
// storage.
type RedefineCheckpointREST struct {
	redefiner vmiCheckpointRedefiner
}

func NewRedefineCheckpointREST(redefiner vmiCheckpointRedefiner) *RedefineCheckpointREST {
	return &RedefineCheckpointREST{redefiner: redefiner}
}

var (
	_ = rest.Storage(&RedefineCheckpointREST{})
	_ = rest.Connecter(&RedefineCheckpointREST{})
)

func (r *RedefineCheckpointREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *RedefineCheckpointREST) Destroy() {}

func (r *RedefineCheckpointREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *RedefineCheckpointREST) ConnectRequestBody(method string) interface{} {
	if method == http.MethodPut {
		return &backupv1alpha1.BackupCheckpoint{}
	}
	return nil
}

func (r *RedefineCheckpointREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *RedefineCheckpointREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to redefine a checkpoint of a VirtualMachineInstance")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if statusErr := r.redefiner.RedefineCheckpointVMI(req.Context(), namespace, name, req.Body); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusOK)
	}), nil
}
