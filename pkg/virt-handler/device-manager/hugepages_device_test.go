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

package device_manager

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginapi "kubevirt.io/kubevirt/pkg/virt-handler/device-manager/deviceplugin/v1beta1"
)

var _ = Describe("Dynamic Hugepages Device Plugin", func() {

	Context("isGigantic", func() {
		It("should classify 2Mi as non-gigantic (order 9)", func() {
			Expect(isGigantic(2048)).To(BeFalse())
		})

		It("should classify 4Mi as non-gigantic (order 10)", func() {
			Expect(isGigantic(4096)).To(BeFalse())
		})

		It("should classify 1Gi as gigantic (order 18)", func() {
			Expect(isGigantic(1048576)).To(BeTrue())
		})

		It("should classify 32Mi as gigantic (order 13)", func() {
			Expect(isGigantic(32768)).To(BeTrue())
		})
	})

	Context("CMA config detection", func() {
		It("should detect hugetlb_cma and hugetlb_cma_only", func() {
			tmpDir := setupFakeProc("BOOT_IMAGE=/vmlinuz hugetlb_cma=16G hugetlb_cma_only quiet")
			cfg := detectCMAConfig(tmpDir + "/")
			Expect(cfg.sizeMiB).To(Equal(int64(16 * 1024)))
			Expect(cfg.cmaOnly).To(BeTrue())
		})

		It("should detect hugetlb_cma without cma_only", func() {
			tmpDir := setupFakeProc("hugetlb_cma=8G quiet")
			cfg := detectCMAConfig(tmpDir + "/")
			Expect(cfg.sizeMiB).To(Equal(int64(8 * 1024)))
			Expect(cfg.cmaOnly).To(BeFalse())
		})

		It("should return zeros when neither is set", func() {
			tmpDir := setupFakeProc("quiet splash")
			cfg := detectCMAConfig(tmpDir + "/")
			Expect(cfg.sizeMiB).To(Equal(int64(0)))
			Expect(cfg.cmaOnly).To(BeFalse())
		})
	})

	Context("parseMemorySize", func() {
		It("should parse G suffix", func() {
			Expect(parseMemorySize("16G")).To(Equal(int64(16384)))
			Expect(parseMemorySize("1g")).To(Equal(int64(1024)))
		})

		It("should parse M suffix", func() {
			Expect(parseMemorySize("4096M")).To(Equal(int64(4096)))
		})

		It("should parse K suffix", func() {
			Expect(parseMemorySize("1048576K")).To(Equal(int64(1024)))
		})

		It("should return 0 for invalid input", func() {
			Expect(parseMemorySize("")).To(Equal(int64(0)))
			Expect(parseMemorySize("abc")).To(Equal(int64(0)))
		})
	})

	Context("sysfs directory parsing", func() {
		It("should parse hugepages-2048kB", func() {
			size, ok := parseHugepageDirName("hugepages-2048kB")
			Expect(ok).To(BeTrue())
			Expect(size).To(Equal(int64(2048)))
		})

		It("should parse hugepages-1048576kB", func() {
			size, ok := parseHugepageDirName("hugepages-1048576kB")
			Expect(ok).To(BeTrue())
			Expect(size).To(Equal(int64(1048576)))
		})

		It("should reject invalid names", func() {
			_, ok := parseHugepageDirName("not-hugepages")
			Expect(ok).To(BeFalse())
		})
	})

	Context("page size labels", func() {
		It("should produce correct labels", func() {
			Expect(pageSizeToLabel(2048)).To(Equal("2mi"))
			Expect(pageSizeToLabel(1048576)).To(Equal("1gi"))
			Expect(pageSizeToLabel(32768)).To(Equal("32mi"))
		})
	})

	Context("discovery — non-gigantic (2Mi)", func() {
		It("should create plugin when nr_overcommit_hugepages > 0", func() {
			tmpDir := setupDynamicNode("quiet", map[string]map[string]string{
				"hugepages-2048kB": {overcommitFile: "4096", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(HaveLen(1))
			Expect(plugins[0].pageLabel).To(Equal("2mi"))
			// 4096 overcommit pages = 4096 devices (one per page)
			Expect(plugins[0].devs).To(HaveLen(4096))
			Expect(plugins[0].GetDeviceName()).To(Equal("dynamic-hugepages-2mi"))
			Expect(plugins[0].resourceName).To(Equal("devices.kubevirt.io/dynamic-hugepages-2mi"))
		})

		It("should skip when nr_overcommit_hugepages is 0", func() {
			tmpDir := setupDynamicNode("quiet", map[string]map[string]string{
				"hugepages-2048kB": {overcommitFile: "0", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(BeEmpty())
		})

		It("should not require CMA for 2Mi pages", func() {
			// No hugetlb_cma in cmdline, but overcommit is set
			tmpDir := setupDynamicNode("quiet", map[string]map[string]string{
				"hugepages-2048kB": {overcommitFile: "2048", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(HaveLen(1))
			Expect(plugins[0].devs).To(HaveLen(2048))
		})
	})

	Context("discovery — gigantic (1Gi)", func() {
		It("should create plugin when both hugetlb_cma and hugetlb_cma_only are set", func() {
			tmpDir := setupDynamicNode("hugetlb_cma=16G hugetlb_cma_only", map[string]map[string]string{
				"hugepages-1048576kB": {overcommitFile: "0", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(HaveLen(1))
			Expect(plugins[0].pageLabel).To(Equal("1gi"))
			// 16Gi = 16384MiB / 1024MiB per page = 16 devices
			Expect(plugins[0].devs).To(HaveLen(16))
		})

		It("should skip 1Gi when hugetlb_cma_only is missing", func() {
			tmpDir := setupDynamicNode("hugetlb_cma=16G", map[string]map[string]string{
				"hugepages-1048576kB": {overcommitFile: "0", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(BeEmpty())
		})

		It("should skip 1Gi when hugetlb_cma is missing", func() {
			tmpDir := setupDynamicNode("hugetlb_cma_only", map[string]map[string]string{
				"hugepages-1048576kB": {overcommitFile: "0", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(BeEmpty())
		})
	})

	Context("discovery — mixed page sizes", func() {
		It("should create both 2Mi and 1Gi plugins when properly configured", func() {
			tmpDir := setupDynamicNode("hugetlb_cma=16G hugetlb_cma_only", map[string]map[string]string{
				"hugepages-2048kB":    {overcommitFile: "8192", nrFile: "0"},
				"hugepages-1048576kB": {overcommitFile: "0", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(HaveLen(2))

			labels := map[string]int{}
			for _, p := range plugins {
				labels[p.pageLabel] = len(p.devs)
			}
			// 2Mi: 8192 overcommit pages
			Expect(labels["2mi"]).To(Equal(8192))
			// 1Gi: 16Gi CMA / 1Gi = 16
			Expect(labels["1gi"]).To(Equal(16))
		})

		It("should only create 2Mi when CMA is not configured", func() {
			tmpDir := setupDynamicNode("quiet", map[string]map[string]string{
				"hugepages-2048kB":    {overcommitFile: "4096", nrFile: "0"},
				"hugepages-1048576kB": {overcommitFile: "0", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(HaveLen(1))
			Expect(plugins[0].pageLabel).To(Equal("2mi"))
		})
	})

	Context("Allocate", func() {
		It("should return empty response (no mounts — virt-controller handles volumes)", func() {
			tmpDir := setupDynamicNode("quiet", map[string]map[string]string{
				"hugepages-2048kB": {overcommitFile: "512", nrFile: "0"},
			})

			plugins := DiscoverHugepageDevicePlugins(tmpDir + "/")
			Expect(plugins).To(HaveLen(1))

			req := &pluginapi.AllocateRequest{
				ContainerRequests: []*pluginapi.ContainerAllocateRequest{
					{DevicesIDs: []string{"dynamic-hugepages-2mi-0", "dynamic-hugepages-2mi-1"}},
				},
			}
			resp, err := plugins[0].Allocate(context.Background(), req)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.ContainerResponses).To(HaveLen(1))
			Expect(resp.ContainerResponses[0].Mounts).To(BeEmpty())
		})
	})
})

func setupFakeProc(cmdline string) string {
	tmpDir := GinkgoT().TempDir()
	procDir := filepath.Join(tmpDir, "proc")
	Expect(os.MkdirAll(procDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(procDir, "cmdline"), []byte(cmdline), 0644)).To(Succeed())
	return tmpDir
}

func setupDynamicNode(cmdline string, pageSizes map[string]map[string]string) string {
	tmpDir := GinkgoT().TempDir()

	procDir := filepath.Join(tmpDir, "proc")
	Expect(os.MkdirAll(procDir, 0755)).To(Succeed())
	Expect(os.WriteFile(filepath.Join(procDir, "cmdline"), []byte(cmdline), 0644)).To(Succeed())

	for dirName, files := range pageSizes {
		sysfsDir := filepath.Join(tmpDir, hugepagesSysfsBase, dirName)
		Expect(os.MkdirAll(sysfsDir, 0755)).To(Succeed())
		for name, content := range files {
			Expect(os.WriteFile(filepath.Join(sysfsDir, name), []byte(content+"\n"), 0644)).To(Succeed())
		}
	}

	return tmpDir
}
