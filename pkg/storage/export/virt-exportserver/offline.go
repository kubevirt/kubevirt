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
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/export/export"
	"kubevirt.io/kubevirt/pkg/storage/nbdclient"
)

const (
	qemuNBDBinary     = "qemu-nbd"
	qemuImgBinary     = "qemu-img"
	qemuNBDMaxClients = "8"
)

// Overridable in tests.
var socketWaitTimeout = 30 * time.Second

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type offlineVolume struct {
	name        string
	overlayPath string
	dataPath    string
	socketPath  string
	incremental bool
	cmd         *exec.Cmd
}

type offlineDataPlane struct {
	volumes []*offlineVolume
}

func (s *exportServer) backupBitmapName(backup export.BackupInfo) string {
	if s.OfflineBackup {
		if s.offline != nil && !s.offline.volumeIncremental(backup.Path) {
			return ""
		}
		return s.BackupBaseCheckpoint
	}
	if backup.Type == string(backupv1.Incremental) && s.BackupCheckpoint != "" {
		return s.BackupCheckpoint
	}
	return ""
}

func (dp *offlineDataPlane) volumeIncremental(name string) bool {
	for _, vol := range dp.volumes {
		if vol.name == name {
			return vol.incremental
		}
	}
	return false
}

func (s *exportServer) isOfflinePush() bool {
	return s.BackupMode == string(backupv1.PushMode)
}

type offlineVolumeTypeResult struct {
	VolumeName string `json:"volumeName"`
	Type       string `json:"type"`
}

// probeOfflineVolumeTypes runs prepareOfflineVolumes' bitmap check standalone so it can run in an
// init container. It reports only volumes that fall back to full (missing base bitmap), keeping the
// payload small for the ~4KB termination message.
func probeOfflineVolumeTypes(ctx context.Context, runner commandRunner, backups []export.BackupInfo, statePath, baseCheckpoint string) ([]offlineVolumeTypeResult, error) {
	var fallbacks []offlineVolumeTypeResult
	for _, bi := range backups {
		overlayPath := filepath.Join(statePath, bi.OverlayPath)
		if _, err := os.Stat(overlayPath); err != nil {
			return nil, fmt.Errorf("CBT overlay for volume %s not found: %w", bi.Path, err)
		}

		if bi.Type != string(backupv1.Incremental) || baseCheckpoint == "" {
			continue
		}
		usable, err := usableBitmap(ctx, runner, overlayPath, baseCheckpoint)
		if err != nil {
			return nil, err
		}
		if !usable {
			log.Log.Infof("Volume %s has no usable bitmap for checkpoint %s; falling back to a full export for this disk", bi.Path, baseCheckpoint)
			fallbacks = append(fallbacks, offlineVolumeTypeResult{VolumeName: bi.Path, Type: string(backupv1.Full)})
		}
	}
	return fallbacks, nil
}

// Keep the report under the kubelet's ~4096-byte termination-message cap. A truncated payload
// would be silently unparseable.
const maxProbePayloadBytes = 3584

// RunOfflineVolumeTypeProbe writes the probe result to the container's termination message, which the
// backup controller reads off the pod status (no in-pod API access needed).
func RunOfflineVolumeTypeProbe(paths *export.ServerPaths, statePath, baseCheckpoint string) error {
	if paths == nil {
		return fmt.Errorf("offline backup type probe requested but no backup volumes configured")
	}
	fallbacks, err := probeOfflineVolumeTypes(context.Background(), runCommand, paths.Backups, statePath, baseCheckpoint)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(fallbacks)
	if err != nil {
		return fmt.Errorf("failed to encode volume type report: %w", err)
	}
	// A truncated payload would parse as invalid JSON and be silently skipped, so fail loud instead.
	if len(payload) > maxProbePayloadBytes {
		_ = os.WriteFile(offlineTerminationLogPath, []byte("offline backup volume type report exceeds the termination-message budget"), 0o644)
		return fmt.Errorf("offline backup volume type report is %d bytes, over the %d-byte termination-message budget", len(payload), maxProbePayloadBytes)
	}
	return os.WriteFile(offlineTerminationLogPath, payload, 0o644)
}

