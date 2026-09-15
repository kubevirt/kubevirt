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
	"encoding/xml"
	"fmt"
	"os"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

var _ = Describe("External snapshot", func() {
	const (
		overlayDir = "/var/run/kubevirt/hotplug-disks/scratch"
		domainName = "default_testvmi"

		rootVolume      = "rootdisk"
		dataVolume      = "datadisk"
		containerVolume = "containerdisk"
		scratchVolume   = "overlay-scratch"
	)

	snapshotFlags := libvirt.DOMAIN_SNAPSHOT_CREATE_DISK_ONLY |
		libvirt.DOMAIN_SNAPSHOT_CREATE_ATOMIC |
		libvirt.DOMAIN_SNAPSHOT_CREATE_NO_METADATA

	// The domain as virt-launcher renders it: two snapshottable disks, a container
	// disk that must not be snapshotted, and the scratch volume, a utility volume
	// and so not part of vmi.Spec.Volumes at all.
	domainDisk := func(volumeName, target, file string) api.Disk {
		return api.Disk{
			Alias:  api.NewUserDefinedAlias(volumeName),
			Target: api.DiskTarget{Device: target},
			Source: api.DiskSource{File: file},
		}
	}
	allDisks := func(rootSource, dataSource string) []api.Disk {
		return []api.Disk{
			domainDisk(rootVolume, "vda", rootSource),
			domainDisk(dataVolume, "vdb", dataSource),
			domainDisk(containerVolume, "vdc", "/var/run/kubevirt/container-disks/disk_0.img"),
			domainDisk(scratchVolume, "vdd", overlayDir+"/disk.img"),
		}
	}
	basePaths := []string{"/var/run/kubevirt-private/vmi-disks/rootdisk/disk.img",
		"/var/run/kubevirt-private/vmi-disks/datadisk/disk.img"}
	overlayPaths := []string{overlayDir + "/ovl-vda.qcow2", overlayDir + "/ovl-vdb.qcow2"}

	domainXMLOf := func(disks []api.Disk) string {
		domainSpec := &api.DomainSpec{}
		domainSpec.Devices.Disks = disks
		domainXML, err := xml.Marshal(domainSpec)
		Expect(err).ToNot(HaveOccurred())
		return string(domainXML)
	}
	onBaseXML := domainXMLOf(allDisks(basePaths[0], basePaths[1]))
	onOverlayXML := domainXMLOf(allDisks(overlayPaths[0], overlayPaths[1]))
	partialXML := domainXMLOf(allDisks(overlayPaths[0], basePaths[1]))

	var (
		ctrl          *gomock.Controller
		mockConn      *cli.MockConnection
		mockDomain    *cli.MockVirDomain
		manager       *StorageManager
		metadataCache *metadata.Cache
		vmi           *v1.VirtualMachineInstance
	)

	isMounted := verifyOverlayDirMounted

	overlayPhase := func() api.SnapshotOverlayPhase {
		overlay, exists := metadataCache.SnapshotOverlay.Load()
		if !exists {
			return ""
		}
		return overlay.Phase
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		metadataCache = metadata.NewCache()
		manager = NewStorageManager(mockConn, metadataCache, nil)

		vmi = libvmi.New(
			libvmi.WithName("testvmi"),
			libvmi.WithNamespace("default"),
			libvmi.WithDataVolume(rootVolume, "root-dv"),
			libvmi.WithPersistentVolumeClaim(dataVolume, "data-pvc"),
			libvmi.WithContainerDisk(containerVolume, "registry:5000/disk"),
		)
		vmi.Status.Conditions = []v1.VirtualMachineInstanceCondition{{
			Type:   v1.VirtualMachineInstanceAgentConnected,
			Status: k8sv1.ConditionTrue,
		}}

		mockConn.EXPECT().LookupDomainByName(domainName).Return(mockDomain, nil).AnyTimes()
		mockDomain.EXPECT().Free().Return(nil).AnyTimes()

		verifyOverlayDirMounted = func(string) error { return nil }
		DeferCleanup(func() { verifyOverlayDirMounted = isMounted })

		// A snapshot leaves a monitor watching the scratch volume, parked here
		// because these specs are about taking the snapshot.
		manager.overlayUsageInterval = time.Hour
	})

	Context("the transaction", func() {
		It("should freeze the guest, snapshot every snapshottable disk and thaw again", func() {
			var snapshotXML string
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"thawed"}`, nil),
				mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().CreateSnapshotXML(gomock.Any(), snapshotFlags).
					DoAndReturn(func(createdXML string, _ libvirt.DomainSnapshotCreateFlags) (*libvirt.DomainSnapshot, error) {
						snapshotXML = createdXML
						return nil, nil
					}),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"frozen"}`, nil),
				mockDomain.EXPECT().FSThaw(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayReady))
			Expect(snapshotXML).To(ContainSubstring(`<disk name="vda" snapshot="external"><source file="` + overlayPaths[0] + `">`))
		})

		It("should report Ready only once the domain is confirmed to be on the overlays", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"thawed"}`, nil),
				mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().CreateSnapshotXML(gomock.Any(), snapshotFlags).Return(nil, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"frozen"}`, nil),
				mockDomain.EXPECT().FSThaw(nil, uint32(0)).Return(nil),
				// Only one of the two disks made it onto its overlay.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(partialXML, nil).Times(2),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
		})

		DescribeTable("should not snapshot a guest that cannot be quiesced", func(prepare func(), expectedMessage string) {
			prepare()
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlaySnapshotFailed))
			overlay, _ := metadataCache.SnapshotOverlay.Load()
			Expect(overlay.Message).To(ContainSubstring(expectedMessage))
			Expect(overlay.Message).To(ContainSubstring(quiesceHint))
		},
			Entry("because the domain is paused", func() {
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			}, "paused domain"),
			Entry("because no guest agent is connected", func() {
				vmi.Status.Conditions = nil
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			}, "without a connected guest agent"),
			Entry("because the guest agent is disconnected", func() {
				vmi.Status.Conditions[0].Status = k8sv1.ConditionFalse
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			}, "without a connected guest agent"),
		)

		It("should refuse to snapshot a migrating VMI", func() {
			now := metav1.Now()
			metadataCache.Migration.Store(api.MigrationMetadata{StartTimestamp: &now})

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(MatchError(ContainSubstring("during migration")))
			Consistently(overlayPhase).Should(BeEmpty())
		})
	})

	Context("a failed transaction", func() {
		It("should report SnapshotFailed when the guest cannot be frozen", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"thawed"}`, nil),
				mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(fmt.Errorf("guest agent timed out")),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlaySnapshotFailed))
			overlay, _ := metadataCache.SnapshotOverlay.Load()
			Expect(overlay.Message).To(ContainSubstring("guest agent timed out"))
			Expect(overlay.Message).To(ContainSubstring(quiesceHint))
		})

		It("should report SnapshotFailed when the scratch volume is not mounted", func() {
			verifyOverlayDirMounted = func(string) error { return fmt.Errorf("not a mount point") }
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlaySnapshotFailed))
		})

		It("should report CommitFailed when a disk is left on an overlay", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"thawed"}`, nil),
				mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().CreateSnapshotXML(gomock.Any(), snapshotFlags).
					Return(nil, fmt.Errorf("transaction aborted")),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"frozen"}`, nil),
				mockDomain.EXPECT().FSThaw(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(partialXML, nil),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
		})

		It("should report CommitFailed when the domain cannot be inspected", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"thawed"}`, nil),
				mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().CreateSnapshotXML(gomock.Any(), snapshotFlags).
					Return(nil, fmt.Errorf("transaction aborted")),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"frozen"}`, nil),
				mockDomain.EXPECT().FSThaw(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return("", fmt.Errorf("libvirt is gone")),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
		})
	})

	Context("idempotency", func() {
		DescribeTable("should not start a second transaction while the overlays are", func(phase api.SnapshotOverlayPhase) {
			metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{Phase: phase})

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Consistently(overlayPhase, 100*time.Millisecond).Should(Equal(phase))
		},
			Entry("being taken", api.SnapshotOverlayInProgress),
			Entry("in use", api.SnapshotOverlayReady),
			Entry("being committed", api.SnapshotOverlayCommitting),
			Entry("left behind by a failed commit", api.SnapshotOverlayCommitFailed),
		)

		It("should retry after a transaction that left nothing behind", func() {
			metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{
				Phase:   api.SnapshotOverlaySnapshotFailed,
				Message: "guest agent timed out",
			})
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"thawed"}`, nil),
				mockDomain.EXPECT().FSFreeze(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().CreateSnapshotXML(gomock.Any(), snapshotFlags).Return(nil, nil),
				mockConn.EXPECT().QemuAgentCommand(gomock.Any(), domainName).Return(`{"return":"frozen"}`, nil),
				mockDomain.EXPECT().FSThaw(nil, uint32(0)).Return(nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
			)

			Expect(manager.ExternalSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayReady))
			overlay, _ := metadataCache.SnapshotOverlay.Load()
			Expect(overlay.Message).To(BeEmpty())
			Expect(overlay.StartTimestamp).ToNot(BeNil())
		})
	})

	Context("the snapshot XML", func() {
		It("should redirect the snapshottable disks and skip the rest", func() {
			snapshotXML, err := buildSnapshotXML(vmi, allDisks(basePaths[0], basePaths[1]), overlayDir)
			Expect(err).ToNot(HaveOccurred())

			domainSnapshot := &api.DomainSnapshot{}
			Expect(xml.Unmarshal([]byte(snapshotXML), domainSnapshot)).To(Succeed())
			Expect(domainSnapshot.SnapshotDisks.Disks).To(ConsistOf(
				api.SnapshotDisk{Name: "vda", Snapshot: "external", Source: &api.SnapshotDiskSource{File: overlayPaths[0]}},
				api.SnapshotDisk{Name: "vdb", Snapshot: "external", Source: &api.SnapshotDiskSource{File: overlayPaths[1]}},
				api.SnapshotDisk{Name: "vdc", Snapshot: "no"},
				api.SnapshotDisk{Name: "vdd", Snapshot: "no"},
			))
		})

		It("should snapshot a memory dump volume", func() {
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name:         "memorydump",
				VolumeSource: v1.VolumeSource{MemoryDump: &v1.MemoryDumpVolumeSource{}},
			})
			disks := append(allDisks(basePaths[0], basePaths[1]),
				domainDisk("memorydump", "vde", "/var/run/kubevirt-private/vmi-disks/memorydump/disk.img"))

			Expect(snapshottableDisks(vmi, disks)).To(HaveLen(3))
		})

		It("should ignore disks with no alias or no target", func() {
			disks := []api.Disk{
				{Target: api.DiskTarget{Device: "vda"}},
				{Alias: api.NewUserDefinedAlias(rootVolume)},
			}

			Expect(snapshottableDisks(vmi, disks)).To(BeEmpty())
		})
	})

	Context("the overlay directory", func() {
		It("should be rejected when it is not a mount point", func() {
			scratchDir, err := os.MkdirTemp("", "overlay")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() { os.RemoveAll(scratchDir) })

			Expect(isMounted(scratchDir)).To(MatchError(ContainSubstring("is not a mount point")))
		})

		It("should be rejected when it does not exist", func() {
			Expect(isMounted("/no/such/overlay/dir")).To(MatchError(ContainSubstring("failed to stat")))
		})
	})
})

