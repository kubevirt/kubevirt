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

package process

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/virt-template-api/core/subresourcesv1alpha1"
	"kubevirt.io/virt-template-api/core/v1alpha1"
	templateclient "kubevirt.io/virt-template-client-go/virttemplate"
	"kubevirt.io/virt-template-engine/template"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	nameFlag        = "name"
	fileFlag        = "file"
	fileFlagShort   = "f"
	outputFlag      = "output"
	outputFlagShort = "o"
	paramFlag       = "param"
	paramFlagShort  = "p"
	paramsFileFlag  = "params-file"
	localFlag       = "local"
	createFlag      = "create"
	printParamsFlag = "print-params"

	formatYAML = "yaml"
	formatJSON = "json"

	// jsonBufferSize determines how far into the stream the decoder will look for JSON
	jsonBufferSize = 1024
)

// GetTemplateClient is used in unit tests to allow overriding the client
var GetTemplateClient = func(c *rest.Config) (templateclient.Interface, error) {
	return templateclient.NewForConfig(c)
}

type process struct {
	name        string
	file        string
	output      string
	params      map[string]string
	paramsFile  string
	local       bool
	create      bool
	printParams bool

	cmd       *cobra.Command
	namespace string
	client    templateclient.Interface
}

func NewCommand() *cobra.Command {
	p := process{}
	cmd := &cobra.Command{
		Use:     "process [name]",
		Short:   "Process a VirtualMachineTemplate and return the created VirtualMachine object.",
		Example: usage(),
		Args:    cobra.RangeArgs(0, 1),
		RunE:    p.run,
	}

	cmd.Flags().StringVar(&p.name, nameFlag, "",
		"Specify the name of the VirtualMachineTemplate.")
	cmd.Flags().StringVarP(&p.file, fileFlag, fileFlagShort, "",
		"Specify a file containing a VirtualMachineTemplate.")
	cmd.Flags().StringVarP(&p.output, outputFlag, outputFlagShort, formatYAML,
		"Specify the output format.")
	cmd.Flags().StringToStringVarP(&p.params, paramFlag, paramFlagShort, nil,
		"Specify parameters that should be used during processing.")
	cmd.Flags().StringVar(&p.paramsFile, paramsFileFlag, "",
		"Specify a file that contains parameters that should be used during processing. Supports JSON and YAML dictionaries.")
	cmd.Flags().BoolVar(&p.local, localFlag, false,
		"Force local processing of VirtualMachineTemplates.")
	cmd.Flags().BoolVar(&p.create, createFlag, false,
		"Use subresource API to create a VM when processing a remote VirtualMachineTemplate.")
	cmd.Flags().BoolVar(&p.printParams, printParamsFlag, false,
		"Only print parameters of the specified VirtualMachineTemplates.")

	cmd.MarkFlagsMutuallyExclusive(fileFlag, nameFlag)
	cmd.MarkFlagsMutuallyExclusive(fileFlag, localFlag)
	cmd.MarkFlagsMutuallyExclusive(paramFlag, paramsFileFlag)
	cmd.MarkFlagsMutuallyExclusive(createFlag, fileFlag)
	cmd.MarkFlagsMutuallyExclusive(createFlag, localFlag)
	cmd.MarkFlagsMutuallyExclusive(createFlag, printParamsFlag)
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	return `  # Process a template from the cluster using positional argument.
  {{ProgramName}} template process mytemplate

  # Process a template by name from the cluster.
  {{ProgramName}} template process --name mytemplate

  # Process a template from a local file.
  {{ProgramName}} template process --file mytemplate.yaml

  # Process a template from stdin.
  {{ProgramName}} template process --file -

  # Process a template in a specific namespace.
  {{ProgramName}} template process --name mytemplate --namespace production

  # Process a template with parameters.
  {{ProgramName}} template process --name mytemplate --param key1=value1 --param key2=value2

  # Process a template with parameters from a file.
  {{ProgramName}} template process --name mytemplate --params-file params.yaml

  # Process a template and output as JSON.
  {{ProgramName}} template process --name mytemplate --output json

  # Process a local template file with parameters.
  {{ProgramName}} template process --file mytemplate.yaml --param diskSize=50Gi --param memory=4Gi

  # Force local processing (instead of server-side processing).
  {{ProgramName}} template process --name mytemplate --local

  # Create a VM in the cluster from a template using the subresource API.
  {{ProgramName}} template process --name mytemplate --create

  # Create a VM in the cluster with parameters.
  {{ProgramName}} template process --name mytemplate --create --param CPU=4 --param MEMORY=8Gi

  # Print only the parameters of a template without processing.
  {{ProgramName}} template process --name mytemplate --print-params

  # Print parameters from a local template file.
  {{ProgramName}} template process --file mytemplate.yaml --print-params

  # Combine local file processing with parameters file and JSON output.
  {{ProgramName}} template process -f mytemplate.yaml --params-file params.json -o json

  # Process template from current directory and redirect output to file.
  {{ProgramName}} template process --file mytemplate.yaml > vm.yaml

  # Process template with multiple parameters and save as JSON.
  {{ProgramName}} template process mytemplate -p cpu=4 -p memory=8Gi -p diskSize=100Gi -o json > vm.json

  # Process a template from different namespace with parameters.
  {{ProgramName}} template process mytemplate -n dev -p diskSize=20Gi

  # Process template and directly apply it to the cluster.
  {{ProgramName}} template process mytemplate -n staging | kubectl apply -f -
  `
}

