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
 */

package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const (
	migrationHookSocket = "/var/run/kubevirt/migration-hook-socket"
	migratedMarkerDir   = "/run/kubevirt-private/backend-storage-meta"
	migratedMarkerPath  = migratedMarkerDir + "/migrated"
	connectTimeout      = 30 * time.Second
	idleTimeout         = 30 * time.Second

	// blockStateDevicePath is the in-pod path of the raw VolumeDevice that backs a
	// Block-mode (VMStateVolumeMode=Block) backend-storage volume. Its presence is the
	// positive signal that the VMI runs with Block-mode backend storage. This is a copy
	// of backendstorage.BlockDevicePath; this hook binary is deliberately stdlib-only and
	// must not pull in the backend-storage package (and its client-go deps), so the
	// constant is duplicated here. Keep the two in sync.
	blockStateDevicePath = "/dev/vm-state"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Hook failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 4 {
		return nil
	}

	// Args: qemu <domain-name> <operation> <sub-operation> [extra]
	// Example: qemu kubevirt-test_testvmi migrate begin -
	op := os.Args[2]
	subOp := os.Args[3]
	extra := ""
	if len(os.Args) > 4 {
		extra = os.Args[4]
	}

	if op == "migrate" && subOp == "begin" {
		return handleMigrationBegin()
	}

	if op == "release" && subOp == "end" && extra == "migrated" {
		return handleMigrationEnd()
	}

	return nil
}

func handleMigrationBegin() error {
	// Connect to the migration hook server with timeout
	dialer := net.Dialer{
		Timeout: connectTimeout,
	}

	conn, err := dialer.Dial("unix", migrationHookSocket)
	if err != nil {
		return fmt.Errorf("failed to connect to migration hook socket: %w", err)
	}
	defer conn.Close()

	// Set deadline for the entire operation
	if err := conn.SetDeadline(time.Now().Add(idleTimeout)); err != nil {
		return fmt.Errorf("failed to set connection deadline: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		// Send domain XML from stdin to server
		if _, err := io.Copy(conn, os.Stdin); err != nil {
			done <- fmt.Errorf("failed to send data: %w", err)
			return
		}

		response, err := io.ReadAll(conn)
		if err != nil {
			done <- fmt.Errorf("failed to read response: %w", err)
			return
		}

		if len(response) == 0 {
			done <- fmt.Errorf("hook processing failed on server")
			return
		}

		if _, err := os.Stdout.Write(response); err != nil {
			done <- fmt.Errorf("failed to write response: %w", err)
			return
		}

		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(idleTimeout):
		return fmt.Errorf("operation timed out after %v", idleTimeout)
	}
}

func handleMigrationEnd() error {
	// The /meta/migrated marker is a Filesystem-mode-only handoff: the recovery Job reads
	// it (via a SubPath mount of the source PVC) to decide whether an interrupted migration
	// completed. In Block mode (VMStateVolumeMode=Block) the backend storage is a raw block
	// device with no filesystem and no meta directory, and the state (EFI NVRAM, vTPM)
	// travels in QEMU's / swtpm's migration stream on RWX-Block PVCs shared by source and
	// target, so no marker handshake is needed.
	//
	// Decide Block-vs-Filesystem positively, by detecting the presence of the Block-mode
	// state device, rather than by the absence of the marker directory. The latter would
	// silently swallow a genuinely missing marker directory in Filesystem mode (a real
	// misconfiguration) as if it were Block mode.
	isBlock, err := isBlockMode()
	if err != nil {
		return err
	}
	if isBlock {
		return nil
	}

	file, err := os.Create(migratedMarkerPath)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	return file.Close()
}

// isBlockMode reports whether this VMI runs with Block-mode backend storage, detected by
// the presence of the raw state device that the controller attaches as a VolumeDevice.
func isBlockMode() (bool, error) {
	info, err := os.Stat(blockStateDevicePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No state device: Filesystem mode. The marker directory must exist; if it
			// doesn't, surface that as an error from handleMigrationEnd rather than here,
			// so the caller's os.Create returns a descriptive failure.
			return false, nil
		}
		return false, fmt.Errorf("failed to stat state device %s: %w", blockStateDevicePath, err)
	}
	return (info.Mode() & os.ModeDevice) != 0, nil
}
