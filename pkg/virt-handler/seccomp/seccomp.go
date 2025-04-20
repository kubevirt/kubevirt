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
 */

package seccomp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containers/common/pkg/seccomp"
)

// Install seccomp, kubeletRoot should be passed in format: /proc/1/root/var/lib/kubelet/
func InstallPolicy(kubeletRoot string) error {
	const errMsgFormat string = "failed to install default seccomp profile: %v"

	installPath := filepath.Join(kubeletRoot, "seccomp/kubevirt")
	if err := os.MkdirAll(installPath, 0700); err != nil {
		return fmt.Errorf(errMsgFormat, err)
	}

	profileBytes, err := json.Marshal(defaultProfile())
	if err != nil {
		return fmt.Errorf(errMsgFormat, fmt.Errorf("internal failure: %v", err))
	}

	profilePath := filepath.Join(installPath, "kubevirt.json")
	currentProfileBytes, err := os.ReadFile(profilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf(errMsgFormat, err)
	}
	if bytes.Equal(currentProfileBytes, profileBytes) {
		return nil
	}

	if err := os.WriteFile(profilePath, profileBytes, 0700); err != nil {
		return fmt.Errorf(errMsgFormat, err)
	}

	return nil
}

func defaultProfile() *seccomp.Seccomp {
	profile := seccomp.DefaultProfile()

	for _, syscalls := range profile.Syscalls {
		found := -1
		for i, syscall := range syscalls.Names {
			// Required for post-copy
			if syscall == "userfaultfd" {
				found = i
				break
			}
		}
		if found == -1 {
			continue
		}

		if syscalls.Action == seccomp.ActErrno {
			names := syscalls.Names[:found]
			found += 1
			if found < len(syscalls.Names) {
				names = append(names, syscalls.Names[found:]...)
			}
			syscalls.Names = names
			break
		}

	}

	profile.Syscalls = append(profile.Syscalls, &seccomp.Syscall{
		Names:  []string{"userfaultfd"},
		Action: seccomp.ActAllow,
		Args:   []*seccomp.Arg{},
	})
	return profile
}
