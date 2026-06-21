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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var _ = Describe("migrationMonitor", func() {
	var mockLibvirt *testing.Libvirt
	var ctrl *gomock.Controller
	var testVirtShareDir string
	var testEphemeralDiskDir string
	var metadataCache *metadata.Cache
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)
	ephemeralDiskCreatorMock := &fake.MockEphemeralDiskImageCreator{}

	newLibvirtDomainManagerDefault := func() (DomainManager, error) {
		return NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, virtconfig.DefaultARCHOVMFPath, ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false, false, nil, v1.KvmHypervisorName, nil, "", false, false)
	}

	BeforeEach(func() {
		testVirtShareDir = fmt.Sprintf("fake-virt-share-%d", GinkgoRandomSeed())
		testEphemeralDiskDir = fmt.Sprintf("fake-ephemeral-disk-%d", GinkgoRandomSeed())
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)
		metadataCache = metadata.NewCache()
		mockLibvirt.DomainEXPECT().GetBlockInfo(gomock.Any(), gomock.Any()).AnyTimes().Return(&libvirt.DomainBlockInfo{Capacity: 0}, nil)
	})

	mockDomainWithFreeExpectation := func(_ string) (cli.VirDomain, error) {
		mockLibvirt.DomainEXPECT().Free()
		return mockLibvirt.VirtDomain, nil
	}

	Context("test migration monitor", func() {
		It("migration should be canceled if it's not progressing", func() {
			migrationDone := make(chan struct{})
			fake_jobinfo := &libvirt.DomainJobInfo{
				Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
				DataRemaining:    32479827394,
				DataRemainingSet: true,
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         2,
				CompletionTimeoutPerGiB: 300,
			}

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			now := metav1.Now()
			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobInfo().Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_UNBOUNDED}, nil)
			mockLibvirt.DomainEXPECT().AbortJob().DoAndReturn(func() error {
				go func() {
					time.Sleep(time.Second)
					close(migrationDone)
				}()
				return nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		})
		It("migration abort should be retried after transient failure", func() {
			migrationDone := make(chan struct{})
			fake_jobinfo := &libvirt.DomainJobInfo{
				Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
				DataRemaining:    32479827394,
				DataRemainingSet: true,
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         2,
				CompletionTimeoutPerGiB: 300,
			}

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			now := metav1.Now()
			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			abortStatus := func() string {
				m, _ := metadataCache.Migration.Load()
				return m.AbortStatus
			}

			Expect(abortStatus()).To(Equal(""))

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)

			// First attempt: cancelMigration sets AbortInProgress, GetJobInfo fails → AbortFailed
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobInfo().DoAndReturn(func() (*libvirt.DomainJobInfo, error) {
				Expect(abortStatus()).To(Equal(string(v1.MigrationAbortInProgress)))
				return nil, fmt.Errorf("transient error")
			})

			// Reaching this point proves the monitor saw AbortFailed and retried.
			// cancelMigration has already overwritten it back to AbortInProgress.
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobInfo().DoAndReturn(func() (*libvirt.DomainJobInfo, error) {
				Expect(abortStatus()).To(Equal(string(v1.MigrationAbortInProgress)))
				return &libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_UNBOUNDED}, nil
			})
			mockLibvirt.DomainEXPECT().AbortJob().DoAndReturn(func() error {
				go func() {
					time.Sleep(time.Second)
					close(migrationDone)
				}()
				return nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
			Eventually(abortStatus, 2*time.Second, 100*time.Millisecond).Should(Equal(string(v1.MigrationAbortSucceeded)))
		})
		DescribeTable("migration should be canceled when GetJobStats does not report progress", func(stubGetJobStats func()) {
			migrationDone := make(chan struct{})

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 150,
			}

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			now := metav1.Now()
			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			stubGetJobStats()
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobInfo().Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_UNBOUNDED}, nil)
			mockLibvirt.DomainEXPECT().AbortJob().DoAndReturn(func() error {
				close(migrationDone)
				return nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		},
			Entry("because GetJobStats keeps failing", func() {
				mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(nil, fmt.Errorf("persistent stats error"))
			}),
			Entry("because GetJobStats always returns JOB_NONE", func() {
				mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(
					&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_NONE}, nil)
			}),
		)
		It("migration should be canceled if timeout has been reached", func() {
			migrationDone := make(chan struct{})
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
				}
			}()

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 150,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			now := metav1.Now()
			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobInfo().Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_UNBOUNDED}, nil)
			mockLibvirt.DomainEXPECT().AbortJob().DoAndReturn(func() error {
				go func() {
					time.Sleep(time.Second)
					close(migrationDone)
				}()
				return nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		})
		It("migration should switch to PostCopy", func() {
			migrationDone := make(chan struct{})
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and close the channel otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					close(migrationDone)
					return &libvirt.DomainJobInfo{}
				}

				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
				}
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 1,
				AllowPostCopy:           true,
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockLibvirt.DomainEXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).Times(1).Return(nil)

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		})

		It("migration should switch to PostCopy eventually", func() {
			migrationDone := make(chan struct{})
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and close the channel otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					close(migrationDone)
					return &libvirt.DomainJobInfo{}
				}

				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
				}
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 1,
				AllowPostCopy:           true,
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})

			counter := 0
			mockLibvirt.DomainEXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).Times(2).DoAndReturn(func(flag uint32) error {
				if counter == 0 {
					counter += 1
					return libvirt.Error{

						Code:    1,
						Domain:  1,
						Message: "internal error: unable to execute QEMU command 'migrate-start-postcopy': Postcopy must be started after migration has been started",
						Level:   libvirt.ERR_ERROR,
					}
				}
				return nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		})
		It("migration should switch to Paused if AllowWorkloadDisruption is allowed and PostCopy is not", func() {
			migrationDone := make(chan struct{})
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and close the channel otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					close(migrationDone)
					return &libvirt.DomainJobInfo{}
				}

				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
				}
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 1,
				AllowPostCopy:           false,
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				paused: pausedVMIs{
					paused: make(map[types.UID]bool),
				},
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockLibvirt.DomainEXPECT().Suspend().Times(1).Return(nil)

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		})
		It("migration should be canceled if Paused workload didn't migrate until timeout was reached", func() {
			migrationDone := make(chan struct{})
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
				}
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 1,
				AllowPostCopy:           false,
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}
			now := metav1.Now()
			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				paused: pausedVMIs{
					paused: make(map[types.UID]bool),
				},
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockLibvirt.DomainEXPECT().Suspend().Times(1).Return(nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobInfo().Return(&libvirt.DomainJobInfo{Type: libvirt.DOMAIN_JOB_UNBOUNDED}, nil)
			mockLibvirt.DomainEXPECT().AbortJob().DoAndReturn(func() error {
				go func() {
					time.Sleep(time.Second)
					close(migrationDone)
				}()
				return nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			monitor.startMonitor(make(chan error, 1))
		})
		// This is incomplete as it is not verifying that we abort. Previously it wasn't even testing anything at all
		It("migration should be canceled when requested", func() {
			migrationUid := types.UID("111222333")

			now := metav1.Now()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   migrationUid,
				StartTimestamp: &now,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			// These lines do not test anything but needs to be here because otherwise test will panic
			mockLibvirt.DomainEXPECT().AbortJob().MaxTimes(1)
			migrationInProgress := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(32479827394),
					DataRemainingSet: true,
				}
			}()
			mockLibvirt.DomainEXPECT().GetJobInfo().MaxTimes(1).Return(migrationInProgress, nil)

			manager, _ := newLibvirtDomainManagerDefault()

			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			migrationMetadata.UID = migrationUid
			metadataCache.Migration.Store(migrationMetadata)

			Expect(manager.CancelVMIMigration(vmi)).To(Succeed())

			// Allow the aync-abort (goroutine) to be processed before finishing.
			// This is required in order to allow the expected calls to occur.
			time.Sleep(2 * time.Second)

			migration, _ := metadataCache.Migration.Load()
			Expect(migration.AbortStatus).To(Equal(string(v1.MigrationAbortSucceeded)))
		})

		It("shouldn't be able to call cancel migration more than once", func() {
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			secondBefore := metav1.Time{Time: now.Add(-time.Second)}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "111222333",
				StartTimestamp: &now,
			}

			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.UID = vmi.Status.MigrationState.MigrationUID
			migrationMetadata.AbortStatus = string(v1.MigrationAbortInProgress)
			migrationMetadata.StartTimestamp = &secondBefore
			metadataCache.Migration.Store(migrationMetadata)

			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			manager, _ := newLibvirtDomainManagerDefault()
			Expect(manager.CancelVMIMigration(vmi)).To(Succeed())
		})
		It("monitor should exit when migration done channel is closed", func() {
			migrationDone := make(chan struct{})

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 150,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.UID = vmi.Status.MigrationState.MigrationUID
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).DoAndReturn(
				func(flags libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
					close(migrationDone)
					return &libvirt.DomainJobInfo{
						Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
						DataRemainingSet: true,
						DataRemaining:    uint64(32479827777),
					}, nil
				},
			)

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			done := make(chan struct{})
			go func() {
				monitor.startMonitor(make(chan error, 1))
				close(done)
			}()
			Eventually(done, 5*time.Second).Should(BeClosed())
		})
		It("monitor should signal error on ready channel if it fails to start", func() {
			migrationDone := make(chan struct{})

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 150,
			}
			vmi := newVMI(testNamespace, testVmName)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, fmt.Errorf("domain not found"))

			monitor := newMigrationMonitor(vmi, manager, options, migrationDone)
			ready := make(chan error, 1)
			monitor.startMonitor(ready)
			Expect(ready).To(Receive(MatchError(ContainSubstring("domain not found"))))
		})
	})
})
