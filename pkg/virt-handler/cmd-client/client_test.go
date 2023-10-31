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

package cmdclient

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

var _ = Describe("Virt remote commands", func() {

	var err error
	var shareDir string
	var socketsDir string
	var podsDir string
	var vmi *v1.VirtualMachineInstance
	var podSocketFile string

	host := "myhost"
	podUID := "poduid123"

	BeforeEach(func() {
		vmi = api.NewMinimalVMI("testvmi")
		vmi.UID = types.UID("1234")
		vmi.Status = v1.VirtualMachineInstanceStatus{
			ActivePods: map[types.UID]string{
				types.UID(podUID): host,
			},
		}

		shareDir, err = os.MkdirTemp("", "kubevirt-share")
		Expect(err).ToNot(HaveOccurred())

		podsDir, err = os.MkdirTemp("", "pods")
		Expect(err).ToNot(HaveOccurred())

		socketsDir = filepath.Join(shareDir, "sockets")
		os.Mkdir(socketsDir, 0755)

		SetLegacyBaseDir(shareDir)
		SetPodsBaseDir(podsDir)

		podSocketFile = SocketFilePathOnHost(podUID)

		os.MkdirAll(filepath.Dir(podSocketFile), 0755)
		f, err := os.Create(podSocketFile)
		Expect(err).ToNot(HaveOccurred())
		f.Close()

	})

	AfterEach(func() {
		os.RemoveAll(shareDir)
		os.RemoveAll(podsDir)
	})

	Context("client", func() {
		It("socket from UID", func() {
			sock, err := FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(sock).To(Equal(podSocketFile))

			sock = SocketOnGuest()
			Expect(sock).To(Equal(filepath.Join(shareDir, "sockets", StandardLauncherSocketFileName)))

			// falls back to returning a legacy socket name if it exists
			// This is for backwards compatibility
			f, err := os.Create(filepath.Join(socketsDir, "1234_sock"))
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			sock, err = FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(sock).To(Equal(filepath.Join(shareDir, "sockets", "1234_sock")))
		})

		It("Listing all sockets", func() {
			// the new socket is already created in the Before function

			// create a legacy socket to ensure we find both new and legacy sockets
			f, err := os.Create(filepath.Join(socketsDir, "1234_sock"))
			Expect(err).ToNot(HaveOccurred())
			f.Close()

			// listing all sockets should detect both the new and legacy sockets
			sockets, err := ListAllSockets()
			Expect(err).ToNot(HaveOccurred())
			Expect(sockets).To(HaveLen(2))
		})

		It("Detect unresponsive socket", func() {
			sock, err := FindSocketOnHost(vmi)
			Expect(err).ToNot(HaveOccurred())

			os.RemoveAll(sock)

			// unresponsive is true when no socket file exists
			unresponsive := IsSocketUnresponsive(sock)
			Expect(unresponsive).To(BeTrue())

			// unresponsive is false when socket file exists
			os.Mkdir(filepath.Dir(sock), 0755)
			f, err := os.Create(sock)
			Expect(err).ToNot(HaveOccurred())
			f.Close()
			unresponsive = IsSocketUnresponsive(sock)
			Expect(unresponsive).To(BeFalse())

			// unresponsive is true when marked as unresponsive
			MarkSocketUnresponsive(sock)
			unresponsive = IsSocketUnresponsive(sock)
			Expect(unresponsive).To(BeTrue())
		})

		It("Determine legacy sockets vs new socket paths", func() {
			legacy := IsLegacySocket("/some/path/something_sock")
			Expect(legacy).To(BeTrue())

			legacy = IsLegacySocket(filepath.Join("/some/path", StandardLauncherSocketFileName))
			Expect(legacy).To(BeFalse())

			monEnabled := SocketMonitoringEnabled("/some/path/something_sock")
			Expect(monEnabled).To(BeFalse())

			monEnabled = SocketMonitoringEnabled(filepath.Join("/some/path", StandardLauncherSocketFileName))
			Expect(monEnabled).To(BeTrue())
		})

		Context("exec", func() {
			var (
				ctrl          *gomock.Controller
				mockCmdClient *cmdv1.MockCmdClient
				client        LauncherClient
			)

			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				mockCmdClient = cmdv1.NewMockCmdClient(ctrl)
				client = newV1Client(mockCmdClient, nil)
			})

			var (
				testDomainName           = "test"
				testCommand              = "testCmd"
				testArgs                 = []string{"-v", "2"}
				testClientErr            = errors.New("client error")
				testStdOut               = "stdOut"
				testTimeoutSeconds int32 = 10

				expectExec = func() *gomock.Call {
					return mockCmdClient.EXPECT().Exec(gomock.Any(), &cmdv1.ExecRequest{
						DomainName:     testDomainName,
						Command:        testCommand,
						Args:           testArgs,
						TimeoutSeconds: testTimeoutSeconds,
					})
				}
				expectGuestPing = func() *gomock.Call {
					return mockCmdClient.EXPECT().GuestPing(gomock.Any(), &cmdv1.GuestPingRequest{
						DomainName:     testDomainName,
						TimeoutSeconds: testTimeoutSeconds,
					})
				}
				expectQemuVersion = func() *gomock.Call {
					return mockCmdClient.EXPECT().GetQemuVersion(gomock.Any(), &cmdv1.EmptyRequest{})
				}
			)
			It("calls cmdclient.GetQemuVersion", func() {
				fakeQemuVersion := "7.2.0"
				expectQemuVersion().Return(&cmdv1.QemuVersionResponse{Version: fakeQemuVersion}, nil)
				qemuVersion, err := client.GetQemuVersion()
				Expect(err).ToNot(HaveOccurred())
				Expect(qemuVersion).To(Equal(fakeQemuVersion))
			})
			It("calls cmdclient.Exec", func() {
				expectExec().Times(1)
				client.Exec(testDomainName, testCommand, testArgs, testTimeoutSeconds)
			})
			It("returns client errors", func() {
				expectExec().Times(1).Return(&cmdv1.ExecResponse{}, testClientErr)
				_, _, err := client.Exec(testDomainName, testCommand, testArgs, testTimeoutSeconds)
				Expect(err).To(HaveOccurred())
			})
			It("returns exitCode and stdOut if possible", func() {
				expectExec().Times(1).Return(&cmdv1.ExecResponse{
					ExitCode: 1,
					StdOut:   testStdOut,
				}, nil)
				exitCode, stdOut, _ := client.Exec(testDomainName, testCommand, testArgs, testTimeoutSeconds)
				Expect(exitCode).To(Equal(1))
				Expect(stdOut).To(Equal(testStdOut))
			})
			It("provide ctx with a shortTimeout", func() {
				expectExec().Times(1).Do(func(ctx context.Context, _ *cmdv1.ExecRequest, _ ...grpc.CallOption) (*cmdv1.ExecResponse, error) {
					_, ok := ctx.Deadline()
					Expect(ok).To(BeTrue())
					return nil, nil
				})
				client.Exec(testDomainName, testCommand, testArgs, testTimeoutSeconds)
			})
			It("calls cmdclient.GuestPing", func() {
				expectGuestPing().Times(1)
				client.GuestPing(testDomainName, testTimeoutSeconds)
			})
			It("returns client error", func() {
				expectGuestPing().Times(1).Return(&cmdv1.GuestPingResponse{}, testClientErr)
				err := client.GuestPing(testDomainName, testTimeoutSeconds)
				Expect(err).To(HaveOccurred())
			})
			It("should succeed", func() {
				expectGuestPing().Times(1).Return(&cmdv1.GuestPingResponse{Response: &cmdv1.Response{Success: true}}, nil)
				err := client.GuestPing(testDomainName, testTimeoutSeconds)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
