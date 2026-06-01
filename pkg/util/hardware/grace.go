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

package hardware

const (
	nvidiaPCIVendorID          = "10DE"
	nvidiaGraceGPUDeviceID2342 = "2342"
	nvidiaGraceGPUDeviceID2348 = "2348"
	nvidiaGraceGPUDeviceID2941 = "2941"
)

var nvidiaGraceGPUDeviceIDs = map[string]struct{}{
	nvidiaGraceGPUDeviceID2342: {},
	nvidiaGraceGPUDeviceID2348: {},
	nvidiaGraceGPUDeviceID2941: {},
}

func IsNVIDIAGraceGPU(vendorID, deviceID string) bool {
	if NormalizePCIID(vendorID) != nvidiaPCIVendorID {
		return false
	}
	_, ok := nvidiaGraceGPUDeviceIDs[NormalizePCIID(deviceID)]
	return ok
}

func IsNVIDIAPCIVendor(vendorID string) bool {
	return NormalizePCIID(vendorID) == nvidiaPCIVendorID
}
