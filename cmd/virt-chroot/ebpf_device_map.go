package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"golang.org/x/sys/unix"

	cgroupconsts "kubevirt.io/kubevirt/pkg/virt-handler/cgroup/constants"
)

const (
	deviceMapPinBase    = "/sys/fs/bpf/kubevirt"
	deviceMapMaxEntries = 256
)

var errNoPrograms = errors.New("no BPF_CGROUP_DEVICE programs attached")

// deviceKey is the BPF map key for device lookups.
// Matches the layout used in the eBPF program: {type, major, minor}.
type deviceKey struct {
	Type  uint32
	Major uint32
	Minor uint32
}

// spliceDeviceMapLookup reads the existing BPF_CGROUP_DEVICE programs attached
// to one or more cgroup paths, appends a BPF hash map lookup before each
// program's final deny, and reattaches the modified programs. A single shared
// map is created and pinned using the first (primary) path. The map is
// initially empty and used for later hotplug updates.
//
// The existing program (set by the container runtime) is a linear if-chain:
//
//	init: load R2=type, R3=access, R4=major, R5=minor
//	block-0: if match → allow
//	block-1: if match → allow
//	...
//	block-N: mov r0, 0; exit  ← final deny
//
// We replace the final deny with: map lookup → if found ALLOW → DENY.
// All existing jump offsets remain valid since we replace at the same position.
func spliceDeviceMapLookup(cgroupPaths []string) (string, error) {
	if len(cgroupPaths) == 0 {
		return "", fmt.Errorf("no cgroup paths provided")
	}

	// If the map is already pinned, the splice was already done. This should
	// not normally happen unless virt-handler was restarted (which clears its
	// in-memory splice cache). Harmless — just skip.
	pinPath := deviceMapPinPath(cgroupPaths[0])
	if _, err := os.Stat(pinPath); err == nil {
		fmt.Fprintf(os.Stderr, "device map already exists at %s, skipping splice\n", pinPath)
		return pinPath, nil
	}

	// Create one shared device map for all paths.
	deviceMap, err := ebpf.NewMap(&ebpf.MapSpec{
		Name:       "hotplug_devices",
		Type:       ebpf.Hash,
		KeySize:    uint32(unsafe.Sizeof(deviceKey{})),
		ValueSize:  4, // u32 permissions bitmask
		MaxEntries: deviceMapMaxEntries,
	})
	if err != nil {
		return "", fmt.Errorf("cannot create device map: %w", err)
	}
	defer deviceMap.Close()

	var spliced int
	for _, cgroupPath := range cgroupPaths {
		if err := spliceSingleCgroup(cgroupPath, deviceMap); err != nil {
			if errors.Is(err, errNoPrograms) {
				fmt.Fprintf(os.Stderr, "no BPF_CGROUP_DEVICE programs at %s, skipping\n", cgroupPath)
				continue
			}
			return "", fmt.Errorf("splice failed for %s: %w", cgroupPath, err)
		}
		spliced++
	}
	if spliced == 0 {
		return "", fmt.Errorf("no BPF_CGROUP_DEVICE programs found on any of the provided cgroup paths: %v", cgroupPaths)
	}

	// Pin the shared map using the primary (first) path.
	if err := os.MkdirAll(filepath.Dir(pinPath), 0o700); err != nil {
		return "", fmt.Errorf("cannot create pin directory: %w", err)
	}
	if err := deviceMap.Pin(pinPath); err != nil {
		return "", fmt.Errorf("cannot pin device map to %s: %w", pinPath, err)
	}

	fmt.Fprintf(os.Stderr, "spliced device map lookup into %d cgroup(s), map pinned at %s\n", spliced, pinPath)
	return pinPath, nil
}

