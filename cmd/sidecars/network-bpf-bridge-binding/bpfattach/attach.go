package bpfattach

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cilium/ebpf"
)

const pinSubdir = "kubevirt-bpf-bridge-binding"

type bridgePorts struct {
	TapIfindex  uint32
	VethIfindex uint32
}

// Attach loads bpf_bridge.o, writes tap/veth ifindexes into bridge_cfg, pins prog + map,
// and attaches the TC program via tc (clsact ingress) on both interfaces.
func Attach(objPath string, tapName, vethName string, tapIdx, vethIdx int) error {
	spec, err := ebpf.LoadCollectionSpec(objPath)
	if err != nil {
		return fmt.Errorf("load BPF spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("load BPF collection: %w", err)
	}
	defer coll.Close()

	cfgMap, ok := coll.Maps["bridge_cfg"]
	if !ok {
		return fmt.Errorf("BPF object missing bridge_cfg map")
	}
	prog, ok := coll.Programs["tc_l2_proxy"]
	if !ok {
		return fmt.Errorf("BPF object missing tc_l2_proxy program")
	}

	k := uint32(0)
	val := bridgePorts{
		TapIfindex:  uint32(tapIdx),
		VethIfindex: uint32(vethIdx),
	}
	if err := cfgMap.Update(k, val, ebpf.UpdateAny); err != nil {
		return fmt.Errorf("update bridge_cfg: %w", err)
	}

	bpffs := "/sys/fs/bpf"
	base := filepath.Join(bpffs, pinSubdir)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", base, err)
	}

	mapPin := filepath.Join(base, "bridge_cfg")
	progPin := filepath.Join(base, "tc_l2_proxy")
	_ = os.Remove(mapPin)
	_ = os.Remove(progPin)

	if err := cfgMap.Pin(mapPin); err != nil {
		return fmt.Errorf("pin map: %w", err)
	}
	if err := prog.Pin(progPin); err != nil {
		return fmt.Errorf("pin program: %w", err)
	}

	for _, dev := range []string{tapName, vethName} {
		if err := ensureClsact(dev); err != nil {
			return err
		}
		if err := replaceIngressBPF(dev, progPin); err != nil {
			return err
		}
	}
	return nil
}

func ensureClsact(dev string) error {
	// replace is idempotent for qdisc type
	out, err := exec.Command("tc", "qdisc", "replace", "dev", dev, "clsact").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tc qdisc replace dev %s clsact: %w: %s", dev, err, out)
	}
	return nil
}

func replaceIngressBPF(dev, progPin string) error {
	_ = exec.Command("tc", "filter", "del", "dev", dev, "ingress").Run()
	out, err := exec.Command(
		"tc", "filter", "add", "dev", dev, "ingress",
		"bpf", "da", "pinned", progPin,
	).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tc filter add dev %s ingress bpf pinned: %w: %s", dev, err, out)
	}
	return nil
}

// Detach removes TC filters and clsact qdisc from the devices; does not unpin pinned objects.
func Detach(devices ...string) {
	for _, dev := range devices {
		_ = exec.Command("tc", "filter", "del", "dev", dev, "ingress").Run()
		_ = exec.Command("tc", "qdisc", "del", "dev", dev, "clsact").Run()
	}
}
