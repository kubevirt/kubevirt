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

package convert

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	ocpv1 "github.com/openshift/api/template/v1"
	ocptemplateclient "github.com/openshift/client-go/template/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	"kubevirt.io/virt-template-api/core/v1alpha1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	nameFlag        = "name"
	fileFlag        = "file"
	fileFlagShort   = "f"
	outputFlag      = "output"
	outputFlagShort = "o"

	formatYAML = "yaml"
	formatJSON = "json"

	// jsonBufferSize determines how far into the stream the decoder will look for JSON
	jsonBufferSize = 1024
)

// GetTemplateClient is used in unit tests to allow overriding the client
var GetTemplateClient = func(c *rest.Config) (ocptemplateclient.Interface, error) {
	return ocptemplateclient.NewForConfig(c)
}

type convert struct {
	name   string
	file   string
	output string

	cmd       *cobra.Command
	namespace string
	client    ocptemplateclient.Interface
}

func NewCommand() *cobra.Command {
	p := convert{}
	cmd := &cobra.Command{
		Use:     "convert [name]",
		Short:   "Convert an OpenShift Template into a VirtualMachineTemplate.",
		Example: usage(),
		Args:    cobra.RangeArgs(0, 1),
		RunE:    p.run,
	}

	cmd.Flags().StringVar(&p.name, nameFlag, "",
		"Specify the name of the OpenShift Template.")
	cmd.Flags().StringVarP(&p.file, fileFlag, fileFlagShort, "",
		"Specify a file containing an OpenShift Template.")
	cmd.Flags().StringVarP(&p.output, outputFlag, outputFlagShort, formatYAML,
		"Specify the output format.")

	cmd.MarkFlagsMutuallyExclusive(fileFlag, nameFlag)
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	return `  # Convert an OpenShift Template from the cluster using positional argument.
  {{ProgramName}} template convert mytemplate

  # Convert a template by name from the cluster.
  {{ProgramName}} template convert --name mytemplate

  # Convert a template from a local file.
  {{ProgramName}} template convert --file mytemplate.yaml

  # Convert a template from stdin.
  {{ProgramName}} template convert --file -

  # Convert a template in a specific namespace.
  {{ProgramName}} template convert --name mytemplate --namespace production

  # Convert a template and output as JSON.
  {{ProgramName}} template convert --name mytemplate --output json

  # Convert a local template file.
  {{ProgramName}} template convert --file mytemplate.yaml

  # Convert template from current directory and redirect output to file.
  {{ProgramName}} template convert --file mytemplate.yaml > vmt.yaml

  # Convert template and save as JSON.
  {{ProgramName}} template convert mytemplate -o json > vmt.json

  # Convert a template from different namespace.
  {{ProgramName}} template convert mytemplate -n dev

  # Convert template and directly apply it to the cluster.
  {{ProgramName}} template convert mytemplate -n staging | kubectl apply -f -
  `
}

func (c *convert) run(cmd *cobra.Command, args []string) error {
	if err := c.validateArgs(args); err != nil {
		return err
	}

	if err := c.setDefaults(cmd); err != nil {
		return err
	}

	tpl, err := c.convertTemplate()
	if err != nil {
		return err
	}

	if err := c.print(tpl); err != nil {
		return err
	}

	return nil
}

func (c *convert) validateArgs(args []string) error {
	if len(args) > 0 {
		if c.file != "" {
			return fmt.Errorf("only one of name or file can be provided")
		}
		if c.name != "" {
			return fmt.Errorf("provide name either from positional argument or from flag")
		}
		c.name = args[0]
	}

	if c.file == "" && c.name == "" {
		return fmt.Errorf("name or file must be provided")
	}

	if c.output != formatYAML && c.output != formatJSON {
		return fmt.Errorf("not supported output format: %s", c.output)
	}

	return nil
}

func (c *convert) setDefaults(cmd *cobra.Command) error {
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	client, err := GetTemplateClient(virtClient.Config())
	if err != nil {
		return err
	}

	c.cmd = cmd
	c.namespace = namespace
	c.client = client

	return nil
}

func (c *convert) convertTemplate() (*v1alpha1.VirtualMachineTemplate, error) {
	ocpTpl, err := c.getTemplate()
	if err != nil {
		return nil, err
	}

	tpl := &v1alpha1.VirtualMachineTemplate{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "VirtualMachineTemplate",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ocpTpl.Name,
			Namespace:   ocpTpl.Namespace,
			Labels:      ocpTpl.Labels,
			Annotations: ocpTpl.Annotations,
		},
		Spec: v1alpha1.VirtualMachineTemplateSpec{
			Message:    ocpTpl.Message,
			Parameters: convertParameters(ocpTpl.Parameters),
		},
	}

	if len(ocpTpl.Objects) != 1 {
		return nil, fmt.Errorf("template must contain exactly one object, found %d", len(ocpTpl.Objects))
	}
	tpl.Spec.VirtualMachine = &ocpTpl.Objects[0]

	return tpl, nil
}

func convertParameters(ocpParams []ocpv1.Parameter) []v1alpha1.Parameter {
	if len(ocpParams) == 0 {
		return nil
	}

	params := make([]v1alpha1.Parameter, len(ocpParams))
	for i, p := range ocpParams {
		params[i] = v1alpha1.Parameter{
			Name:        p.Name,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Value:       p.Value,
			Generate:    p.Generate,
			From:        p.From,
			Required:    p.Required,
		}
	}
	return params
}

func (c *convert) getTemplate() (*ocpv1.Template, error) {
	var (
		ocpTpl *ocpv1.Template
		err    error
	)
	if c.name != "" {
		ocpTpl, err = c.fetchRemote()
	} else {
		ocpTpl, err = c.readFromFileOrStdin()
	}
	return ocpTpl, err
}

func (c *convert) fetchRemote() (*ocpv1.Template, error) {
	ocpTpl, err := c.client.TemplateV1().Templates(c.namespace).Get(c.cmd.Context(), c.name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error fetching remote Template %s/%s: %w", c.namespace, c.name, err)
	}
	return ocpTpl, nil
}

func (c *convert) readFromFileOrStdin() (*ocpv1.Template, error) {
	var data io.Reader
	if c.file == "-" {
		data = os.Stdin
	} else {
		f, err := os.Open(c.file)
		if err != nil {
			return nil, fmt.Errorf("error opening file %s: %w", c.file, err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				c.cmd.PrintErrf("error closing file %s: %v", c.file, err)
			}
		}()
		data = f
	}

	ocpTpl := &ocpv1.Template{}
	if err := k8syaml.NewYAMLOrJSONDecoder(data, jsonBufferSize).Decode(ocpTpl); err != nil {
		return nil, fmt.Errorf("error decoding Template: %w", err)
	}

	return ocpTpl, nil
}

func (c *convert) print(obj any) error {
	var (
		data []byte
		err  error
	)
	switch c.output {
	case formatYAML:
		data, err = yaml.Marshal(obj)
	case formatJSON:
		data, err = json.MarshalIndent(obj, "", " ")
	}
	if err != nil {
		return err
	}

	c.cmd.Print(string(data))

	return nil
}
