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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package hostdevice_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