func (p *process) run(cmd *cobra.Command, args []string) error {
	if err := p.validateArgs(args); err != nil {
		return err
	}

	if err := p.setDefaults(cmd); err != nil {
		return err
	}

	if p.printParams {
		if err := p.printTemplateParams(); err != nil {
			return fmt.Errorf("error printing template parameters: %w", err)
		}
		return nil
	}

	if p.paramsFile != "" {
		if err := p.readParamFile(); err != nil {
			return fmt.Errorf("error reading params file: %w", err)
		}
	}

	vm, msg, err := p.processTemplate()
	if err != nil {
		return err
	}

	if p.create {
		if kg, err := getKindGroup(&vm.TypeMeta); err != nil {
			return err
		} else {
			p.cmd.Printf("%s/%s created\n", kg, vm.Name)
		}
	} else {
		if err := p.print(vm, msg); err != nil {
			return err
		}
	}

	return nil
}

func getKindGroup(tm *metav1.TypeMeta) (string, error) {
	gv, err := schema.ParseGroupVersion(tm.APIVersion)
	if err != nil {
		return "", fmt.Errorf("error parsing GroupVersion: %w", err)
	}
	return strings.ToLower(tm.Kind + "." + gv.Group), nil
}

func (p *process) validateArgs(args []string) error {
	if len(args) > 0 {
		if p.file != "" {
			return fmt.Errorf("only one of name or file can be provided")
		}
		if p.name != "" {
			return fmt.Errorf("provide name either from positional argument or from flag")
		}
		p.name = args[0]
	}

	if p.file == "" && p.name == "" {
		return fmt.Errorf("name or file must be provided")
	}

	if p.output != formatYAML && p.output != formatJSON {
		return fmt.Errorf("not supported output format: %s", p.output)
	}

	return nil
}

func (p *process) setDefaults(cmd *cobra.Command) error {
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	client, err := GetTemplateClient(virtClient.Config())
	if err != nil {
		return err
	}

	p.cmd = cmd
	p.namespace = namespace
	p.client = client

	return nil
}

func (p *process) printTemplateParams() error {
	tpl, err := p.getTemplate()
	if err != nil {
		return err
	}
	return p.print(tpl.Spec.Parameters, "")
}

func (p *process) getTemplate() (*v1alpha1.VirtualMachineTemplate, error) {
	var (
		tpl *v1alpha1.VirtualMachineTemplate
		err error
	)
	if p.name != "" {
		tpl, err = p.fetchRemote()
	} else {
		tpl, err = p.readFromFileOrStdin()
	}
	return tpl, err
}

