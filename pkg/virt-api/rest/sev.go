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
 * Copyright The KubeVirt Authors
 *
 */

package rest

import (
	"context"
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	kutil "kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

const (
	vmiNoAttestationErr = "Attestation not requested for VMI"
)

func (app *SubresourceAPIApp) ensureSEVEnabled(response *restful.Response) bool {
	if !app.clusterConfig.WorkloadEncryptionSEVEnabled() {
		writeError(errors.NewBadRequest(fmt.Sprintf(featureGateDisabledErrFmt, featuregate.WorkloadEncryptionSEV)), response)
		return false
	}
	return true
}

func (app *SubresourceAPIApp) SEVFetchCertChainRequestHandler(request *restful.Request, response *restful.Response) {
	if !app.ensureSEVEnabled(response) {
		return
	}

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if !vmi.IsScheduled() && !vmi.IsRunning() {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not assigned to a node yet"))
		}
		if !kutil.IsSEVAttestationRequested(vmi) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNoAttestationErr))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVFetchCertChainURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.SEVPlatformInfo{})
}

func (app *SubresourceAPIApp) SEVQueryLaunchMeasurementHandler(request *restful.Request, response *restful.Response) {
	if !app.ensureSEVEnabled(response) {
		return
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVQueryLaunchMeasurementURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validateVMIForSEVAttestation, getURL, v1.SEVMeasurementInfo{})
}

func (app *SubresourceAPIApp) SEVSetupSessionHandler(request *restful.Request, response *restful.Response) {
	if !app.ensureSEVEnabled(response) {
		return
	}

	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("Request with no body: SEV session parameters are required"), response)
		return
	}

	opts := &v1.SEVSessionOptions{}
	if err := decodeBody(request, opts); err != nil {
		writeError(err, response)
		return
	}

	if opts.Session == "" {
		writeError(errors.NewBadRequest("Session blob is required"), response)
		return
	}

	if opts.DHCert == "" {
		writeError(errors.NewBadRequest("DH cert is required"), response)
		return
	}

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if !vmi.IsScheduled() {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not in %s phase", v1.Scheduled))
		}
		if !kutil.IsSEVAttestationRequested(vmi) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNoAttestationErr))
		}
		sev := vmi.Spec.Domain.LaunchSecurity.SEV
		if sev.Session != "" || sev.DHCert != "" {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Session already defined"))
		}
		return nil
	}

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	vmi, statusError := app.fetchAndValidateVirtualMachineInstance(namespace, name, validate)
	if statusError != nil {
		writeError(statusError, response)
		return
	}

	oldSEV := vmi.Spec.Domain.LaunchSecurity.SEV
	newSEV := oldSEV.DeepCopy()
	newSEV.Session = opts.Session
	newSEV.DHCert = opts.DHCert
	patch, err := patch.GenerateTestReplacePatch("/spec/domain/launchSecurity/sev", oldSEV, newSEV)
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	log.Log.Object(vmi).Infof("Patching vmi: %s", string(patch))
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patch, metav1.PatchOptions{}); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to patch vmi")
		writeError(errors.NewInternalError(err), response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) SEVInjectLaunchSecretHandler(request *restful.Request, response *restful.Response) {
	if !app.ensureSEVEnabled(response) {
		return
	}

	if request.Request.Body == nil {
		writeError(errors.NewBadRequest("Request with no body: SEV secret parameters are required"), response)
		return
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVInjectLaunchSecretURI(vmi)
	}

	app.putRequestHandler(request, response, validateVMIForSEVAttestation, getURL, false)
}

// Validate a VMI for SEV attestation: Running, Paused and with Attestation requested.
func validateVMIForSEVAttestation(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
	}
	if !kutil.IsSEVAttestationRequested(vmi) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNoAttestationErr))
	}
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotPaused))
	}
	return nil
}
