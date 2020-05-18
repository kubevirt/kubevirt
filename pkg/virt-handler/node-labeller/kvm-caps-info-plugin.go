package nodelabeller

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
int msr_list_index(void *data, int n, __u32 *index) {
    struct kvm_msr_list *msrs = (struct kvm_msr_list*)data;
    if (n >= msrs->nmsrs) {
        return 1;
    }
    *index = msrs->indices[n];
    return 0;
}
*/
import "C"

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"kubevirt.io/client-go/log"
)

const (
	HV_X64_MSR_CRASH_CTL               = 0x40000105
	HV_X64_MSR_RESET                   = 0x40000003
	HV_X64_MSR_VP_INDEX                = 0x40000002
	HV_X64_MSR_VP_RUNTIME              = 0x40000010
	HV_X64_MSR_SCONTROL                = 0x40000080
	HV_X64_MSR_STIMER0_CONFIG          = 0x400000B0
	HV_X64_MSR_TSC_FREQUENCY           = 0x40000022
	HV_X64_MSR_REENLIGHTENMENT_CONTROL = 0x40000106
)

type tmsr struct {
	nmsrs   C.__u32
	indices []C.__u32
}

type CapScan struct {
	supportedMSRS map[uint32]bool
	fd            uintptr
	logger        *log.FilteredLogger
}

type capability struct {
	Name      string
	Extension uintptr
	MSR       int
}

func NewCapScanner() *CapScan {
	capScan := new(CapScan)
	capScan.logger = log.DefaultLogger()
	return capScan
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

	// Find our list of MSR indices.
	// A page should be more than enough here,
	// eventually if it's not we'll end up with
	// a failed system call for some reason other
	// than E2BIG (which just says n is wrong).
	msrIndices := make([]byte, 4096, 4096)
	msrs := make([]uint32, 0, 0)
	for {

		_, err := kvmIOCtl(
			fd,
			uintptr(C.IoctlGetMsrIndexList),
			uintptr(unsafe.Pointer(&msrIndices[0])))
		if err == syscall.E2BIG {
			// The nmsrs field will now have been
			// adjusted, and we can run it again.
			continue
		} else if err != 0 {
			return nil, err
		}

		// We're good!
		break
	}

	// Extract each msr individually.
	for i := 0; ; i += 1 {
		// Is there a valid index?
		var index C.__u32
		e := C.msr_list_index(
			unsafe.Pointer(&msrIndices[0]),
			C.int(i),
			&index)

		// Any left?
		if e != 0 {
			break
		}

		// Add this MSR.
		msrs = append(msrs, uint32(index))
	}
	return msrs, nil
}

func hasCapExtension(fd uintptr, extension uintptr) bool {

	res, err := kvmIOCtl(fd, uintptr(C.IoctlCheckExtension), extension)
	if res != 1 || err != 0 {
		return true
	}

	return false
}

func kvmIOCtl(fd, flag, arg uintptr) (res uintptr, err syscall.Errno) {
	res, _, err = syscall.Syscall(syscall.SYS_IOCTL,
		fd,
		flag,
		arg)
	return
}

func (s *CapScan) populateCaps() {
	msrs, err := s.getKVMMSRs()
	if err != nil {
		return
	}
	s.supportedMSRS = make(map[uint32]bool)
	for _, kvmMsr := range msrs {
		s.supportedMSRS[kvmMsr] = true
	}
}

func (s *CapScan) getKVMMSRs() ([]uint32, error) {
	devkvm, err := os.OpenFile("/dev/kvm", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		devkvm.Close()
		return nil, err
	}
	defer devkvm.Close()

	fd := devkvm.Fd()
	s.fd = fd
	msrs, err := availableMsrs(fd)
	if err != nil {
		return nil, err
	}
	return msrs, nil
}

func (s *CapScan) exposeCapabilities(exposeEnabled bool) []string {
	exposedCaps := []string{}
	for _, capb := range CapsDesc {
		shouldExpose := true
		if capb.MSR != 0 {
			_, isEnabled := s.supportedMSRS[uint32(capb.MSR)]
			shouldExpose = isEnabled
		}
		if capb.Extension != 0 {
			res := hasCapExtension(s.fd, capb.Extension)
			shouldExpose = shouldExpose && res
		}
		if exposeEnabled && shouldExpose {
			exposedCaps = append(exposedCaps, capb.Name)
		}
	}
	return exposedCaps

}

func (s *CapScan) getLabels() map[string]bool {
	prefix := "/kvm-info-cap-hyperv-"

	s.populateCaps()
	caps := s.exposeCapabilities(true)
	labels := make(map[string]bool)
	for _, capb := range caps {
		labels[fmt.Sprintf("%s%s", prefix, capb)] = true
	}
	return labels
}
