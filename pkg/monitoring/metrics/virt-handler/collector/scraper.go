/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package collector

import k6tv1 "kubevirt.io/api/core/v1"

type MetricsScraper interface {
	Scrape(key string, vmi *k6tv1.VirtualMachineInstance)
	Complete()
}
