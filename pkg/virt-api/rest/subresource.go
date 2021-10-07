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
	"reflect"
	"strings"
	"sync"

	"github.com/emicklei/go-restful"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"

	"kubevirt.io/kubevirt/pkg/util/status"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const defaultProfilerComponentPort = 8443

type SubresourceAPIApp struct {
	virtCli                 kubecli.KubevirtClient
	consoleServerPort       int
	profilerComponentPort   int
	handlerTLSConfiguration *tls.Config
	credentialsLock         *sync.Mutex
	statusUpdater           *status.VMStatusUpdater
	clusterConfig           *virtconfig.ClusterConfig
}

func NewSubresourceAPIApp(virtCli kubecli.KubevirtClient, consoleServerPort int, tlsConfiguration *tls.Config, clusterConfig *virtconfig.ClusterConfig) *SubresourceAPIApp {
	return &SubresourceAPIApp{
		virtCli:                 virtCli,
		consoleServerPort:       consoleServerPort,
		profilerComponentPort:   defaultProfilerComponentPort,
		credentialsLock:         &sync.Mutex{},
		handlerTLSConfiguration: tlsConfiguration,
		statusUpdater:           status.NewVMStatusUpdater(virtCli),
		clusterConfig:           clusterConfig,
	}
}

type validation func(*v1.VirtualMachineInstance) (err *errors.StatusError)
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

