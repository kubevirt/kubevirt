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
	"fmt"
	"io"

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

func (app *SubresourceAPIApp) sevEnabled() *errors.StatusError {
	if !app.clusterConfig.WorkloadEncryptionSEVEnabled() {
		return errors.NewBadRequest(fmt.Sprintf(featureGateDisabledErrFmt, featuregate.WorkloadEncryptionSEV))
	}
	return nil
}

// SEVFetchCertChain proxies the SEV platform certificate chain of a VMI from
// virt-handler. It is served by the aggregated apiserver under the nested
// subresource path virtualmachineinstances/sev/fetchcertchain
func (app *SubresourceAPIApp) SEVFetchCertChain(ctx context.Context, namespace, name string) (interface{}, *errors.StatusError) {
	if statusErr := app.sevEnabled(); statusErr != nil {
		return nil, statusErr
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

	result, err := app.httpGetVirtHandler(ctx, namespace, name, validate, getURL, v1.SEVPlatformInfo{})
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	return result, nil
}

// SEVQueryLaunchMeasurement proxies the SEV launch measurement of a VMI from
// virt-handler. It is served under the nested subresource path
// virtualmachineinstances/sev/querylaunchmeasurement
func (app *SubresourceAPIApp) SEVQueryLaunchMeasurement(ctx context.Context, namespace, name string) (interface{}, *errors.StatusError) {
	if statusErr := app.sevEnabled(); statusErr != nil {
		return nil, statusErr
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVQueryLaunchMeasurementURI(vmi)
	}

	result, err := app.httpGetVirtHandler(ctx, namespace, name, validateVMIForSEVAttestation, getURL, v1.SEVMeasurementInfo{})
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	return result, nil
}

// SEVSetupSession patches the VMI with the SEV session parameters. It is served
// under the nested subresource path virtualmachineinstances/sev/setupsession
func (app *SubresourceAPIApp) SEVSetupSession(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	if statusErr := app.sevEnabled(); statusErr != nil {
		return statusErr
	}

	if body == nil {
		return errors.NewBadRequest("Request with no body: SEV session parameters are required")
	}

	opts := &v1.SEVSessionOptions{}
	defer body.Close()
	if statusErr := decodeBodyReader(body, opts); statusErr != nil {
		return statusErr
	}

	if opts.Session == "" {
		return errors.NewBadRequest("Session blob is required")
	}

	if opts.DHCert == "" {
		return errors.NewBadRequest("DH cert is required")
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

	vmi, statusError := app.fetchAndValidateVirtualMachineInstance(ctx, namespace, name, validate)
	if statusError != nil {
		return statusError
	}

	oldSEV := vmi.Spec.Domain.LaunchSecurity.SEV
	newSEV := oldSEV.DeepCopy()
	newSEV.Session = opts.Session
	newSEV.DHCert = opts.DHCert
	patchBytes, err := patch.GenerateTestReplacePatch("/spec/domain/launchSecurity/sev", oldSEV, newSEV)
	if err != nil {
		return errors.NewInternalError(err)
	}

	log.Log.Object(vmi).Infof("Patching vmi: %s", string(patchBytes))
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(ctx, vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to patch vmi")
		return errors.NewInternalError(err)
	}

	return nil
}

// SEVInjectLaunchSecret proxies the SEV launch secret to virt-handler. It is
// served under the nested subresource path virtualmachineinstances/sev/injectlaunchsecret
func (app *SubresourceAPIApp) SEVInjectLaunchSecret(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	if statusErr := app.sevEnabled(); statusErr != nil {
		return statusErr
	}

	if body == nil {
		return errors.NewBadRequest("Request with no body: SEV secret parameters are required")
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVInjectLaunchSecretURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validateVMIForSEVAttestation, nil, getURL, false)
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
