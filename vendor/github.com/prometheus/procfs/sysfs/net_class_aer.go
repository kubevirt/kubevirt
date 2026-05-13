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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/util"
)

// CorrectableAerCounters contains values from /sys/class/net/<iface>/device/aer_dev_correctable
// for single interface (iface).
type CorrectableAerCounters struct {
	RxErr       uint64
	BadTLP      uint64
	BadDLLP     uint64
	Rollover    uint64
	Timeout     uint64
	NonFatalErr uint64
	CorrIntErr  uint64
	HeaderOF    uint64
}

// UncorrectableAerCounters contains values from /sys/class/net/<iface>/device/aer_dev_[non]fatal
// for single interface (iface).
type UncorrectableAerCounters struct {
	Undefined        uint64
	DLP              uint64
	SDES             uint64
	TLP              uint64
	FCP              uint64
	CmpltTO          uint64
	CmpltAbrt        uint64
	UnxCmplt         uint64
	RxOF             uint64
	MalfTLP          uint64
	ECRC             uint64
	UnsupReq         uint64
	ACSViol          uint64
	UncorrIntErr     uint64
	BlockedTLP       uint64
	AtomicOpBlocked  uint64
	TLPBlockedErr    uint64
	PoisonTLPBlocked uint64
}

// AerCounters contains AER counters from files in /sys/class/net/<iface>/device
// for single interface (iface).
type AerCounters struct {
	Name        string // Interface name
	Correctable CorrectableAerCounters
	Fatal       UncorrectableAerCounters
	NonFatal    UncorrectableAerCounters
}

// AllAerCounters is collection of AER counters for every interface (iface) in /sys/class/net.
// The map keys are interface (iface) names.
type AllAerCounters map[string]AerCounters

// AerCounters returns info for a single net interfaces (iface).
func (fs FS) AerCountersByIface(devicePath string) (*AerCounters, error) {
	_, err := fs.NetClassByIface(devicePath)
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	counters, err := parseAerCounters(filepath.Join(path, devicePath))
	if err != nil {
		return nil, err
	}
	counters.Name = devicePath

	return counters, nil
}

// AerCounters returns AER counters for all net interfaces (iface) read from /sys/class/net/<iface>/device.
func (fs FS) AerCounters() (AllAerCounters, error) {
	devices, err := fs.NetClassDevices()
	if err != nil {
		return nil, err
	}

	path := fs.sys.Path(netclassPath)
	allAerCounters := AllAerCounters{}
	for _, devicePath := range devices {
		counters, err := parseAerCounters(filepath.Join(path, devicePath))
		if err != nil {
			return nil, err
		}
		counters.Name = devicePath
		allAerCounters[devicePath] = *counters
	}

	return allAerCounters, nil
}

// parseAerCounters scans predefined files in /sys/class/net/<iface>/device
// directory and gets their contents.
func parseAerCounters(devicePath string) (*AerCounters, error) {
	counters := AerCounters{}
	err := parseCorrectableAerCounters(devicePath, &counters.Correctable)
	if err != nil {
		return nil, err
	}
	err = parseUncorrectableAerCounters(devicePath, "fatal", &counters.Fatal)
	if err != nil {
		return nil, err
	}
	err = parseUncorrectableAerCounters(devicePath, "nonfatal", &counters.NonFatal)
	if err != nil {
		return nil, err
	}
	return &counters, nil
}

// parseCorrectableAerCounters parses correctable error counters in
// /sys/class/net/<iface>/device/aer_dev_correctable.
func parseCorrectableAerCounters(devicePath string, counters *CorrectableAerCounters) error {
	path := filepath.Join(devicePath, "device", "aer_dev_correctable")
	value, err := util.SysReadFile(path)
	if err != nil {
		if canIgnoreError(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}

	for line := range strings.SplitSeq(string(value), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("unexpected number of fields: %v", fields)
		}
		counterName := fields[0]
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing value for %s: %w", counterName, err)
		}

		switch counterName {
		case "RxErr":
			counters.RxErr = value
		case "BadTLP":
			counters.BadTLP = value
		case "BadDLLP":
			counters.BadDLLP = value
		case "Rollover":
			counters.Rollover = value
		case "Timeout":
			counters.Timeout = value
		case "NonFatalErr":
			counters.NonFatalErr = value
		case "CorrIntErr":
			counters.CorrIntErr = value
		case "HeaderOF":
			counters.HeaderOF = value
		default:
			continue
		}
	}

	return nil
}

// parseUncorrectableAerCounters parses uncorrectable error counters in
// /sys/class/net/<iface>/device/aer_dev_[non]fatal.
func parseUncorrectableAerCounters(devicePath string, counterType string,
	counters *UncorrectableAerCounters) error {
	path := filepath.Join(devicePath, "device", "aer_dev_"+counterType)
	value, err := util.ReadFileNoStat(path)
	if err != nil {
		if canIgnoreError(err) {
			return nil
		}
		return fmt.Errorf("failed to read file %q: %w", path, err)
	}

	for line := range strings.SplitSeq(string(value), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return fmt.Errorf("unexpected number of fields: %v", fields)
		}
		counterName := fields[0]
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing value for %s: %w", counterName, err)
		}

		switch counterName {
		case "Undefined":
			counters.Undefined = value
		case "DLP":
			counters.DLP = value
		case "SDES":
			counters.SDES = value
		case "TLP":
			counters.TLP = value
		case "FCP":
			counters.FCP = value
		case "CmpltTO":
			counters.CmpltTO = value
		case "CmpltAbrt":
			counters.CmpltAbrt = value
		case "UnxCmplt":
			counters.UnxCmplt = value
		case "RxOF":
			counters.RxOF = value
		case "MalfTLP":
			counters.MalfTLP = value
		case "ECRC":
			counters.ECRC = value
		case "UnsupReq":
			counters.UnsupReq = value
		case "ACSViol":
			counters.ACSViol = value
		case "UncorrIntErr":
			counters.UncorrIntErr = value
		case "BlockedTLP":
			counters.BlockedTLP = value
		case "AtomicOpBlocked":
			counters.AtomicOpBlocked = value
		case "TLPBlockedErr":
			counters.TLPBlockedErr = value
		case "PoisonTLPBlocked":
			counters.PoisonTLPBlocked = value
		default:
			continue
		}
	}

	return nil
}
