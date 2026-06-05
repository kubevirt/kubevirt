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
	"os"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

// HandleStandaloneMode checks for STANDALONE_VMI env var and syncs if present.
func HandleStandaloneMode(domainManager virtwrap.DomainManager) {
	if vmiObjStr, ok := os.LookupEnv("STANDALONE_VMI"); ok {
		var vmi v1.VirtualMachineInstance
		// Try YAML unmarshal
		if err := yaml.Unmarshal([]byte(vmiObjStr), &vmi); err != nil {
			// Fallback to JSON if YAML fails
			if jsonErr := json.Unmarshal([]byte(vmiObjStr), &vmi); jsonErr != nil {
				log.Log.Reason(err).Error("Failed to unmarshal VMI from STANDALONE_VMI as YAML/JSON")
				panic(err)
			}
		}

		log.Log.Object(&vmi).Infof("Standalone mode: syncing VMI")
		if _, err := domainManager.SyncVMI(&vmi, true, nil); err != nil {
			log.Log.Object(&vmi).Reason(err).Error("Failed to sync VMI, quitting")
			panic(err)
		}
	}
}