func (s *exportServer) startOfflineDataPlane(ctx context.Context) error {
	dp, err := s.prepareOfflineVolumes(ctx)
	if err != nil {
		return err
	}

	s.offline = dp
	s.nbdSource = dp.nbdSource()

	log.Log.Infof("Offline backup data plane ready for %d volume(s)", len(dp.volumes))
	return nil
}

// Output layout matches online push mode (<target>/<vm>/<backup>-<time>/<backup>-<volume>.qcow2)
// so existing restore tooling works unchanged.
func (s *exportServer) runOfflinePush(ctx context.Context) error {
	dp, err := s.prepareOfflineVolumes(ctx)
	if err != nil {
		return err
	}
	defer dp.stop()

	if err := os.MkdirAll(s.BackupTargetDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup target dir %s: %w", s.BackupTargetDir, err)
	}

	for _, vol := range dp.volumes {
		target := filepath.Join(s.BackupTargetDir, fmt.Sprintf("%s-%s.qcow2", s.BackupName, vol.name))
		if err := qemuImgConvertToTarget(ctx, s.CommandRunner, vol, target, vol.incremental, s.BackupBaseCheckpoint); err != nil {
			return err
		}
		log.Log.Infof("Offline push backup wrote %s", target)
	}

	return nil
}

func (s *exportServer) prepareOfflineVolumes(ctx context.Context) (*offlineDataPlane, error) {
	if s.Paths == nil || len(s.Paths.Backups) == 0 {
		return nil, fmt.Errorf("offline backup requested but no backup volumes configured")
	}

	dp := &offlineDataPlane{}

	for _, bi := range s.Paths.Backups {
		incremental := bi.Type == string(backupv1.Incremental)
		vol := &offlineVolume{
			name:        bi.Path,
			overlayPath: filepath.Join(s.BackupStatePath, bi.OverlayPath),
			dataPath:    bi.DiskPath,
			socketPath:  filepath.Join(s.SocketDir, bi.Path+".sock"),
			incremental: incremental,
		}
		if _, err := os.Stat(vol.overlayPath); err != nil {
			dp.stop()
			return nil, fmt.Errorf("CBT overlay for volume %s not found: %w", vol.name, err)
		}

		if incremental && s.BackupBaseCheckpoint != "" {
			usable, err := usableBitmap(ctx, s.CommandRunner, vol.overlayPath, s.BackupBaseCheckpoint)
			if err != nil {
				dp.stop()
				return nil, err
			}
			if !usable {
				log.Log.Infof("Volume %s has no usable bitmap for checkpoint %s; falling back to a full export for this disk", vol.name, s.BackupBaseCheckpoint)
				vol.incremental = false
			}
		}

		var keep []string
		if s.BackupBaseCheckpoint != "" {
			keep = append(keep, s.BackupBaseCheckpoint)
		}
		if s.BackupCheckpoint != "" {
			keep = append(keep, s.BackupCheckpoint)
		}
		keep = append(keep, s.BackupKeepCheckpoints...)
		if err := pruneStaleBitmaps(ctx, s.CommandRunner, vol, keep...); err != nil {
			dp.stop()
			return nil, err
		}

		if s.BackupCheckpoint != "" && s.BackupTracked {
			if err := addCheckpointBitmap(ctx, s.CommandRunner, vol, s.BackupCheckpoint); err != nil {
				dp.stop()
				return nil, err
			}
		}

		if err := dp.startQemuNBD(ctx, vol, vol.incremental, s.BackupBaseCheckpoint); err != nil {
			dp.stop()
			return nil, err
		}
		dp.volumes = append(dp.volumes, vol)
	}

	return dp, nil
}

