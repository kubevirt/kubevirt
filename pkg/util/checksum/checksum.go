package checksum

import (
	"crypto/sha256"
	"encoding/hex"

	"k8s.io/apimachinery/pkg/util/json"
	virtv1 "kubevirt.io/api/core/v1"
)

func FromVMISpec(vmiSpec *virtv1.VirtualMachineInstanceSpec) (string, error) {
	data, err := json.Marshal(vmiSpec)
	if err != nil {
		return "", err
	}
	return FromBytes(data), nil
}

func FromBytes(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	sum := hasher.Sum(nil)
	return hex.EncodeToString(sum)
}
