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

package spice

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "spice (VMI)",
		Short:   "Open a spice connection to a virtual machine instance.",
		Example: usage(),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := Spice{clientConfig: clientConfig}
			return c.Run(cmd, args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

type Spice struct {
	clientConfig clientcmd.ClientConfig
}

func (o *Spice) Run(cmd *cobra.Command, args []string) error {
	namespace, _, err := o.clientConfig.Namespace()
	if err != nil {
		return err
	}

	vmi := args[0]

	virtCli, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return err
	}

	// get connection token for VM
	spiceOption, err := virtCli.VirtualMachineInstance(namespace).Spice(vmi)
	if err != nil {
		return fmt.Errorf("Can't access VMI %s: %s", vmi, err.Error())
	}

	dir, err := os.Getwd()
	spiceConfig := fmt.Sprintf(`[virt-viewer]
type=spice
host=%s
port=%d
password=%s
`, spiceOption.Host, spiceOption.Port, spiceOption.Token)

	return ioutil.WriteFile(path.Join(dir, fmt.Sprintf("%s.%s", vmi, "vv")), []byte(spiceConfig), 0644)
}

func usage() string {
	usage := "  # Connect to 'testvmi' via remote-viewer:\n"
	usage += "  virtctl spice testvmi"
	return usage
}
