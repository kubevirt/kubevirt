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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package fips

// int bridge_FIPS_mode(void *f) {
//     int (*FIPS_mode)(void) = (int (*)(void))f;
//     return FIPS_mode();
// }
import "C"

import (
	"errors"
	"os"
	"regexp"

	"github.com/coreos/pkg/dlopen"

	"kubevirt.io/kubevirt/pkg/util/sysctl"
)

var libPaths = []string{"/lib64", "/usr/lib64", "/lib", "/usr/lib"}

func isSysctlFipsEnabled() (bool, error) {
	sys := sysctl.New()
	fipsEnabled, err := sys.GetSysctl(sysctl.CryptoFipsEnabled)
	if err != nil {
		return false, err
	}
	return fipsEnabled == 1, nil
}

func findCryptoLibsInDir(dir string) []string {
	dirInfo, err := os.Stat(dir)
	if err != nil || dirInfo.Mode()&os.ModeSymlink != 0 {
		return []string{}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	libs := []string{}
	validLibName := regexp.MustCompile(`^libcrypto.*\.so($|\..*)`)

	for i := range entries {
		if !entries[i].IsDir() && entries[i].Type().IsRegular() {
			if validLibName.MatchString(entries[i].Name()) {
				libs = append(libs, entries[i].Name())
			}
		}
	}
	return libs
}

func findCryptoLibs() ([]string, error) {
	libCryptoPaths := []string{}

	for i := range libPaths {
		if cryptoLibs := findCryptoLibsInDir(libPaths[i]); len(cryptoLibs) > 0 {
			libCryptoPaths = append(libCryptoPaths, cryptoLibs...)
		}
	}

	if len(libCryptoPaths) == 0 {
		return []string{}, errors.New("The OpenSSL library is not installed")
	}
	return libCryptoPaths, nil
}

func IsFipsEnabled() (bool, error) {
	if ok, err := isSysctlFipsEnabled(); !ok || err != nil {
		return false, err
	}

	cryptoLibs, err := findCryptoLibs()
	if err != nil {
		return false, err
	}

	var retErr error
	for i := range cryptoLibs {
		lib, err := dlopen.GetHandle([]string{cryptoLibs[i]})
		if err != nil {
			retErr = err
			continue
		}
		defer lib.Close()

		fipsModeFuncPtr, err := lib.GetSymbolPointer("FIPS_mode")
		if err != nil {
			retErr = errors.New("The installed OpenSSL library is not FIPS-capable")
			continue
		}
		fipsMode := C.bridge_FIPS_mode(fipsModeFuncPtr)
		return fipsMode != 0, nil
	}
	return false, retErr
}
