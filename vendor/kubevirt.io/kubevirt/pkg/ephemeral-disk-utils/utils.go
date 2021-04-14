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
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	v1 "kubevirt.io/client-go/api/v1"
)

// TODO this should be part of structs, instead of a global
var DefaultOwnershipManager OwnershipManagerInterface = &OwnershipManager{user: "qemu"}

// For testing
func MockDefaultOwnershipManager() {
	owner, err := user.Current()
	if err != nil {
		panic(err)
	}

	DefaultOwnershipManager = &OwnershipManager{user: owner.Username}
}

type OwnershipManager struct {
	user string
}

func (om *OwnershipManager) SetFileOwnership(file string) error {
	owner, err := user.Lookup(om.user)
	if err != nil {
		return fmt.Errorf("failed to look up user %s: %v", om.user, err)
	}

	uid, err := strconv.Atoi(owner.Uid)
	if err != nil {
		return fmt.Errorf("failed to convert UID %s of user %s: %v", owner.Uid, om.user, err)
	}

	gid, err := strconv.Atoi(owner.Gid)
	if err != nil {
		return fmt.Errorf("failed to convert GID %s of user %s: %v", owner.Gid, om.user, err)
	}
	return os.Chown(file, uid, gid)
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

// Lists all vmis ephemeral disk has local data for
func ListVmWithEphemeralDisk(localPath string) ([]*v1.VirtualMachineInstance, error) {
	var keys []*v1.VirtualMachineInstance

	exists, err := FileExists(localPath)
	if err != nil {
		return nil, err
	}
	if exists == false {
		return nil, nil
	}

	err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() == false {
			return nil
		}

		relativePath := strings.TrimPrefix(path, localPath+"/")
		if relativePath == "" {
			return nil
		}
		dirs := strings.Split(relativePath, "/")
		if len(dirs) != 2 {
			return nil
		}

		namespace := dirs[0]
		domain := dirs[1]
		if namespace == "" || domain == "" {
			return nil
		}
		keys = append(keys, v1.NewVMIReferenceFromNameWithNS(dirs[0], dirs[1]))
		return nil
	})

	return keys, err
}

type OwnershipManagerInterface interface {
	SetFileOwnership(file string) error
}
