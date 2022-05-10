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
	"io"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

const (
	failedRetrieveVMI      = "Failed to retrieve VMI"
	failedDetectCmdClient  = "Failed to detect cmd client"
	failedConnectCmdClient = "Failed to connect cmd client"
)

type LifecycleHandler struct {
	recorder     record.EventRecorder
	vmiInformer  cache.SharedIndexInformer
	virtShareDir string
}

func NewLifecycleHandler(recorder record.EventRecorder, vmiInformer cache.SharedIndexInformer, virtShareDir string) *LifecycleHandler {
	return &LifecycleHandler{
		recorder:     recorder,
		vmiInformer:  vmiInformer,
		virtShareDir: virtShareDir,
	}
}

func (lh *LifecycleHandler) PauseHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	err = client.PauseVirtualMachine(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to pause VMI")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) UnpauseHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	err = client.UnpauseVirtualMachine(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to unpause VMI")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) FreezeHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	unfreezeTimeout := &v1.FreezeUnfreezeTimeout{}
	if request.Request.Body == nil {
		log.Log.Object(vmi).Reason(err).Error("No unfreeze timeout in freeze request")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to retrieve unfreeze timeout"))
		return
	}

	defer request.Request.Body.Close()
	err = yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(unfreezeTimeout)
	switch err {
	case io.EOF, nil:
		break
	default:
		log.Log.Object(vmi).Reason(err).Error("Failed to unmarshal unfreeze timeout in freeze request")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to unmarshal unfreeze timeout"))
		return
	}

	if unfreezeTimeout.UnfreezeTimeout == nil {
		log.Log.Object(vmi).Reason(err).Error("Unfreeze timeout in freeze request is not set")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("Unfreeze timeout in freeze request is not set"))
		return
	}

	unfreezeTimeoutSeconds := int32(unfreezeTimeout.UnfreezeTimeout.Seconds())
	err = client.FreezeVirtualMachine(vmi, unfreezeTimeoutSeconds)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to freeze VMI")
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) UnfreezeHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	err = client.UnfreezeVirtualMachine(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to unfreeze VMI")
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) SoftRebootHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	err = client.SoftRebootVirtualMachine(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to soft reboot VMI")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	lh.recorder.Eventf(vmi, k8sv1.EventTypeNormal, "SoftRebooted", "VirtualMachineInstance soft rebooted")
	response.WriteHeader(http.StatusAccepted)
}

func (lh *LifecycleHandler) GetGuestInfo(request *restful.Request, response *restful.Response) {
	log.Log.Info("Retreiving guestinfo")
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	log.Log.Object(vmi).Infof("Retreiving guestinfo from %s", vmi.Name)

	guestInfo, err := client.GetGuestInfo()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get guest info")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	log.Log.Object(vmi).Infof("returning guestinfo :%v", guestInfo)
	response.WriteEntity(guestInfo)
}

func (lh *LifecycleHandler) GetUsers(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	log.Log.Object(vmi).Infof("Retreiving userlist from %s", vmi.Name)

	userList, err := client.GetUsers()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get user list")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(userList)
}

func (lh *LifecycleHandler) GetFilesystems(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	log.Log.Object(vmi).Infof("Retreiving filesystem list from %s", vmi.Name)

	fsList, err := client.GetFilesystems()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get file systems")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(fsList)
}

func (lh *LifecycleHandler) getVMILauncherClient(request *restful.Request, response *restful.Response) (*v1.VirtualMachineInstance, cmdclient.LauncherClient, error) {
	vmi, code, err := getVMI(request, lh.vmiInformer)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedRetrieveVMI)
		response.WriteError(code, err)
		return nil, nil, err
	}

	sockFile, err := cmdclient.FindSocketOnHost(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedDetectCmdClient)
		response.WriteError(http.StatusInternalServerError, err)
		return nil, nil, err
	}
	client, err := cmdclient.NewClient(sockFile)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error(failedConnectCmdClient)
		response.WriteError(http.StatusInternalServerError, err)
		return nil, nil, err
	}

	return vmi, client, nil
}

func (lh *LifecycleHandler) SEVFetchCertChainHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	log.Log.Object(vmi).Infof("Retrieving SEV platform info")

	sevPlatformInfo, err := client.GetSEVInfo()
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get SEV platform info")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(sevPlatformInfo)
}

func (lh *LifecycleHandler) SEVQueryLaunchMeasurementHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	log.Log.Object(vmi).Infof("Retreiving SEV launch measurement")

	sevMeasurementInfo, err := client.GetLaunchMeasurement(vmi)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to get VMI launch measurement")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(sevMeasurementInfo)
}

func (lh *LifecycleHandler) SEVInjectLaunchSecretHandler(request *restful.Request, response *restful.Response) {
	vmi, client, err := lh.getVMILauncherClient(request, response)
	if err != nil {
		return
	}

	if request.Request.Body == nil {
		log.Log.Object(vmi).Reason(err).Error("Request with no body: SEV secret parameters are required")
		response.WriteError(http.StatusBadRequest, fmt.Errorf("failed to retrieve SEV secret parameters from request"))
		return
	}

	opts := &v1.SEVSecretOptions{}
	err = yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
	switch err {
	case io.EOF, nil:
		break
	default:
		log.Log.Object(vmi).Reason(err).Error("Failed to decode SEV secret parameters")
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	log.Log.Object(vmi).Infof("Injecting SEV launch secret")

	if err := client.InjectLaunchSecret(vmi, opts); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to inject SEV launch secret")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}
