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
	"net"
	"time"

	"kubevirt.io/kubevirt/pkg/virtctl/usbredir"

	"kubevirt.io/kubevirt/tests/decorators"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	kvcorev1 "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/typed/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests"
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
			vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
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
			vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
			name = vmi.ObjectMeta.Name
			namespace = vmi.ObjectMeta.Namespace
		})

		It("Should fail when limit is reached", func() {
			cancelFns := make([]context.CancelFunc, v1.UsbClientPassthroughMaxNumberOf+1)
			errors := make([]chan error, v1.UsbClientPassthroughMaxNumberOf+1)
			for i := 0; i <= v1.UsbClientPassthroughMaxNumberOf; i++ {
				ctx, cancelFn := context.WithCancel(context.Background())
				cancelFns[i] = cancelFn
				errors[i] = make(chan error)
				go runConnectGoroutine(virtClient, name, namespace, ctx, errors[i])
				// avoid too fast requests which might get denied by server (to avoid flakyness)
				time.Sleep(100 * time.Millisecond)
			}

			for i := 0; i <= v1.UsbClientPassthroughMaxNumberOf; i++ {
				select {
				case err := <-errors[i]:
					Expect(err).To(MatchError(ContainSubstring("websocket: bad handshake")))
					Expect(i).To(Equal(v1.UsbClientPassthroughMaxNumberOf))
				case <-time.After(time.Second):
					cancelFns[i]()
					Expect(i).ToNot(Equal(v1.UsbClientPassthroughMaxNumberOf))
				}
			}
		})

		It("Should work in parallel", func() {
			cancelFns := make([]context.CancelFunc, v1.UsbClientPassthroughMaxNumberOf)
			errors := make([]chan error, v1.UsbClientPassthroughMaxNumberOf)
			for i := 0; i < v1.UsbClientPassthroughMaxNumberOf; i++ {
				errors[i] = make(chan error)
				ctx, cancelFn := context.WithCancel(context.Background())
				cancelFns[i] = cancelFn
				go runConnectGoroutine(virtClient, name, namespace, ctx, errors[i])
				// avoid too fast requests which might get denied by server (to avoid flakyness)
				time.Sleep(100 * time.Millisecond)
			}

			for i := 0; i < v1.UsbClientPassthroughMaxNumberOf; i++ {
				select {
				case err := <-errors[i]:
					Expect(err).ToNot(HaveOccurred())
				case <-time.After(time.Second):
					cancelFns[i]()
				}
			}
		})

		It("Should work several times", func() {
			for i := 0; i < 4*v1.UsbClientPassthroughMaxNumberOf; i++ {
				ctx, cancelFn := context.WithCancel(context.Background())
				errch := make(chan error)
				go runConnectGoroutine(virtClient, name, namespace, ctx, errch)

				select {
				case err := <-errch:
					Expect(err).ToNot(HaveOccurred())
				case <-time.After(time.Second):
				}
				cancelFn()
				time.Sleep(time.Second)
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
		conn, err := net.Dial("tcp", address)
		Expect(err).ToNot(HaveOccurred())
		defer conn.Close()

		buf := make([]byte, 1024, 1024)

		// write hello message to remote (VMI)
		nw, err := conn.Write([]byte(helloMessageLocal))
		Expect(err).ToNot(HaveOccurred())
		Expect(nw).To(Equal(len(helloMessageLocal)))

		// reading hello message from remote (VMI)
		nr, err := conn.Read(buf)
		Expect(err).ToNot(HaveOccurred())
		Expect(buf[0:nr]).ToNot(BeEmpty(), "response should not be empty")
		Expect(buf[0:nr]).To(HaveLen(len(helloMessageRemote)))
		select {
		case <-inCtx.Done():
			return inCtx.Err()
		}
	}

	run := make(chan error)
	go func() {
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
