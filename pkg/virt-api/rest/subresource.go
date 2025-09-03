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
	goerror "errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/emicklei/go-restful/v3"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype/expand"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/pointer"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	kutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
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
	vmiNoAttestationErr                      = "Attestation not requested for VMI"
	prepConnectionErrFmt                     = "Cannot prepare connection %s"
	getRequestErrFmt                         = "Cannot GET request %s"
	pvcVolumeModeErr                         = "pvc should be filesystem pvc"
	pvcAccessModeErr                         = "pvc access mode can't be read only"
	pvcSizeErrFmt                            = "pvc size [%s] should be bigger then [%s]"
	memoryDumpNameConflictErr                = "can't request memory dump for pvc [%s] while pvc [%s] is still associated as the memory dump pvc"
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
type errorPostProcessing func(*v1.VirtualMachineInstance, error) (err error)
type URLResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

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

// get either the interface with the provided name or the first available interface
// if no interface is present, return error
func getTargetInterfaceIP(vmi *v1.VirtualMachineInstance) (string, error) {
	interfaces := vmi.Status.Interfaces
	if len(interfaces) < 1 {
		return "", fmt.Errorf("no network interfaces are present")
	}
	return interfaces[0].IP, nil
}

func (app *SubresourceAPIApp) getVirtHandlerConnForVMI(vmi *v1.VirtualMachineInstance) (kubecli.VirtHandlerConn, error) {
	if !vmi.IsRunning() && !vmi.IsScheduled() {
		return nil, goerror.New(fmt.Sprintf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s or %s", vmi.Status.Phase, v1.Running, v1.Scheduled))
	}
	return kubecli.NewVirtHandlerClient(app.virtCli, app.handlerHttpClient).Port(app.consoleServerPort).ForNode(vmi.Status.NodeName), nil
}

func getChangeRequestJson(vm *v1.VirtualMachine, changes ...v1.VirtualMachineStateChangeRequest) ([]byte, error) {
	patchSet := patch.New()
	// Special case: if there's no status field at all, add one.
	newStatus := v1.VirtualMachineStatus{}
	if equality.Semantic.DeepEqual(vm.Status, newStatus) {
		newStatus.StateChangeRequests = append(newStatus.StateChangeRequests, changes...)
		patchSet.AddOption(patch.WithAdd("/status", newStatus))
	} else {
		patchSet.AddOption(patch.WithTest("/status/stateChangeRequests", vm.Status.StateChangeRequests))
		switch {
		case len(vm.Status.StateChangeRequests) == 0:
			patchSet.AddOption(patch.WithAdd("/status/stateChangeRequests", changes))
		case len(changes) == 1 && changes[0].Action == v1.StopRequest:
			// If this is a stopRequest, replace all existing StateChangeRequests.
			patchSet.AddOption(patch.WithReplace("/status/stateChangeRequests", changes))
		default:
			return nil, fmt.Errorf("unable to complete request: stop/start already underway")
		}
	}

	if vm.Status.StartFailure != nil {
		patchSet.AddOption(patch.WithRemove("/status/startFailure"))
	}

	return patchSet.GeneratePayload()
}

func getRunningJson(vm *v1.VirtualMachine, running bool) string {
	runStrategy := v1.RunStrategyHalted
	if running {
		runStrategy = v1.RunStrategyAlways
	}
	if vm.Spec.RunStrategy != nil {
		return fmt.Sprintf("{\"spec\":{\"runStrategy\": \"%s\"}}", runStrategy)
	} else {
		return fmt.Sprintf("{\"spec\":{\"running\": %t}}", running)
	}
}

func getUpdateTerminatingSecondsGracePeriod(gracePeriod int64) string {
	return fmt.Sprintf("{\"spec\":{\"terminationGracePeriodSeconds\": %d }}", gracePeriod)
}

func (app *SubresourceAPIApp) patchVMStatusStopped(vmi *v1.VirtualMachineInstance, vm *v1.VirtualMachine, response *restful.Response, bodyStruct *v1.StopOptions) (error, error) {
	patchBytes, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return nil, err
	}
	log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
	_, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	return err, nil
}