func (app *SubresourceAPIApp) putRequestHandler(request *restful.Request, response *restful.Response, validate validation, getVirtHandlerURL URLResolver, dryRun bool) {
	_, url, conn, statusErr := app.prepareConnection(request, validate, getVirtHandlerURL)
	if statusErr != nil {
		writeError(statusErr, response)
		return
	}

	if dryRun {
		return
	}
	err := conn.Put(url, app.handlerTLSConfiguration, request.Request.Body)
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
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
	if !vmi.IsRunning() {
		return nil, goerror.New(fmt.Sprintf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s", vmi.Status.Phase, v1.Running))
	}
	return kubecli.NewVirtHandlerClient(app.virtCli).Port(app.consoleServerPort).ForNode(vmi.Status.NodeName), nil
}

func getChangeRequestJson(vm *v1.VirtualMachine, changes ...v1.VirtualMachineStateChangeRequest) (string, error) {

	var ops []string

	verb := "add"
	// Special case: if there's no status field at all, add one.
	newStatus := v1.VirtualMachineStatus{}
	if reflect.DeepEqual(vm.Status, newStatus) {
		for _, change := range changes {
			newStatus.StateChangeRequests = append(newStatus.StateChangeRequests, change)
		}
		statusJson, err := json.Marshal(newStatus)
		if err != nil {
			return "", err
		}
		ops = append(ops, fmt.Sprintf(`{ "op": "%s", "path": "/status", "value": %s}`, verb, string(statusJson)))
	} else {

		failOnConflict := true
		if len(changes) == 1 && changes[0].Action == v1.StopRequest {
			// If this is a stopRequest, replace all existing StateChangeRequests.
			failOnConflict = false
		}

		if len(vm.Status.StateChangeRequests) != 0 {
			if failOnConflict {
				return "", fmt.Errorf("unable to complete request: stop/start already underway")
			} else {
				verb = "replace"
			}
		}

		changeRequests := []v1.VirtualMachineStateChangeRequest{}
		for _, change := range changes {
			changeRequests = append(changeRequests, change)
		}

		oldChangeRequestsJson, err := json.Marshal(vm.Status.StateChangeRequests)
		if err != nil {
			return "", err
		}

		newChangeRequestsJson, err := json.Marshal(changeRequests)
		if err != nil {
			return "", err
		}

		test := fmt.Sprintf(`{ "op": "test", "path": "/status/stateChangeRequests", "value": %s}`, string(oldChangeRequestsJson))
		update := fmt.Sprintf(`{ "op": "%s", "path": "/status/stateChangeRequests", "value": %s}`, verb, string(newChangeRequestsJson))

		ops = append(ops, test)
		ops = append(ops, update)
	}

	if vm.Status.StartFailure != nil {
		ops = append(ops, `{ "op": "remove", "path": "/status/startFailure" }`)
	}

	return string(controller.GeneratePatchBytes(ops)), nil
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

func (app *SubresourceAPIApp) MigrateVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, err := app.fetchVirtualMachine(name, namespace)
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
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running")), response)
		return
	}

	for _, c := range vm.Status.Conditions {
		if c.Type == v1.VirtualMachinePaused && c.Status == v12.ConditionTrue {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is paused")), response)
			return
		}
	}

	createMigrationJob := func() *errors.StatusError {
		_, err := app.virtCli.VirtualMachineInstanceMigration(namespace).Create(&v1.VirtualMachineInstanceMigration{
			ObjectMeta: k8smetav1.ObjectMeta{
				GenerateName: "kubevirt-migrate-vm-",
			},
			Spec: v1.VirtualMachineInstanceMigrationSpec{
				VMIName: name,
			},
		})
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
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.RestartOptions{}

	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s", err)), response)
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

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}
	if runStrategy == v1.RunStrategyHalted {
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support manual restart requests", v1.RunStrategyHalted)), response)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			writeError(errors.NewInternalError(err), response)
			return
		}
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running: %v", v1.RunStrategyHalted)), response)
		return
	}

	bodyString, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
		v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
	if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	log.Log.Object(vm).V(4).Infof("Patching VM: %s", bodyString)
	err = app.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(bodyString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	if err != nil {
		if strings.Contains(err.Error(), "jsonpatch test operation does not apply") {
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
			// set termincationGracePeriod and delete the VMI pod to trigger a forced restart
			err = app.virtCli.CoreV1().Pods(namespace).Delete(context.Background(), vmiPodname, k8smetav1.DeleteOptions{GracePeriodSeconds: bodyStruct.GracePeriodSeconds})
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

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
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

	startPaused := false
	startChangeRequestData := make(map[string]string)
	bodyStruct := &v1.StartOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s", err)), response)
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
	switch runStrategy {
	case v1.RunStrategyHalted:
		pausedStartStrategy := v1.StartStrategyPaused
		// Send start request if VM should start paused. virt-controller will update RunStrategy upon this request.
		// No need to send the request if StartStrategy is already set to Paused in VMI Spec.
		if startPaused && (vm.Spec.Template == nil || vm.Spec.Template.Spec.StartStrategy != &pausedStartStrategy) {
			patchString, err := getChangeRequestJson(vm, v1.VirtualMachineStateChangeRequest{
				Action: v1.StartRequest,
				Data:   startChangeRequestData,
			})
			if err != nil {
				writeError(errors.NewInternalError(err), response)
				return
			}
			log.Log.Object(vm).V(4).Infof("Patching VM status: %s", patchString)
			patchErr = app.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(patchString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
		} else {
			patchString := getRunningJson(vm, true)
			log.Log.Object(vm).V(4).Infof("Patching VM: %s", patchString)
			_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(vm.GetName(), types.MergePatchType, []byte(patchString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
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

		var bodyString string
		if needsRestart {
			bodyString, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		} else {
			bodyString, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest, Data: startChangeRequestData})
		}
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
		log.Log.Object(vm).V(4).Infof("Patching VM status: %s", bodyString)
		patchErr = app.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(bodyString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	case v1.RunStrategyAlways:
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support manual start requests", v1.RunStrategyAlways)), response)
		return
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), "jsonpatch test operation does not apply") {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, patchErr), response)
		} else {
			writeError(errors.NewInternalError(patchErr), response)
		}
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) StopVMRequestHandler(request *restful.Request, response *restful.Response) {
	// RunStrategyHalted         -> doesn't make sense
	// RunStrategyManual         -> send stop request
	// RunStrategyAlways         -> spec.running = false
	// RunStrategyRerunOnFailure -> spec.running = false

	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	bodyStruct := &v1.StopOptions{}
	if request.Request.Body != nil {
		err := yaml.NewYAMLOrJSONDecoder(request.Request.Body, 1024).Decode(&bodyStruct)
		switch err {
		case io.EOF, nil:
			break
		default:
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s", err)), response)
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
	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		hasVMI = false
	} else if err != nil {
		writeError(errors.NewInternalError(err), response)
		return
	}

	patchType := types.MergePatchType
	var patchErr error
	if hasVMI && !vmi.IsFinal() && bodyStruct.GracePeriod != nil {
		bodyString := getUpdateTerminatingSecondsGracePeriod(*bodyStruct.GracePeriod)
		log.Log.Object(vmi).V(2).Infof("Patching VMI: %s", bodyString)
		_, err = app.virtCli.VirtualMachineInstance(namespace).Patch(vmi.GetName(), patchType, []byte(bodyString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
	}

	switch runStrategy {
	case v1.RunStrategyHalted:
		if !hasVMI || vmi.IsFinal() {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running")), response)
			return
		}
		writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("%v does not support manual stop requests", v1.RunStrategyHalted)), response)
		return
	case v1.RunStrategyManual:
		if !hasVMI || vmi.IsFinal() {
			writeError(errors.NewConflict(v1.Resource("virtualmachine"), name, fmt.Errorf("VM is not running")), response)
			return
		}
		// pass the buck and ask virt-controller to stop the VM. this way the
		// VM will retain RunStrategy = manual
		patchType = types.JSONPatchType
		bodyString, err := getChangeRequestJson(vm,
			v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
		if err != nil {
			writeError(errors.NewInternalError(err), response)
			return
		}
		log.Log.Object(vm).V(4).Infof("Patching VM status: %s", bodyString)
		patchErr = app.statusUpdater.PatchStatus(vm, patchType, []byte(bodyString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyAlways:
		bodyString := getRunningJson(vm, false)
		log.Log.Object(vm).V(4).Infof("Patching VM: %s", bodyString)
		_, patchErr = app.virtCli.VirtualMachine(namespace).Patch(vm.GetName(), patchType, []byte(bodyString), &k8smetav1.PatchOptions{DryRun: bodyStruct.DryRun})
	}

	if patchErr != nil {
		if strings.Contains(patchErr.Error(), "jsonpatch test operation does not apply") {
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
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VM is not running"))
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
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s", err)), response)
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

	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not paused"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstancePaused) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not paused"))
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
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s", err)), response)
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
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VM is not running"))
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
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not running"))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UnfreezeURI(vmi)
	}
	app.putRequestHandler(request, response, validate, getURL, false)

}

func (app *SubresourceAPIApp) fetchVirtualMachine(name string, namespace string) (*v1.VirtualMachine, *errors.StatusError) {

	vm, err := app.virtCli.VirtualMachine(namespace).Get(name, &k8smetav1.GetOptions{})
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

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
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
	vm, err := app.virtCli.VirtualMachine(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.NewNotFound(v1.Resource("virtualmachine"), name)
		}
		return nil, errors.NewInternalError(fmt.Errorf("unable to retrieve vm [%s]: %v", name, err))
	}

	if !vm.Status.Created {
		return nil, errors.NewConflict(v1.Resource("virtualmachine"), vm.Name, fmt.Errorf("VMI is not started"))
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
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
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not running"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI does not have guest agent connected"))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.GuestInfoURI(vmi)
	}

	_, url, conn, err := app.prepareConnection(request, validate, getURL)
	if err != nil {
		log.Log.Errorf("Cannot prepare connection %s", err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, conErr := conn.Get(url, app.handlerTLSConfiguration)
	if conErr != nil {
		log.Log.Errorf("Cannot GET request %s", conErr.Error())
		response.WriteError(http.StatusInternalServerError, conErr)
		return
	}

	guestInfo := v1.VirtualMachineInstanceGuestAgentInfo{}
	if err := json.Unmarshal([]byte(resp), &guestInfo); err != nil {
		log.Log.Reason(err).Error("error unmarshalling guest agent response")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(guestInfo)
}

// UserList handles the subresource for providing VM guest user list
func (app *SubresourceAPIApp) UserList(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not running"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI does not have guest agent connected"))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UserListURI(vmi)
	}

	_, url, conn, err := app.prepareConnection(request, validate, getURL)
	if err != nil {
		log.Log.Errorf("Cannot prepare connection %s", err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, conErr := conn.Get(url, app.handlerTLSConfiguration)
	if conErr != nil {
		log.Log.Errorf("Cannot GET request %s", conErr.Error())
		response.WriteError(http.StatusInternalServerError, conErr)
		return
	}

	userList := v1.VirtualMachineInstanceGuestOSUserList{}
	if err := json.Unmarshal([]byte(resp), &userList); err != nil {
		log.Log.Reason(err).Error("error unmarshalling user list response")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(userList)
}

// FilesystemList handles the subresource for providing guest filesystem list
func (app *SubresourceAPIApp) FilesystemList(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi == nil || vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI is not running"))
		}
		condManager := controller.NewVirtualMachineInstanceConditionManager()
		if !condManager.HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf("VMI does not have guest agent connected"))
		}
		return nil
	}
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FilesystemListURI(vmi)
	}

	_, url, conn, err := app.prepareConnection(request, validate, getURL)
	if err != nil {
		log.Log.Errorf("Cannot prepare connection %s", err.Error())
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	resp, conErr := conn.Get(url, app.handlerTLSConfiguration)
	if conErr != nil {
		log.Log.Errorf("Cannot GET request %s", conErr.Error())
		response.WriteError(http.StatusInternalServerError, conErr)
		return
	}

	filesystemList := v1.VirtualMachineInstanceFileSystemList{}
	if err := json.Unmarshal([]byte(resp), &filesystemList); err != nil {
		log.Log.Reason(err).Error("error unmarshalling file system list response")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	response.WriteEntity(filesystemList)
}

func generateVMVolumeRequestPatch(vm *v1.VirtualMachine, volumeRequest *v1.VirtualMachineVolumeRequest) (string, error) {
	verb := getPatchVerb(vm.Status.VolumeRequests)
	vmCopy := vm.DeepCopy()

	// We only validate the list against other items in the list at this point.
	// The VM validation webhook will validate the list against the VMI spec
	// during the Patch command
	if volumeRequest.AddVolumeOptions != nil {
		if err := addAddVolumeRequests(vm, volumeRequest, vmCopy); err != nil {
			return "", err
		}
	} else if volumeRequest.RemoveVolumeOptions != nil {
		if err := addRemoveVolumeRequests(vm, volumeRequest, vmCopy); err != nil {
			return "", err
		}
	}

	oldJson, err := json.Marshal(vm.Status.VolumeRequests)
	if err != nil {
		return "", err
	}
	newJson, err := json.Marshal(vmCopy.Status.VolumeRequests)
	if err != nil {
		return "", err
	}

	test := fmt.Sprintf(`{ "op": "test", "path": "/status/volumeRequests", "value": %s}`, string(oldJson))
	update := fmt.Sprintf(`{ "op": "%s", "path": "/status/volumeRequests", "value": %s}`, verb, string(newJson))
	patch := fmt.Sprintf("[%s, %s]", test, update)

	return patch, nil
}

func getPatchVerb(requests []v1.VirtualMachineVolumeRequest) string {
	verb := "add"
	if len(requests) > 0 {
		verb = "replace"
	}
	return verb
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

func generateVMIVolumeRequestPatch(vmi *v1.VirtualMachineInstance, volumeRequest *v1.VirtualMachineVolumeRequest) (string, error) {

	volumeVerb := "add"
	diskVerb := "add"

	if len(vmi.Spec.Volumes) > 0 {
		volumeVerb = "replace"
	}

	if len(vmi.Spec.Domain.Devices.Disks) > 0 {
		diskVerb = "replace"
	}

	foundRemoveVol := false
	for _, volume := range vmi.Spec.Volumes {
		if volumeRequest.AddVolumeOptions != nil && volume.Name == volumeRequest.AddVolumeOptions.Name {
			return "", fmt.Errorf("Unable to add volume [%s] because it already exists", volume.Name)
		} else if volumeRequest.RemoveVolumeOptions != nil && volume.Name == volumeRequest.RemoveVolumeOptions.Name {
			foundRemoveVol = true
		}
	}

	if volumeRequest.RemoveVolumeOptions != nil && !foundRemoveVol {
		return "", fmt.Errorf("Unable to remove volume [%s] because it does not exist", volumeRequest.RemoveVolumeOptions.Name)
	}

	vmiCopy := vmi.DeepCopy()
	vmiCopy.Spec = *controller.ApplyVolumeRequestOnVMISpec(&vmiCopy.Spec, volumeRequest)

	oldVolumesJson, err := json.Marshal(vmi.Spec.Volumes)
	if err != nil {
		return "", err
	}

	newVolumesJson, err := json.Marshal(vmiCopy.Spec.Volumes)
	if err != nil {
		return "", err
	}

	oldDisksJson, err := json.Marshal(vmi.Spec.Domain.Devices.Disks)
	if err != nil {
		return "", err
	}

	newDisksJson, err := json.Marshal(vmiCopy.Spec.Domain.Devices.Disks)
	if err != nil {
		return "", err
	}

	testVolumes := fmt.Sprintf(`{ "op": "test", "path": "/spec/volumes", "value": %s}`, string(oldVolumesJson))
	updateVolumes := fmt.Sprintf(`{ "op": "%s", "path": "/spec/volumes", "value": %s}`, volumeVerb, string(newVolumesJson))

	testDisks := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/devices/disks", "value": %s}`, string(oldDisksJson))
	updateDisks := fmt.Sprintf(`{ "op": "%s", "path": "/spec/domain/devices/disks", "value": %s}`, diskVerb, string(newDisksJson))

	patch := fmt.Sprintf("[%s, %s, %s, %s]", testVolumes, testDisks, updateVolumes, updateDisks)

	return patch, nil
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
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s", err)), response)
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
			writeError(errors.NewBadRequest(fmt.Sprintf("Can not unmarshal Request body to struct, error: %s",
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
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, fmt.Errorf("VMI is not running"))
	}

	patch, err := generateVMIVolumeRequestPatch(vmi, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), name, err)
	}

	log.Log.Object(vmi).V(4).Infof("Patching VMI: %s", patch)
	if _, err := app.virtCli.VirtualMachineInstance(vmi.Namespace).Patch(vmi.Name, types.JSONPatchType, []byte(patch), &k8smetav1.PatchOptions{}); err != nil {
		log.Log.Object(vmi).V(1).Errorf("unable to patch vmi: %v", err)
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

	patch, err := generateVMVolumeRequestPatch(vm, volumeRequest)
	if err != nil {
		return errors.NewConflict(v1.Resource("virtualmachine"), name, err)
	}

	log.Log.Object(vm).V(4).Infof("Patching VM: %s", patch)
	if err := app.statusUpdater.PatchStatus(vm, types.JSONPatchType, []byte(patch), &k8smetav1.PatchOptions{}); err != nil {
		log.Log.Object(vm).V(1).Errorf("unable to patch vm status: %v", err)
		if errors.IsInvalid(err) {
			if statErr, ok := err.(*errors.StatusError); ok {
				return statErr
			}
		}
		return errors.NewInternalError(fmt.Errorf("unable to patch vm status: %v", err))
	}
	return nil
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
