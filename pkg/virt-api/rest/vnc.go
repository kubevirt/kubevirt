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

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

// Screenshot proxies the VNC screenshot of a VirtualMachineInstance from
// virt-handler and returns the raw image bytes. It is served by the aggregated
// apiserver under the nested subresource path virtualmachineinstances/vnc/screenshot
func (app *SubresourceAPIApp) Screenshot(ctx context.Context, namespace, name string) ([]byte, *errors.StatusError) {
	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.ScreenshotURI(vmi)
	}

	// Screenshot without Display fails with:
	//   `Requested operation is not valid: no screens to take screenshot from`
	_, url, conn, statusErr := app.prepareConnection(ctx, namespace, name, vmiHasDisplay, getURL)
	if statusErr != nil {
		log.Log.Errorf(prepConnectionErrFmt, statusErr.Error())
		return nil, statusErr
	}

	resp, conErr := conn.Get(url, "")
	if conErr != nil {
		log.Log.Errorf(getRequestErrFmt, conErr.Error())
		return nil, errors.NewInternalError(conErr)
	}

	return []byte(resp), nil
}

func vmiHasDisplay(vmi *v1.VirtualMachineInstance) *errors.StatusError {
	// If there are no graphics devices present, we can't proceed
	if vmi.Spec.Domain.Devices.AutoattachGraphicsDevice != nil && !*vmi.Spec.Domain.Devices.AutoattachGraphicsDevice {
		err := fmt.Errorf("No graphics devices are present.")
		log.Log.Object(vmi).Reason(err).Error("Can't establish VNC connection.")
		return errors.NewBadRequest(err.Error())
	}
	if !vmi.IsRunning() {
		return errors.NewBadRequest(vmiNotRunning)
	}
	return nil
}