func (app *SubresourceAPIApp) MigrateVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.MigrateOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	}
	_, err := app.fetchVirtualMachine(name, namespace)
	if err != nil {
		writeError(err, response)
		return
	}

	vmi, err := app.FetchVirtualMachineInstance(namespace, name)
	if err != nil {
		writeError(err, response)
		return
	}

	if vmi.Status.Phase != v1.Running {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning)), response)
		return
	}

	createMigrationJob := func() *errors.StatusError {
		_, err := app.virtCli.VirtualMachineInstanceMigration(namespace).Create(context.Background(), &v1.VirtualMachineInstanceMigration{
			ObjectMeta: k8smetav1.ObjectMeta{
				GenerateName: "kubevirt-migrate-vm-",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: name,
			},
		}, k8smetav1.CreateOptions{DryRun: bodyStruct.DryRun})
		if err != nil {
			return errors.NewInternalError(err)
		}
		return nil
	}

	if err = createMigrationJob(); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) RestartVMRequestHandler(request *restful.Request, response *restful.Response) {
	// RunStrategyHalted         -> doesn't make sense
	// RunStrategyManual         -> send restart request
	// RunStrategyAlways         -> send restart request
	// RunStrategyRerunOnFailure -> send restart request
	// RunStrategyOnce           -> doesn't make sense
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.RestartOptions{}

	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	}
	if bodyStruct.GracePeriodSeconds != nil {
		if *bodyStruct.GracePeriodSeconds > 0 {
			writeError(errors.NewBadRequest(fmt.Sprintf("For force restart, only gracePeriod=0 is supported for now")), response)
			return
		} else if *bodyStruct.GracePeriodSeconds < 0 {
			writeError(errors.NewBadRequest(fmt.Sprintf("gracePeriod has to be greater or equal to 0")), response)
			return
		}
	}

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}
	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm,
		v1.VirtualMachineConditionType(v1.VirtualMachineInstanceVolumesChange), v12.ConditionTrue) {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(volumeMigrationManualRecoveryRequiredErr)), response)
		return
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	if runStrategy == v1.RunStrategyHalted || runStrategy == v1.RunStrategyOnce {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("RunStategy %v does not support manual restart requests", runStrategy)), response)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			writeError(errors.NewInternalError(err), response)
			return
		}
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running: %v", v1.RunStrategyHalted)), response)
		return
	}

	patchBytes, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
		v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	_, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	if err != nil {
		if strings.Contains(err.Error(), jsonpatchTestErr) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, err), response)
		} else {
			writeError(errors.NewInternalError(err), response)
		}
		return
	}

	// Only force restart with GracePeriodSeconds=0 is supported for now
	// Here we are deleting the Pod because CRDs don't support gracePeriodSeconds at the moment
	if bodyStruct.GracePeriodSeconds != nil {
		if *bodyStruct.GracePeriodSeconds == 0 {
			vmiPodname, err := app.findPod(namespace, vmi)
			if err != nil {
				writeError(errors.NewInternalError(err), response)
				return
			}
			if vmiPodname == "" {
				response.WriteHeader(http.StatusAccepted)
				return
			}
			// set terminationGracePeriod to 1 (which is the shorted safe restart period) and delete the VMI pod to trigger a swift restart.
			err = app.virtCli.CoreV1().Pods(namespace).Delete(context.Background(), vmiPodname, k8smetav1.DeleteOptions{GracePeriodSeconds: pointer.P(int64(1))})
			if err != nil {
				if !errors.IsNotFound(err) {
					writeError(errors.NewInternalError(err), response)
					return
				}
			}
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) findPod(namespace string, vmi *v1.VirtualMachineInstance) (string, error) {
	fieldSelector := fields.ParseSelectorOrDie("status.phase==" + string(v12.PodRunning))
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel + "=virt-launcher," + v1.CreatedByLabel + "=" + string(vmi.UID)))
	if err != nil {
		return "", err
	}
	selector := k8smetav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
	podList, err := app.virtCli.CoreV1().Pods(namespace).List(context.Background(), selector)
	if err != nil {
		return "", err
	}
	if len(podList.Items) == 0 {
		return "", nil
	} else if len(podList.Items) == 1 {
		return podList.Items[0].ObjectMeta.Name, nil
	} else {
		// If we have 2 running pods, we might have a migration. Find the new pod!
		if vmi.Status.MigrationState != nil && vmi.Status.MigrationState.Completed {
			for _, pod := range podList.Items {
				if pod.Name == vmi.Status.MigrationState.TargetPod {
					return pod.Name, nil
				}
			}
		} else {
			// fallback to old behaviour
			return podList.Items[0].ObjectMeta.Name, nil
		}
	}
	return "", nil
}

