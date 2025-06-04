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
	"fmt"
	"os"
	"os/exec"
)

const executableName = "passt-repair"

type Command interface {
	Start() error
	Wait() error
	String() string
}

type PasstRepairCommand struct {
	cmd *exec.Cmd
}

func NewPasstRepairCommand(ctx context.Context, unixDomainSocketPath string) *PasstRepairCommand {
	c := exec.CommandContext(ctx, executableName, unixDomainSocketPath)
	c.Stderr = os.Stderr
	return &PasstRepairCommand{cmd: c}
}

func (r *PasstRepairCommand) Start() error {
	return r.cmd.Start()
}

func (r *PasstRepairCommand) Wait() error {
	return r.cmd.Wait()
}

func (r *PasstRepairCommand) String() string {
	if r.cmd.Process != nil {
		return fmt.Sprintf("%s, PID: %d", r.cmd.String(), r.cmd.Process.Pid)
	}
	return r.cmd.String()
}
