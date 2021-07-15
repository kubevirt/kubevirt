package cgroup

import (
	"fmt"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
	"math"
	"os/exec"
	"strconv"

	"github.com/cilium/ebpf/asm"
	runc_cgroups "github.com/opencontainers/runc/libcontainer/cgroups"
	devices2 "github.com/opencontainers/runc/libcontainer/cgroups/devices"
	"github.com/opencontainers/runc/libcontainer/cgroups/ebpf"
	runc_fs "github.com/opencontainers/runc/libcontainer/cgroups/fs2"
	runc_configs "github.com/opencontainers/runc/libcontainer/configs"
	"github.com/opencontainers/runc/libcontainer/devices"
	"github.com/pkg/errors"
	"golang.org/x/sys/unix"

	"kubevirt.io/client-go/log"
)

type v2Manager struct {
	runc_cgroups.Manager
}

func newV2Manager(config *runc_configs.Cgroup, dirPath string, rootless bool) (Manager, error) {
	runcManager, err := runc_fs.NewManager(config, dirPath, rootless)
	manager := v2Manager{runcManager}

	return &manager, err
}

func (v *v2Manager) GetBasePathToHostController(controller string) (string, error) {
	return getBasePathToHostController(controller)
}

func (v *v2Manager) Set(r *runc_configs.Resources) error {
	//if err := setDevices(v.Manager.Path(""), r); err != nil {
	//	log.Log.Infof("hotplug [SETv2] - setting device rules. err: %v", err)
	//	return err
	//}

	//resourcesWithoutDevices := *r
	//resourcesWithoutDevices.Devices = nil
	//
	//return v.Manager.Set(&resourcesWithoutDevices)

	//return v.Manager.Set(r)

	cgroupsV2DeviceMutatorArgs := []string{
		"create-tap",
		"--tap-name", "tapName",
	}
	// #nosec No risk for attacket injection. createTapDeviceArgs includes predefined strings
	cmd := exec.Command("virt-chroot", cgroupsV2DeviceMutatorArgs...)
	//return selinux.NewContextExecutorFromPid(virtLauncherPID, cmd)
}

type program struct {
	insts        asm.Instructions
	defaultAllow bool
	blockID      int
}

func setDevices(dirPath string, r *runc_configs.Resources) error {
	if r.SkipDevices {
		return nil
	}
	insts, license, err := DeviceFilter(r.Devices)
	log.Log.Infof("hotplug [setDevices] - got device filter. err: %v", err)
	if err != nil {
		return err
	}
	dirFD, err := unix.Open(dirPath, unix.O_DIRECTORY|unix.O_RDONLY, 0o600)
	log.Log.Infof("hotplug [setDevices] - openning dir (%s). err: %v", dirPath, err)
	if err != nil {
		return errors.Errorf("cannot get dir FD for %s", dirPath)
	}
	defer unix.Close(dirFD)
	if _, err := ebpf.LoadAttachCgroupDeviceFilter(insts, license, dirFD); err != nil {
		log.Log.Infof("hotplug [setDevices] - err loading filter: %v", err)
		return err
	}
	return nil
}

func DeviceFilter(rules []*devices.Rule) (asm.Instructions, string, error) {
	// Generate the minimum ruleset for the device rules we are given. While we
	// don't care about minimum transitions in cgroupv2, using the emulator
	// gives us a guarantee that the behaviour of devices filtering is the same
	// as cgroupv1, including security hardenings to avoid misconfiguration
	// (such as punching holes in wildcard rules).
	emu := new(devices2.Emulator)
	for _, rule := range rules {
		if err := emu.Apply(*rule); err != nil {
			return nil, "", err
		}
	}
	//var err error
	//cleanRules := rules
	cleanRules, err := emu.Rules()
	if err != nil {
		return nil, "", err
	}

	debugToRuleSlice := func(rules []*devices.Rule) []devices.Rule {
		newRules := make([]devices.Rule, 0)
		for _, rule := range rules {
			newRules = append(newRules, *rule)
		}
		return newRules
	}
	log.Log.Infof("hotplug [DeviceFilter] original rules: %+v", debugToRuleSlice(rules))
	log.Log.Infof("hotplug [DeviceFilter] cleaned rules: %+v", debugToRuleSlice(cleanRules))

	p := &program{
		defaultAllow: emu.IsBlacklist(),
	}
	p.init()

	for idx, rule := range cleanRules {
		log.Log.Infof("hotplug [DeviceFilter] APPLYING RULE: %+v", rule)
		if rule.Type == devices.WildcardDevice {
			// We can safely skip over wildcard entries because there should
			// only be one (at most) at the very start to instruct cgroupv1 to
			// go into allow-list mode. However we do double-check this here.
			if idx != 0 || rule.Allow != emu.IsBlacklist() {
				log.Log.Infof("hotplug [DeviceFilter] [internal error] emulated cgroupv2 devices ruleset had bad wildcard at idx %v (%s)", idx, rule.CgroupString())
				return nil, "", errors.Errorf("[internal error] emulated cgroupv2 devices ruleset had bad wildcard at idx %v (%s)", idx, rule.CgroupString())
			}
			continue
		}
		if rule.Allow == p.defaultAllow {
			// There should be no rules which have an action equal to the
			// default action, the emulator removes those.
			log.Log.Infof("hotplug [DeviceFilter] [internal error] emulated cgroupv2 devices ruleset had no-op rule at idx %v (%s)", idx, rule.CgroupString())
			return nil, "", errors.Errorf("[internal error] emulated cgroupv2 devices ruleset had no-op rule at idx %v (%s)", idx, rule.CgroupString())
		}
		if err := p.appendRule(rule); err != nil {
			log.Log.Infof("hotplug [DeviceFilter] appendRule err: ", err)
			return nil, "", err
		}
	}
	insts, err := p.finalize()
	log.Log.Infof("hotplug [DeviceFilter] finalize err: %v", err)
	return insts, "Apache", err
}

