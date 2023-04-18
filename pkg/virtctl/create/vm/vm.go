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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package vm

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virtctl/create/params"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	VM = "vm"

	NameFlag                   = "name"
	RunStrategyFlag            = "run-strategy"
	TerminationGracePeriodFlag = "termination-grace-period"
	InstancetypeFlag           = "instancetype"
	PreferenceFlag             = "preference"
	ContainerdiskVolumeFlag    = "volume-containerdisk"
	DataSourceVolumeFlag       = "volume-datasource"
	ClonePvcVolumeFlag         = "volume-clone-pvc"
	PvcVolumeFlag              = "volume-pvc"
	BlankVolumeFlag            = "volume-blank"
	CloudInitUserDataFlag      = "cloud-init-user-data"
	CloudInitNetworkDataFlag   = "cloud-init-network-data"
	InferInstancetypeFlag      = "infer-instancetype"
	InferPreferenceFlag        = "infer-preference"

	cloudInitDisk = "cloudinitdisk"
)

type createVM struct {
	name                   string
	terminationGracePeriod int64
	runStrategy            string
	instancetype           string
	preference             string
	containerdiskVolumes   []string
	dataSourceVolumes      []string
	clonePvcVolumes        []string
	blankVolumes           []string
	pvcVolumes             []string
	cloudInitUserData      string
	cloudInitNetworkData   string
	inferInstancetype      bool
	inferPreference        bool
}

type cloneVolume struct {
	Name   string             `param:"name"`
	Source string             `param:"src"`
	Size   *resource.Quantity `param:"size"`
}

type containerdiskVolume struct {
	Name   string `param:"name"`
	Source string `param:"src"`
}

type pvcVolume struct {
	Name   string `param:"name"`
	Source string `param:"src"`
}

type blankVolume struct {
	Name string             `param:"name"`
	Size *resource.Quantity `param:"size"`
}

type optionFn func(*createVM, *v1.VirtualMachine) error

var optFns = map[string]optionFn{
	RunStrategyFlag:          withRunStrategy,
	InstancetypeFlag:         withInstancetype,
	InferInstancetypeFlag:    withInferredInstancetype,
	PreferenceFlag:           withPreference,
	InferPreferenceFlag:      withInferredPreference,
	ContainerdiskVolumeFlag:  withContainerdiskVolume,
	DataSourceVolumeFlag:     withDataSourceVolume,
	ClonePvcVolumeFlag:       withClonePvcVolume,
	PvcVolumeFlag:            withPvcVolume,
	BlankVolumeFlag:          withBlankVolume,
	CloudInitUserDataFlag:    withCloudInitUserData,
	CloudInitNetworkDataFlag: withCloudInitNetworkData,
}

// Until a param to control the boot order is introduced volumes have the following fixed boot order:
// Containerdisk > DataSource > Clone PVC > PVC
// This is controlled by the order in which flags are processed.
var flags = []string{
	RunStrategyFlag,
	InstancetypeFlag,
	PreferenceFlag,
	ContainerdiskVolumeFlag,
	DataSourceVolumeFlag,
	ClonePvcVolumeFlag,
	PvcVolumeFlag,
	BlankVolumeFlag,
	CloudInitUserDataFlag,
	CloudInitNetworkDataFlag,
	InferInstancetypeFlag,
	InferPreferenceFlag,
}

var runStrategies = []string{
	string(v1.RunStrategyAlways),
	string(v1.RunStrategyManual),
	string(v1.RunStrategyHalted),
	string(v1.RunStrategyOnce),
	string(v1.RunStrategyRerunOnFailure),
}

