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
 * Copyright 2021
 *
 */

package cgroup

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type ProcCgroupData struct {
	id         int
	controller string
	slice      string
}

var _ = Describe("Cgroup", func() {
	var (
		cgroupFS      string
		procFS        string
		procCgroupFmt string
	)

	mockCgroupFS := func() (baseDir string) {
		baseDir, err := ioutil.TempDir("", "cgroupfs-*")
		Expect(err).ToNot(HaveOccurred())
		return
	}

	mockProcFS := func() (baseDir string, pidCgroupFmt string) {
		baseDir, err := ioutil.TempDir("", "procfs-*")
		Expect(err).ToNot(HaveOccurred())
		pidFmt := filepath.Join(baseDir, "%d")
		err = os.MkdirAll(fmt.Sprintf(pidFmt, os.Getpid()), os.ModePerm)
		Expect(err).ToNot(HaveOccurred())
		pidCgroupFmt = filepath.Join(pidFmt, "cgroup")
		return
	}

	prepareProcCgroupData := func(pidCgroupFmt string, data []ProcCgroupData) {
		f, err := os.Create(fmt.Sprintf(pidCgroupFmt, os.Getpid()))
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()
		for _, d := range data {
			_, err := fmt.Fprintf(f, "%d:%s:%s\n", d.id, d.controller, d.slice)
			Expect(err).ToNot(HaveOccurred())
		}
	}

	BeforeEach(func() {
		cgroupFS = mockCgroupFS()
		procFS, procCgroupFmt = mockProcFS()
	})

	AfterEach(func() {
		err := os.RemoveAll(cgroupFS)
		Expect(err).ToNot(HaveOccurred())
		err = os.RemoveAll(procFS)
		Expect(err).ToNot(HaveOccurred())
	})

	It("Should return an error if there is no cgroup data in procfs", func() {
		_, err := newParser(true, procFS, cgroupFS).Parse(os.Getpid())
		Expect(err).To(HaveOccurred())
		_, err = newParser(false, procFS, cgroupFS).Parse(os.Getpid())
		Expect(err).To(HaveOccurred())
	})

	Context("With Control Group v1", func() {
		const isCgroup2UnifiedMode = false

		var (
			procCgroupV1Data = []ProcCgroupData{
				{12, "devices", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{11, "rdma", "/"},
				{10, "memory", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{9, "cpu,cpuacct", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{8, "freezer", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{7, "perf_event", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{6, "net_cls,net_prio", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{5, "pids", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{4, "blkio", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{3, "hugetlb", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{2, "cpuset", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{1, "name=systemd", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podf02d4bde_4ff6_4e62_8069_65daed637113.slice/docker-ad2bc8dce287c58d4d5d1e83566dbe93c5f2a6a0bb95cb87f150185abf5e3a28.scope"},
				{0, "", "/"},
			}
		)

		prepareCgroupData := func(cgroupPath, slice string) {
			path := filepath.Join(cgroupPath, "devices", slice)
			err := os.MkdirAll(path, os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
		}

		BeforeEach(func() {
			prepareProcCgroupData(procCgroupFmt, procCgroupV1Data)
			prepareCgroupData(cgroupFS, procCgroupV1Data[0].slice)
		})

		It("Should parse the cgroup data from procfs", func() {
			data, err := newParser(isCgroup2UnifiedMode, procFS, cgroupFS).Parse(os.Getpid())
			Expect(err).ToNot(HaveOccurred())
			Expect(len(data)).To(Equal(len(procCgroupV1Data) + 2)) // +2 for cpu,cpuacct and net_cls,net_prio
			for _, d := range procCgroupV1Data {
				for _, c := range strings.Split(d.controller, ",") {
					slice, ok := data[c]
					Expect(ok).To(BeTrue())
					Expect(slice).To(Equal(d.slice))
				}
			}
		})
	})

	Context("With Control Group v2", func() {
		const isCgroup2UnifiedMode = true

		var (
			procCgroupV2Data = []ProcCgroupData{
				{0, "", "/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-podcb9f952b_8903_4be9_b3ab_e6c3e19b2750.slice/crio-17b7313ee71796c899c44001d64d9635922a661c98317a2e757cfd8da4334613.scope"},
			}

			availableControllers = []string{"cpuset", "cpu", "io", "memory", "hugetlb", "pids", "rdma"}
			pseudoControllers    = []string{"freezer", "devices"}
		)

		prepareCgroupData := func(cgroupPath, slice string, controllers []string) {
			path := filepath.Join(cgroupPath, slice)
			err := os.MkdirAll(path, os.ModePerm)
			Expect(err).ToNot(HaveOccurred())
			f, err := os.Create(filepath.Join(path, "cgroup.controllers"))
			Expect(err).ToNot(HaveOccurred())
			defer f.Close()
			f.WriteString(strings.Join(controllers, " "))
		}

		BeforeEach(func() {
			prepareProcCgroupData(procCgroupFmt, procCgroupV2Data)
			prepareCgroupData(cgroupFS, procCgroupV2Data[0].slice, availableControllers)
		})

		It("Should parse the cgroup data from procfs", func() {
			var allControllers []string
			allControllers = append(allControllers, availableControllers...)
			allControllers = append(allControllers, pseudoControllers...)
			data, err := newParser(isCgroup2UnifiedMode, procFS, cgroupFS).Parse(os.Getpid())
			Expect(err).ToNot(HaveOccurred())
			Expect(len(data)).To(Equal(len(allControllers)))
			for _, c := range allControllers {
				slice, ok := data[c]
				Expect(ok).To(BeTrue())
				Expect(slice).To(Equal(procCgroupV2Data[0].slice))
			}
		})
	})
})
