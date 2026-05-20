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
 *
 */

package domainstats

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var hostDeviceInfo = operatormetrics.NewGauge(
	operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_host_device_info",
		Help: "Information about host devices assigned to the VMI.",
	},
)

type hostDeviceMetrics struct{}

func (hostDeviceMetrics) Describe() []operatormetrics.Metric {
	return []operatormetrics.Metric{
		hostDeviceInfo,
	}
}

func (hostDeviceMetrics) Collect(vmiReport *VirtualMachineInstanceReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if vmiReport.vmiStats.Domain == nil {
		return crs
	}

	for _, hostDev := range vmiReport.vmiStats.Domain.Spec.Devices.HostDevices {
		labels := map[string]string{}

		if hostDev.Alias != nil {
			labels["alias"] = hostDev.Alias.GetName()
		}

		if hostDev.Address != nil && hostDev.Address.Type == api.AddressPCI {
			labels["pci_bus_id"] = formatPCIBusID(hostDev.Address)
		}

		crs = append(crs, vmiReport.newCollectorResultWithLabels(hostDeviceInfo, 1, labels))
	}

	return crs
}

func formatPCIBusID(addr *api.Address) string {
	domain, errDomain := parseHex(addr.Domain)
	bus, errBus := parseHex(addr.Bus)
	slot, errSlot := parseHex(addr.Slot)
	function, errFunction := parseHex(addr.Function)

	if errDomain != nil || errBus != nil || errSlot != nil || errFunction != nil {
		log.Log.Errorf(
			"failed to parse PCI address components: domain=%q (err=%v), bus=%q (err=%v), slot=%q (err=%v), function=%q (err=%v)",
			addr.Domain, errDomain, addr.Bus, errBus, addr.Slot, errSlot, addr.Function, errFunction,
		)
	}

	return fmt.Sprintf(
		"%08x:%02x:%02x.%01x",
		domain, bus, slot, function,
	)
}

func parseHex(s string) (uint64, error) {
	s = strings.TrimPrefix(s, "0x")
	return strconv.ParseUint(s, 16, 64)
}
