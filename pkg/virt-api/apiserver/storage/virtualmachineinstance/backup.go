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

type vmiBackuper interface {
	BackupVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *apierrors.StatusError
}

// BackupREST serves the virtualmachineinstances/backup subresource PUT of the
// subresources.kubevirt.io API group as a rest.Connecter storage.
type BackupREST struct {
	backuper vmiBackuper
}

func NewBackupREST(backuper vmiBackuper) *BackupREST {
	return &BackupREST{backuper: backuper}
}

var (
	_ = rest.Storage(&BackupREST{})
	_ = rest.Connecter(&BackupREST{})
)

func (r *BackupREST) New() runtime.Object {
	return &v1.VirtualMachineInstance{}
}

func (r *BackupREST) Destroy() {}

func (r *BackupREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *BackupREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *BackupREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to back up a VirtualMachineInstance")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if statusErr := r.backuper.BackupVMI(req.Context(), namespace, name, req.Body); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusOK)
	}), nil
}
