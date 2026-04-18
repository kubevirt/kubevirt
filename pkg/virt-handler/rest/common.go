/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package rest

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

func getVMI(request *restful.Request, vmiStore cache.Store) (*v1.VirtualMachineInstance, int, error) {
	key := fmt.Sprintf("%s/%s", request.PathParameter("namespace"), request.PathParameter("name"))
	vmiObj, vmiExists, err := vmiStore.GetByKey(key)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if !vmiExists {
		return nil, http.StatusNotFound, fmt.Errorf("VMI %s does not exist", key)
	}
	return vmiObj.(*v1.VirtualMachineInstance), 0, nil
}
