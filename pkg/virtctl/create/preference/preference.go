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

package preference

import (
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/yaml"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/preference/validation"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/create/params"
)

const (
	CPUTopologyFlag         = "cpu-topology"
	VolumeStorageClassFlag  = "volume-storage-class"
	MachineTypeFlag         = "machine-type"
	NameFlag                = "name"
	NamespacedFlag          = "namespaced"
	defaultNameSuffixLength = 5
)

type createPreference struct {
	namespace             string
	name                  string
	namespaced            bool
	cpuTopology           string
	machineType           string
	preferredStorageClass string
}

func NewCommand() *cobra.Command {
	c := createPreference{}
	cmd := &cobra.Command{
		Use:     "preference",
		Short:   "Create a VirtualMachinePreference or VirtualMachineClusterPreference manifest.",
		Example: c.usage(),
		RunE:    c.run,
	}
	cmd.Flags().BoolVar(&c.namespaced,
		NamespacedFlag,
		c.namespaced,
		"Specify if VirtualMachinePreference should be created. By default VirtualMachineClusterPreference is created.")
	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the Preference.")
	cmd.Flags().StringVar(&c.preferredStorageClass, VolumeStorageClassFlag, c.preferredStorageClass, "Defines the preferred storage class")
	cmd.Flags().StringVar(&c.machineType, MachineTypeFlag, c.machineType, "Defines the preferred machine type to use.")
	cmd.Flags().StringVar(&c.cpuTopology, CPUTopologyFlag, c.cpuTopology, "Defines the preferred guest visible CPU topology.")

	return cmd
}

func (c *createPreference) setDefaults(cmd *cobra.Command) error {
	_, _, namespace, overridden, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
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
		c.name = "preference-" + rand.String(defaultNameSuffixLength)
	} else {
		c.name = "clusterpreference-" + rand.String(defaultNameSuffixLength)
	}

	return nil
}

func (c *createPreference) optFns() map[string]func(*instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	return map[string]func(*instancetypev1beta1.VirtualMachinePreferenceSpec) error{
		VolumeStorageClassFlag: c.withVolumeStorageClass,
		MachineTypeFlag:        c.withMachineType,
		CPUTopologyFlag:        c.withCPUTopology,
	}
}

func (c *createPreference) withVolumeStorageClass(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	preferenceSpec.Volumes = &instancetypev1beta1.VolumePreferences{
		PreferredStorageClassName: c.preferredStorageClass,
	}
	return nil
}

func (c *createPreference) withMachineType(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	preferenceSpec.Machine = &instancetypev1beta1.MachinePreferences{
		PreferredMachineType: c.machineType,
	}
	return nil
}

func (c *createPreference) withCPUTopology(preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec) error {
	preferredCPUTopology := instancetypev1beta1.PreferredCPUTopology(c.cpuTopology)
	if !validation.IsPreferredTopologySupported(preferredCPUTopology) {
		return params.FlagErr(CPUTopologyFlag, "CPU topology must have a value of sockets, cores, threads or spread")
	}
	preferenceSpec.CPU = &instancetypev1beta1.CPUPreferences{
		PreferredCPUTopology: &preferredCPUTopology,
	}
	return nil
}

func (c *createPreference) usage() string {
	return `  # Create a manifest for a ClusterPreference with a random name:
  {{ProgramName}} create preference
	
  # Create a manifest for a ClusterPreference with a specified CPU topology:
  {{ProgramName}} create preference --cpu-topology sockets

  # Create a manifest for a Preference with a specified CPU topology:
  {{ProgramName}} create preference --cpu-topology sockets --namespaced
	
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
	for flag := range c.optFns() {
		if cmd.Flags().Changed(flag) {
			if err := c.optFns()[flag](preferenceSpec); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *createPreference) run(cmd *cobra.Command, _ []string) error {
	if err := c.setDefaults(cmd); err != nil {
		return err
	}

	var out []byte
	var err error
	if c.namespaced {
		preference := c.newPreference()

		err = c.applyFlags(cmd, &preference.Spec)
		if err != nil {
			return err
		}

		out, err = yaml.Marshal(preference)
		if err != nil {
			return err
		}
	} else {
		clusterPreference := c.newClusterPreference()

		err = c.applyFlags(cmd, &clusterPreference.Spec)
		if err != nil {
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
