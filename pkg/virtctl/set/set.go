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
 * Copyright 2024 The KubeVirt Contributors
 *
 */

package set

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewSetCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "set (VM) [--cpu=CPU_COUNT] [--memory=SIZE]",
		Short:   "Set the number of CPUs and/or memory for a running virtual machine instance.",
		Example: setUsage(),
		Args:    templates.ExactArgs("set", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, args, clientConfig)
		},
	}

	cmd.Flags().Uint32("cpu", 0, "Number of CPUs (sockets) to set (must be greater than 0)")
	cmd.Flags().String("memory", "", "Memory size to set (e.g., 2Gi)")
	cmd.Flags().Duration("timeout", 10*time.Second, "Timeout duration (e.g., 10s, 2m)")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func setUsage() string {
	return `  # Set the number of CPUs (sockets) to 2 and memory to 1Gi for VirtualMachine 'myvm':
  {{ProgramName}} set myvm --cpu=2 --memory=1Gi`
}

func run(cmd *cobra.Command, args []string, clientConfig clientcmd.ClientConfig) error {
	vmName := args[0]
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}

	cpuSocketsCount, err := cmd.Flags().GetUint32("cpu")
	if err != nil {
		return err
	}

	guestMemorySizeStr, err := cmd.Flags().GetString("memory")
	if err != nil {
		return err
	}

	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return err
	}

	var cpuSet, memorySet bool
	var guestMemorySize resource.Quantity

	if cmd.Flags().Changed("cpu") {
		cpuSet = true
		if cpuSocketsCount == 0 {
			return fmt.Errorf("invalid CPU count: %d; must be greater than 0", cpuSocketsCount)
		}
	}

	if cmd.Flags().Changed("memory") {
		memorySet = true
		guestMemorySize, err = resource.ParseQuantity(guestMemorySizeStr)
		if err != nil {
			return fmt.Errorf("invalid memory size: %s", guestMemorySizeStr)
		}
		if guestMemorySize.IsZero() || guestMemorySize.Sign() == -1 {
			return fmt.Errorf("memory size must be greater than zero")
		}
	}

	if !cpuSet && !memorySet {
		return fmt.Errorf("at least one of --cpu or --memory must be set")
	}

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	vm, err := virtCli.VirtualMachine(namespace).Get(ctx, vmName, k8smetav1.GetOptions{})
	if err != nil {
		return err
	}

	messages, err := setCPUAndMemory(ctx, virtCli, vm, cpuSet, cpuSocketsCount, memorySet, guestMemorySize)
	if err != nil {
		return err
	}

	for _, message := range messages {
		cmd.Printf("%s\n", message)
	}
	return nil
}

func setCPUAndMemory(ctx context.Context, virtCli kubecli.KubevirtClient, vm *v1.VirtualMachine, cpuSet bool, cpuSocketsCount uint32, memorySet bool, guestMemorySize resource.Quantity) ([]string, error) {
	var messages []string
	patchSet := patch.New()

	if cpuSet {
		cpuToPatch := vm.Spec.Template.Spec.Domain.CPU
		if cpuToPatch == nil {
			cpuToPatch = &v1.CPU{}
		}
		cpuToPatch.Sockets = cpuSocketsCount

		patchSet.AddOption(patch.WithReplace("/spec/template/spec/domain/cpu", cpuToPatch))
		messages = append(messages, fmt.Sprintf("Successfully set %d CPUs for %s", cpuSocketsCount, vm.Name))
	}

	if memorySet {
		memoryToPatch := vm.Spec.Template.Spec.Domain.Memory
		if memoryToPatch == nil {
			memoryToPatch = &v1.Memory{}
		}
		memoryToPatch.Guest = &guestMemorySize

		patchSet.AddOption(patch.WithReplace("/spec/template/spec/domain/memory", memoryToPatch))
		messages = append(messages, fmt.Sprintf("Successfully set memory to %s for %s", guestMemorySize.String(), vm.Name))
	}

	patchPayload, err := patchSet.GeneratePayload()
	if err != nil {
		return nil, fmt.Errorf("failed to generate patch payload: %w", err)
	}

	if _, err := virtCli.VirtualMachine(vm.Namespace).Patch(ctx, vm.Name, types.JSONPatchType, patchPayload, k8smetav1.PatchOptions{}); err != nil {
		return nil, fmt.Errorf("could not patch the virtual machine: %w", err)
	}

	return messages, nil
}
