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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package rest

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

func getVMI(request *restful.Request, vmiInformer cache.SharedIndexInformer) (*v1.VirtualMachineInstance, int, error) {
	key := fmt.Sprintf("%s/%s", request.PathParameter("namespace"), request.PathParameter("name"))
	vmiObj, vmiExists, err := vmiInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if !vmiExists {
		return nil, http.StatusNotFound, fmt.Errorf("VMI %s does not exist", key)
	}
	return vmiObj.(*v1.VirtualMachineInstance), 0, nil
}
