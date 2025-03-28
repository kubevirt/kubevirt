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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/api/errors"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
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
	virtCli                 kubecli.KubevirtClient
	consoleServerPort       int
	profilerComponentPort   int
	handlerTLSConfiguration *tls.Config
	credentialsLock         *sync.Mutex
	clusterConfig           *virtconfig.ClusterConfig
	instancetypeExpander    instancetypeVMExpander
	handlerHttpClient       *http.Client
}

func NewSubresourceAPIApp(virtCli kubecli.KubevirtClient, consoleServerPort int, tlsConfiguration *tls.Config, clusterConfig *virtconfig.ClusterConfig) *SubresourceAPIApp {
	// When this method is called from tools/openapispec.go when running 'make generate',
	// the virtCli is nil, and accessing GeneratedKubeVirtClient() would cause nil dereference.
	var instancetypeExpander instancetypeVMExpander
	if virtCli != nil {
		instancetypeExpander = expand.New(
			clusterConfig,
			find.NewSpecFinder(nil, nil, nil, virtCli),
			preferenceFind.NewSpecFinder(nil, nil, nil, virtCli),
		)
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfiguration,
		},
		Timeout: 10 * time.Second,
	}

	return &SubresourceAPIApp{
		virtCli:                 virtCli,
		consoleServerPort:       consoleServerPort,
		profilerComponentPort:   defaultProfilerComponentPort,
		credentialsLock:         &sync.Mutex{},
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
type (
	errorPostProcessing func(*v1.VirtualMachineInstance, error) (err error)
	URLResolver         func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)
)

func (app *SubresourceAPIApp) prepareConnection(request *restful.Request, validate validation, getVirtHandlerURL URLResolver) (vmi *v1.VirtualMachineInstance, url string, conn kubecli.VirtHandlerConn, statusError *errors.StatusError) {
	vmiName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vmi, statusError = app.fetchAndValidateVirtualMachineInstance(namespace, vmiName, validate)
	if statusError != nil {
		return
	}

	url, conn, statusError = app.getVirtHandlerFor(vmi, getVirtHandlerURL)
	if statusError != nil {
		return
	}

	return
}

func (app *SubresourceAPIApp) fetchAndValidateVirtualMachineInstance(namespace, vmiName string, validate validation) (vmi *v1.VirtualMachineInstance, statusError *errors.StatusError) {
	vmi, statusError = app.FetchVirtualMachineInstance(namespace, vmiName)
	if statusError != nil {
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
	if preValidate == nil {
		preValidate = func(vmi *v1.VirtualMachineInstance) *errors.StatusError { return nil }
	}
	if errorPostProcessing == nil {
		errorPostProcessing = func(vmi *v1.VirtualMachineInstance, err error) error { return err }
	}

	vmi, url, conn, statusErr := app.prepareConnection(request, preValidate, getVirtHandlerURL)
	if statusErr != nil {
		err := errorPostProcessing(vmi, fmt.Errorf(statusErr.ErrStatus.Message))
		statusErr.ErrStatus.Message = err.Error()
		writeError(statusErr, response)
		return
	}

	if dryRun {
		return
	}
	err := conn.Put(url, request.Request.Body)
	if err != nil {
		err = errorPostProcessing(vmi, err)
		writeError(errors.NewInternalError(err), response)
		return
	}
}

func (app *SubresourceAPIApp) httpGetRequestHandler(request *restful.Request, response *restful.Response, validate validation, getURL URLResolver, v interface{}) {
	_, url, conn, err := app.prepareConnection(request, validate, getURL)
	if err != nil {
		log.Log.Errorf(prepConnectionErrFmt, err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, conErr := conn.Get(url)
	if conErr != nil {
		log.Log.Errorf(getRequestErrFmt, conErr.Error())
		response.WriteError(http.StatusInternalServerError, conErr)
		return
	}

	if err := json.Unmarshal([]byte(resp), &v); err != nil {
		log.Log.Reason(err).Error("error unmarshalling response")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(v)
}

func (app *SubresourceAPIApp) fetchVirtualMachine(name string, namespace string) (*v1.VirtualMachine, *errors.StatusError) {
	vm, err := app.virtCli.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
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
	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
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
	vm, err := app.virtCli.VirtualMachine(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	if !vm.Status.Created {
		return nil, errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf("VMI is not started"))
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
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

// GuestOSInfo handles the subresource for providing VM guest agent information
func (app *SubresourceAPIApp) GuestOSInfo(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.GuestInfoURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.VirtualMachineInstanceGuestAgentInfo{})
}

// UserList handles the subresource for providing VM guest user list
func (app *SubresourceAPIApp) UserList(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UserListURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.VirtualMachineInstanceGuestOSUserList{})
}

// FilesystemList handles the subresource for providing guest filesystem list
func (app *SubresourceAPIApp) FilesystemList(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FilesystemListURI(vmi)
	}

	app.httpGetRequestHandler(request, response, validate, getURL, v1.VirtualMachineInstanceFileSystemList{})
}

func decodeBody(request *restful.Request, bodyStruct interface{}) *errors.StatusError {
	err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
	switch err {
	case io.EOF, nil:
		return nil
	default:
		return errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err))
	}
}