func dirtyExtents(ctx context.Context, vol *offlineVolume, baseCheckpoint string) ([]nbdclient.Extent, error) {
	var dirty []nbdclient.Extent
	for extent, err := range nbdclient.NewNBDClient(vol.socketPath).Map(ctx, vol.name, baseCheckpoint, 0, 0) {
		if err != nil {
			return nil, fmt.Errorf("failed to map volume %s: %w", vol.name, err)
		}
		if extent.IsDirty() {
			dirty = append(dirty, extent)
		}
	}
	return dirty, nil
}

func qemuImgConvertToTarget(ctx context.Context, runner commandRunner, vol *offlineVolume, target string, incremental bool, baseCheckpoint string) error {
	if !incremental || baseCheckpoint == "" {
		srcOpts := fmt.Sprintf("driver=nbd,server.type=unix,server.path=%s,export=%s", vol.socketPath, vol.name)
		out, err := runner(ctx, qemuImgBinary, "convert", "--image-opts", srcOpts, "-O", "qcow2", target)
		if err != nil {
			return fmt.Errorf("qemu-img convert failed for volume %s: %w (%s)", vol.name, err, string(out))
		}
		return nil
	}
	return convertDirtyExtentsToTarget(ctx, runner, vol, target, baseCheckpoint)
}

// A plain "convert" copies the whole disk, so convert each dirty extent into its own window and
// leave the rest of the target unallocated.
func convertDirtyExtentsToTarget(ctx context.Context, runner commandRunner, vol *offlineVolume, target string, baseCheckpoint string) error {
	extents, err := dirtyExtents(ctx, vol, baseCheckpoint)
	if err != nil {
		return err
	}
	return copyExtentsToTarget(ctx, runner, vol, target, extents)
}

func copyExtentsToTarget(ctx context.Context, runner commandRunner, vol *offlineVolume, target string, extents []nbdclient.Extent) error {
	infoOpts := fmt.Sprintf("driver=nbd,server.type=unix,server.path=%s,export=%s", vol.socketPath, vol.name)
	out, err := runner(ctx, qemuImgBinary, "info", "--output=json", "--image-opts", infoOpts)
	if err != nil {
		return fmt.Errorf("qemu-img info failed for volume %s: %w (%s)", vol.name, err, string(out))
	}
	var info struct {
		VirtualSize uint64 `json:"virtual-size"`
	}
	if err := json.Unmarshal(out, &info); err != nil {
		return fmt.Errorf("failed to parse qemu-img info for volume %s: %w", vol.name, err)
	}

	if out, err := runner(ctx, qemuImgBinary, "create", "-f", "qcow2", target, fmt.Sprintf("%d", info.VirtualSize)); err != nil {
		return fmt.Errorf("failed to create incremental target %s: %w (%s)", target, err, string(out))
	}

	for _, e := range extents {
		srcOpts := fmt.Sprintf("driver=raw,offset=%d,size=%d,file.driver=nbd,file.server.type=unix,file.server.path=%s,file.export=%s",
			e.Offset, e.Length, vol.socketPath, vol.name)
		dstOpts := fmt.Sprintf("driver=raw,offset=%d,file.driver=qcow2,file.file.driver=file,file.file.filename=%s", e.Offset, target)
		out, err := runner(ctx, qemuImgBinary, "convert", "-n", "--image-opts", srcOpts, "--target-image-opts", dstOpts)
		if err != nil {
			return fmt.Errorf("qemu-img convert failed for extent [%d,%d) of volume %s: %w (%s)", e.Offset, e.Offset+e.Length, vol.name, err, string(out))
		}
	}
	return nil
}

func (dp *offlineDataPlane) startQemuNBD(ctx context.Context, vol *offlineVolume, incremental bool, baseCheckpoint string) error {
	imageOpts, err := overlayImageOpts(vol)
	if err != nil {
		return err
	}

	args := []string{
		"--socket", vol.socketPath,
		"--export-name", vol.name,
		"--persistent",
		"--shared", qemuNBDMaxClients,
		"--read-only",
		"--image-opts", imageOpts,
	}
	if incremental && baseCheckpoint != "" {
		args = append(args, "--bitmap", baseCheckpoint)
	}

	cmd := exec.CommandContext(ctx, qemuNBDBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start qemu-nbd for volume %s: %w", vol.name, err)
	}
	vol.cmd = cmd

	if err := waitForSocket(vol.socketPath); err != nil {
		stopVolumeNBD(vol)
		return fmt.Errorf("qemu-nbd socket for volume %s not ready: %w", vol.name, err)
	}
	return nil
}

