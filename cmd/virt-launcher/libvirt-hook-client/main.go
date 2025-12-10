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
	"os"
)

const (
	migratedMarkerPath = "/run/kubevirt-private/backend-storage-meta/migrated"
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

	if op == "release" && subOp == "end" && extra == "migrated" {
		return handleMigrationEnd()
	}

	return nil
}

func handleMigrationEnd() error {
	file, err := os.Create(migratedMarkerPath)
	if err != nil {
		return fmt.Errorf("failed to create marker file: %w", err)
	}

	return file.Close()
}