func (app *SubresourceAPIApp) StartVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			writeError(errors.NewInternalError(err), response)
			return
		}
	}
	if vmi != nil && !vmi.IsFinal() && vmi.Status.Phase != v1.Unknown && vmi.Status.Phase != v1.VmPhaseUnset {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is already running")), response)
		return
	}
	if controller.NewVirtualMachineConditionManager().HasConditionWithStatus(vm, v1.VirtualMachineManualRecoveryRequired, v12.ConditionTrue) {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(volumeMigrationManualRecoveryRequiredErr)), response)
		return
	}

	startPaused := false
	startChangeRequestData := make(map[string]string)
	bodyStruct := &v1.StartOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
		startPaused = bodyStruct.Paused
	}
	if startPaused {
		startChangeRequestData[v1.StartRequestDataPausedKey] = v1.StartRequestDataPausedTrue
	}

	var patchErr error

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	// RunStrategyHalted         -> spec.running = true / send start request for paused start
	// RunStrategyManual         -> send start request
	// RunStrategyAlways         -> doesn't make sense
	// RunStrategyRerunOnFailure -> doesn't make sense
	// RunStrategyOnce           -> doesn't make sense
	switch runStrategy {
	case v1.RunStrategyHalted:
		pausedStartStrategy := v1.StartStrategyPaused
		// Send start request if VM should start paused. virt-controller will update RunStrategy upon this request.
		// No need to send the request if StartStrategy is already set to Paused in VMI Spec.
		if startPaused && (vm.Spec.Template == nil || vm.Spec.Template.Spec.StartStrategy != &pausedStartStrategy) {
			patchBytes, err := getChangeRequestJson(vm, v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
				Data:   startChangeRequestData,
			})
			if err != nil {
				writeError(errors.NewInternalError(err), response)
				return
			}
			log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
			_, patchErr = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
		} else {
			patchString := getRunningJson(vm, true)
			log.Log.Object(vm).V(4).Infof(patchingVMFmt, patchString)
			_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(context.Background(), vm.GetName(), types.MergePatchType, []byte(patchString), k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
		}

	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		needsRestart := false
		if (runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Succeeded) ||
			(runStrategy == v1.RunStrategyManual && vmi != nil && vmi.IsFinal()) {
			needsRestart = true
		} else if runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Failed {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support starting VM from failed state", v1.RunStrategyRerunOnFailure)), response)
			return
		}

		var patchBytes []byte
		if needsRestart {
			patchBytes, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		} else {
			patchBytes, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		}
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
		log.Log.Object(vm).V(4).Infof(patchingVMStatusFmt, string(patchBytes))
		_, patchErr = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support manual start requests", runStrategy)), response)
		return
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr), response)
		} else {
			writeError(errors.NewInternalError(patchErr), response)
		}
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) StopVMRequestHandler(request *restful.Request, response *restful.Response) {
	// RunStrategyHalted         -> force stop if grace period in request is shorter than before, otherwise doesn't make sense
	// RunStrategyManual         -> send stop request
	// RunStrategyAlways         -> spec.running = false
	// RunStrategyRerunOnFailure -> send stop request
	// RunStrategyOnce           -> spec.running = false

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.StopOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	}

	vm, statusErr := app.fetchVirtualMachine(name, namespace)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	hasVMI := true
	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(context.Background(), name, k8smetav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		hasVMI = false
	} else if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	var oldGracePeriodSeconds int64
	patchType := types.MergePatchType
	var patchErr error
	if hasVMI && !vmi.IsFinal() && bodyStruct.GracePeriod != nil {
		// used for stopping a VM with RunStrategyHalted
		if vmi.Spec.TerminationGracePeriodSeconds != nil {
			oldGracePeriodSeconds = *vmi.Spec.TerminationGracePeriodSeconds
		}

		bodyString := getUpdateTerminatingSecondsGracePeriod(*bodyStruct.GracePeriod)
		log.Log.Object(vmi).V(2).Infof("Patching VMI: %s", bodyString)
		_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(context.Background(), vmi.GetName(), patchType, []byte(bodyString), k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
	}

	switch runStrategy {
	case v1.RunStrategyHalted:
		if !hasVMI || vmi.IsFinal() {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning)), response)
			return
		}
		if bodyStruct.GracePeriod == nil || (vmi.Spec.TerminationGracePeriodSeconds != nil && *bodyStruct.GracePeriod >= oldGracePeriodSeconds) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v only supports manual stop requests with a shorter graceperiod", v1.RunStrategyHalted)), response)
			return
		}
		// same behavior as RunStrategyManual
		patchErr, err = app.patchVMStatusStopped(vmi, vm, response, bodyStruct)
		if err != nil {
			return
		}
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		if !hasVMI || vmi.IsFinal() {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmNotRunning)), response)
			return
		}
		// pass the buck and ask virt-controller to stop the VM. this way the
		// VM will retain RunStrategy = manual
		patchErr, err = app.patchVMStatusStopped(vmi, vm, response, bodyStruct)
		if err != nil {
			return
		}
	case v1.RunStrategyAlways, v1.RunStrategyOnce:
		bodyString := getRunningJson(vm, false)
		log.Log.Object(vm).V(4).Infof(patchingVMFmt, bodyString)
		_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(context.Background(), vm.GetName(), patchType, []byte(bodyString), k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), jsonpatchTestErr) {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr), response)
		} else {
			writeError(errors.NewInternalError(patchErr), response)
		}
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) PauseVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		if vmi.Spec.LivenessProbe != nil {
			return errors.NewForbidden(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("Pausing VMIs with LivenessProbe is currently not supported"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is already paused"))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.PauseURI(vmi)
	}

	bodyStruct := &v1.PauseOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == k8smetav1.DryRunAll {
		dryRun = true
	}
	app.putRequestHandler(request, response, validate, getURL, dryRun)

}

func (app *SubresourceAPIApp) UnpauseVMIRequestHandler(request *restful.Request, response *restful.Response) {

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	// Check VM status - only continue if VM doesn't exist or if it exists without snapshot in progress
	vm, err := app.fetchVirtualMachine(name, namespace)
	if err != nil {
		if !errors.IsNotFound(err) {
			writeError(err, response)
			return
		}
	} else {
		// VM exists - check if snapshot is in progress
		if vm.Status.SnapshotInProgress != nil {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf(vmSnapshotInprogress)), response)
			return
		}
	}

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotPaused))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnpauseURI(vmi)
	}

	bodyStruct := &v1.UnpauseOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	}
	var dryRun bool
	if len(bodyStruct.DryRun) > 0 && bodyStruct.DryRun[0] == k8smetav1.DryRunAll {
		dryRun = true
	}
	app.putRequestHandler(request, response, validate, getURL, dryRun)

}

