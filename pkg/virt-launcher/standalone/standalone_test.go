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
	"bytes"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/standalone"
)

var _ = Describe("HandleStandaloneMode", func() {
	var (
		mockCtrl *gomock.Controller
		mockDM   *mockDomainManager
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockDM = &mockDomainManager{mockCtrl: mockCtrl}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	It("should do nothing if STANDALONE_VMI env var is not set", func() {
		os.Unsetenv("STANDALONE_VMI")
		standalone.HandleStandaloneMode(mockDM)
	})

	It("should panic on invalid JSON in STANDALONE_VMI", func() {
		os.Setenv("VMI_OBJ", "invalid json")
		defer os.Unsetenv("VMI_OBJ")

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(Panic())
	})

	It("should panic if SyncVMI fails", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(true, fmt.Errorf("sync error"))

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).To(PanicWith("sync error"))
	})

	It("should succeed with valid JSON and successful SyncVMI", func() {
		vmiJSON := `{"apiVersion":"kubevirt.io/v1","kind":"VirtualMachineInstance","metadata":{"name":"testvmi"}}`
		os.Setenv("STANDALONE_VMI", vmiJSON)
		defer os.Unsetenv("STANDALONE_VMI")

		mockDM.EXPECT().SyncVMI(gomock.Any(), true, nil).Return(true, nil)

		Expect(func() {
			standalone.HandleStandaloneMode(mockDM)
		}).NotTo(Panic())
	})
})

type mockDomainManager struct {
	mockCtrl *gomock.Controller
}

func (m *mockDomainManager) SyncVMI(vmi *v1.VirtualMachineInstance, allowEmulation bool, secretUUID *string) (bool, error) {
	return true, nil
}
