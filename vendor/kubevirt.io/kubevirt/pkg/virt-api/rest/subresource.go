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
	"crypto/tls"
	"crypto/x509"
	goerror "errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/util/cert"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
)

type SubresourceAPIApp struct {
	virtCli                 kubecli.KubevirtClient
	consoleServerPort       int
	consoleTLSConfiguration *tls.Config
	credentialsLock         *sync.Mutex
}

func NewSubresourceAPIApp(virtCli kubecli.KubevirtClient, consoleServerPort int) *SubresourceAPIApp {
	return &SubresourceAPIApp{
		virtCli:           virtCli,
		consoleServerPort: consoleServerPort,
		credentialsLock:   &sync.Mutex{},
	}
}

type requestType struct {
	socketName string
}

const (
	clientCertBytesValue      = "client-cert-bytes"
	clientKeyBytesValue       = "client-key-bytes"
	signingCertBytesValue     = "signing-cert-bytes"
	virtHandlerCertSecretName = "kubevirt-virt-handler-certs"
)

type validation func(*v1.VirtualMachineInstance) error
type URLResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

func (app *SubresourceAPIApp) streamRequestHandler(request *restful.Request, response *restful.Response, validate validation, getConsoleURL URLResolver) {
	vmiName := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vmi, code, err := app.fetchVirtualMachineInstance(vmiName, namespace)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to gather vmi %s in namespace %s.", vmiName, namespace)
		response.WriteError(code, err)
		return
	}

	if err := validate(vmi); err != nil {
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	var url string
	if conn, err := app.getVirtHandlerConnForVMI(vmi); err == nil {
		if url, err = getConsoleURL(vmi, conn); err != nil {
			log.Log.Object(vmi).Reason(err).Error("Unable to retrieve target console URL")
			response.WriteError(http.StatusBadRequest, err)
			return
		}
	} else {
		log.Log.Object(vmi).Reason(err).Error("Unable to establish connection to virt-handler")
		response.WriteError(http.StatusBadRequest, err)
		return
	}

	if app.consoleTLSConfiguration == nil {
		setTLSConfiguration := func() error {
			app.credentialsLock.Lock()
			defer app.credentialsLock.Unlock()
			if app.consoleTLSConfiguration == nil {
				tlsConfig, err := app.getConsoleTLSConfig()
				if err != nil {
					return err
				}
				app.consoleTLSConfiguration = tlsConfig
			}
			return nil
		}
		if err := setTLSConfiguration(); err != nil {
			log.Log.Object(vmi).Reason(err).Error("Failed to set TLS configuration for console/vnc connection")
			response.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	upgrader := kubecli.NewUpgrader()
	clientSocket, err := upgrader.Upgrade(response.ResponseWriter, request.Request, nil)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to upgrade client websocket connection")
		response.WriteError(http.StatusBadRequest, err)
		return
	}
	defer clientSocket.Close()

	conn, _, err := kubecli.Dial(url, app.consoleTLSConfiguration)
	if err != nil {
		log.Log.Object(vmi).Reason(err).Error("failed to dial virt-handler for a console connection")
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	defer conn.Close()

	copyErr := make(chan error)
	go func() {
		_, err := kubecli.Copy(clientSocket, conn)
		log.Log.Object(vmi).Reason(err).Error("error encountered reading from virt-handler stream")
		copyErr <- err
	}()

	go func() {
		_, err := kubecli.Copy(conn, clientSocket)
		log.Log.Object(vmi).Reason(err).Error("error encountered reading from client stream")
		copyErr <- err
	}()

	// wait for copy to finish and check the result
	if err = <-copyErr; err == nil || err == io.EOF {
		response.WriteHeader(http.StatusOK)
	} else {
		log.Log.Object(vmi).Reason(err).Error("Error in websocket proxy")
		response.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *SubresourceAPIApp) getConsoleTLSConfig() (*tls.Config, error) {
	ns, err := clientutil.GetNamespace()
	if err != nil {
		return nil, err
	}
	secret, err := app.virtCli.CoreV1().Secrets(ns).Get(virtHandlerCertSecretName, k8smetav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var clientCertBytes, clientKeyBytes, signingCertBytes []byte
	var ok bool
	// retrieve self signed cert info from secret
	if clientCertBytes, ok = secret.Data[clientCertBytesValue]; !ok {
		return nil, fmt.Errorf("%s value not found in %s virt-api secret", clientCertBytesValue, virtHandlerCertSecretName)
	}
	if clientKeyBytes, ok = secret.Data[clientKeyBytesValue]; !ok {
		return nil, fmt.Errorf("%s value not found in %s virt-api secret", clientKeyBytesValue, virtHandlerCertSecretName)
	}
	if signingCertBytes, ok = secret.Data[signingCertBytesValue]; !ok {
		return nil, fmt.Errorf("%s value not found in %s virt-api secret", signingCertBytesValue, virtHandlerCertSecretName)
	}
	clientCert, err := tls.X509KeyPair(clientCertBytes, clientKeyBytes)
	if err != nil {
		return nil, err
	}
	caCert, err := cert.ParseCertsPEM(signingCertBytes)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	for _, crt := range caCert {
		certPool.AddCert(crt)
	}

	// we use the same TLS configuration that is used for live migrations
	consoleTLSConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientCAs:  certPool,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
			return &clientCert, nil
		},
		GetCertificate: func(info *tls.ClientHelloInfo) (i *tls.Certificate, e error) {
			return &clientCert, nil
		},
		// Neither the client nor the server should validate anything itself, `VerifyPeerCertificate` is still executed
		InsecureSkipVerify: true,
		// XXX: We need to verify the cert ourselves because we don't have DNS or IP on the certs at the moment
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {

			// impossible with RequireAnyClientCert
			if len(rawCerts) == 0 {
				return fmt.Errorf("no client certificate provided.")
			}

			c, err := x509.ParseCertificate(rawCerts[0])
			if err != nil {
				return fmt.Errorf("failed to parse peer certificate: %v", err)
			}
			_, err = c.Verify(x509.VerifyOptions{
				Roots:     certPool,
				KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			})

			if err != nil {
				return fmt.Errorf("could not verify peer certificate: %v", err)
			}
			return nil
		},
		ClientAuth: tls.RequireAndVerifyClientCert,
	}
	return consoleTLSConfig, nil
}

func (app *SubresourceAPIApp) VNCRequestHandler(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) error {
		// If there are no graphics devices present, we can't proceed
		if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && *vmi.Spec.Domain.Devices.AutoattachGraphicsDevice == false {
			err := fmt.Errorf("No graphics devices are present.")
			log.Log.Object(vmi).Reason(err).Error("Can't establish VNC connection.")
			return err
		}
		return nil
	}
	getConsoleURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SetPort(app.consoleServerPort).VNCURI(vmi)
	}
	app.streamRequestHandler(request, response, validate, getConsoleURL)
}

