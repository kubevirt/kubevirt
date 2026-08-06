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

package backup

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const vmNotRunning = "VM is not running"

type validation func(*v1.VirtualMachineInstance) *errors.StatusError
type errorPostProcessing func(*v1.VirtualMachineInstance, error) error
type urlResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

// Handler contains the VirtualMachineInstance backup operations served by
// the GenericAPIServer storage Connecters.
type Handler struct {
	virtCli           kubecli.KubevirtClient
	consoleServerPort int
	httpClient        *http.Client
}

func NewHandler(virtCli kubecli.KubevirtClient, consoleServerPort int, tlsConfig *tls.Config) *Handler {
	return &Handler{
		virtCli:           virtCli,
		consoleServerPort: consoleServerPort,
		httpClient: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   10 * time.Second,
		},
	}
}

func (h *Handler) BackupVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.BackupURI(vmi)
	}

	return h.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (h *Handler) RedefineCheckpointVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.ChangedBlockTracking == nil ||
			vmi.Status.ChangedBlockTracking.State != v1.ChangedBlockTrackingEnabled {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name,
				fmt.Errorf("ChangedBlockTracking is not enabled"))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.RedefineCheckpointURI(vmi)
	}

	return h.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
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
