package main

import (
	"flag"
	"os"

	"kubevirt.io/client-go/log"
	virtlauncher "kubevirt.io/kubevirt/pkg/virt-launcher"
)

func main() {
	containerDiskDir := flag.String("container-disk-dir", "/var/run/kubevirt/container-disks", "Base directory for container disk data")
	flag.Parse()

	exitCode, err := virtlauncher.ForkAndMonitor(*containerDiskDir, true)
	if err != nil {
		log.Log.Reason(err).Error("monitoring virt-launcher failed")
		os.Exit(1)
	}
	os.Exit(exitCode)
}