func (app *SubresourceAPIApp) FreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FreezeURI(vmi)
	}

	app.putRequestHandler(request, response, validate, getURL, false)
}

func (app *SubresourceAPIApp) UnfreezeVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnfreezeURI(vmi)
	}
	app.putRequestHandler(request, response, validate, getURL, false)

}

func (app *SubresourceAPIApp) ResetVMIRequestHandler(request *restful.Request, response *restful.Response) {

	// Post process any error responses in order to append human
	// readable explanation for why the reset may have failed.
	errorPostProcessing := func(vmi *v1.VirtualMachineInstance, err error) error {
		// VMI reset request could have been sent while VMI was in the process of transitioning
		// from scheduled to running.
		if vmi != nil && !vmi.IsRunning() {
			return fmt.Errorf("Failed to reset non-running VMI with phase %s: %v", vmi.Status.Phase, err)
		}
		return err
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.ResetURI(vmi)
	}

	app.putRequestHandlerWithErrorPostProcessing(request, response, nil, errorPostProcessing, getURL, false)
}

func (app *SubresourceAPIApp) SoftRebootVMIRequestHandler(request *restful.Request, response *restful.Response) {

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if condManager.HasConditionWithStatus(vmi, v1.VirtualMachineInstancePaused, v12.ConditionTrue) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is paused"))
		}
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			if features := vmi.Spec.Domain.Features; features != nil && features.ACPI.Enabled != nil && !(*features.ACPI.Enabled) {
				return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI neither have the agent connected nor the ACPI feature enabled"))
			}
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SoftRebootURI(vmi)
	}

	app.putRequestHandler(request, response, validate, getURL, false)
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

func generateVMVolumeRequestPatch(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	vmCopy := vm.DeepCopy()

	// We only validate the list against other items in the list at this point.
	// The VM validation webhook will validate the list against the VMI spec
	// during the Patch command
	if volumeRequest.AddVolumeOptions != nil {
		if err := addAddVolumeRequests(vm, volumeRequest, vmCopy); err != nil {
			return nil, err
		}
	} else if volumeRequest.RemoveVolumeOptions != nil {
		if err := addRemoveVolumeRequests(vm, volumeRequest, vmCopy); err != nil {
			return nil, err
		}
	}

	patchSet := patch.New(
		patch.WithTest("/status/volumeRequests", vm.Status.VolumeRequests),
	)

	if len(vm.Status.VolumeRequests) > 0 {
		patchSet.AddOption(patch.WithReplace("/status/volumeRequests", vmCopy.Status.VolumeRequests))
	} else {
		patchSet.AddOption(patch.WithAdd("/status/volumeRequests", vmCopy.Status.VolumeRequests))
	}
	return patchSet.GeneratePayload()
}

func addAddVolumeRequests(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest, vmCopy *v1.VirtualMachine) error {
	name := volumeRequest.AddVolumeOptions.Name
	for _, request := range vm.Status.VolumeRequests {
		if err := validateAddVolumeRequest(request, name); err != nil {
			return err
		}
	}
	vmCopy.Status.VolumeRequests = append(vm.Status.VolumeRequests, *volumeRequest)
	return nil
}

