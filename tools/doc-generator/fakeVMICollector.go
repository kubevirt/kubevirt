package main

import (
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/vmistats"
)

type fakeVMICollector struct {
}

func (fc fakeVMICollector) Describe(_ chan<- *prometheus.Desc) {
}

// Collect needs to report all metrics to see it in docs
func (fc fakeVMICollector) Collect(ch chan<- prometheus.Metric) {

	vmis := []*k6tv1.VirtualMachineInstance{
		createVMI(),
	}
	ps := vmistats.VmiPrometheusScraper(ch, nil, vmis)
	ps.Scrape()
}

func RegisterFakeVMICollector() {
	prometheus.MustRegister(fakeVMICollector{})
}

func createVMI() *k6tv1.VirtualMachineInstance {
	liveMigrateStrategy := k6tv1.EvictionStrategyLiveMigrate
	return &k6tv1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Namespace: "test-ns", Name: "test-vm", CreationTimestamp: metav1.Now()},
		Status: k6tv1.VirtualMachineInstanceStatus{
			NodeName: "test-node",
			Phase:    k6tv1.Running,
			Conditions: []k6tv1.VirtualMachineInstanceCondition{
				{
					Type:               k6tv1.VirtualMachineInstanceIsMigratable,
					Status:             "false",
					Reason:             "any",
					LastTransitionTime: metav1.Now(),
				},
			},
		},
		Spec: k6tv1.VirtualMachineInstanceSpec{
			EvictionStrategy: &liveMigrateStrategy,
		},
	}
}
