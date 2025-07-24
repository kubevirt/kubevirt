package standalone

import (
	"encoding/json"
	"os"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap"
)

// HandleStandaloneMode checks for VMI_OBJ env var and syncs if present.
func HandleStandaloneMode(domainManager virtwrap.DomainManager) {
	if vmiObjStr, ok := os.LookupEnv("STANDALONE_VMI"); ok {
		var vmi v1.VirtualMachineInstance
		if err := json.Unmarshal([]byte(vmiObjStr), &vmi); err != nil {
			log.Log.Reason(err).Error("Failed to unmarshal VMI from STANDALONE_VMI")
			panic(err)
		}

		if _, err := domainManager.SyncVMI(&vmi, true, nil); err != nil {
			log.Log.Object(&vmi).Reason(err).Error("Failed to sync VMI, quitting")
			panic(err)
		}
	}
}