func validateAddVolumeRequest(request v1.VirtualMachineVolumeRequest, name string) error {
	if addVolumeRequestExists(request, name) {
		return fmt.Errorf("add volume request for volume [%s] already exists", name)
	}
	if removeVolumeRequestExists(request, name) {
		return fmt.Errorf("unable to add volume since a remove volume request for volume [%s] already exists and is still being processed", name)
	}
	return nil
}

func addRemoveVolumeRequests(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest, vmCopy *v1.VirtualMachine) error {
	name := volumeRequest.RemoveVolumeOptions.Name
	var volumeRequestsList []v1.VirtualMachineVolumeRequest
	for _, request := range vm.Status.VolumeRequests {
		if addVolumeRequestExists(request, name) {
			// Filter matching AddVolume requests from the new list.
			continue
		}
		if removeVolumeRequestExists(request, name) {
			return fmt.Errorf("a remove volume request for volume [%s] already exists and is still being processed", name)
		}
		volumeRequestsList = append(volumeRequestsList, request)
	}
	volumeRequestsList = append(volumeRequestsList, *volumeRequest)
	vmCopy.Status.VolumeRequests = volumeRequestsList
	return nil
}

func removeVolumeRequestExists(request v1.VirtualMachineVolumeRequest, name string) bool {
	return request.RemoveVolumeOptions != nil && request.RemoveVolumeOptions.Name == name
}

func addVolumeRequestExists(request v1.VirtualMachineVolumeRequest, name string) bool {
	return request.AddVolumeOptions != nil && request.AddVolumeOptions.Name == name
}

func volumeHotpluggable(volume v1.Volume) bool {
	return (volume.DataVolume != nil && volume.DataVolume.Hotpluggable) || (volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.Hotpluggable)
}

func volumeNameExists(volume v1.Volume, volumeName string) bool {
	return volume.Name == volumeName
}

func volumeSourceName(volumeSource *v1.HotplugVolumeSource) string {
	if volumeSource.DataVolume != nil {
		return volumeSource.DataVolume.Name
	}
	if volumeSource.PersistentVolumeClaim != nil {
		return volumeSource.PersistentVolumeClaim.ClaimName
	}
	return ""
}

func volumeSourceExists(volume v1.Volume, volumeName string) bool {
	return (volume.DataVolume != nil && volume.DataVolume.Name == volumeName) ||
		(volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == volumeName)
}

func volumeExists(volume v1.Volume, volumeName string) bool {
	return volumeNameExists(volume, volumeName) || volumeSourceExists(volume, volumeName)
}

func verifyVolumeOption(volumes []v1.Volume, volumeRequest *v1.VirtualMachineVolumeRequest) error {
	foundRemoveVol := false
	for _, volume := range volumes {
		if volumeRequest.AddVolumeOptions != nil {
			volSourceName := volumeSourceName(volumeRequest.AddVolumeOptions.VolumeSource)
			if volumeNameExists(volume, volumeRequest.AddVolumeOptions.Name) {
				return fmt.Errorf("Unable to add volume [%s] because volume with that name already exists", volumeRequest.AddVolumeOptions.Name)
			}
			if volumeSourceExists(volume, volSourceName) {
				return fmt.Errorf("Unable to add volume source [%s] because it already exists", volSourceName)
			}
		} else if volumeRequest.RemoveVolumeOptions != nil && volumeExists(volume, volumeRequest.RemoveVolumeOptions.Name) {
			if !volumeHotpluggable(volume) {
				return fmt.Errorf("Unable to remove volume [%s] because it is not hotpluggable", volume.Name)
			}
			foundRemoveVol = true
			break
		}
	}

	if volumeRequest.RemoveVolumeOptions != nil && !foundRemoveVol {
		return fmt.Errorf("Unable to remove volume [%s] because it does not exist", volumeRequest.RemoveVolumeOptions.Name)
	}

	return nil
}

func generateVMIVolumeRequestPatch(vmi *v1.VirtualMachineInstance, volumeRequest *v1.VirtualMachineVolumeRequest) ([]byte, error) {
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Spec = *controller.ApplyVolumeRequestOnVMISpec(&vmiCopy.Spec, volumeRequest)

	patchSet := patch.New(
		patch.WithTest("/spec/volumes", vmi.Spec.Volumes),
		patch.WithTest("/spec/domain/devices/disks", vmi.Spec.Domain.Devices.Disks),
	)

	if len(vmi.Spec.Volumes) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/volumes", vmiCopy.Spec.Volumes))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/volumes", vmiCopy.Spec.Volumes))
	}

	if len(vmi.Spec.Domain.Devices.Disks) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/domain/devices/disks", vmiCopy.Spec.Domain.Devices.Disks))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/domain/devices/disks", vmiCopy.Spec.Domain.Devices.Disks))
	}

	return patchSet.GeneratePayload()
}