var _ = Describe("Snapshot overlay commit", func() {
	const (
		overlayDir = "/var/run/kubevirt/hotplug-disks/scratch"
		domainName = "default_testvmi"

		rootVolume      = "rootdisk"
		dataVolume      = "datadisk"
		containerVolume = "containerdisk"
		scratchVolume   = "overlay-scratch"

		registrationID = 7
	)

	domainDisk := func(volumeName, target, file string, mirror *api.DiskMirror) api.Disk {
		return api.Disk{
			Alias:  api.NewUserDefinedAlias(volumeName),
			Target: api.DiskTarget{Device: target},
			Source: api.DiskSource{File: file},
			Mirror: mirror,
		}
	}
	allDisks := func(rootSource, dataSource string, mirror *api.DiskMirror) []api.Disk {
		return []api.Disk{
			domainDisk(rootVolume, "vda", rootSource, mirror),
			domainDisk(dataVolume, "vdb", dataSource, mirror),
			domainDisk(containerVolume, "vdc", "/var/run/kubevirt/container-disks/disk_0.img", nil),
			domainDisk(scratchVolume, "vdd", overlayDir+"/disk.img", nil),
		}
	}
	basePaths := []string{"/var/run/kubevirt-private/vmi-disks/rootdisk/disk.img",
		"/var/run/kubevirt-private/vmi-disks/datadisk/disk.img"}
	overlayPaths := []string{overlayDir + "/ovl-vda.qcow2", overlayDir + "/ovl-vdb.qcow2"}

	domainXMLOf := func(disks []api.Disk) string {
		domainSpec := &api.DomainSpec{}
		domainSpec.Devices.Disks = disks
		domainXML, err := xml.Marshal(domainSpec)
		Expect(err).ToNot(HaveOccurred())
		return string(domainXML)
	}
	onBaseXML := domainXMLOf(allDisks(basePaths[0], basePaths[1], nil))
	onOverlayXML := domainXMLOf(allDisks(overlayPaths[0], overlayPaths[1], nil))
	mirrorReadyXML := domainXMLOf(allDisks(overlayPaths[0], overlayPaths[1], &api.DiskMirror{Ready: "yes"}))

	var (
		ctrl          *gomock.Controller
		mockConn      *cli.MockConnection
		mockDomain    *cli.MockVirDomain
		manager       *StorageManager
		metadataCache *metadata.Cache
		vmi           *v1.VirtualMachineInstance

		removed []string
	)

	// blockJobCallback is what the running commit registered with libvirt, and
	// fireEvent delivers an event to it the way the libvirt event loop would.
	// Held atomically because the commit registers it from its own goroutine.
	blockJobCallback := &atomic.Pointer[libvirt.DomainEventBlockJobCallback]{}
	fireEvent := func(disk string, status libvirt.ConnectDomainEventBlockJobStatus) {
		callback := blockJobCallback.Load()
		Expect(callback).ToNot(BeNil())
		(*callback)(nil, nil, &libvirt.DomainEventBlockJob{Disk: disk, Status: status})
	}

	removeFile := removeOverlayFile
	gracePeriod, readyTimeout, pollInterval := commitConvergenceGracePeriod, commitReadyTimeout, commitReadyPollInterval

	overlayPhase := func() api.SnapshotOverlayPhase {
		overlay, exists := metadataCache.SnapshotOverlay.Load()
		if !exists {
			return ""
		}
		return overlay.Phase
	}
	overlayMessage := func() string {
		overlay, _ := metadataCache.SnapshotOverlay.Load()
		return overlay.Message
	}

	// expectCommitOf sets up the block job calls of a disk that commits cleanly:
	// no job running, a commit started with ACTIVE alone, and a pivot.
	expectCommitOf := func(target string) {
		mockDomain.EXPECT().GetBlockJobInfo(target, libvirt.DomainBlockJobInfoFlags(0)).
			Return(&libvirt.DomainBlockJobInfo{}, nil)
		mockDomain.EXPECT().BlockCommit(target, "", "", uint64(0), libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE).Return(nil)
		mockDomain.EXPECT().BlockJobAbort(target, libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).Return(nil)
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		metadataCache = metadata.NewCache()
		manager = NewStorageManager(mockConn, metadataCache, nil)

		vmi = libvmi.New(
			libvmi.WithName("testvmi"),
			libvmi.WithNamespace("default"),
			libvmi.WithDataVolume(rootVolume, "root-dv"),
			libvmi.WithPersistentVolumeClaim(dataVolume, "data-pvc"),
			libvmi.WithContainerDisk(containerVolume, "registry:5000/disk"),
		)

		mockConn.EXPECT().LookupDomainByName(domainName).Return(mockDomain, nil).AnyTimes()
		mockDomain.EXPECT().Free().Return(nil).AnyTimes()

		blockJobCallback.Store(nil)
		mockConn.EXPECT().VolatileDomainEventBlockJobRegister(gomock.Any()).
			DoAndReturn(func(callback libvirt.DomainEventBlockJobCallback) (int, error) {
				blockJobCallback.Store(&callback)
				return registrationID, nil
			}).AnyTimes()
		mockConn.EXPECT().DomainEventDeregister(registrationID).Return(nil).AnyTimes()

		removed = nil
		removeOverlayFile = func(path string) error {
			removed = append(removed, path)
			return nil
		}

		// The commit is a multi-minute operation in production. Shortened here so a
		// spec does not wait on a poll, and lengthened again in the specs about
		// pausing the guest or giving up, where the timing is the thing under test.
		commitConvergenceGracePeriod = time.Hour
		commitReadyTimeout = time.Hour
		commitReadyPollInterval = 5 * time.Millisecond

		DeferCleanup(func() {
			removeOverlayFile = removeFile
			commitConvergenceGracePeriod, commitReadyTimeout, commitReadyPollInterval = gracePeriod, readyTimeout, pollInterval
		})

		metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{Phase: api.SnapshotOverlayReady})
	})

	Context("the commit", func() {
		It("should commit every snapshottable disk back onto its base and clear the state", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				// Readiness of each disk, then the two verification reads.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			expectCommitOf("vda")
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
			Expect(removed).To(ConsistOf(overlayPaths[0], overlayPaths[1]))
		})

		It("should commit with ACTIVE alone, never asking libvirt to delete the overlay", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			var flags []libvirt.DomainBlockCommitFlags
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).Return(&libvirt.DomainBlockJobInfo{}, nil).Times(2)
			mockDomain.EXPECT().BlockCommit(gomock.Any(), "", "", uint64(0), gomock.Any()).
				DoAndReturn(func(_, _, _ string, _ uint64, f libvirt.DomainBlockCommitFlags) error {
					flags = append(flags, f)
					return nil
				}).Times(2)
			mockDomain.EXPECT().BlockJobAbort(gomock.Any(), libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).Return(nil).Times(2)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
			Expect(flags).To(HaveEach(libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE))
			for _, flag := range flags {
				// We unlink the overlay after the disks are verified back on base.
				// Handing DELETE to libvirt would drop it at the pivot, before
				// anything has confirmed the pivot happened.
				Expect(flag & libvirt.DOMAIN_BLOCK_COMMIT_DELETE).To(BeZero())
			}
		})

		It("should pivot on a block job ready event without waiting for a poll", func() {
			commitReadyPollInterval = time.Hour

			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				// Not ready on the initial read of either disk, so only the event
				// can move the commit along.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			expectCommitOf("vda")
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(blockJobCallback.Load).ShouldNot(BeNil())
			Eventually(func() api.SnapshotOverlayPhase {
				fireEvent("vda", libvirt.DOMAIN_BLOCK_JOB_READY)
				fireEvent("vdb", libvirt.DOMAIN_BLOCK_JOB_READY)
				return overlayPhase()
			}).Should(BeEmpty())
		})

		It("should not read the domain XML on a poll where the job still has bytes outstanding", func() {
			// vda reports progress for twenty polls before it drains. Each of
			// those polls would read the XML without the gate, and the exact
			// expectations below are what catches it.
			const outstanding = 20
			polls := 0

			gomock.InOrder(
				// The disk list, then vda not ready yet.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil).Times(2),
				// vda once it has drained, then vdb ready on its first read.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)

			running := map[string]bool{}
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).
				DoAndReturn(func(target string, _ libvirt.DomainBlockJobInfoFlags) (*libvirt.DomainBlockJobInfo, error) {
					if !running[target] {
						return &libvirt.DomainBlockJobInfo{}, nil
					}
					polls++
					info := &libvirt.DomainBlockJobInfo{
						Type: libvirt.DOMAIN_BLOCK_JOB_TYPE_ACTIVE_COMMIT,
						Cur:  storagetypes.MiB,
						End:  2 * storagetypes.MiB,
					}
					if polls > outstanding {
						info.Cur = info.End
					}
					return info, nil
				}).AnyTimes()
			mockDomain.EXPECT().BlockCommit(gomock.Any(), "", "", uint64(0), libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE).
				DoAndReturn(func(target, _, _ string, _ uint64, _ libvirt.DomainBlockCommitFlags) error {
					running[target] = true
					return nil
				}).Times(2)
			mockDomain.EXPECT().BlockJobAbort(gomock.Any(), libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).
				DoAndReturn(func(target string, _ libvirt.DomainBlockJobAbortFlags) error {
					delete(running, target)
					return nil
				}).Times(2)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
			Expect(polls).To(BeNumerically(">", outstanding))
		})

		It("should skip a disk that is already back on its base image", func() {
			// vda committed on an earlier attempt, only vdb is left.
			partialXML := domainXMLOf(allDisks(basePaths[0], overlayPaths[1], nil))
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(partialXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
		})

		It("should wait for a commit that is already running instead of starting a second", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).
				Return(&libvirt.DomainBlockJobInfo{Type: libvirt.DOMAIN_BLOCK_JOB_TYPE_ACTIVE_COMMIT}, nil).Times(2)
			mockDomain.EXPECT().BlockCommit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			mockDomain.EXPECT().BlockJobAbort(gomock.Any(), libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).Return(nil).Times(2)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
		})

		It("should treat a pivot of a job that already finished as done", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil),
				// The abort fails, and the disk turns out to be on base already.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).Return(&libvirt.DomainBlockJobInfo{}, nil).Times(2)
			mockDomain.EXPECT().BlockCommit(gomock.Any(), "", "", uint64(0), libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE).Return(nil).Times(2)
			gomock.InOrder(
				mockDomain.EXPECT().BlockJobAbort("vda", libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).
					Return(fmt.Errorf("domain does not have an active block job")),
				mockDomain.EXPECT().BlockJobAbort("vdb", libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).Return(nil),
			)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
		})
	})

	Context("convergence", func() {
		// A commit that only converges once the guest stops writing cannot be a
		// fixed sequence of reads, so these specs answer from the state the commit
		// reached: the mirror goes ready when the guest is paused, and the disks
		// read back on base once both have pivoted.
		convergesWhenPaused := func(converged *bool) {
			pivots := 0
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).DoAndReturn(func(libvirt.DomainXMLFlags) (string, error) {
				switch {
				case pivots == len(overlayPaths):
					return onBaseXML, nil
				case *converged:
					return mirrorReadyXML, nil
				default:
					return onOverlayXML, nil
				}
			}).AnyTimes()
			// Not Times(2): the progress log queries the job on every poll too.
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).Return(&libvirt.DomainBlockJobInfo{}, nil).AnyTimes()
			mockDomain.EXPECT().BlockCommit(gomock.Any(), "", "", uint64(0), libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE).Return(nil).Times(2)
			mockDomain.EXPECT().BlockJobAbort(gomock.Any(), libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT).
				DoAndReturn(func(string, libvirt.DomainBlockJobAbortFlags) error {
					pivots++
					return nil
				}).Times(2)
		}

		It("should pause the guest when the commit does not converge, and resume it after the pivot", func() {
			commitConvergenceGracePeriod = 10 * time.Millisecond

			converged := false
			convergesWhenPaused(&converged)

			resumed := make(chan struct{})
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().Suspend().DoAndReturn(func() error { converged = true; return nil })
			mockDomain.EXPECT().Resume().DoAndReturn(func() error { close(resumed); return nil })

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
			Expect(resumed).To(BeClosed())
		})

		It("should not resume a guest it did not pause", func() {
			commitConvergenceGracePeriod = 10 * time.Millisecond

			// Somebody else paused the guest, and the commit converges on its own.
			converged := false
			convergesWhenPaused(&converged)

			mockDomain.EXPECT().GetState().
				DoAndReturn(func() (libvirt.DomainState, int, error) {
					converged = true
					return libvirt.DOMAIN_PAUSED, 1, nil
				}).MinTimes(1)
			mockDomain.EXPECT().Suspend().Times(0)
			mockDomain.EXPECT().Resume().Times(0)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
		})

		It("should report CommitFailed when a disk never converges", func() {
			commitReadyTimeout = 30 * time.Millisecond
			commitConvergenceGracePeriod = time.Hour

			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil).MinTimes(1)
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).Return(&libvirt.DomainBlockJobInfo{}, nil).AnyTimes()
			mockDomain.EXPECT().BlockCommit("vda", "", "", uint64(0), libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE).Return(nil)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
			Expect(overlayMessage()).To(ContainSubstring("did not converge"))
		})
	})

	Context("a failed commit", func() {
		It("should report CommitFailed when the block commit cannot be started", func() {
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil)
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).Return(&libvirt.DomainBlockJobInfo{}, nil)
			mockDomain.EXPECT().BlockCommit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(fmt.Errorf("no space left on device"))

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
			Expect(overlayMessage()).To(ContainSubstring("no space left on device"))
		})

		It("should report CommitFailed on a block job failed event", func() {
			commitReadyPollInterval = time.Hour
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
			)
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).Return(&libvirt.DomainBlockJobInfo{}, nil)
			mockDomain.EXPECT().BlockCommit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(blockJobCallback.Load).ShouldNot(BeNil())
			Eventually(func() api.SnapshotOverlayPhase {
				fireEvent("vda", libvirt.DOMAIN_BLOCK_JOB_FAILED)
				return overlayPhase()
			}).Should(Equal(api.SnapshotOverlayCommitFailed))
		})

		It("should refuse a disk that is busy with an unrelated block job", func() {
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil)
			mockDomain.EXPECT().GetBlockJobInfo(gomock.Any(), gomock.Any()).
				Return(&libvirt.DomainBlockJobInfo{Type: libvirt.DOMAIN_BLOCK_JOB_TYPE_COPY}, nil)
			mockDomain.EXPECT().BlockCommit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
			Expect(overlayMessage()).To(ContainSubstring("busy with a block job"))
		})

		It("should report CommitFailed and keep the overlays when a disk is left on one", func() {
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				// The final verification still sees vda on its overlay.
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(domainXMLOf(allDisks(overlayPaths[0], basePaths[1], nil)), nil),
			)
			expectCommitOf("vda")
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
			Expect(removed).To(BeEmpty())
		})

		It("should report CommitFailed when an overlay cannot be removed", func() {
			removeOverlayFile = func(string) error { return fmt.Errorf("read-only file system") }
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			expectCommitOf("vda")
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitFailed))
			Expect(overlayMessage()).To(ContainSubstring("read-only file system"))
		})

		It("should tolerate an overlay that is already gone", func() {
			removeOverlayFile = func(string) error { return os.ErrNotExist }
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			expectCommitOf("vda")
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
		})
	})

	Context("idempotency", func() {
		DescribeTable("should not start a commit", func(phase api.SnapshotOverlayPhase) {
			metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{Phase: phase})

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Consistently(overlayPhase, 100*time.Millisecond, 10*time.Millisecond).Should(Equal(phase))
		},
			Entry("when the VMI never had overlays", api.SnapshotOverlayPhase("")),
			Entry("when the snapshot is still being taken", api.SnapshotOverlayInProgress),
			Entry("when a commit is already running", api.SnapshotOverlayCommitting),
			Entry("when the snapshot failed and left nothing behind", api.SnapshotOverlaySnapshotFailed),
		)

		It("should retry a commit that failed", func() {
			metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{
				Phase:   api.SnapshotOverlayCommitFailed,
				Message: "no space left on device",
			})
			gomock.InOrder(
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onOverlayXML, nil),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(mirrorReadyXML, nil).Times(2),
				mockDomain.EXPECT().GetXMLDesc(gomock.Any()).Return(onBaseXML, nil).Times(2),
			)
			expectCommitOf("vda")
			expectCommitOf("vdb")

			Expect(manager.CommitSnapshot(vmi, overlayDir)).To(Succeed())

			Eventually(overlayPhase).Should(BeEmpty())
			Expect(overlayMessage()).To(BeEmpty())
		})

		It("should refuse to commit while the VMI is migrating", func() {
			now := metav1.Now()
			metadataCache.Migration.Store(api.MigrationMetadata{StartTimestamp: &now})

			Expect(manager.CommitSnapshot(vmi, overlayDir)).ToNot(Succeed())

			Expect(overlayPhase()).To(Equal(api.SnapshotOverlayReady))
		})
	})
})

