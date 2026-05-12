// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package sysfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/prometheus/procfs/internal/util"
)

// Documentation of the sysfs path
// https://docs.nvidia.com/networking/display/mlnxofedv571020/explicit+congestion+notification+(ecn)
// https://enterprise-support.nvidia.com/s/article/dcqcn-parameters

// Ecn contains values from /sys/class/net/<iface>/ecn/roce_np/
// for single interface (iface).
type RoceNpEcn struct {
	// A map from a priority to it's enabled status
	Ecn map[uint8]bool
	// Minimum time between sending CNPs from the port, in microseconds.
	// Range: 0-4095, Default: 4
	MinTimeBetweenCnps uint64
	// The DSCP value for CNPs.
	// Range: 0-63, Default: 48
	CnpDscp uint64
	// The PCP value for CNPs.
	// Range: 0-7, Default: 6
	Cnp802pPriority uint64
}

// Ecn contains values from /sys/class/net/<iface>/ecn/roce_rp/
// for single interface (iface).
type RoceRpEcn struct {
	// A map from a priority to it's enabled status
	Ecn map[uint8]bool

	// Alpha Update

	// Every Alpha Update Period alpha is updated.
	// If the CNP is received during this period, alpha is incremented.
	// Otherwise, it is decremented.
	// Range: 0-1023, Default: 1019
	DceTCPG uint64
	// The Alpha Update Period used in the formula for DceTCPG. Unit is microseconds.
	// Range: 1-131071, Default: 1
	DceTCPRtt uint64
	// This parameter sets the initial value of alpha that should be used when receiving
	// the first CNP for a flow. Fixed point with 10 bits in the fraction part.
	// Range: 1-1023, Default: 1023
	InitialAlphaValue uint64

	// Rate Decrease

	// Rates (current, target) on first CNP (0 – 85% of line rate) in Mbps.
	// Range: 0, 1-line rate, Default: 0
	RateToSetOnFirstCnp uint64
	// This parameter defines the maximal ratio of rate decrease in a single event.
	// Range: 0-100, Default: 50
	RpgMinDecFac uint64
	// This parameter defines the minimal rate limit of the QP in Mbps.
	// Range: 1-line rate, Default: 1
	RpgMinRate uint64
	// The coefficient between alpha and the rate reduction factor.
	// Range: 10-11, Default: 11
	RpgGd uint64
	// The time period between rate reductions in microseconds.
	// Range: 0-UINT32, Default: 4
	RateReduceMonitorPeriod uint64

	// Rate Increase

	// If set, every rate decreases. The target rate is updated to the current rate.
	// Otherwise, the target rate is updated to the current rate only on the first
	// decrement after the increment event.
	ClampTgtRate bool
	// The time period between rate increase events in microseconds.
	// Range: 1-131071, Default: 300
	RpgTimeReset uint64
	// The sent bytes counter between rate increase events.
	// Range: 1-32767, Default: 32767
	RpgByteReset uint64
	// The threshold of rate increase events for moving to next rate increase phase.
	// Range: 1-31, Default: 1
	RpgThreshold uint64
	// The rate increase value in the Additive Increase phase in Mbps.
	// Range: 1-line rate, Default: 5
	RpgAiRate uint64
	// The rate increase value in the Hyper Increase phase in Mbps.
	// Range: 1-line rate, Default: 1
	RpgHaiRate uint64
}

// EcnIface contains Ecn info from files in /sys/class/net/<iface>/ecn/
// for single interface (iface).
type EcnIface struct {
	Name string // Interface name
	// protocols
	RoceNpEcn RoceNpEcn // Notification point
	RoceRpEcn RoceRpEcn // Reaction point
}

// AllEcnIface is collection of Ecn info for every interface (iface) in /sys/class/net.
// The map keys are interface (iface) names.
type AllEcnIface map[string]EcnIface

