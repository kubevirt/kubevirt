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
	"sort"
	"strings"

	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/clientcmd"
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
	MemoryFlag                 = "memory"
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
	InferInstancetypeFromFlag  = "infer-instancetype-from"
	InferPreferenceFlag        = "infer-preference"
	InferPreferenceFromFlag    = "infer-preference-from"
	VolumeImportFlag           = "volume-import"

	cloudInitDisk = "cloudinitdisk"
	blank         = "blank"
	http          = "http"
	imageIO       = "imageio"
	pvc           = "pvc"
	registry      = "registry"
	s3            = "s3"
	vddk          = "vddk"
	snapshot      = "snapshot"

	InvalidInferenceVolumeError = "inference of instancetype or preference works only with DataSources, DataVolumes or PersistentVolumeClaims"
)

type createVM struct {
	namespace              string
	name                   string
	terminationGracePeriod int64
	runStrategy            string
	memory                 string
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
	inferInstancetypeFrom  string
	inferPreference        bool
	inferPreferenceFrom    string
	volumeImport           []string

	clientConfig clientcmd.ClientConfig
	bootOrders   map[uint]string

	explicitInstancetypeInference bool
	explicitPreferenceInference   bool
	memoryChanged                 bool
}

type cloneVolume struct {
	Name      string             `param:"name"`
	Source    string             `param:"src"`
	BootOrder *uint              `param:"bootorder"`
	Size      *resource.Quantity `param:"size"`
}

type containerdiskVolume struct {
	Name      string `param:"name"`
	Source    string `param:"src"`
	BootOrder *uint  `param:"bootorder"`
}

type pvcVolume struct {
	Name      string `param:"name"`
	Source    string `param:"src"`
	BootOrder *uint  `param:"bootorder"`
}

type blankVolume struct {
	Name string             `param:"name"`
	Size *resource.Quantity `param:"size"`
}

type optionFn func(*createVM, *v1.VirtualMachine) error

var optFns = map[string]optionFn{
	RunStrategyFlag:          withRunStrategy,
	InstancetypeFlag:         withInstancetype,
	PreferenceFlag:           withPreference,
	ContainerdiskVolumeFlag:  withContainerdiskVolume,
	DataSourceVolumeFlag:     withDataSourceVolume,
	ClonePvcVolumeFlag:       withClonePvcVolume,
	PvcVolumeFlag:            withPvcVolume,
	BlankVolumeFlag:          withBlankVolume,
	CloudInitUserDataFlag:    withCloudInitUserData,
	CloudInitNetworkDataFlag: withCloudInitNetworkData,
	VolumeImportFlag:         withImportedVolume,
}

// Unless the boot order is specified by the user volumes have the following fixed boot order:
// Containerdisk > DataSource > Clone PVC > PVC
// Flags dependent on the boot order (e.g. InferInstancetype or InferPreference) need to run last.
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
	VolumeImportFlag,
	CloudInitUserDataFlag,
	CloudInitNetworkDataFlag,
}

type dataVolumeSourceBlank struct {
	Size *resource.Quantity `param:"size"`
	Type string             `param:"type"`
	Name string             `param:"name"`
}

type dataVolumeSourceHttp struct {
	CertConfigMap      string             `param:"certconfigmap"`
	ExtraHeaders       []string           `param:"extraheaders"`
	SecretExtraHeaders []string           `param:"secretextraheaders"`
	SecretRef          string             `param:"secretref"`
	URL                string             `param:"url"`
	Size               *resource.Quantity `param:"size"`
	Type               string             `param:"type"`
	Name               string             `param:"name"`
}

type dataVolumeSourceImageIO struct {
	CertConfigMap string             `param:"certconfigmap"`
	DiskId        string             `param:"diskid"`
	SecretRef     string             `param:"secretref"`
	URL           string             `param:"url"`
	Size          *resource.Quantity `param:"size"`
	Type          string             `param:"type"`
	Name          string             `param:"name"`
}

type dataVolumeSourcePVC struct {
	Name   string             `param:"name"`
	Source string             `param:"src"`
	Size   *resource.Quantity `param:"size"`
	Type   string             `param:"type"`
}

