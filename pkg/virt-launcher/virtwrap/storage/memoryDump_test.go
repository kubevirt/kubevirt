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

package storage

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

var _ = Describe("MemoryDump", func() {
	var (
		ctrl          *gomock.Controller
		mockConn      *cli.MockConnection
		mockDomain    *cli.MockVirDomain
		manager       *StorageManager
		metadataCache *metadata.Cache
	)

	const (
		testVmName    = "testvmi"
		testNamespace = "testnamespace"
		testDumpPath  = "/test/dump/path/vol1.memory.dump"
	)

	var testDomainName string

	mockDomainWithFreeExpectation := func(_ string) (cli.VirDomain, error) {
		// Make sure that we always free the domain after use
		mockDomain.EXPECT().Free()
		return mockDomain, nil
	}

	newVMI := func(namespace, name string) *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				UID:       "test-uid",
			},
		}
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		metadataCache = metadata.NewCache()
		manager = NewStorageManager(mockConn, metadataCache)
		testDomainName = fmt.Sprintf("%s_%s", testNamespace, testVmName)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should update domain with memory dump info when completed successfully", func() {
		mockConn.EXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
		mockDomain.EXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Return(nil)

		vmi := newVMI(testNamespace, testVmName)
		Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())
		// Expect extra call to memory dump not to impact
		Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())

		Eventually(func() bool {
			memoryDump, _ := metadataCache.MemoryDump.Load()
			return memoryDump.Completed
		}, 5*time.Second, 2).Should(BeTrue())
	})

	It("should skip memory dump if the same dump command already completed successfully", func() {
		mockConn.EXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
		mockDomain.EXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Times(1).Return(nil)

		vmi := newVMI(testNamespace, testVmName)
		Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())
		// Expect extra call to memory dump not to impact
		Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())

		Eventually(func() bool {
			memoryDump, _ := metadataCache.MemoryDump.Load()
			return memoryDump.Completed
		}, 5*time.Second, 2).Should(BeTrue())
		// Expect extra call to memory dump after completion
		// not to call core dump command again
		Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())
	})

	It("should update domain with memory dump info if memory dump failed", func() {
		mockConn.EXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
		dumpFailure := fmt.Errorf("Memory dump failed!!")
		mockDomain.EXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Return(dumpFailure)

		vmi := newVMI(testNamespace, testVmName)
		err := manager.MemoryDump(vmi, testDumpPath)
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() bool {
			memoryDump, _ := metadataCache.MemoryDump.Load()
			return memoryDump.Failed
		}, 5*time.Second).Should(BeTrue(), "failed memory dump result wasn't set")
	})
})