func NewCommand() *cobra.Command {
	c := defaultCreateVM()
	cmd := &cobra.Command{
		Use:     VM,
		Short:   "Create a VirtualMachine manifest.",
		Long:    "Create a VirtualMachine manifest.\n\nPlease note that volumes currently have the following fixed boot order:\nContainerdisk > DataSource > Clone PVC > PVC",
		Example: c.usage(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.run(cmd)
		},
	}

	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the VM.")
	cmd.Flags().StringVar(&c.runStrategy, RunStrategyFlag, c.runStrategy, "Specify the RunStrategy of the VM.")
	cmd.Flags().Int64Var(&c.terminationGracePeriod, TerminationGracePeriodFlag, c.terminationGracePeriod, "Specify the termination grace period of the VM.")

	cmd.Flags().StringVar(&c.instancetype, InstancetypeFlag, c.instancetype, "Specify the Instance Type of the VM.")
	cmd.Flags().BoolVar(&c.inferInstancetype, InferInstancetypeFlag, c.inferInstancetype, "Specify that the Instance Type of the VM is inferred from the booted volume.")
	cmd.MarkFlagsMutuallyExclusive(InstancetypeFlag, InferInstancetypeFlag)

	cmd.Flags().StringVar(&c.preference, PreferenceFlag, c.preference, "Specify the Preference of the VM.")
	cmd.Flags().BoolVar(&c.inferPreference, InferPreferenceFlag, c.inferPreference, "Specify that the Preference of the VM is inferred from the booted volume.")
	cmd.MarkFlagsMutuallyExclusive(PreferenceFlag, InferPreferenceFlag)

	cmd.Flags().StringArrayVar(&c.containerdiskVolumes, ContainerdiskVolumeFlag, c.containerdiskVolumes, fmt.Sprintf("Specify a containerdisk to be used by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(containerdiskVolume{})))
	cmd.Flags().StringArrayVar(&c.dataSourceVolumes, DataSourceVolumeFlag, c.dataSourceVolumes, fmt.Sprintf("Specify a DataSource to be cloned by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(cloneVolume{})))
	cmd.Flags().StringArrayVar(&c.clonePvcVolumes, ClonePvcVolumeFlag, c.clonePvcVolumes, fmt.Sprintf("Specify a PVC to be cloned by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(cloneVolume{})))
	cmd.Flags().StringArrayVar(&c.pvcVolumes, PvcVolumeFlag, c.pvcVolumes, fmt.Sprintf("Specify a PVCs to be used by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(pvcVolume{})))
	cmd.Flags().StringArrayVar(&c.blankVolumes, BlankVolumeFlag, c.dataSourceVolumes, fmt.Sprintf("Specify a blank volume to be used by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(blankVolume{})))

	cmd.Flags().StringVar(&c.cloudInitUserData, CloudInitUserDataFlag, c.cloudInitUserData, "Specify the base64 encoded cloud-init user data of the VM.")
	cmd.Flags().StringVar(&c.cloudInitNetworkData, CloudInitNetworkDataFlag, c.cloudInitNetworkData, "Specify the base64 encoded cloud-init network data of the VM.")

	cmd.Flags().SortFlags = false
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func defaultCreateVM() createVM {
	return createVM{
		terminationGracePeriod: 180,
		runStrategy:            string(v1.RunStrategyAlways),
	}
}

func checkVolumeExists(flag string, vols []v1.Volume, name string) error {
	for _, vol := range vols {
		if vol.Name == name {
			return params.FlagErr(flag, "there is already a Volume with name '%s'", name)
		}
	}

	return nil
}

func (c *createVM) run(cmd *cobra.Command) error {
	c.setDefaults()

	vm := newVM(c)
	for _, flag := range flags {
		if cmd.Flags().Changed(flag) {
			if err := optFns[flag](c, vm); err != nil {
				return err
			}
		}
	}

	out, err := yaml.Marshal(vm)
	if err != nil {
		return err
	}

	cmd.Print(string(out))
	return nil
}

func (c *createVM) setDefaults() {
	if c.name == "" {
		c.name = "vm-" + rand.String(5)
	}
}

func (c *createVM) usage() string {
	return `  # Create a manifest for a VirtualMachine with a random name:
  {{ProgramName}} create vm

  # Create a manifest for a VirtualMachine with a specified name and RunStrategy Always
  {{ProgramName}} create vm --name=my-vm --run-strategy=Always

  # Create a manifest for a VirtualMachine with a specified VirtualMachineClusterInstancetype
  {{ProgramName}} create vm --instancetype=my-instancetype

  # Create a manifest for a VirtualMachine with a specified VirtualMachineInstancetype (namespaced)
  {{ProgramName}} create vm --instancetype=virtualmachineinstancetype/my-instancetype

  # Create a manifest for a VirtualMachine with a specified VirtualMachineClusterPreference
  {{ProgramName}} create vm --preference=my-preference

  # Create a manifest for a VirtualMachine with a specified VirtualMachinePreference (namespaced)
  {{ProgramName}} create vm --preference=virtualmachinepreference/my-preference

  # Create a manifest for a VirtualMachine with an ephemeral containerdisk volume
  {{ProgramName}} create vm --volume-containerdisk=src:my.registry/my-image:my-tag

  # Create a manifest for a VirtualMachine with a cloned DataSource in namespace and specified size
  {{ProgramName}} create vm --volume-datasource=src:my-ns/my-ds,size:50Gi

  # Create a manifest for a VirtualMachine with a cloned DataSource and inferred instancetype and preference
  {{ProgramName}} create vm --volume-datasource=src:my-annotated-ds --infer-instancetype --infer-preference

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and cloned PVC
  {{ProgramName}} create vm --volume-clone-pvc=my-ns/my-pvc

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and directly used PVC
  {{ProgramName}} create vm --volume-pvc=my-pvc

  # Create a manifest for a VirtualMachine with a clone DataSource and a blank volume
  {{ProgramName}} create vm --volume-datasource=src:my-ns/my-ds --volume-blank=size:50Gi

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and cloned DataSource
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-datasource=src:my-ds

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and two cloned DataSources (flag can be provided multiple times)
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-datasource=src:my-ds1 --volume-datasource=src:my-ds2

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and directly used PVC
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-pvc=my-pvc`
}

func newVM(c *createVM) *v1.VirtualMachine {
	runStrategy := v1.VirtualMachineRunStrategy(c.runStrategy)
	return &v1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1.VirtualMachineGroupVersionKind.Kind,
			APIVersion: v1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.name,
		},
		Spec: v1.VirtualMachineSpec{
			RunStrategy: &runStrategy,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				Spec: v1.VirtualMachineInstanceSpec{
					TerminationGracePeriodSeconds: &c.terminationGracePeriod,
				},
			},
		},
	}
}

func withRunStrategy(c *createVM, vm *v1.VirtualMachine) error {
	for _, runStrategy := range runStrategies {
		if runStrategy == c.runStrategy {
			vmRunStrategy := v1.VirtualMachineRunStrategy(c.runStrategy)
			vm.Spec.RunStrategy = &vmRunStrategy
			return nil
		}
	}

	return params.FlagErr(RunStrategyFlag, "invalid RunStrategy \"%s\", supported values are: %s", c.runStrategy, strings.Join(runStrategies, ", "))
}

func withInstancetype(c *createVM, vm *v1.VirtualMachine) error {
	kind, name, err := params.SplitPrefixedName(c.instancetype)
	if err != nil {
		return params.FlagErr(InstancetypeFlag, "%w", err)
	}

	if kind != "" && kind != instancetype.SingularResourceName && kind != instancetype.ClusterSingularResourceName {
		return params.FlagErr(InstancetypeFlag, "invalid instancetype kind \"%s\", supported values are: %s, %s", kind, instancetype.SingularResourceName, instancetype.ClusterSingularResourceName)
	}

	// If kind is empty we rely on the vm-mutator to fill in the default value VirtualMachineClusterInstancetype
	vm.Spec.Instancetype = &v1.InstancetypeMatcher{
		Name: name,
		Kind: kind,
	}

	return nil
}

func withInferredInstancetype(_ *createVM, vm *v1.VirtualMachine) error {
	if len(vm.Spec.Template.Spec.Volumes) < 1 {
		return params.FlagErr(InferInstancetypeFlag, "at least one volume is needed to infer instancetype")
	}

	// TODO Expand this in the future to take a string containing the volume name to infer
	// the instancetype from. For now this is inferring the instancetype from the first volume
	// in the VM spec.
	vm.Spec.Instancetype = &v1.InstancetypeMatcher{
		InferFromVolume: vm.Spec.Template.Spec.Volumes[0].Name,
	}

	return nil
}

func withPreference(c *createVM, vm *v1.VirtualMachine) error {
	kind, name, err := params.SplitPrefixedName(c.preference)
	if err != nil {
		return params.FlagErr(PreferenceFlag, "%w", err)
	}

	if kind != "" && kind != instancetype.SingularPreferenceResourceName && kind != instancetype.ClusterSingularPreferenceResourceName {
		return params.FlagErr(InstancetypeFlag, "invalid preference kind \"%s\", supported values are: %s, %s", kind, instancetype.SingularPreferenceResourceName, instancetype.ClusterSingularPreferenceResourceName)
	}

	// If kind is empty we rely on the vm-mutator to fill in the default value VirtualMachineClusterPreference
	vm.Spec.Preference = &v1.PreferenceMatcher{
		Name: name,
		Kind: kind,
	}

	return nil
}

func withInferredPreference(_ *createVM, vm *v1.VirtualMachine) error {
	if len(vm.Spec.Template.Spec.Volumes) < 1 {
		return params.FlagErr(InferPreferenceFlag, "at least one volume is needed to infer preference")
	}

	// TODO Expand this in the future to take a string containing the volume name to infer
	// the preference from. For now this is inferring the preference from the first volume
	// in the VM spec.
	vm.Spec.Preference = &v1.PreferenceMatcher{
		InferFromVolume: vm.Spec.Template.Spec.Volumes[0].Name,
	}

	return nil
}

func withContainerdiskVolume(c *createVM, vm *v1.VirtualMachine) error {
	for i, containerdiskVol := range c.containerdiskVolumes {
		vol := containerdiskVolume{}
		err := params.Map(ContainerdiskVolumeFlag, containerdiskVol, &vol)
		if err != nil {
			return err
		}

		if vol.Source == "" {
			return params.FlagErr(ContainerdiskVolumeFlag, "src must be specified")
		}

		if vol.Name == "" {
			vol.Name = fmt.Sprintf("%s-containerdisk-%d", vm.Name, i)
		}

		if err := checkVolumeExists(ContainerdiskVolumeFlag, vm.Spec.Template.Spec.Volumes, vol.Name); err != nil {
			return err
		}

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: vol.Name,
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{
					Image: vol.Source,
				},
			},
		})
	}

	return nil
}

func withDataSourceVolume(c *createVM, vm *v1.VirtualMachine) error {
	for _, dataSourceVol := range c.dataSourceVolumes {
		vol := cloneVolume{}
		err := params.Map(DataSourceVolumeFlag, dataSourceVol, &vol)
		if err != nil {
			return err
		}

		if vol.Source == "" {
			return params.FlagErr(DataSourceVolumeFlag, "src must be specified")
		}

		namespace, name, err := params.SplitPrefixedName(vol.Source)
		if err != nil {
			return params.FlagErr(DataSourceVolumeFlag, "src invalid: %w", err)
		}

		if vol.Name == "" {
			vol.Name = fmt.Sprintf("%s-ds-%s", vm.Name, name)
		}

		if err := checkVolumeExists(DataSourceVolumeFlag, vm.Spec.Template.Spec.Volumes, vol.Name); err != nil {
			return err
		}

		dvt := v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: vol.Name,
			},
			Spec: cdiv1.DataVolumeSpec{
				Storage: &cdiv1.StorageSpec{},
				SourceRef: &cdiv1.DataVolumeSourceRef{
					Kind: "DataSource",
					Name: name,
				},
			},
		}
		if namespace != "" {
			dvt.Spec.SourceRef.Namespace = &namespace
		}
		if vol.Size != nil {
			dvt.Spec.Storage.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceStorage: *vol.Size,
			}
		}
		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, dvt)

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: vol.Name,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: vol.Name,
				},
			},
		})
	}

	return nil
}

