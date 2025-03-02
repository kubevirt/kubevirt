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
 * Copyright 2018 Red Hat, Inc.
 *
 */
//nolint:funlen
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"kubevirt.io/api/migrations/v1alpha1"

	"kubevirt.io/kubevirt/tools/util"

	admissionv1 "k8s.io/api/admission/v1"
	authenticationv1 "k8s.io/api/authentication/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"

	"kubevirt.io/kubevirt/pkg/testutils"
	validating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook/admitters"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
)

func main() {
	flag.StringVar(&utils.DockerPrefix, "container-prefix", utils.DockerPrefix, "")
	flag.StringVar(&utils.DockerTag, "container-tag", utils.DockerTag, "")
	genDir := flag.String("generated-vms-dir", "", "")
	flag.Parse()
	permit := true

	config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: []string{"DataVolumes", "LiveMigration", "SRIOV", "GPU", "HostDisk", "Macvtap", "HostDevices", "Sidecar"},
		},
		NetworkConfiguration: &v1.NetworkConfiguration{
			DeprecatedPermitSlirpInterface:    &permit,
			PermitBridgeInterfaceOnPodNetwork: &permit,
		},
		PermittedHostDevices: &v1.PermittedHostDevices{
			PciHostDevices: []v1.PciHostDevice{
				{
					PCIVendorSelector:        "10DE:1EB8",
					ResourceName:             "nvidia.com/GP102GL_Tesla_P40",
					ExternalResourceProvider: true,
				},
			},
		},
	})

	priorityClasses := map[string]*schedulingv1.PriorityClass{
		utils.Preemtible:    utils.GetPreemtible(),
		utils.NonPreemtible: utils.GetNonPreemtible(),
	}

	vms := map[string]*v1.VirtualMachine{
		utils.VMCirros:                         utils.GetVMCirros(),
		utils.VMAlpineMultiPvc:                 utils.GetVMMultiPvc(),
		utils.VMAlpineDataVolume:               utils.GetVMDataVolume(),
		utils.VMPriorityClass:                  utils.GetVMPriorityClass(),
		utils.VMCirrosSata:                     utils.GetVMCirrosSata(),
		utils.VMCirrosWithHookSidecarConfigMap: utils.GetVMCirrosWithHookSidecarConfigMap(),
	}

	vmis := map[string]*v1.VirtualMachineInstance{
		utils.VMIEphemeral:                utils.GetVMIEphemeral(),
		utils.VMIMigratable:               utils.GetVMIMigratable(),
		utils.VMISata:                     utils.GetVMISata(),
		utils.VMIFedora:                   utils.GetVMIEphemeralFedora(),
		utils.VMIFedoraIsolated:           utils.GetVMIEphemeralFedoraIsolated(),
		utils.VMISecureBoot:               utils.GetVMISecureBoot(),
		utils.VMIAlpineEFI:                utils.GetVMIAlpineEFI(),
		utils.VMINoCloud:                  utils.GetVMINoCloud(),
		utils.VMIPVC:                      utils.GetVMIPvc(),
		utils.VMIWindows:                  utils.GetVMIWindows(),
		utils.VMISRIOV:                    utils.GetVMISRIOV(),
		utils.VMIWithHookSidecar:          utils.GetVMIWithHookSidecar(),
		utils.VMIWithHookSidecarConfigMap: utils.GetVMIWithHookSidecarConfigMap(),
		utils.VMIMultusPtp:                utils.GetVMIMultusPtp(),
		utils.VMIMultusMultipleNet:        utils.GetVMIMultusMultipleNet(),
		utils.VMIMasquerade:               utils.GetVMIMasquerade(),
		utils.VMIHostDisk:                 utils.GetVMIHostDisk(),
		utils.VMIGPU:                      utils.GetVMIGPU(),
		utils.VMIKernelBoot:               utils.GetVMIKernelBoot(),
		utils.VMIUSB:                      utils.GetVMIUSB(),
	}

	vmireplicasets := map[string]*v1.VirtualMachineInstanceReplicaSet{
		utils.VMIReplicaSetCirros: utils.GetVMIReplicaSetCirros(),
	}

	vmpools := map[string]*poolv1.VirtualMachinePool{
		utils.VMPoolCirros: utils.GetVMPoolCirros(),
	}

	migrations := map[string]*v1.VirtualMachineInstanceMigration{
		utils.VMIMigration: utils.GetVMIMigration(),
	}

	migrationPolicies := map[string]*v1alpha1.MigrationPolicy{
		utils.MigrationPolicyName: utils.GetMigrationPolicy(),
	}

	handleError := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			panic(err)
		}
	}

	handleCauses := func(causes []metav1.StatusCause, name, objType string) {
		if len(causes) > 0 {
			for _, cause := range causes {
				fmt.Fprintf(
					os.Stderr,
					"Failed to validate %s spec: failed to admit yaml for %s: %s at %s: %s\n",
					objType, name, cause.Type, cause.Field, cause.Message)
			}
			panic(fmt.Errorf("failed to admit %s of type %s", name, objType))
		}
	}

	dumpObject := func(name string, obj interface{}) error {
		filename := filepath.Join(*genDir, fmt.Sprintf("%s.yaml", name))
		const permMode = 0o644
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, permMode)
		if err != nil {
			return fmt.Errorf("failed to open file %v, %v", filename, err)
		}
		defer file.Close()
		return util.MarshallObject(obj, file)
	}

	// Having no generics is lots of fun
	for name, obj := range vms {
		causes := validating_webhook.ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &obj.Spec, config, false)
		handleCauses(causes, name, "vm")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmis {
		causes := validating_webhook.ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &obj.Spec, config)
		handleCauses(causes, name, "vmi")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmireplicasets {
		causes := validating_webhook.ValidateVMIRSSpec(k8sfield.NewPath("spec"), &obj.Spec, config)
		handleCauses(causes, name, "vmi replica set")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmpools {
		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				UserInfo: authenticationv1.UserInfo{
					Username: "user-account",
				},
				Operation: admissionv1.Create,
			},
		}
		causes := validating_webhook.ValidateVMPoolSpec(ar, k8sfield.NewPath("spec"), obj, config, false)
		handleCauses(causes, name, "vm pool")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range migrations {
		causes := validating_webhook.ValidateVirtualMachineInstanceMigrationSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi migration")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range priorityClasses {
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range migrationPolicies {
		handleError(dumpObject(name, *obj))
	}
}
