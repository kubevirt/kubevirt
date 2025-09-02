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

package vm

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	volumeNameArg         = "volume-name"
	notDefinedGracePeriod = -1
	dryRunCommandUsage    = "--dry-run=false: Flag used to set whether to perform a dry run or not. If true the command will be executed without performing any changes."

	dryRunArg      = "dry-run"
	forceArg       = "force"
	gracePeriodArg = "grace-period"
	persistArg     = "persist"

	YAML = "yaml"
	JSON = "json"
)

var (
	forceRestart bool
	gracePeriod  int64
	volumeName   string
	persist      bool
	dryRun       bool
)

type Command struct {
	command string
}

func usage(cmd string) string {
	if cmd == COMMAND_USERLIST || cmd == COMMAND_FSLIST || cmd == COMMAND_GUESTOSINFO {
		return fmt.Sprintf("  # %s a virtual machine instance called 'myvm':\n  {{ProgramName}} %s myvm", strings.Title(cmd), cmd)
	}

	return fmt.Sprintf("  # %s a virtual machine called 'myvm':\n  {{ProgramName}} %s myvm", strings.Title(cmd), cmd)
}

func setDryRunOption(dryRun bool) []string {
	if dryRun {
		fmt.Printf("Dry Run execution\n")
		return []string{metav1.DryRunAll}
	}

	return nil
}
