// Copyright 2015 CNI authors
// Copyright 2016 Red Hat
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/spf13/pflag"

	"kubevirt.io/kubevirt/pkg/networking"
)

const (
	EnvCapabilityArgs = "CAP_ARGS"

	CmdAdd = "add"
	CmdDel = "del"
)

func parseArgs(args string) ([][2]string, error) {
	var result [][2]string

	pairs := strings.Split(args, ";")
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
			return nil, fmt.Errorf("invalid CNI_ARGS pair %q", pair)
		}

		result = append(result, [2]string{kv[0], kv[1]})
	}

	return result, nil
}

func main() {

	argsFromCmdline := ""
	cniConfigPath := ""
	cniPath := ""
	var from uint
	var to uint

	pflag.StringVar(&argsFromCmdline, "args", "", "CNI arguments as semicolon separated list of key value")
	pflag.StringVarP(&cniConfigPath, "cni-config-path", "c", "/etc/cni/net.d", "Path to CNI configs")
	pflag.StringVarP(&cniPath, "cni-path", "p", "/tools/plugins", "Path to CNI binaries")
	pflag.UintVarP(&from, "from-ns", "f", 1, "From which PID to take the network namespace to execute the CNI plugin")
	pflag.UintVarP(&to, "to-ns", "t", 1, "From which PID to take the network namespace for the new device")

	pflag.Usage = usage
	pflag.Parse()

	if pflag.NArg() != 4 {
		pflag.Usage()
		os.Exit(1)
	}

	cmd := pflag.Args()[0]
	id := pflag.Args()[1]
	configName := pflag.Args()[2]
	ifName := pflag.Args()[3]

	netconf, err := libcni.LoadConfList(cniConfigPath, configName)
	handleErr(err)

	var capabilityArgs map[string]interface{}
	capabilityArgsValue := os.Getenv(EnvCapabilityArgs)
	if len(capabilityArgsValue) > 0 {
		err = json.Unmarshal([]byte(capabilityArgsValue), &capabilityArgs)
		handleErr(err)
	}

	var cniArgs [][2]string
	if len(argsFromCmdline) > 0 {
		cniArgs, err = parseArgs(argsFromCmdline)
		handleErr(err)
	}

	cninet := &libcni.CNIConfig{
		Path: filepath.SplitList(cniPath),
	}

	rt := &libcni.RuntimeConf{
		ContainerID:    id,
		NetNS:          networking.GetNSFromPID(to),
		IfName:         ifName,
		Args:           cniArgs,
		CapabilityArgs: capabilityArgs,
	}

	err = ns.WithNetNSPath(networking.GetNSFromPID(from), func(_ ns.NetNS) error {
		switch cmd {
		case CmdAdd:
			result, err := cninet.AddNetworkList(netconf, rt)
			if result != nil {
				_ = result.Print()
			}
			return err
		case CmdDel:
			return cninet.DelNetworkList(netconf, rt)
		}
		return fmt.Errorf("Unknown command: %s", cmd)
	})

	handleErr(err)
}

func usage() {
	fmt.Print("Usage: cnitool add|del ID config ifname [options]\n\n")
	pflag.PrintDefaults()
	fmt.Println("\nCapabilities can be passed in via the CAP_ARGS environment variable.")
}


func handleErr(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}