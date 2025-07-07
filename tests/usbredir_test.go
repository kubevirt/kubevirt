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

package tests_test

import (
	"context"
	"errors"
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
	"kubevirt.io/client-go/log"

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

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] USB Redirection", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient
	const enoughMemForSafeBiosEmulation = "32Mi"
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance without usbredir support", func() {

		var vmi *v1.VirtualMachineInstance
		BeforeEach(func() {
			vmi = libvmi.New(libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
		})

		It("should fail to connect to VMI's usbredir socket", func() {
			usbredirVMI, err := virtClient.VirtualMachineInstance(vmi.ObjectMeta.Namespace).USBRedir(vmi.ObjectMeta.Name)
			Expect(err).To(HaveOccurred())
			Expect(usbredirVMI).To(BeNil())
		})
	})

	Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component] A VirtualMachineInstance with usbredir support", func() {

		var vmi *v1.VirtualMachineInstance
		var name, namespace string

		BeforeEach(func() {
			// A VMI for each test to have fresh stack on server side
			vmi = libvmi.New(libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation), withClientPassthrough())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
			name = vmi.ObjectMeta.Name
			namespace = vmi.ObjectMeta.Namespace
		})

		It("Should fail when limit is reached", func() {
			type session struct {
				cancel  context.CancelFunc
				connect chan struct{}
				err     chan error
			}

			var tests []session
			for i := 0; i <= v1.UsbClientPassthroughMaxNumberOf; i++ {
			retry_loop:
				for try := 0; try < 3; try++ {
					ctx, cancelFn := context.WithCancel(context.Background())
					test := session{
						cancel:  cancelFn,
						connect: make(chan struct{}),
						err:     make(chan error),
					}
					ctx = context.WithValue(ctx, "connected", test.connect)
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
						if !errors.Is(err, syscall.ECONNRESET) {
							log.Log.Reason(err).Info("Failed early. Unexpected error.")
							Fail("Improve error handling or fix underlying issue")
						}
						log.Log.Reason(err).Infof("Failed early. Try again (%d)", try)
					case <-time.After(time.Second):
						log.Log.Infof("Took too long. Try again (%d)", try)
						test.cancel()
					}

				}
			}

			numOfErrors := 0
			for i := 0; i <= v1.UsbClientPassthroughMaxNumberOf; i++ {
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
			for i := 0; i < 4*v1.UsbClientPassthroughMaxNumberOf; i++ {
			retry_loop:
				for try := 0; try < 3; try++ {
					ctx, cancelFn := context.WithCancel(context.Background())
					errch := make(chan error)
					go runConnectGoroutine(virtClient, name, namespace, ctx, errch)

					select {
					case err := <-errch:
						cancelFn()
						time.Sleep(100 * time.Millisecond)
						if try < 3 {
							log.Log.Reason(err).Infof("Failed. Try again (%d)", try)
						} else {
							Fail("Tried 3 times. Something is wrong")
						}
					case <-time.After(time.Second):
						cancelFn()
						time.Sleep(100 * time.Millisecond)
						break retry_loop
					}
				}
			}
		})
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

func usbredirConnect(
	stream kvcorev1.StreamInterface,
	ctx context.Context,
) {

	usbredirClient, err := usbredir.NewUSBRedirClient(ctx, "localhost:0", stream)
	Expect(err).ToNot(HaveOccurred())

	usbredirClient.ClientConnect = func(inCtx context.Context, device, address string) error {
		defer GinkgoRecover()
		conn, err := net.Dial("tcp", address)
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		buf := make([]byte, 1024, 1024)

		// write hello message to remote (VMI)
		if nw, err := conn.Write([]byte(helloMessageLocal)); err != nil {
			return err
		} else {
			Expect(nw).To(Equal(len(helloMessageLocal)))
		}

		// reading hello message from remote (VMI)
		if nr, err := conn.Read(buf); err != nil {
			return err
		} else {
			Expect(buf[0:nr]).ToNot(BeEmpty(), "response should not be empty")
			Expect(buf[0:nr]).To(HaveLen(len(helloMessageRemote)))
		}

		// Signal connected after read/write to be sure no TCP operation failed too
		if connected, ok := inCtx.Value("connected").(chan struct{}); ok {
			connected <- struct{}{}
		}

		select {
		case <-inCtx.Done():
			return inCtx.Err()
		}
	}

	run := make(chan error)
	go func() {
		defer GinkgoRecover()
		run <- usbredirClient.Redirect("dead:beef")
	}()

	select {
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
