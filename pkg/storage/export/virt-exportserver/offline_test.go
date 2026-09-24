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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
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
			Expect(s.backupBitmapName()).To(Equal("base"))
		})

		It("returns the checkpoint for online incremental backups", func() {
			s := &exportServer{ExportServerConfig: ExportServerConfig{BackupType: string(backupv1.Incremental), BackupCheckpoint: "new"}}
			Expect(s.backupBitmapName()).To(Equal("new"))
		})

		It("is empty for online full backups", func() {
			Expect((&exportServer{ExportServerConfig: ExportServerConfig{BackupType: string(backupv1.Full)}}).backupBitmapName()).To(BeEmpty())
		})
	})

	Describe("qemuImgConvertToTarget", func() {
		var (
			gotArgs []string
			restore func()
		)

		BeforeEach(func() {
			orig := runCommand
			runCommand = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				gotArgs = args
				return nil, nil
			}
			restore = func() { runCommand = orig }
		})

		AfterEach(func() { restore() })

		vol := &offlineVolume{name: "rootdisk", socketPath: "/sockets/rootdisk.sock"}

		It("converts a full backup without a dirty bitmap", func() {
			Expect(qemuImgConvertToTarget(context.Background(), vol, "/backup-target/out.qcow2", false, "")).To(Succeed())
			joined := strings.Join(gotArgs, " ")
			Expect(joined).To(ContainSubstring("convert"))
			Expect(joined).To(ContainSubstring("driver=nbd,server.type=unix,server.path=/sockets/rootdisk.sock,export=rootdisk"))
			Expect(joined).To(ContainSubstring("-O qcow2 /backup-target/out.qcow2"))
			Expect(joined).ToNot(ContainSubstring("x-dirty-bitmap"))
		})

		It("filters an incremental backup by the base bitmap", func() {
			Expect(qemuImgConvertToTarget(context.Background(), vol, "/backup-target/out.qcow2", true, "base")).To(Succeed())
			Expect(strings.Join(gotArgs, " ")).To(ContainSubstring("x-dirty-bitmap=qemu:dirty-bitmap:base"))
		})
	})

	Describe("dataFileOptions", func() {
		It("errors when the data path is missing", func() {
			_, _, err := dataFileOptions("/does/not/exist")
			Expect(err).To(HaveOccurred())
		})
	})
})
