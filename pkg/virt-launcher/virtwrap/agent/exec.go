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

package agent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type execReturn struct {
	Return execReturnData `json:"return"`
}
type execReturnData struct {
	Pid int `json:"pid"`
}

type execStatusReturn struct {
	Return execStatusReturnData `json:"return"`
}
type execStatusReturnData struct {
	Exited   bool   `json:"exited"`
	ExitCode int    `json:"exitcode"`
	OutData  string `json:"out-data"`
}

// ExecExitCode returned at non-zero return codes
type ExecExitCode struct {
	ExitCode int
}

func (e ExecExitCode) Error() string {
	return fmt.Sprint("exited with error code:", e.ExitCode)
}

// GuestExec sends the provided command and args to the guest agent for execution and returns an error on an unsucessful exit code
// The resulting stdout will be returned as a string
func GuestExec(virConn cli.Connection, domName string, command string, args []string, timeoutSeconds int32) (string, error) {
	stdOut := ""
	argsStr := ""
	for _, arg := range args {
		if argsStr == "" {
			argsStr = fmt.Sprintf("\"%s\"", arg)
		} else {
			argsStr = argsStr + fmt.Sprintf(", \"%s\"", arg)
		}
	}

	cmdExec := fmt.Sprintf(`{"execute": "guest-exec", "arguments": { "path": "%s", "arg": [ %s ], "capture-output":true } }`, command, argsStr)
	output, err := virConn.QemuAgentCommand(cmdExec, domName)
	if err != nil {
		return "", err
	}
	execRes := &execReturn{}
	err = json.Unmarshal([]byte(output), execRes)
	if err != nil {
		return "", err
	}

	if execRes.Return.Pid <= 0 {
		return "", fmt.Errorf("Invalid pid [%d] returned from qemu agent during access credential injection: %s", execRes.Return.Pid, output)
	}

	exited := false
	exitCode := 0
	statusCheck := time.NewTicker(time.Duration(timeoutSeconds) * 100 * time.Millisecond)
	defer statusCheck.Stop()
	checkUntil := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for {
		cmdExecStatus := fmt.Sprintf(`{"execute": "guest-exec-status", "arguments": { "pid": %d } }`, execRes.Return.Pid)
		output, err := virConn.QemuAgentCommand(cmdExecStatus, domName)
		if err != nil {
			return "", err
		}
		execStatusRes := &execStatusReturn{}
		err = json.Unmarshal([]byte(output), execStatusRes)
		if err != nil {
			return "", err
		}

		if execStatusRes.Return.Exited {
			stdOutBytes, err := base64.StdEncoding.DecodeString(execStatusRes.Return.OutData)
			if err != nil {
				return "", err
			}
			stdOut = string(stdOutBytes)
			exitCode = execStatusRes.Return.ExitCode
			exited = true
			break
		}

		if checkUntil.Before(<-statusCheck.C) {
			break
		}
	}

	if !exited {
		return "", fmt.Errorf("Timed out waiting for guest pid [%d] for command [%s] to exit", execRes.Return.Pid, command)
	} else if exitCode != 0 {
		return stdOut, ExecExitCode{exitCode}
	}

	return stdOut, nil
}