func (app *SubresourceAPIApp) addVolumeRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.HotplugVolumesEnabled() {
		writeError(errors.NewBadRequest("Unable to Add Volume because HotplugVolumes feature gate is not enabled."), response)
		return
	}

	opts := &v1.AddVolumeOptions{}
	if request.Request.Body != nil {
		defer request.Request.Body.Close()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	} else {
		writeError(errors.NewBadRequest("Request with no body, a new name is expected as the request body"), response)
		return
	}

	if opts.Name == "" {
		writeError(errors.NewBadRequest("AddVolumeOptions requires name to be set"), response)
		return
	} else if opts.Disk == nil {
		writeError(errors.NewBadRequest("AddVolumeOptions requires disk to not be nil"), response)
		return
	} else if opts.VolumeSource == nil {
		writeError(errors.NewBadRequest("AddVolumeOptions requires VolumeSource to not be nil"), response)
		return
	}

	opts.Disk.Name = opts.Name
	volumeRequest := v1.VirtualMachineVolumeRequest{
		AddVolumeOptions: opts,
	}
	if opts.VolumeSource.DataVolume != nil {
		opts.VolumeSource.DataVolume.Hotpluggable = true
	} else if opts.VolumeSource.PersistentVolumeClaim != nil {
		opts.VolumeSource.PersistentVolumeClaim.Hotpluggable = true
	}

	// inject into VMI if ephemeral, else set as a request on the VM to both make permanent and hotplug.
	if ephemeral {
		if err := app.vmiVolumePatch(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	} else {
		if err := app.vmVolumePatchStatus(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) removeVolumeRequestHandler(request *restful.Request, response *restful.Response, ephemeral bool) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.HotplugVolumesEnabled() {
		writeError(errors.NewBadRequest("Unable to Remove Volume because HotplugVolumes feature gate is not enabled."), response)
		return
	}

	opts := &v1.RemoveVolumeOptions{}
	if request.Request.Body != nil {
		defer request.Request.Body.Close()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt,
				err)), response)
			return
		}
	} else {
		writeError(errors.NewBadRequest("Request with no body, a new name is expected as the request body"),
			response)
		return
	}

	if opts.Name == "" {
		writeError(errors.NewBadRequest("RemoveVolumeOptions requires name to be set"), response)
		return
	}
	volumeRequest := v1.VirtualMachineVolumeRequest{
		RemoveVolumeOptions: opts,
	}

	// inject into VMI if ephemeral, else set as a request on the VM to both make permanent and hotplug.
	if ephemeral {
		if err := app.vmiVolumePatch(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	} else {
		if err := app.vmVolumePatchStatus(name, namespace, &volumeRequest); err != nil {
			writeError(err, response)
			return
		}
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) vmiVolumePatch(name, namespace string, volumeRequest *v1.VirtualMachineVolumeRequest) *errors.StatusError {
	vmi, statErr := app.FetchVirtualMachineInstance(namespace, name)
	if statErr != nil {
		return statErr
	}

	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, fmt.Errorf(vmiNotRunning))
	}

	err := verifyVolumeOption(vmi.Spec.Volumes, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	patchBytes, err := generateVMIVolumeRequestPatch(vmi, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	dryRunOption := app.getDryRunOption(volumeRequest)
	log.Log.Object(vmi).V(4).Infof("Patching VMI: %s", string(patchBytes))
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: dryRunOption}); err != nil {
		log.Log.Object(vmi).Errorf("unable to patch vmi: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vmi: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) vmVolumePatchStatus(name, namespace string, volumeRequest *v1.VirtualMachineVolumeRequest) *errors.StatusError {
	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		return statErr
	}

	err := verifyVolumeOption(vm.Spec.Template.Spec.Volumes, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	patchBytes, err := generateVMVolumeRequestPatch(vm, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	dryRunOption := app.getDryRunOption(volumeRequest)
	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	if _, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{DryRun: dryRunOption}); err != nil {
		log.Log.Object(vm).Errorf("unable to patch vm status: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) getDryRunOption(volumeRequest *v1.VirtualMachineVolumeRequest) []string {
	var dryRunOption []string
	if options := volumeRequest.AddVolumeOptions; options != nil && options.DryRun != nil && options.DryRun[0] == k8smetav1.DryRunAll {
		dryRunOption = volumeRequest.AddVolumeOptions.DryRun
	} else if options := volumeRequest.RemoveVolumeOptions; options != nil && options.DryRun != nil && options.DryRun[0] == k8smetav1.DryRunAll {
		dryRunOption = volumeRequest.RemoveVolumeOptions.DryRun
	}
	return dryRunOption
}

// VMAddVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMAddVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.addVolumeRequestHandler(request, response, false)
}

// VMRemoveVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMRemoveVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.removeVolumeRequestHandler(request, response, false)
}

// VMIAddVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMIAddVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.addVolumeRequestHandler(request, response, true)
}

