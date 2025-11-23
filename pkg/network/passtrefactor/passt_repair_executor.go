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
 *
 */

package passtrefactor

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"
)

type passtRepairRunner struct{}

func newRepairHandler() *passtRepairRunner {
	return &passtRepairRunner{}
}

func (h passtRepairRunner) RunCommand(socketOrDir string, vmi *v1.VirtualMachineInstance) {

	const passtRepairEnforcedTimeout = 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), passtRepairEnforcedTimeout)
	defer cancel()

	const passtRepairBinaryName = "passt-repair"
	passtRepairCommand := exec.CommandContext(ctx, passtRepairBinaryName, socketOrDir)

	const debugLevel = 4
	log.Log.V(debugLevel).Infof("executing passt-repair : %s", passtRepairCommand.String())

	if stdOutErr, err := passtRepairCommand.CombinedOutput(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Log.Errorf("deadline exceeded running: %s, %q, %s", passtRepairCommand.String(), context.DeadlineExceeded, stdOutErr)
			return
		}
		log.Log.Errorf("failed to run %s, %v, %s", passtRepairCommand.String(), err, stdOutErr)
		return
	}
	log.Log.V(debugLevel).Infof("execution of: %s has completed", passtRepairCommand.String())
}

func (h passtRepairRunner) FindSocket(socketDir string) (string, error) {

	const passtRepairSocketSuffix = ".socket.repair"

	pattern := filepath.Join(socketDir, "*"+passtRepairSocketSuffix)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("glob %q: %w", pattern, err)
	}
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("passt-repair socket not found in %s", socketDir)
}