func withClonePvcVolume(c *createVM, vm *v1.VirtualMachine) error {
	for _, clonePvcVol := range c.clonePvcVolumes {
		vol := cloneVolume{}
		err := params.Map(ClonePvcVolumeFlag, clonePvcVol, &vol)
		if err != nil {
			return err
		}

		if vol.Source == "" {
			return params.FlagErr(ClonePvcVolumeFlag, "src must be specified")
		}

		namespace, name, err := params.SplitPrefixedName(vol.Source)
		if err != nil {
			return params.FlagErr(ClonePvcVolumeFlag, "src invalid: %w", err)
		}
		if namespace == "" {
			return params.FlagErr(ClonePvcVolumeFlag, "namespace of pvc '%s' must be specified", name)
		}

		if vol.Name == "" {
			vol.Name = fmt.Sprintf("%s-pvc-%s", vm.Name, name)
		}

		if err := checkVolumeExists(ClonePvcVolumeFlag, vm.Spec.Template.Spec.Volumes, vol.Name); err != nil {
			return err
		}

		dvt := v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: vol.Name,
			},
			Spec: cdiv1.DataVolumeSpec{
				Storage: &cdiv1.StorageSpec{},
				Source: &cdiv1.DataVolumeSource{
					PVC: &cdiv1.DataVolumeSourcePVC{
						Namespace: namespace,
						Name:      name,
					},
				},
			},
		}
		if vol.Size != nil {
			dvt.Spec.Storage.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceStorage: *vol.Size,
			}
		}
		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, dvt)

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: vol.Name,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: vol.Name,
				},
			},
		})
	}

	return nil
}

