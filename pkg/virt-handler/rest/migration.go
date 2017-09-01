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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package rest

import (
	"net/http"

	"github.com/emicklei/go-restful"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-handler/virtwrap/isolation"
)

type MigrationHostInfo struct {
	isolationDetector isolation.PodIsolationDetector
}

func NewMigrationHostInfo(isolationDetector isolation.PodIsolationDetector) *MigrationHostInfo {
	return &MigrationHostInfo{isolationDetector}
}

func (t *MigrationHostInfo) MigrationHostInfo(request *restful.Request, response *restful.Response) {
	vmName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	vm := v1.NewVMReferenceFromNameWithNS(namespace, vmName)
	result, err := t.isolationDetector.Detect(vm)
	if err != nil {
		response.WriteErrorString(http.StatusNotFound, err.Error())
		return
	}
	body := &v1.MigrationHostInfo{
		PidNS:      result.PidNS(),
		Controller: result.Controller(),
		Slice:      result.Slice(),
	}
	response.WriteHeader(http.StatusOK)
	response.WriteAsJson(body)
}
