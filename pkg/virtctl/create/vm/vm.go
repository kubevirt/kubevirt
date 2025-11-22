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

package vm

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/yaml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virtctl/clientconfig"
	"kubevirt.io/kubevirt/pkg/virtctl/create/params"
)

const (
	NameFlag                   = "name"
	RunStrategyFlag            = "run-strategy"
	TerminationGracePeriodFlag = "termination-grace-period"

	MemoryFlag                = "memory"
	InstancetypeFlag          = "instancetype"
	InferInstancetypeFlag     = "infer-instancetype"
	InferInstancetypeFromFlag = "infer-instancetype-from"

	PreferenceFlag          = "preference"
	InferPreferenceFlag     = "infer-preference"
	InferPreferenceFromFlag = "infer-preference-from"

	ContainerdiskVolumeFlag = "volume-containerdisk"
	PvcVolumeFlag           = "volume-pvc"
	VolumeImportFlag        = "volume-import"
	SysprepVolumeFlag       = "volume-sysprep"

	UserFlag         = "user"
	PasswordFileFlag = "password-file"
	SSHKeyFlag       = "ssh-key"
	GAManageSSHFlag  = "ga-manage-ssh"
	AccessCredFlag   = "access-cred"

	CloudInitFlag            = "cloud-init"
	CloudInitUserDataFlag    = "cloud-init-user-data"
	CloudInitNetworkDataFlag = "cloud-init-network-data"

	// Deprecated flags
	DataSourceVolumeFlag = "volume-datasource"
	ClonePvcVolumeFlag   = "volume-clone-pvc"
	BlankVolumeFlag      = "volume-blank"

	sysprepDisk      = "sysprepdisk"
	sysprepConfigMap = "configmap"
	sysprepSecret    = "secret"

	cloudInitDisk         = "cloudinitdisk"
	cloudInitNoCloud      = "nocloud"
	cloudInitConfigDrive  = "configdrive"
	cloudInitNone         = "none"
	cloudInitConfigHeader = "#cloud-config"

	accessCredTypeSSH      = "ssh"
	accessCredTypePassword = "password"
	accessCredMethodGA     = "ga"

	blank    = "blank"
	gcs      = "gcs"
	http     = "http"
	imageIO  = "imageio"
	pvc      = "pvc"
	registry = "registry"
	s3       = "s3"
	vddk     = "vddk"
	snapshot = "snapshot"
	ds       = "ds"

	volumeExistsErrorFmt                 = "there is already a volume with name \"%s\""
	accessCredUserInvalidError           = "user cannot be specified with selected access credential type and method"
	accessCredMethodFlagMismatchErrorFmt = "method param and value passed to --%s have to match: %s vs %s"
	randSuffixLength                     = 5
)

type createVM struct {
	name                   string
	runStrategy            string
	terminationGracePeriod int64

	memory                string
	instancetype          string
	inferInstancetype     bool
	inferInstancetypeFrom string

	preference          string
	inferPreference     bool
	inferPreferenceFrom string

	containerdiskVolumes []string
	pvcVolumes           []string
	volumeImport         []string
	sysprepVolume        string

	user         string
	passwordFile string
	sshKeys      []string
	gaManageSSH  bool
	accessCreds  []string

	cloudInit            string
	cloudInitUserData    string
	cloudInitNetworkData string

	// Deprecated fields
	dataSourceVolumes []string
	clonePvcVolumes   []string
	blankVolumes      []string

	namespace                     string
	explicitInstancetypeInference bool
	explicitPreferenceInference   bool
	memoryChanged                 bool

	cmd        *cobra.Command
	bootOrders map[uint]string
}

// Unless the boot order is specified by the user volumes have the following fixed boot order:
// Containerdisk > PVC > DataSource > Clone PVC > Blank > Imported volumes
// This is controlled by the order in which flags are processed.
// Also note that flags can only change values of other flags that are processed afterward.
// For example, the AccessCred flag can change the values of cloud-init-related flags,
// as these are processed after the AccessCred flag.
var flags = []string{
	RunStrategyFlag,
	InstancetypeFlag,
	PreferenceFlag,
	ContainerdiskVolumeFlag,
	PvcVolumeFlag,
	DataSourceVolumeFlag,
	ClonePvcVolumeFlag,
	BlankVolumeFlag,
	VolumeImportFlag,
	SysprepVolumeFlag,
	AccessCredFlag,
}

var volumeImportOptions = map[string]func(string) (*cdiv1.DataVolumeSpec, *uint, error){
	blank:    withVolumeSourceBlank,
	gcs:      withVolumeSourceGcs,
	http:     withVolumeSourceHTTP,
	imageIO:  withVolumeSourceImageIO,
	pvc:      withVolumeSourcePVC,
	registry: withVolumeSourceRegistry,
	s3:       withVolumeSourceS3,
	vddk:     withVolumeSourceVDDK,
	snapshot: withVolumeSourceSnapshot,
	ds:       withVolumeSourceRefDataSource,
}

var volumeImportSizeOptional = map[string]bool{
	pvc:      true,
	snapshot: true,
	ds:       true,
}