var _ = Describe("Overlay usage monitor", func() {
	const (
		overlayDir = "/var/run/kubevirt/hotplug-disks/scratch"
		domainName = "default_testvmi"
	)

	var (
		ctrl          *gomock.Controller
		mockConn      *cli.MockConnection
		manager       *StorageManager
		metadataCache *metadata.Cache
		vmi           *v1.VirtualMachineInstance

		// usage is what the next measurement of the scratch volume returns, and
		// measured is fed once per measurement so a spec can wait for the next one.
		usage    func() (int64, error)
		measured chan struct{}
	)

	overlayPhase := func() api.SnapshotOverlayPhase {
		overlay, exists := metadataCache.SnapshotOverlay.Load()
		if !exists {
			return ""
		}
		return overlay.Phase
	}
	overlayMessage := func() string {
		overlay, _ := metadataCache.SnapshotOverlay.Load()
		return overlay.Message
	}

	// runMonitor starts the monitor and hands back a channel that closes when it
	// stops watching the overlays. A monitor left running is stopped the way
	// anything stops it, by taking the overlays off the guest.
	runMonitor := func() chan struct{} {
		stopped := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(stopped)
			manager.monitorOverlayUsage(vmi, overlayDir)
		}()

		DeferCleanup(func() {
			metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{Phase: api.SnapshotOverlayCommitting})
			Eventually(stopped).Should(BeClosed())
		})
		return stopped
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		metadataCache = metadata.NewCache()
		manager = NewStorageManager(mockConn, metadataCache, nil)

		vmi = libvmi.New(
			libvmi.WithName("testvmi"),
			libvmi.WithNamespace("default"),
		)

		// A commit the monitor starts is held at its first call, so the phase and
		// the message it claimed with stay put for the spec to read.
		release := make(chan struct{})
		mockConn.EXPECT().LookupDomainByName(domainName).
			DoAndReturn(func(string) (cli.VirDomain, error) {
				<-release
				return nil, fmt.Errorf("the spec is over")
			}).AnyTimes()

		usage = func() (int64, error) { return 0, nil }
		measured = make(chan struct{}, 1024)
		manager.overlayUsage = func(dir string) (int64, error) {
			Expect(dir).To(Equal(overlayDir))
			used, err := usage()
			select {
			case measured <- struct{}{}:
			default:
			}
			return used, err
		}
		// Production measures every few seconds, shortened here so a spec does not
		// wait on a poll.
		manager.overlayUsageInterval = 5 * time.Millisecond

		DeferCleanup(func() { close(release) })

		metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{Phase: api.SnapshotOverlayReady})
	})

	Context("watching the scratch volume", func() {
		It("should leave the overlays alone while the volume has room", func() {
			usage = func() (int64, error) { return 69, nil }

			runMonitor()

			Consistently(overlayPhase, 100*time.Millisecond, 10*time.Millisecond).
				Should(Equal(api.SnapshotOverlayReady))
		})

		It("should commit the overlays once the volume is full enough", func() {
			usage = func() (int64, error) { return 70, nil }

			stopped := runMonitor()

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitting))
			// The commit it started is what ends it.
			Eventually(stopped).Should(BeClosed())
		})

		It("should say how full the volume was", func() {
			usage = func() (int64, error) { return 85, nil }

			runMonitor()

			Eventually(overlayMessage).Should(ContainSubstring("85% full"))
		})

		It("should keep measuring a volume it cannot read", func() {
			var polls int
			usage = func() (int64, error) {
				polls++
				if polls < 3 {
					return 0, fmt.Errorf("no such file or directory")
				}
				return 90, nil
			}

			runMonitor()

			Eventually(overlayPhase).Should(Equal(api.SnapshotOverlayCommitting))
		})

		It("should keep measuring while the commit is refused", func() {
			now := metav1.Now()
			metadataCache.Migration.Store(api.MigrationMetadata{StartTimestamp: &now})
			usage = func() (int64, error) { return 99, nil }

			runMonitor()

			Eventually(measured).Should(Receive())
			Eventually(measured).Should(Receive())
			Expect(overlayPhase()).To(Equal(api.SnapshotOverlayReady))
		})

		DescribeTable("should stop watching overlays that are no longer the guest's write target", func(phase api.SnapshotOverlayPhase) {
			metadataCache.SnapshotOverlay.Store(api.SnapshotOverlayMetadata{Phase: phase})
			usage = func() (int64, error) { return 99, nil }

			Eventually(runMonitor()).Should(BeClosed())

			Expect(overlayPhase()).To(Equal(phase))
		},
			Entry("when a commit is already running", api.SnapshotOverlayCommitting),
			Entry("when the overlays are gone", api.SnapshotOverlayPhase("")),
			Entry("when the snapshot failed and left nothing behind", api.SnapshotOverlaySnapshotFailed),
		)
	})

	Context("measuring the scratch volume", func() {
		It("should report how full the volume holding the overlays is", func() {
			used, err := overlayUsedPercent(GinkgoT().TempDir())

			Expect(err).ToNot(HaveOccurred())
			Expect(used).To(And(BeNumerically(">=", 0), BeNumerically("<=", 100)))
		})

		It("should fail on a directory that is not there", func() {
			_, err := overlayUsedPercent(overlayDir)

			Expect(err).To(HaveOccurred())
		})
	})
})