// spliceSingleCgroup reads the attached program for one cgroup path, appends
// the map lookup tail referencing deviceMap, and replaces the old program.
func spliceSingleCgroup(cgroupPath string, deviceMap *ebpf.Map) error {
	dirFD, err := unix.Open(cgroupPath, unix.O_DIRECTORY|unix.O_RDONLY, 0o600)
	if err != nil {
		return fmt.Errorf("cannot open cgroup dir %s: %w", cgroupPath, err)
	}
	defer unix.Close(dirFD)

	oldProgs, err := findAttachedDeviceProgs(dirFD)
	if err != nil {
		return fmt.Errorf("cannot query attached device programs: %w", err)
	}
	if len(oldProgs) == 0 {
		return fmt.Errorf("%w to %s", errNoPrograms, cgroupPath)
	}
	defer func() {
		for _, p := range oldProgs {
			p.Close()
		}
	}()

	// Read the instructions from the first (primary) attached program.
	info, err := oldProgs[0].Info()
	if err != nil {
		return fmt.Errorf("cannot get program info: %w", err)
	}
	insns, err := info.Instructions()
	if err != nil {
		return fmt.Errorf("cannot read program instructions: %w", err)
	}

	n := len(insns)
	if n < 2 {
		return fmt.Errorf("program too short (%d instructions)", n)
	}

	fmt.Fprintf(os.Stderr, "original program for %s (%d instructions):\n%v\n", cgroupPath, n, insns)

	// Verify the last 2 instructions are: mov r0, 0 ; exit (default deny).
	// We accept both Mov.Imm32 (runc/containerd/Docker/Mirantis via
	// opencontainers/cgroups devicefilter.go) and Mov.Imm aka MOV64_IMM
	// (crun via ebpf.c, default on CRI-O/OpenShift).
	lastMov := insns[n-2]
	lastExit := insns[n-1]
	movOK := lastMov.Dst == asm.R0 && lastMov.Constant == 0 &&
		(lastMov.OpCode == asm.Mov.Imm(asm.R0, 0).OpCode || lastMov.OpCode == asm.Mov.Imm32(asm.R0, 0).OpCode)
	if !movOK {
		return fmt.Errorf("unexpected second-to-last instruction (expected mov r0, 0): %v", lastMov)
	}
	if lastExit.OpCode != asm.Return().OpCode {
		return fmt.Errorf("unexpected last instruction (expected exit): %v", lastExit)
	}

	// Truncate the final deny block.
	insns = insns[:n-2]

	// Build the map lookup tail.
	// At this point R2=type, R3=access, R4=major, R5=minor are live
	// (never clobbered by the if-chain, only R1 is used as temp).
	loadMap := asm.LoadMapPtr(asm.R1, 0) // FD placeholder, resolved via AssociateMap
	if err := loadMap.AssociateMap(deviceMap); err != nil {
		return fmt.Errorf("cannot associate device map with instruction: %w", err)
	}

	mapLookupInsns := asm.Instructions{
		// Build key on stack: {type, major, minor}
		asm.StoreMem(asm.R10, -12, asm.R2, asm.Word), // *(u32*)(fp-12) = type
		asm.StoreMem(asm.R10, -8, asm.R4, asm.Word),  // *(u32*)(fp-8)  = major
		asm.StoreMem(asm.R10, -4, asm.R5, asm.Word),  // *(u32*)(fp-4)  = minor

		// Save R3 (requested access) on stack — BPF helper clobbers R1-R5
		asm.StoreMem(asm.R10, -16, asm.R3, asm.Word), // *(u32*)(fp-16) = access

		loadMap,
		// R2 = &key
		asm.Mov.Reg(asm.R2, asm.R10),
		asm.Add.Imm(asm.R2, -12),

		asm.FnMapLookupElem.Call(), // R0 = bpf_map_lookup_elem(map, &key)

		asm.JEq.Imm(asm.R0, 0, "dm_deny"), // if NULL goto deny

		// Found: check permissions.
		// R0 = pointer to map value (u32 allowed_perms)
		asm.LoadMem(asm.R1, asm.R0, 0, asm.Word),    // R1 = allowed_perms
		asm.LoadMem(asm.R2, asm.R10, -16, asm.Word), // R2 = requested_access (saved)
		asm.Mov.Reg32(asm.R0, asm.R2),               // R0 = requested_access
		asm.And.Reg32(asm.R0, asm.R1),               // R0 = requested & allowed
		asm.JNE.Reg32(asm.R0, asm.R2, "dm_deny"),    // if not all perms granted → deny

		asm.Mov.Imm(asm.R0, 1), // ALLOW
		asm.Return(),
	}

	denyInsns := asm.Instructions{
		asm.Mov.Imm(asm.R0, 0).WithSymbol("dm_deny"), // DENY
		asm.Return(),
	}

	insns = append(insns, mapLookupInsns...)
	insns = append(insns, denyInsns...)

	fmt.Fprintf(os.Stderr, "spliced tail for %s (%d new instructions appended at position %d):\n%v\n", cgroupPath, len(insns)-(n-2), n-2, insns[n-2:])

	// Load the modified program.
	newProg, err := ebpf.NewProgram(&ebpf.ProgramSpec{
		Type:         ebpf.CGroupDevice,
		Instructions: insns,
		License:      "Apache",
	})
	if err != nil {
		return fmt.Errorf("cannot load modified device filter program: %w", err)
	}

	// Attach new program with ALLOW_MULTI.
	err = link.RawAttachProgram(link.RawAttachProgramOptions{
		Target:  dirFD,
		Program: newProg,
		Attach:  ebpf.AttachCGroupDevice,
		Flags:   unix.BPF_F_ALLOW_MULTI,
	})
	if err != nil {
		newProg.Close()
		return fmt.Errorf("cannot attach modified program: %w", err)
	}

	// Detach all old programs.
	for _, oldProg := range oldProgs {
		_ = link.RawDetachProgram(link.RawDetachProgramOptions{
			Target:  dirFD,
			Program: oldProg,
			Attach:  ebpf.AttachCGroupDevice,
		})
	}

	// Re-pin the spliced program over the container runtime's pin so that
	// systemd re-attaches the spliced version on scope property changes
	// (e.g. AllowedCPUs). Without this, systemd would re-read the original
	// program from the pin and the AND semantics of BPF_F_ALLOW_MULTI
	// would deny hotplugged devices.
	//
	// On SELinux-enforcing systems systemd (init_t) will get "Permission denied"
	// when trying to re-attach because the spliced program has the spc_t label
	// from virt-chroot rather than the container_runtime_t label from crun.
	// This is benign: systemd leaves the already-attached spliced program untouched.
	if pinPath, err := bpfPinPathFromSystemd(cgroupPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: cannot determine BPFProgram pin path for %s: %v\n", cgroupPath, err)
	} else if pinPath != "" {
		if err := os.Remove(pinPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "warning: cannot remove old pin %s: %v\n", pinPath, err)
		}
		if err := newProg.Pin(pinPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot re-pin spliced program to %s: %v\n", pinPath, err)
		} else {
			fmt.Fprintf(os.Stderr, "re-pinned spliced program to %s\n", pinPath)
		}
	}

	return nil
}

