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
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	migrationHookSocket = "/var/run/kubevirt/migration-hook-socket"
	migratedMarkerPath  = "/run/kubevirt-private/backend-storage-meta/migrated"
	connectTimeout      = 30 * time.Second
	idleTimeout         = 30 * time.Second
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
	dir := filepath.Dir(migratedMarkerPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.Create(migratedMarkerPath)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	return file.Close()
}
