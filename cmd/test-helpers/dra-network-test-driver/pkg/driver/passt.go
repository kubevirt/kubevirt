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

package driver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"kubevirt.io/client-go/log"
)

const (
	binary      = "/usr/bin/passt"
	socketName  = "vhost.sock"
	pidFileName = "passt.pid"
)

// pidFilePath returns the file passt writes its PID to. It lives in the claim's
// host directory, so it is keyed by claim UID like the socket, and RemoveAll on
// that directory cleans it up.
func pidFilePath(hostPath string) string {
	return path.Join(hostPath, pidFileName)
}

// socketPathFor returns the vhost-user socket passt serves for a claim.
func socketPathFor(hostPath string) string {
	return path.Join(hostPath, socketName)
}

// runningPasstPID returns the PID of the passt instance serving hostPath, or 0
// if there is none. A PID file left behind by a passt that already exited is
// reported as not running, since the PID may since have been recycled: onto an
// unrelated process, which must never be signalled, or onto another claim's
// passt, which would stop the wrong backend.
func runningPasstPID(hostPath string) int {
	data, err := os.ReadFile(pidFilePath(hostPath))
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0
	}
	// Arguments in /proc/<pid>/cmdline are NUL-separated, argv[0] is the binary.
	// Requiring this claim's socket among them ties the PID to this claim, not
	// merely to some passt.
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil ||
		!strings.HasPrefix(string(cmdline), binary+"\x00") ||
		!strings.Contains(string(cmdline), socketPathFor(hostPath)) {
		return 0
	}
	return pid
}

// stopBackendDevice stops the passt instance serving hostPath, if any. passt is
// started with --one-off and exits by itself once a client disconnects, so this
// only has an effect for a claim whose consumer never connected.
func stopBackendDevice(hostPath string) {
	pid := runningPasstPID(hostPath)
	if pid == 0 {
		return
	}
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		log.Log.Reason(err).Errorf("failed to stop passt (pid %d) for %s", pid, hostPath)
		return
	}
	log.Log.Infof("Stopped passt (pid %d) for %s", pid, hostPath)
}

func executeBackendDevice(ctx context.Context, hostPath, targetInterface string, targetPort int) error {
	// The kubelet may replay PrepareResourceClaims for a claim that is already
	// prepared. Starting a second passt on the same socket would fail to bind.
	if pid := runningPasstPID(hostPath); pid != 0 {
		log.Log.Infof("passt (pid %d) is already serving %s", pid, hostPath)
		return nil
	}

	socketPath := socketPathFor(hostPath)
	gwIP, err := ipv4Addr(targetInterface)
	if err != nil {
		return err
	}

	args := []string{
		"--vhost-user",
		"-s", socketPath,
		"--outbound-if4", targetInterface,
		"-g", gwIP,
		"-4",
		"--one-off",
		"-t", strconv.Itoa(targetPort),
		"--pid", pidFilePath(hostPath),
	}

	cmd := exec.Command(binary, args...)

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start %s: %w", binary, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Poll until the socket file appears on the filesystem
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to check socket path %s: %w", socketPath, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for socket %s to be created", socketPath)
		case <-time.After(50 * time.Millisecond):
		}
	}

	if err := os.Chown(socketPath, qemuUID, qemuGID); err != nil {
		return fmt.Errorf("failed to chown %s: %w", socketPath, err)
	}

	return nil
}

func ipv4Addr(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", fmt.Errorf("failed to find interface %s: %w", ifaceName, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for %s: %w", ifaceName, err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip := ipNet.IP.To4(); ip != nil && ip.IsGlobalUnicast() {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address found on %s", ifaceName)
}
