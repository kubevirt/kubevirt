package admitters

import (
	"k8s.io/apimachinery/pkg/api/equality"
	v1 "kubevirt.io/api/core/v1"
)

func equalDiskIgnoreSerial(newDisk, oldDisk v1.Disk) bool {
	return equality.Semantic.DeepEqual(newDisk.Name, oldDisk.Name) &&
		equality.Semantic.DeepEqual(newDisk.DiskDevice, oldDisk.DiskDevice) &&
		equality.Semantic.DeepEqual(newDisk.BootOrder, oldDisk.BootOrder) &&
		equality.Semantic.DeepEqual(newDisk.DedicatedIOThread, oldDisk.DedicatedIOThread) &&
		equality.Semantic.DeepEqual(newDisk.Cache, oldDisk.Cache) &&
		equality.Semantic.DeepEqual(newDisk.IO, oldDisk.IO) &&
		equality.Semantic.DeepEqual(newDisk.Tag, oldDisk.Tag) &&
		equality.Semantic.DeepEqual(newDisk.BlockSize, oldDisk.BlockSize) &&
		equality.Semantic.DeepEqual(newDisk.Shareable, oldDisk.Shareable) &&
		equality.Semantic.DeepEqual(newDisk.ErrorPolicy, oldDisk.ErrorPolicy)
}