func (p *program) init() {
	// struct bpf_cgroup_dev_ctx: https://elixir.bootlin.com/linux/v5.3.6/source/include/uapi/linux/bpf.h#L3423
	/*
		u32 access_type
		u32 major
		u32 minor
	*/
	// R2 <- type (lower 16 bit of u32 access_type at R1[0])
	p.insts = append(p.insts,
		asm.LoadMem(asm.R2, asm.R1, 0, asm.Word),
		asm.And.Imm32(asm.R2, 0xFFFF))

	// R3 <- access (upper 16 bit of u32 access_type at R1[0])
	p.insts = append(p.insts,
		asm.LoadMem(asm.R3, asm.R1, 0, asm.Word),
		// RSh: bitwise shift right
		asm.RSh.Imm32(asm.R3, 16))

	// R4 <- major (u32 major at R1[4])
	p.insts = append(p.insts,
		asm.LoadMem(asm.R4, asm.R1, 4, asm.Word))

	// R5 <- minor (u32 minor at R1[8])
	p.insts = append(p.insts,
		asm.LoadMem(asm.R5, asm.R1, 8, asm.Word))
}

func (p *program) appendRule(rule *devices.Rule) error {
	if p.blockID < 0 {
		return errors.New("the program is finalized")
	}

	var bpfType int32
	switch rule.Type {
	case devices.CharDevice:
		bpfType = int32(unix.BPF_DEVCG_DEV_CHAR)
	case devices.BlockDevice:
		bpfType = int32(unix.BPF_DEVCG_DEV_BLOCK)
	default:
		// We do not permit 'a', nor any other types we don't know about.
		return errors.Errorf("invalid type %q", string(rule.Type))
	}
	if rule.Major > math.MaxUint32 {
		return errors.Errorf("invalid major %d", rule.Major)
	}
	if rule.Minor > math.MaxUint32 {
		return errors.Errorf("invalid minor %d", rule.Major)
	}
	hasMajor := rule.Major >= 0 // if not specified in OCI json, major is set to -1
	hasMinor := rule.Minor >= 0
	bpfAccess := int32(0)
	for _, r := range rule.Permissions {
		switch r {
		case 'r':
			bpfAccess |= unix.BPF_DEVCG_ACC_READ
		case 'w':
			bpfAccess |= unix.BPF_DEVCG_ACC_WRITE
		case 'm':
			bpfAccess |= unix.BPF_DEVCG_ACC_MKNOD
		default:
			return errors.Errorf("unknown device access %v", r)
		}
	}
	// If the access is rwm, skip the check.
	hasAccess := bpfAccess != (unix.BPF_DEVCG_ACC_READ | unix.BPF_DEVCG_ACC_WRITE | unix.BPF_DEVCG_ACC_MKNOD)

	var (
		blockSym         = "block-" + strconv.Itoa(p.blockID)
		nextBlockSym     = "block-" + strconv.Itoa(p.blockID+1)
		prevBlockLastIdx = len(p.insts) - 1
	)
	p.insts = append(p.insts,
		// if (R2 != bpfType) goto next
		asm.JNE.Imm(asm.R2, bpfType, nextBlockSym),
	)
	if hasAccess {
		p.insts = append(p.insts,
			// if (R3 & bpfAccess != R3 /* use R1 as a temp var */) goto next
			asm.Mov.Reg32(asm.R1, asm.R3),
			asm.And.Imm32(asm.R1, bpfAccess),
			asm.JNE.Reg(asm.R1, asm.R3, nextBlockSym),
		)
	}
	if hasMajor {
		p.insts = append(p.insts,
			// if (R4 != major) goto next
			asm.JNE.Imm(asm.R4, int32(rule.Major), nextBlockSym),
		)
	}
	if hasMinor {
		p.insts = append(p.insts,
			// if (R5 != minor) goto next
			asm.JNE.Imm(asm.R5, int32(rule.Minor), nextBlockSym),
		)
	}
	p.insts = append(p.insts, acceptBlock(rule.Allow)...)
	// set blockSym to the first instruction we added in this iteration
	p.insts[prevBlockLastIdx+1] = p.insts[prevBlockLastIdx+1].Sym(blockSym)
	p.blockID++
	return nil
}

func (p *program) finalize() (asm.Instructions, error) {
	var v int32
	if p.defaultAllow {
		v = 1
	}
	blockSym := "block-" + strconv.Itoa(p.blockID)
	p.insts = append(p.insts,
		// R0 <- v
		asm.Mov.Imm32(asm.R0, v).Sym(blockSym),
		asm.Return(),
	)
	p.blockID = -1
	return p.insts, nil
}

func acceptBlock(accept bool) asm.Instructions {
	var v int32
	if accept {
		v = 1
	}
	return []asm.Instruction{
		// R0 <- v
		asm.Mov.Imm32(asm.R0, v),
		asm.Return(),
	}
}
