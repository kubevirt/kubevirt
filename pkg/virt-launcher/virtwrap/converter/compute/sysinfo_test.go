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

package compute_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
)

var _ = Describe("SysInfo Domain Configurator", func() {
	const (
		expectedUUID   = "1234567890"
		expectedSerial = "abcdefghijklmnopqrstuvwxyz"

		expectedManufacturer = "manufacturer"
		expectedFamily       = "family"
		expectedProduct      = "product"
		expectedSKU          = "sku"
		expectedVersion      = "version"

		expectedChassisManufacturer = "chassisManufacturer"
		expectedChassisVersion      = "chassisVersion"
		expectedChassisSerial       = "chassisSerial"
		expectedChassisAsset        = "chassisAsset"
		expectedChassisSKU          = "chassisSKU"
	)

	var (
		clusterWideSMBIOS compute.SMBIOS
		chassis           v1.Chassis
	)

	BeforeEach(func() {
		clusterWideSMBIOS = compute.SMBIOS{
			Manufacturer: expectedManufacturer,
			Family:       expectedFamily,
			Product:      expectedProduct,
			SKU:          expectedSKU,
			Version:      expectedVersion,
		}

		chassis = v1.Chassis{
			Manufacturer: expectedChassisManufacturer,
			Version:      expectedChassisVersion,
			Serial:       expectedChassisSerial,
			Asset:        expectedChassisAsset,
			Sku:          expectedChassisSKU,
		}
	})

	DescribeTable("Should configure SysInfo",
		func(vmi *v1.VirtualMachineInstance, smBIOS *compute.SMBIOS, expected api.SysInfo) {
			var domain api.Domain

			Expect(compute.NewSysInfoDomainConfigurator(smBIOS).Configure(vmi, &domain)).To(Succeed())

			expectedDomain := api.Domain{
				Spec: api.DomainSpec{
					SysInfo: &expected,
				},
			}

			Expect(domain).To(Equal(expectedDomain))
		},
		Entry(
			"Without firmware chassis and cluster-wide SMBIOS",
			libvmi.New(),
			nil,
			api.SysInfo{Type: "smbios"},
		),
		Entry(
			"With firmware UUID",
			libvmi.New(libvmi.WithFirmwareUUID(expectedUUID)),
			nil,
			api.SysInfo{
				Type:   "smbios",
				System: []api.Entry{{Name: "uuid", Value: expectedUUID}},
			},
		),
		Entry(
			"With firmware UUID and serial",
			libvmi.New(
				libvmi.WithFirmwareUUID(expectedUUID),
				withFirmwareSerial(expectedSerial),
			),
			nil,
			api.SysInfo{
				Type: "smbios",
				System: []api.Entry{
					{Name: "uuid", Value: expectedUUID},
					{Name: "serial", Value: expectedSerial},
				},
			},
		),
		Entry(
			"With firmware UUID, serial and cluster-wide SMBIOS",
			libvmi.New(
				libvmi.WithFirmwareUUID(expectedUUID),
				withFirmwareSerial(expectedSerial),
			),
			&clusterWideSMBIOS,
			api.SysInfo{
				Type: "smbios",
				System: []api.Entry{
					{Name: "uuid", Value: expectedUUID},
					{Name: "serial", Value: expectedSerial},
					{Name: "manufacturer", Value: expectedManufacturer},
					{Name: "family", Value: expectedFamily},
					{Name: "product", Value: expectedProduct},
					{Name: "sku", Value: expectedSKU},
					{Name: "version", Value: expectedVersion},
				},
			},
		),
		Entry(
			"With cluster-wide SMBIOS",
			libvmi.New(),
			&clusterWideSMBIOS,
			api.SysInfo{
				Type: "smbios",
				System: []api.Entry{
					{Name: "manufacturer", Value: expectedManufacturer},
					{Name: "family", Value: expectedFamily},
					{Name: "product", Value: expectedProduct},
					{Name: "sku", Value: expectedSKU},
					{Name: "version", Value: expectedVersion},
				},
			},
		),
		Entry(
			"With firmware UUID, serial, chassis and cluster-wide SMBIOS",
			libvmi.New(
				libvmi.WithFirmwareUUID(expectedUUID),
				withFirmwareSerial(expectedSerial),
				withChassis(&chassis),
			),
			&clusterWideSMBIOS,
			api.SysInfo{
				Type: "smbios",
				System: []api.Entry{
					{Name: "uuid", Value: expectedUUID},
					{Name: "serial", Value: expectedSerial},
					{Name: "manufacturer", Value: expectedManufacturer},
					{Name: "family", Value: expectedFamily},
					{Name: "product", Value: expectedProduct},
					{Name: "sku", Value: expectedSKU},
					{Name: "version", Value: expectedVersion},
				},
				Chassis: []api.Entry{
					{Name: "manufacturer", Value: expectedChassisManufacturer},
					{Name: "version", Value: expectedChassisVersion},
					{Name: "serial", Value: expectedChassisSerial},
					{Name: "asset", Value: expectedChassisAsset},
					{Name: "sku", Value: expectedChassisSKU},
				},
			},
		),
		Entry(
			"With chassis",
			libvmi.New(withChassis(&chassis)),
			nil,
			api.SysInfo{
				Type: "smbios",
				Chassis: []api.Entry{
					{Name: "manufacturer", Value: expectedChassisManufacturer},
					{Name: "version", Value: expectedChassisVersion},
					{Name: "serial", Value: expectedChassisSerial},
					{Name: "asset", Value: expectedChassisAsset},
					{Name: "sku", Value: expectedChassisSKU},
				},
			},
		),
	)
})

func withFirmwareSerial(serial string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = &v1.Firmware{}
		}

		vmi.Spec.Domain.Firmware.Serial = serial
	}
}

func withChassis(chassis *v1.Chassis) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Chassis = chassis
	}
}
