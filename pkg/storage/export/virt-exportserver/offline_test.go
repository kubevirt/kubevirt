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

package virtexportserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	backupv1 "kubevirt.io/api/backup/v1alpha1"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/storage/export/export"
	"kubevirt.io/kubevirt/pkg/storage/nbdclient"
)

var _ = Describe("Offline backup data plane", func() {
	Describe("isOfflinePush", func() {
		It("is true only for push mode", func() {
			Expect((&exportServer{ExportServerConfig: ExportServerConfig{BackupMode: string(backupv1.PushMode)}}).isOfflinePush()).To(BeTrue())
			Expect((&exportServer{ExportServerConfig: ExportServerConfig{BackupMode: string(backupv1.PullMode)}}).isOfflinePush()).To(BeFalse())
			Expect((&exportServer{}).isOfflinePush()).To(BeFalse())
		})
	})

	Describe("backupBitmapName", func() {
		It("returns the base checkpoint for offline backups", func() {
			s := &exportServer{ExportServerConfig: ExportServerConfig{OfflineBackup: true, BackupBaseCheckpoint: "base", BackupCheckpoint: "new"}}
			Expect(s.backupBitmapName(export.BackupInfo{Path: "rootdisk"})).To(Equal("base"))
		})

		It("returns the checkpoint for online incremental backups", func() {
			s := &exportServer{ExportServerConfig: ExportServerConfig{BackupCheckpoint: "new"}}
			Expect(s.backupBitmapName(export.BackupInfo{Path: "rootdisk", Type: string(backupv1.Incremental)})).To(Equal("new"))
		})

		It("is empty for online full backups", func() {
			Expect((&exportServer{}).backupBitmapName(export.BackupInfo{Path: "rootdisk", Type: string(backupv1.Full)})).To(BeEmpty())
		})

		It("is empty for an offline volume that fell back to a full export", func() {
			s := &exportServer{
				ExportServerConfig: ExportServerConfig{OfflineBackup: true, BackupBaseCheckpoint: "base"},
				offline: &offlineDataPlane{volumes: []*offlineVolume{
					{name: "rootdisk", incremental: true},
					{name: "newdisk", incremental: false},
				}},
			}
			Expect(s.backupBitmapName(export.BackupInfo{Path: "rootdisk"})).To(Equal("base"))
			Expect(s.backupBitmapName(export.BackupInfo{Path: "newdisk"})).To(BeEmpty())
		})
	})

	Describe("qemuImgConvertToTarget", func() {
		vol := &offlineVolume{name: "rootdisk", socketPath: "/sockets/rootdisk.sock"}

		It("converts a full backup without a dirty bitmap", func() {
			var gotArgs []string
			runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
				gotArgs = args
				return nil, nil
			}
			Expect(qemuImgConvertToTarget(context.Background(), runner, vol, "/backup-target/out.qcow2", false, "")).To(Succeed())
			joined := strings.Join(gotArgs, " ")
			Expect(joined).To(ContainSubstring("convert"))
			Expect(joined).To(ContainSubstring("driver=nbd,server.type=unix,server.path=/sockets/rootdisk.sock,export=rootdisk"))
			Expect(joined).To(ContainSubstring("-O qcow2 /backup-target/out.qcow2"))
		})

		// dirtyExtents now reads the dirty map over libnbd (covered in the nbdclient package), so the
		// copy loop is exercised directly with the extents it would return.
		It("copies only the dirty extents for an incremental backup, leaving the rest unallocated", func() {
			var calls [][]string
			runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
				calls = append(calls, args)
				if args[0] == "info" {
					return []byte(`{"virtual-size":1048576}`), nil
				}
				return nil, nil
			}
			extents := []nbdclient.Extent{{Offset: 0, Length: 65536}}

			Expect(copyExtentsToTarget(context.Background(), runner, vol, "/backup-target/out.qcow2", extents)).To(Succeed())

			Expect(calls).To(HaveLen(3), "info, create, and one convert per dirty extent")
			Expect(strings.Join(calls[0], " ")).To(Equal(
				"info --output=json --image-opts driver=nbd,server.type=unix,server.path=/sockets/rootdisk.sock,export=rootdisk"))
			Expect(strings.Join(calls[1], " ")).To(Equal("create -f qcow2 /backup-target/out.qcow2 1048576"))
			Expect(strings.Join(calls[2], " ")).To(Equal(
				"convert -n --image-opts driver=raw,offset=0,size=65536,file.driver=nbd,file.server.type=unix,file.server.path=/sockets/rootdisk.sock,file.export=rootdisk " +
					"--target-image-opts driver=raw,offset=0,file.driver=qcow2,file.file.driver=file,file.file.filename=/backup-target/out.qcow2"))
		})

		It("issues one convert call per dirty extent", func() {
			convertCalls := 0
			runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
				switch args[0] {
				case "info":
					return []byte(`{"virtual-size":1048576}`), nil
				case "convert":
					convertCalls++
				}
				return nil, nil
			}
			extents := []nbdclient.Extent{{Offset: 0, Length: 65536}, {Offset: 131072, Length: 65536}}
			Expect(copyExtentsToTarget(context.Background(), runner, vol, "/backup-target/out.qcow2", extents)).To(Succeed())
			Expect(convertCalls).To(Equal(2))
		})
	})

	Describe("dataFileOptions", func() {
		It("errors when the data path is missing", func() {
			_, _, err := dataFileOptions("/does/not/exist")
			Expect(err).To(HaveOccurred())
		})
	})

	// These use real qemu-img/qemu-nbd processes against a temp overlay instead of
	// a mocked commandRunner, since qemu-nbd is a long-running process rather than
	// a run-to-completion command and can't be mocked the same way.
	Describe("pruneStaleBitmaps and bitmap enumeration", func() {
		var vol *offlineVolume

		newOverlay := func() *offlineVolume {
			dir := GinkgoT().TempDir()
			// dataFileOptions treats a directory as the filesystem-PVC case, pointing
			// qemu-img at <dir>/disk.img via the "file" driver. A plain file would be
			// treated as a raw block device instead, which a regular file isn't.
			dataDir := filepath.Join(dir, "data")
			Expect(os.Mkdir(dataDir, 0755)).To(Succeed())
			dataPath := filepath.Join(dataDir, "disk.img")
			_, err := runCommand(context.Background(), "qemu-img", "create", "-f", "raw", dataPath, "2M")
			Expect(err).ToNot(HaveOccurred())
			overlayPath := filepath.Join(dir, "overlay.qcow2")
			_, err = runCommand(context.Background(), "qemu-img", "create", "-f", "qcow2",
				"-o", fmt.Sprintf("data_file=%s,data_file_raw=on", dataPath), overlayPath, "2M")
			Expect(err).ToNot(HaveOccurred())
			return &offlineVolume{name: "rootdisk", overlayPath: overlayPath, dataPath: dataDir}
		}

		addBitmap := func(name string) {
			imageOpts, err := overlayImageOpts(vol)
			Expect(err).ToNot(HaveOccurred())
			_, err = runCommand(context.Background(), qemuImgBinary, "bitmap", "--add", "--image-opts", imageOpts, name)
			Expect(err).ToNot(HaveOccurred())
		}

		BeforeEach(func() {
			vol = newOverlay()
		})

		It("removes bitmaps that aren't in the keep list", func() {
			addBitmap("stale-1")
			addBitmap("stale-2")
			addBitmap("keep-me")

			Expect(pruneStaleBitmaps(context.Background(), runCommand, vol, "keep-me")).To(Succeed())

			names, err := listBitmapNames(context.Background(), runCommand, vol.overlayPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(names).To(ConsistOf("keep-me"))
		})

		It("keeps every bitmap that matches the keep list", func() {
			addBitmap("base")
			addBitmap("new")

			Expect(pruneStaleBitmaps(context.Background(), runCommand, vol, "base", "new")).To(Succeed())

			names, err := listBitmapNames(context.Background(), runCommand, vol.overlayPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(names).To(ConsistOf("base", "new"))
		})

		It("does nothing when the overlay has no bitmaps yet", func() {
			Expect(pruneStaleBitmaps(context.Background(), runCommand, vol, "base")).To(Succeed())

			names, err := listBitmapNames(context.Background(), runCommand, vol.overlayPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(names).To(BeEmpty())
		})
	})

	Describe("prepareOfflineVolumes keep-list", func() {
		var (
			overlayPath string
			socketDir   string
			dataDir     string
			s           *exportServer
		)

		addBitmap := func(name string) {
			vol := &offlineVolume{name: "rootdisk", overlayPath: overlayPath, dataPath: dataDir}
			imageOpts, err := overlayImageOpts(vol)
			Expect(err).ToNot(HaveOccurred())
			_, err = runCommand(context.Background(), qemuImgBinary, "bitmap", "--add", "--image-opts", imageOpts, name)
			Expect(err).ToNot(HaveOccurred())
		}

		bitmapNames := func() []string {
			names, err := listBitmapNames(context.Background(), runCommand, overlayPath)
			Expect(err).ToNot(HaveOccurred())
			return names
		}

		BeforeEach(func() {
			dir := GinkgoT().TempDir()
			dataDir = filepath.Join(dir, "data")
			Expect(os.Mkdir(dataDir, 0755)).To(Succeed())
			dataPath := filepath.Join(dataDir, "disk.img")
			_, err := runCommand(context.Background(), "qemu-img", "create", "-f", "raw", dataPath, "2M")
			Expect(err).ToNot(HaveOccurred())

			statePath := filepath.Join(dir, "state")
			overlayRelPath := cbt.GetQCOW2OverlayPVCPath("rootdisk")
			Expect(os.MkdirAll(filepath.Dir(filepath.Join(statePath, overlayRelPath)), 0755)).To(Succeed())
			overlayPath = filepath.Join(statePath, overlayRelPath)
			_, err = runCommand(context.Background(), "qemu-img", "create", "-f", "qcow2",
				"-o", fmt.Sprintf("data_file=%s,data_file_raw=on", dataPath), overlayPath, "2M")
			Expect(err).ToNot(HaveOccurred())

			socketDir = filepath.Join(dir, "sockets")
			Expect(os.Mkdir(socketDir, 0755)).To(Succeed())

			s = &exportServer{ExportServerConfig: ExportServerConfig{
				CommandRunner:    runCommand,
				BackupStatePath:  statePath,
				SocketDir:        socketDir,
				BackupCheckpoint: "backup-this",
				Paths: &export.ServerPaths{
					Backups: []export.BackupInfo{{Path: "rootdisk", Type: string(backupv1.Full), DiskPath: dataDir, OverlayPath: overlayRelPath}},
				},
			}}
		})

		It("protects other trackers' checkpoints and skips recording its own when untracked", func() {
			addBitmap("backup-other-tracker")
			addBitmap("stale-unrelated")
			s.BackupKeepCheckpoints = []string{"backup-other-tracker"}
			s.BackupTracked = false

			dp, err := s.prepareOfflineVolumes(context.Background())
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(dp.stop)

			Expect(bitmapNames()).To(ConsistOf("backup-other-tracker"))
		})

		It("keeps its own new checkpoint bitmap once the backup belongs to a tracker", func() {
			addBitmap("backup-other-tracker")
			s.BackupKeepCheckpoints = []string{"backup-other-tracker"}
			s.BackupTracked = true

			dp, err := s.prepareOfflineVolumes(context.Background())
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(dp.stop)

			Expect(bitmapNames()).To(ConsistOf("backup-other-tracker", "backup-this"))
		})
	})

	Describe("probeOfflineVolumeTypes", func() {
		var (
			statePath      string
			overlayRelPath string
			dataDir        string
		)

		addBitmap := func(name string) {
			vol := &offlineVolume{name: "rootdisk", overlayPath: filepath.Join(statePath, overlayRelPath), dataPath: dataDir}
			imageOpts, err := overlayImageOpts(vol)
			Expect(err).ToNot(HaveOccurred())
			_, err = runCommand(context.Background(), qemuImgBinary, "bitmap", "--add", "--image-opts", imageOpts, name)
			Expect(err).ToNot(HaveOccurred())
		}

		incremental := func() []export.BackupInfo {
			return []export.BackupInfo{{Path: "rootdisk", Type: string(backupv1.Incremental), DiskPath: dataDir, OverlayPath: overlayRelPath}}
		}

		BeforeEach(func() {
			dir := GinkgoT().TempDir()
			dataDir = filepath.Join(dir, "data")
			Expect(os.Mkdir(dataDir, 0755)).To(Succeed())
			dataPath := filepath.Join(dataDir, "disk.img")
			_, err := runCommand(context.Background(), "qemu-img", "create", "-f", "raw", dataPath, "2M")
			Expect(err).ToNot(HaveOccurred())

			statePath = filepath.Join(dir, "state")
			overlayRelPath = cbt.GetQCOW2OverlayPVCPath("rootdisk")
			Expect(os.MkdirAll(filepath.Dir(filepath.Join(statePath, overlayRelPath)), 0755)).To(Succeed())
			_, err = runCommand(context.Background(), "qemu-img", "create", "-f", "qcow2",
				"-o", fmt.Sprintf("data_file=%s,data_file_raw=on", dataPath), filepath.Join(statePath, overlayRelPath), "2M")
			Expect(err).ToNot(HaveOccurred())
		})

		It("reports nothing when the base bitmap exists", func() {
			addBitmap("base")
			out, err := probeOfflineVolumeTypes(context.Background(), runCommand, incremental(), statePath, "base")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(BeEmpty())
		})

		It("reports only the volume that fell back to full", func() {
			// No "base" bitmap added, so the incremental volume must fall back.
			out, err := probeOfflineVolumeTypes(context.Background(), runCommand, incremental(), statePath, "base")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(ConsistOf(offlineVolumeTypeResult{VolumeName: "rootdisk", Type: string(backupv1.Full)}))
		})

		It("reports nothing for a full-requested volume", func() {
			full := []export.BackupInfo{{Path: "rootdisk", Type: string(backupv1.Full), DiskPath: dataDir, OverlayPath: overlayRelPath}}
			out, err := probeOfflineVolumeTypes(context.Background(), runCommand, full, statePath, "base")
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(BeEmpty())
		})
	})

	Describe("stopVolumeNBD", func() {
		var (
			vol *offlineVolume
			dp  *offlineDataPlane
		)

		BeforeEach(func() {
			dir := GinkgoT().TempDir()
			dataDir := filepath.Join(dir, "data")
			Expect(os.Mkdir(dataDir, 0755)).To(Succeed())
			dataPath := filepath.Join(dataDir, "disk.img")
			_, err := runCommand(context.Background(), "qemu-img", "create", "-f", "raw", dataPath, "2M")
			Expect(err).ToNot(HaveOccurred())
			overlayPath := filepath.Join(dir, "overlay.qcow2")
			_, err = runCommand(context.Background(), "qemu-img", "create", "-f", "qcow2",
				"-o", fmt.Sprintf("data_file=%s,data_file_raw=on", dataPath), overlayPath, "2M")
			Expect(err).ToNot(HaveOccurred())

			vol = &offlineVolume{name: "rootdisk", overlayPath: overlayPath, dataPath: dataDir, socketPath: filepath.Join(dir, "rootdisk.sock")}
			dp = &offlineDataPlane{volumes: []*offlineVolume{vol}}

			Expect(dp.startQemuNBD(context.Background(), vol, false, "")).To(Succeed())
			DeferCleanup(dp.stop)
		})

		It("releases the overlay's write lock once stopped", func() {
			stopVolumeNBD(vol)

			imageOpts, err := overlayImageOpts(vol)
			Expect(err).ToNot(HaveOccurred())
			_, err = runCommand(context.Background(), qemuImgBinary, "bitmap", "--add", "--image-opts", imageOpts, "proof-lock-released")
			Expect(err).ToNot(HaveOccurred())
		})

		It("removes the socket file once stopped", func() {
			stopVolumeNBD(vol)
			_, err := os.Stat(vol.socketPath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("startQemuNBD socket timeout", func() {
		It("kills the qemu-nbd process instead of leaking it", func() {
			dir := GinkgoT().TempDir()
			dataDir := filepath.Join(dir, "data")
			Expect(os.Mkdir(dataDir, 0755)).To(Succeed())
			dataPath := filepath.Join(dataDir, "disk.img")
			_, err := runCommand(context.Background(), "qemu-img", "create", "-f", "raw", dataPath, "2M")
			Expect(err).ToNot(HaveOccurred())
			overlayPath := filepath.Join(dir, "overlay.qcow2")
			_, err = runCommand(context.Background(), "qemu-img", "create", "-f", "qcow2",
				"-o", fmt.Sprintf("data_file=%s,data_file_raw=on", dataPath), overlayPath, "2M")
			Expect(err).ToNot(HaveOccurred())

			vol := &offlineVolume{name: "rootdisk", overlayPath: overlayPath, dataPath: dataDir, socketPath: filepath.Join(dir, "rootdisk.sock")}
			dp := &offlineDataPlane{}

			originalTimeout := socketWaitTimeout
			socketWaitTimeout = 0
			DeferCleanup(func() { socketWaitTimeout = originalTimeout })

			Expect(dp.startQemuNBD(context.Background(), vol, false, "")).To(MatchError(ContainSubstring("socket for volume rootdisk not ready")))

			Expect(vol.cmd).ToNot(BeNil())
			Expect(vol.cmd.ProcessState).ToNot(BeNil(), "qemu-nbd should have been killed and reaped on socket timeout, not left running")
		})
	})
})
