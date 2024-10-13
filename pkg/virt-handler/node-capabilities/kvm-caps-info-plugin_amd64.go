//go:build amd64

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

package nodecapabilities

/*
#include <linux/kvm.h>
const int IoctlGetMsrIndexList = KVM_GET_MSR_INDEX_LIST;
const int IoctlCheckExtension = KVM_CHECK_EXTENSION;
// Capabilities (extensions).
const int CapHyperv = KVM_CAP_HYPERV;
const int CapHypervTime = KVM_CAP_HYPERV_TIME;
const int CapHypervVpIndex = KVM_CAP_HYPERV_VP_INDEX;
const int CapHypervTlbflush = KVM_CAP_HYPERV_TLBFLUSH;
const int CapHypervSendIPI = KVM_CAP_HYPERV_SEND_IPI;
const int CapHypervSynic = KVM_CAP_HYPERV_SYNIC;
const int CapHypervSynic2 = KVM_CAP_HYPERV_SYNIC2;
__u32 msr_list_get(void* data, int index) {
	struct kvm_msr_list *msrs = (struct kvm_msr_list*)data;
	return msrs->indices[index];
}

void msr_list_copy_nmsrs(void* old_list, void* new_list) {
	((struct kvm_msr_list*)new_list)->nmsrs = ((struct kvm_msr_list*)old_list)->nmsrs;
}

int msr_list_length(void* data) {
	struct kvm_msr_list *msrs = (struct kvm_msr_list*)data;
	return msrs->nmsrs;
}

*/
import "C"

import (
	"os"
	"syscall"
	"unsafe"

	"kubevirt.io/client-go/log"

	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const (
	HV_X64_MSR_RESET                   = 0x40000003
	HV_X64_MSR_VP_INDEX                = 0x40000002
	HV_X64_MSR_VP_RUNTIME              = 0x40000010
	HV_X64_MSR_SCONTROL                = 0x40000080
	HV_X64_MSR_STIMER0_CONFIG          = 0x400000B0
	HV_X64_MSR_TSC_FREQUENCY           = 0x40000022
	HV_X64_MSR_REENLIGHTENMENT_CONTROL = 0x40000106
)

type capability struct {
	Name      string
	Extension uintptr
	MSR       int
}

var CapsDesc = []capability{
	{
		Extension: uintptr(C.CapHyperv),
		Name:      "base",
	},
	{
		Extension: uintptr(C.CapHypervTime),
		Name:      "time",
	},
	{
		Extension: uintptr(C.CapHypervVpIndex),
		MSR:       HV_X64_MSR_VP_INDEX,
		Name:      "vpindex",
	},
	{
		Extension: uintptr(C.CapHypervTlbflush),
		Name:      "tlbflush",
	},
	{
		Extension: uintptr(C.CapHypervSendIPI),
		Name:      "ipi",
	},
	{
		Extension: uintptr(C.CapHypervSynic),
		MSR:       HV_X64_MSR_SCONTROL,
		Name:      "synic",
	},
	{
		Extension: uintptr(C.CapHypervSynic2),
		MSR:       HV_X64_MSR_SCONTROL,
		Name:      "synic2",
	},
	{
		MSR:  HV_X64_MSR_TSC_FREQUENCY,
		Name: "frequencies",
	},
	{
		MSR:  HV_X64_MSR_RESET,
		Name: "reset",
	},
	{
		MSR:  HV_X64_MSR_VP_RUNTIME,
		Name: "runtime",
	},
	{
		MSR:  HV_X64_MSR_STIMER0_CONFIG,
		Name: "synictimer",
	},
	{
		MSR:  HV_X64_MSR_REENLIGHTENMENT_CONTROL,
		Name: "reenlightenment",
	},
}

func availableMsrs(fd uintptr) ([]uint32, error) {
	buffer := make([]byte, 8, 8)
	msrsListPtr := unsafe.Pointer(&buffer[0])
	_, err := kvmIOCtl(
		fd,
		uintptr(C.IoctlGetMsrIndexList),
		uintptr(msrsListPtr))
	if err != 0 && err != syscall.E2BIG {
		return nil, err
	}

	length := C.msr_list_length(msrsListPtr)

	bufferSize := 4 + 4*length

	newBuffer := make([]byte, bufferSize, bufferSize)

	C.msr_list_copy_nmsrs(unsafe.Pointer(&buffer[0]), unsafe.Pointer(&newBuffer[0]))

	msrsListPtr = unsafe.Pointer(&newBuffer[0])

	_, err = kvmIOCtl(
		fd,
		uintptr(C.IoctlGetMsrIndexList),
		uintptr(msrsListPtr))
	if err != 0 {
		return nil, err
	}
	msrs := make([]uint32, 0, 0)

	for i := 0; i < int(length); i++ {
		res := C.msr_list_get(
			msrsListPtr,
			C.int(i))

		// Add this MSR.
		msrs = append(msrs, uint32(res))
	}
	return msrs, nil
}

func hasCapExtension(fd uintptr, extension uintptr) bool {
	res, _ := kvmIOCtl(fd, uintptr(C.IoctlCheckExtension), extension)

	// Returns: 0 if unsupported; 1 (or some other positive integer) if supported
	// https://www.kernel.org/doc/Documentation/virtual/kvm/api.txt
	return res > 0
}

func kvmIOCtl(fd uintptr, flag, arg uintptr) (res uintptr, err syscall.Errno) {
	res, _, err = syscall.Syscall(syscall.SYS_IOCTL,
		fd,
		flag,
		arg)
	return
}

func populateCaps(fd uintptr) (map[uint32]bool, error) {
	msrs, err := availableMsrs(fd)
	if err != nil {
		return nil, err
	}

	supportedMSRS := make(map[uint32]bool)
	for _, kvmMsr := range msrs {
		supportedMSRS[kvmMsr] = true
	}
	return supportedMSRS, nil
}

func exposeCapabilities(fd uintptr, supportedMSRS map[uint32]bool) []string {
	exposedCaps := []string{}
	for _, capb := range CapsDesc {
		shouldExpose := false
		if capb.MSR != 0 {
			_, isEnabled := supportedMSRS[uint32(capb.MSR)]
			shouldExpose = isEnabled
		}
		if capb.Extension != 0 {
			res := hasCapExtension(fd, capb.Extension)
			shouldExpose = res
		}

		if shouldExpose {
			exposedCaps = append(exposedCaps, capb.Name)
		}
	}
	return exposedCaps

}

func GetCapLabels() []string {
	devkvm, err := os.OpenFile(util.KVMPath, syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		log.DefaultLogger().Errorf("something happened during opening kvm file: " + err.Error())
		return nil
	}
	defer devkvm.Close()

	fd := devkvm.Fd()

	supportedMSRS, err := populateCaps(fd)
	if err != nil {
		log.DefaultLogger().Errorf("something happened during populating kvm caps: " + err.Error())
		return nil
	}

	return exposeCapabilities(fd, supportedMSRS)
}
