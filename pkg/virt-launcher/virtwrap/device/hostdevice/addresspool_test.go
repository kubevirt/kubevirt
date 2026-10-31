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

package hostdevice_test

import (
	"bytes"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice"
)

type envData struct {
	Name  string
	Value string
}

const (
	resourcePrefix = "TEST-PREFIX"
	resource0      = "resource0"
	resource1      = "resource1"
	pciAddresses0  = "0000:81:01.0"
	pciAddresses1  = "0000:81:01.1"
)

var _ = Describe("Address Pool", func() {
	It("fails to pop an address given no resource in env", func() {
		pool := hostdevice.NewAddressPool(resourcePrefix, []string{resource0})
		expectPoolPopFailure(pool, resource0)
	})

	It("fails to pop an address given no addresses for resource", func() {
		env := []envData{newResourceEnv(resourcePrefix, resource0)}
		withEnvironmentContext(env, func() {
			pool := hostdevice.NewAddressPool(resourcePrefix, []string{resource0})
			expectPoolPopFailure(pool, resource0)
		})
	})

	It("succeeds to pop 2 addresses from same resource", func() {
		env := []envData{newResourceEnv(resourcePrefix, resource0, pciAddresses0, pciAddresses1)}
		withEnvironmentContext(env, func() {
			pool := hostdevice.NewAddressPool(resourcePrefix, []string{resource0})
			Expect(pool.Pop(resource0)).To(Equal(pciAddresses0))
			Expect(pool.Pop(resource0)).To(Equal(pciAddresses1))
		})
	})

	It("succeeds to pop 2 addresses from two resources", func() {
		env := []envData{
			newResourceEnv(resourcePrefix, resource0, pciAddresses0),
			newResourceEnv(resourcePrefix, resource1, pciAddresses1),
		}
		withEnvironmentContext(env, func() {
			pool := hostdevice.NewAddressPool(resourcePrefix, []string{resource0, resource1})
			Expect(pool.Pop(resource0)).To(Equal(pciAddresses0))
			Expect(pool.Pop(resource1)).To(Equal(pciAddresses1))
		})
	})

	Context("missing resource variable warning", func() {
		var logs bytes.Buffer

		BeforeEach(func() {
			logs.Reset()
			log.Log.SetIOWriter(&logs)
			DeferCleanup(func() {
				log.Log.SetIOWriter(GinkgoWriter)
			})
		})

		DescribeTable("should warn only when no sibling prefix advertises the resource",
			func(poolPrefix string, env []envData, expectWarning bool) {
				withEnvironmentContext(env, func() {
					hostdevice.NewAddressPool(poolPrefix, []string{resource0})
				})
				if expectWarning {
					Expect(logs.String()).To(ContainSubstring("not set for resource " + resource0))
				} else {
					Expect(logs.String()).ToNot(ContainSubstring("not set for resource"))
				}
			},
			Entry("PCI pool, resource advertised as MDEV",
				v1.PCIResourcePrefix, []envData{newResourceEnv(v1.MDevResourcePrefix, resource0, pciAddresses0)}, false),
			Entry("MDEV pool, resource advertised as PCI",
				v1.MDevResourcePrefix, []envData{newResourceEnv(v1.PCIResourcePrefix, resource0, pciAddresses0)}, false),
			Entry("PCI pool, resource advertised under neither prefix",
				v1.PCIResourcePrefix, nil, true),
			Entry("MDEV pool, resource advertised under neither prefix",
				v1.MDevResourcePrefix, nil, true),
			Entry("USB pool has no sibling, even with the PCI variable set",
				v1.USBResourcePrefix, []envData{newResourceEnv(v1.PCIResourcePrefix, resource0, pciAddresses0)}, true),
			Entry("unknown prefix has no sibling",
				resourcePrefix, nil, true),
		)
	})
})

func newResourceEnv(prefix, resourceName string, addresses ...string) envData {
	resourceName = strings.ToUpper(resourceName)
	return envData{
		Name:  strings.Join([]string{prefix, resourceName}, "_"),
		Value: strings.Join(addresses, ","),
	}
}

func withEnvironmentContext(envDataList []envData, f func()) {
	for _, envVar := range envDataList {
		if os.Setenv(envVar.Name, envVar.Value) == nil {
			defer os.Unsetenv(envVar.Name)
		}
	}
	f()
}

func expectPoolPopFailure(pool *hostdevice.AddressPool, resource string) {
	address, err := pool.Pop(resource)
	ExpectWithOffset(1, err).To(HaveOccurred())
	ExpectWithOffset(1, address).To(BeEmpty())
}
