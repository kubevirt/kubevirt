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

package sev

// Invalidate stale bazel remote cache entries after CI CacheNotFoundException.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	kutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
)

const (
	vmiNoAttestationErr       = "Attestation not requested for VMI"
	vmiNotRunning             = "VMI is not running"
	vmiNotPaused              = "VMI is not paused"
	unmarshalRequestErrFmt    = "Can not unmarshal Request body to struct, error: %s"
	featureGateDisabledErrFmt = "'%s' feature gate is not enabled"
	prepConnectionErrFmt      = "Cannot prepare connection %s"
	getRequestErrFmt          = "Cannot GET request %s"
)

type validation func(*v1.VirtualMachineInstance) *errors.StatusError
type errorPostProcessing func(*v1.VirtualMachineInstance, error) error
type urlResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

// Handler contains the VirtualMachineInstance SEV operations served by the
// GenericAPIServer storage Connecter.
type Handler struct {
	virtCli           kubecli.KubevirtClient
	consoleServerPort int
	httpClient        *http.Client
	clusterConfig     *virtconfig.ClusterConfig
}

func NewHandler(virtCli kubecli.KubevirtClient, consoleServerPort int, tlsConfig *tls.Config, clusterConfig *virtconfig.ClusterConfig) *Handler {
	return &Handler{
		virtCli:           virtCli,
		consoleServerPort: consoleServerPort,
		httpClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   10 * time.Second,
		},
		clusterConfig: clusterConfig,
	}
}

func (h *Handler) sevEnabled() *errors.StatusError {
	if !h.clusterConfig.WorkloadEncryptionSEVEnabled() {
		return errors.NewBadRequest(fmt.Sprintf(featureGateDisabledErrFmt, featuregate.WorkloadEncryptionSEV))
	}
	return nil
}

