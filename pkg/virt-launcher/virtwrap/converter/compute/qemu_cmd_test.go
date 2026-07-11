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

package compute_test

import (
	"fmt"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ignition"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("QemuCmd Domain Configurator", func() {
	const (
		vmiName           = "test-vmi"
		vmiNamespace      = "test-ns"
		verboseLogEnabled = true
	)

	DescribeTable("ignition arguments", func(annotation string, expectedQEMUCmd *api.Commandline) {
		vmi := libvmi.New(
			libvmi.WithName(vmiName),
			libvmi.WithNamespace(vmiNamespace),
			libvmi.WithAnnotation(v1.IgnitionAnnotation, annotation),
		)
		var domain api.Domain

		configurator := compute.NewQemuCmdDomainConfigurator(!verboseLogEnabled)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithQEMUCmd(expectedQEMUCmd)))
	},
		Entry("added when annotation contains 'ignition'", "ignition", &api.Commandline{
			QEMUArg: []api.Arg{
				{Value: "-fw_cfg"},
				{Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s/%s",
					ignition.GetDomainBasePath(vmiName, vmiNamespace), ignition.IgnitionFile)},
			},
		}),
		Entry("not added when annotation does not contain 'ignition'", "some-other-data", nil),
		Entry("not added when annotation value is empty", "", nil),
	)

	DescribeTable("SeaBios debug pipe arguments", func(verboseLogEnabled bool, envValue string, expectedQEMUCmd *api.Commandline) {
		os.Setenv(util.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, envValue)
		DeferCleanup(os.Unsetenv, util.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY)

		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.NewQemuCmdDomainConfigurator(verboseLogEnabled)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		Expect(domain).To(Equal(newDomainWithQEMUCmd(expectedQEMUCmd)))
	},
		Entry("added when verbosity exceeds threshold",
			verboseLogEnabled,
			strconv.Itoa(util.EXT_LOG_VERBOSITY_THRESHOLD+1),
			&api.Commandline{
				QEMUArg: []api.Arg{
					{Value: "-chardev"},
					{Value: fmt.Sprintf("file,id=firmwarelog,path=%s", compute.QEMUSeaBiosDebugPipe)},
					{Value: "-device"},
					{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"},
				},
			},
		),
		Entry("not added when verbosity equals threshold",
			verboseLogEnabled,
			strconv.Itoa(util.EXT_LOG_VERBOSITY_THRESHOLD),
			nil,
		),
		Entry("not added when verbosity env is not a number",
			verboseLogEnabled,
			"",
			nil,
		),
		Entry("not added when verbose logging is disabled",
			!verboseLogEnabled,
			strconv.Itoa(util.EXT_LOG_VERBOSITY_THRESHOLD+1),
			nil,
		),
	)

	It("should configure both ignition and verbose logging arguments", func() {
		os.Setenv(util.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, strconv.Itoa(util.EXT_LOG_VERBOSITY_THRESHOLD+1))
		DeferCleanup(os.Unsetenv, util.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY)

		vmi := libvmi.New(
			libvmi.WithName(vmiName),
			libvmi.WithNamespace(vmiNamespace),
			libvmi.WithAnnotation(v1.IgnitionAnnotation, "ignition"),
		)
		var domain api.Domain

		configurator := compute.NewQemuCmdDomainConfigurator(verboseLogEnabled)
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		ignitionPath := fmt.Sprintf("%s/%s", ignition.GetDomainBasePath(vmiName, vmiNamespace), ignition.IgnitionFile)
		Expect(domain).To(Equal(newDomainWithQEMUCmd(&api.Commandline{
			QEMUArg: []api.Arg{
				{Value: "-fw_cfg"},
				{Value: fmt.Sprintf("name=opt/com.coreos/config,file=%s", ignitionPath)},
				{Value: "-chardev"},
				{Value: fmt.Sprintf("file,id=firmwarelog,path=%s", compute.QEMUSeaBiosDebugPipe)},
				{Value: "-device"},
				{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"},
			},
		})))
	})
})

func newDomainWithQEMUCmd(qemuCmd *api.Commandline) api.Domain {
	return api.Domain{
		Spec: api.DomainSpec{
			QEMUCmd: qemuCmd,
		},
	}
}