// nbdSource builds an nbdSource that dispatches to each volume's own local
// qemu-nbd client by export name, since offline backup runs one qemu-nbd per
// volume instead of a single multi-export server.
func (dp *offlineDataPlane) nbdSource() nbdSource {
	router := &offlineNBDRouter{clients: map[string]*nbdclient.NBDClient{}}
	for _, vol := range dp.volumes {
		router.clients[vol.name] = nbdclient.NewNBDClient(vol.socketPath)
	}
	return router
}

func stopVolumeNBD(vol *offlineVolume) {
	if vol.cmd != nil && vol.cmd.Process != nil {
		_ = vol.cmd.Process.Kill()
		_ = vol.cmd.Wait()
	}
	_ = os.Remove(vol.socketPath)
}

func (dp *offlineDataPlane) stop() {
	if dp == nil {
		return
	}
	for _, vol := range dp.volumes {
		stopVolumeNBD(vol)
	}
}

type offlineNBDRouter struct {
	clients map[string]*nbdclient.NBDClient
}

func (r *offlineNBDRouter) Serving(timeout time.Duration) error {
	for _, client := range r.clients {
		if err := client.Serving(timeout); err != nil {
			return err
		}
	}
	return nil
}

func (r *offlineNBDRouter) Map(ctx context.Context, exportName, bitmapName string, offset, length uint64) iter.Seq2[nbdclient.Extent, error] {
	client, ok := r.clients[exportName]
	if !ok {
		return func(yield func(nbdclient.Extent, error) bool) {
			yield(nbdclient.Extent{}, fmt.Errorf("unknown export %q", exportName))
		}
	}
	return client.Map(ctx, exportName, bitmapName, offset, length)
}

func (r *offlineNBDRouter) Read(ctx context.Context, exportName string, offset, length uint64, w io.Writer) error {
	client, ok := r.clients[exportName]
	if !ok {
		return fmt.Errorf("unknown export %q", exportName)
	}
	return client.Read(ctx, exportName, offset, length, w)
}

// The data-file path baked into the overlay points at the virt-launcher mount,
// which doesn't exist here, so every qemu invocation must open it via this instead.
func overlayImageOpts(vol *offlineVolume) (string, error) {
	dataDriver, dataFile, err := dataFileOptions(vol.dataPath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("driver=qcow2,file.driver=file,file.filename=%s,data-file.driver=%s,data-file.filename=%s",
		vol.overlayPath, dataDriver, dataFile), nil
}

func dataFileOptions(dataPath string) (string, string, error) {
	info, err := os.Stat(dataPath)
	if err != nil {
		return "", "", fmt.Errorf("data path %s not accessible: %w", dataPath, err)
	}
	if info.IsDir() {
		return "file", filepath.Join(dataPath, "disk.img"), nil
	}
	return "host_device", dataPath, nil
}

type qemuImgBitmap struct {
	Name  string   `json:"name"`
	Flags []string `json:"flags"`
}

// usable reports whether the bitmap is a sound incremental base. qemu marks a consistent persistent
// bitmap "auto"; an "in-use" bitmap (e.g. the VM was killed instead of shut down) may have missed
// writes, and qemu-nbd --bitmap refuses it outright.
func (b qemuImgBitmap) usable() bool {
	return slices.Contains(b.Flags, "auto") && !slices.Contains(b.Flags, "in-use")
}

type qemuImgFormatInfo struct {
	Data struct {
		Bitmaps []qemuImgBitmap `json:"bitmaps"`
	} `json:"data"`
}

type qemuImgInfo struct {
	FormatSpecific qemuImgFormatInfo `json:"format-specific"`
}

func listBitmaps(ctx context.Context, runner commandRunner, overlayPath string) ([]qemuImgBitmap, error) {
	out, err := runner(ctx, qemuImgBinary, "info", "--output=json", overlayPath)
	if err != nil {
		return nil, fmt.Errorf("qemu-img info failed for %s: %w (%s)", overlayPath, err, string(out))
	}
	var info qemuImgInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, fmt.Errorf("failed to parse qemu-img info for %s: %w", overlayPath, err)
	}
	return info.FormatSpecific.Data.Bitmaps, nil
}

