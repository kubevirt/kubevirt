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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	"kubevirt.io/kubevirt/pkg/vsock"
	"kubevirt.io/kubevirt/pkg/vsock/mode"
)

var _ = Describe("VSOCK Domain Configurator", func() {
	var fakeProc string

	BeforeEach(func() {
		fakeProc = GinkgoT().TempDir()
	})

	It("Should not configure VSOCK when VSOCKCID is absent", func() {
		vmi := libvmi.New()
		var domain api.Domain

		configurator := compute.VSOCKDomainConfigurator{ProcPath: fakeProc}
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())
		Expect(domain).To(Equal(api.Domain{}))
	})

	It("Should configure VSOCK when VSOCKCID is present in global mode", func() {
		const expectedVSOCKID = uint32(50)
		vmiStatus := v1.VirtualMachineInstanceStatus{
			VSOCKCID: pointer.P(expectedVSOCKID),
		}
		vmi := libvmi.New(
			libvmistatus.WithStatus(vmiStatus),
		)
		var domain api.Domain

		createFakeVsockModeFile(fakeProc, mode.ModeGlobal)

		configurator := compute.VSOCKDomainConfigurator{ProcPath: fakeProc}
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					VSOCK: &api.VSOCK{
						Model: "virtio-non-transitional",
						CID: api.CID{
							Auto:    "no",
							Address: expectedVSOCKID,
						},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})

	It("Should configure VSOCK when VSOCKCID is present in local mode", func() {
		vmiStatus := v1.VirtualMachineInstanceStatus{
			VSOCKCID: pointer.P[uint32](1234),
		}
		vmi := libvmi.New(
			libvmistatus.WithStatus(vmiStatus),
		)
		var domain api.Domain

		createFakeVsockModeFile(fakeProc, mode.ModeLocal)

		configurator := compute.VSOCKDomainConfigurator{ProcPath: fakeProc}
		Expect(configurator.Configure(vmi, &domain)).To(Succeed())

		expectedDomain := api.Domain{
			Spec: api.DomainSpec{
				Devices: api.Devices{
					VSOCK: &api.VSOCK{
						Model: "virtio-non-transitional",
						CID: api.CID{
							Auto:    "no",
							Address: vsock.LocalCID,
						},
					},
				},
			},
		}
		Expect(domain).To(Equal(expectedDomain))
	})
})

func createFakeVsockModeFile(tmpDir, vsockMode string) {
	vsockPath := filepath.Join(tmpDir, "sys", "net", "vsock")
	Expect(os.MkdirAll(vsockPath, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(vsockPath, "ns_mode"), []byte(vsockMode+"\n"), 0600)).To(Succeed())
}
