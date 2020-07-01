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

	"github.com/emicklei/go-restful"

	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type LifecycleHandler struct {
	vmiInformer   cache.SharedIndexInformer
	virtShareDir  string
	clientFactory cmdclient.ReadOnlyVMIClientFactory
}

func NewLifecycleHandler(vmiInformer cache.SharedIndexInformer, virtShareDir string, factory cmdclient.ReadOnlyVMIClientFactory) *LifecycleHandler {
	return &LifecycleHandler{
		vmiInformer:   vmiInformer,
		virtShareDir:  virtShareDir,
		clientFactory: factory,
	}
}

func (lh *LifecycleHandler) PauseHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, lh.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}

	client, _, _ := lh.clientFactory.ClientForVMIIfExists(vmi)
	if client == nil {
		log.Log.Object(vmi).Error("Failed to connect cmd client")
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("No connection to the VMI"))
	}

	err = client.PauseVirtualMachine(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to pause VMI")
		response.WriteError(http.StatusInternalServerError, err)
	}

	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) UnpauseHandler(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, lh.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}

	client, _, _ := lh.clientFactory.ClientForVMIIfExists(vmi)
	if client == nil {
		log.Log.Object(vmi).Error("Failed to connect cmd client")
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("No connection to the VMI"))
	}

	err = client.UnpauseVirtualMachine(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to unpause VMI")
		response.WriteError(http.StatusInternalServerError, err)
	}

	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) GetGuestInfo(request *restful.Request, response *restful.Response) {
	log.Log.Info("Retreiving guestinfo")
	vmi, code, err := getVMI(request, lh.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}

	log.Log.Object(vmi).Infof("Retreiving guestinfo from %s", vmi.Name)

	client, _, _ := lh.clientFactory.ClientForVMIIfExists(vmi)
	if client == nil {
		log.Log.Object(vmi).Error("Failed to connect cmd client")
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("No connection to the VMI"))
	}

	guestInfo, err := client.GetGuestInfo()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get guest info")
		response.WriteError(http.StatusInternalServerError, err)
	}

	log.Log.Object(vmi).Infof("returning guestinfo :%v", guestInfo)
	response.WriteEntity(guestInfo)
}

func (lh *LifecycleHandler) GetUsers(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, lh.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}

	log.Log.Object(vmi).Infof("Retreiving userlist from %s", vmi.Name)

	client, _, _ := lh.clientFactory.ClientForVMIIfExists(vmi)
	if client == nil {
		log.Log.Object(vmi).Error("Failed to connect cmd client")
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("No connection to the VMI"))
	}

	userList, err := client.GetUsers()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get user list")
		response.WriteError(http.StatusInternalServerError, err)
	}

	response.WriteEntity(userList)
}

func (lh *LifecycleHandler) GetFilesystems(request *restful.Request, response *restful.Response) {
	vmi, code, err := getVMI(request, lh.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to retrieve VMI")
		response.WriteError(code, err)
		return
	}

	log.Log.Object(vmi).Infof("Retreiving filesystem list from %s", vmi.Name)

	client, _, _ := lh.clientFactory.ClientForVMIIfExists(vmi)
	if client == nil {
		log.Log.Object(vmi).Error("Failed to connect cmd client")
		response.WriteError(http.StatusInternalServerError, fmt.Errorf("No connection to the VMI"))
	}

	fsList, err := client.GetFilesystems()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get guest info")
		response.WriteError(http.StatusInternalServerError, err)
	}

	response.WriteEntity(fsList)
}
