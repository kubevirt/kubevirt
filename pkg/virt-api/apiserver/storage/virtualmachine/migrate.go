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

type vmMigrator interface {
	MigrateVM(ctx context.Context, namespace, name string, opts *v1.MigrateOptions) *apierrors.StatusError
}

type MigrateREST struct {
	migrator vmMigrator
}

func NewMigrateREST(migrator vmMigrator) *MigrateREST {
	return &MigrateREST{migrator: migrator}
}

var (
	_ = rest.Storage(&MigrateREST{})
	_ = rest.Connecter(&MigrateREST{})
)

func (r *MigrateREST) New() runtime.Object {
	return &v1.VirtualMachine{}
}

func (r *MigrateREST) Destroy() {}

func (r *MigrateREST) ConnectMethods() []string {
	return []string{http.MethodPut}
}

func (r *MigrateREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

func (r *MigrateREST) Connect(ctx context.Context, name string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, apierrors.NewBadRequest("namespace is required to migrate a VirtualMachine")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		migrateOptions := &v1.MigrateOptions{}
		if req.Body != nil {
			if err := yaml.NewYAMLOrJSONDecoder(req.Body, 1024).Decode(migrateOptions); err != nil && err != io.EOF {
				responder.Error(apierrors.NewBadRequest(fmt.Sprintf("unable to decode MigrateOptions: %v", err)))
				return
			}
		}

		if statusErr := r.migrator.MigrateVM(req.Context(), namespace, name, migrateOptions); statusErr != nil {
			responder.Error(statusErr)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}), nil
}
