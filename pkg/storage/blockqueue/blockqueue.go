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

package blockqueue

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// ReadAttr reads /sys/dev/block/<maj>:<min>/queue/<attr> for rdev.
//
// The kernel registers queue/ only on the gendisk (blk_register_queue); partitions
// have a "partition" file but no queue/, and inherit those attrs from the parent.
// See Documentation/block/queue-sysfs.rst and lsblk(8).
func ReadAttr(rdev uint64, attr string) ([]byte, error) {
	base := fmt.Sprintf("/sys/dev/block/%d:%d", unix.Major(rdev), unix.Minor(rdev))
	raw, err := os.ReadFile(base + "/queue/" + attr)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		return raw, err
	}
	if _, e := os.Stat(base + "/partition"); e != nil {
		return nil, err
	}
	return os.ReadFile(base + "/../queue/" + attr)
}