// EcnByIface returns info for a single net interfaces (iface).
func (fs FS) EcnByIface(devicePath string) (*EcnIface, error) {
	_, err := fs.NetClassByIface(devicePath)
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	ecnPath := filepath.Join(path, devicePath, "ecn")
	validPath, err := PathExistsAndIsDir(ecnPath)
	if err != nil {
		return nil, err
	}
	if !validPath {
		// this device doesn't have ECN values at this path
		return nil, fmt.Errorf("does not have ECN values: %q", devicePath)
	}

	ecnIface, err := ParseEcnIfaceInfo(ecnPath)
	if err != nil {
		return nil, err
	}
	ecnIface.Name = devicePath

	return ecnIface, nil
}

// EcnDevices returns EcnIface for all net interfaces (iface) read from /sys/class/net/<iface>/ecn.
func (fs FS) EcnDevices() (AllEcnIface, error) {
	devices, err := fs.NetClassDevices()
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	allEcnIface := AllEcnIface{}
	for _, devicePath := range devices {
		ecnPath := filepath.Join(path, devicePath, "ecn")
		validPath, err := PathExistsAndIsDir(ecnPath)
		if err != nil {
			return nil, err
		}
		if !validPath {
			// this device doesn't have ECN values at this path
			continue
		}
		ecnIface, err := ParseEcnIfaceInfo(ecnPath)
		if err != nil {
			return nil, err
		}
		ecnIface.Name = devicePath
		allEcnIface[devicePath] = *ecnIface
	}

	return allEcnIface, nil
}

// ParseEcnIfaceInfo scans predefined files in /sys/class/net/<iface>/ecn
// directory and gets their contents.
func ParseEcnIfaceInfo(ecnPath string) (*EcnIface, error) {
	ecnIface := EcnIface{}
	err := ParseRoceNpEcnInfo(filepath.Join(ecnPath, "roce_np"), &ecnIface.RoceNpEcn)
	if err != nil {
		return nil, err
	}

	err = ParseRoceRpEcnInfo(filepath.Join(ecnPath, "roce_rp"), &ecnIface.RoceRpEcn)
	if err != nil {
		return nil, err
	}

	return &ecnIface, nil
}

// ParseEcnIfaceInfo scans predefined files in /sys/class/net/<iface>/ecn/roce_np/
// directory and gets their contents.
func ParseRoceNpEcnInfo(ecnPath string, ecn *RoceNpEcn) error {
	value, err := ParseEcnEnable(filepath.Join(ecnPath, "enable"))
	if err != nil {
		return err
	}
	ecn.Ecn = value

	files, err := os.ReadDir(ecnPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.Type().IsRegular() {
			continue
		}
		if err := ParseRoceNpEcnAttribute(ecnPath, f.Name(), ecn); err != nil {
			return err
		}
	}
	return nil
}

