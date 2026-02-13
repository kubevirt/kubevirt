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

package util

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var (
	// panicLogLineRegex matches QEMU log lines containing panic info
	// Format example: "2024-09-30 16:30:49.506+0000: panic hyper-v: arg1='0x5a', arg2='0x1', ..."
	panicLogLineRegex = regexp.MustCompile(`panic\s+(\S+):\s*(.*)`)
	// argRegex parses arg1='0x...' style arguments from panic log lines
	argRegex = regexp.MustCompile(`arg(\d+)=['"]?(0x[0-9a-fA-F]+)['"]?`)
)

// qemuGuestPanicEvent represents the GUEST_PANICKED QEMU monitor event details
type qemuGuestPanicEvent struct {
	Action string              `json:"action"`
	Info   *api.GuestPanicInfo `json:"info,omitempty"`
}

// ParseGuestPanicEvent parses QEMU GUEST_PANICKED event details JSON
func ParseGuestPanicEvent(jsonDetails string) *api.GuestPanicInfo {
	if jsonDetails == "" {
		return nil
	}

	var event qemuGuestPanicEvent
	if err := json.Unmarshal([]byte(jsonDetails), &event); err != nil {
		log.Log.Reason(err).Warningf("Failed to parse GUEST_PANICKED event details: %s", jsonDetails)
		return nil
	}

	if event.Info == nil {
		return &api.GuestPanicInfo{}
	}

	return event.Info
}

// FormatGuestPanicInfo formats the panic info for display in VMI status
func FormatGuestPanicInfo(info *api.GuestPanicInfo) string {
	if info == nil {
		return "GuestPanicked"
	}

	if info.Type == "hyper-v" {
		// Format Windows BSOD parameters (bugcheck code and parameters)
		return fmt.Sprintf("GuestPanicked: '%#x', '%#x', '%#x', '%#x', '%#x'",
			info.Arg1, info.Arg2, info.Arg3, info.Arg4, info.Arg5)
	}

	if info.Type != "" {
		return fmt.Sprintf("GuestPanicked: type=%s", info.Type)
	}

	return "GuestPanicked"
}

// parseHexArg parses a hex argument value from format arg='0x...' or arg=0x...
func parseHexArg(s string) uint64 {
	// Remove quotes and 0x prefix
	s = strings.Trim(s, "'\"")
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	val, _ := strconv.ParseUint(s, 16, 64)
	return val
}

// ParseGuestPanicLogLine parses a log line containing panic info
// Returns nil if line doesn't contain panic info
func ParseGuestPanicLogLine(line string) *api.GuestPanicInfo {
	matches := panicLogLineRegex.FindStringSubmatch(line)
	if matches == nil {
		return nil
	}

	panicType := matches[1]
	argsStr := matches[2]

	info := &api.GuestPanicInfo{
		Type: panicType,
	}

	argMatches := argRegex.FindAllStringSubmatch(argsStr, -1)
	for _, m := range argMatches {
		argNum, _ := strconv.Atoi(m[1])
		val := parseHexArg(m[2])
		switch argNum {
		case 1:
			info.Arg1 = val
		case 2:
			info.Arg2 = val
		case 3:
			info.Arg3 = val
		case 4:
			info.Arg4 = val
		case 5:
			info.Arg5 = val
		}
	}

	return info
}

// ReadPanicInfoFromLog reads the QEMU log file and extracts panic info.
// It scans from the end of the file for efficiency since panics are typically logged last.
func ReadPanicInfoFromLog(logPath string) (*api.GuestPanicInfo, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	}

	const chunkSize = 4096
	offset := stat.Size()
	remainder := ""

	for offset > 0 {
		readSize := min(offset, chunkSize)
		offset -= readSize

		buf := make([]byte, readSize)
		if _, err := file.ReadAt(buf, offset); err != nil {
			return nil, fmt.Errorf("failed to read log file: %w", err)
		}

		lines := strings.Split(string(buf)+remainder, "\n")
		remainder = lines[0]

		for i := len(lines) - 1; i >= 1; i-- {
			if info := ParseGuestPanicLogLine(lines[i]); info != nil {
				return info, nil
			}
		}
	}

	if info := ParseGuestPanicLogLine(remainder); info != nil {
		return info, nil
	}

	return &api.GuestPanicInfo{}, nil
}
