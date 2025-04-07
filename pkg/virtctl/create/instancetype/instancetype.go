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

package instancetype

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/yaml"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/create/params"
)

const (
	CPUFlag             = "cpu"
	MemoryFlag          = "memory"
	GPUFlag             = "gpu"
	HostDeviceFlag      = "hostdevice"
	IOThreadsPolicyFlag = "iothreadspolicy"
	NameFlag            = "name"
	NamespacedFlag      = "namespaced"

	nameErr       = "name must be specified"
	deviceNameErr = "deviceName must be specified"
)

type createInstancetype struct {
	namespace       string
	name            string
	cpu             uint32
	memory          string
	gpus            []string
	hostDevices     []string
	ioThreadsPolicy string
	namespaced      bool
}

type gpu struct {
	Name       string `param:"name"`
	DeviceName string `param:"devicename"`
}

type hostDevice struct {
	Name       string `param:"name"`
	DeviceName string `param:"devicename"`
}

func NewCommand() *cobra.Command {
	c := createInstancetype{}
	cmd := &cobra.Command{
		Use:     "instancetype",
		Short:   "Create VirtualMachineInstancetype or VirtualMachineClusterInstancetype manifest.",
		Example: c.usage(),
		RunE:    c.run,
	}
	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the Instancetype.")
	cmd.Flags().Uint32Var(&c.cpu, CPUFlag, c.cpu, "Specify the count of CPUs of the Instancetype.")
	cmd.Flags().StringVar(&c.memory, MemoryFlag, c.memory, "Specify the amount of memory of the Instancetype.")
	cmd.Flags().StringVar(&c.ioThreadsPolicy, IOThreadsPolicyFlag, c.ioThreadsPolicy, "Specify IOThreadsPolicy to be used. Only valid values are \"auto\" and \"shared\".")
	cmd.Flags().BoolVar(&c.namespaced, NamespacedFlag, false, "Specify if VirtualMachineInstancetype should be created. By default VirtualMachineClusterInstancetype is created.")
	cmd.Flags().StringArrayVar(&c.gpus, GPUFlag, c.gpus, "Specify the list of vGPUs to passthrough. Can be provided multiple times.")
	cmd.Flags().StringArrayVar(&c.hostDevices, HostDeviceFlag, c.hostDevices, "Specify list of HostDevices to passthrough. Can be provided multiple times.")

	if err := cmd.MarkFlagRequired(CPUFlag); err != nil {
		panic(err)
	}
	if err := cmd.MarkFlagRequired(MemoryFlag); err != nil {
		panic(err)
	}

	return cmd
}

func (c *createInstancetype) setDefaults(cmd *cobra.Command) error {
	_, namespace, overridden, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
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
		c.name = "instancetype-" + rand.String(5)
	} else {
		c.name = "clusterinstancetype-" + rand.String(5)
	}

	return nil
}

func (c *createInstancetype) optFns() map[string]func(*instancetypev1beta1.VirtualMachineInstancetypeSpec) error {
	return map[string]func(*instancetypev1beta1.VirtualMachineInstancetypeSpec) error{
		GPUFlag:             c.withGPUs,
		HostDeviceFlag:      c.withHostDevices,
		IOThreadsPolicyFlag: c.withIOThreadsPolicy,
	}
}

func (c *createInstancetype) withGPUs(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec) error {
	for _, param := range c.gpus {
		obj := gpu{}
		if err := params.Map(GPUFlag, param, &obj); err != nil {
			return err
		}

		if obj.Name == "" {
			return params.FlagErr(GPUFlag, nameErr)
		}
		if obj.DeviceName == "" {
			return params.FlagErr(GPUFlag, deviceNameErr)
		}

		instancetypeSpec.GPUs = append(instancetypeSpec.GPUs, v1.GPU{Name: obj.Name, DeviceName: obj.DeviceName})
	}

	return nil
}

