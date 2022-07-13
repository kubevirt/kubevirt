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

package softreboot

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	COMMAND_SOFT_REBOOT = "soft-reboot"
)

func NewSoftRebootCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "soft-reboot (VMI)",
		Short:   "Soft reboot a virtual machine instance",
		Long:    `Soft reboot a virtual machine instance`,
		Args:    templates.ExactArgs(COMMAND_SOFT_REBOOT, 1),
		Example: usage(COMMAND_SOFT_REBOOT),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := SoftReboot{
				clientConfig: clientConfig,
			}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage(cmd string) string {
	usage := fmt.Sprintf("  # %s a virtualmachineinstance called 'myvmi':\n", strings.Title(cmd))
	usage += fmt.Sprintf("  {{ProgramName}} %s myvmi", cmd)
	return usage
}

type SoftReboot struct {
	clientConfig clientcmd.ClientConfig
}

func (o *SoftReboot) Run(args []string) error {
	vmi := args[0]

	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	if err = virtClient.VirtualMachineInstance(namespace).SoftReboot(vmi); err != nil {
		return fmt.Errorf("Error soft rebooting VirtualMachineInstance %s: %v", vmi, err)
	}

	fmt.Printf("VMI %s was scheduled to %s\n", vmi, COMMAND_SOFT_REBOOT)
	return nil
}
