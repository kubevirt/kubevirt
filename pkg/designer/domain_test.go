/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package designer_test

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	// To register k8s API types via init()
	_ "k8s.io/client-go/kubernetes"
	kubeapi "k8s.io/client-go/pkg/api"
	kubeapiv1 "k8s.io/client-go/pkg/api/v1"

	apiv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/designer"
	"kubevirt.io/kubevirt/pkg/logging"
	"kubevirt.io/kubevirt/pkg/testutil"
)

var _ = Describe("Domain", func() {
	RegisterFailHandler(Fail)

	logging.DefaultLogger().SetIOWriter(GinkgoWriter)

	var (
		server *ghttp.Server
	)

	BeforeEach(func() {
		server = testutil.NewKubeServer([]testutil.Resource{
			&testutil.TestPersistentVolumeClaimISCSI,
			&testutil.TestPersistentVolumeISCSI,
		})
	})

	AfterEach(func() {
		server.Close()
	})

	Context("Design XML", func() {
		It("designs XML from the VM", func() {
			bootTimeout := uint(10)
			vm := &apiv1.VM{
				ObjectMeta: kubeapiv1.ObjectMeta{
					Name:      "testvm",
					Namespace: kubeapi.NamespaceDefault,
					UID:       "9838eeb0-663e-485d-a544-3163fb1f0b28",
				},
				Spec: apiv1.VMSpec{
					Domain: &apiv1.DomainSpec{
						Type: "kvm",
						Memory: apiv1.Memory{
							Value: 8,
						},
						OS: apiv1.OS{
							Type: apiv1.OSType{
								OS:      "hvm",
								Arch:    "x86_64",
								Machine: "q35",
							},
							SMBios: &apiv1.SMBios{
								Mode: "sysinfo",
							},
							BootMenu: &apiv1.BootMenu{
								Enabled: true,
								Timeout: &bootTimeout,
							},
						},
						SysInfo: &apiv1.SysInfo{
							Type: "smbios",
							BIOS: []apiv1.Entry{
								{Name: "vendor", Value: "ACME"},
							},
							System: []apiv1.Entry{
								{Name: "manufacturer", Value: "ACME"},
								{Name: "product", Value: "RoadRunner"},
							},
							BaseBoard: []apiv1.Entry{
								{Name: "manufacturer", Value: "ACME"},
								{Name: "product", Value: "Wile E Coyote"},
							},
						},
						Devices: apiv1.Devices{
							Disks: []apiv1.Disk{
								{
									Device: "disk",
									Source: apiv1.DiskSource{
										PersistentVolumeClaim: &apiv1.DiskSourcePersistentVolumeClaim{
											ClaimName: testutil.TestPersistentVolumeClaimISCSI.ObjectMeta.Name,
										},
									},
									Target: apiv1.DiskTarget{
										Device: "vda",
										Bus:    "virtio",
									},
								},
								{
									Device: "disk",
									Source: apiv1.DiskSource{
										ISCSI: &apiv1.DiskSourceISCSI{
											TargetPortal: "127.0.0.1:6543",
											Lun:          2,
											IQN:          "iqn.2009-02.com.test:for.all",
										},
									},
									Target: apiv1.DiskTarget{
										Device: "vdb",
										Bus:    "virtio",
									},
								},
							},
						},
					},
				},
			}

			expectXML := strings.Join([]string{
				`<domain type="kvm">`,
				`  <name>testvm</name>`,
				`  <uuid>9838eeb0-663e-485d-a544-3163fb1f0b28</uuid>`,
				`  <memory unit="MiB">8</memory>`,
				`  <sysinfo type="smbios">`,
				`    <system>`,
				`      <entry name="manufacturer">ACME</entry>`,
				`      <entry name="product">RoadRunner</entry>`,
				`    </system>`,
				`    <bios>`,
				`      <entry name="vendor">ACME</entry>`,
				`    </bios>`,
				`    <baseBoard>`,
				`      <entry name="manufacturer">ACME</entry>`,
				`      <entry name="product">Wile E Coyote</entry>`,
				`    </baseBoard>`,
				`  </sysinfo>`,
				`  <os>`,
				`    <type arch="x86_64" machine="q35">hvm</type>`,
				`    <bootmenu enabled="yes" timeout="10"></bootmenu>`,
				`    <smbios mode="sysinfo"></smbios>`,
				`  </os>`,
				`  <devices>`,
				`    <disk type="network" device="disk">`,
				`      <source protocol="iscsi" name="iqn.2009-02.com.test:for.all/1">`,
				`        <host transport="tcp" name="127.0.0.1" port="6543"></host>`,
				`      </source>`,
				`      <target dev="vda" bus="virtio"></target>`,
				`    </disk>`,
				`    <disk type="network" device="disk">`,
				`      <source protocol="iscsi" name="iqn.2009-02.com.test:for.all/2">`,
				`        <host transport="tcp" name="127.0.0.1" port="6543"></host>`,
				`      </source>`,
				`      <target dev="vdb" bus="virtio"></target>`,
				`    </disk>`,
				`  </devices>`,
				`</domain>`,
			}, "\n")

			restClient, err := testutil.NewKubeRESTClient(server.URL())
			Expect(err).NotTo(HaveOccurred())

			domdesign := designer.NewDomainDesigner(restClient, kubeapi.NamespaceDefault)

			err = domdesign.ApplySpec(vm)
			Expect(err).NotTo(HaveOccurred())

			actualXML, err := domdesign.Domain.Marshal()
			Expect(err).NotTo(HaveOccurred())

			Expect(actualXML).To(Equal(expectXML))
		})
	})

})

func TestDomain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Domain")
}
