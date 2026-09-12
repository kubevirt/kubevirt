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

package envvar

import (
	"fmt"
	"strings"
)

const (
	ENV_VAR_SHARED_FILESYSTEM_PATHS     = "SHARED_FILESYSTEM_PATHS"
	ENV_VAR_LIBVIRT_DEBUG_LOGS          = "LIBVIRT_DEBUG_LOGS"
	ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY = "VIRT_LAUNCHER_LOG_VERBOSITY"
)

// ResourceNameToEnvVar converts a Kubernetes resource name (e.g. "nvidia.com/gpu")
// into the env var name used by device plugins (e.g. "PCIDEVICE_NVIDIA_COM_GPU").
func ResourceNameToEnvVar(prefix string, resourceName string) string {
	varName := strings.ToUpper(resourceName)
	// ReplaceAll is idiomatic for unconditional global replacement (Go 1.12+).
	varName = strings.ReplaceAll(varName, "/", "_")
	varName = strings.ReplaceAll(varName, ".", "_")
	return fmt.Sprintf("%s_%s", prefix, varName)
}