func (app *SubresourceAPIApp) getVirtHandlerConnForVMI(vmi *v1.VirtualMachineInstance) (kubecli.VirtHandlerConn, error) {
	if !vmi.IsRunning() {
		return nil, goerror.New(fmt.Sprintf("Unable to connect to VirtualMachineInstance because phase is %s instead of %s", vmi.Status.Phase, v1.Running))
	}
	return kubecli.NewVirtHandlerClient(app.virtCli).ForNode(vmi.Status.NodeName), nil
}

func (app *SubresourceAPIApp) ConsoleRequestHandler(request *restful.Request, response *restful.Response) {
	validate := func(vmi *v1.VirtualMachineInstance) error {
		// always valid
		return nil
	}
	getConsoleURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.SetPort(app.consoleServerPort).ConsoleURI(vmi)
	}
	app.streamRequestHandler(request, response, validate, getConsoleURL)
}

func getChangeRequestJson(vm *v1.VirtualMachine, changes ...v1.VirtualMachineStateChangeRequest) (string, error) {
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
		update := fmt.Sprintf(`{ "op": "%s", "path": "/status", "value": %s}`, verb, string(statusJson))

		return fmt.Sprintf("[%s]", update), nil
	}

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
	return fmt.Sprintf("[%s, %s]", test, update), nil
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

