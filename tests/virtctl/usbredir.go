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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package virtctl

import (
	"context"
	"fmt"
	"net"
	"syscall"
	"time"

	"kubevirt.io/kubevirt/pkg/virtctl/usbredir"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libvmops"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"
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

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] USB Redirection", decorators.SigCompute, func() {
	const (
		enoughMemForSafeBiosEmulation = "32Mi"
		vmiRunTimeout                 = 90
		delayToCleanup                = 100 * time.Millisecond
		numTries                      = 3
	)
	var (
		virtClient      kubecli.KubevirtClient
		vmi             *v1.VirtualMachineInstance
		name, namespace string
	)

	BeforeEach(func() {
		// A VMI for each test to have fresh stack on server side
		vmi = libvmi.New(libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation), withClientPassthrough())
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, vmiRunTimeout)
		name = vmi.ObjectMeta.Name
		namespace = vmi.ObjectMeta.Namespace
		virtClient = kubevirt.Client()
	})

	It("Should fail when limit is reached", func() {
		type session struct {
			cancel  context.CancelFunc
			connect chan struct{}
			err     chan error
		}

		var tests []session
		for i := range v1.UsbClientPassthroughMaxNumberOf + 1 {
		retry_loop:
			for range numTries {
				ctx, cancelFn := context.WithCancel(context.Background())
				test := session{
					cancel:  cancelFn,
					connect: make(chan struct{}),
					err:     make(chan error),
				}
				ctx = context.WithValue(ctx, connectedKey, test.connect)
				go runConnectGoroutine(virtClient, name, namespace, ctx, test.err)

				if i == v1.UsbClientPassthroughMaxNumberOf {
					// Last test is meant to fail.
					tests = append(tests, test)
					break
				}

				// Till the last test, all sockets must be connected
				select {
				case <-test.connect:
					tests = append(tests, test)
					break retry_loop
				case err := <-test.err:
					Expect(err).To(MatchError(syscall.ECONNRESET))
				case <-time.After(time.Second):
					test.cancel()
				}

			}
		}

		numOfErrors := 0
		for i := range v1.UsbClientPassthroughMaxNumberOf + 1 {
			select {
			case err := <-tests[i].err:
				Expect(err).To(MatchError(ContainSubstring("websocket: bad handshake")))
				numOfErrors++
			case <-time.After(time.Second):
				tests[i].cancel()
			}
		}
		Expect(numOfErrors).To(Equal(1), "Only one connection should fail")
	})

	It("Should work several times", func() {
		for range 4 * v1.UsbClientPassthroughMaxNumberOf {
		retry_loop:
			for try := range numTries {
				ctx, cancelFn := context.WithCancel(context.Background())
				errch := make(chan error)
				go runConnectGoroutine(virtClient, name, namespace, ctx, errch)

				select {
				case err := <-errch:
					cancelFn()
					time.Sleep(delayToCleanup)
					Expect(err).To(MatchError(syscall.ECONNRESET))
					Expect(try).To(BeNumerically("<", numTries-1))
				case <-time.After(time.Second):
					cancelFn()
					time.Sleep(delayToCleanup)
					break retry_loop
				}
			}
		}
	})
})

func runConnectGoroutine(
	virtClient kubecli.KubevirtClient,
	name string,
	namespace string,
	ctx context.Context,
	errch chan error,
) {
	defer GinkgoRecover()

	usbredirStream, err := virtClient.VirtualMachineInstance(namespace).USBRedir(name)
	if err != nil {
		errch <- err
		return
	}
	usbredirConnect(usbredirStream, ctx)
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

func usbredirConnect(
	stream kvcorev1.StreamInterface,
	ctx context.Context,
) {
	usbredirClient, err := usbredir.NewUSBRedirClient(ctx, "localhost:0", stream)
	Expect(err).ToNot(HaveOccurred())
	usbredirClient.LaunchClient = false

	conn := make(chan error)
	go func() {
		defer GinkgoRecover()
		conn <- mockClientConnection(ctx, usbredirClient.GetProxyAddress())
	}()

	run := make(chan error)
	go func() {
		defer GinkgoRecover()
		run <- usbredirClient.Redirect("dead:beef")
	}()

	select {
	case err = <-conn:
		Expect(err).ToNot(HaveOccurred())
	case err = <-run:
		Expect(err).ToNot(HaveOccurred())
	case <-ctx.Done():
		err = <-run
		Expect(err).To(MatchError(ContainSubstring("context canceled")))
	}
}

func withClientPassthrough() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Devices.ClientPassthrough = &v1.ClientPassthroughDevices{}
	}
}
