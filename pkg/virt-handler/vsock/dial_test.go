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
 *
 */

package vsock_test

import (
	"errors"
	"net"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/mdlayher/vsock"
	mount "github.com/moby/sys/mountinfo"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virthandlervsock "kubevirt.io/kubevirt/pkg/virt-handler/vsock"
	virtvsock "kubevirt.io/kubevirt/pkg/vsock"
	"kubevirt.io/kubevirt/pkg/vsock/mode"
)

var _ = Describe("Dialer", func() {
	const (
		testPid         = 12345
		testPort uint32 = 9999
		testCID  uint32 = 123
	)

	var (
		fakeIsolation *fakeIsolationDetector
		vmi           *v1.VirtualMachineInstance
		procPath      string
		dialer        *virthandlervsock.Dialer

		dialFn          func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error)
		dialFnCalled    bool
		netnsDoFn       func(pid int, fn func() error) error
		netnsDoFnCalled bool
		tlsWrapperFn    func(conn net.Conn) *fakeTLSConn
	)

	BeforeEach(func() {
		procPath = GinkgoT().TempDir()

		vmi = libvmi.New()
		vmi.Status.VSOCKCID = pointer.P(testCID)

		fakeIsolation = &fakeIsolationDetector{
			pidResult: testPid,
			err:       nil,
		}

		dialFnCalled = false
		dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
			return &vsock.Conn{}, nil
		}

		netnsDoFnCalled = false
		netnsDoFn = func(pid int, fn func() error) error {
			return fn()
		}

		tlsWrapperFn = func(conn net.Conn) *fakeTLSConn {
			Fail("tlsWrapperFn called unexpectedly - tests that use TLS should override this function")
			return nil
		}

		dialer = virthandlervsock.NewDialer(fakeIsolation, procPath,
			func(pid int, fn func() error) error {
				netnsDoFnCalled = true
				return netnsDoFn(pid, fn)
			},
			func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				dialFnCalled = true
				return dialFn(contextID, port, cfg)
			},
			func(conn net.Conn) virthandlervsock.TLSConn {
				return tlsWrapperFn(conn)
			})
	})

	Context("with valid configuration", func() {
		It("should dial successfully without TLS", func() {
			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})

		It("should use the VMI's VSOCK CID", func() {
			customCID := uint32(456)
			vmi.Status.VSOCKCID = &customCID

			dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				Expect(contextID).To(Equal(customCID))
				return &vsock.Conn{}, nil
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})

		It("should use the specified port", func() {
			customPort := uint32(8080)

			dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				Expect(port).To(Equal(customPort))
				return &vsock.Conn{}, nil
			}

			conn, err := dialer.Dial(vmi, customPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})
	})

	Context("VSOCK namespace mode", func() {
		setupVsockMode := func(sysctlValue string) {
			sysDir := filepath.Join(procPath, "sys", "net", "vsock")
			Expect(os.MkdirAll(sysDir, 0o755)).To(Succeed())
			filePath := filepath.Join(sysDir, "ns_mode")
			Expect(os.WriteFile(filePath, []byte(sysctlValue+"\n"), 0o600)).To(Succeed())
		}

		It("should use local CID in local mode", func() {
			setupVsockMode(mode.ModeLocal)

			dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				Expect(contextID).To(Equal(virtvsock.LocalCID))
				Expect(port).To(Equal(testPort))
				return &vsock.Conn{}, nil
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})

		It("should use VMI CID in global mode", func() {
			setupVsockMode(mode.ModeGlobal)

			dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				Expect(contextID).To(Equal(testCID))
				Expect(port).To(Equal(testPort))
				return &vsock.Conn{}, nil
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})

		It("should default to VMI CID when mode cannot be determined", func() {
			dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				Expect(contextID).To(Equal(testCID))
				Expect(port).To(Equal(testPort))
				return &vsock.Conn{}, nil
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})
	})

	Context("network namespace", func() {
		It("should execute dial in the correct namespace", func() {
			netnsDoFn = func(pid int, fn func() error) error {
				Expect(pid).To(Equal(testPid))
				return fn()
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})
	})

	Context("with invalid configuration", func() {
		It("should return error when isolation detection fails", func() {
			expectedErr := errors.New("isolation detection failed")
			fakeIsolation.err = expectedErr

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).To(MatchError(expectedErr))
			Expect(conn).To(BeNil())
			Expect(netnsDoFnCalled).To(BeFalse())
			Expect(dialFnCalled).To(BeFalse())
		})

		It("should return error when VSOCK is not enabled", func() {
			vmi.Status.VSOCKCID = nil

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).To(MatchError("VSOCK is not enabled for the VM"))
			Expect(conn).To(BeNil())
			Expect(netnsDoFnCalled).To(BeFalse())
			Expect(dialFnCalled).To(BeFalse())
		})
	})

	Context("when dial fails", func() {
		It("should return error from dial function", func() {
			expectedErr := errors.New("connection refused")
			dialFn = func(contextID, port uint32, cfg *vsock.Config) (*vsock.Conn, error) {
				return nil, expectedErr
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).To(MatchError(expectedErr))
			Expect(conn).To(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
		})

		It("should return error from netns Do function", func() {
			expectedErr := errors.New("netns error")
			netnsDoFn = func(pid int, fn func() error) error {
				Expect(pid).To(Equal(testPid))
				return expectedErr
			}

			conn, err := dialer.Dial(vmi, testPort, false)
			Expect(err).To(MatchError(expectedErr))
			Expect(conn).To(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeFalse())
		})
	})

	Context("TLS parameter", func() {
		It("should wrap connection with TLS when useTLS is true", func() {
			tlsWrapperCalled := false
			tlsWrapperFn = func(conn net.Conn) *fakeTLSConn {
				tlsWrapperCalled = true
				return &fakeTLSConn{Conn: conn}
			}

			conn, err := dialer.Dial(vmi, testPort, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(conn).ToNot(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
			Expect(tlsWrapperCalled).To(BeTrue())

			mockConn, ok := conn.(*fakeTLSConn)
			Expect(ok).To(BeTrue())
			Expect(mockConn.handshakeCalled).To(BeTrue())
		})

		It("should return error when TLS handshake fails", func() {
			expectedErr := errors.New("handshake failed")
			tlsWrapperCalled := false
			tlsWrapperFn = func(conn net.Conn) *fakeTLSConn {
				tlsWrapperCalled = true
				return &fakeTLSConn{
					Conn:         conn,
					handshakeErr: expectedErr,
				}
			}

			conn, err := dialer.Dial(vmi, testPort, true)
			Expect(err).To(MatchError(expectedErr))
			Expect(conn).To(BeNil())
			Expect(netnsDoFnCalled).To(BeTrue())
			Expect(dialFnCalled).To(BeTrue())
			Expect(tlsWrapperCalled).To(BeTrue())
		})

		It("should close connection when TLS handshake fails", func() {
			expectedErr := errors.New("handshake failed")
			var capturedMockConn *fakeTLSConn
			tlsWrapperFn = func(conn net.Conn) *fakeTLSConn {
				capturedMockConn = &fakeTLSConn{
					Conn:         conn,
					handshakeErr: expectedErr,
				}
				return capturedMockConn
			}

			conn, err := dialer.Dial(vmi, testPort, true)
			Expect(err).To(MatchError(expectedErr))
			Expect(conn).To(BeNil())
			Expect(capturedMockConn.closeCalled).To(BeTrue())
		})
	})
})

type fakeIsolationResult struct {
	pid int
}

func (f *fakeIsolationResult) Pid() int {
	return f.pid
}

func (f *fakeIsolationResult) PPid() int {
	panic("should not be called")
}

func (f *fakeIsolationResult) PIDNamespace() string {
	panic("should not be called")
}

func (f *fakeIsolationResult) MountRoot() (*safepath.Path, error) {
	panic("should not be called")
}

func (f *fakeIsolationResult) MountNamespace() string {
	panic("should not be called")
}

func (f *fakeIsolationResult) Mounts(_ mount.FilterFunc) ([]*mount.Info, error) {
	panic("should not be called")
}

type fakeIsolationDetector struct {
	pidResult int
	err       error
}

func (f *fakeIsolationDetector) Detect(_ *v1.VirtualMachineInstance) (isolation.IsolationResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &fakeIsolationResult{pid: f.pidResult}, nil
}

func (f *fakeIsolationDetector) DetectForSocket(_ string) (isolation.IsolationResult, error) {
	panic("should not be called")
}

type fakeTLSConn struct {
	net.Conn
	handshakeCalled bool
	handshakeErr    error
	closeCalled     bool
	closeErr        error
}

func (m *fakeTLSConn) Close() error {
	m.closeCalled = true
	return m.closeErr
}

func (m *fakeTLSConn) Handshake() error {
	m.handshakeCalled = true
	return m.handshakeErr
}
