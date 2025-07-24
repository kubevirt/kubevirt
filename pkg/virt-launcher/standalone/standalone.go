package standalone

import (
	"encoding/json"
	"os"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"sigs.k8s.io/yaml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

// HandleStandaloneMode checks for VMI_OBJ env var and syncs if present.
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

		if _, err := domainManager.SyncVMI(&vmi, true, nil); err != nil {
			log.Log.Object(&vmi).Reason(err).Error("Failed to sync VMI, quitting")
			panic(err)
		}
	}
}
