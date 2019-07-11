package main

import (
	_ "kubevirt.io/kubevirt/pkg/monitoring/client/prometheus"    // import for prometheus metrics
	_ "kubevirt.io/kubevirt/pkg/monitoring/reflector/prometheus" // import for prometheus metrics
	_ "kubevirt.io/kubevirt/pkg/monitoring/workqueue/prometheus" // import for prometheus metrics
	virt_spice "kubevirt.io/kubevirt/pkg/virt-spice"
)

func main() {
	app := virt_spice.NewVirtSpice()
	app.Execute()
}
