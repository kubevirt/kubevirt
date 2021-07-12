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

package legacy_test

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device/hostdevice/legacy"
)

type envData struct {
	Name  string
	Value string
}

var _ = Describe("NewGPUPCIAddressPool", func() {
	const (
		gpuResource0 = "GPU_PASSTHROUGH_DEVICES_SOME_VENDOR"
		gpuResource1 = "GPU_PASSTHROUGH_DEVICES_OTHER_VENDOR"

		pci0 = "2609:19:90.0"
		pci1 = "2609:19:90.1"
		pci2 = "2609:19:91.0"
		pci3 = "2609:19:91.1"
	)

	It("has no PCI addresses", func() {
		pool := legacy.NewGPUPCIAddressPool()
		Expect(pool.Len()).To(Equal(0))
		_, err := pool.Pop()
		Expect(err).To(HaveOccurred())
	})

	It("has a single PCI address", func() {
		env := []envData{{gpuResource0, strings.Join([]string{pci0, ""}, ",")}}
		withEnvironmentContext(env, func() {
			pool := legacy.NewGPUPCIAddressPool()

			Expect(pool.Len()).To(Equal(1))
			Expect(pool.Pop()).To(Equal(pci0))
		})
	})

	It("has multiple PCI addresses, from the same resource", func() {
		env := []envData{{gpuResource0, strings.Join([]string{pci0, pci1}, ",")}}
		withEnvironmentContext(env, func() {
			pool := legacy.NewGPUPCIAddressPool()

			Expect(pool.Len()).To(Equal(2))
			Expect(pool.Pop()).To(Equal(pci0))
			Expect(pool.Pop()).To(Equal(pci1))
		})
	})

	It("has multiple PCI addresses, from the multiple resources", func() {
		env := []envData{
			{gpuResource0, strings.Join([]string{pci0, pci1}, ",")},
			{gpuResource1, strings.Join([]string{pci2, pci3}, ",")},
		}
		withEnvironmentContext(env, func() {
			pool := legacy.NewGPUPCIAddressPool()

			Expect(pool.Len()).To(Equal(4))
			Expect(pool.Pop()).To(Equal(pci0))
			Expect(pool.Pop()).To(Equal(pci1))
			Expect(pool.Pop()).To(Equal(pci2))
			Expect(pool.Pop()).To(Equal(pci3))
		})
	})
})

var _ = Describe("NewVGPUMdevAddressPool", func() {
	const (
		vgpuResource0 = "VGPU_PASSTHROUGH_DEVICES_SOME_VENDOR"
		vgpuResource1 = "VGPU_PASSTHROUGH_DEVICES_OTHER_VENDOR"

		uuid0 = "aa618089-8b16-4d01-a136-25a0f3c73123"
		uuid1 = "aa618089-8b16-4d01-a136-25a0f3c73124"
		uuid2 = "ba618089-8b16-4d01-a136-25a0f3c73123"
		uuid3 = "ba618089-8b16-4d01-a136-25a0f3c73124"
	)

	It("has no MDEV addresses", func() {
		pool := legacy.NewVGPUMdevAddressPool()
		Expect(pool.Len()).To(Equal(0))
		_, err := pool.Pop()
		Expect(err).To(HaveOccurred())
	})

	It("has a single MDEV address", func() {
		env := []envData{{vgpuResource0, strings.Join([]string{uuid0, ""}, ",")}}
		withEnvironmentContext(env, func() {
			pool := legacy.NewVGPUMdevAddressPool()

			Expect(pool.Len()).To(Equal(1))
			Expect(pool.Pop()).To(Equal(uuid0))
		})
	})

	It("has multiple MDEV addresses, from the same resource", func() {
		env := []envData{{vgpuResource0, strings.Join([]string{uuid0, uuid1}, ",")}}
		withEnvironmentContext(env, func() {
			pool := legacy.NewVGPUMdevAddressPool()

			Expect(pool.Len()).To(Equal(2))
			Expect(pool.Pop()).To(Equal(uuid0))
			Expect(pool.Pop()).To(Equal(uuid1))
		})
	})

	It("has multiple MDEV addresses, from the multiple resources", func() {
		env := []envData{
			{vgpuResource0, strings.Join([]string{uuid0, uuid1}, ",")},
			{vgpuResource1, strings.Join([]string{uuid2, uuid3}, ",")},
		}
		withEnvironmentContext(env, func() {
			pool := legacy.NewVGPUMdevAddressPool()

			Expect(pool.Len()).To(Equal(4))
			Expect(pool.Pop()).To(Equal(uuid0))
			Expect(pool.Pop()).To(Equal(uuid1))
			Expect(pool.Pop()).To(Equal(uuid2))
			Expect(pool.Pop()).To(Equal(uuid3))
		})
	})
})

func withEnvironmentContext(envDataList []envData, f func()) {
	for _, envVar := range envDataList {
		if os.Setenv(envVar.Name, envVar.Value) == nil {
			defer os.Unsetenv(envVar.Name)
		}
	}
	f()
}
