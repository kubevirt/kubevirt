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

package storage

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"errors"
	"os"
	"syscall"

	"kubevirt.io/kubevirt/pkg/util"
)

type DirectIOChecker interface {
	CheckBlockDevice(path string) (bool, error)
	CheckFile(path string) (bool, error)
}

type directIOChecker struct{}

func NewDirectIOChecker() DirectIOChecker {
	return &directIOChecker{}
}

func (c *directIOChecker) CheckBlockDevice(path string) (bool, error) {
	return c.check(path, syscall.O_RDONLY)
}

func (c *directIOChecker) CheckFile(path string) (bool, error) {
	flags := syscall.O_RDONLY
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// try to create the file and perform the check
		flags |= syscall.O_CREAT
		defer os.Remove(path)
	}
	return c.check(path, flags)
}

// based on https://gitlab.com/qemu-project/qemu/-/blob/master/util/osdep.c#L344
func (c *directIOChecker) check(path string, flags int) (bool, error) {
	// #nosec No risk for path injection as we only open the file, not read from it.
	// The function leaks only whether the directory to `path` exists.
	f, err := os.OpenFile(path, flags|syscall.O_DIRECT, 0o600)
	if err != nil {
		// EINVAL is returned if the filesystem does not support the O_DIRECT flag
		if err, ok := err.(*os.PathError); ok && err.Err == syscall.EINVAL {
			// #nosec No risk for path injection as we only open the file, not read from it.
			// The function leaks only whether the directory to `path` exists.
			f, err := os.OpenFile(path, flags & ^syscall.O_DIRECT, 0o600)
			if err == nil {
				defer util.CloseIOAndCheckErr(f, nil)
				return false, nil
			}
		}
		return false, err
	}
	defer util.CloseIOAndCheckErr(f, nil)
	return true, nil
}
