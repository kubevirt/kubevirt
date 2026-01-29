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

package create

import (
	"fmt"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	templateapi "kubevirt.io/virt-template-api/core"
	"kubevirt.io/virt-template-api/core/v1alpha1"
	templateclient "kubevirt.io/virt-template-client-go/virttemplate"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	vmNameFlag      = "vm-name"
	vmNamespaceFlag = "vm-namespace"
	nameFlag        = "name"
)

// GetTemplateClient is used in unit tests to allow overriding the client
var GetTemplateClient = func(c *rest.Config) (templateclient.Interface, error) {
	return templateclient.NewForConfig(c)
}

type create struct {
	vmName      string
	vmNamespace string
	name        string

	cmd       *cobra.Command
	namespace string
	client    templateclient.Interface
}

func NewCommand() *cobra.Command {
	c := create{}
	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a VirtualMachineTemplate from an existing VirtualMachine on the cluster.",
		Example: usage(),
		Args:    cobra.NoArgs,
		RunE:    c.run,
	}

	cmd.Flags().StringVar(&c.vmName, vmNameFlag, "",
		"Specify the name of the VirtualMachine to create a template from.")
	cmd.Flags().StringVar(&c.vmNamespace, vmNamespaceFlag, "",
		"Specify the namespace of the VirtualMachine. If not specified, uses the current namespace.")
	cmd.Flags().StringVar(&c.name, nameFlag, "",
		"Specify the name for the created VirtualMachineTemplate. If not specified, uses a randomized name.")

	if err := cmd.MarkFlagRequired(vmNameFlag); err != nil {
		panic(err)
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func usage() string {
	return `  # Create a VirtualMachineTemplateRequest for a VM in the current namespace.
  {{ProgramName}} template create --vm-name my-vm

  # Create a VirtualMachineTemplateRequest for a VM in a specific namespace.
  {{ProgramName}} template create --vm-name my-vm --vm-namespace production

  # Create a VirtualMachineTemplateRequest with a specific template name.
  {{ProgramName}} template create --vm-name my-vm --name my-template

  # Create a VirtualMachineTemplateRequest and watch for completion.
  {{ProgramName}} template create --vm-name my-vm && kubectl get vmtr -w
  `
}

func (c *create) run(cmd *cobra.Command, _ []string) error {
	if err := c.setDefaults(cmd); err != nil {
		return err
	}

	tplReq, err := c.createTemplateRequest()
	if err != nil {
		return err
	}

	c.cmd.Printf("%s.%s/%s created\n", templateapi.SingularRequestResourceName, v1alpha1.GroupVersion.Group, tplReq.Name)

	return nil
}

func (c *create) setDefaults(cmd *cobra.Command) error {
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

	if c.vmNamespace == "" {
		c.vmNamespace = namespace
	}

	return nil
}

func (c *create) createTemplateRequest() (*v1alpha1.VirtualMachineTemplateRequest, error) {
	tplReq := &v1alpha1.VirtualMachineTemplateRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: c.vmName + "-",
			Namespace:    c.namespace,
		},
		Spec: v1alpha1.VirtualMachineTemplateRequestSpec{
			VirtualMachineRef: v1alpha1.VirtualMachineReference{
				Namespace: c.vmNamespace,
				Name:      c.vmName,
			},
			TemplateName: c.name,
		},
	}

	tplReq, err := c.client.TemplateV1alpha1().VirtualMachineTemplateRequests(c.namespace).
		Create(c.cmd.Context(), tplReq, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating VirtualMachineTemplateRequest: %w", err)
	}

	return tplReq, nil
}