//nolint:funlen
func NewCommand() *cobra.Command {
	c := defaultCreateVM()
	cmd := &cobra.Command{
		Use:   "vm",
		Short: "Create a VirtualMachine manifest.",
		Long: "Create a VirtualMachine manifest.\n\n" +
			"If no boot order was specified volumes have the following fixed boot order:\n" +
			"Containerdisk > PVC > DataSource > Clone PVC > Blank > Imported volumes",
		Args:    cobra.NoArgs,
		Example: c.usage(),
		RunE:    c.run,
	}

	cmd.Flags().StringVar(&c.name, NameFlag, c.name, "Specify the name of the VM.")
	cmd.Flags().StringVar(&c.runStrategy, RunStrategyFlag, c.runStrategy, "Specify the RunStrategy of the VM.")
	cmd.Flags().Int64Var(&c.terminationGracePeriod, TerminationGracePeriodFlag, c.terminationGracePeriod,
		"Specify the termination grace period of the VM.")

	cmd.Flags().StringVar(&c.memory, MemoryFlag, c.memory,
		"Specify the memory of the VM.")
	cmd.Flags().StringVar(&c.instancetype, InstancetypeFlag, c.instancetype,
		"Specify the Instance Type of the VM. Mutually exclusive with instancetype inference flags.")
	cmd.Flags().BoolVar(&c.inferInstancetype, InferInstancetypeFlag, c.inferInstancetype,
		"Specify if the Instance Type of the VM should be inferred from the first boot disk. Mutually exclusive with --infer-instancetype-from.")
	cmd.Flags().StringVar(&c.inferInstancetypeFrom, InferInstancetypeFromFlag, c.inferInstancetypeFrom,
		"Specify the volume to infer the Instance Type of the VM from. Mutually exclusive with --infer-instancetype.")
	cmd.MarkFlagsMutuallyExclusive(MemoryFlag, InstancetypeFlag, InferInstancetypeFlag, InferInstancetypeFromFlag)

	cmd.Flags().StringVar(&c.preference, PreferenceFlag, c.preference,
		"Specify the Preference of the VM. Mutually exclusive with preference inference flags.")
	cmd.Flags().BoolVar(&c.inferPreference, InferPreferenceFlag, c.inferPreference,
		"Specify if the Preference of the VM should be inferred from the first boot disk. Mutually exclusive with --infer-preference-from.")
	cmd.Flags().StringVar(&c.inferPreferenceFrom, InferPreferenceFromFlag, c.inferPreferenceFrom,
		"Specify the volume to infer the Preference of the VM from. Mutually exclusive with --infer-preference.")
	cmd.MarkFlagsMutuallyExclusive(PreferenceFlag, InferPreferenceFlag, InferPreferenceFromFlag)

	cmd.Flags().StringArrayVar(&c.containerdiskVolumes, ContainerdiskVolumeFlag, c.containerdiskVolumes,
		fmt.Sprintf("Specify a containerdisk to be used by the VM. Can be provided multiple times.\n"+
			"Supported parameters: %s", params.Supported(volumeSource{})))
	cmd.Flags().StringArrayVar(&c.pvcVolumes, PvcVolumeFlag, c.pvcVolumes,
		fmt.Sprintf("Specify a PVCs to be used by the VM. Can be provided multiple times.\n"+
			"Supported parameters: %s", params.Supported(volumeSource{})))

	cmd.Flags().StringArrayVar(
		&c.volumeImport, VolumeImportFlag, c.volumeImport,
		fmt.Sprintf(
			"Specify the source for DataVolume. Can be provided multiple times.\n"+
				"Supported parameters:\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s\n"+
				"  type %s - %s",
			blank, params.Supported(dataVolumeSourceBlank{}),
			gcs, params.Supported(dataVolumeSourceGcs{}),
			http, params.Supported(dataVolumeSourceHTTP{}),
			imageIO, params.Supported(dataVolumeSourceImageIO{}),
			pvc, params.Supported(dataVolumeSource{}),
			registry, params.Supported(dataVolumeSourceRegistry{}),
			s3, params.Supported(dataVolumeSourceS3{}),
			vddk, params.Supported(dataVolumeSourceVDDK{}),
			snapshot, params.Supported(dataVolumeSource{}),
			ds, params.Supported(dataVolumeSource{}),
		))

	cmd.Flags().StringVar(&c.sysprepVolume, SysprepVolumeFlag, c.sysprepVolume,
		fmt.Sprintf("Specify a ConfigMap or Secret to be used as sysprep volume by the VM.\n"+
			"Supported parameters: %s", params.Supported(sysprepVolumeSource{})))

	cmd.Flags().StringVar(&c.user, UserFlag, c.user, "Specify the user in the cloud-init user data that is added to the VM.")
	cmd.Flags().StringVar(&c.passwordFile, PasswordFileFlag, c.passwordFile,
		"Specify a file to read the password from for the cloud-init user data that is added to the VM.")
	cmd.Flags().StringSliceVar(&c.sshKeys, SSHKeyFlag, c.sshKeys,
		"Specify one or more SSH authorized keys in the cloud-init user data that is added to the VM.")
	cmd.Flags().BoolVar(&c.gaManageSSH, GAManageSSHFlag, c.gaManageSSH,
		"Specify if the qemu-guest-agent should be able to manage SSH in the cloud-init user data that is added to the VM.\n"+
			"This is useful in combination with the \"credentials add-ssh-key\" command or when using the --access-cred flag.")
	cmd.Flags().StringArrayVar(&c.accessCreds, AccessCredFlag, c.accessCreds,
		fmt.Sprintf("Specify an access credential to be injected into the VM. Can be provided multiple times.\n"+
			"Supported parameters: %s", params.Supported(accessCredential{})))

	cmd.Flags().StringVar(&c.cloudInit, CloudInitFlag, c.cloudInit,
		fmt.Sprintf("Specify the type of the generated cloud-init data source.\n"+
			"Supported values: %s, %s, %s", cloudInitNoCloud, cloudInitConfigDrive, cloudInitNone))
	cmd.Flags().StringVar(&c.cloudInitUserData, CloudInitUserDataFlag, c.cloudInitUserData,
		"Specify the base64 encoded cloud-init user data of the VM.")
	cmd.Flags().StringVar(&c.cloudInitNetworkData, CloudInitNetworkDataFlag, c.cloudInitNetworkData,
		"Specify the base64 encoded cloud-init network data of the VM.")
	cmd.MarkFlagsMutuallyExclusive(CloudInitUserDataFlag, UserFlag)
	cmd.MarkFlagsMutuallyExclusive(CloudInitUserDataFlag, PasswordFileFlag)
	cmd.MarkFlagsMutuallyExclusive(CloudInitUserDataFlag, SSHKeyFlag)
	cmd.MarkFlagsMutuallyExclusive(CloudInitUserDataFlag, GAManageSSHFlag)

	// Deprecated flags
	cmd.Flags().StringArrayVar(&c.dataSourceVolumes, DataSourceVolumeFlag, c.dataSourceVolumes,
		"Specify a DataSource to be cloned by the VM. Can be provided multiple times.\n"+
			"Supported parameters: name:string,src:string,bootorder:uint,size:resource.Quantity\n"+
			"DEPRECATED: Use --volume-import with type:ds and same params instead.")
	cmd.Flags().StringArrayVar(&c.clonePvcVolumes, ClonePvcVolumeFlag, c.clonePvcVolumes,
		"Specify a PVC to be cloned by the VM. Can be provided multiple times.\n"+
			"Supported parameters: name:string,src:string,bootorder:uint,size:resource.Quantity\n"+
			"DEPRECATED: Use --volume-import with type:pvc and same params instead.")
	cmd.Flags().StringArrayVar(&c.blankVolumes, BlankVolumeFlag, c.blankVolumes,
		"Specify a blank volume to be used by the VM. Can be provided multiple times.\n"+
			"Supported parameters: name:string,size:resource.Quantity\n"+
			"DEPRECATED: Use --volume-import with type:blank and same params instead.")

	cmd.Flags().SortFlags = false

	return cmd
}