type dataVolumeSourceRegistry struct {
	CertConfigMap string             `param:"certconfigmap"`
	ImageStream   string             `param:"imagestream"`
	PullMethod    string             `param:"pullmethod"`
	SecretRef     string             `param:"secretref"`
	URL           string             `param:"url"`
	Size          *resource.Quantity `param:"size"`
	Type          string             `param:"type"`
	Name          string             `param:"name"`
}

type dataVolumeSourceS3 struct {
	CertConfigMap string             `param:"certconfigmap"`
	SecretRef     string             `param:"secretref"`
	URL           string             `param:"url"`
	Size          *resource.Quantity `param:"size"`
	Type          string             `param:"type"`
	Name          string             `param:"name"`
}

type dataVolumeSourceVDDK struct {
	BackingFile  string             `param:"backingfile"`
	InitImageUrl string             `param:"initimageurl"`
	SecretRef    string             `param:"secretref"`
	ThumbPrint   string             `param:"thumbprint"`
	URL          string             `param:"url"`
	UUID         string             `param:"uuid"`
	Size         *resource.Quantity `param:"size"`
	Type         string             `param:"type"`
	Name         string             `param:"name"`
}

type dataVolumeSourceSnapshot struct {
	Name   string             `param:"name"`
	Source string             `param:"src"`
	Size   *resource.Quantity `param:"size"`
	Type   string             `param:"type"`
}

type volumeImportFn func(string, *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error)

var volumeImportOptions = map[string]volumeImportFn{
	blank:    withVolumeSourceBlank,
	http:     withVolumeSourceHttp,
	imageIO:  withVolumeSourceImageIO,
	pvc:      withVolumeSourcePVC,
	registry: withVolumeSourceRegistry,
	s3:       withVolumeSourceS3,
	vddk:     withVolumeSourceVDDK,
	snapshot: withVolumeSourceSnapshot,
}

var runStrategies = []string{
	string(v1.RunStrategyAlways),
	string(v1.RunStrategyManual),
	string(v1.RunStrategyHalted),
	string(v1.RunStrategyOnce),
	string(v1.RunStrategyRerunOnFailure),
}

func NewCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	c := defaultCreateVM(clientConfig)
	cmd := &cobra.Command{
		Use:     VM,
		Short:   "Create a VirtualMachine manifest.",
		Long:    "Create a VirtualMachine manifest.\n\nIf no boot order was specified volumes have the following fixed boot order:\nContainerdisk > DataSource > Clone PVC > PVC",
		Args:    cobra.NoArgs,
		Example: c.usage(),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return c.run(cmd)
		},
	}

	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the VM.")
	cmd.Flags().StringVar(&c.runStrategy, RunStrategyFlag, c.runStrategy, "Specify the RunStrategy of the VM.")
	cmd.Flags().Int64Var(&c.terminationGracePeriod, TerminationGracePeriodFlag, c.terminationGracePeriod, "Specify the termination grace period of the VM.")

	cmd.Flags().StringVar(&c.memory, MemoryFlag, c.memory, "Specify the memory of the VM.")
	cmd.Flags().StringVar(&c.instancetype, InstancetypeFlag, c.instancetype, "Specify the Instance Type of the VM. Mutually exclusive with instancetype inference flags.")
	cmd.Flags().BoolVar(&c.inferInstancetype, InferInstancetypeFlag, c.inferInstancetype, "Specify if the Instance Type of the VM should be inferred from the first boot disk. Mutually exclusive with --infer-instancetype-from.")
	cmd.Flags().StringVar(&c.inferInstancetypeFrom, InferInstancetypeFromFlag, c.inferInstancetypeFrom, "Specify the volume to infer the Instance Type of the VM from. Mutually exclusive with --infer-instancetype.")
	cmd.MarkFlagsMutuallyExclusive(MemoryFlag, InstancetypeFlag, InferInstancetypeFlag, InferInstancetypeFromFlag)

	cmd.Flags().StringVar(&c.preference, PreferenceFlag, c.preference, "Specify the Preference of the VM. Mutually exclusive with preference inference flags.")
	cmd.Flags().BoolVar(&c.inferPreference, InferPreferenceFlag, c.inferPreference, "Specify if the Preference of the VM should be inferred from the first boot disk. Mutually exclusive with --infer-preference-from.")
	cmd.Flags().StringVar(&c.inferPreferenceFrom, InferPreferenceFromFlag, c.inferPreferenceFrom, "Specify the volume to infer the Preference of the VM from. Mutually exclusive with --infer-preference.")
	cmd.MarkFlagsMutuallyExclusive(PreferenceFlag, InferPreferenceFlag, InferPreferenceFromFlag)

	cmd.Flags().StringArrayVar(&c.containerdiskVolumes, ContainerdiskVolumeFlag, c.containerdiskVolumes, fmt.Sprintf("Specify a containerdisk to be used by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(containerdiskVolume{})))
	cmd.Flags().StringArrayVar(&c.dataSourceVolumes, DataSourceVolumeFlag, c.dataSourceVolumes, fmt.Sprintf("Specify a DataSource to be cloned by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(cloneVolume{})))
	cmd.Flags().StringArrayVar(&c.clonePvcVolumes, ClonePvcVolumeFlag, c.clonePvcVolumes, fmt.Sprintf("Specify a PVC to be cloned by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(cloneVolume{})))
	cmd.Flags().StringArrayVar(&c.pvcVolumes, PvcVolumeFlag, c.pvcVolumes, fmt.Sprintf("Specify a PVCs to be used by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(pvcVolume{})))
	cmd.Flags().StringArrayVar(&c.blankVolumes, BlankVolumeFlag, c.blankVolumes, fmt.Sprintf("Specify a blank volume to be used by the VM. Can be provided multiple times.\nSupported parameters: %s", params.Supported(blankVolume{})))
	cmd.Flags().StringArrayVar(&c.volumeImport, VolumeImportFlag, c.volumeImport, fmt.Sprintf(
		"Specify the source for DataVolume. Can be provided multiple times.\nSupported parameters:\n  type %s - %s\n  type %s - %s\n  type %s - %s\n  type %s - %s\n  type %s - %s\n  type %s - %s\n  type %s - %s\n  type %s - %s",
		blank, params.Supported(dataVolumeSourceBlank{}),
		http, params.Supported(dataVolumeSourceHttp{}),
		imageIO, params.Supported(dataVolumeSourceImageIO{}),
		pvc, params.Supported(dataVolumeSourcePVC{}),
		registry, params.Supported(dataVolumeSourceRegistry{}),
		s3, params.Supported(dataVolumeSourceS3{}),
		vddk, params.Supported(dataVolumeSourceVDDK{}),
		snapshot, params.Supported(dataVolumeSourceSnapshot{}),
	))

	cmd.Flags().StringVar(&c.cloudInitUserData, CloudInitUserDataFlag, c.cloudInitUserData, "Specify the base64 encoded cloud-init user data of the VM.")
	cmd.Flags().StringVar(&c.cloudInitNetworkData, CloudInitNetworkDataFlag, c.cloudInitNetworkData, "Specify the base64 encoded cloud-init network data of the VM.")

	cmd.Flags().SortFlags = false
	cmd.SetUsageTemplate(templates.UsageTemplate())

	return cmd
}

func defaultCreateVM(clientConfig clientcmd.ClientConfig) createVM {
	return createVM{
		terminationGracePeriod: 180,
		runStrategy:            string(v1.RunStrategyAlways),
		memory:                 "512Mi",
		inferInstancetype:      true,
		inferPreference:        true,
		clientConfig:           clientConfig,
		bootOrders:             map[uint]string{},
	}
}

func volumeExists(vm *v1.VirtualMachine, name string) *v1.Volume {
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		if vol.Name == name {
			return &vol
		}
	}

	return nil
}

func volumeShouldExist(vm *v1.VirtualMachine, name string) (*v1.Volume, error) {
	if vol := volumeExists(vm, name); vol != nil {
		return vol, nil
	}

	return nil, fmt.Errorf("there is no volume with name '%s'", name)
}

func volumeShouldNotExist(flag string, vm *v1.VirtualMachine, name string) error {
	if vol := volumeExists(vm, name); vol != nil {
		return params.FlagErr(flag, "there is already a volume with name '%s'", name)
	}

	return nil
}

func volumeValidToInferFrom(vm *v1.VirtualMachine, vol *v1.Volume) error {
	if vol.DataVolume != nil {
		return dataVolumeValidToInferFrom(vm, vol)
	}

	if vol.PersistentVolumeClaim != nil {
		return nil
	}

	return fmt.Errorf(InvalidInferenceVolumeError)
}

func dataVolumeValidToInferFrom(vm *v1.VirtualMachine, vol *v1.Volume) error {
	for _, dvt := range vm.Spec.DataVolumeTemplates {
		if dvt.Name == vol.DataVolume.Name {
			if dvt.Spec.Source != nil && dvt.Spec.Source.PVC != nil {
				return nil
			}

			if dvt.Spec.SourceRef != nil && dvt.Spec.SourceRef.Kind == "DataSource" {
				return nil
			}

			return fmt.Errorf(InvalidInferenceVolumeError)
		}
	}

	return nil
}

func (c *createVM) run(cmd *cobra.Command) error {
	if err := c.setDefaults(cmd); err != nil {
		return err
	}

	vm, err := c.newVM()
	if err != nil {
		return err
	}

	for _, flag := range flags {
		if cmd.Flags().Changed(flag) {
			if err := optFns[flag](c, vm); err != nil {
				return err
			}
		}
	}

	if err := c.inferFromVolume(vm); err != nil {
		return err
	}

	out, err := yaml.Marshal(vm)
	if err != nil {
		return err
	}

	cmd.Print(string(out))
	return nil
}

func (c *createVM) setDefaults(cmd *cobra.Command) error {
	namespace, overridden, err := c.clientConfig.Namespace()
	if err != nil {
		return err
	}
	if overridden {
		c.namespace = namespace
	}

	if c.name == "" {
		c.name = "vm-" + rand.String(5)
	}

	c.explicitInstancetypeInference = cmd.Flags().Changed(InferInstancetypeFlag) ||
		cmd.Flags().Changed(InferInstancetypeFromFlag)

	c.explicitPreferenceInference = cmd.Flags().Changed(InferPreferenceFlag) ||
		cmd.Flags().Changed(InferPreferenceFromFlag)

	c.memoryChanged = cmd.Flags().Changed(MemoryFlag)

	return nil
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

  # Create a manifest for a VirtualMachine with specified memory and an ephemeral containerdisk volume
  {{ProgramName}} create vm --memory=1Gi --volume-containerdisk=src:my.registry/my-image:my-tag

  # Create a manifest for a VirtualMachine with a cloned DataSource in namespace and specified size
  {{ProgramName}} create vm --volume-datasource=src:my-ns/my-ds,size:50Gi

  # Create a manifest for a VirtualMachine with a cloned DataSource and inferred instancetype and preference
  {{ProgramName}} create vm --volume-datasource=src:my-annotated-ds --infer-instancetype --infer-preference

  # Create a manifest for a VirtualMachine with multiple volumes and specified boot order
  {{ProgramName}} create vm --volume-containerdisk=src:my.registry/my-image:my-tag --volume-datasource=src:my-ds,bootorder:1

  # Create a manifest for a VirtualMachine with multiple volumes and inferred instancetype and preference with specified volumes
  {{ProgramName}} create vm --volume-datasource=src:my-annotated-ds --volume-pvc=my-annotated-pvc --infer-instancetype=my-annotated-ds --infer-preference=my-annotated-pvc

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and cloned PVC
  {{ProgramName}} create vm --volume-clone-pvc=src:my-ns/my-pvc

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and directly used PVC
  {{ProgramName}} create vm --volume-pvc=src:my-pvc

  # Create a manifest for a VirtualMachine with a clone DataSource and a blank volume
  {{ProgramName}} create vm --volume-datasource=src:my-ns/my-ds --volume-blank=size:50Gi

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and cloned DataSource
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-datasource=src:my-ds

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and two cloned DataSources (flag can be provided multiple times)
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-datasource=src:my-ds1 --volume-datasource=src:my-ds2

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and directly used PVC
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-pvc=my-pvc

  # Create a manifest for a VirtualMachine with a specified DataVolumeTemplate
  {{ProgramName}} create vm --volume-import type:pvc,name:my-pvc,namespace:default,size:256Mi`
}

func (c *createVM) newVM() (*v1.VirtualMachine, error) {
	runStrategy := v1.VirtualMachineRunStrategy(c.runStrategy)
	memory, err := resource.ParseQuantity(c.memory)
	if err != nil {
		return nil, params.FlagErr(MemoryFlag, "%w", err)

	}

	vm := &v1.VirtualMachine{
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
					Domain: v1.DomainSpec{
						Memory: &v1.Memory{
							Guest: &memory,
						},
					},
				},
			},
		},
	}

	if c.namespace != "" {
		vm.Namespace = c.namespace
	}

	return vm, nil
}

func (c *createVM) addDiskWithBootOrder(flag string, vm *v1.VirtualMachine, name string, bootOrder *uint) error {
	if bootOrder != nil {
		if *bootOrder == 0 {
			return params.FlagErr(flag, "bootorder must be greater than 0")
		}

		if _, ok := c.bootOrders[*bootOrder]; ok {
			return params.FlagErr(flag, "bootorder %d was specified multiple times", *bootOrder)
		}

		vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, v1.Disk{
			Name:      name,
			BootOrder: bootOrder,
		})

		c.bootOrders[*bootOrder] = name
	}

	return nil
}

func (c *createVM) inferFromVolume(vm *v1.VirtualMachine) error {
	if c.inferInstancetype && c.instancetype == "" && !c.memoryChanged {
		if err := c.withInferredInstancetype(vm); err != nil && c.explicitInstancetypeInference {
			return err
		}
	}

	if c.inferPreference && c.preference == "" {
		if err := c.withInferredPreference(vm); err != nil && c.explicitPreferenceInference {
			return err
		}
	}

	return nil
}

func (c *createVM) withInferredInstancetype(vm *v1.VirtualMachine) error {
	if c.inferInstancetypeFrom == "" {
		vol, err := c.getInferFromVolume(vm)
		if err != nil {
			return err
		}
		c.inferInstancetypeFrom = vol
	}

	vol, err := volumeShouldExist(vm, c.inferInstancetypeFrom)
	if err != nil {
		return err
	}

	if err := volumeValidToInferFrom(vm, vol); err != nil {
		return err
	}

	vm.Spec.Instancetype = &v1.InstancetypeMatcher{
		InferFromVolume: c.inferInstancetypeFrom,
	}

	if c.explicitInstancetypeInference {
		// If inferring explicitly the default guest memory should be cleared.
		vm.Spec.Template.Spec.Domain.Memory = nil
	} else {
		// If inferring implicitly possible errors during inference should be ignored
		// on the backend because the executed command possibly still was valid.
		// The guest memory should not be cleared to provide a fallback value when inference failed.
		failurePolicy := v1.IgnoreInferFromVolumeFailure
		vm.Spec.Instancetype.InferFromVolumeFailurePolicy = &failurePolicy
	}

	return nil
}

func (c *createVM) withInferredPreference(vm *v1.VirtualMachine) error {
	if c.inferPreferenceFrom == "" {
		vol, err := c.getInferFromVolume(vm)
		if err != nil {
			return err
		}
		c.inferPreferenceFrom = vol
	}

	vol, err := volumeShouldExist(vm, c.inferPreferenceFrom)
	if err != nil {
		return err
	}

	if err := volumeValidToInferFrom(vm, vol); err != nil {
		return err
	}

	vm.Spec.Preference = &v1.PreferenceMatcher{
		InferFromVolume: c.inferPreferenceFrom,
	}

	// If inferring implicitly possible errors during inference should be ignored
	// on the backend because the executed command possibly still was valid.
	if !c.explicitPreferenceInference {
		failurePolicy := v1.IgnoreInferFromVolumeFailure
		vm.Spec.Preference.InferFromVolumeFailurePolicy = &failurePolicy
	}

	return nil
}

// getInferFromVolume returns the volume to infer the instancetype or preference from.
// It returns either the disk with the lowest boot order or the first volume in the VM spec.
func (c *createVM) getInferFromVolume(vm *v1.VirtualMachine) (string, error) {
	if len(vm.Spec.Template.Spec.Volumes) < 1 {
		return "", fmt.Errorf("at least one volume is needed to infer an instance type or preference")
	}

	// Find the lowest boot order and return associated disk name
	if len(c.bootOrders) > 0 {
		var keys []uint
		for k := range c.bootOrders {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		return c.bootOrders[keys[0]], nil
	}

	// Default to the first volume if no boot order was specified
	return vm.Spec.Template.Spec.Volumes[0].Name, nil
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
	vm.Spec.Template.Spec.Domain.Memory = nil
	vm.Spec.Instancetype = &v1.InstancetypeMatcher{
		Name: name,
		Kind: kind,
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

		if err := volumeShouldNotExist(ContainerdiskVolumeFlag, vm, vol.Name); err != nil {
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

		if err := c.addDiskWithBootOrder(ContainerdiskVolumeFlag, vm, vol.Name, vol.BootOrder); err != nil {
			return err
		}
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

		if err := volumeShouldNotExist(DataSourceVolumeFlag, vm, vol.Name); err != nil {
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

		if err := c.addDiskWithBootOrder(DataSourceVolumeFlag, vm, vol.Name, vol.BootOrder); err != nil {
			return err
		}
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

		if err := volumeShouldNotExist(ClonePvcVolumeFlag, vm, vol.Name); err != nil {
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

		if err := c.addDiskWithBootOrder(ClonePvcVolumeFlag, vm, vol.Name, vol.BootOrder); err != nil {
			return err
		}
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

		if err := volumeShouldNotExist(PvcVolumeFlag, vm, vol.Name); err != nil {
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

		if err := c.addDiskWithBootOrder(PvcVolumeFlag, vm, vol.Name, vol.BootOrder); err != nil {
			return err
		}
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

		if err := volumeShouldNotExist(BlankVolumeFlag, vm, vol.Name); err != nil {
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
	if err := volumeShouldNotExist(flag, vm, cloudInitDisk); err != nil {
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

func withImportedVolume(c *createVM, vm *v1.VirtualMachine) error {
	for _, volume := range c.volumeImport {
		volumeSourceType, err := params.GetParamByName("type", volume)
		if err != nil {
			return err
		}

		sourceFn, found := volumeImportOptions[volumeSourceType]
		if !found {
			return params.FlagErr(VolumeImportFlag, fmt.Sprintf("unknown source type used - %s", volumeSourceType))
		}

		source, err := sourceFn(volume, vm)
		if err != nil {
			return err
		}

		size, err := params.GetParamByName("size", volume)
		if err != nil {
			return err
		}

		name, err := params.GetParamByName("name", volume)
		if err != nil {
			name = fmt.Sprintf("imported-volume-%s", rand.String(4))
		}

		if err := createVolumeWithSource(source, size, name, vm); err != nil {
			return err
		}
	}
	return nil
}

func withVolumeSourceBlank(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceBlank{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	source := cdiv1.DataVolumeSource{
		Blank: &cdiv1.DataVolumeBlankImage{},
	}

	return &source, nil
}

func withVolumeSourceHttp(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceHttp{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	if sourceStruct.URL == "" {
		return nil, params.FlagErr(VolumeImportFlag, "URL is required with http volume source")
	}

	source := cdiv1.DataVolumeSource{
		HTTP: &cdiv1.DataVolumeSourceHTTP{
			URL:                sourceStruct.URL,
			SecretRef:          sourceStruct.SecretRef,
			CertConfigMap:      sourceStruct.CertConfigMap,
			ExtraHeaders:       sourceStruct.ExtraHeaders,
			SecretExtraHeaders: sourceStruct.SecretExtraHeaders,
		},
	}

	return &source, nil
}

func withVolumeSourceImageIO(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceImageIO{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	if sourceStruct.URL == "" || sourceStruct.DiskId == "" {
		return nil, params.FlagErr(VolumeImportFlag, "URL and diskid are both required with imageIO volume source")
	}

	source := cdiv1.DataVolumeSource{
		Imageio: &cdiv1.DataVolumeSourceImageIO{
			URL:           sourceStruct.URL,
			DiskID:        sourceStruct.DiskId,
			SecretRef:     sourceStruct.SecretRef,
			CertConfigMap: sourceStruct.CertConfigMap,
		},
	}

	return &source, nil
}

func withVolumeSourcePVC(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourcePVC{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	if sourceStruct.Source == "" {
		return nil, params.FlagErr(VolumeImportFlag, "src must be specified")
	}

	namespace, name, err := params.SplitPrefixedName(sourceStruct.Source)
	if err != nil {
		return nil, params.FlagErr(VolumeImportFlag, "src invalid: %w", err)
	}

	if namespace == "" {
		return nil, params.FlagErr(VolumeImportFlag, "namespace of pvc '%s' must be specified", name)
	}

	source := cdiv1.DataVolumeSource{
		PVC: &cdiv1.DataVolumeSourcePVC{
			Name:      name,
			Namespace: namespace,
		},
	}

	return &source, nil
}

func withVolumeSourceRegistry(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceRegistry{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	if sourceStruct.PullMethod != "" &&
		(sourceStruct.PullMethod != string(cdiv1.RegistryPullPod) && sourceStruct.PullMethod != string(cdiv1.RegistryPullNode)) {
		return nil, params.FlagErr(VolumeImportFlag, "pullmethod must be set to pod or node")
	}

	if (sourceStruct.URL == "" && sourceStruct.ImageStream == "") ||
		(sourceStruct.URL != "" && sourceStruct.ImageStream != "") {
		return nil, params.FlagErr(VolumeImportFlag, "exactly one of url or imagestream must be defined")
	}

	source := cdiv1.DataVolumeSource{Registry: &cdiv1.DataVolumeSourceRegistry{}}

	if sourceStruct.PullMethod != "" {
		source.Registry.PullMethod = (*cdiv1.RegistryPullMethod)(&sourceStruct.PullMethod)
	}

	if sourceStruct.CertConfigMap != "" {
		source.Registry.CertConfigMap = &sourceStruct.CertConfigMap
	}

	if sourceStruct.ImageStream != "" {
		source.Registry.ImageStream = &sourceStruct.ImageStream
	}

	if sourceStruct.URL != "" {
		source.Registry.URL = &sourceStruct.URL
	}

	if sourceStruct.SecretRef != "" {
		source.Registry.SecretRef = &sourceStruct.SecretRef
	}

	return &source, nil
}

func withVolumeSourceS3(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceS3{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	if sourceStruct.URL == "" {
		return nil, params.FlagErr(VolumeImportFlag, "URL is required with S3 volume source")
	}

	source := cdiv1.DataVolumeSource{
		S3: &cdiv1.DataVolumeSourceS3{
			URL:           sourceStruct.URL,
			CertConfigMap: sourceStruct.CertConfigMap,
			SecretRef:     sourceStruct.SecretRef,
		},
	}

	return &source, nil
}

func withVolumeSourceVDDK(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceVDDK{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	source := cdiv1.DataVolumeSource{
		VDDK: &cdiv1.DataVolumeSourceVDDK{
			URL:          sourceStruct.URL,
			UUID:         sourceStruct.UUID,
			Thumbprint:   sourceStruct.ThumbPrint,
			SecretRef:    sourceStruct.SecretRef,
			InitImageURL: sourceStruct.InitImageUrl,
			BackingFile:  sourceStruct.BackingFile,
		},
	}

	return &source, nil
}

func withVolumeSourceSnapshot(paramStr string, vm *v1.VirtualMachine) (*cdiv1.DataVolumeSource, error) {
	sourceStruct := dataVolumeSourceSnapshot{}
	if err := params.Map(VolumeImportFlag, paramStr, &sourceStruct); err != nil {
		return nil, err
	}

	if sourceStruct.Source == "" {
		return nil, params.FlagErr(VolumeImportFlag, "src must be specified")
	}

	namespace, name, err := params.SplitPrefixedName(sourceStruct.Source)
	if err != nil {
		return nil, params.FlagErr(VolumeImportFlag, "src invalid: %w", err)
	}

	if namespace == "" {
		return nil, params.FlagErr(VolumeImportFlag, "namespace of snapshot '%s' must be specified", name)
	}

	source := cdiv1.DataVolumeSource{
		Snapshot: &cdiv1.DataVolumeSourceSnapshot{
			Name:      name,
			Namespace: namespace,
		},
	}

	return &source, nil
}

func createVolumeWithSource(source *cdiv1.DataVolumeSource, size string, name string, vm *v1.VirtualMachine) error {
	if err := volumeShouldNotExist(VolumeImportFlag, vm, name); err != nil {
		return err
	}

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, v1.DataVolumeTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: cdiv1.DataVolumeSpec{
			Source: source,
			Storage: &cdiv1.StorageSpec{
				Resources: k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceStorage: resource.MustParse(size),
					},
				},
			},
		},
	})

	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: name,
			},
		},
	})

	return nil
}
