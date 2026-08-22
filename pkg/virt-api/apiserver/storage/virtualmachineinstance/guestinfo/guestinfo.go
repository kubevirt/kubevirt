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

package guestinfo

// Invalidate stale bazel remote cache entries after CI CacheNotFoundException.

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	vmiNotRunning    = "VMI is not running"
	vmiGuestAgentErr = "VMI does not have guest agent connected"
	prepConnErrFmt   = "Cannot prepare connection %s"
	getRequestErrFmt = "Cannot GET request %s"
)

type validation func(*v1.VirtualMachineInstance) *errors.StatusError
type urlResolver func(*v1.VirtualMachineInstance, kubecli.VirtHandlerConn) (string, error)

// Handler contains the VirtualMachineInstance guest-info operations served by
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

func (h *Handler) GetGuestOSInfo(ctx context.Context, namespace, name string) (interface{}, error) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.GuestInfoURI(vmi)
	}
	return h.httpGetVirtHandler(ctx, namespace, name, guestAgentValidation, getURL, v1.VirtualMachineInstanceGuestAgentInfo{})
}

func (h *Handler) GetUserList(ctx context.Context, namespace, name string) (interface{}, error) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.UserListURI(vmi)
	}
	return h.httpGetVirtHandler(ctx, namespace, name, guestAgentValidation, getURL, v1.VirtualMachineInstanceGuestOSUserList{})
}

func (h *Handler) GetFilesystemList(ctx context.Context, namespace, name string) (interface{}, error) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.FilesystemListURI(vmi)
	}
	return h.httpGetVirtHandler(ctx, namespace, name, guestAgentValidation, getURL, v1.VirtualMachineInstanceFileSystemList{})
}

func guestAgentValidation(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	if vmi == nil || vmi.Status.Phase != v1.Running {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiNotRunning))
	}
	if !controller.NewVirtualMachineInstanceConditionManager().HasCondition(vmi, v1.VirtualMachineInstanceAgentConnected) {
		return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmiGuestAgentErr))
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
		log.Log.Errorf(prepConnErrFmt, statusErr.Error())
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
