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
 * Copyright The KubeVirt Authors
 *
 */

package reset

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_RESET = "reset"
)

func NewResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reset (VMI)",
		Short:   "Reset a virtual machine instance",
		Args:    cobra.ExactArgs(1),
		Example: usage(COMMAND_RESET),
		RunE:    Run,
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage(cmd string) string {
	usage := fmt.Sprintf("  # %s a virtualmachineinstance called 'myvmi':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s myvmi", cmd)
	return usage
}

func Run(cmd *cobra.Command, args []string) error {
	vmi := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	if err = virtClient.VirtualMachineInstance(namespace).Reset(context.Background(), vmi); err != nil {
		return fmt.Errorf("Error reseting VirtualMachineInstance %s: %v", vmi, err)
	}

	cmd.Printf("VMI %s was scheduled to %s\n", vmi, COMMAND_RESET)

	return nil
}
