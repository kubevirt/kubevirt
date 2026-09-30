package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/godbus/dbus/v5"
	cgroups "github.com/opencontainers/cgroups"
	"github.com/opencontainers/cgroups/systemd"
)

const (
	// hostSystemdPrivateSocket is the private socket of the host's systemd.
	// It is reached through the host root so that no D-Bus daemon or socket
	// mount is needed inside the virt-handler pod.
	hostSystemdPrivateSocket = "unix:path=/proc/1/root/run/systemd/private"
	systemdCallTimeout       = 2 * time.Second
)

// syncSystemdDeviceAllow writes the device rules into DeviceAllow= of the
// systemd scope unit that owns one of the given cgroup paths.
//
// With the systemd cgroup driver runc hands the container's device rules to
// systemd as DeviceAllow= with DevicePolicy=strict. systemd compiles its own
// BPF_CGROUP_DEVICE program from them and attaches it again each time it
// re-applies the unit's cgroup settings, e.g. on daemon-reload or when
// AllowedCPUs= changes. The kernel grants a device access only if every
// attached program allows it, so a device allowed only in the program
// installed by virt-handler becomes denied again. Keeping DeviceAllow= equal
// to the rules virt-handler applies makes the program systemd generates allow
// the same devices.
//
// This follows what the systemd cgroup manager of runc does: it first sets the
// unit properties and then applies the same resources through the cgroupfs
// manager. systemd cannot express some device rules (e.g. "*:N" wildcard-major
// rules), so the properties are a best-effort base that keeps systemd from
// reverting the cgroup, and the exact rules come from the program attached
// by the cgroupfs manager afterwards. See UnifiedManager.Set and
// systemdProperties:
// https://github.com/opencontainers/cgroups/blob/v0.0.6/systemd/v2.go#L500-L517
// https://github.com/opencontainers/cgroups/blob/v0.0.6/devices/systemd.go#L104-L106
//
// A scope with BPFProgram=device:... (crun) gets its device program from the
// pinned file instead of DeviceAllow=, so it is left untouched.
func syncSystemdDeviceAllow(paths map[string]string, resources *cgroups.Resources) error {
	unit := scopeUnitName(paths)
	if unit == "" {
		return nil
	}
	if systemd.GenerateDeviceProps == nil {
		return fmt.Errorf("systemd device properties generator is not registered")
	}

	ctx, cancel := context.WithTimeout(context.Background(), systemdCallTimeout)
	defer cancel()

	conn, err := systemdDbus.NewConnection(func() (*dbus.Conn, error) {
		return dialHostSystemd(ctx)
	})
	if err != nil {
		return fmt.Errorf("cannot connect to systemd: %w", err)
	}
	defer conn.Close()

	if hasDeviceBPFProgram(ctx, conn, unit) {
		return nil
	}

	versionStr, err := conn.GetManagerProperty("Version")
	if err != nil {
		return fmt.Errorf("cannot get systemd version: %w", err)
	}
	version, err := systemdVersionAtoi(versionStr)
	if err != nil {
		return err
	}

	// The generated properties reset DeviceAllow= first, so the unit ends up
	// with exactly the given rules and removed devices are dropped as well.
	props, err := systemd.GenerateDeviceProps(resources, version)
	if err != nil {
		return fmt.Errorf("cannot generate systemd device properties: %w", err)
	}
	if len(props) == 0 {
		return nil
	}

	if err := conn.SetUnitPropertiesContext(ctx, unit, true, props...); err != nil {
		return fmt.Errorf("cannot set device properties of unit %s: %w", unit, err)
	}
	return nil
}

// scopeUnitName returns the name of the systemd scope unit among the cgroup
// paths, or an empty string if the cgroups are not managed by systemd.
// With crun the target path is the "container" sub-cgroup and the scope is
// passed as the parent path.
func scopeUnitName(paths map[string]string) string {
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if name := filepath.Base(paths[key]); strings.HasSuffix(name, ".scope") {
			return name
		}
	}
	return ""
}

// hasDeviceBPFProgram reports whether the unit has a device program set via
// BPFProgram=. An error, e.g. an unknown property on systemd older than v249,
// is treated as not set.
func hasDeviceBPFProgram(ctx context.Context, conn *systemdDbus.Conn, unit string) bool {
	prop, err := conn.GetUnitTypePropertyContext(ctx, unit, "Scope", "BPFProgram")
	if err != nil {
		return false
	}

	var entries []struct {
		Type string
		Path string
	}
	if err := prop.Value.Store(&entries); err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.Type == "device" && entry.Path != "" {
			return true
		}
	}
	return false
}

// dialHostSystemd opens a direct connection to the host's systemd. The
// private socket does not take the Hello call, only EXTERNAL authentication.
func dialHostSystemd(ctx context.Context) (*dbus.Conn, error) {
	conn, err := dbus.Dial(hostSystemdPrivateSocket, dbus.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if err := conn.Auth([]dbus.Auth{dbus.AuthExternal(strconv.Itoa(os.Getuid()))}); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

// systemdVersionAtoi extracts the numeric systemd version from strings such
// as "v245.4-1.fc32", "245" or "\"255.4-1\"". It is a copy of the unexported
// helper in github.com/opencontainers/cgroups/systemd.
func systemdVersionAtoi(str string) (int, error) {
	str = strings.TrimLeft(str, `"v`)
	for i := range len(str) {
		if str[i] < '0' || str[i] > '9' {
			str = str[:i]
			break
		}
	}
	version, err := strconv.Atoi(str)
	if err != nil {
		return -1, fmt.Errorf("cannot parse systemd version %q: %w", str, err)
	}
	return version, nil
}
