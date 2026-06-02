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
	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
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
		testVmName    = "testvmi"
		testNamespace = "testnamespace"
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

	loadFSFreezeStatus := func() api.FSFreeze {
		status, _ := metadataCache.FSFreezeStatus.Load()
		return status
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		metadataCache = metadata.NewCache()
		manager = NewStorageManager(mockConn, metadataCache, nil)
		testDomainName = fmt.Sprintf("%s_%s", testNamespace, testVmName)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("should freeze a VirtualMachineInstance", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))
	})

	It("should not set frozen when FSFreeze fails", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(fmt.Errorf("freeze error"))

		Expect(manager.FreezeVMI(vmi, 0)).To(MatchError(ContainSubstring("freeze error")))
		Expect(loadFSFreezeStatus().Status).ToNot(Equal(api.FSFrozen))
		Expect(manager.IsFreezing()).To(BeFalse())

		// Verify that a subsequent freeze attempt is not blocked by a stale freezing state
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil)

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))
	})

	It("should not set frozen when domain lookup fails during freeze", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(nil, fmt.Errorf("domain not found"))

		Expect(manager.FreezeVMI(vmi, 0)).To(MatchError(ContainSubstring("domain not found")))
		Expect(loadFSFreezeStatus().Status).ToNot(Equal(api.FSFrozen))
		Expect(manager.IsFreezing()).To(BeFalse())
	})

	It("should return early when freeze is already in progress", func() {
		vmi := newVMI(testNamespace, testVmName)

		freezeStarted := make(chan struct{})
		freezeBlocked := make(chan struct{})

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).DoAndReturn(func(_ []string, _ uint32) error {
			close(freezeStarted)
			<-freezeBlocked
			return nil
		})

		done := make(chan error, 1)
		go func() {
			done <- manager.FreezeVMI(vmi, 0)
		}()

		<-freezeStarted

		Expect(manager.FreezeVMI(vmi, 0)).To(MatchError(ContainSubstring("freezing is already in progress")))

		close(freezeBlocked)
		Expect(<-done).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))
	})

	It("should be idempotent when already frozen", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
	})

	It("should fail freeze a VirtualMachineInstance during migration", func() {
		vmi := newVMI(testNamespace, testVmName)
		now := metav1.Now()
		migrationMetadata, _ := metadataCache.Migration.Load()
		migrationMetadata.StartTimestamp = &now
		metadataCache.Migration.Store(migrationMetadata)

		Expect(manager.FreezeVMI(vmi, 0)).To(MatchError(ContainSubstring("VMI is currently during migration")))
		Expect(loadFSFreezeStatus().Status).ToNot(Equal(api.FSFrozen))
	})

	It("should unfreeze a VirtualMachineInstance", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(2)
		mockDomain.EXPECT().Free().Times(2)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Times(1)

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSThawed))
	})

	It("should keep frozen when FSThaw fails", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(2)
		mockDomain.EXPECT().Free().Times(2)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Return(fmt.Errorf("thaw error"))

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))

		Expect(manager.UnfreezeVMI(vmi)).To(MatchError(ContainSubstring("thaw error")))
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))
	})

	It("should keep frozen when domain lookup fails during unfreeze", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(1)
		mockDomain.EXPECT().Free().Times(1)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil)

		Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(nil, fmt.Errorf("domain not found"))

		Expect(manager.UnfreezeVMI(vmi)).To(MatchError(ContainSubstring("domain not found")))
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))
	})

	It("should be idempotent when already thawed", func() {
		vmi := newVMI(testNamespace, testVmName)

		Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).ToNot(Equal(api.FSFrozen))
	})

	It("should automatically unfreeze after a timeout a frozen VirtualMachineInstance", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(2)
		mockDomain.EXPECT().Free().Times(2)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Times(1)

		var unfreezeTimeout time.Duration = 3 * time.Second
		Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSFrozen))
		// wait for the unfreeze timeout
		time.Sleep(unfreezeTimeout + 2*time.Second)
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSThawed))
	})

	It("should freeze and unfreeze a VirtualMachineInstance without a trigger to the unfreeze timeout", func() {
		vmi := newVMI(testNamespace, testVmName)

		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil).Times(2)
		mockDomain.EXPECT().Free().Times(2)
		mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Times(1)
		mockDomain.EXPECT().FSThaw(nil, uint32(0)).Times(1)

		var unfreezeTimeout time.Duration = 3 * time.Second
		Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
		time.Sleep(time.Second)
		Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
		Expect(loadFSFreezeStatus().Status).To(Equal(api.FSThawed))
		// wait for the unfreeze timeout to confirm it doesn't trigger again
		time.Sleep(unfreezeTimeout + 2*time.Second)
	})
})
