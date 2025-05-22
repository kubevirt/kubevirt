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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"google.golang.org/grpc"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha3 "kubevirt.io/kubevirt/pkg/hooks/v1alpha3"
)

type dynamicInfoServer struct {
	hookName          string
	hookPointName     string
	hookPointPriority int32
}

func (s dynamicInfoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	fmt.Fprintf(GinkgoWriter, "Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: s.hookName,
		Versions: []string{
			hooksV1alpha3.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     s.hookPointName,
				Priority: s.hookPointPriority,
			},
		},
	}, nil
}

func hookListenAndServe(socketPath string, hookName string, hookPointName string, hookPointPriority int32) (net.Listener, error) {
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, dynamicInfoServer{
		hookName:          hookName,
		hookPointName:     hookPointName,
		hookPointPriority: hookPointPriority,
	})
	fmt.Fprintf(GinkgoWriter, "Starting hook server exposing 'info' services on socket %s", socketPath)
	go func() {
		server.Serve(socket)
	}()
	return socket, nil
}

var _ = Describe("HooksManager", func() {
	Context("With existing sockets", func() {
		var socketDir string

		BeforeEach(func() {
			var err error
			socketDir, err = os.MkdirTemp("", "hooksocketdir")
			Expect(err).ToNot(HaveOccurred())
			os.MkdirAll(socketDir, os.ModePerm)
		})

		It("Should find sidecar", func() {
			hookPointName := hooksInfo.OnDefineDomainHookPointName

			hookPath := filepath.Join(socketDir, "hook-sidecar-0")
			os.MkdirAll(hookPath, os.ModePerm)
			socketPath := filepath.Join(hookPath, "hook1.sock")
			socket, err := hookListenAndServe(socketPath, "hook1", hookPointName, 0)
			Expect(err).ToNot(HaveOccurred())
			defer socket.Close()
			defer os.Remove(socketPath)

			manager := newManager(socketDir)
			err = manager.Collect(1, 10*time.Second)
			Expect(err).ToNot(HaveOccurred())

			callbackMaps := manager.CallbacksPerHookPoint
			Expect(callbackMaps).Should(HaveKey(hookPointName))
			Expect(callbackMaps[hookPointName]).Should(HaveLen(1))
		})

		It("Should find multiple sidecars on the same hook point", func() {
			hookPointName := hooksInfo.OnDefineDomainHookPointName
			hookNames := []string{"hook1", "hook2"}

			for i, hookName := range hookNames {
				hookPath := filepath.Join(socketDir, "hook-sidecar-"+strconv.Itoa(i))
				os.MkdirAll(hookPath, os.ModePerm)
				socketPath := filepath.Join(hookPath, fmt.Sprintf("%s.sock", hookName))
				socket, err := hookListenAndServe(socketPath, hookName, hookPointName, 0)
				Expect(err).ToNot(HaveOccurred())
				defer socket.Close()
				defer os.Remove(socketPath)
			}

			manager := newManager(socketDir)
			err := manager.Collect(uint(len(hookNames)), 10*time.Second)
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
			for i, hook := range hookNameList {
				hookPath := filepath.Join(socketDir, "hook-sidecar-"+strconv.Itoa(i))
				os.MkdirAll(hookPath, os.ModePerm)
				socketPath := filepath.Join(hookPath, fmt.Sprintf("%s.sock", hook.hookName))
				socket, err := hookListenAndServe(socketPath, hook.hookName, hook.hookPointName, 0)
				Expect(err).ToNot(HaveOccurred())
				defer socket.Close()
				defer os.Remove(socketPath)
			}

			manager := newManager(socketDir)
			err := manager.Collect(uint(len(hookNameList)), 10*time.Second)
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
