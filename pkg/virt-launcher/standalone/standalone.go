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

package standalone

import (
	"encoding/json"
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"sigs.k8s.io/yaml"

	launcherconfig "kubevirt.io/kubevirt/pkg/virt-launcher/config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

// HandleStandaloneMode checks for STANDALONE_VMI env var and syncs if present.
// Uses the global config which was initialized at application startup.
func HandleStandaloneMode(domainManager virtwrap.DomainManager) {
	HandleStandaloneModeWithConfig(domainManager, launcherconfig.GetGlobalConfig())
}

// HandleStandaloneModeWithConfig checks the provided config for standalone VMI and syncs if present.
// This allows for explicit dependency injection of configuration values.
func HandleStandaloneModeWithConfig(domainManager virtwrap.DomainManager, cfg *launcherconfig.Config) {
	if cfg == nil || !cfg.IsStandaloneMode() {
		return
	}

	var vmi v1.VirtualMachineInstance
	// Try YAML unmarshal
	if err := yaml.Unmarshal([]byte(cfg.StandaloneVMI), &vmi); err != nil {
		// Fallback to JSON if YAML fails
		if jsonErr := json.Unmarshal([]byte(cfg.StandaloneVMI), &vmi); jsonErr != nil {
			log.Log.Reason(err).Error(fmt.Sprintf("Failed to unmarshal VMI from %s as YAML/JSON", launcherconfig.EnvVarStandaloneVMI))
			panic(err)
		}
	}

	log.Log.Object(&vmi).Infof("Standalone mode: syncing VMI")
	if _, err := domainManager.SyncVMI(&vmi, true, nil); err != nil {
		log.Log.Object(&vmi).Reason(err).Error("Failed to sync VMI, quitting")
		panic(err)
	}
}
