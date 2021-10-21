package vms

import k6tv1 "kubevirt.io/client-go/apis/core/v1"

type MetricsScraper interface {
	Scrape(key string, vmi *k6tv1.VirtualMachineInstance)
}
