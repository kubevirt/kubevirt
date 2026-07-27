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
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype/expand"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const (
	unmarshalRequestErrFmt                   = "Can not unmarshal Request body to struct, error: %s"
	vmNotRunning                             = "VM is not running"
	vmSnapshotInprogress                     = "VM snapshot is in progress"
	patchingVMFmt                            = "Patching VM: %s"
	jsonpatchTestErr                         = "jsonpatch test operation does not apply"
	patchingVMStatusFmt                      = "Patching VM status: %s"
	vmiNotRunning                            = "VMI is not running"
	vmiNotPaused                             = "VMI is not paused"
	vmiGuestAgentErr                         = "VMI does not have guest agent connected"
	prepConnectionErrFmt                     = "Cannot prepare connection %s"
	getRequestErrFmt                         = "Cannot GET request %s"
	featureGateDisabledErrFmt                = "'%s' feature gate is not enabled"
	defaultProfilerComponentPort             = 8443
	volumeMigrationManualRecoveryRequiredErr = "VM recovery required: Volume migration failed, leaving some volumes pointing to non-consistent targets; manual intervention is needed to reassign them to their original volumes."
)

type instancetypeVMExpander interface {
	Expand(vm *v1.VirtualMachine) (*v1.VirtualMachine, error)
}

type SubresourceAPIApp struct {
	virtClient              kubecli.KubevirtClient
	k8sClient               kubernetes.Interface
	consoleServerPort       int
	profilerComponentPort   int
	handlerTLSConfiguration *tls.Config
	clusterConfig           *virtconfig.ClusterConfig
	instancetypeExpander    instancetypeVMExpander
	handlerHttpClient       *http.Client
}

func NewSubresourceAPIApp(virtClient kubecli.KubevirtClient, k8sClient kubernetes.Interface, consoleServerPort int, tlsConfiguration *tls.Config, clusterConfig *virtconfig.ClusterConfig) *SubresourceAPIApp {
	// When this method is called from tools/openapispec.go when running 'make generate',
	// the virtClient is nil, and accessing GeneratedKubeVirtClient() would cause nil dereference.
	var instancetypeExpander instancetypeVMExpander
	if virtClient != nil {
		instancetypeExpander = expand.New(
			clusterConfig,
			find.NewSpecFinder(nil, nil, nil, virtClient),
			preferenceFind.NewSpecFinder(nil, nil, nil, virtClient),
		)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfiguration,
		},
		Timeout: 10 * time.Second,
	}

	return &SubresourceAPIApp{
		virtClient:              virtClient,
		k8sClient:               k8sClient,
		consoleServerPort:       consoleServerPort,
		profilerComponentPort:   defaultProfilerComponentPort,
		handlerTLSConfiguration: tlsConfiguration,
		clusterConfig:           clusterConfig,
		instancetypeExpander:    instancetypeExpander,
		handlerHttpClient:       httpClient,
	}
}

type validation func(*v1.VirtualMachineInstance) (err *errors.StatusError)

// This function prototype is used with putRequestHandlerWithErrorPostProcessing.
// The errorPostProcessing function will get called if an error occurs when attempting
// to make a request to virt-handler. Depending on where in the stack the error occurred
// the VMI might be nil.
//
// Use this function to inject more human readible context into the error response.
type errorPostProcessing func(*v1.VirtualMachineInstance, error) (err error)
type URLResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

func (app *SubresourceAPIApp) prepareConnection(ctx context.Context, namespace, name string, validate validation, getVirtHandlerURL URLResolver) (vmi *v1.VirtualMachineInstance, url string, conn kubecli.VirtHandlerConn, statusError *errors.StatusError) {

	vmi, statusError = app.fetchAndValidateVirtualMachineInstance(ctx, namespace, name, validate)
	if statusError != nil {
		return
	}

	url, conn, statusError = app.getVirtHandlerFor(vmi, getVirtHandlerURL)
	if statusError != nil {
		return
	}

	return
}