func (p *process) fetchRemote() (*v1alpha1.VirtualMachineTemplate, error) {
	tpl, err := p.client.TemplateV1alpha1().VirtualMachineTemplates(p.namespace).Get(p.cmd.Context(), p.name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error fetching remote VirtualMachineTemplate %s/%s: %w", p.namespace, p.name, err)
	}
	return tpl, nil
}

func (p *process) readFromFileOrStdin() (*v1alpha1.VirtualMachineTemplate, error) {
	var data io.Reader
	if p.file == "-" {
		data = os.Stdin
	} else {
		f, err := os.Open(p.file)
		if err != nil {
			return nil, fmt.Errorf("error opening file %s: %w", p.file, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				p.cmd.PrintErrf("error closing file %s: %v", p.file, err)
			}
		}()
		data = f
	}

	tpl := &v1alpha1.VirtualMachineTemplate{}
	if err := k8syaml.NewYAMLOrJSONDecoder(data, jsonBufferSize).Decode(tpl); err != nil {
		return nil, fmt.Errorf("error decoding VirtualMachineTemplate: %w", err)
	}

	return tpl, nil
}

func (p *process) print(obj any, msg string) error {
	var (
		data []byte
		err  error
	)
	switch p.output {
	case formatYAML:
		data, err = yaml.Marshal(obj)
	case formatJSON:
		data, err = json.MarshalIndent(obj, "", " ")
	}
	if err != nil {
		return err
	}

	if msg != "" {
		p.cmd.PrintErr(msg)
	}
	p.cmd.Print(string(data))

	return nil
}

func (p *process) readParamFile() error {
	f, err := os.Open(p.paramsFile)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			p.cmd.PrintErrf("error closing file %s: %v", p.paramsFile, err)
		}
	}()
	return k8syaml.NewYAMLOrJSONDecoder(f, jsonBufferSize).Decode(&p.params)
}

func (p *process) processTemplate() (*virtv1.VirtualMachine, string, error) {
	if p.create {
		return p.createRemote()
	}
	if !p.local && p.name != "" {
		return p.processRemote()
	} else {
		return p.processLocal()
	}
}

func (p *process) createRemote() (*virtv1.VirtualMachine, string, error) {
	opts := subresourcesv1alpha1.ProcessOptions{
		Parameters: p.params,
	}

	created, err := p.client.TemplateV1alpha1().VirtualMachineTemplates(p.namespace).CreateVirtualMachine(p.cmd.Context(), p.name, opts)
	if err != nil {
		return nil, "", fmt.Errorf("error creating VirtualMachine from remote VirtualMachineTemplate %s/%s: %w", p.namespace, p.name, err)
	}

	return created.VirtualMachine, created.Message, nil
}

func (p *process) processRemote() (*virtv1.VirtualMachine, string, error) {
	opts := subresourcesv1alpha1.ProcessOptions{
		Parameters: p.params,
	}

	processed, err := p.client.TemplateV1alpha1().VirtualMachineTemplates(p.namespace).Process(p.cmd.Context(), p.name, opts)
	if err != nil {
		return nil, "", fmt.Errorf("error processing remote VirtualMachineTemplate %s/%s: %w", p.namespace, p.name, err)
	}

	return processed.VirtualMachine, processed.Message, nil
}

func (p *process) processLocal() (*virtv1.VirtualMachine, string, error) {
	tpl, err := p.getTemplate()
	if err != nil {
		return nil, "", err
	}

	tpl.Spec.Parameters, err = template.MergeParameters(tpl.Spec.Parameters, p.params)
	if err != nil {
		return nil, "", fmt.Errorf("error merging template parameters: %w", err)
	}

	warnings, errs := template.ValidateParameterReferences(tpl)
	for _, w := range warnings {
		p.cmd.PrintErrf("Warning: %s\n", w)
	}
	if len(errs) > 0 {
		return nil, "", fmt.Errorf("error validating template parameters: %w", errs.ToAggregate())
	}

	vm, msg, pErr := template.GetDefaultProcessor().Process(tpl)
	if pErr != nil {
		return nil, "", fmt.Errorf("error processing VirtualMachineTemplate locally: %w", pErr)
	}

	return vm, msg, nil
}
