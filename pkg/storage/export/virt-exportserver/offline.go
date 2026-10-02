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
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	"kubevirt.io/client-go/log"

	nbdv1 "kubevirt.io/kubevirt/pkg/storage/cbt/nbd/v1"
	"kubevirt.io/kubevirt/pkg/storage/nbdclient"
)

const (
	cbtOverlaySubDir  = "cbt"
	qemuNBDBinary     = "qemu-nbd"
	qemuImgBinary     = "qemu-img"
	qemuNBDMaxClients = "8"
	socketWaitTimeout = 30 * time.Second
)

// runCommand is overridable in tests.
var runCommand = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

type offlineVolume struct {
	name        string
	overlayPath string
	dataPath    string
	socketPath  string
	cmd         *exec.Cmd
}

type offlineDataPlane struct {
	volumes    []*offlineVolume
	grpcServer *grpc.Server
	grpcConn   *grpc.ClientConn
}

func (s *exportServer) backupBitmapName() string {
	if s.OfflineBackup {
		return s.BackupBaseCheckpoint
	}
	if s.BackupType == string(backupv1.Incremental) {
		return s.BackupCheckpoint
	}
	return ""
}

func (s *exportServer) isOfflinePush() bool {
	return s.BackupMode == string(backupv1.PushMode)
}

func (s *exportServer) startOfflineDataPlane(ctx context.Context) error {
	dp, err := s.prepareOfflineVolumes(ctx)
	if err != nil {
		return err
	}

	client, err := dp.startLoopback(s.SocketDir)
	if err != nil {
		dp.stop()
		return err
	}

	s.offline = dp
	s.nbdMu.Lock()
	s.nbdClient = client
	s.tunnelEstablished = true
	s.nbdMu.Unlock()

	log.Log.Infof("Offline backup data plane ready for %d volume(s)", len(dp.volumes))
	return nil
}

