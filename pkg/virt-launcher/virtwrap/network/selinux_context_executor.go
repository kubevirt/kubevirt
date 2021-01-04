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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"
	"runtime"

	"github.com/opencontainers/selinux/go-selinux"

	kvselinux "kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

const virtHandlerSELinuxLabel = "system_u:system_r:spc_t:s0"

func isSELinuxEnabled() bool {
	_, selinuxEnabled, err := kvselinux.NewSELinux()
	return err == nil && selinuxEnabled
}

func setVirtHandlerSELinuxSocketContext(virtLauncherPID int) error {
	virtLauncherSELinuxLabel, err := getProcessCurrentSELinuxLabel(virtLauncherPID)
	if err != nil {
		return fmt.Errorf("error reading virt-launcher %d selinux label. Reason: %v", virtLauncherPID, err)
	}

	runtime.LockOSThread()
	if err := selinux.SetSocketLabel(virtLauncherSELinuxLabel); err != nil {
		return fmt.Errorf("failed to set selinux socket context to %s. Reason: %v", virtLauncherSELinuxLabel, err)
	}
	return nil
}

func resetVirtHandlerSELinuxSocketContext() error {
	err := selinux.SetSocketLabel(virtHandlerSELinuxLabel)
	runtime.UnlockOSThread()
	return err
}
