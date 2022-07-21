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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package clientcmd

import (
	"os/exec"

	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libcli"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	commandPipeFailed    = "command pipe failed"
	commandPipeFailedFmt = "command pipe failed: %v"

	serverName = "--server"
)

func GetK8sCmdClient() string {
	// use oc if it exists, otherwise use kubectl
	if flags.KubeVirtOcPath != "" {
		return "oc"
	}

	return "kubectl"
}

func RunCommand(cmdName string, args ...string) (string, string, error) {
	return RunCommandWithNS(util.NamespaceTestDefault, cmdName, args...)
}

func RunCommandWithNS(namespace string, cmdName string, args ...string) (string, string, error) {
	return libcli.RunCommandWithNSAndInput(namespace, nil, getCmdPathFromName(cmdName), args...)
}

func RunCommandPipe(commands ...[]string) (string, string, error) {
	return libcli.RunCommandPipeWithNS(cmdPathFromName(), util.NamespaceTestDefault, commands...)
}

func CreateCommandWithNS(namespace string, cmdName string, args ...string) (string, *exec.Cmd, error) {
	return libcli.CreateCommandWithNS(namespace, getCmdPathFromName(cmdName), args...)
}

func cmdPathFromName() map[string]string {
	return map[string]string{
		"oc":      flags.KubeVirtOcPath,
		"kubectl": flags.KubeVirtKubectlPath,
		"virtctl": flags.KubeVirtVirtctlPath,
		"gocli":   flags.KubeVirtGoCliPath,
	}
}

func getCmdPathFromName(cmdName string) string {
	cmdPath, ok := cmdPathFromName()[cmdName]
	if !ok {
		return ""
	}
	return cmdPath
}
