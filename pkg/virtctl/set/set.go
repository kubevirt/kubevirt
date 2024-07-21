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

type Set struct {
	clientConfig    clientcmd.ClientConfig
	cpuSocketsCount uint32
	memorySize      string
}

func NewSetCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	s := &Set{clientConfig: clientConfig}

	cmd := &cobra.Command{
		Use:     "set (VM) [--cpu=CPU_COUNT] [--memory=SIZE]",
		Short:   "Set the number of CPUs and/or memory for a running virtual machine.",
		Example: setUsage(),
		Args:    templates.ExactArgs("set", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return s.Run(cmd, args)
		},
	}

	cmd.Flags().Uint32Var(&s.cpuSocketsCount, "cpu", 0, "Number of CPUs (sockets) to set (must be greater than 0)")
	cmd.Flags().StringVar(&s.memorySize, "memory", "", "Guest Memory size to set (e.g., 2Gi)")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func setUsage() string {
	return `  # Set the number of CPUs (sockets) to 2 and memory to 1Gi for VirtualMachine 'myvm':
  {{ProgramName}} set myvm --cpu=2 --memory=1Gi`
}

func (s *Set) Run(cmd *cobra.Command, args []string) error {
	vmName := args[0]
	namespace, _, err := s.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(s.clientConfig)
	if err != nil {
		return err
	}

	vm, err := virtCli.VirtualMachine(namespace).Get(context.Background(), vmName, k8smetav1.GetOptions{})
	if err != nil {
		return err
	}

	patchSet := patch.New()
	var messages []string

	if cmd.Flags().Changed("cpu") {
		if vm.Spec.Template.Spec.Domain.CPU == nil {
			cpuToAdd := &v1.CPU{Sockets: s.cpuSocketsCount}
			patchSet.AddOption(patch.WithAdd("/spec/template/spec/domain/cpu", cpuToAdd))
			messages = append(messages, fmt.Sprintf("Successfully added CPU with %d sockets for %s", s.cpuSocketsCount, vm.Name))
		} else {
			patchSet.AddOption(patch.WithReplace("/spec/template/spec/domain/cpu/sockets", s.cpuSocketsCount))
			messages = append(messages, fmt.Sprintf("Successfully set CPU sockets to %d for %s", s.cpuSocketsCount, vm.Name))
		}
	}

	if cmd.Flags().Changed("memory") {
		guestMemorySize, err := resource.ParseQuantity(s.memorySize)
		if err != nil {
			return fmt.Errorf("invalid memory size: %s", s.memorySize)
		}

		if vm.Spec.Template.Spec.Domain.Memory == nil {
			// Memory field does not exist, add the entire Memory structure
			memoryToAdd := &v1.Memory{Guest: &guestMemorySize}
			patchSet.AddOption(patch.WithAdd("/spec/template/spec/domain/memory", memoryToAdd))
			messages = append(messages, fmt.Sprintf("Successfully added memory with size %s for %s", guestMemorySize.String(), vm.Name))
		} else {
			patchSet.AddOption(patch.WithReplace("/spec/template/spec/domain/memory/guest", guestMemorySize))
			messages = append(messages, fmt.Sprintf("Successfully set memory to %s for %s", guestMemorySize.String(), vm.Name))
		}
	}

	if !patchSet.IsEmpty() {
		patchPayload, err := patchSet.GeneratePayload()
		if err != nil {
			return fmt.Errorf("failed to generate patch payload: %w", err)
		}

		if _, err := virtCli.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchPayload, k8smetav1.PatchOptions{}); err != nil {
			return fmt.Errorf("could not patch the virtual machine: %w", err)
		}
	} else {
		cmd.Help()
		return fmt.Errorf("at least one of --cpu or --memory must be set")
	}

	for _, message := range messages {
		cmd.Printf("%s\n", message)
	}
	return nil
}
