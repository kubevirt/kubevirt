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
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"github.com/onsi/ginkgo/v2"

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

	consoleConnectionTimeout = 30 * time.Second
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
	virtClient := kubevirt.Client()

	expecter, _, err := NewExpecter(virtClient, vmi, consoleConnectionTimeout)
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
	virtClient := kubevirt.Client()
	expecter, _, err := NewExpecter(virtClient, vmi, consoleConnectionTimeout)
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
		return fmt.Errorf("failed to run [%s] at VMI %s, error: %v", command, vmi.Name, err)
	}
	return nil
}

// RunCommandAndStoreOutput runs the command line from `command` connecting to an already logged in console in vmi.
// The output of `command` is returned as a string
func RunCommandAndStoreOutput(vmi *v1.VirtualMachineInstance, command string, timeout time.Duration) (string, error) {
	virtClient := kubevirt.Client()

	opts := &kubecli.SerialConsoleOptions{ConnectionTimeout: timeout}
	stream, err := virtClient.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, opts)
	if err != nil {
		return "", err
	}
	conn := stream.AsConn()
	defer conn.Close()

	_, err = fmt.Fprintf(conn, "%s\n", command)
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(conn)
	if !skipInput(scanner) {
		return "", fmt.Errorf("failed to run [%s] at VMI %s (skip input)", command, vmi.Name)
	}
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to run [%s] at VMI %s", command, vmi.Name)
	}
	return scanner.Text(), nil
}

func skipInput(scanner *bufio.Scanner) bool {
	return scanner.Scan()
}

// SecureBootExpecter should be called on a VMI that has EFI enabled
// It will parse the kernel output (dmesg) and succeed if it finds that Secure boot is enabled
// The VMI was just created and may not be running yet. This is because we want to catch early boot logs.
func SecureBootExpecter(vmi *v1.VirtualMachineInstance) error {
	virtClient := kubevirt.Client()
	expecter, _, err := NewExpecter(virtClient, vmi, consoleConnectionTimeout)
	if err != nil {
		return err
	}
	defer expecter.Close()

	b := []expect.Batcher{&expect.BExp{R: "secureboot: Secure boot enabled"}}
	const expectBatchTimeout = 180 * time.Second
	res, err := expecter.ExpectBatch(b, expectBatchTimeout)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Kernel: %+v", res)
	}

	return err
}

// NetBootExpecter should be called on a VMI that has BIOS serial logging enabled
// It will parse the SeaBIOS output and succeed if it finds the string "iPXE"
// The VMI was just created and may not be running yet. This is because we want to catch early boot logs.
func NetBootExpecter(vmi *v1.VirtualMachineInstance) error {
	virtClient := kubevirt.Client()
	expecter, _, err := NewExpecter(virtClient, vmi, consoleConnectionTimeout)
	if err != nil {
		return err
	}
	defer expecter.Close()

	esc := UTFPosEscape
	b := []expect.Batcher{
		// SeaBIOS can use escape (\u001b) combinations for letter placement on screen
		// The regex below looks for the string "iPXE" and can detect it
		// even when these escape sequences are present
		&expect.BExp{R: "i(PXE|" + esc + "P" + esc + "X" + esc + "E)"},
	}
	const expectBatchTimeout = 30 * time.Second
	res, err := expecter.ExpectBatch(b, expectBatchTimeout)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("BIOS: %+v", res)
	}

	return err
}

// LinuxExpecter should be called early in the VMI boot process.
// It will catch the first logs printed by the Linux kernel.
// If this function succeeds, the VMI is guaranteed to be post-firmware (BIOS/EFI).
func LinuxExpecter(vmi *v1.VirtualMachineInstance) error {
	virtClient := kubevirt.Client()
	expecter, _, err := NewExpecter(virtClient, vmi, consoleConnectionTimeout)
	if err != nil {
		return err
	}
	defer expecter.Close()

	b := []expect.Batcher{
		&expect.BExp{R: `\[    0.000000\] Linux version`},
	}
	res, err := expecter.ExpectBatch(b, time.Minute)
	if err != nil {
		log.DefaultLogger().Object(vmi).Infof("Failed to find Linux boot logs in: %+v", res)
	}

	return err
}

// NewExpecter will connect to an already logged in VMI console and return the generated expecter it will wait `timeout` for the connection.
func NewExpecter(
	virtCli kubecli.KubevirtClient,
	vmi *v1.VirtualMachineInstance,
	timeout time.Duration,
	opts ...expect.Option) (expect.Expecter, <-chan error, error) {
	vmiReader, vmiWriter := io.Pipe()
	expecterReader, expecterWriter := io.Pipe()
	resCh := make(chan error)

	startTime := time.Now()
	serialConsoleOptions := &kubecli.SerialConsoleOptions{ConnectionTimeout: timeout}
	con, err := virtCli.VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, serialConsoleOptions)
	if err != nil {
		return nil, nil, err
	}
	serialConsoleCreateDuration := time.Since(startTime)
	if timeout-serialConsoleCreateDuration <= 0 {
		return nil, nil,
			fmt.Errorf(
				"creation of SerialConsole took %s - longer than given expecter timeout %s",
				serialConsoleCreateDuration.String(),
				timeout.String(),
			)
	}
	timeout -= serialConsoleCreateDuration

	go func() {
		resCh <- con.Stream(kubecli.StreamOptions{
			In:  vmiReader,
			Out: expecterWriter,
		})
	}()

	opts = append(opts, expect.SendTimeout(timeout), expect.Verbose(true), expect.VerboseWriter(ginkgo.GinkgoWriter))
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
//   - Use of `BatchSwitchCase`
//   - Multiline commands
//   - No more than one sequential send or receive
func ExpectBatchWithValidatedSend(expecter expect.Expecter, batch []expect.Batcher, timeout time.Duration) ([]expect.BatchRes, error) {
	sendFlag := false
	expectFlag := false
	previousSend := ""

	const minimumRequiredBatches = 2
	if len(batch) < minimumRequiredBatches {
		return nil, fmt.Errorf("ExpectBatchWithValidatedSend requires at least 2 batchers, supplied %v", batch)
	}

	for i, batcher := range batch {
		switch batcher.Cmd() {
		case expect.BatchExpect:
			if expectFlag {
				return nil, fmt.Errorf("two sequential expect.BExp are not allowed")
			}
			expectFlag = true
			sendFlag = false
			if _, ok := batch[i].(*expect.BExp); !ok {
				return nil, fmt.Errorf("ExpectBatchWithValidatedSend support only expect of type BExp")
			}
			bExp, _ := batch[i].(*expect.BExp)
			previousSend = regexp.QuoteMeta(previousSend)

			// Remove the \n since it is translated by the console to \r\n.
			previousSend = strings.TrimSuffix(previousSend, "\n")
			bExp.R = fmt.Sprintf("%s%s%s", previousSend, "((?s).*)", bExp.R)
		case expect.BatchSend:
			if sendFlag {
				return nil, fmt.Errorf("two sequential expect.BSend are not allowed")
			}
			sendFlag = true
			expectFlag = false
			previousSend = batcher.Arg()
		case expect.BatchSwitchCase:
			return nil, fmt.Errorf("ExpectBatchWithValidatedSend doesn't support BatchSwitchCase")
		default:
			return nil, fmt.Errorf("unknown command: ExpectBatchWithValidatedSend supports only BatchExpect and BatchSend")
		}
	}

	res, err := expecter.ExpectBatch(batch, timeout)
	return res, err
}

func RetValue(retcode string) string {
	return "\n" + retcode + CRLF + ".*" + PromptExpression
}