// bpfPinPathFromSystemd returns the bpffs pin path for the BPFProgram=device
// property of the systemd scope owning cgroupPath, or ("", nil) if unset.
func bpfPinPathFromSystemd(cgroupPath string) (string, error) {
	conn, err := systemdDbus.NewWithContext(context.Background())
	if err != nil {
		return "", err
	}
	defer conn.Close()

	prop, err := conn.GetUnitTypePropertyContext(context.Background(), filepath.Base(cgroupPath), "Scope", "BPFProgram")
	if err != nil {
		return "", err
	}

	// D-Bus type a(ss): [(attach_type, path), ...]
	if entries, ok := prop.Value.Value().([][]interface{}); ok {
		for _, e := range entries {
			if len(e) == 2 {
				if t, _ := e[0].(string); t == "device" {
					if p, _ := e[1].(string); p != "" {
						return p, nil
					}
				}
			}
		}
	}
	return "", nil
}

// updateDeviceMap adds or removes a device entry in a pinned BPF device map.
func updateDeviceMap(pinPath string, devType uint32, major uint32, minor uint32, permissions uint32, allow bool) error {
	m, err := ebpf.LoadPinnedMap(pinPath, nil)
	if err != nil {
		return fmt.Errorf("cannot open pinned map %s: %w", pinPath, err)
	}
	defer m.Close()

	key := deviceKey{Type: devType, Major: major, Minor: minor}

	if allow {
		if err := m.Put(&key, &permissions); err != nil {
			return fmt.Errorf("cannot add device (%d, %d:%d) to map: %w", devType, major, minor, err)
		}
	} else {
		if err := m.Delete(&key); err != nil {
			if errors.Is(err, ebpf.ErrKeyNotExist) {
				fmt.Fprintf(os.Stderr, "warning: device (%d, %d:%d) not in map, nothing to remove\n", devType, major, minor)
				return nil
			}
			return fmt.Errorf("cannot remove device (%d, %d:%d) from map: %w", devType, major, minor, err)
		}
	}
	return nil
}

// deviceMapPinPath returns a deterministic bpffs pin path for a cgroup's device map.
func deviceMapPinPath(cgroupPath string) string {
	h := sha256.Sum256([]byte(cgroupPath))
	return filepath.Join(deviceMapPinBase, hex.EncodeToString(h[:16]))
}

