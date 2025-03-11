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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package pause

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtV1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type virtCommand struct {
	dryRun bool
}

func NewCommand() *cobra.Command {
	c := virtCommand{}
	cmd := &cobra.Command{
		Use:   "pause vm|vmi (VM)|(VMI)",
		Short: "Pause a virtual machine",
		Long: `Pauses a virtual machine by freezing it. Machine state is kept in memory.
First argument is the resource type, possible types are (case insensitive, both singular and plural forms) virtualmachineinstance (vmi) or virtualmachine (vm).
Second argument is the name of the resource.`,
		Args:    cobra.ExactArgs(2),
		Example: usage(),
		RunE:    c.Run,
	}

	cmd.Flags().BoolVar(&c.dryRun, "dry-run", false, "--dry-run=false: Flag used to set whether to perform a dry run or not. If true the command will be executed without performing any changes.")

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	return "  # Pause a virtualmachine called 'myvm':\n  {{ProgramName}} pause vm myvm"
}

func (vc *virtCommand) Run(cmd *cobra.Command, args []string) error {
	resourceType := strings.ToLower(args[0])
	resourceName := args[1]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	var dryRunOption []string
	if vc.dryRun {
		cmd.Println("Dry Run execution")
		dryRunOption = []string{v1.DryRunAll}
	}

	return executePauseCMD(virtClient, namespace, resourceType, resourceName, dryRunOption)
}

func executePauseCMD(client kubecli.KubevirtClient, namespace, resourceType, resourceName string, dryRunOption []string) error {
	switch resourceType {
	case "virtualmachine", "vm":
		vm, err := client.VirtualMachine(namespace).Get(context.Background(), resourceName, v1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Error getting VirtualMachine %s: %v", resourceName, err)
		}

		err = client.VirtualMachineInstance(namespace).Pause(context.Background(), vm.Name, &kubevirtV1.PauseOptions{DryRun: dryRunOption})
		if errors.IsNotFound(err) {
			return handleNotFoundError(vm)
		}
		if err != nil {
			return fmt.Errorf("Error pausing VirtualMachineInstance %s: %v", vm.Name, err)
		}

	case "virtualmachineinstance", "vmi":
		err := client.VirtualMachineInstance(namespace).Pause(context.Background(), resourceName, &kubevirtV1.PauseOptions{DryRun: dryRunOption})
		if err != nil {
			return fmt.Errorf("Error pausing VirtualMachineInstance %s: %v", resourceName, err)
		}
	}
	fmt.Printf("VMI %s was scheduled to pause\n", resourceName)

	return nil
}

func handleNotFoundError(vm *kubevirtV1.VirtualMachine) error {
	runningStrategy, err := vm.RunStrategy()
	if err != nil {
		return fmt.Errorf("Error pausing VirtualMachineInstance %s: %v", vm.Name, err)
	}
	if runningStrategy == kubevirtV1.RunStrategyHalted {
		return fmt.Errorf("Error pausing VirtualMachineInstance %s. VirtualMachine %s is not set to run", vm.Name, vm.Name)
	}
	return fmt.Errorf("Error pausing VirtualMachineInstance %s, it was not found", vm.Name)
}
