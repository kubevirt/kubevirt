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

package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"kubevirt.io/client-go/log"
)

var runCommand = func(cmd *exec.Cmd) error {
	return cmd.Run()
}

func RunLocalClient(kind, namespace, name string, options *SSHOptions, clientArgs []string) error {
	args := []string{"-o"}
	args = append(args, buildProxyCommandOption(kind, namespace, name, options.SSHPort))

	if len(options.AdditionalSSHLocalOptions) > 0 {
		args = append(args, options.AdditionalSSHLocalOptions...)
	}
	if options.IdentityFilePathProvided {
		args = append(args, "-i", options.IdentityFilePath)
	}

	args = append(args, clientArgs...)

	cmd := exec.Command(options.LocalClientName, args...)
	log.Log.V(3).Infof("running: %v", cmd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return runCommand(cmd)
}

func buildProxyCommandOption(kind, namespace, name string, port int) string {
	proxyCommand := strings.Builder{}
	proxyCommand.WriteString("ProxyCommand=")
	proxyCommand.WriteString(os.Args[0])
	proxyCommand.WriteString(" port-forward --stdio=true ")
	proxyCommand.WriteString(fmt.Sprintf("%s/%s/%s", kind, name, namespace))
	proxyCommand.WriteString(" ")

	proxyCommand.WriteString(strconv.Itoa(port))

	return proxyCommand.String()
}

func (o *SSH) buildSSHTarget(kind, namespace, name string) (opts []string) {
	target := strings.Builder{}
	if len(o.options.SSHUsername) > 0 {
		target.WriteString(o.options.SSHUsername)
		target.WriteRune('@')
	}
	target.WriteString(kind)
	target.WriteString(".")
	target.WriteString(name)
	target.WriteString(".")
	target.WriteString(namespace)

	opts = append(opts, target.String())
	if o.command != "" {
		opts = append(opts, o.command)
	}
	return
}
