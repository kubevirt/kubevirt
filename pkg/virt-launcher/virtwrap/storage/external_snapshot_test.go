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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
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
