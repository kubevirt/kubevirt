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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/monitoring/metrics/testing"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	driverNameLabel    = "driver_name"
	driverVersionLabel = "driver_version"
	deviceIDLabel      = "device_id"

	ethernetDriverName     = "Red Hat VirtIO Ethernet Adapter"
	ethernetDriverVersion  = "100.95.104.26200"
	ethernetDriverDate     = 1721001600000000000
	ethernetDriverDateSecs = 1721001600.0
	ethernetDeviceID       = 4161
	ethernetDeviceIDHex    = "1041"

	scsiDriverName     = "Red Hat VirtIO SCSI pass-through controller"
	scsiDriverVersion  = "100.95.104.26200"
	scsiDriverDate     = 1721088000000000000
	scsiDriverDateSecs = 1721088000.0
	scsiDeviceID       = 4162
	scsiDeviceIDHex    = "1042"
)

type stubClusterConfig struct {
	guestDeviceMetrics bool
}

func (s *stubClusterConfig) GuestDeviceMetricsEnabled() bool {
	return s.guestDeviceMetrics
}

var _ = Describe("device metrics", func() {
	Context("on Collect", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi-1",
				Namespace: "test-ns-1",
			},
		}

		BeforeEach(func() {
			settings = &collectorSettings{clusterConfig: &stubClusterConfig{guestDeviceMetrics: true}}
		})

		It("should collect metrics values and labels", func() {
			vmiStats := &VirtualMachineInstanceStats{
				DeviceStats: []api.GuestDevice{
					{
						DriverName:    ethernetDriverName,
						DriverVersion: ethernetDriverVersion,
						DriverDate:    ethernetDriverDate,
						DeviceID:      ethernetDeviceID,
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)
			crs := deviceMetrics{}.Collect(vmiReport)

			Expect(crs).To(HaveLen(1))
			Expect(crs).To(ContainElement(testing.GomegaContainsCollectorResultMatcher(guestDeviceDriverDateSeconds, ethernetDriverDateSecs)))

			Expect(crs[0].ConstLabels).To(HaveKeyWithValue(driverNameLabel, ethernetDriverName))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue(driverVersionLabel, ethernetDriverVersion))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue(deviceIDLabel, ethernetDeviceIDHex))
		})

		It("should collect metrics for multiple devices with distinct labels", func() {
			vmiStats := &VirtualMachineInstanceStats{
				DeviceStats: []api.GuestDevice{
					{
						DriverName:    ethernetDriverName,
						DriverVersion: ethernetDriverVersion,
						DriverDate:    ethernetDriverDate,
						DeviceID:      ethernetDeviceID,
					},
					{
						DriverName:    scsiDriverName,
						DriverVersion: scsiDriverVersion,
						DriverDate:    scsiDriverDate,
						DeviceID:      scsiDeviceID,
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)
			crs := deviceMetrics{}.Collect(vmiReport)

			Expect(crs).To(HaveLen(2))

			resultsByDeviceID := map[string]operatormetrics.CollectorResult{}
			for _, cr := range crs {
				resultsByDeviceID[cr.ConstLabels[deviceIDLabel]] = cr
			}

			ethernet := resultsByDeviceID[ethernetDeviceIDHex]
			Expect(ethernet.Value).To(Equal(ethernetDriverDateSecs))
			Expect(ethernet.ConstLabels).To(HaveKeyWithValue(driverNameLabel, ethernetDriverName))
			Expect(ethernet.ConstLabels).To(HaveKeyWithValue(driverVersionLabel, ethernetDriverVersion))

			scsi := resultsByDeviceID[scsiDeviceIDHex]
			Expect(scsi.Value).To(Equal(scsiDriverDateSecs))
			Expect(scsi.ConstLabels).To(HaveKeyWithValue(driverNameLabel, scsiDriverName))
			Expect(scsi.ConstLabels).To(HaveKeyWithValue(driverVersionLabel, scsiDriverVersion))
		})

		It("should deduplicate devices sharing the same driver and device id", func() {
			vmiStats := &VirtualMachineInstanceStats{
				DeviceStats: []api.GuestDevice{
					{
						DriverName:    ethernetDriverName,
						DriverVersion: ethernetDriverVersion,
						DriverDate:    ethernetDriverDate,
						DeviceID:      ethernetDeviceID,
					},
					{
						DriverName:    ethernetDriverName,
						DriverVersion: ethernetDriverVersion,
						DriverDate:    ethernetDriverDate,
						DeviceID:      ethernetDeviceID,
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)
			crs := deviceMetrics{}.Collect(vmiReport)

			Expect(crs).To(HaveLen(1))
			Expect(crs[0].ConstLabels).To(HaveKeyWithValue(deviceIDLabel, ethernetDeviceIDHex))
		})

		It("should return empty when feature gate is disabled", func() {
			settings = &collectorSettings{clusterConfig: &stubClusterConfig{guestDeviceMetrics: false}}
			vmiStats := &VirtualMachineInstanceStats{
				DeviceStats: []api.GuestDevice{
					{
						DriverName:    ethernetDriverName,
						DriverVersion: ethernetDriverVersion,
						DriverDate:    ethernetDriverDate,
						DeviceID:      ethernetDeviceID,
					},
				},
			}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)
			crs := deviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})

		It("should return empty when no device stats exist", func() {
			vmiStats := &VirtualMachineInstanceStats{}
			vmiReport := newVirtualMachineInstanceReport(vmi, vmiStats)
			crs := deviceMetrics{}.Collect(vmiReport)
			Expect(crs).To(BeEmpty())
		})
	})
})
