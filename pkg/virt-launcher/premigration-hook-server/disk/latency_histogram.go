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
 */

package disk

import (
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	converterstorage "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/storage"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func LatencyHistogramHook(
	_ *convertertypes.ConverterContext,
	_ *v1.VirtualMachineInstance,
	domain *libvirtxml.Domain,
) error {
	if domain.Devices == nil {
		return nil
	}

	for i := range domain.Devices.Disks {
		driver := domain.Devices.Disks[i].Driver
		if driver == nil {
			continue
		}

		ensureLatencyHistograms(driver)
	}

	return nil
}

func ensureLatencyHistograms(driver *libvirtxml.DomainDiskDriver) {
	if driver.Statistics == nil {
		driver.Statistics = &libvirtxml.DomainDiskStatistics{}
	}

	existingOperations := map[string]struct{}{}

	for _, histogram := range driver.Statistics.LatencyHistogram {
		// An omitted type applies the same histogram to every operation.
		if histogram.Type == "" {
			return
		}

		existingOperations[histogram.Type] = struct{}{}
	}

	for _, operation := range converterstorage.DefaultLatencyHistogramOperations() {
		if _, exists := existingOperations[operation]; exists {
			continue
		}

		driver.Statistics.LatencyHistogram = append(
			driver.Statistics.LatencyHistogram,
			defaultLatencyHistogram(operation),
		)
	}
}

func defaultLatencyHistogram(operation string) libvirtxml.DomainDiskLatencyHistogram {
	starts := converterstorage.DefaultLatencyHistogramBinStarts()
	bins := make([]libvirtxml.DomainDiskLatencyHistogramBin, len(starts))

	for i, start := range starts {
		bins[i] = libvirtxml.DomainDiskLatencyHistogramBin{
			Start: uint(start),
		}
	}

	return libvirtxml.DomainDiskLatencyHistogram{
		Type: operation,
		Bin:  bins,
	}
}
