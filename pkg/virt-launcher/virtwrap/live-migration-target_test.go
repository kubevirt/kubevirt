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

package virtwrap

import (
	"fmt"
	"os"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/watch"
	api2 "kubevirt.io/client-go/api"
	"libvirt.org/go/libvirt"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

type eventNotifier struct {
}

func (e eventNotifier) SendEvent(_ watch.Event) error {
	return nil
}

func (e eventNotifier) UpdateEvents(_ watch.Event) {
}

var _ = Describe("client", func() {
	var shareDir string

	AfterEach(func() {
		os.RemoveAll(shareDir)
	})

	DescribeTable("Should monitor the end of a migration", func(type1, type2 libvirt.DomainJobType, op1, op2 libvirt.DomainJobOperationType, err error) {
		var domainJobType = type1
		var domainJobOperation = op1
		var domainJobError = err
		var lock sync.Mutex

		By("Creating and starting a target migration monitor")
		ctrl := gomock.NewController(GinkgoT())
		mockLibvirt := testing.NewLibvirt(ctrl)
		mockLibvirt.ConnectionEXPECT().LookupDomainByName(gomock.Any()).Return(mockLibvirt.VirtDomain, nil).AnyTimes()
		mockLibvirt.DomainEXPECT().GetJobInfo().DoAndReturn(func() (*libvirt.DomainJobInfo, error) {
			lock.Lock()
			defer lock.Unlock()
			return &libvirt.DomainJobInfo{
				Type:      domainJobType,
				Operation: domainJobOperation,
			}, domainJobError
		}).AnyTimes()
		mockLibvirt.DomainEXPECT().Free().Return(nil).AnyTimes()
		eventChan := make(chan watch.Event, 100)
		vmi := api2.NewMinimalVMI("fake-vmi")
		domain := api.NewMinimalDomain("test")
		metadataCache := metadata.NewCache()
		notifier := &eventNotifier{}
		monitor := NewTargetMigrationMonitor(mockLibvirt.VirtConnection, eventChan, vmi, domain, metadataCache, notifier)
		monitor.StartMonitor()

		By("Ensuring that nothing gets added to the metadata cache as long as the migration is running")
		Consistently(func() bool {
			_, exists := metadataCache.Migration.Load()
			return exists
		}).WithPolling(200 * time.Millisecond).WithTimeout(2 * time.Second).Should(BeFalse())

		By("Simulating the end of the migration")
		lock.Lock()
		domainJobType = type2
		domainJobOperation = op2
		domainJobError = nil
		lock.Unlock()

		By("Ensuring an entry with an endTimestamp gets added to the metadata cache")
		Eventually(func() bool {
			migrationMetadata, exists := metadataCache.Migration.Load()
			return exists && migrationMetadata.EndTimestamp != nil
		}).WithPolling(200 * time.Millisecond).WithTimeout(2 * time.Second).Should(BeTrue())
	},
		Entry("with a migration then no migration", libvirt.DOMAIN_JOB_BOUNDED, libvirt.DOMAIN_JOB_NONE,
			libvirt.DOMAIN_JOB_OPERATION_MIGRATION_IN, libvirt.DOMAIN_JOB_OPERATION_UNKNOWN,
			nil),
		Entry("with an error then no migration", libvirt.DOMAIN_JOB_NONE, libvirt.DOMAIN_JOB_NONE,
			libvirt.DOMAIN_JOB_OPERATION_UNKNOWN, libvirt.DOMAIN_JOB_OPERATION_UNKNOWN,
			fmt.Errorf("error")),
		Entry("with a migration then another operation", libvirt.DOMAIN_JOB_BOUNDED, libvirt.DOMAIN_JOB_BOUNDED,
			libvirt.DOMAIN_JOB_OPERATION_MIGRATION_IN, libvirt.DOMAIN_JOB_OPERATION_BACKUP,
			nil),
		Entry("with an error then another operation", libvirt.DOMAIN_JOB_NONE, libvirt.DOMAIN_JOB_BOUNDED,
			libvirt.DOMAIN_JOB_OPERATION_UNKNOWN, libvirt.DOMAIN_JOB_OPERATION_BACKUP,
			fmt.Errorf("error")))
})
