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

package vsock

import (
	"os"
	"strings"

	"kubevirt.io/client-go/log"
)

const (
	ModeLocal  = "local"
	ModeGlobal = "global"

	ChildNsModePath = "/proc/sys/net/vsock/child_ns_mode"
	NsModePath      = "/proc/sys/net/vsock/ns_mode"

	LocalCID = 3
)

func DetectChildNsMode() string {
	return readVSOCKMode(ChildNsModePath)
}

func IsLocalMode() bool {
	return readVSOCKMode(NsModePath) == ModeLocal
}

func readVSOCKMode(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Log.Infof("Failed to read %q, using default value \"global\": %v", path, err)
		return ModeGlobal
	}
	if strings.TrimSpace(string(data)) == ModeLocal {
		return ModeLocal
	}
	return ModeGlobal
}
