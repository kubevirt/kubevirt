/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Used https://github.com/kubernetes/kubernetes/blob/master/pkg/version/verflag/verflag.go as a template

// Package verflag defines utility functions to handle command line flags
// related to version of CDI.
package verflag

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"kubevirt.io/containerized-data-importer/pkg/version"
)

type versionValue int

// Enum specifying which format to use for printing CDI version
const (
	versionFalse versionValue = 0
	versionTrue  versionValue = 1
	versionRaw   versionValue = 2
)

const strRawVersion string = "raw"

func (v *versionValue) IsBoolFlag() bool {
	return true
}

func (v *versionValue) Get() interface{} {
	return versionValue(*v)
}

func (v *versionValue) Set(s string) error {
	if s == strRawVersion {
		*v = versionRaw
		return nil
	}
	boolVal, err := strconv.ParseBool(s)
	if boolVal {
		*v = versionTrue
	} else {
		*v = versionFalse
	}
	return err
}

func (v *versionValue) String() string {
	if *v == versionRaw {
		return strRawVersion
	}
	return fmt.Sprintf("%v", bool(*v == versionTrue))
}

// The type of the flag as required by the pflag.Value interface
func (v *versionValue) Type() string {
	return "version"
}

// versionVar defines a "version" flag
func versionVar(p *versionValue, name string, value versionValue, usage string) {
	*p = value
	flag.Var(p, name, usage)
}

func verflag(name string, value versionValue, usage string) *versionValue {
	p := new(versionValue)
	versionVar(p, name, value, usage)
	return p
}

const versionFlagName = "version"

var (
	versionFlag = verflag(versionFlagName, versionFalse, "Print version information and quit")
)

// PrintAndExitIfRequested will check if the -version flag was passed
// and, if so, print the version and exit.
func PrintAndExitIfRequested() {
	if *versionFlag == versionRaw {
		fmt.Printf("%#v\n", version.Get())
		os.Exit(0)
	} else if *versionFlag == versionTrue {
		fmt.Printf("Containerized Data Importer %s\n", version.Get())
		os.Exit(0)
	}
}