func (app *SubresourceAPIApp) fetchAndValidateVirtualMachineInstance(ctx context.Context, namespace, vmiName string, validate validation) (vmi *v1.VirtualMachineInstance, statusError *errors.StatusError) {
	vmi, err := app.virtClient.VirtualMachineInstance(namespace).Get(ctx, vmiName, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			statusError = errors.NewNotFound(v1.Resource("virtualmachineinstance"), vmiName)
		} else {
			statusError = errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", vmiName, err))
		}
		log.Log.Reason(statusError).Errorf("Failed to gather vmi %s in namespace %s.", vmiName, namespace)
		return
	}

	if statusError = validate(vmi); statusError != nil {
		return
	}
	return
}

func (app *SubresourceAPIApp) putRequestHandler(request *restful.Request, response *restful.Response, preValidate validation, getVirtHandlerURL URLResolver, dryRun bool) {

	app.putRequestHandlerWithErrorPostProcessing(request, response, preValidate, nil, getVirtHandlerURL, dryRun)
}

func (app *SubresourceAPIApp) putRequestHandlerWithErrorPostProcessing(request *restful.Request, response *restful.Response, preValidate validation, errorPostProcessing errorPostProcessing, getVirtHandlerURL URLResolver, dryRun bool) {

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if statusErr := app.connectVirtHandler(request.Request.Context(), namespace, name, request.Request.Body, preValidate, errorPostProcessing, getVirtHandlerURL, dryRun); statusErr != nil {
		writeError(statusErr, response)
	}
}

// connectVirtHandler validates the VMI, resolves the virt-handler URL and, unless
// dryRun is set, proxies the request body to virt-handler via a PUT
func (app *SubresourceAPIApp) connectVirtHandler(ctx context.Context, namespace, name string, body io.ReadCloser, preValidate validation, errorPostProcessing errorPostProcessing, getVirtHandlerURL URLResolver, dryRun bool) *errors.StatusError {

	if preValidate == nil {
		preValidate = func(vmi *v1.VirtualMachineInstance) *errors.StatusError { return nil }
	}
	if errorPostProcessing == nil {
		errorPostProcessing = func(vmi *v1.VirtualMachineInstance, err error) error { return err }
	}

	vmi, url, conn, statusErr := app.prepareConnection(ctx, namespace, name, preValidate, getVirtHandlerURL)
	if statusErr != nil {
		err := errorPostProcessing(vmi, fmt.Errorf("%s", statusErr.ErrStatus.Message))
		statusErr.ErrStatus.Message = err.Error()
		return statusErr
	}

	if dryRun {
		return nil
	}
	if err := conn.Put(url, body); err != nil {
		err = errorPostProcessing(vmi, err)
		return errors.NewInternalError(err)
	}
	return nil
}

func (app *SubresourceAPIApp) httpGetVirtHandler(ctx context.Context, namespace, name string, validate validation, getURL URLResolver, v interface{}) (interface{}, error) {
	_, url, conn, err := app.prepareConnection(ctx, namespace, name, validate, getURL)
	if err != nil {
		log.Log.Errorf(prepConnectionErrFmt, err.Error())
		return nil, err
	}

	resp, conErr := conn.Get(url, restful.MIME_JSON)
	if conErr != nil {
		log.Log.Errorf(getRequestErrFmt, conErr.Error())
		return nil, conErr
	}

	if err := json.Unmarshal([]byte(resp), &v); err != nil {
		log.Log.Reason(err).Error("error unmarshalling response")
		return nil, err
	}

	return v, nil
}

func (app *SubresourceAPIApp) httpGetRequestHandler(request *restful.Request, response *restful.Response, validate validation, getURL URLResolver, v interface{}) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	result, err := app.httpGetVirtHandler(request.Request.Context(), namespace, name, validate, getURL, v)
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(result)
}

