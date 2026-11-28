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

package isolation

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/unsafepath"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

var _ = Describe("Isolation Detector", func() {

	Context("With an existing socket", func() {

		var socket net.Listener
		var tmpDir string
		var podUID string
		var finished chan struct{} = nil

		podUID = "pid-uid-1234"
		vm := api.NewMinimalVMIWithNS("default", "testvm")
		vm.UID = "1234"
		vm.Status = v1.VirtualMachineInstanceStatus{
			ActivePods: map[types.UID]string{
				types.UID(podUID): "myhost",
			},
		}
		vm.Status.NodeName = "myhost"

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())

			cmdclient.SetPodsBaseDir(tmpDir)

			socketFile := cmdclient.SocketFilePathOnHost(podUID)
			os.MkdirAll(filepath.Dir(socketFile), os.ModePerm)
			socket, err = net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			finished = make(chan struct{})
			go func() {
				for {
					conn, err := socket.Accept()
					if err != nil {
						close(finished)
						// closes when socket listener is closed
						return
					}
					conn.Close()
				}
			}()
		})

		AfterEach(func() {
			socket.Close()
			os.RemoveAll(tmpDir)
			if finished != nil {
				<-finished
			}
		})

		It("Should detect the PID of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector().Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Pid()).To(Equal(os.Getpid()))
		})

		It("Should detect the PID namespace of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector().Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PIDNamespace()).To(Equal(fmt.Sprintf("/proc/%d/ns/pid", os.Getpid())))
		})

		It("Should detect the Parent PID of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector().Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.PPid()).To(Equal(os.Getppid()))
		})

		It("Should detect the Mount root of the test suite", func() {
			result, err := NewSocketBasedIsolationDetector().Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			root, err := result.MountRoot()
			Expect(err).ToNot(HaveOccurred())
			Expect(unsafepath.UnsafeAbsolute(root.Raw())).To(Equal(fmt.Sprintf("/proc/%d/root", os.Getpid())))
		})

		It("Should cache the isolation result on repeated Detect calls", func() {
			detector := NewSocketBasedIsolationDetector()
			result1, err := detector.Detect(vm)
			Expect(err).ToNot(HaveOccurred())

			// Close the socket to verify subsequent Detect calls use cache rather than dialing
			socket.Close()
			<-finished
			finished = nil

			result2, err := detector.Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result2).To(Equal(result1))
		})

		It("Should discard the cached isolation result on Teardown", func() {
			detector := NewSocketBasedIsolationDetector()
			_, err := detector.Detect(vm)
			Expect(err).ToNot(HaveOccurred())

			// Close socket and teardown VM
			socket.Close()
			<-finished
			finished = nil

			detector.Teardown(vm)

			// Next call must fail because cache is cleared and socket is closed
			_, err = detector.Detect(vm)
			Expect(err).To(HaveOccurred())
		})

		It("Should not cache detection errors", func() {
			// Close socket so initial detect fails
			socket.Close()
			<-finished
			finished = nil

			detector := NewSocketBasedIsolationDetector()
			_, err := detector.Detect(vm)
			Expect(err).To(HaveOccurred())

			// Recreate the socket
			socketFile := cmdclient.SocketFilePathOnHost(podUID)
			socket, err = net.Listen("unix", socketFile)
			Expect(err).ToNot(HaveOccurred())
			finished = make(chan struct{})
			go func() {
				for {
					conn, err := socket.Accept()
					if err != nil {
						close(finished)
						return
					}
					conn.Close()
				}
			}()

			result, err := detector.Detect(vm)
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Pid()).To(Equal(os.Getpid()))
		})

		It("Should coordinate Teardown with in-flight Detect and not cache or reuse stale results", func() {
			detector := &socketBasedIsolationDetector{}

			var newDetectErr error
			var newDetectRes IsolationResult
			secondDetectDone := make(chan struct{})

			detector.detectCallback = func() {
				// While the first Detect is in-flight, Teardown is called concurrently
				detector.Teardown(vm)

				// Close the socket so that subsequent dials cannot succeed
				socket.Close()
				<-finished
				finished = nil

				// Concurrently launch a new Detect call after Teardown
				// It must not join the old singleflight operation nor reuse the old result
				go func() {
					defer close(secondDetectDone)
					newDetectRes, newDetectErr = detector.Detect(vm)
				}()
			}

			// In-flight Detect call
			_, inFlightErr := detector.Detect(vm)
			// The in-flight Detect was invalidated by Teardown
			Expect(inFlightErr).To(MatchError(ContainSubstring("isolation detection invalidated by teardown")))

			// Wait for the new Detect call launched after Teardown to finish
			<-secondDetectDone
			// It should fail because it did not join the old singleflight and socket was closed
			Expect(newDetectErr).To(HaveOccurred())
			Expect(newDetectRes).To(BeNil())

			// Verify that the cache is empty and no stale launcher PID remains cached
			_, err := detector.Detect(vm)
			Expect(err).To(HaveOccurred())
		})

		It("Should handle concurrent Detect and Teardown calls safely", func() {
			detector := NewSocketBasedIsolationDetector()
			var wg sync.WaitGroup
			numGoroutines := 20

			for range numGoroutines {
				wg.Add(2)
				go func() {
					defer wg.Done()
					_, _ = detector.Detect(vm)
				}()
				go func() {
					defer wg.Done()
					detector.Teardown(vm)
				}()
			}
			wg.Wait()
		})
	})
})
