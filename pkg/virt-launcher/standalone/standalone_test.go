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

package standalone_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	launcherconfig "kubevirt.io/kubevirt/pkg/virt-launcher/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/standalone"
	virtwrap "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

var _ = Describe("HandleStandaloneMode", func() {
	var (
		mockCtrl *gomock.Controller
		mockDM   *virtwrap.MockDomainManager
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDM = virtwrap.NewMockDomainManager(mockCtrl)
		// Reset global config before each test
		launcherconfig.ResetGlobalConfig()
	})

	AfterEach(func() {
		mockCtrl.Finish()
		// Reset global config after each test
		launcherconfig.ResetGlobalConfig()
	})

	It("should do nothing if STANDALONE_VMI is not set", func() {
		cfg := &launcherconfig.Config{
			StandaloneVMI: "",
		}
		standalone.HandleStandaloneModeWithConfig(mockDM, cfg)
	})

	It("should do nothing with nil config", func() {
		standalone.HandleStandaloneModeWithConfig(mockDM, nil)
	})

	It("should panic on invalid JSON in STANDALONE_VMI", func() {
		cfg := &launcherconfig.Config{
			StandaloneVMI: "invalid json",
		}

		Expect(func() {
			standalone.HandleStandaloneModeWithConfig(mockDM, cfg)
		}).To(Panic())
	})

	It("should panic if SyncVMI fails", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		cfg := &launcherconfig.Config{
			StandaloneVMI: vmiJSON,
		}

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, fmt.Errorf("sync error"))

		Expect(func() {
			standalone.HandleStandaloneModeWithConfig(mockDM, cfg)
		}).To(PanicWith(MatchError(ContainSubstring("sync error"))))
	})

	It("should succeed with valid JSON and successful SyncVMI", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		cfg := &launcherconfig.Config{
			StandaloneVMI: vmiJSON,
		}

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, nil)

		Expect(func() {
			standalone.HandleStandaloneModeWithConfig(mockDM, cfg)
		}).NotTo(Panic())
	})

	It("should succeed with valid YAML and successful SyncVMI", func() {
		vmiYAML := `apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: testvmi-yaml`
		cfg := &launcherconfig.Config{
			StandaloneVMI: vmiYAML,
		}

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(nil, nil)

		Expect(func() {
			standalone.HandleStandaloneModeWithConfig(mockDM, cfg)
		}).NotTo(Panic())
	})

	It("should panic on invalid YAML in STANDALONE_VMI", func() {
		cfg := &launcherconfig.Config{
			StandaloneVMI: "invalid: yaml: here",
		}

		Expect(func() {
			standalone.HandleStandaloneModeWithConfig(mockDM, cfg)
		}).To(Panic())
	})
})