func (c *createInstancetype) withHostDevices(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec) error {
	for _, param := range c.hostDevices {
		obj := hostDevice{}
		if err := params.Map(HostDeviceFlag, param, &obj); err != nil {
			return err
		}

		if obj.Name == "" {
			return params.FlagErr(HostDeviceFlag, nameErr)
		}
		if obj.DeviceName == "" {
			return params.FlagErr(HostDeviceFlag, deviceNameErr)
		}

		instancetypeSpec.HostDevices = append(instancetypeSpec.HostDevices, v1.HostDevice{Name: obj.Name, DeviceName: obj.DeviceName})
	}

	return nil
}

func (c *createInstancetype) withIOThreadsPolicy(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec) error {
	var policy v1.IOThreadsPolicy
	switch c.ioThreadsPolicy {
	case string(v1.IOThreadsPolicyAuto):
		policy = v1.IOThreadsPolicyAuto
	case string(v1.IOThreadsPolicyShared):
		policy = v1.IOThreadsPolicyShared
	default:
		return params.FlagErr(IOThreadsPolicyFlag, "IOThread must be of value auto or shared")
	}
	instancetypeSpec.IOThreadsPolicy = &policy

	return nil
}

func (c *createInstancetype) usage() string {
	return `  # Create a manifest for a ClusterInstancetype with a random name:
  {{ProgramName}} create instancetype --cpu 2 --memory 256Mi
  
  # Create a manifest for a ClusterInstancetype with a specified name:
  {{ProgramName}} create instancetype --name my-instancetype --cpu 2 --memory 256Mi

  # Create a manifest for a ClusterInstancetype with a specified gpu:
  {{ProgramName}} create instancetype --cpu 2 --memory 256Mi --gpu name:gpu1,devicename:nvidia
  
  # Create a manifest for a Instancetype with a specified name and cpu:
  {{ProgramName}} create instancetype --namespaced --name my-instancetype --cpu 2 --memory 256Mi
  
  # Create a manifest for a ClusterInstancetype and use it to create a resource with kubectl
  {{ProgramName}} create instancetype --cpu 2 --memory 256Mi | kubectl create -f -`
}

func (c *createInstancetype) newInstancetype() *instancetypev1beta1.VirtualMachineInstancetype {
	instancetype := &instancetypev1beta1.VirtualMachineInstancetype{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineInstancetype",
			APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.name,
		},
		Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: c.cpu,
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest: resource.MustParse(c.memory),
			},
		},
	}

	if c.namespace != "" {
		instancetype.Namespace = c.namespace
	}

	return instancetype
}

func (c *createInstancetype) newClusterInstancetype() *instancetypev1beta1.VirtualMachineClusterInstancetype {
	return &instancetypev1beta1.VirtualMachineClusterInstancetype{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachineClusterInstancetype",
			APIVersion: instancetypev1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.name,
		},
		Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
			CPU: instancetypev1beta1.CPUInstancetype{
				Guest: c.cpu,
			},
			Memory: instancetypev1beta1.MemoryInstancetype{
				Guest: resource.MustParse(c.memory),
			},
		},
	}
}

func (c *createInstancetype) applyFlags(cmd *cobra.Command, instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec) error {
	for flag := range c.optFns() {
		if cmd.Flags().Changed(flag) {
			if err := c.optFns()[flag](instancetypeSpec); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *createInstancetype) validateFlags() error {
	if _, err := resource.ParseQuantity(c.memory); err != nil {
		return err
	}

	if c.cpu <= 0 {
		return fmt.Errorf("cpu value must be greater than zero")
	}

	return nil
}

func (c *createInstancetype) run(cmd *cobra.Command, _ []string) error {
	if err := c.setDefaults(cmd); err != nil {
		return err
	}

	if err := c.validateFlags(); err != nil {
		return err
	}

	var out []byte
	var err error
	if c.namespaced {
		instancetype := c.newInstancetype()

		if err = c.applyFlags(cmd, &instancetype.Spec); err != nil {
			return err
		}

		out, err = yaml.Marshal(instancetype)
		if err != nil {
			return err
		}
	} else {
		clusterInstancetype := c.newClusterInstancetype()

		if err = c.applyFlags(cmd, &clusterInstancetype.Spec); err != nil {
			return err
		}

		out, err = yaml.Marshal(clusterInstancetype)
		if err != nil {
			return err
		}
	}

	cmd.Print(string(out))

	return nil
}
