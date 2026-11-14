/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package framework

import (
	"context"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Webhook readiness", func() {
	It("waits for admission to become effective beyond one polling interval", func(ctx SpecContext) {
		ctrl := gomock.NewController(GinkgoT())
		virtClient := kubecli.NewMockKubevirtClient(ctrl)
		vmClient := kubecli.NewMockVirtualMachineInterface(ctrl)
		virtClient.EXPECT().VirtualMachine(testNamespace).Return(vmClient).AnyTimes()
		// Simulate an API server that has not yet observed the webhook configurations.
		activationTime := time.Now().Add(2 * webhookAdmissionInterval)
		validated := false
		opts := metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}
		vmClient.EXPECT().Create(gomock.Any(), gomock.Any(), opts).DoAndReturn(
			func(_ context.Context, vm *v1.VirtualMachine, _ metav1.CreateOptions) (*v1.VirtualMachine, error) {
				if time.Now().Before(activationTime) {
					return vm.DeepCopy(), nil
				}
				if len(vm.Spec.Template.Spec.Domain.Devices.Disks) > 0 {
					validated = true
					return nil, k8serrors.NewInvalid(v1.Kind("VirtualMachine"), vm.Name, field.ErrorList{
						field.Invalid(field.NewPath("spec"), "missing-volume", "virtualmachine-validator.kubevirt.io"),
					})
				}
				mutated := vm.DeepCopy()
				mutated.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{UUID: "test-uuid"}
				return mutated, nil
			},
		).AnyTimes()
		f := &Framework{ctx: ctx, virtClient: virtClient}
		f.waitForWebhookAdmission()
		Expect(validated).To(BeTrue(), "readiness must also observe the validating webhook")
	}, SpecTimeout(5*time.Second))
})

var _ = Describe("Webhook shutdown", func() {
	It("waits for an active handler to finish", func(ctx SpecContext) {
		entered := make(chan struct{})
		release := make(chan struct{})
		var releaseOnce sync.Once
		unblock := func() { releaseOnce.Do(func() { close(release) }) }
		DeferCleanup(unblock)
		f := &Framework{
			webhookServer: &http.Server{
				ReadHeaderTimeout: webhookReadHeaderTimeout,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					close(entered)
					<-release
					w.WriteHeader(http.StatusNoContent)
				}),
			},
			webhookDone: make(chan struct{}),
		}
		var lc net.ListenConfig
		listener, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())
		go func() {
			defer GinkgoRecover()
			defer close(f.webhookDone)
			Expect(f.webhookServer.Serve(listener)).To(MatchError(http.ErrServerClosed))
		}()
		DeferCleanup(func() { Expect(f.webhookServer.Close()).To(Succeed()) })

		requestDone := make(chan error, 1)
		go func() {
			req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listener.Addr().String(), http.NoBody)
			if requestErr == nil {
				var response *http.Response
				response, requestErr = http.DefaultClient.Do(req)
				if requestErr == nil {
					_, requestErr = io.Copy(io.Discard, response.Body)
					_ = response.Body.Close()
				}
			}
			requestDone <- requestErr
		}()
		Eventually(ctx, entered).Should(BeClosed())
		stopped := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(stopped)
			f.Stop()
		}()
		Eventually(ctx, f.webhookDone).Should(BeClosed(), "Serve should exit while the handler is still active")
		Consistently(ctx, stopped, 100*time.Millisecond).ShouldNot(BeClosed())
		unblock()
		Eventually(ctx, requestDone).Should(Receive(Succeed()))
		Eventually(ctx, stopped).Should(BeClosed())
	}, SpecTimeout(5*time.Second))
})
