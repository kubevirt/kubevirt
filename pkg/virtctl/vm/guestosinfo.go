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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const COMMAND_GUESTOSINFO = "guestosinfo"

func NewGuestOsInfoCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "guestosinfo (VMI)",
		Short:   "Return guest agent info about operating system.",
		Example: usage(COMMAND_GUESTOSINFO),
		Args:    templates.ExactArgs("guestosinfo", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{clientConfig: clientConfig}
			return c.guestOsInfoRun(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func (o *Command) guestOsInfoRun(args []string) error {
	vmiName := args[0]

	virtClient, namespace, err := GetNamespaceAndClient(o.clientConfig)
	if err != nil {
		return err
	}

	guestosinfo, err := virtClient.VirtualMachineInstance(namespace).GuestOsInfo(context.Background(), vmiName)
	if err != nil {
		return fmt.Errorf("Error getting guestosinfo of VirtualMachineInstance %s, %v", vmiName, err)
	}

	data, err := json.MarshalIndent(guestosinfo, "", "  ")
	if err != nil {
		return fmt.Errorf("Cannot marshal guestosinfo %v", err)
	}

	fmt.Printf("%s\n", string(data))
	return nil
}
