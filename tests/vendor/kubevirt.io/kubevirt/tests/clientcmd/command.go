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
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/onsi/ginkgo/v2"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tests/flags"
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

func SkipIfNoCmd(cmdName string) {
	var cmdPath string
	switch strings.ToLower(cmdName) {
	case "oc":
		cmdPath = flags.KubeVirtOcPath
	case "kubectl":
		cmdPath = flags.KubeVirtKubectlPath
	case "virtctl":
		cmdPath = flags.KubeVirtVirtctlPath
	case "gocli":
		cmdPath = flags.KubeVirtGoCliPath
	}
	if cmdPath == "" {
		ginkgo.Skip(fmt.Sprintf("Skip test that requires %s binary", cmdName))
	}
}

func RunCommand(cmdName string, args ...string) (string, string, error) {
	return RunCommandWithNS(util.NamespaceTestDefault, cmdName, args...)
}

func RunCommandWithNS(namespace string, cmdName string, args ...string) (string, string, error) {
	return RunCommandWithNSAndInput(namespace, nil, cmdName, args...)
}

func RunCommandWithNSAndInput(namespace string, input io.Reader, cmdName string, args ...string) (string, string, error) {
	commandString, cmd, err := CreateCommandWithNS(namespace, cmdName, args...)
	if err != nil {
		return "", "", err
	}

	var output, stderr bytes.Buffer
	captureOutputBuffers := func() (string, string) {
		trimNullChars := func(buf bytes.Buffer) string {
			return string(bytes.Trim(buf.Bytes(), "\x00"))
		}
		return trimNullChars(output), trimNullChars(stderr)
	}

	cmd.Stdin, cmd.Stdout, cmd.Stderr = input, &output, &stderr

	if err := cmd.Run(); err != nil {
		outputString, stderrString := captureOutputBuffers()
		log.Log.Reason(err).With("command", commandString, "output", outputString, "stderr", stderrString).Error("command failed: cannot run command")
		return outputString, stderrString, fmt.Errorf("command failed: cannot run command %q: %v", commandString, err)
	}

	outputString, stderrString := captureOutputBuffers()
	return outputString, stderrString, nil
}

func CreateCommandWithNS(namespace string, cmdName string, args ...string) (string, *exec.Cmd, error) {
	cmdPath := ""
	commandString := func() string {
		c := cmdPath
		if cmdPath == "" {
			c = cmdName
		}
		return strings.Join(append([]string{c}, args...), " ")
	}

	cmdName = strings.ToLower(cmdName)
	switch cmdName {
	case "oc":
		cmdPath = flags.KubeVirtOcPath
	case "kubectl":
		cmdPath = flags.KubeVirtKubectlPath
	case "virtctl":
		cmdPath = flags.KubeVirtVirtctlPath
	case "gocli":
		cmdPath = flags.KubeVirtGoCliPath
	}

	if cmdPath == "" {
		err := fmt.Errorf("no %s binary specified", cmdName)
		log.Log.Reason(err).With("command", commandString()).Error("command failed")
		return "", nil, fmt.Errorf("command failed: %v", err)
	}

	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		err := errors.New("cannot find kubeconfig")
		log.Log.Reason(err).With("command", commandString()).Error("command failed")
		return "", nil, fmt.Errorf("command failed: %v", err)
	}

	master := flag.Lookup("master").Value
	if master != nil && master.String() != "" {
		args = append(args, serverName, master.String())
	}
	if namespace != "" {
		args = append([]string{"-n", namespace}, args...)
	}

	cmd := exec.Command(cmdPath, args...)
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())
	cmd.Env = append(os.Environ(), kubeconfEnv)

	return commandString(), cmd, nil
}

func RunCommandPipe(commands ...[]string) (string, string, error) {
	return RunCommandPipeWithNS(util.NamespaceTestDefault, commands...)
}

func RunCommandPipeWithNS(namespace string, commands ...[]string) (string, string, error) {
	commandPipeString := func() string {
		commandStrings := []string{}
		for _, command := range commands {
			commandStrings = append(commandStrings, strings.Join(command, " "))
		}
		return strings.Join(commandStrings, " | ")
	}

	if len(commands) < 2 {
		err := errors.New("requires at least two commands")
		log.Log.Reason(err).With("command", commandPipeString()).Error(commandPipeFailed)
		return "", "", fmt.Errorf(commandPipeFailedFmt, err)
	}

	for i, command := range commands {
		cmdPath := ""
		cmdName := strings.ToLower(command[0])
		switch cmdName {
		case "oc":
			cmdPath = flags.KubeVirtOcPath
		case "kubectl":
			cmdPath = flags.KubeVirtKubectlPath
		case "virtctl":
			cmdPath = flags.KubeVirtVirtctlPath
		}
		if cmdPath == "" {
			err := fmt.Errorf("no %s binary specified", cmdName)
			log.Log.Reason(err).With("command", commandPipeString()).Error(commandPipeFailed)
			return "", "", fmt.Errorf(commandPipeFailedFmt, err)
		}
		commands[i][0] = cmdPath
	}

	kubeconfig := flag.Lookup("kubeconfig").Value
	if kubeconfig == nil || kubeconfig.String() == "" {
		err := errors.New("cannot find kubeconfig")
		log.Log.Reason(err).With("command", commandPipeString()).Error(commandPipeFailed)
		return "", "", fmt.Errorf(commandPipeFailedFmt, err)
	}
	kubeconfEnv := fmt.Sprintf("KUBECONFIG=%s", kubeconfig.String())

	master := flag.Lookup("master").Value
	cmds := make([]*exec.Cmd, len(commands))
	for i := range cmds {
		if master != nil && master.String() != "" {
			commands[i] = append(commands[i], serverName, master.String())
		}
		if namespace != "" {
			commands[i] = append(commands[i], "-n", namespace)
		}
		cmds[i] = exec.Command(commands[i][0], commands[i][1:]...)
		cmds[i].Env = append(os.Environ(), kubeconfEnv)
	}

	var output, stderr bytes.Buffer
	captureOutputBuffers := func() (string, string) {
		trimNullChars := func(buf bytes.Buffer) string {
			return string(bytes.Trim(buf.Bytes(), "\x00"))
		}
		return trimNullChars(output), trimNullChars(stderr)
	}

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			cmdArgString := strings.Join(cmd.Args, " ")
			log.Log.Reason(err).With("command", commandPipeString()).Errorf("command pipe failed: cannot attach stdout pipe to command %q", cmdArgString)
			return "", "", fmt.Errorf("command pipe failed: cannot attach stdout pipe to command %q: %v", cmdArgString, err)
		}
		cmd.Stderr = &stderr
	}
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			outputString, stderrString := captureOutputBuffers()
			cmdArgString := strings.Join(cmd.Args, " ")
			log.Log.Reason(err).With("command", commandPipeString(), "output", outputString, "stderr", stderrString).Errorf("command pipe failed: cannot start command %q", cmdArgString)
			return outputString, stderrString, fmt.Errorf("command pipe failed: cannot start command %q: %v", cmdArgString, err)
		}
	}

	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			outputString, stderrString := captureOutputBuffers()
			cmdArgString := strings.Join(cmd.Args, " ")
			log.Log.Reason(err).With("command", commandPipeString(), "output", outputString, "stderr", stderrString).Errorf("command pipe failed: error while waiting for command %q", cmdArgString)
			return outputString, stderrString, fmt.Errorf("command pipe failed: error while waiting for command %q: %v", cmdArgString, err)
		}
	}

	outputString, stderrString := captureOutputBuffers()
	return outputString, stderrString, nil
}