func (app *SubresourceAPIApp) httpGetRequestBinaryHandler(request *restful.Request, response *restful.Response, validate validation, getURL URLResolver) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")
	_, url, conn, err := app.prepareConnection(request.Request.Context(), namespace, name, validate, getURL)
	if err != nil {
		log.Log.Errorf(prepConnectionErrFmt, err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, conErr := conn.Get(url, "")
	if conErr != nil {
		log.Log.Errorf(getRequestErrFmt, conErr.Error())
		response.WriteError(http.StatusInternalServerError, conErr)
		return
	}

	if nbytes, err := response.Write([]byte(resp)); err != nil {
		log.Log.Reason(err).Error("Failed to write response")
		response.WriteError(http.StatusInternalServerError, err)
	} else if nbytes != len(resp) {
		err = fmt.Errorf("Failed to write full response: %d of %d written", nbytes, len(resp))
		log.Log.Reason(err).Error("Incomplete message written")
		response.WriteError(http.StatusInternalServerError, err)
	}
}

func (app *SubresourceAPIApp) fetchVirtualMachine(name string, namespace string) (*v1.VirtualMachine, *errors.StatusError) {

	vm, err := app.virtClient.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}
	return vm, nil
}

// FetchVirtualMachineInstance by namespace and name
func (app *SubresourceAPIApp) FetchVirtualMachineInstance(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {

	vmi, err := app.virtClient.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}
	return vmi, nil
}

// FetchVirtualMachineInstanceForVM by namespace and name
func (app *SubresourceAPIApp) FetchVirtualMachineInstanceForVM(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vm, err := app.virtClient.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	if !vm.Status.Created {
		return nil, errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf("VMI is not started"))
	}

	vmi, err := app.virtClient.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachineinstance"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s]: %v", name, err))
	}

	for _, ref := range vmi.OwnerReferences {
		if ref.UID == vm.UID {
			return vmi, nil
		}
	}

	return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vmi [%s] for vm: %v", name, err))
}

func writeError(error *errors.StatusError, response *restful.Response) {
	errStatus := error.ErrStatus.DeepCopy()
	errStatus.Kind = "Status"
	errStatus.APIVersion = "v1"
	err := response.WriteHeaderAndJson(int(error.Status().Code), errStatus, restful.MIME_JSON)
	if err != nil {
		log.Log.Reason(err).Error("Failed to write http response.")
	}
}

func vmiGuestAgentValidation(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if vmi == nil || vmi.Status.Phase != v1.Running {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
	}
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
	}
	return nil
}

func (app *SubresourceAPIApp) GetGuestOSInfo(ctx context.Context, namespace, name string) (interface{}, error) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.GuestInfoURI(vmi)
	}
	return app.httpGetVirtHandler(ctx, namespace, name, vmiGuestAgentValidation, getURL, v1.VirtualMachineInstanceGuestAgentInfo{})
}

// GetUserList proxies the guest OS active user list from virt-handler
func (app *SubresourceAPIApp) GetUserList(ctx context.Context, namespace, name string) (interface{}, error) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UserListURI(vmi)
	}
	return app.httpGetVirtHandler(ctx, namespace, name, vmiGuestAgentValidation, getURL, v1.VirtualMachineInstanceGuestOSUserList{})
}

// GetFilesystemList proxies the guest filesystem list from virt-handler
func (app *SubresourceAPIApp) GetFilesystemList(ctx context.Context, namespace, name string) (interface{}, error) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FilesystemListURI(vmi)
	}
	return app.httpGetVirtHandler(ctx, namespace, name, vmiGuestAgentValidation, getURL, v1.VirtualMachineInstanceFileSystemList{})
}

// GuestOSInfo handles the subresource for providing VM guest agent information
func (app *SubresourceAPIApp) GuestOSInfo(request *restful.Request, response *restful.Response) {
	result, err := app.GetGuestOSInfo(request.Request.Context(), request.PathParameter("namespace"), request.PathParameter("name"))
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(result)
}

// UserList handles the subresource for providing VM guest user list
func (app *SubresourceAPIApp) UserList(request *restful.Request, response *restful.Response) {
	result, err := app.GetUserList(request.Request.Context(), request.PathParameter("namespace"), request.PathParameter("name"))
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(result)
}

// FilesystemList handles the subresource for providing guest filesystem list
func (app *SubresourceAPIApp) FilesystemList(request *restful.Request, response *restful.Response) {
	result, err := app.GetFilesystemList(request.Request.Context(), request.PathParameter("namespace"), request.PathParameter("name"))
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	response.WriteEntity(result)
}

func decodeBody(request *restful.Request, bodyStruct interface{}) *errors.StatusError {
	return decodeBodyReader(request.Request.Body, bodyStruct)
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
