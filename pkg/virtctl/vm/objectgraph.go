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
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const COMMAND_OBJECT_GRAPH = "objectgraph"

var vmi bool

func NewObjectGraphCommand() *cobra.Command {
	c := Command{}
	cmd := &cobra.Command{
		Use:     "objectgraph (VM)",
		Short:   "Returns dependency graph related to a VM/VMI.",
		Example: usage(COMMAND_OBJECT_GRAPH),
		Args:    cobra.ExactArgs(1),
		RunE:    c.objectGraphRun,
	}
	cmd.Flags().BoolVar(&vmi, "vmi", false, "returns the object graph from the VMI instead of the VM.")
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func (o *Command) objectGraphRun(cmd *cobra.Command, args []string) error {
	vmName := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	objectGraph := v1.ObjectGraphNodeList{}
	if !vmi {
		objectGraph, err = virtClient.VirtualMachine(namespace).ObjectGraph(context.Background(), vmName)
		if err != nil {
			return fmt.Errorf("Error listing object graph of VirtualMachine %s, %v\n", vmName, err)
		}
	} else {
		objectGraph, err = virtClient.VirtualMachineInstance(namespace).ObjectGraph(context.Background(), vmName)
		if err != nil {
			return fmt.Errorf("Error listing object graph of VirtualMachineInstance %s, %v", vmName, err)
		}
	}

	data, err := json.MarshalIndent(objectGraph, "", "  ")
	if err != nil {
		return fmt.Errorf("Cannot marshal object graph %v", err)
	}

	fmt.Printf("%s\n", string(data))
	return nil
}
