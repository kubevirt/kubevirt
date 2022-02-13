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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package console

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"

	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
)

const (
	PromptExpression = `(\$ |\# )`
	CRLF             = "\r\n"
	UTFPosEscape     = "\u001b\\[[0-9]+;[0-9]+H"
)

var (
	ShellSuccess       = RetValue("0")
	ShellFail          = RetValue("[1-9].*")
	ShellSuccessRegexp = regexp.MustCompile(ShellSuccess)
	ShellFailRegexp    = regexp.MustCompile(ShellFail)
)

// ExpectBatch runs the batch from `expected` connecting to the `vmi` console and
// wait `timeout` for the batch to return.
// NOTE: there is a safer version that validates sended commands `SafeExpectBatch` refer to it about limitations.
func ExpectBatch(vmi *v1.VirtualMachineInstance, expected []expect.Batcher, timeout time.Duration) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}

	expecter, _, err := NewExpecter(virtClient, vmi, 30*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	resp, err := expecter.ExpectBatch(expected, timeout)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
	}
	return err
}

// SafeExpectBatch runs the batch from `expected`, connecting to a VMI's console and
// waiting `wait` seconds for the batch to return.
// It validates that the commands arrive to the console.
// NOTE: This functions heritage limitations from `ExpectBatchWithValidatedSend` refer to it to check them.
func SafeExpectBatch(vmi *v1.VirtualMachineInstance, expected []expect.Batcher, wait int) error {
	_, err := SafeExpectBatchWithResponse(vmi, expected, wait)
	return err
}

// SafeExpectBatchWithResponse runs the batch from `expected`, connecting to a VMI's console and
// waiting `wait` seconds for the batch to return with a response.
// It validates that the commands arrive to the console.
// NOTE: This functions inherits limitations from `ExpectBatchWithValidatedSend`, refer to it for more information.
func SafeExpectBatchWithResponse(vmi *v1.VirtualMachineInstance, expected []expect.Batcher, wait int) ([]expect.BatchRes, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}
	expecter, _, err := NewExpecter(virtClient, vmi, 30*time.Second)
	if err != nil {
		return nil, err
	}
	defer expecter.Close()

	resp, err := ExpectBatchWithValidatedSend(expecter, expected, time.Second*time.Duration(wait))
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("%v", resp)
	}
	return resp, err
}

// RunCommand runs the command line from `command` connecting to an already logged in console at vmi
// and waiting `timeout` for command to return.
// NOTE: The safer version `ExpectBatchWithValidatedSend` is not used here since it does not support cases.
func RunCommand(vmi *v1.VirtualMachineInstance, command string, timeout time.Duration) error {
	err := ExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: PromptExpression},
		&expect.BSnd{S: command + "\n"},
		&expect.BExp{R: PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				R: ShellSuccessRegexp,
				T: expect.OK(),
			},
			&expect.Case{
				R: ShellFailRegexp,
				T: expect.Fail(expect.NewStatus(codes.Unavailable, command+" failed")),
			},
		}},
	}, timeout)
	if err != nil {
		return fmt.Errorf("Failed to run [%s] at VMI %s, error: %v", command, vmi.Name, err)
	}
	return nil
}

// SecureBootExpecter should be called on a VMI that has EFI enabled
// It will parse the kernel output (dmesg) and succeed if it finds that Secure boot is enabled
func SecureBootExpecter(vmi *v1.VirtualMachineInstance) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}
	expecter, _, err := NewExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	b := append([]expect.Batcher{
		&expect.BExp{R: "secureboot: Secure boot enabled"},
	})
	res, err := expecter.ExpectBatch(b, 180*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Kernel: %+v", res)
		return err
	}

	return err
}

