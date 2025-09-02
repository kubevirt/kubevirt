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

package testing

import (
	"bytes"

	"kubevirt.io/kubevirt/pkg/virtctl"
)

func NewRepeatableVirtctlCommand(args ...string) func() error {
	return func() error {
		cmd := virtctl.NewVirtctlCommand()
		cmd.SetArgs(args)
		return cmd.Execute()
	}
}

func NewRepeatableVirtctlCommandWithOut(args ...string) func() ([]byte, error) {
	return func() ([]byte, error) {
		out := &bytes.Buffer{}
		cmd := virtctl.NewVirtctlCommand()
		cmd.SetArgs(args)
		cmd.SetOut(out)
		err := cmd.Execute()
		return out.Bytes(), err
	}
}
