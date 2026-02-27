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

package common

import (
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/compute"
	converter_types "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func ConvertCmdv1SMBIOSToComputeSMBIOS(input *cmdv1.SMBios) *compute.SMBIOS {
	if input == nil {
		return nil
	}

	return &compute.SMBIOS{
		Manufacturer: input.Manufacturer,
		Product:      input.Product,
		Version:      input.Version,
		SKU:          input.Sku,
		Family:       input.Family,
	}
}

func ConvertEFIConfiguration(input *converter_types.EFIConfiguration) *compute.EFIConfiguration {
	if input == nil {
		return nil
	}

	return &compute.EFIConfiguration{
		EFICode:      input.EFICode,
		EFIVars:      input.EFIVars,
		SecureLoader: input.SecureLoader,
	}
}