// Parses all of the attributes in for ROCE NP protocol.
func ParseRoceNpEcnAttribute(ecnPath string, attrName string, ecn *RoceNpEcn) error {
	attrPath := filepath.Join(ecnPath, attrName)
	value, err := util.SysReadFile(attrPath)
	if err != nil {
		if canIgnoreError(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", attrPath, err)
	}

	vp := util.NewValueParser(value)
	switch attrName {
	case "min_time_between_cnps":
		ecn.MinTimeBetweenCnps = *vp.PUInt64()
	case "cnp_802p_prio":
		ecn.Cnp802pPriority = *vp.PUInt64()
	case "cnp_dscp":
		ecn.CnpDscp = *vp.PUInt64()
	default:
		return nil
	}

	return nil
}

// ParseRoceRpEcnInfo scans predefined files in /sys/class/net/<iface>/ecn/roce_rp/
// directory and gets their contents.
func ParseRoceRpEcnInfo(ecnPath string, ecn *RoceRpEcn) error {
	value, err := ParseEcnEnable(filepath.Join(ecnPath, "enable"))
	if err != nil {
		return err
	}
	ecn.Ecn = value

	files, err := os.ReadDir(ecnPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !f.Type().IsRegular() {
			continue
		}
		if err := ParseRoceRpEcnAttribute(ecnPath, f.Name(), ecn); err != nil {
			return err
		}
	}
	return nil
}

// Parses all of the attributes in for ROCE RP protocol.
func ParseRoceRpEcnAttribute(ecnPath string, attrName string, ecn *RoceRpEcn) error {
	attrPath := filepath.Join(ecnPath, attrName)
	value, err := util.SysReadFile(attrPath)
	if err != nil {
		if canIgnoreError(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", attrPath, err)
	}

	vp := util.NewValueParser(value)
	switch attrName {
	case "clamp_tgt_rate":
		switch *vp.PUInt64() {
		case 0:
			ecn.ClampTgtRate = false
		case 1:
			ecn.ClampTgtRate = true
		default:
			return fmt.Errorf("failed to parse file %q: %w", attrPath, err)
		}
	case "dce_tcp_g":
		ecn.DceTCPG = *vp.PUInt64()
	case "dce_tcp_rtt":
		ecn.DceTCPRtt = *vp.PUInt64()
	case "initial_alpha_value":
		ecn.InitialAlphaValue = *vp.PUInt64()
	case "rate_reduce_monitor_period":
		ecn.RateReduceMonitorPeriod = *vp.PUInt64()
	case "rate_to_set_on_first_cnp":
		ecn.RateToSetOnFirstCnp = *vp.PUInt64()
	case "rpg_ai_rate":
		ecn.RpgAiRate = *vp.PUInt64()
	case "rpg_byte_reset":
		ecn.RpgByteReset = *vp.PUInt64()
	case "rpg_gd":
		ecn.RpgGd = *vp.PUInt64()
	case "rpg_hai_rate":
		ecn.RpgHaiRate = *vp.PUInt64()
	case "rpg_min_dec_fac":
		ecn.RpgMinDecFac = *vp.PUInt64()
	case "rpg_min_rate":
		ecn.RpgMinRate = *vp.PUInt64()
	case "rpg_threshold":
		ecn.RpgThreshold = *vp.PUInt64()
	case "rpg_time_reset":
		ecn.RpgTimeReset = *vp.PUInt64()
	default:
		return nil
	}

	return nil
}

// parses the ECN enable directory. It takes a path which should be a directory.
// This directory should have filenames that are uint8 and the content of the file is
// either 0 or 1.
func ParseEcnEnable(path string) (map[uint8]bool, error) {
	// Read the files in the directory
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	ecn := make(map[uint8]bool)
	// Iterate through each file in the directory
	for _, file := range files {
		// Only process files (skip directories)
		if file.IsDir() {
			continue
		}

		// Extract the file name (which should be the integer key)
		filename := file.Name()

		// Attempt to convert the file name to an integer
		filenameInt, err := strconv.ParseUint(filename, 10, 8)
		if err != nil {
			// Skip the file if the name cannot be converted to an integer
			continue
		}

		value, err := util.SysReadFile(filepath.Join(path, filename))
		if err != nil {
			if canIgnoreError(err) {
				return nil, err
			}
			return nil, fmt.Errorf("failed to read file %q: %w", filename, err)
		}

		vp := util.NewValueParser(value)
		fileValue := *vp.PUInt64()
		switch fileValue {
		case 0:
			ecn[uint8(filenameInt)] = false
		case 1:
			ecn[uint8(filenameInt)] = true
		default:
			return nil, fmt.Errorf("failed to parse file %q: %q", filename, value)
		}
	}

	return ecn, nil
}

// Utility function that given a path will return if the path is a dir or not.
func PathExistsAndIsDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil // Path does not exist
		}
		return false, err // Some other error occurred
	}
	return info.IsDir(), nil // Check if the path is a directory
}
