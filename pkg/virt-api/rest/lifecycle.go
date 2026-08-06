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
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

func (app *SubresourceAPIApp) BackupVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
	validate := func(vmi *v1.VirtualMachineInstance) *errors.StatusError {
		if vmi.Status.Phase != v1.Running {
			return errors.NewConflict(v1.Resource("virtualmachineinstance"), vmi.Name, fmt.Errorf(vmNotRunning))
		}
		return nil
	}

	getURL := func(vmi *v1.VirtualMachineInstance, conn kubecli.VirtHandlerConn) (string, error) {
		return conn.BackupURI(vmi)
	}

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}

func (app *SubresourceAPIApp) RedefineCheckpointVMI(ctx context.Context, namespace, name string, body io.ReadCloser) *errors.StatusError {
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

	return app.connectVirtHandler(ctx, namespace, name, body, validate, nil, getURL, false)
}
