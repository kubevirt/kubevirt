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

package compute

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ignition"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const QEMUSeaBiosDebugPipe = "/var/run/kubevirt-private/QEMUSeaBiosDebugPipe"

type QemuCmdDomainConfigurator struct {
	verboseLogEnabled bool
}

func NewQemuCmdDomainConfigurator(verboseLogEnabled bool) QemuCmdDomainConfigurator {
	return QemuCmdDomainConfigurator{verboseLogEnabled: verboseLogEnabled}
}

func (q QemuCmdDomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	// Add Ignition Command Line if present
	ignitiondata := vmi.Annotations[v1.IgnitionAnnotation]
	if ignitiondata != "" && strings.Contains(ignitiondata, "ignition") {
		initializeQEMUCmdAndQEMUArg(domain)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-fw_cfg"})
		ignitionpath := fmt.Sprintf("%s/%s", ignition.GetDomainBasePath(vmi.Name, vmi.Namespace), ignition.IgnitionFile)
		domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s", ignitionpath)})
	}

	if q.verboseLogEnabled {
		virtLauncherLogVerbosity, err := strconv.Atoi(os.Getenv(util.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY))
		if err == nil && virtLauncherLogVerbosity > util.EXT_LOG_VERBOSITY_THRESHOLD {
			// isa-debugcon device is only for x86_64
			initializeQEMUCmdAndQEMUArg(domain)

			domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg,
				api.Arg{Value: "-chardev"},
				api.Arg{Value: fmt.Sprintf("file,id=firmwarelog,path=%s", QEMUSeaBiosDebugPipe)},
				api.Arg{Value: "-device"},
				api.Arg{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"})
		}
	}

	return nil
}

func initializeQEMUCmdAndQEMUArg(domain *api.Domain) {
	if domain.Spec.QEMUCmd == nil {
		domain.Spec.QEMUCmd = &api.Commandline{}
	}

	if domain.Spec.QEMUCmd.QEMUArg == nil {
		domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
	}
}
