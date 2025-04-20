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
 */

package configuration

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

func NewListPermittedDevices() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "permitted-devices",
		Short:   "List the permitted devices for vmis.",
		Example: usage(),
		Args:    cobra.ExactArgs(0),
		RunE:    run,
	}

	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd
}

func usage() string {
	return "# Print the permitted devices for VMIs:\n  {{ProgramName}} permitted-devices"
}

func run(cmd *cobra.Command, _ []string) error {
	virtClient, namespace, _, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return fmt.Errorf("cannot obtain KubeVirt client: %v", err)
	}

	kvList, err := virtClient.KubeVirt(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(kvList.Items) == 0 {
		return fmt.Errorf("no KubeVirt resource in %q namespace", namespace)
	}

	kv := kvList.Items[0]

	var (
		hostDeviceList []string
		gpuDeviceList  []string
	)

	if kv.Spec.Configuration.PermittedHostDevices != nil {
		for _, hd := range kv.Spec.Configuration.PermittedHostDevices.PciHostDevices {
			hostDeviceList = append(hostDeviceList, hd.ResourceName)
		}

		for _, hd := range kv.Spec.Configuration.PermittedHostDevices.MediatedDevices {
			gpuDeviceList = append(gpuDeviceList, hd.ResourceName)
		}
	}

	cmd.Printf("Permitted Devices: \nHost Devices: \n%s \nGPU Devices: \n%s\n",
		strings.Join(hostDeviceList, ", "),
		strings.Join(gpuDeviceList, ", "),
	)

	return nil
}