// runOfflinePush copies each volume's changed blocks into a qcow2 on the mounted
// target PVC and returns so the export pod exits, matching the online push output
// layout (<target>/<vm>/<backup>-<time>/<backup>-<volume>.qcow2).
func (s *exportServer) runOfflinePush(ctx context.Context) error {
	dp, err := s.prepareOfflineVolumes(ctx)
	if err != nil {
		return err
	}
	defer dp.stop()

	if err := os.MkdirAll(s.BackupTargetDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup target dir %s: %w", s.BackupTargetDir, err)
	}

	incremental := s.BackupType == string(backupv1.Incremental)
	for _, vol := range dp.volumes {
		target := filepath.Join(s.BackupTargetDir, fmt.Sprintf("%s-%s.qcow2", s.BackupName, vol.name))
		if err := qemuImgConvertToTarget(ctx, vol, target, incremental, s.BackupBaseCheckpoint); err != nil {
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
	incremental := s.BackupType == string(backupv1.Incremental)

	for _, bi := range s.Paths.Backups {
		vol := &offlineVolume{
			name:        bi.Path,
			overlayPath: filepath.Join(s.BackupStatePath, cbtOverlaySubDir, bi.Path+".qcow2"),
			dataPath:    bi.DiskPath,
			socketPath:  filepath.Join(s.SocketDir, bi.Path+".sock"),
		}
		if _, err := os.Stat(vol.overlayPath); err != nil {
			dp.stop()
			return nil, fmt.Errorf("CBT overlay for volume %s not found: %w", vol.name, err)
		}

		if incremental && s.BackupBaseCheckpoint != "" {
			exists, err := bitmapExists(ctx, vol.overlayPath, s.BackupBaseCheckpoint)
			if err != nil {
				dp.stop()
				return nil, err
			}
			if !exists {
				dp.stop()
				return nil, fmt.Errorf("base checkpoint bitmap %s missing from overlay %s", s.BackupBaseCheckpoint, vol.overlayPath)
			}
		}

		if s.BackupCheckpoint != "" {
			if err := addCheckpointBitmap(ctx, vol, s.BackupCheckpoint); err != nil {
				dp.stop()
				return nil, err
			}
		}

		if err := dp.startQemuNBD(ctx, vol, incremental, s.BackupBaseCheckpoint); err != nil {
			dp.stop()
			return nil, err
		}
		dp.volumes = append(dp.volumes, vol)
	}
	return dp, nil
}

// qemuImgConvertToTarget streams the volume from its qemu-nbd export into a fresh
// qcow2. For incremental backups the x-dirty-bitmap client option makes qemu-img
// treat only the base checkpoint's dirty extents as data, so unchanged clusters
// stay unallocated in the output.
func qemuImgConvertToTarget(ctx context.Context, vol *offlineVolume, target string, incremental bool, baseCheckpoint string) error {
	srcOpts := fmt.Sprintf("driver=nbd,server.type=unix,server.path=%s,export=%s", vol.socketPath, vol.name)
	if incremental && baseCheckpoint != "" {
		srcOpts += fmt.Sprintf(",x-dirty-bitmap=qemu:dirty-bitmap:%s", baseCheckpoint)
	}
	out, err := runCommand(ctx, qemuImgBinary, "convert", "--image-opts", srcOpts, "-O", "qcow2", target)
	if err != nil {
		return fmt.Errorf("qemu-img convert failed for volume %s: %w (%s)", vol.name, err, string(out))
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
		return fmt.Errorf("qemu-nbd socket for volume %s not ready: %w", vol.name, err)
	}
	return nil
}

func (dp *offlineDataPlane) startLoopback(socketDir string) (nbdv1.NBDClient, error) {
	router := &offlineNBDRouter{clients: map[string]*nbdclient.NBDClient{}}
	for _, vol := range dp.volumes {
		router.clients[vol.name] = nbdclient.NewNBDClient(vol.socketPath)
	}

	grpcSocket := filepath.Join(socketDir, "nbd-grpc.sock")
	_ = os.Remove(grpcSocket)
	lis, err := net.Listen("unix", grpcSocket)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on gRPC socket: %w", err)
	}

	dp.grpcServer = grpc.NewServer()
	nbdv1.RegisterNBDServer(dp.grpcServer, router)
	go func() {
		if err := dp.grpcServer.Serve(lis); err != nil {
			log.Log.Reason(err).Warning("offline NBD gRPC server stopped")
		}
	}()

	conn, err := grpc.NewClient("unix://"+grpcSocket, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to dial offline NBD gRPC server: %w", err)
	}
	dp.grpcConn = conn
	return nbdv1.NewNBDClient(conn), nil
}

func (dp *offlineDataPlane) stop() {
	if dp == nil {
		return
	}
	if dp.grpcConn != nil {
		dp.grpcConn.Close()
	}
	if dp.grpcServer != nil {
		dp.grpcServer.Stop()
	}
	for _, vol := range dp.volumes {
		if vol.cmd != nil && vol.cmd.Process != nil {
			_ = vol.cmd.Process.Kill()
			_ = vol.cmd.Wait()
		}
		_ = os.Remove(vol.socketPath)
	}
}

type offlineNBDRouter struct {
	clients map[string]*nbdclient.NBDClient
}

func (r *offlineNBDRouter) Map(req *nbdv1.MapRequest, stream nbdv1.NBD_MapServer) error {
	client, ok := r.clients[req.ExportName]
	if !ok {
		return fmt.Errorf("unknown export %q", req.ExportName)
	}
	return client.Map(req, stream)
}

func (r *offlineNBDRouter) Read(req *nbdv1.ReadRequest, stream nbdv1.NBD_ReadServer) error {
	client, ok := r.clients[req.ExportName]
	if !ok {
		return fmt.Errorf("unknown export %q", req.ExportName)
	}
	return client.Read(req, stream)
}

// overlayImageOpts builds the qemu --image-opts string that opens the CBT overlay
// against the export pod's data PVC mount. Every qemu invocation that opens the
// overlay must use it: the data-file path baked into the qcow2 points at the
// virt-launcher mount, which does not exist here.
func overlayImageOpts(vol *offlineVolume) (string, error) {
	dataDriver, dataFile, err := dataFileOptions(vol.dataPath)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("driver=qcow2,file.driver=file,file.filename=%s,data-file.driver=%s,data-file.filename=%s",
		vol.overlayPath, dataDriver, dataFile), nil
}

// dataFileOptions returns the qcow2 data-file driver and filename for opening the
// overlay against the export pod's mounted data PVC.
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
	Name string `json:"name"`
}

type qemuImgFormatInfo struct {
	Data struct {
		Bitmaps []qemuImgBitmap `json:"bitmaps"`
	} `json:"data"`
}

type qemuImgInfo struct {
	FormatSpecific qemuImgFormatInfo `json:"format-specific"`
}

func bitmapExists(ctx context.Context, overlayPath, name string) (bool, error) {
	out, err := runCommand(ctx, qemuImgBinary, "info", "--output=json", overlayPath)
	if err != nil {
		return false, fmt.Errorf("qemu-img info failed for %s: %w (%s)", overlayPath, err, string(out))
	}
	var info qemuImgInfo
	if err := json.Unmarshal(out, &info); err != nil {
		return false, fmt.Errorf("failed to parse qemu-img info for %s: %w", overlayPath, err)
	}
	for _, b := range info.FormatSpecific.Data.Bitmaps {
		if b.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func addCheckpointBitmap(ctx context.Context, vol *offlineVolume, name string) error {
	exists, err := bitmapExists(ctx, vol.overlayPath, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	imageOpts, err := overlayImageOpts(vol)
	if err != nil {
		return err
	}
	out, err := runCommand(ctx, qemuImgBinary, "bitmap", "--add", "--image-opts", imageOpts, name)
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
