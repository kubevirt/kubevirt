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

package dmetrics_manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/testutils"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

func TestDownwardMetricsManager(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

var _ = Describe("DownwardMetrics manager", func() {
	type attempt struct {
		ctx    context.Context
		onExit func()
	}
	var manager *DownwardMetricsManager
	var vmi *v1.VirtualMachineInstance
	var attempts []attempt
	var retries chan struct{}
	var requestRetry func()

	BeforeEach(func() {
		GinkgoT().Setenv("NODE_NAME", "node")
		cmdclient.SetPodsBaseDir(GinkgoT().TempDir())
		DeferCleanup(cmdclient.SetPodsBaseDir, "/pods")
		socketPath := cmdclient.SocketFilePathOnHost("pod")
		Expect(os.MkdirAll(filepath.Dir(socketPath), 0755)).To(Succeed())
		Expect(os.WriteFile(socketPath, nil, 0600)).To(Succeed())

		vmi = v1.NewVMIReferenceWithUUID("default", "testvmi", "vmi")
		vmi.Spec.Domain.Devices.DownwardMetrics = &v1.DownwardMetrics{}
		vmi.Status.Phase = v1.Running
		vmi.Status.ActivePods = map[types.UID]string{"pod": "node"}

		manager = NewDownwardMetricsManager("node")
		DeferCleanup(manager.stop)
		attempts = nil
		manager.runServer = func(ctx context.Context, _, _, _ string, onExit func()) error {
			attempts = append(attempts, attempt{ctx: ctx, onExit: onExit})
			return nil
		}
		retries = make(chan struct{}, 1)
		requestRetry = func() { retries <- struct{}{} }
	})

	It("should release an exited server and allow the next sync to restart it", func() {
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		Expect(attempts).To(HaveLen(1))

		attempts[0].onExit()
		Expect(retries).To(Receive())
		Expect(attempts[0].ctx.Err()).To(MatchError(context.Canceled))
		Expect(manager.stopServer).To(BeEmpty())

		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		Expect(attempts).To(HaveLen(2))
		Expect(attempts[1].ctx.Err()).NotTo(HaveOccurred())
	})

	It("should handle an exit before StartServer has registered the server", func() {
		manager.runServer = func(_ context.Context, _, _, _ string, onExit func()) error {
			started := make(chan struct{})
			go func() {
				close(started)
				onExit()
			}()
			<-started
			return nil
		}
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		Eventually(retries).Should(Receive())
		manager.lock.Lock()
		defer manager.lock.Unlock()
		Expect(manager.stopServer).To(BeEmpty())
	})

	It("should request retry after releasing the manager lock", func() {
		lockWasFree := false
		Expect(manager.StartServer(vmi, 1, func() {
			lockWasFree = manager.lock.TryLock()
			if lockWasFree {
				manager.lock.Unlock()
			}
		})).To(Succeed())
		attempts[0].onExit()
		Expect(lockWasFree).To(BeTrue())
	})

	It("should return synchronous failures without recording a server or scheduling retry", func() {
		var serverCtx context.Context
		manager.runServer = func(ctx context.Context, _, _, _ string, _ func()) error {
			serverCtx = ctx
			return errors.New("setup failed")
		}
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(MatchError(ContainSubstring("setup failed")))
		Expect(serverCtx.Err()).To(MatchError(context.Canceled))
		Expect(manager.stopServer).To(BeEmpty())
		Expect(retries).NotTo(Receive())
	})

	It("should stop using only the VMI identity without scheduling retry", func() {
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		manager.StopServer(v1.NewVMIReferenceWithUUID(vmi.Namespace, vmi.Name, vmi.UID))
		attempts[0].onExit()
		Expect(attempts[0].ctx.Err()).To(MatchError(context.Canceled))
		Expect(manager.stopServer).To(BeEmpty())
		Expect(retries).NotTo(Receive())
	})

	It("should ignore an old callback after a replacement has started", func() {
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		manager.StopServer(vmi)
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())

		attempts[0].onExit()
		Expect(retries).NotTo(Receive())
		Expect(attempts[1].ctx.Err()).NotTo(HaveOccurred())
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		Expect(attempts).To(HaveLen(2))
		manager.StopServer(vmi)
		Expect(attempts[1].ctx.Err()).To(MatchError(context.Canceled))
	})

	It("should cancel all servers and prevent retry or restart after shutdown", func() {
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		otherVMI := vmi.DeepCopy()
		otherVMI.UID = "other-vmi"
		Expect(manager.StartServer(otherVMI, 1, requestRetry)).To(Succeed())
		manager.stop()
		for _, server := range attempts {
			Expect(server.ctx.Err()).To(MatchError(context.Canceled))
			server.onExit()
		}
		Expect(retries).NotTo(Receive())
		Expect(manager.stopServer).To(BeEmpty())
		Expect(manager.StartServer(vmi, 1, requestRetry)).To(Succeed())
		Expect(attempts).To(HaveLen(2))
	})
})