func listBitmapNames(ctx context.Context, runner commandRunner, overlayPath string) ([]string, error) {
	bitmaps, err := listBitmaps(ctx, runner, overlayPath)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(bitmaps))
	for _, b := range bitmaps {
		names = append(names, b.Name)
	}
	return names, nil
}

func findBitmap(ctx context.Context, runner commandRunner, overlayPath, name string) (*qemuImgBitmap, error) {
	bitmaps, err := listBitmaps(ctx, runner, overlayPath)
	if err != nil {
		return nil, err
	}
	for i := range bitmaps {
		if bitmaps[i].Name == name {
			return &bitmaps[i], nil
		}
	}
	return nil, nil
}

// usableBitmap reports whether overlayPath has a bitmap for name that can base an incremental.
func usableBitmap(ctx context.Context, runner commandRunner, overlayPath, name string) (bool, error) {
	b, err := findBitmap(ctx, runner, overlayPath, name)
	if err != nil {
		return false, err
	}
	return b != nil && b.usable(), nil
}

// A bitmap ends up stale if a prior attempt committed it but failed before the tracker advanced,
// since nothing else cleans those up.
//
// TODO: VEP 399 (multiple retained checkpoints) breaks the one-base-one-new assumption here.
// Revisit the keep-list when multi-checkpoint support lands.
func pruneStaleBitmaps(ctx context.Context, runner commandRunner, vol *offlineVolume, keep ...string) error {
	names, err := listBitmapNames(ctx, runner, vol.overlayPath)
	if err != nil {
		return err
	}
	for _, name := range names {
		if slices.Contains(keep, name) {
			continue
		}
		if err := removeBitmap(ctx, runner, vol, name); err != nil {
			return err
		}
		log.Log.Infof("Removed stale checkpoint bitmap %s from volume %s", name, vol.name)
	}
	return nil
}

func removeBitmap(ctx context.Context, runner commandRunner, vol *offlineVolume, name string) error {
	imageOpts, err := overlayImageOpts(vol)
	if err != nil {
		return err
	}
	out, err := runner(ctx, qemuImgBinary, "bitmap", "--remove", "--image-opts", imageOpts, name)
	if err != nil {
		return fmt.Errorf("failed to remove bitmap %s from %s: %w (%s)", name, vol.overlayPath, err, string(out))
	}
	return nil
}

func addCheckpointBitmap(ctx context.Context, runner commandRunner, vol *offlineVolume, name string) error {
	existing, err := findBitmap(ctx, runner, vol.overlayPath, name)
	if err != nil {
		return err
	}
	if existing != nil {
		if existing.usable() {
			return nil
		}
		// A leftover bitmap with this name that isn't consistent (e.g. a prior attempt was killed
		// mid-write) can't be recorded into, so drop it and add a fresh one.
		if err := removeBitmap(ctx, runner, vol, name); err != nil {
			return err
		}
	}
	imageOpts, err := overlayImageOpts(vol)
	if err != nil {
		return err
	}
	out, err := runner(ctx, qemuImgBinary, "bitmap", "--add", "--image-opts", imageOpts, name)
	if err != nil {
		return fmt.Errorf("failed to add checkpoint bitmap %s to %s: %w (%s)", name, vol.overlayPath, err, string(out))
	}
	return nil
}

func waitForSocket(socketPath string) error {
	deadline := time.Now().Add(socketWaitTimeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for socket %s", socketPath)
}