// VMIRemoveVolumeRequestHandler handles the subresource for hot plugging a volume and disk.
func (app *SubresourceAPIApp) VMIRemoveVolumeRequestHandler(request *restful.Request, response *restful.Response) {
	app.removeVolumeRequestHandler(request, response, true)
}

func addMemoryDumpRequest(vm, vmCopy *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) error {
	claimName := memoryDumpReq.ClaimName
	if vm.Status.MemoryDumpRequest != nil {
		if vm.Status.MemoryDumpRequest.Phase == v1.MemoryDumpDissociating {
			return fmt.Errorf("can't dump memory for pvc [%s] a remove memory dump request is in progress", claimName)
		}
		if vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpCompleted && vm.Status.MemoryDumpRequest.Phase != v1.MemoryDumpFailed {
			return fmt.Errorf("memory dump request for pvc [%s] already in progress", claimName)
		}
	}
	vmCopy.Status.MemoryDumpRequest = memoryDumpReq
	return nil
}

func removeMemoryDumpRequest(vm, vmCopy *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) error {
	if vm.Status.MemoryDumpRequest == nil {
		return fmt.Errorf("can't remove memory dump association for vm %s, no association found", vm.Name)
	}

	claimName := vm.Status.MemoryDumpRequest.ClaimName
	if vm.Status.MemoryDumpRequest.Remove {
		return fmt.Errorf("memory dump remove request for pvc [%s] already exists", claimName)
	}
	memoryDumpReq.ClaimName = claimName
	vmCopy.Status.MemoryDumpRequest = memoryDumpReq
	return nil
}

func generateVMMemoryDumpRequestPatch(vm *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest, removeRequest bool) ([]byte, error) {
	vmCopy := vm.DeepCopy()

	if !removeRequest {
		if err := addMemoryDumpRequest(vm, vmCopy, memoryDumpReq); err != nil {
			return nil, err
		}
	} else {
		if err := removeMemoryDumpRequest(vm, vmCopy, memoryDumpReq); err != nil {
			return nil, err
		}
	}

	patchSet := patch.New(patch.WithTest("/status/memoryDumpRequest", vm.Status.MemoryDumpRequest))
	if vm.Status.MemoryDumpRequest != nil {
		patchSet.AddOption(patch.WithReplace("/status/memoryDumpRequest", vmCopy.Status.MemoryDumpRequest))
	} else {
		patchSet.AddOption(patch.WithAdd("/status/memoryDumpRequest", vmCopy.Status.MemoryDumpRequest))
	}

	return patchSet.GeneratePayload()
}

func (app *SubresourceAPIApp) fetchPersistentVolumeClaim(name string, namespace string) (*v12.PersistentVolumeClaim, *errors.StatusError) {
	pvc, err := app.virtCli.CoreV1().PersistentVolumeClaims(namespace).Get(context.TODO(), name, k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("persistentvolumeclaim"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve pvc [%s]: %v", name, err))
	}
	return pvc, nil
}

func (app *SubresourceAPIApp) fetchCDIConfig() (*cdiv1.CDIConfig, *errors.StatusError) {
	cdiConfig, err := app.virtCli.CdiClient().CdiV1beta1().CDIConfigs().Get(context.Background(), storagetypes.ConfigName, k8smetav1.GetOptions{})
	if errors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve cdi config: %v", err))
	}
	return cdiConfig, nil
}

func (app *SubresourceAPIApp) validateMemoryDumpClaim(vmi *v1.VirtualMachineInstance, claimName, namespace string) *errors.StatusError {
	pvc, err := app.fetchPersistentVolumeClaim(claimName, namespace)
	if err != nil {
		return err
	}
	if storagetypes.IsPVCBlock(pvc.Spec.VolumeMode) {
		return errors.NewConflict(v1.Resource("persistentvolumeclaim"), claimName, fmt.Errorf(pvcVolumeModeErr))
	}

	if storagetypes.IsReadOnlyAccessMode(pvc.Spec.AccessModes) {
		return errors.NewConflict(v1.Resource("persistentvolumeclaim"), claimName, fmt.Errorf(pvcAccessModeErr))
	}

	pvcSize := pvc.Spec.Resources.Requests.Storage()
	scaledPvcSize := resource.NewScaledQuantity(pvcSize.ScaledValue(resource.Kilo), resource.Kilo)

	expectedMemoryDumpSize := kutil.CalcExpectedMemoryDumpSize(vmi)
	cdiConfig, err := app.fetchCDIConfig()
	if err != nil {
		return err
	}
	var expectedPvcSize *resource.Quantity
	var overheadErr error
	if cdiConfig == nil {
		log.Log.Object(vmi).V(3).Infof(storagetypes.FSOverheadMsg)
		expectedPvcSize, overheadErr = storagetypes.GetSizeIncludingDefaultFSOverhead(expectedMemoryDumpSize)
	} else {
		expectedPvcSize, overheadErr = storagetypes.GetSizeIncludingFSOverhead(expectedMemoryDumpSize, pvc.Spec.StorageClassName, pvc.Spec.VolumeMode, cdiConfig)
	}
	if overheadErr != nil {
		return errors.NewInternalError(overheadErr)
	}
	if scaledPvcSize.Cmp(*expectedPvcSize) < 0 {
		return errors.NewConflict(v1.Resource("persistentvolumeclaim"), claimName, fmt.Errorf(pvcSizeErrFmt, scaledPvcSize.String(), expectedPvcSize.String()))
	}

	return nil
}