// findAttachedDeviceProgs queries BPF_CGROUP_DEVICE programs attached to a cgroup.
// Based on opencontainers/cgroups/devices/ebpf_linux.go:findAttachedCgroupDeviceFilters
// (unexported, so we maintain a copy).
func findAttachedDeviceProgs(dirFD int) ([]*ebpf.Program, error) {
	type bpfAttrQuery struct {
		TargetFd    uint32
		AttachType  uint32
		QueryType   uint32
		AttachFlags uint32
		ProgIds     uint64
		ProgCnt     uint32
	}

	size := 64
	for retries := 0; retries < 10; retries++ {
		progIDs := make([]uint32, size)
		query := bpfAttrQuery{
			TargetFd:   uint32(dirFD),
			AttachType: uint32(unix.BPF_CGROUP_DEVICE),
			ProgIds:    uint64(uintptr(unsafe.Pointer(&progIDs[0]))),
			ProgCnt:    uint32(len(progIDs)),
		}

		_, _, errno := unix.Syscall(unix.SYS_BPF,
			uintptr(unix.BPF_PROG_QUERY),
			uintptr(unsafe.Pointer(&query)),
			unsafe.Sizeof(query))
		size = int(query.ProgCnt)
		runtime.KeepAlive(query)
		if errno != 0 {
			if errno == unix.ENOSPC {
				continue
			}
			return nil, fmt.Errorf("BPF_PROG_QUERY(BPF_CGROUP_DEVICE) failed: %w", errno)
		}

		progIDs = progIDs[:size]
		programs := make([]*ebpf.Program, 0, len(progIDs))
		for _, id := range progIDs {
			p, err := ebpf.NewProgramFromID(ebpf.ProgramID(id))
			if err != nil {
				if errors.Is(err, os.ErrPermission) {
					continue
				}
				return nil, fmt.Errorf("cannot get program from id %d: %w", id, err)
			}
			programs = append(programs, p)
		}
		runtime.KeepAlive(progIDs)
		return programs, nil
	}
	return nil, fmt.Errorf("could not get list of attached CGROUP_DEVICE programs")
}

// listDeviceMap iterates a pinned BPF device map and returns all entries.
func listDeviceMap(pinPath string) ([]cgroupconsts.DeviceMapEntry, error) {
	m, err := ebpf.LoadPinnedMap(pinPath, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open pinned map %s: %w", pinPath, err)
	}
	defer m.Close()

	var entries []cgroupconsts.DeviceMapEntry
	var key deviceKey
	var perms uint32

	iter := m.Iterate()
	for iter.Next(&key, &perms) {
		entries = append(entries, cgroupconsts.DeviceMapEntry{
			Type:        u32ToDeviceType(key.Type),
			Major:       key.Major,
			Minor:       key.Minor,
			Permissions: u32ToPermissions(perms),
		})
	}
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("error iterating device map: %w", err)
	}

	return entries, nil
}

func u32ToDeviceType(t uint32) string {
	switch t {
	case unix.BPF_DEVCG_DEV_BLOCK:
		return cgroupconsts.BlockDevice
	case unix.BPF_DEVCG_DEV_CHAR:
		return cgroupconsts.CharDevice
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

func u32ToPermissions(p uint32) string {
	var s []byte
	if p&unix.BPF_DEVCG_ACC_READ != 0 {
		s = append(s, 'r')
	}
	if p&unix.BPF_DEVCG_ACC_WRITE != 0 {
		s = append(s, 'w')
	}
	if p&unix.BPF_DEVCG_ACC_MKNOD != 0 {
		s = append(s, 'm')
	}
	return string(s)
}

// permissionsToU32 converts a string like "rwm" to a bitmask.
func permissionsToU32(perms string) uint32 {
	var v uint32
	for _, c := range perms {
		switch c {
		case 'r':
			v |= unix.BPF_DEVCG_ACC_READ
		case 'w':
			v |= unix.BPF_DEVCG_ACC_WRITE
		case 'm':
			v |= unix.BPF_DEVCG_ACC_MKNOD
		}
	}
	return v
}

// deviceTypeToU32 converts 'b' or 'c' to BPF device type constant.
func deviceTypeToU32(t string) (uint32, error) {
	switch t {
	case cgroupconsts.BlockDevice:
		return unix.BPF_DEVCG_DEV_BLOCK, nil
	case cgroupconsts.CharDevice:
		return unix.BPF_DEVCG_DEV_CHAR, nil
	default:
		return 0, fmt.Errorf("unknown device type %q (expected %q or %q)", t, cgroupconsts.BlockDevice, cgroupconsts.CharDevice)
	}
}
