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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

var _ = Describe("FSFreeze", func() {
	var (
		ctrl          *gomock.Controller
		mockConn      *cli.MockConnection
		mockDomain    *cli.MockVirDomain
		manager       *StorageManager
		metadataCache *metadata.Cache
	)

	const (
		testVmName           = "testvmi"
		testNamespace        = "testnamespace"
		expectedThawedOutput = `{"return":"thawed"}`
		expectedFrozenOutput = `{"return":"frozen"}`
	)

	var testDomainName string

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

	It("IsFreezeInProgress is true while FSFreeze is in progress", func() {
		vmi := newVMI(testNamespace, testVmName)
		blockCh := make(chan struct{})
		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).DoAndReturn(func(_ []string, _ uint32) error {
			<-blockCh
			return nil
		})

		done := make(chan struct{})
		go func() {
			defer close(done)
			_ = manager.FreezeVMI(vmi, 0)
		}()

		Eventually(manager.IsFreezeInProgress, 2*time.Second, 10*time.Millisecond).Should(BeTrue())
		close(blockCh)
		<-done
		Expect(manager.IsFreezeInProgress()).To(BeFalse())
	})

	It("should freeze a VirtualMachineInstance", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
	})

	It("should fail freeze a VirtualMachineInstance during migration", func() {
		vmi := newVMI(testNamespace, testVmName)
		now := metav1.Now()
		migrationMetadata, _ := metadataCache.Migration.Load()
		migrationMetadata.StartTimestamp = &now
		metadataCache.Migration.Store(migrationMetadata)

		Expect(manager.FreezeVMI(vmi, 0)).To(MatchError(ContainSubstring("VMI is currently during migration")))
	})

	It("should unfreeze a VirtualMachineInstance", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Times(1)

		Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
	})

	It("should automatically unfreeze after a timeout a frozen VirtualMachineInstance", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(2)
		mockDomain.EXPECT().Free().Times(2)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Times(1)

		var unfreezeTimeout time.Duration = 3 * time.Second
		Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
		// wait for the unfreeze timeout
		time.Sleep(unfreezeTimeout + 2*time.Second)
	})

	It("should freeze and unfreeze a VirtualMachineInstance without a trigger to the unfreeze timeout", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
		mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(2)
		mockDomain.EXPECT().Free().Times(2)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Times(1)

		var unfreezeTimeout time.Duration = 3 * time.Second
		Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
		time.Sleep(time.Second)
		Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
		// wait for the unfreeze timeout
		time.Sleep(unfreezeTimeout + 2*time.Second)
	})
})