// SEVFetchCertChain proxies the SEV platform certificate chain of a VMI from
// virt-handler. It is served by the aggregated apiserver under the nested
// subresource path virtualmachineinstances/sev/fetchcertchain
func (h *Handler) SEVFetchCertChain(ctx context.Context, namespace, name string) (interface{}, *errors.StatusError) {
	if statusErr := h.sevEnabled(); statusErr != nil {
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

	result, err := h.httpGetVirtHandler(ctx, namespace, name, validate, getURL, v1.SEVPlatformInfo{})
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	return result, nil
}

// SEVQueryLaunchMeasurement proxies the SEV launch measurement of a VMI from
// virt-handler. It is served under the nested subresource path
// virtualmachineinstances/sev/querylaunchmeasurement
func (h *Handler) SEVQueryLaunchMeasurement(ctx context.Context, namespace, name string) (interface{}, *errors.StatusError) {
	if statusErr := h.sevEnabled(); statusErr != nil {
		return nil, statusErr
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVQueryLaunchMeasurementURI(vmi)
	}

	result, err := h.httpGetVirtHandler(ctx, namespace, name, validateVMIForSEVAttestation, getURL, v1.SEVMeasurementInfo{})
	if err != nil {
		return nil, errors.NewInternalError(err)
	}
	return result, nil
}

// SEVSetupSession patches the VMI with the SEV session parameters. It is served
// under the nested subresource path virtualmachineinstances/sev/setupsession
func (h *Handler) SEVSetupSession(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	if statusErr := h.sevEnabled(); statusErr != nil {
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

	vmi, statusError := h.fetchAndValidateVirtualMachineInstance(ctx, namespace, name, validate)
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
	if _, err := h.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(ctx, vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{}); err != nil {
		log.Log.Object(vmi).Reason(err).Errorf("Failed to patch vmi")
		return errors.NewInternalError(err)
	}

	return nil
}

// SEVInjectLaunchSecret proxies the SEV launch secret to virt-handler. It is
// served under the nested subresource path virtualmachineinstances/sev/injectlaunchsecret
func (h *Handler) SEVInjectLaunchSecret(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	if statusErr := h.sevEnabled(); statusErr != nil {
		return statusErr
	}

	if body == nil {
		return errors.NewBadRequest("Request with no body: SEV secret parameters are required")
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SEVInjectLaunchSecretURI(vmi)
	}

	return h.connectVirtHandler(ctx, namespace, name, body, validateVMIForSEVAttestation, nil, getURL, false)
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

func (h *Handler) httpGetVirtHandler(
	ctx context.Context,
	namespace, name string,
	validate validation,
	getURL urlResolver,
	v interface{},
) (interface{}, error) {
	_, url, conn, statusErr := h.prepareConnection(ctx, namespace, name, validate, getURL)
	if statusErr != nil {
		log.Log.Errorf(prepConnectionErrFmt, statusErr.Error())
		return nil, statusErr
	}

	resp, err := conn.Get(url, "application/json")
	if err != nil {
		log.Log.Errorf(getRequestErrFmt, err.Error())
		return nil, err
	}

	if err := json.Unmarshal([]byte(resp), &v); err != nil {
		log.Log.Reason(err).Error("error unmarshalling response")
		return nil, err
	}
	return v, nil
}

func (h *Handler) connectVirtHandler(
	ctx context.Context,
	namespace, name string,
	body io.ReadCloser,
	preValidate validation,
	postProcessError errorPostProcessing,
	getURL urlResolver,
	dryRun bool,
) *errors.StatusError {
	if preValidate == nil {
		preValidate = func(*v1.VirtualMachineInstance) *errors.StatusError { return nil }
	}
	if postProcessError == nil {
		postProcessError = func(_ *v1.VirtualMachineInstance, err error) error { return err }
	}

	vmi, url, conn, statusErr := h.prepareConnection(ctx, namespace, name, preValidate, getURL)
	if statusErr != nil {
		err := postProcessError(vmi, fmt.Errorf("%s", statusErr.ErrStatus.Message))
		statusErr.ErrStatus.Message = err.Error()
		return statusErr
	}
	if dryRun {
		return nil
	}
	if err := conn.Put(url, body); err != nil {
		return errors.NewInternalError(postProcessError(vmi, err))
	}
	return nil
}

func (h *Handler) prepareConnection(
	ctx context.Context,
	namespace, name string,
	validate validation,
	getURL urlResolver,
) (*v1.VirtualMachineInstance, string, kubecli.VirtHandlerConn, *errors.StatusError) {
	vmi, statusErr := h.fetchAndValidateVirtualMachineInstance(ctx, namespace, name, validate)
	if statusErr != nil {
		return vmi, "", nil, statusErr
	}

	conn, err := h.getVirtHandlerConnection(vmi)
	if err != nil {
		statusErr = errors.NewBadRequest(err.Error())
		log.Log.Object(vmi).Reason(statusErr).Error("Unable to establish connection to virt-handler")
		return vmi, "", nil, statusErr
	}

	url, err := getURL(vmi, conn)
	if err != nil {
		statusErr = errors.NewBadRequest(err.Error())
		log.Log.Object(vmi).Reason(statusErr).Error("Unable to retrieve target handler URL")
		return vmi, "", conn, statusErr
	}
	return vmi, url, conn, nil
}

func (h *Handler) fetchAndValidateVirtualMachineInstance(
	ctx context.Context,
	namespace, name string,
	validate validation,
) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := h.virtCli.VirtualMachineInstance(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		var statusErr *errors.StatusError
		if errors.IsNotFound(err) {
			statusErr = errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		} else {
			statusErr = errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
		}
		log.Log.Reason(statusErr).Errorf("Failed to gather vmi %s in namespace %s.", name, namespace)
		return nil, statusErr
	}
	if statusErr := validate(vmi); statusErr != nil {
		return vmi, statusErr
	}
	return vmi, nil
}

func (h *Handler) getVirtHandlerConnection(vmi *v1.VirtualMachineInstance) (kubecli.VirtHandlerConn, error) {
	if !vmi.IsRunning() && !vmi.IsScheduled() {
		return nil, fmt.Errorf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s or %s", vmi.Status.Phase, v1.Running, v1.Scheduled)
	}
	return kubecli.NewVirtHandlerClient(h.virtCli, h.httpClient).Port(h.consoleServerPort).ForNode(vmi.Status.NodeName), nil
}

func decodeBodyReader(body io.Reader, bodyStruct interface{}) *errors.StatusError {
	err := yaml.NewYAMLOrJSONDecoder(body, 1024).Decode(bodyStruct)
	switch err {
	case io.EOF, nil:
		return nil
	default:
		return errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err))
	}
}
