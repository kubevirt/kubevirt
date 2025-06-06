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
)

type PasstRepairRunner struct {
	cmd Command
}

func NewPasstRepairRunner(command Command) *PasstRepairRunner {
	return &PasstRepairRunner{
		cmd: command,
	}
}

func (s *PasstRepairRunner) RunContextual(ctx context.Context, cleanupFunc func()) error {
	defer cleanupFunc()
	if err := s.cmd.Start(); err != nil {
		log.Log.Reason(err).Errorf("failed to start: %s", s.cmd.String())
		return err
	}
	log.Log.V(4).Infof("passt-repair executed: %s", s.cmd.String())

	if err := s.cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Log.Warningf("deadline exceeded running: %s", s.cmd.String())
			return context.DeadlineExceeded
		}
		log.Log.Reason(err).Errorf("failure waiting for %s to complete", s.cmd.String())
		return err
	}
	log.Log.V(4).Infof("execution of:  %s has completed", s.cmd.String())
	return nil
}
