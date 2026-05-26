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

package mode

import (
	"os"
	"path/filepath"
	"strings"

	"kubevirt.io/client-go/log"
)

const (
	ModeLocal  = "local"
	ModeGlobal = "global"

	DefaultProcPath = "/proc"
)

func VsockChildNsMode(procPath string) string {
	const childNsMode = "sys/net/vsock/child_ns_mode"
	return readVSOCKMode(procPath, childNsMode)
}

func VsockNsMode(procPath string) string {
	const nsMode = "sys/net/vsock/ns_mode"
	return readVSOCKMode(procPath, nsMode)
}

func readVSOCKMode(procPath, path string) string {
	fullPath := filepath.Join(procPath, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		log.Log.Infof("Failed to read %q, using default value \"global\": %v", fullPath, err)
		return ModeGlobal
	}
	mode := strings.TrimSpace(string(data))
	switch mode {
	case ModeGlobal:
		return ModeGlobal
	case ModeLocal:
		return ModeLocal
	default:
		log.Log.Infof("Unexpected value in %q, using default value \"global\": %v", fullPath, mode)
		return ModeGlobal
	}
}
