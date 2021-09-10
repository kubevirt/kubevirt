/*
 * This file is part of the kubevirt project
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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package ephemeraldiskutils

import (
	"fmt"
	"os"
	"syscall"
)

// TODO this should be part of structs, instead of a global
var DefaultOwnershipManager OwnershipManagerInterface = &OwnershipManager{uid: 107, gid: 107}

// For testing
func MockDefaultOwnershipManager() {
	DefaultOwnershipManager = &nonOpManager{}
}

type nonOpManager struct {
}

func (no *nonOpManager) SetFileOwnership(file string) error {
	return nil
}

type OwnershipManager struct {
	uid, gid int
}

func (om *OwnershipManager) SetFileOwnership(file string) error {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return err
	}

	if stat, ok := fileInfo.Sys().(*syscall.Stat_t); ok {
		if om.uid == int(stat.Uid) && om.gid == int(stat.Gid) {
			return nil
		}
	} else {
		return fmt.Errorf("failed to convert stat info")
	}

	return os.Chown(file, om.uid, om.gid)
}

func RemoveFilesIfExist(paths ...string) error {
	var err error
	for _, path := range paths {
		err = os.Remove(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	exists := false

	if err == nil {
		exists = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return exists, err
}

type OwnershipManagerInterface interface {
	SetFileOwnership(file string) error
}