func defaultCreateVM() createVM {
	return createVM{
		runStrategy:            string(v1.RunStrategyAlways),
		terminationGracePeriod: 180,
		memory:                 "512Mi",
		inferInstancetype:      true,
		inferPreference:        true,
		cloudInit:              cloudInitNoCloud,
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

	return nil, fmt.Errorf("there is no volume with name \"%s\"", name)
}

func volumeShouldNotExist(flag string, vm *v1.VirtualMachine, name string) error {
	if vol := volumeExists(vm, name); vol != nil {
		return params.FlagErr(flag, volumeExistsErrorFmt, name)
	}

	return nil
}

func volumeValidToInferFrom(vm *v1.VirtualMachine, vol *v1.Volume) error {
	if vol.DataVolume != nil {
		return dataVolumeValidToInferFrom(vm, vol.DataVolume.Name)
	}

	if vol.PersistentVolumeClaim != nil {
		return nil
	}

	return fmt.Errorf("inference of instancetype or preference works only with datasources, datavolumes or pvcs")
}

func dataVolumeValidToInferFrom(vm *v1.VirtualMachine, name string) error {
	for _, dvt := range vm.Spec.DataVolumeTemplates {
		if dvt.Name == name {
			if dvt.Spec.Source != nil && (dvt.Spec.Source.PVC != nil || dvt.Spec.Source.Registry != nil || dvt.Spec.Source.Snapshot != nil) {
				return nil
			}
			if dvt.Spec.SourceRef != nil && dvt.Spec.SourceRef.Kind == "DataSource" {
				return nil
			}
			return fmt.Errorf(
				"this datavolume is not valid to infer an instancetype or preference from " +
					"(source needs to be PVC, Registry or Snapshot, sourceRef needs to be DataSource)")
		}
	}

	return nil
}

func isCloudInitSourceEmpty(src *v1.VolumeSource) bool {
	return (src.CloudInitNoCloud != nil &&
		src.CloudInitNoCloud.UserData == cloudInitConfigHeader &&
		src.CloudInitNoCloud.NetworkDataBase64 == "") ||
		(src.CloudInitConfigDrive != nil &&
			src.CloudInitConfigDrive.UserData == cloudInitConfigHeader &&
			src.CloudInitConfigDrive.NetworkDataBase64 == "")
}

func (c *createVM) run(cmd *cobra.Command, _ []string) error {
	if err := c.setDefaults(cmd); err != nil {
		return err
	}

	vm, err := c.newVM()
	if err != nil {
		return err
	}

	for _, flag := range flags {
		if cmd.Flags().Changed(flag) {
			if flagErr := c.optFns()[flag](vm); flagErr != nil {
				return flagErr
			}
		}
	}

	if cloudInitErr := c.cloudInitConfig(vm); cloudInitErr != nil {
		return cloudInitErr
	}

	if inferErr := c.inferFromVolume(vm); inferErr != nil {
		return inferErr
	}

	out, err := yaml.Marshal(vm)
	if err != nil {
		return err
	}

	cmd.Print(string(out))

	return nil
}

func (c *createVM) setDefaults(cmd *cobra.Command) error {
	c.cmd = cmd

	_, namespace, overridden, err := clientconfig.ClientAndNamespaceFromContext(cmd.Context())
	if err != nil {
		return err
	}
	if overridden {
		c.namespace = namespace
	}

	if c.name == "" {
		c.name = "vm-" + rand.String(randSuffixLength)
	}

	c.explicitInstancetypeInference = cmd.Flags().Changed(InferInstancetypeFlag) ||
		cmd.Flags().Changed(InferInstancetypeFromFlag)

	c.explicitPreferenceInference = cmd.Flags().Changed(InferPreferenceFlag) ||
		cmd.Flags().Changed(InferPreferenceFromFlag)

	c.memoryChanged = cmd.Flags().Changed(MemoryFlag)

	return nil
}

func (c *createVM) optFns() map[string]func(*v1.VirtualMachine) error {
	return map[string]func(*v1.VirtualMachine) error{
		RunStrategyFlag:         c.withRunStrategy,
		InstancetypeFlag:        c.withInstancetype,
		PreferenceFlag:          c.withPreference,
		ContainerdiskVolumeFlag: c.withContainerdiskVolume,
		DataSourceVolumeFlag:    c.withDataSourceVolume,
		ClonePvcVolumeFlag:      c.withClonePvcVolume,
		PvcVolumeFlag:           c.withPvcVolume,
		BlankVolumeFlag:         c.withBlankVolume,
		VolumeImportFlag:        c.withImportedVolume,
		SysprepVolumeFlag:       c.withSysprepVolume,
		AccessCredFlag:          c.withAccessCredential,
	}
}

//nolint:lll
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
  {{ProgramName}} create vm --volume-import=type:ds,src:my-ns/my-ds,size:50Gi

  # Create a manifest for a VirtualMachine with a cloned DataSource and inferred instancetype and preference
  {{ProgramName}} create vm --volume-import=type:ds,src:my-annotated-ds --infer-instancetype --infer-preference

  # Create a manifest for a VirtualMachine with multiple volumes and specified boot order
  {{ProgramName}} create vm --volume-containerdisk=src:my.registry/my-image:my-tag --volume-import=type:ds,src:my-ds,bootorder:1

  # Create a manifest for a VirtualMachine with multiple volumes and inferred instancetype and preference with specified volumes
  {{ProgramName}} create vm --volume-import=type:ds,src:my-annotated-ds --volume-pvc=my-annotated-pvc --infer-instancetype=my-annotated-ds --infer-preference=my-annotated-pvc

  # Create a manifest for a VirtualMachine with a cloned PVC
  {{ProgramName}} create vm --volume-import=type:pvc,src:my-ns/my-pvc

  # Create a manifest for a VirtualMachine using a PVC without cloning it
  {{ProgramName}} create vm --volume-pvc=src:my-pvc

  # Create a manifest for a VirtualMachine with a clone DataSource and a blank volume
  {{ProgramName}} create vm --volume-import=type:ds,src:my-ns/my-ds --volume-import=type:blank,size:50Gi

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and cloned DataSource
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-import=type:ds,src:my-ds

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and two cloned DataSources (flag can be provided multiple times)
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-import=type:ds,src:my-ds1 --volume-import=type:ds,src:my-ds2

  # Create a manifest for a VirtualMachine with a specified VirtualMachineCluster{Instancetype,Preference} and directly used PVC
  {{ProgramName}} create vm --instancetype=my-instancetype --preference=my-preference --volume-pvc=my-pvc

  # Create a manifest for a VirtualMachine with a specified DataVolumeTemplate
  {{ProgramName}} create vm --volume-import type:pvc,name:my-pvc,namespace:default,size:256Mi

  # Create a manifest for a VirtualMachine with a generated cloud-init config setting the user and adding an ssh authorized key
  {{ProgramName}} create vm --user cloud-user --ssh-key="ssh-ed25519 AAAA...."

  # Create a manifest for a VirtualMachine with a generated cloud-init config setting the user and setting the password from a file
  {{ProgramName}} create vm --user cloud-user --password-file=/path/to/file
	
  # Create a manifest for a VirtualMachine with SSH public keys injected into the VM from a secret called my-keys to the user also specified in the cloud-init config
  {{ProgramName}} create vm --user cloud-user --access-cred=type:ssh,src:my-keys

  # Create a manifest for a VirtualMachine with SSH public keys injected into the VM from a secret called my-keys to a user specified as param
  {{ProgramName}} create vm --access-cred=type:ssh,src:my-keys,user:myuser

  # Create a manifest for a VirtualMachine with password injected into the VM from a secret called my-pws
  {{ProgramName}} create vm --access-cred=type:password,src:my-pws

  # Create a manifest for a VirtualMachine with a Containerdisk and a Sysprep volume (source ConfigMap needs to exist)
  {{ProgramName}} create vm --memory=1Gi --volume-containerdisk=src:my.registry/my-image:my-tag --volume-sysprep=src:my-cm`
}

func (c *createVM) newVM() (*v1.VirtualMachine, error) {
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
			RunStrategy: pointer.P(v1.VirtualMachineRunStrategy(c.runStrategy)),
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
	if bootOrder == nil {
		return nil
	}

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

func (c *createVM) cloudInitFlagsChanged() bool {
	return c.cmd.Flags().Changed(UserFlag) ||
		c.cmd.Flags().Changed(PasswordFileFlag) ||
		c.cmd.Flags().Changed(SSHKeyFlag) ||
		c.cmd.Flags().Changed(GAManageSSHFlag) ||
		c.cmd.Flags().Changed(CloudInitFlag) ||
		c.cmd.Flags().Changed(CloudInitUserDataFlag) ||
		c.cmd.Flags().Changed(CloudInitNetworkDataFlag)
}

func (c *createVM) cloudInitConfig(vm *v1.VirtualMachine) error {
	if !c.cloudInitFlagsChanged() {
		return nil
	}

	if c.cmd.Flags().Changed(PasswordFileFlag) {
		c.cmd.PrintErrf("WARNING: --%s: The password is stored in cleartext in the VM definition!\n", PasswordFileFlag)
	}

	// Make sure cloudInitDisk does not already exist
	if vol := volumeExists(vm, cloudInitDisk); vol != nil {
		return fmt.Errorf(volumeExistsErrorFmt, cloudInitDisk)
	}

	var (
		src *v1.VolumeSource
		err error
	)
	switch strings.ToLower(c.cloudInit) {
	case cloudInitNoCloud:
		src, err = c.noCloudVolumeSource()
	case cloudInitConfigDrive:
		src, err = c.configDriveVolumeSource()
	case cloudInitNone:
		if c.cmd.Flags().Changed(PasswordFileFlag) ||
			c.cmd.Flags().Changed(SSHKeyFlag) ||
			c.cmd.Flags().Changed(GAManageSSHFlag) ||
			c.cmd.Flags().Changed(CloudInitUserDataFlag) ||
			c.cmd.Flags().Changed(CloudInitNetworkDataFlag) {
			c.cmd.PrintErrf("WARNING: --%s: was set to none, not creating a data source although other cloud-init options were set", CloudInitFlag)
		}
		return nil
	default:
		return params.FlagErr(CloudInitFlag,
			"invalid cloud-init data source type \"%s\", supported values are: %s, %s, %s",
			c.cloudInit, cloudInitNoCloud, cloudInitConfigDrive, cloudInitNone)
	}
	if err != nil {
		return err
	}

	// Do not add cloud-init config if it was not requested and only the header is present
	if !c.cmd.Flags().Changed(CloudInitFlag) && isCloudInitSourceEmpty(src) {
		return nil
	}

	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
		Name:         cloudInitDisk,
		VolumeSource: *src,
	})

	return nil
}

func (c *createVM) noCloudVolumeSource() (*v1.VolumeSource, error) {
	src := &v1.CloudInitNoCloudSource{}

	if c.cloudInitNetworkData != "" {
		src.NetworkDataBase64 = c.cloudInitNetworkData
	}

	if c.cloudInitUserData != "" {
		src.UserDataBase64 = c.cloudInitUserData
	} else {
		config, err := c.buildCloudInitConfig()
		if err != nil {
			return nil, err
		}
		src.UserData = config
	}
	return &v1.VolumeSource{
		CloudInitNoCloud: src,
	}, nil
}

func (c *createVM) configDriveVolumeSource() (*v1.VolumeSource, error) {
	src := &v1.CloudInitConfigDriveSource{}

	if c.cloudInitNetworkData != "" {
		src.NetworkDataBase64 = c.cloudInitNetworkData
	}

	if c.cloudInitUserData != "" {
		src.UserDataBase64 = c.cloudInitUserData
	} else {
		config, err := c.buildCloudInitConfig()
		if err != nil {
			return nil, err
		}
		src.UserData = config
	}
	return &v1.VolumeSource{
		CloudInitConfigDrive: src,
	}, nil
}

func (c *createVM) buildCloudInitConfig() (string, error) {
	config := cloudInitConfigHeader

	if c.user != "" {
		config += "\nuser: " + c.user
	}

	if c.passwordFile != "" {
		data, err := os.ReadFile(c.passwordFile)
		if err != nil {
			return "", params.FlagErr(PasswordFileFlag, "%w", err)
		}
		if password := strings.TrimSpace(string(data)); password != "" {
			config += fmt.Sprintf("\npassword: %s\nchpasswd: { expire: False }", password)
		}
	}

	if len(c.sshKeys) > 0 {
		config += "\nssh_authorized_keys:"
		for _, key := range c.sshKeys {
			config += "\n  - " + key
		}
	}

	if c.gaManageSSH {
		config += "\nruncmd:\n  - [ setsebool, -P, 'virt_qemu_ga_manage_ssh', 'on' ]"
	}

	return config, nil
}

func (c *createVM) withRunStrategy(vm *v1.VirtualMachine) error {
	runStrategies := []string{
		string(v1.RunStrategyAlways),
		string(v1.RunStrategyManual),
		string(v1.RunStrategyHalted),
		string(v1.RunStrategyOnce),
		string(v1.RunStrategyRerunOnFailure),
	}

	for _, runStrategy := range runStrategies {
		if strings.EqualFold(runStrategy, c.runStrategy) {
			vm.Spec.RunStrategy = pointer.P(v1.VirtualMachineRunStrategy(runStrategy))
			return nil
		}
	}

	return params.FlagErr(RunStrategyFlag, "invalid run strategy \"%s\", supported values are: %s",
		c.runStrategy, strings.Join(runStrategies, ", "))
}

func (c *createVM) withInstancetype(vm *v1.VirtualMachine) error {
	kind, name, err := params.SplitPrefixedName(c.instancetype)
	if err != nil {
		return params.FlagErr(InstancetypeFlag, "%w", err)
	}

	kind = strings.ToLower(kind)
	if kind != "" && kind != instancetype.SingularResourceName && kind != instancetype.ClusterSingularResourceName {
		return params.FlagErr(InstancetypeFlag, "invalid instancetype kind \"%s\", supported values are: %s, %s",
			kind, instancetype.SingularResourceName, instancetype.ClusterSingularResourceName)
	}

	// If kind is empty we rely on the vm-mutator to fill in the default value VirtualMachineClusterInstancetype
	vm.Spec.Template.Spec.Domain.Memory = nil
	vm.Spec.Instancetype = &v1.InstancetypeMatcher{
		Name: name,
		Kind: kind,
	}

	return nil
}

func (c *createVM) withPreference(vm *v1.VirtualMachine) error {
	kind, name, err := params.SplitPrefixedName(c.preference)
	if err != nil {
		return params.FlagErr(PreferenceFlag, "%w", err)
	}

	kind = strings.ToLower(kind)
	if kind != "" && kind != instancetype.SingularPreferenceResourceName && kind != instancetype.ClusterSingularPreferenceResourceName {
		return params.FlagErr(PreferenceFlag, "invalid preference kind \"%s\", supported values are: %s, %s",
			kind, instancetype.SingularPreferenceResourceName, instancetype.ClusterSingularPreferenceResourceName)
	}

	// If kind is empty we rely on the vm-mutator to fill in the default value VirtualMachineClusterPreference
	vm.Spec.Preference = &v1.PreferenceMatcher{
		Name: name,
		Kind: kind,
	}

	return nil
}

func (c *createVM) withContainerdiskVolume(vm *v1.VirtualMachine) error {
	for i, containerdiskVol := range c.containerdiskVolumes {
		src := volumeSource{}
		if err := params.Map(ContainerdiskVolumeFlag, containerdiskVol, &src); err != nil {
			return err
		}

		if src.Source == "" {
			return params.FlagErr(ContainerdiskVolumeFlag, "src must be specified")
		}

		if src.Name == "" {
			src.Name = fmt.Sprintf("%s-containerdisk-%d", vm.Name, i)
		}

		if errs := validation.IsDNS1123Label(src.Name); len(errs) > 0 {
			return params.FlagErr(ContainerdiskVolumeFlag, "invalid name \"%s\": %s", src.Name, strings.Join(errs, ","))
		}

		if err := volumeShouldNotExist(ContainerdiskVolumeFlag, vm, src.Name); err != nil {
			return err
		}

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: src.Name,
			VolumeSource: v1.VolumeSource{
				ContainerDisk: &v1.ContainerDiskSource{
					Image: src.Source,
				},
			},
		})

		if err := c.addDiskWithBootOrder(ContainerdiskVolumeFlag, vm, src.Name, src.BootOrder); err != nil {
			return err
		}
	}

	return nil
}

func (c *createVM) withPvcVolume(vm *v1.VirtualMachine) error {
	for _, pvcVol := range c.pvcVolumes {
		src := volumeSource{}
		if err := params.Map(PvcVolumeFlag, pvcVol, &src); err != nil {
			return err
		}

		if src.Source == "" {
			return params.FlagErr(PvcVolumeFlag, "src must be specified")
		}

		namespace, name, err := params.SplitPrefixedName(src.Source)
		if err != nil {
			return params.FlagErr(PvcVolumeFlag, "src invalid: %w", err)
		}

		if namespace != "" {
			return params.FlagErr(PvcVolumeFlag, "not allowed to specify namespace of pvc \"%s\"", name)
		}

		if src.Name == "" {
			src.Name = name
		}

		if errs := validation.IsDNS1123Label(src.Name); len(errs) > 0 {
			return params.FlagErr(PvcVolumeFlag, "invalid name \"%s\": %s", src.Name, strings.Join(errs, ","))
		}

		if err := volumeShouldNotExist(PvcVolumeFlag, vm, src.Name); err != nil {
			return err
		}

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
			Name: src.Name,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: name,
					},
				},
			},
		})

		if err := c.addDiskWithBootOrder(PvcVolumeFlag, vm, src.Name, src.BootOrder); err != nil {
			return err
		}
	}

	return nil
}

func (c *createVM) withSysprepVolume(vm *v1.VirtualMachine) error {
	src := sysprepVolumeSource{}
	if err := params.Map(SysprepVolumeFlag, c.sysprepVolume, &src); err != nil {
		return err
	}

	if src.Source == "" {
		return params.FlagErr(SysprepVolumeFlag, "src must be specified")
	}

	namespace, name, err := params.SplitPrefixedName(src.Source)
	if err != nil {
		return params.FlagErr(SysprepVolumeFlag, "src invalid: %w", err)
	}

	if namespace != "" {
		return params.FlagErr(SysprepVolumeFlag, "not allowed to specify namespace of configmap or secret \"%s\"", name)
	}

	if src.Type == "" {
		src.Type = sysprepConfigMap
	}

	if err := volumeShouldNotExist(SysprepVolumeFlag, vm, sysprepDisk); err != nil {
		return err
	}

	var sysprep *v1.SysprepSource
	switch strings.ToLower(src.Type) {
	case sysprepConfigMap:
		sysprep = &v1.SysprepSource{
			ConfigMap: &k8sv1.LocalObjectReference{
				Name: src.Source,
			},
		}
	case sysprepSecret:
		sysprep = &v1.SysprepSource{
			Secret: &k8sv1.LocalObjectReference{
				Name: src.Source,
			},
		}
	default:
		return params.FlagErr(SysprepVolumeFlag,
			"invalid sysprep source type \"%s\", supported values are: %s, %s",
			src.Type, sysprepConfigMap, sysprepSecret)
	}

	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, v1.Volume{
		Name: sysprepDisk,
		VolumeSource: v1.VolumeSource{
			Sysprep: sysprep,
		},
	})

	return nil
}

func (c *createVM) withImportedVolume(vm *v1.VirtualMachine) error {
	for _, volume := range c.volumeImport {
		srcType, err := params.GetParamByName("type", volume)
		if err != nil {
			return params.FlagErr(VolumeImportFlag, "%w", err)
		}

		srcFn, found := volumeImportOptions[srcType]
		if !found {
			return params.FlagErr(VolumeImportFlag, "invalid volume import type \"%s\", see help for supported values", srcType)
		}

		spec, bootOrder, err := srcFn(volume)
		if err != nil {
			return err
		}

		size, err := params.GetParamByName("size", volume)
		if err != nil &&
			(!volumeImportSizeOptional[srcType] || !errors.Is(err, params.NotFoundError{Name: "size"})) {
			return params.FlagErr(VolumeImportFlag, "%w", err)
		}

		name, err := params.GetParamByName("name", volume)
		if err != nil {
			name = "imported-volume-" + rand.String(randSuffixLength)
		}

		if err := createDataVolume(spec, size, name, vm); err != nil {
			return err
		}

		if err := c.addDiskWithBootOrder(VolumeImportFlag, vm, name, bootOrder); err != nil {
			return err
		}
	}
	return nil
}

func withVolumeSourceBlank(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceBlank{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	spec := &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			Blank: &cdiv1.DataVolumeBlankImage{},
		},
	}

	return spec, src.BootOrder, nil
}

func withVolumeSourceGcs(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceGcs{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.URL == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "url is required with gcs volume source")
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			GCS: &cdiv1.DataVolumeSourceGCS{
				URL:       src.URL,
				SecretRef: src.SecretRef,
			},
		},
	}, src.BootOrder, nil
}

func withVolumeSourceHTTP(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceHTTP{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.URL == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "url is required with http volume source")
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			HTTP: &cdiv1.DataVolumeSourceHTTP{
				URL:                src.URL,
				SecretRef:          src.SecretRef,
				CertConfigMap:      src.CertConfigMap,
				ExtraHeaders:       src.ExtraHeaders,
				SecretExtraHeaders: src.SecretExtraHeaders,
			},
		},
	}, src.BootOrder, nil
}

func withVolumeSourceImageIO(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceImageIO{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.URL == "" || src.DiskID == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "url and diskid are both required with imageio volume source")
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			Imageio: &cdiv1.DataVolumeSourceImageIO{
				URL:           src.URL,
				DiskID:        src.DiskID,
				SecretRef:     src.SecretRef,
				CertConfigMap: src.CertConfigMap,
			},
		},
	}, src.BootOrder, nil
}

func parseVolumeSource(paramStr, kind string) (name, namespace string, bootOrder *uint, err error) {
	src := dataVolumeSource{}

	err = params.Map(VolumeImportFlag, paramStr, &src)
	if err != nil {
		return "", "", nil, err
	}

	if src.Source == "" {
		return "", "", nil, params.FlagErr(VolumeImportFlag, "src must be specified")
	}

	namespace, name, err = params.SplitPrefixedName(src.Source)
	if err != nil {
		return "", "", nil, params.FlagErr(VolumeImportFlag, "src invalid: %w", err)
	}

	if namespace == "" {
		return "", "", nil, params.FlagErr(VolumeImportFlag, "namespace of %s \"%s\" must be specified", kind, name)
	}

	return name, namespace, src.BootOrder, nil
}

func withVolumeSourcePVC(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	name, namespace, bootOrder, err := parseVolumeSource(paramStr, "pvc")
	if err != nil {
		return nil, nil, err
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			PVC: &cdiv1.DataVolumeSourcePVC{
				Name:      name,
				Namespace: namespace,
			},
		},
	}, bootOrder, nil
}

func withVolumeSourceRegistry(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceRegistry{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.PullMethod != "" &&
		(src.PullMethod != string(cdiv1.RegistryPullPod) && src.PullMethod != string(cdiv1.RegistryPullNode)) {
		return nil, nil, params.FlagErr(VolumeImportFlag, "pullmethod must be set to pod or node")
	}

	if (src.URL == "" && src.ImageStream == "") ||
		(src.URL != "" && src.ImageStream != "") {
		return nil, nil, params.FlagErr(VolumeImportFlag, "exactly one of url or imagestream must be defined")
	}

	spec := &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			Registry: &cdiv1.DataVolumeSourceRegistry{},
		},
	}

	if src.PullMethod != "" {
		spec.Source.Registry.PullMethod = (*cdiv1.RegistryPullMethod)(&src.PullMethod)
	}

	if src.CertConfigMap != "" {
		spec.Source.Registry.CertConfigMap = &src.CertConfigMap
	}

	if src.ImageStream != "" {
		spec.Source.Registry.ImageStream = &src.ImageStream
	}

	if src.URL != "" {
		spec.Source.Registry.URL = &src.URL
	}

	if src.SecretRef != "" {
		spec.Source.Registry.SecretRef = &src.SecretRef
	}

	return spec, src.BootOrder, nil
}

func withVolumeSourceS3(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceS3{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.URL == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "url is required with s3 volume source")
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			S3: &cdiv1.DataVolumeSourceS3{
				URL:           src.URL,
				CertConfigMap: src.CertConfigMap,
				SecretRef:     src.SecretRef,
			},
		},
	}, src.BootOrder, nil
}

func withVolumeSourceVDDK(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSourceVDDK{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.URL == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "url is required with vddk volume source")
	}

	if src.UUID == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "uuid is required with vddk volume source")
	}

	if src.ThumbPrint == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "thumbprint is required with vddk volume source")
	}

	if src.SecretRef == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "secretref is required with vddk volume source")
	}

	if src.BackingFile == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "backingfile is required with vddk volume source")
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			VDDK: &cdiv1.DataVolumeSourceVDDK{
				URL:          src.URL,
				UUID:         src.UUID,
				Thumbprint:   src.ThumbPrint,
				SecretRef:    src.SecretRef,
				InitImageURL: src.InitImageURL,
				BackingFile:  src.BackingFile,
			},
		},
	}, src.BootOrder, nil
}

func withVolumeSourceSnapshot(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	name, namespace, bootOrder, err := parseVolumeSource(paramStr, "snapshot")
	if err != nil {
		return nil, nil, err
	}

	return &cdiv1.DataVolumeSpec{
		Source: &cdiv1.DataVolumeSource{
			Snapshot: &cdiv1.DataVolumeSourceSnapshot{
				Name:      name,
				Namespace: namespace,
			},
		},
	}, bootOrder, nil
}

func withVolumeSourceRefDataSource(paramStr string) (*cdiv1.DataVolumeSpec, *uint, error) {
	src := dataVolumeSource{}
	if err := params.Map(VolumeImportFlag, paramStr, &src); err != nil {
		return nil, nil, err
	}

	if src.Source == "" {
		return nil, nil, params.FlagErr(VolumeImportFlag, "src must be specified")
	}

	namespace, name, err := params.SplitPrefixedName(src.Source)
	if err != nil {
		return nil, nil, params.FlagErr(VolumeImportFlag, "src invalid: %w", err)
	}

	spec := &cdiv1.DataVolumeSpec{
		SourceRef: &cdiv1.DataVolumeSourceRef{
			Kind: "DataSource",
			Name: name,
		},
	}
	if namespace != "" {
		spec.SourceRef.Namespace = &namespace
	}

	return spec, src.BootOrder, nil
}

func createDataVolume(spec *cdiv1.DataVolumeSpec, size, name string, vm *v1.VirtualMachine) error {
	if errs := validation.IsDNS1123Label(name); len(errs) > 0 {
		return params.FlagErr(VolumeImportFlag, "invalid name \"%s\": %s", name, strings.Join(errs, ","))
	}

	if err := volumeShouldNotExist(VolumeImportFlag, vm, name); err != nil {
		return err
	}

	dvt := v1.DataVolumeTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: *spec,
	}

	dvt.Spec.Storage = &cdiv1.StorageSpec{}
	if size != "" {
		dvt.Spec.Storage.Resources = k8sv1.VolumeResourceRequirements{
			Requests: k8sv1.ResourceList{
				k8sv1.ResourceStorage: resource.MustParse(size),
			},
		}
	}

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, dvt)
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

func (c *createVM) withAccessCredential(vm *v1.VirtualMachine) error {
	for _, accessCred := range c.accessCreds {
		src := &accessCredential{}
		if err := params.Map(AccessCredFlag, accessCred, src); err != nil {
			return err
		}

		if src.Source == "" {
			return params.FlagErr(AccessCredFlag, "src must be specified")
		}

		namespace, name, err := params.SplitPrefixedName(src.Source)
		if err != nil {
			return params.FlagErr(AccessCredFlag, "src invalid: %w", err)
		}

		if namespace != "" {
			return params.FlagErr(AccessCredFlag, "not allowed to specify namespace of secret \"%s\"", name)
		}

		var apiAccessCred *v1.AccessCredential
		switch strings.ToLower(src.Type) {
		case accessCredTypeSSH, "":
			if apiAccessCred, err = c.withAccessCredentialSSH(src); err != nil {
				return err
			}
		case accessCredTypePassword:
			if apiAccessCred, err = c.withAccessCredentialPassword(src); err != nil {
				return err
			}
		default:
			return params.FlagErr(AccessCredFlag,
				"invalid access credential type \"%s\", supported values are: %s, %s",
				src.Type, accessCredTypeSSH, accessCredTypePassword)
		}

		vm.Spec.Template.Spec.AccessCredentials = append(vm.Spec.Template.Spec.AccessCredentials, *apiAccessCred)
	}

	return nil
}

func (c *createVM) withAccessCredentialSSH(src *accessCredential) (*v1.AccessCredential, error) {
	switch strings.ToLower(src.Method) {
	case accessCredMethodGA, "":
		return c.withAccessCredentialSSHMethodGA(src)
	case cloudInitNoCloud:
		return c.withAccessCredentialSSHMethodNoCloud(src)
	case cloudInitConfigDrive:
		return c.withAccessCredentialSSHMethodConfigDrive(src)
	default:
		return nil, params.FlagErr(AccessCredFlag,
			"invalid access credentials ssh method \"%s\", supported values are: %s, %s, %s",
			src.Method, accessCredMethodGA, cloudInitNoCloud, cloudInitConfigDrive)
	}
}

func (c *createVM) withAccessCredentialSSHMethodGA(src *accessCredential) (*v1.AccessCredential, error) {
	// Take user from --user flag, override with user provided in params
	user := c.user
	if src.User != "" {
		user = src.User
	}
	if user == "" {
		return nil, params.FlagErr(AccessCredFlag,
			"user must be specified with access credential ssh method ga (\"--user\" flag or param \"user\")")
	}

	// Set --ga-manage-ssh flag to allow the guest agent to write public keys if it was not changed
	if !c.cmd.Flags().Lookup(GAManageSSHFlag).Changed {
		if err := c.cmd.Flags().Set(GAManageSSHFlag, "true"); err != nil {
			return nil, err
		}
	}

	return createAccessCredentialSSH(src.Source, &v1.SSHPublicKeyAccessCredentialPropagationMethod{
		QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{
			Users: []string{user},
		},
	}), nil
}

func (c *createVM) withAccessCredentialSSHGeneric(
	src *accessCredential, expectedCloudInit string, method *v1.SSHPublicKeyAccessCredentialPropagationMethod,
) (*v1.AccessCredential, error) {
	if src.User != "" {
		return nil, params.FlagErr(AccessCredFlag, accessCredUserInvalidError)
	}

	if !c.cmd.Flags().Lookup(CloudInitFlag).Changed {
		if err := c.cmd.Flags().Set(CloudInitFlag, expectedCloudInit); err != nil {
			return nil, err
		}
	}

	if !strings.EqualFold(c.cloudInit, expectedCloudInit) {
		return nil, params.FlagErr(AccessCredFlag, accessCredMethodFlagMismatchErrorFmt,
			CloudInitFlag, expectedCloudInit, c.cloudInit)
	}

	return createAccessCredentialSSH(src.Source, method), nil
}

func (c *createVM) withAccessCredentialSSHMethodNoCloud(src *accessCredential) (*v1.AccessCredential, error) {
	return c.withAccessCredentialSSHGeneric(src, cloudInitNoCloud, &v1.SSHPublicKeyAccessCredentialPropagationMethod{
		NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
	})
}

func (c *createVM) withAccessCredentialSSHMethodConfigDrive(src *accessCredential) (*v1.AccessCredential, error) {
	return c.withAccessCredentialSSHGeneric(src, cloudInitConfigDrive, &v1.SSHPublicKeyAccessCredentialPropagationMethod{
		ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
	})
}

func createAccessCredentialSSH(src string, method *v1.SSHPublicKeyAccessCredentialPropagationMethod) *v1.AccessCredential {
	return &v1.AccessCredential{
		SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
			Source: v1.SSHPublicKeyAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{
					SecretName: src,
				},
			},
			PropagationMethod: *method,
		},
	}
}

func (c *createVM) withAccessCredentialPassword(src *accessCredential) (*v1.AccessCredential, error) {
	if src.Method != "" && !strings.EqualFold(src.Method, accessCredMethodGA) {
		return nil, params.FlagErr(AccessCredFlag,
			"invalid access credentials password method \"%s\", supported values are: %s",
			src.Method, accessCredMethodGA)
	}

	if src.User != "" {
		return nil, params.FlagErr(AccessCredFlag, accessCredUserInvalidError)
	}

	return &v1.AccessCredential{
		UserPassword: &v1.UserPasswordAccessCredential{
			Source: v1.UserPasswordAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{
					SecretName: src.Source,
				},
			},
			PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
				QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
			},
		},
	}, nil
}

// Deprecated optFns

func (c *createVM) withDataSourceVolume(_ *v1.VirtualMachine) error {
	return aliasToVolumeImport(c.cmd, DataSourceVolumeFlag, ds, c.dataSourceVolumes, &c.volumeImport)
}

func (c *createVM) withClonePvcVolume(_ *v1.VirtualMachine) error {
	return aliasToVolumeImport(c.cmd, ClonePvcVolumeFlag, pvc, c.clonePvcVolumes, &c.volumeImport)
}

func (c *createVM) withBlankVolume(_ *v1.VirtualMachine) error {
	return aliasToVolumeImport(c.cmd, BlankVolumeFlag, blank, c.blankVolumes, &c.volumeImport)
}

func aliasToVolumeImport(cmd *cobra.Command, flag, volType string, vols []string, volumeImport *[]string) error {
	// Print directly to os.Stderr to avoid tainting the regular output.
	// This is necessary because cobra is writing deprecation messages to the regular output.
	// Because of this cmd.Flags.MarkDeprecated() cannot be used to mark this flag as deprecated.
	// See https://github.com/spf13/cobra/issues/1708
	if _, err := fmt.Fprintf(os.Stderr, "Flag --%s has been deprecated, use flag --volume-import instead\n", flag); err != nil {
		return err
	}

	for _, vol := range vols {
		if vol == "" {
			return params.FlagErr(VolumeImportFlag, "params may not be empty")
		}
		// Prepend the volume to keep the documented boot order
		*volumeImport = append([]string{fmt.Sprintf("type:%s,%s", volType, vol)}, *volumeImport...)
	}

	cmd.Flags().Lookup(VolumeImportFlag).Changed = true

	return nil
}