func (app *SubresourceAPIApp) RestartVMRequestHandler(request *restful.Request, response *restful.Response) {
	// RunStrategyHalted         -> doesn't make sense
	// RunStrategyManual         -> send restart request
	// RunStrategyAlways         -> send restart request
	// RunStrategyRerunOnFailure -> send restart request
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, code, err := app.fetchVirtualMachine(name, namespace)
	if err != nil {
		response.WriteError(code, err)
		return
	}

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	if runStrategy == v1.RunStrategyHalted {
		response.WriteError(http.StatusForbidden, fmt.Errorf("%v does not support manual restart requests", v1.RunStrategyHalted))
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
		response.WriteError(http.StatusForbidden, fmt.Errorf("VM is not running"))
		return
	}

	bodyString, err := getChangeRequestJson(vm,
		v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
		v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}

	log.Log.Object(vm).V(4).Infof("Patching VM: %s", bodyString)
	_, err = app.virtCli.VirtualMachine(namespace).Patch(vm.GetName(), types.JSONPatchType, []byte(bodyString))
	if err != nil {
		errCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "jsonpatch test operation does not apply") {
			errCode = http.StatusConflict
		}
		response.WriteError(errCode, fmt.Errorf("%v: %s", err, bodyString))
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) StartVMRequestHandler(request *restful.Request, response *restful.Response) {
	name := request.PathParameter("name")
	namespace := request.PathParameter("namespace")

	vm, code, err := app.fetchVirtualMachine(name, namespace)
	if err != nil {
		response.WriteError(code, err)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
	}
	if vmi != nil && !vmi.IsFinal() && vmi.Status.Phase != v1.Unknown && vmi.Status.Phase != v1.VmPhaseUnset {
		response.WriteError(http.StatusForbidden, fmt.Errorf("VM is already running"))
		return
	}

	bodyString := ""
	patchType := types.MergePatchType

	runStrategy, err := vm.RunStrategy()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	// RunStrategyHalted         -> spec.running = true
	// RunStrategyManual         -> send start request
	// RunStrategyAlways         -> doesn't make sense
	// RunStrategyRerunOnFailure -> doesn't make sense
	switch runStrategy {
	case v1.RunStrategyHalted:
		bodyString = getRunningJson(vm, true)
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyManual:
		patchType = types.JSONPatchType

		needsRestart := false
		if (runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Succeeded) ||
			(runStrategy == v1.RunStrategyManual && vmi != nil && vmi.IsFinal()) {
			needsRestart = true
		} else if runStrategy == v1.RunStrategyRerunOnFailure && vmi != nil && vmi.Status.Phase == v1.Failed {
			response.WriteError(http.StatusForbidden, fmt.Errorf("%v does not support starting VM from failed state", v1.RunStrategyRerunOnFailure))
			return
		}

		if needsRestart {
			bodyString, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID},
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
		} else {
			bodyString, err = getChangeRequestJson(vm,
				v1.VirtualMachineStateChangeRequest{Action: v1.StartRequest})
		}
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
	case v1.RunStrategyAlways:
		response.WriteError(http.StatusForbidden, fmt.Errorf("%v does not support manual start requests", v1.RunStrategyAlways))
		return
	}

	log.Log.Object(vm).V(4).Infof("Patching VM: %s", bodyString)
	_, err = app.virtCli.VirtualMachine(namespace).Patch(vm.GetName(), patchType, []byte(bodyString))
	if err != nil {
		errCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "jsonpatch test operation does not apply") {
			errCode = http.StatusConflict
		}
		response.WriteError(errCode, fmt.Errorf("%v: %s", err, bodyString))
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

	vm, code, err := app.fetchVirtualMachine(name, namespace)
	if err != nil {
		response.WriteError(code, err)
		return
	}

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			response.WriteError(http.StatusInternalServerError, err)
			return
		} else {
			response.WriteError(http.StatusForbidden, fmt.Errorf("VM is not running"))
			return
		}
	}
	if vmi == nil || vmi.IsFinal() || vmi.Status.Phase == v1.Unknown || vmi.Status.Phase == v1.VmPhaseUnset {
		response.WriteError(http.StatusForbidden, fmt.Errorf("VM is not running"))
		return
	}

	bodyString := ""
	patchType := types.MergePatchType
	runStrategy, err := vm.RunStrategy()
	if err != nil {
		response.WriteError(http.StatusInternalServerError, err)
		return
	}
	switch runStrategy {
	case v1.RunStrategyHalted:
		response.WriteError(http.StatusForbidden, fmt.Errorf("%v does not support manual stop requests", v1.RunStrategyHalted))
		return
	case v1.RunStrategyManual:
		// pass the buck and ask virt-controller to stop the VM. this way the
		// VM will retain RunStrategy = manual
		patchType = types.JSONPatchType
		bodyString, err = getChangeRequestJson(vm,
			v1.VirtualMachineStateChangeRequest{Action: v1.StopRequest, UID: &vmi.UID})
		if err != nil {
			response.WriteError(http.StatusInternalServerError, err)
			return
		}
	case v1.RunStrategyRerunOnFailure, v1.RunStrategyAlways:
		bodyString = getRunningJson(vm, false)
	}

	log.Log.Object(vm).V(4).Infof("Patching VM: %s", bodyString)
	_, err = app.virtCli.VirtualMachine(namespace).Patch(vm.GetName(), patchType, []byte(bodyString))
	if err != nil {
		errCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "jsonpatch test operation does not apply") {
			errCode = http.StatusConflict
		}
		response.WriteError(errCode, fmt.Errorf("%v: %s", err, bodyString))
		return
	}

	response.WriteHeader(http.StatusAccepted)
}

func (app *SubresourceAPIApp) fetchVirtualMachine(name string, namespace string) (*v1.VirtualMachine, int, error) {

	vm, err := app.virtCli.VirtualMachine(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, http.StatusNotFound, fmt.Errorf("VirtualMachine %s in namespace %s not found", name, namespace)
		}
		return nil, http.StatusInternalServerError, err
	}
	return vm, http.StatusOK, nil
}

func (app *SubresourceAPIApp) fetchVirtualMachineInstance(name string, namespace string) (*v1.VirtualMachineInstance, int, error) {

	vmi, err := app.virtCli.VirtualMachineInstance(namespace).Get(name, &k8smetav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, http.StatusNotFound, goerror.New(fmt.Sprintf("VirtualMachineInstance %s in namespace %s not found.", name, namespace))
		}
		return nil, http.StatusInternalServerError, err
	}
	return vmi, 0, nil
}