func withPvcVolume(c *createVM, vm *v1.VirtualMachine) error {
	for _, pvcVol := range c.pvcVolumes {
		vol := pvcVolume{}
		err := params.Map(PvcVolumeFlag, pvcVol, &vol)
		if err != nil {
			return err
		}

		if vol.Source == "" {
			return params.FlagErr(PvcVolumeFlag, "src must be specified")
		}

		namespace, name, err := params.SplitPrefixedName(vol.Source)
		if err != nil {
			return params.FlagErr(PvcVolumeFlag, "src invalid: %w", err)
		}
		if namespace != "" {
			return params.FlagErr(PvcVolumeFlag, "not allowed to specify namespace of pvc '%s'", name)
		}

		if vol.Name == "" {
			vol.Name = name
		}

		if err := checkVolumeExists(PvcVolumeFlag, vm.Spec.Template.Spec.Volumes, vol.Name); err != nil {
			return err
		}

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: vol.Name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: name,
					},
				},
			},
		})
	}

	return nil
}

func withBlankVolume(c *createVM, vm *v1.VirtualMachine) error {
	for i, blankVol := range c.blankVolumes {
		vol := blankVolume{}
		err := params.Map(BlankVolumeFlag, blankVol, &vol)
		if err != nil {
			return err
		}

		if vol.Size == nil {
			return params.FlagErr(BlankVolumeFlag, "size must be specified")
		}

		if vol.Name == "" {
			vol.Name = fmt.Sprintf("%s-blank-%d", vm.Name, i)
		}

		if err := checkVolumeExists(BlankVolumeFlag, vm.Spec.Template.Spec.Volumes, vol.Name); err != nil {
			return err
		}

		vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Name: vol.Name,
			},
			Spec: cdiv1.DataVolumeSpec{
				Storage: &cdiv1.StorageSpec{
					Resources: k8sv1.ResourceRequirements{
						Requests: k8sv1.ResourceList{
							k8sv1.ResourceStorage: *vol.Size,
						},
					},
				},
				Source: &cdiv1.DataVolumeSource{
					Blank: &cdiv1.DataVolumeBlankImage{},
				},
			},
		})

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: vol.Name,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: vol.Name,
				},
			},
		})
	}

	return nil
}

