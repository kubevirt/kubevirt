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

package hooks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/util/rand"

	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha3 "kubevirt.io/kubevirt/pkg/hooks/v1alpha3"
)

type infoServer struct {
	hooksInfo.InfoResult
}

func (s *infoServer) Info(
	_ context.Context,
	_ *hooksInfo.InfoParams,
) (*hooksInfo.InfoResult, error) {
	GinkgoWriter.Println("Hook's Info method has been called")
	return &s.InfoResult, nil
}

type testCase struct {
	socketPath string
	info       infoServer

	// error from the Run(), will be read on Stop()
	errch  chan error
	ctx    context.Context
	cancel context.CancelFunc
}

// Create boilerplate for the test
func newTestCase(socketDir, name string) *testCase {
	hookPath := filepath.Join(socketDir, "hook-sidecar-"+rand.String(5))
	Expect(os.MkdirAll(hookPath, os.ModePerm)).ToNot(HaveOccurred())
	socketPath := filepath.Join(hookPath, fmt.Sprintf("%s.sock", name))

	ctx, cancel := context.WithCancel(context.Background())
	return &testCase{
		socketPath: socketPath,
		info: infoServer{
			hooksInfo.InfoResult{
				Name:       name,
				Versions:   []string{hooksV1alpha3.Version},
				HookPoints: []*hooksInfo.HookPoint{},
			},
		},
		errch:  make(chan error),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (t *testCase) Run() {
	grpcDone := make(chan error)
	server := grpc.NewServer([]grpc.ServerOption{}...)

	go func(server *grpc.Server, grpcDone chan<- error) {
		defer GinkgoRecover()
		socket, err := net.Listen("unix", t.socketPath)
		Expect(err).ToNot(HaveOccurred())
		defer socket.Close()

		hooksInfo.RegisterInfoServer(server, &t.info)

		GinkgoWriter.Printf("Starting hook server exposing 'info' services on socket %s\n", t.socketPath)
		grpcDone <- server.Serve(socket)
	}(server, grpcDone)

	go func(server *grpc.Server, grpcDone <-chan error) {
		defer GinkgoRecover()
		var err error
		select {
		case err = <-grpcDone:
		case <-t.ctx.Done():
			err = t.ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			server.Stop()
		}

		// Wait Stop() read it
		t.errch <- err
	}(server, grpcDone)
}

func (t *testCase) Stop() error {
	defer os.Remove(t.socketPath)

	// Check if error already
	select {
	case err := <-t.errch:
		return err
	default:
	}

	t.cancel()
	return <-t.errch
}

const collectTimeout = 10 * time.Second

var _ = Describe("HooksManager", func() {
	Context("With existing sockets", func() {
		var socketDir string

		BeforeEach(func() {
			socketDir = GinkgoT().TempDir()
		})

		It("Should find sidecar", func() {
			hookPointName := hooksInfo.OnDefineDomainHookPointName

			t := newTestCase(socketDir, "hook1")
			t.info.HookPoints = append(t.info.HookPoints, &hooksInfo.HookPoint{
				Name:     hookPointName,
				Priority: 0,
			})
			t.Run()

			manager := newManager(socketDir)
			err := manager.Collect(1, collectTimeout)
			Expect(err).ToNot(HaveOccurred())

			callbackMaps := manager.CallbacksPerHookPoint
			Expect(callbackMaps).Should(HaveKey(hookPointName))
			Expect(callbackMaps[hookPointName]).Should(HaveLen(1))
			Expect(t.Stop()).ToNot(HaveOccurred())
		})

		It("Should find multiple sidecars on the same hook point", func() {
			hookPointName := hooksInfo.OnDefineDomainHookPointName
			hookNames := []string{"hook1", "hook2"}

			for _, hookName := range hookNames {
				t := newTestCase(socketDir, hookName)
				t.info.HookPoints = append(t.info.HookPoints, &hooksInfo.HookPoint{
					Name:     hookPointName,
					Priority: 0,
				})
				t.Run()
				DeferCleanup(func() { Expect(t.Stop()).ToNot(HaveOccurred()) })
			}

			manager := newManager(socketDir)
			err := manager.Collect(uint(len(hookNames)), collectTimeout)
			Expect(err).ToNot(HaveOccurred())

			callbackMaps := manager.CallbacksPerHookPoint
			Expect(callbackMaps).Should(HaveKey(hookPointName))
			Expect(callbackMaps[hookPointName]).Should(HaveLen(len(hookNames)))
		})

		It("Should find multiple sidecars on different hook points", func() {
			hookNameList := []struct {
				hookName      string
				hookPointName string
			}{
				{"hook1", hooksInfo.OnDefineDomainHookPointName},
				{"hook2", hooksInfo.PreCloudInitIsoHookPointName},
			}
			for _, hook := range hookNameList {
				t := newTestCase(socketDir, hook.hookName)
				t.info.HookPoints = append(t.info.HookPoints, &hooksInfo.HookPoint{
					Name:     hook.hookPointName,
					Priority: 0,
				})
				t.Run()
				DeferCleanup(func() { Expect(t.Stop()).ToNot(HaveOccurred()) })
			}

			manager := newManager(socketDir)
			err := manager.Collect(uint(len(hookNameList)), collectTimeout)
			Expect(err).ToNot(HaveOccurred())

			callbackMaps := manager.CallbacksPerHookPoint

			for _, hook := range hookNameList {
				Expect(callbackMaps).Should(HaveKey(hook.hookPointName))
				Expect(callbackMaps[hook.hookPointName]).Should(HaveLen(1))
			}
		})

		AfterEach(func() {
			os.RemoveAll(socketDir)
		})
	})
})
