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

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"kubevirt.io/kubevirt/tools/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	validating_webhook "kubevirt.io/kubevirt/pkg/virt-api/webhooks/validating-webhook"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
)

func main() {
	flag.StringVar(&utils.DockerPrefix, "container-prefix", utils.DockerPrefix, "")
	flag.StringVar(&utils.DockerTag, "container-tag", utils.DockerTag, "")
	genDir := flag.String("generated-vms-dir", "", "")
	flag.Parse()

	// Required to validate DataVolume usage
	os.Setenv("FEATURE_GATES", "DataVolumes,LiveMigration,SRIOV")

	var vms = map[string]*v1.VirtualMachine{
		utils.VmCirros:           utils.GetVMCirros(),
		utils.VmAlpineMultiPvc:   utils.GetVMMultiPvc(),
		utils.VmAlpineDataVolume: utils.GetVMDataVolume(),
	}

	var vmis = map[string]*v1.VirtualMachineInstance{
		utils.VmiEphemeral:         utils.GetVMIEphemeral(),
		utils.VmiMigratable:        utils.GetVMIMigratable(),
		utils.VmiFlavorSmall:       utils.GetVMIFlavorSmall(),
		utils.VmiSata:              utils.GetVMISata(),
		utils.VmiFedora:            utils.GetVMIEphemeralFedora(),
		utils.VmiAlpineEFI:         utils.GetVMIAlpineEFI(),
		utils.VmiNoCloud:           utils.GetVMINoCloud(),
		utils.VmiPVC:               utils.GetVMIPvc(),
		utils.VmiBlockPVC:          utils.GetVMIBlockPvc(),
		utils.VmiWindows:           utils.GetVMIWindows(),
		utils.VmiSlirp:             utils.GetVMISlirp(),
		utils.VmiSRIOV:             utils.GetVMISRIOV(),
		utils.VmiWithHookSidecar:   utils.GetVMIWithHookSidecar(),
		utils.VmiMultusPtp:         utils.GetVMIMultusPtp(),
		utils.VmiMultusMultipleNet: utils.GetVMIMultusMultipleNet(),
		utils.VmiGeniePtp:          utils.GetVMIGeniePtp(),
		utils.VmiGenieMultipleNet:  utils.GetVMIGenieMultipleNet(),
		utils.VmiMasquerade:        utils.GetVMIMasquerade(),
		utils.VmiHostDisk:          utils.GetVMIHostDisk(),
	}

	var vmireplicasets = map[string]*v1.VirtualMachineInstanceReplicaSet{
		utils.VmiReplicaSetCirros: utils.GetVMIReplicaSetCirros(),
	}

	var vmipresets = map[string]*v1.VirtualMachineInstancePreset{
		utils.VmiPresetSmall: utils.GetVMIPresetSmall(),
	}

	var migrations = map[string]*v1.VirtualMachineInstanceMigration{
		utils.VmiMigration: utils.GetVMIMigration(),
	}

	var templates = map[string]*utils.Template{
		utils.VmTemplateFedora:  utils.GetTemplateFedora(),
		utils.VmTemplateRHEL7:   utils.GetTemplateRHEL7(),
		utils.VmTemplateWindows: utils.GetTemplateWindows(),
	}

	handleError := func(err error) {
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			panic(err)
		}
	}

	handleCauses := func(causes []metav1.StatusCause, name string, objType string) {
		if len(causes) > 0 {
			for _, cause := range causes {
				fmt.Fprintf(
					os.Stderr,
					"Failed to validate %s spec: failed to admit yaml for %s: %s at %s: %s\n",
					objType, name, cause.Type, cause.Field, cause.Message)
			}
			panic(fmt.Errorf("Failed to admit %s of type %s", name, objType))
		}
	}

	dumpObject := func(name string, obj interface{}) error {

		filename := filepath.Join(*genDir, fmt.Sprintf("%s.yaml", name))
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Failed to open file %v, %v", filename, err)
		}
		defer file.Close()

		util.MarshallObject(obj, file)

		return nil
	}

	// Having no generics is lots of fun
	for name, obj := range vms {
		causes := validating_webhook.ValidateVirtualMachineSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vm")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmis {
		causes := validating_webhook.ValidateVirtualMachineInstanceSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmireplicasets {
		causes := validating_webhook.ValidateVMIRSSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi replica set")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range vmipresets {
		causes := validating_webhook.ValidateVMIPresetSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi preset")
		handleError(dumpObject(name, *obj))
	}

	for name, obj := range migrations {
		causes := validating_webhook.ValidateVirtualMachineInstanceMigrationSpec(k8sfield.NewPath("spec"), &obj.Spec)
		handleCauses(causes, name, "vmi preset")
		handleError(dumpObject(name, *obj))
	}

	// TODO:(ihar) how to validate templates?
	for name, obj := range templates {
		handleError(dumpObject(name, *obj))
	}
}