func withCloudInitUserData(c *createVM, vm *v1.VirtualMachine) error {
	return withCloudInitData(CloudInitUserDataFlag, c, vm)
}

func withCloudInitNetworkData(c *createVM, vm *v1.VirtualMachine) error {
	// Skip if cloudInitUserData is not empty, only one cloudInitDisk can be created
	if c.cloudInitUserData != "" {
		return nil
	}

	return withCloudInitData(CloudInitNetworkDataFlag, c, vm)
}

func withCloudInitData(flag string, c *createVM, vm *v1.VirtualMachine) error {
	// Make sure cloudInitDisk does not already exist
	if err := checkVolumeExists(flag, vm.Spec.Template.Spec.Volumes, cloudInitDisk); err != nil {
		return err
	}

	// TODO This is using the NoCloud method to provide cloudInit data.
	// The ConfigDrive method can be implemented in the future if needed.
	cloudInitNoCloud := &v1.CloudInitNoCloudSource{}
	if c.cloudInitNetworkData != "" {
		cloudInitNoCloud.NetworkDataBase64 = c.cloudInitNetworkData
	}
	if c.cloudInitUserData != "" {
		cloudInitNoCloud.UserDataBase64 = c.cloudInitUserData
	}
	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
		Name: cloudInitDisk,
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: cloudInitNoCloud,
		},
	})

	return nil
}
