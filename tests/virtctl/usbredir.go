//go:build !s390x

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
 * Copyright 2025 The KubeVirt Authors
 *
 */

package virtctl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libvmops"
)

// Capabilities from client side
var helloMessageLocal = []byte{
	0x00, 0x00, 0x00, 0x00, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x75, 0x73, 0x62, 0x72,
	0x65, 0x64, 0x69, 0x72, 0x20, 0x30, 0x2e, 0x31, 0x30, 0x2e, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00,
}

// Expected capabilities from QEMU's usbredir
var helloMessageRemote = []byte{
	0x00, 0x00, 0x00, 0x00, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x71, 0x65, 0x6d, 0x75,
	0x20, 0x75, 0x73, 0x62, 0x2d, 0x72, 0x65, 0x64, 0x69, 0x72, 0x20, 0x67, 0x75, 0x65, 0x73, 0x74,
	0x20, 0x35, 0x2e, 0x32, 0x2e, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x00, 0x00, 0x00,
}

type ctxKeyType string

const connectedKey ctxKeyType = "connected"

var _ = Describe(SIG("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] usbredir", decorators.SigCompute, func() {
	const (
		enoughMemForSafeBiosEmulation = "32Mi"
		vmiRunTimeout                 = 90
	)
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		// A VMI for each test to have fresh stack on server side
		vmi = libvmi.New(libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation), withClientPassthrough())
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, vmiRunTimeout)
	})

	It("Should fail when limit is reached", func() {
		var errchs []chan error
		for i := range v1.UsbClientPassthroughMaxNumberOf + 1 {
			Eventually(func(g Gomega) {
				cmd := newVirtctlCommand("usbredir",
					"--namespace", vmi.ObjectMeta.Namespace,
					"--no-launch", vmi.ObjectMeta.Name)
				connect := make(chan struct{})
				ctx := context.WithValue(cmd.Context(), connectedKey, connect)
				ctx, cancel := context.WithCancel(ctx)
				cmd.SetContext(ctx)
				DeferCleanup(cancel)

				err := make(chan error, 1)
				go runConnectGoroutine(cmd, err)

				if i == v1.UsbClientPassthroughMaxNumberOf {
					// Last test is meant to fail.
					errchs = append(errchs, err)
					return
				}

				// Till the last test, all sockets must be connected
				select {
				case <-connect:
					errchs = append(errchs, err)
				case err := <-err:
					g.Expect(err).To(MatchError(syscall.ECONNRESET))
				}
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		}

		numOfErrors := 0
		for i := range v1.UsbClientPassthroughMaxNumberOf + 1 {
			select {
			case err := <-errchs[i]:
				Expect(err).To(MatchError(ContainSubstring("websocket: bad handshake")))
				numOfErrors++
			case <-time.After(time.Second):
			}
		}
		Expect(numOfErrors).To(Equal(1), "Only one connection should fail")
	})

	It("Should work several times", func() {
		for range 4 * v1.UsbClientPassthroughMaxNumberOf {
			Eventually(func(g Gomega) {
				cmd := newVirtctlCommand("usbredir",
					"--namespace", vmi.ObjectMeta.Namespace,
					"--no-launch", vmi.ObjectMeta.Name)
				connect := make(chan struct{})
				ctx := context.WithValue(cmd.Context(), connectedKey, connect)
				ctx, cancel := context.WithTimeout(ctx, time.Second)
				cmd.SetContext(ctx)
				defer cancel()

				err := make(chan error, 1)
				go runConnectGoroutine(cmd, err)

				select {
				case <-connect:
					// Sent and Received message back. No errors.
				case err := <-err:
					g.Expect(err).To(MatchError(syscall.ECONNRESET))
				case <-ctx.Done():
					g.Expect(ctx.Err()).To(MatchError(ContainSubstring("context canceled")))
				}
			}, 5*time.Second, 1*time.Second).Should(Succeed())
		}
	})
}))

func runConnectGoroutine(cmd *cobra.Command, errch chan error) {
	defer GinkgoRecover()
	ctx := cmd.Context()
	// To find ip/port to connect
	rOut, wOut := io.Pipe()
	cmd.SetOut(wOut)
	defer rOut.Close()

	// To find errors
	rErr, wErr := io.Pipe()
	cmd.SetErr(wErr)
	defer rErr.Close()

	// Make remote buffer non-blocking for select
	remote := make(chan error, 1)
	go func() {
		defer GinkgoRecover()
		remote <- cmd.Execute()
		// Ends when we cancel() on the test.
	}()

	go func(r *io.PipeReader) {
		defer GinkgoRecover()
		scanner := bufio.NewScanner(r)
		if scanner.Scan() {
			// stderr should only be logging errors but we catch only known ones
			errch <- fmt.Errorf("virtctl stderr: %s", scanner.Text())
		}
		// Ends when PipeWriter closes or on error.
	}(rErr)

	addr := make(chan string, 1)
	go func(r *io.PipeReader) {
		defer GinkgoRecover()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if line := scanner.Text(); strings.Contains(line, "User can connect") {
				start := strings.Index(line, ":")
				addr <- strings.TrimSpace(line[start+1:])
				break
			}
		}
		// Ends when PipeWritter closes or after find ip.
	}(rOut)

	// Make local buffer non-blocking for select
	local := make(chan error, 1)
	go func() {
		defer GinkgoRecover()
		address := <-addr
		local <- mockClientConnection(ctx, address)
	}()

	select {
	case err := <-remote:
		// Remote errors can happen and are tested too.
		errch <- err
	case err := <-local:
		// Local errors happens on CI lanes e.g: TCP write/read failures
		errch <- err
	case <-ctx.Done():
		// Cancel happens on caller. Only expected error from context.
		Expect(ctx.Err()).To(MatchError(ContainSubstring("context canceled")))
	}
}

func mockClientConnection(ctx context.Context, address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	defer conn.Close()

	const bufSize = 1024
	buf := make([]byte, bufSize)

	// write hello message to remote (VMI)
	if nw, err := conn.Write(helloMessageLocal); err != nil {
		return err
	} else if nw != len(helloMessageLocal) {
		return fmt.Errorf("write: %d != %d", len(helloMessageLocal), nw)
	}

	// reading hello message from remote (VMI)
	if nr, err := conn.Read(buf); err != nil {
		return err
	} else if nr != len(helloMessageRemote) {
		return fmt.Errorf("read: %d != %d", len(helloMessageRemote), nr)
	}

	// Signal connected after read/write to be sure no TCP operation failed too
	if connected, ok := ctx.Value(connectedKey).(chan struct{}); ok {
		connected <- struct{}{}
	}

	<-ctx.Done()
	return ctx.Err()
}

func withClientPassthrough() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.ClientPassthrough = &v1.ClientPassthroughDevices{}
	}
}
