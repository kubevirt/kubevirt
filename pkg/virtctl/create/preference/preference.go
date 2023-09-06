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

package preference

import (
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/clientcmd"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virtctl/create/params"
)

const (
	Preference = "preference"

	CPUTopologyFlag        = "cpu-topology"
	VolumeStorageClassFlag = "volume-storage-class"
	MachineTypeFlag        = "machine-type"
	NameFlag               = "name"
	NamespacedFlag         = "namespaced"
	CPUTopologyErr         = "CPU topology must have a value of preferCores, preferSockets or preferThreads"

	stringPreferCores   = string(instancetypev1beta1.PreferCores)
	stringPreferSockets = string(instancetypev1beta1.PreferSockets)
	stringPreferThreads = string(instancetypev1beta1.PreferThreads)
)

type createPreference struct {
	namespace             string
	name                  string
	namespaced            bool
	CPUTopology           string
	machineType           string
	preferredStorageClass string

	clientConfig clientcmd.ClientConfig
}

type optionFn func(*createPreference, *instancetypev1beta1.VirtualMachinePreferenceSpec) error

var optFns = map[string]optionFn{
	VolumeStorageClassFlag: withVolumeStorageClass,
	MachineTypeFlag:        withMachineType,
	CPUTopologyFlag:        withCPUTopology,
}

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	c := createPreference{
		clientConfig: clientConfig,
	}
	cmd := &cobra.Command{
		Use:     Preference,
		Short:   "Create a VirtualMachinePreference or VirtualMachineClusterPreference manifest.",
		Example: c.usage(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.run(cmd)
		},
	}
	cmd.Flags().BoolVar(&c.namespaced, NamespacedFlag, c.namespaced, "Specify if VirtualMachinePreference should be created. By default VirtualMachineClusterPreference is created.")
	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the Preference.")
	cmd.Flags().StringVar(&c.preferredStorageClass, VolumeStorageClassFlag, c.preferredStorageClass, "Defines the preferred storage class")
	cmd.Flags().StringVar(&c.machineType, MachineTypeFlag, c.machineType, "Defines the preferred machine type to use.")
	cmd.Flags().StringVar(&c.CPUTopology, CPUTopologyFlag, c.CPUTopology, "Defines the preferred guest visible CPU topology.")

	return cmd
}

func (c *createPreference) setDefaults(cmd *cobra.Command) error {
	namespace, overridden, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	if overridden {
		c.namespace = namespace
		c.namespaced = true
	}

	if cmd.Flags().Changed(NameFlag) {
		return nil
	}

	if c.namespaced {
		c.name = "preference-" + rand.String(5)
	} else {
		c.name = "clusterpreference-" + rand.String(5)
	}

	return nil
}

func withVolumeStorageClass(c *createPreference, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	preferenceSpec.Volumes = &instancetypev1beta1.VolumePreferences{
		PreferredStorageClassName: c.preferredStorageClass,
	}
	return nil
}

func withMachineType(c *createPreference, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	preferenceSpec.Machine = &instancetypev1beta1.MachinePreferences{
		PreferredMachineType: c.machineType,
	}
	return nil
}

func withCPUTopology(c *createPreference, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	if c.CPUTopology != stringPreferCores &&
		c.CPUTopology != stringPreferSockets &&
		c.CPUTopology != stringPreferThreads {
		return params.FlagErr(CPUTopologyFlag, CPUTopologyErr)
	}

	preferredCPUTopology := instancetypev1beta1.PreferredCPUTopology(c.CPUTopology)
	preferenceSpec.CPU = &instancetypev1beta1.CPUPreferences{
		PreferredCPUTopology: &preferredCPUTopology,
	}
	return nil
}

func (c *createPreference) usage() string {
	return `  # Create a manifest for a ClusterPreference with a random name:
  {{ProgramName}} create preference
	
  # Create a manifest for a ClusterPreference with a specified CPU topology:
  {{ProgramName}} create preference --cpu-topology preferSockets

  # Create a manifest for a Preference with a specified CPU topology:
  {{ProgramName}} create preference --cpu-topology preferSockets --namespaced
	
  # Create a manifest for a ClusterPreference and use it to create a resource with kubectl
  {{ProgramName}} create preference --volume-storage-class hostpath-provisioner | kubectl create -f -`
}

func (c *createPreference) newClusterPreference() *instancetypev1beta1.VirtualMachineClusterPreference {
	return &instancetypev1beta1.VirtualMachineClusterPreference{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineClusterPreference",
			APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.name,
		},
	}
}

func (c *createPreference) newPreference() *instancetypev1beta1.VirtualMachinePreference {
	preference := &instancetypev1beta1.VirtualMachinePreference{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachinePreference",
			APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.name,
		},
	}

	if c.namespace != "" {
		preference.Namespace = c.namespace
	}

	return preference
}

func (c *createPreference) applyFlags(cmd *cobra.Command, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	for flag := range optFns {
		if cmd.Flags().Changed(flag) {
			if err := optFns[flag](c, preferenceSpec); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *createPreference) run(cmd *cobra.Command) error {
	var out []byte
	var err error

	c.setDefaults(cmd)

	if c.namespaced {
		preference := c.newPreference()

		if err = c.applyFlags(cmd, &preference.Spec); err != nil {
			return err
		}

		out, err = yaml.Marshal(preference)
		if err != nil {
			return err
		}
	} else {
		clusterPreference := c.newClusterPreference()

		if err = c.applyFlags(cmd, &clusterPreference.Spec); err != nil {
			return err
		}

		out, err = yaml.Marshal(clusterPreference)
		if err != nil {
			return err
		}
	}

	cmd.Print(string(out))

	return nil
}
