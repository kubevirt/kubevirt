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

package vsock

import (
	"fmt"
	"net"

	"github.com/mdlayher/vsock"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virtvsock "kubevirt.io/kubevirt/pkg/vsock"
	"kubevirt.io/kubevirt/pkg/vsock/mode"
)

type vsockDialFunc func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error)

type netnsDoFunc func(pid int, fn func() error) error

type Dialer struct {
	isolationDetector isolation.PodIsolationDetector
	procPath          string
	netnsDoFn         netnsDoFunc
	dialFn            vsockDialFunc
}

func NewDialer(isolationDetector isolation.PodIsolationDetector, procPath string, netnsFn netnsDoFunc, dialFn vsockDialFunc) *Dialer {
	return &Dialer{
		isolationDetector: isolationDetector,
		procPath:          procPath,
		dialFn:            dialFn,
		netnsDoFn:         netnsFn,
	}
}

func (d *Dialer) Dial(vmi *v1.VirtualMachineInstance, port uint32) (net.Conn, error) {
	if vmi.Status.VSOCKCID == nil {
		return nil, fmt.Errorf("VSOCK is not enabled for the VM")
	}

	isolationRes, err := d.isolationDetector.Detect(vmi)
	if err != nil {
		return nil, fmt.Errorf("failed to detect pod isolation: %w", err)
	}

	cid := *vmi.Status.VSOCKCID
	vsockMode := mode.ModeGlobal

	var conn net.Conn
	nsErr := d.netnsDoFn(isolationRes.Pid(), func() error {
		if mode.VsockNsMode(d.procPath) == mode.ModeLocal {
			cid = virtvsock.LocalCID
			vsockMode = mode.ModeLocal
		}

		log.Log.Object(vmi).Infof("Connecting to %d:%d in %s mode", cid, port, vsockMode)
		c, err := d.dialFn(cid, port, &vsock.Config{})
		if err != nil {
			return fmt.Errorf("failed to dial VSOCK %d:%d: %w", cid, port, err)
		}
		log.Log.Object(vmi).Infof("Connected to %d:%d in %s mode", cid, port, vsockMode)
		conn = c
		return nil
	})
	if nsErr != nil {
		return nil, nsErr
	}
	return conn, nil
}
