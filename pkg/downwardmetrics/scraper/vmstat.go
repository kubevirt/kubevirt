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

package downwardmetrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type vmStat struct {
	pswpin  uint64
	pswpout uint64
}

// readVMStat reads specific fields from the /proc/vmstat file.
// We implement it here, because it is not implemented in "github.com/prometheus/procfs"
// library.
func readVMStat(path string) (*vmStat, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := &vmStat{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 {
			return nil, fmt.Errorf("malformed line: %q", s.Text())
		}

		var resultField *uint64
		switch fields[0] {
		case "pswpin":
			resultField = &(result.pswpin)
		case "pswpout":
			resultField = &(result.pswpout)
		default:
			continue
		}

		value, err := strconv.ParseUint(fields[1], 0, 64)
		if err != nil {
			return nil, err
		}

		*resultField = value
	}

	return result, nil
}
