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

package objectgraph

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

type command struct {
	vmi            bool
	shouldExclude  bool
	labelSelectors map[string]string
	outputFormat   string
}

func NewCommand() *cobra.Command {
	c := command{}
	cmd := &cobra.Command{
		Use:     "objectgraph (VM|VMI)",
		Short:   "Returns dependency graph related to a VM|VMI.",
		Example: usageObjectGraph(),
		Args:    cobra.ExactArgs(1),
		RunE:    c.objectGraphRun,
	}

	cmd.Flags().BoolVar(&c.vmi, "vmi", false, "Returns the object graph from the VMI instead of the VM.")
	cmd.Flags().BoolVar(&c.shouldExclude, "exclude-optional", false, "Exclude optional nodes from the object graph.")
	cmd.Flags().StringToStringVar(&c.labelSelectors, "selector", map[string]string{}, "Label selectors to filter the object graph (multiple labels can be specified).")
	cmd.Flags().StringVarP(&c.outputFormat, "output", "o", "json", "Output format. One of: json|yaml")
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

// usageObjectGraph provides several valid usage examples of objectgraph
func usageObjectGraph() string {
	return `
  # Get the object graph for a VirtualMachine named 'my-vm'
  {{ProgramName}} objectgraph my-vm

  # Get the object graph for a VirtualMachineInstance named 'my-vmi'
  {{ProgramName}} objectgraph --vmi my-vmi

  # Exclude optional nodes in the object graph for a VM
  {{ProgramName}} objectgraph my-vm --exclude-optional

  # Filter the object graph by label selector
  {{ProgramName}} objectgraph my-vm --selector="kubevirt.io/dependency-type=storage,kubevirt.io/dependency-type=compute"

  # Get the object graph in YAML format
  {{ProgramName}} objectgraph my-vm --output yaml
`
}

func (c *command) objectGraphRun(cmd *cobra.Command, args []string) error {
	vmName := args[0]

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	include := !c.shouldExclude
	opts := &v1.ObjectGraphOptions{
		IncludeOptionalNodes: &include,
	}

	if len(c.labelSelectors) > 0 {
		opts.LabelSelector = &metav1.LabelSelector{
			MatchLabels: c.labelSelectors,
		}
	}

	var objectGraph v1.ObjectGraphNode
	if c.vmi {
		objectGraph, err = virtClient.VirtualMachineInstance(namespace).ObjectGraph(cmd.Context(), vmName, opts)
		if err != nil {
			return fmt.Errorf("error listing object graph of VirtualMachineInstance %s: %v", vmName, err)
		}
	} else {
		objectGraph, err = virtClient.VirtualMachine(namespace).ObjectGraph(cmd.Context(), vmName, opts)
		if err != nil {
			return fmt.Errorf("error listing object graph of VirtualMachine %s: %v", vmName, err)
		}
	}

	var output []byte
	switch c.outputFormat {
	case "json":
		output, err = json.MarshalIndent(objectGraph, "", "  ")
		if err != nil {
			return fmt.Errorf("cannot marshal object graph to JSON: %v", err)
		}
	case "yaml":
		output, err = yaml.Marshal(objectGraph)
		if err != nil {
			return fmt.Errorf("cannot marshal object graph to YAML: %v", err)
		}
	default:
		return fmt.Errorf("unsupported output format: %s (must be 'json' or 'yaml')", c.outputFormat)
	}

	cmd.Println(string(output))
	return nil
}
