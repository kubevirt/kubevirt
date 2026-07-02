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

package rest

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

func (app *SubresourceAPIApp) ensureTDXEnabled(response *restful.Response) bool {
	if !app.clusterConfig.WorkloadEncryptionTDXEnabled() {
		writeError(errors.NewBadRequest(fmt.Sprintf(featureGateDisabledErrFmt, featuregate.WorkloadEncryptionTDX)), response)
		return false
	}
	return true
}

func (app *SubresourceAPIApp) TDXInjectInitdataHandler(request *restful.Request, response *restful.Response) {
	if !app.ensureTDXEnabled(response) {
		return
	}

	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("Request with no body: TDX initdata parameters are required"), response)
		return
	}

	opts := &v1.TDXInitdataOptions{}
	if err := decodeBody(request, opts); err != nil {
		writeError(err, response)
		return
	}

	if len(opts.MRConfigId) == 0 {
		writeError(errors.NewBadRequest("MRConfigId is required"), response)
		return
	}

	if len(opts.OEMStrings) == 0 {
		writeError(errors.NewBadRequest("OEMStrings is required"), response)
		return
	}

	decodedMRConfigId, err := base64.StdEncoding.DecodeString(opts.MRConfigId)
	if err != nil || len(decodedMRConfigId) != 48 {
		writeError(errors.NewBadRequest("MRConfigId must be a base64-encoded value of exactly 48 bytes"), response)
		return
	}

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if !vmi.IsScheduled() {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not in %s phase", v1.Scheduled))
		}
		if !kutil.IsTDXAttestationRequested(vmi) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNoAttestationErr))
		}
		tdx := vmi.Spec.Domain.LaunchSecurity.TDX
		if tdx.MRConfigId != "" {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("MRConfigId already defined"))
		}
		return nil
	}

	// fetch and validate the VMI
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	vmi, statusError := app.fetchAndValidateVirtualMachineInstance(namespace, name, validate)
	if statusError != nil {
		writeError(statusError, response)
		return
	}

	oldTDX := vmi.Spec.Domain.LaunchSecurity.TDX
	newTDX := oldTDX.DeepCopy()
	newTDX.MRConfigId = opts.MRConfigId

	oldFirmware := vmi.Spec.Domain.Firmware
	if oldFirmware == nil {
		writeError(errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI has no firmware configuration")), response)
		return
	}
	newFirmware := oldFirmware.DeepCopy()
	newFirmware.OEMStrings = opts.OEMStrings

	patchSet := patch.New(
		patch.WithTest("/spec/domain/launchSecurity/tdx", oldTDX),
		patch.WithReplace("/spec/domain/launchSecurity/tdx", newTDX),
		patch.WithTest("/spec/domain/firmware", oldFirmware),
		patch.WithReplace("/spec/domain/firmware", newFirmware),
	)

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	log.Log.Object(vmi).Infof("Patching vmi: %s", string(patchBytes))
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to patch vmi")
		writeError(errors.NewInternalError(err), response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}
