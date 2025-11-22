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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	yml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
)

const (
	COMMAND_EXPAND       = "expand"
	vmArg                = "vm"
	filePathArg          = "file"
	filePathArgShort     = "f"
	outputFormatArg      = "output"
	outputFormatArgShort = "o"
)

var (
	vmName       string
	filePath     string
	outputFormat string
)

func NewExpandCommand() *cobra.Command {
	c := Command{}
	cmd := &cobra.Command{
		Use:     "expand (VM)",
		Short:   "Return the VirtualMachine object with expanded instancetype and preference.",
		Example: usageExpand(),
		Args:    cobra.MatchAll(cobra.ExactArgs(0), expandArgs()),
		RunE:    c.expandRun,
	}
	cmd.Flags().StringVar(&vmName, vmArg, "", "Specify VirtualMachine name that should be expanded. Mutually exclusive with \"--file\" flag.")
	cmd.Flags().StringVarP(&filePath, filePathArg, filePathArgShort, "", "If present, the Virtual Machine spec in provided file will be expanded. Mutually exclusive with \"--vm\" flag.")
	cmd.Flags().StringVarP(&outputFormat, outputFormatArg, outputFormatArgShort, YAML, "Specify a format that will be used to display output.")
	cmd.MarkFlagsMutuallyExclusive(filePathArg, vmArg)
	return cmd
}

func (o *Command) expandRun(cmd *cobra.Command, args []string) error {
	var expandedVm *v1.VirtualMachine
	var err error

	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}

	if vmName != "" {
		expandedVm, err = virtClient.VirtualMachine(namespace).GetWithExpandedSpec(context.Background(), vmName)
		if err != nil {
			return fmt.Errorf("error expanding VirtualMachine - %s in namespace - %s: %w", vmName, namespace, err)
		}
	} else {
		vm, err := readVMFromFileOrStdin(filePath)
		if err != nil {
			return err
		}

		expandedVm, err = virtClient.ExpandSpec(namespace).ForVirtualMachine(vm)
		if err != nil {
			return fmt.Errorf("error expanding VirtualMachine - %s in namespace - %s: %w", vm.Name, namespace, err)
		}
	}

	output, err := applyOutputFormat(outputFormat, expandedVm)
	if err != nil {
		return err
	}

	cmd.Print(output)
	return nil
}

func usageExpand() string {
	return `  #Expand a virtual machine called 'myvm'.
  {{ProgramName}} expand --vm myvm
  
  # Expand a virtual machine from file called myvm.yaml.
  {{ProgramName}} expand --file myvm.yaml

  # Expand a virtual machine from stdin. Press Ctrl+D (Unix) or Ctrl+Z (Windows) to signal the end of input.
  {{ProgramName}} expand --file -

  # Expand a virtual machine called myvm and display output in json format.
  {{ProgramName}} expand --vm myvm --output json
  `
}

func expandArgs() cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if filePath == "" && vmName == "" {
			return fmt.Errorf("error invalid arguments - VirtualMachine name or file must be provided")
		}

		if outputFormat != YAML && outputFormat != JSON {
			return fmt.Errorf("error not supported output format defined: %s", outputFormat)
		}

		return nil
	}
}

func readVMFromFileOrStdin(payload string) (*v1.VirtualMachine, error) {
	vm := &v1.VirtualMachine{}

	var data []byte
	var err error

	if payload == "-" {
		if data, err = io.ReadAll(os.Stdin); err != nil {
			return nil, fmt.Errorf("error reading from stdin: %+w", err)
		}
	} else {
		if data, err = os.ReadFile(payload); err != nil {
			return nil, fmt.Errorf("error reading file %+w", err)
		}
	}

	if err = yml.NewYAMLOrJSONDecoder(bytes.NewReader(data), 1024).Decode(&vm); err != nil {
		return nil, fmt.Errorf("error decoding VirtualMachine %+w", err)
	}

	return vm, nil
}

func applyOutputFormat(outputFormat string, expandedVm *v1.VirtualMachine) (string, error) {
	var formatedOutput []byte
	var err error

	switch outputFormat {
	case JSON:
		formatedOutput, err = json.MarshalIndent(expandedVm, "", " ")
	case YAML:
		formatedOutput, err = yaml.Marshal(expandedVm)
	}

	if err != nil {
		return "", err
	}

	return string(formatedOutput), nil
}
