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

const COMMAND_FSLIST = "fslist"

func NewFSListCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "fslist (VMI)",
		Short:   "Return full list of filesystems available on the guest machine.",
		Example: usage(COMMAND_FSLIST),
		Args:    templates.ExactArgs("fslist", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Command{clientConfig: clientConfig}
			return c.fsListRun(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func (o *Command) fsListRun(args []string) error {
	vmiName := args[0]

	virtClient, namespace, err := GetNamespaceAndClient(o.clientConfig)
	if err != nil {
		return err
	}

	fslist, err := virtClient.VirtualMachineInstance(namespace).FilesystemList(context.Background(), vmiName)
	if err != nil {
		return fmt.Errorf("Error listing filesystems of VirtualMachineInstance %s, %v", vmiName, err)
	}

	data, err := json.MarshalIndent(fslist, "", "  ")
	if err != nil {
		return fmt.Errorf("Cannot marshal filesystem list %v", err)
	}

	fmt.Printf("%s\n", string(data))
	return nil
}
