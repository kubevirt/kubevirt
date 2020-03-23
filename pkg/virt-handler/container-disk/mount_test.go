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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package container_disk

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/client-go/api/v1"
)

var _ = Describe("ContainerDisk", func() {
	var tmpDir string
	var m *Mounter
	var err error
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tmpDir, err = ioutil.TempDir("", "containerdisktest")
		Expect(err).ToNot(HaveOccurred())
		vmi = v1.NewMinimalVMI("fake-vmi")
		vmi.UID = "1234"

		m = &Mounter{
			MountStateDir: tmpDir,
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	Describe("container-disk", func() {
		Context("verify mount target recording for vmi", func() {
			It("should set and get same results", func() {

				// verify reading non-existent results just returns empty slice
				record, err := m.getMountTargetRecord(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(record.MountTargetEntries)).To(Equal(0))

				// verify setting a result works
				err = m.setMountTargetRecordEntry(vmi, "sometargetfile", "somesocketfile")
				Expect(err).ToNot(HaveOccurred())

				// verify we can read a result
				record, err = m.getMountTargetRecord(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(record.MountTargetEntries)).To(Equal(1))
				Expect(record.MountTargetEntries[0].TargetFile).To(Equal("sometargetfile"))
				Expect(record.MountTargetEntries[0].SocketFile).To(Equal("somesocketfile"))

				// verify appending more results works
				err = m.setMountTargetRecordEntry(vmi, "sometargetfile2", "somesocketfile2")
				Expect(err).ToNot(HaveOccurred())

				// verify we return all results when multiples exist
				record, err = m.getMountTargetRecord(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(record.MountTargetEntries)).To(Equal(2))
				Expect(record.MountTargetEntries[0].TargetFile).To(Equal("sometargetfile"))
				Expect(record.MountTargetEntries[0].SocketFile).To(Equal("somesocketfile"))
				Expect(record.MountTargetEntries[1].TargetFile).To(Equal("sometargetfile2"))
				Expect(record.MountTargetEntries[1].SocketFile).To(Equal("somesocketfile2"))

				// verify delete results
				err = m.deleteMountTargetRecord(vmi)
				Expect(err).ToNot(HaveOccurred())

				// verify deleting results that don't exist won't fail
				err = m.deleteMountTargetRecord(vmi)
				Expect(err).ToNot(HaveOccurred())

				// verify reading deleted results just returns empty slice
				record, err = m.getMountTargetRecord(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(len(record.MountTargetEntries)).To(Equal(0))
			})
		})
	})
})