// NetBootExpecter should be called on a VMI that has BIOS serial logging enabled
// It will parse the SeaBIOS output and succeed if it finds the string "iPXE"
func NetBootExpecter(vmi *v1.VirtualMachineInstance) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}
	expecter, _, err := NewExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	esc := UTFPosEscape
	b := append([]expect.Batcher{
		// SeaBIOS uses escape (\u001b) combinations for letter placement on screen
		// The regex below effectively grep for "iPXE" while ignoring those
		//&expect.BExp{R: "\u001b\\[7;27Hi\u001b\\[7;28HP\u001b\\[7;29HX\u001b\\[7;30HE"},
		&expect.BExp{R: esc + "i" + esc + "P" + esc + "X" + esc + "E"},
	})
	res, err := expecter.ExpectBatch(b, 30*time.Second)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("BIOS: %+v", res)
		return err
	}

	return err
}

// NewExpecter will connect to an already logged in VMI console and return the generated expecter it will wait `timeout` for the connection.
func NewExpecter(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, timeout time.Duration, opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	vmiReader, vmiWriter := io.Pipe()
	expecterReader, expecterWriter := io.Pipe()
	resCh := make(chan error)

	startTime := time.Now()
	con, err := virtCli.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kubecli.SerialConsoleOptions{ConnectionTimeout: timeout})
	if err != nil {
		return nil, nil, err
	}
	timeout = timeout - time.Now().Sub(startTime)

	go func() {
		resCh <- con.Stream(kubecli.StreamOptions{
			In:  vmiReader,
			Out: expecterWriter,
		})
	}()

	opts = append(opts, expect.SendTimeout(timeout))
	opts = append(opts, expect.Verbose(true))
	opts = append(opts, expect.VerboseWriter(GinkgoWriter))
	return expect.SpawnGeneric(&expect.GenOptions{
		In:  vmiWriter,
		Out: expecterReader,
		Wait: func() error {
			return <-resCh
		},
		Close: func() error {
			expecterWriter.Close()
			vmiReader.Close()
			return nil
		},
		Check: func() bool { return true },
	}, timeout, opts...)
}

// ExpectBatchWithValidatedSend adds the expect.BSnd command to the exect.BExp expression.
// It is done to make sure the match was found in the result of the expect.BSnd
// command and not in a leftover that wasn't removed from the buffer.
// NOTE: the method contains the following limitations:
//       - Use of `BatchSwitchCase`
//       - Multiline commands
//       - No more than one sequential send or receive
func ExpectBatchWithValidatedSend(expecter expect.Expecter, batch []expect.Batcher, timeout time.Duration) ([]expect.BatchRes, error) {
	sendFlag := false
	expectFlag := false
	previousSend := ""

	if len(batch) < 2 {
		return nil, fmt.Errorf("ExpectBatchWithValidatedSend requires at least 2 batchers, supplied %v", batch)
	}

	for i, batcher := range batch {
		switch batcher.Cmd() {
		case expect.BatchExpect:
			if expectFlag == true {
				return nil, fmt.Errorf("Two sequential expect.BExp are not allowed")
			}
			expectFlag = true
			sendFlag = false
			if _, ok := batch[i].(*expect.BExp); !ok {
				return nil, fmt.Errorf("ExpectBatchWithValidatedSend support only expect of type BExp")
			}
			bExp, _ := batch[i].(*expect.BExp)
			previousSend := regexp.QuoteMeta(previousSend)

			// Remove the \n since it is translated by the console to \r\n.
			previousSend = strings.TrimSuffix(previousSend, "\n")
			bExp.R = fmt.Sprintf("%s%s%s", previousSend, "((?s).*)", bExp.R)
		case expect.BatchSend:
			if sendFlag == true {
				return nil, fmt.Errorf("Two sequential expect.BSend are not allowed")
			}
			sendFlag = true
			expectFlag = false
			previousSend = batcher.Arg()
		case expect.BatchSwitchCase:
			return nil, fmt.Errorf("ExpectBatchWithValidatedSend doesn't support BatchSwitchCase")
		default:
			return nil, fmt.Errorf("Unknown command: ExpectBatchWithValidatedSend supports only BatchExpect and BatchSend")
		}
	}

	res, err := expecter.ExpectBatch(batch, timeout)
	return res, err
}

func RetValue(retcode string) string {
	return "\n" + retcode + CRLF + ".*" + PromptExpression
}
