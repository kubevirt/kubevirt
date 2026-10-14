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

package grpc

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/grpc"

	cmdinfo "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	notifyinfo "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

const (
	timeout  = 5 * time.Second
	settling = 200 * time.Millisecond
)

var _ = Describe("RunServer", func() {

	var (
		tmpDir     string
		socketPath string
		stop       chan struct{}
		stopServer func()
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "kubevirt-grpc-test")
		Expect(err).To(Succeed())
		DeferCleanup(func() {
			Expect(os.RemoveAll(tmpDir)).To(Succeed())
		})

		socketPath = filepath.Join(tmpDir, "server.sock")

		stop = make(chan struct{})
		var once sync.Once
		stopServer = func() {
			once.Do(func() { close(stop) })
		}
		// make sure a spec never leaves the serving goroutines behind
		DeferCleanup(stopServer)
	})

	It("should hand the same server to every register function", func() {
		var registered []*grpc.Server
		record := func(server *grpc.Server) {
			registered = append(registered, server)
		}

		done, err := RunServer(stop, socketPath, record, record, record)
		Expect(err).To(Succeed())
		Expect(done).ToNot(BeNil())

		Expect(registered).To(HaveLen(3))
		Expect(registered[0]).ToNot(BeNil())
		for _, server := range registered {
			Expect(server).To(BeIdenticalTo(registered[0]))
		}
	})

	It("should start without any register function", func() {
		done, err := RunServer(stop, socketPath)
		Expect(err).To(Succeed())
		Expect(done).ToNot(BeNil())

		Expect(socketPath).To(BeAnExistingFile())
	})

	It("should serve every registered service on the socket", func() {
		notifyService := &fakeNotifyInfoServer{versions: []uint32{1, 2}}
		cmdService := &fakeCmdInfoServer{versions: []uint32{3}}

		_, err := RunServer(stop, socketPath,
			func(server *grpc.Server) { notifyinfo.RegisterNotifyInfoServer(server, notifyService) },
			func(server *grpc.Server) { cmdinfo.RegisterCmdInfoServer(server, cmdService) },
		)
		Expect(err).To(Succeed())

		conn, err := grpcutil.DialSocket(socketPath)
		Expect(err).To(Succeed())
		DeferCleanup(func() {
			Expect(conn.Close()).To(Succeed())
		})

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		DeferCleanup(cancel)

		notifyResponse, err := notifyinfo.NewNotifyInfoClient(conn).Info(ctx, &notifyinfo.NotifyInfoRequest{})
		Expect(err).To(Succeed())
		Expect(notifyResponse.SupportedNotifyVersions).To(Equal([]uint32{1, 2}))

		cmdResponse, err := cmdinfo.NewCmdInfoClient(conn).Info(ctx, &cmdinfo.CmdInfoRequest{})
		Expect(err).To(Succeed())
		Expect(cmdResponse.SupportedCmdVersions).To(Equal([]uint32{3}))
	})

	It("should replace a socket left behind by an earlier run", func() {
		Expect(os.WriteFile(socketPath, []byte("stale"), 0600)).To(Succeed())

		service := &fakeNotifyInfoServer{versions: []uint32{1}}
		_, err := RunServer(stop, socketPath,
			func(server *grpc.Server) { notifyinfo.RegisterNotifyInfoServer(server, service) },
		)
		Expect(err).To(Succeed())

		conn, err := grpcutil.DialSocket(socketPath)
		Expect(err).To(Succeed())
		DeferCleanup(func() {
			Expect(conn.Close()).To(Succeed())
		})

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		DeferCleanup(cancel)

		response, err := notifyinfo.NewNotifyInfoClient(conn).Info(ctx, &notifyinfo.NotifyInfoRequest{})
		Expect(err).To(Succeed())
		Expect(response.SupportedNotifyVersions).To(Equal([]uint32{1}))
	})

	It("should return an error when the socket cannot be created", func() {
		blocker := filepath.Join(tmpDir, "regular-file")
		Expect(os.WriteFile(blocker, []byte("not a directory"), 0600)).To(Succeed())

		done, err := RunServer(stop, filepath.Join(blocker, "nested", "server.sock"))
		Expect(err).To(MatchError(syscall.ENOTDIR))
		Expect(done).To(BeNil())
	})

	It("should keep serving until it is stopped", func() {
		done, err := RunServer(stop, socketPath)
		Expect(err).To(Succeed())

		Consistently(done, settling).ShouldNot(BeClosed())

		stopServer()

		Eventually(done, timeout).Should(BeClosed())
	})

	It("should remove the socket once it is stopped", func() {
		done, err := RunServer(stop, socketPath)
		Expect(err).To(Succeed())
		Expect(socketPath).To(BeAnExistingFile())

		stopServer()
		Eventually(done, timeout).Should(BeClosed())

		Expect(socketPath).ToNot(BeAnExistingFile())
	})
})

type fakeNotifyInfoServer struct {
	versions []uint32
}

func (f *fakeNotifyInfoServer) Info(_ context.Context, _ *notifyinfo.NotifyInfoRequest) (*notifyinfo.NotifyInfoResponse, error) {
	return &notifyinfo.NotifyInfoResponse{SupportedNotifyVersions: f.versions}, nil
}

type fakeCmdInfoServer struct {
	versions []uint32
}

func (f *fakeCmdInfoServer) Info(_ context.Context, _ *cmdinfo.CmdInfoRequest) (*cmdinfo.CmdInfoResponse, error) {
	return &cmdinfo.CmdInfoResponse{SupportedCmdVersions: f.versions}, nil
}
