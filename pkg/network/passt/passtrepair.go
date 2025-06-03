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

package passt

import (
	"context"
	"errors"
	"kubevirt.io/client-go/log"
	"os"
	"os/exec"
)

const executableName = "passt-repair"

type PasstRepairRunner struct {
	cmd                  string
	unixDomainSocketPath string
}

func NewPasstRepairRunner(unixDomainSocketPath string) *PasstRepairRunner {
	return &PasstRepairRunner{
		cmd:                  executableName,
		unixDomainSocketPath: unixDomainSocketPath,
	}
}

func (s *PasstRepairRunner) RunContextual(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, s.cmd, s.unixDomainSocketPath)
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Log.Reason(err).Errorf("failed to start passt-repair with %s", s.unixDomainSocketPath)
		return err
	}
	log.Log.V(4).Infof("passt-repair executed with %s", s.unixDomainSocketPath)

	if err := cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Log.Warningf("deadline exceeded running passt-repair with %s", s.unixDomainSocketPath)
			return context.DeadlineExceeded
		}
		log.Log.Reason(err).Errorf("failure waiting for passt-repair with %s", s.unixDomainSocketPath)
		return err
	}
	log.Log.V(4).Infof("passt-repair execution completed with %s", s.unixDomainSocketPath)
	return nil
}