func (app *SubresourceAPIApp) validateMemoryDumpRequest(vm *v1.VirtualMachine, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest) *errors.StatusError {
	if memoryDumpReq.ClaimName == "" && vm.Status.MemoryDumpRequest == nil {
		return errors.NewBadRequest("Memory dump requires claim name to be set")
	} else if vm.Status.MemoryDumpRequest != nil && memoryDumpReq.ClaimName != "" {
		if vm.Status.MemoryDumpRequest.ClaimName != memoryDumpReq.ClaimName {
			return errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf(memoryDumpNameConflictErr, memoryDumpReq.ClaimName, vm.Status.MemoryDumpRequest.ClaimName))
		}
	} else if vm.Status.MemoryDumpRequest != nil {
		memoryDumpReq.ClaimName = vm.Status.MemoryDumpRequest.ClaimName
	}

	vmi, statErr := app.FetchVirtualMachineInstance(vm.Namespace, vm.Name)
	if statErr != nil {
		return statErr
	}

	if !vmi.IsRunning() {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vm.Name, fmt.Errorf(vmiNotRunning))
	}

	if statErr = app.validateMemoryDumpClaim(vmi, memoryDumpReq.ClaimName, vm.Namespace); statErr != nil {
		return statErr
	}

	return nil
}

func (app *SubresourceAPIApp) vmMemoryDumpRequestPatchStatus(name, namespace string, memoryDumpReq *v1.VirtualMachineMemoryDumpRequest, removeRequest bool) *errors.StatusError {
	vm, statErr := app.fetchVirtualMachine(name, namespace)
	if statErr != nil {
		return statErr
	}

	if !removeRequest {
		statErr = app.validateMemoryDumpRequest(vm, memoryDumpReq)
		if statErr != nil {
			return statErr
		}
	}

	patchBytes, err := generateVMMemoryDumpRequestPatch(vm, memoryDumpReq, removeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	log.Log.Object(vm).V(4).Infof(patchingVMFmt, string(patchBytes))
	if _, err = app.virtCli.VirtualMachine(vm.Namespace).PatchStatus(context.Background(), vm.Name, types.JSONPatchType, patchBytes, k8smetav1.PatchOptions{}); err != nil {
		log.Log.Object(vm).Errorf("unable to patch vm status: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return nil
}

func (app *SubresourceAPIApp) MemoryDumpVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	if !app.clusterConfig.HotplugVolumesEnabled() {
		writeError(errors.NewBadRequest("Unable to memory dump because HotplugVolumes feature gate is not enabled."), response)
		return
	}

	memoryDumpReq := &v1.VirtualMachineMemoryDumpRequest{}
	if request.Request.Body != nil {
		defer request.Request.Body.Close()
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(memoryDumpReq)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
			return
		}
	} else {
		writeError(errors.NewBadRequest("Request with no body"), response)
		return
	}

	memoryDumpReq.Phase = v1.MemoryDumpAssociating
	isRemoveRequest := false
	if err := app.vmMemoryDumpRequestPatchStatus(name, namespace, memoryDumpReq, isRemoveRequest); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) RemoveMemoryDumpVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	removeReq := &v1.VirtualMachineMemoryDumpRequest{
		Phase:  v1.MemoryDumpDissociating,
		Remove: true,
	}
	isRemoveRequest := true
	if err := app.vmMemoryDumpRequestPatchStatus(name, namespace, removeReq, isRemoveRequest); err != nil {
		writeError(err, response)
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

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
	err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(opts)
	switch err {
	case io.EOF, nil:
		break
	default:
		writeError(errors.NewBadRequest(fmt.Sprintf(unmarshalRequestErrFmt, err)), response)
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
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patch, k8smetav1.PatchOptions{}); err != nil {
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
