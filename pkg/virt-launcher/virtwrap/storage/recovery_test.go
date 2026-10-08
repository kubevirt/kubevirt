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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"kubevirt.io/kubevirt/pkg/pointer"
	hostdisk "kubevirt.io/kubevirt/pkg/storage/host-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestRemainingOverlayRecovery(t *testing.T) {
	reference := storagetypes.SnapshotOverlayReference{ContentUID: "content", ScratchPVCUID: "scratch"}
	originalInfo := overlayDiskInfo
	defer func() { overlayDiskInfo = originalInfo }()
	for _, scenario := range []string{"pending", "first-committed", "all-committed", "wrong-backing", "wrong-format", "wrong-base", "missing-driver"} {
		t.Run(scenario, func(t *testing.T) {
			g := NewWithT(t)
			dir := t.TempDir()
			vmi := libvmi.New(libvmi.WithDataVolume("root", "root-dv"), libvmi.WithPersistentVolumeClaim("data", "data-pvc"))
			for _, volume := range vmi.Spec.Volumes {
				vmi.Status.VolumeStatus = append(vmi.Status.VolumeStatus, v1.VolumeStatus{
					Name: volume.Name,
					PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
						Capacity: k8sv1.ResourceList{k8sv1.ResourceStorage: resource.MustParse("1Gi")},
					},
				})
			}
			// SyncVMI receives filesystem PVCs and DataVolumes as HostDisk sources.
			g.Expect(hostdisk.ReplacePVCByHostDisk(vmi)).To(Succeed())
			disks := []api.Disk{
				{Alias: api.NewUserDefinedAlias("root"), Target: api.DiskTarget{Device: "vda"}, Driver: &api.DiskDriver{Type: "raw"}, Source: api.DiskSource{File: "/base/root.img"}},
				{Alias: api.NewUserDefinedAlias("data"), Target: api.DiskTarget{Device: "vdb"}, Driver: &api.DiskDriver{Type: "raw"}, Source: api.DiskSource{File: "/base/data.img"}},
			}
			for _, disk := range disks {
				g.Expect(os.WriteFile(overlayPath(dir, disk.Alias.GetName()), []byte("qcow2"), 0600)).To(Succeed())
			}
			// Device letters change after a replacement launcher. Volume names do not.
			disks[0].Target.Device, disks[1].Target.Device = "vdc", "vda"
			domain := &api.Domain{}
			domain.Spec.Devices.Disks = disks
			overlayDiskInfo = func(path string) (*osdisk.DiskInfo, error) {
				if _, err := os.Stat(path); err != nil {
					return nil, err
				}
				name := filepath.Base(path)
				base := "/base/root.img"
				if name == "ovl-data.qcow2" {
					base = "/base/data.img"
				}
				if scenario == "wrong-backing" && name == "ovl-data.qcow2" {
					base = "/base/root.img"
				}
				format := "qcow2"
				if scenario == "wrong-format" {
					format = "raw"
				}
				return &osdisk.DiskInfo{Format: format, BackingFile: base}, nil
			}
			switch scenario {
			case "first-committed", "all-committed":
				g.Expect(os.Remove(overlayPath(dir, "root"))).To(Succeed())
				if scenario == "all-committed" {
					g.Expect(os.Remove(overlayPath(dir, "data"))).To(Succeed())
				}
			case "wrong-base":
				domain.Spec.Devices.Disks[1].Source.File = "/base/replacement.img"
			case "missing-driver":
				domain.Spec.Devices.Disks[1].Driver = nil

			}
			err := recoverOverlayDisks(vmi, domain, dir)
			if scenario == "all-committed" {
				mountDir, mounted := overlayVolumeMountDir, verifyOverlayDirMounted
				defer func() { overlayVolumeMountDir = mountDir; verifyOverlayDirMounted = mounted }()
				overlayVolumeMountDir = func(string) string { return dir }
				verifyOverlayDirMounted = func(string) error { return nil }
				refJSON, _ := json.Marshal(reference)
				vmi.Annotations = map[string]string{v1.PendingOverlayRecoveryAnnotation: "snap-scratch-content", storagetypes.OverlaySnapshotReferenceAnnotation: string(refJSON)}
				vmi.Spec.UtilityVolumes = []v1.UtilityVolume{{Name: "snap-scratch-content", PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{ClaimName: "snap-scratch-content"}, Type: pointer.P(v1.SnapshotOverlay)}}
				cache := metadata.NewCache()
				manager := NewStorageManager(nil, cache, nil)
				recovered, err := manager.RecoverOverlays(vmi, domain)
				if err != nil || !recovered {
					t.Fatalf("empty scratch silently skipped completion: %v", err)
				}
				observed, _ := cache.SnapshotOverlay.Load()
				if observed.Phase != api.SnapshotOverlayCommitting || observed.ContentUID != reference.ContentUID || observed.ScratchPVCUID != reference.ScratchPVCUID {
					t.Fatal("empty scratch did not enter matching commit flow")
				}
			}
			success := scenario == "pending" || scenario == "first-committed" || scenario == "all-committed"
			if !success {
				if err == nil {
					t.Fatal("unsafe recovery was accepted")
				}
				if domain.Spec.Devices.Disks[0].Source.File != "/base/root.img" {
					t.Fatal("partial validation modified the first disk")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for i, disk := range domain.Spec.Devices.Disks {
				expected := overlayPath(dir, disk.Alias.GetName())
				if scenario == "all-committed" || scenario == "first-committed" && i == 0 {
					expected = fmt.Sprintf("/base/%s.img", disk.Alias.GetName())
				}
				if disk.Source.File != expected {
					t.Fatalf("volume %s selected %s, expected %s", disk.Alias.GetName(), disk.Source.File, expected)
				}
				if disk.Source.File == overlayPath(dir, disk.Alias.GetName()) && disk.Driver.Type != "qcow2" {
					t.Fatal("overlay started with base format")
				}
			}
		})
	}
}
