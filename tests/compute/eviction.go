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

package compute

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	policy "k8s.io/api/policy/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIG("Eviction", func() {

	It("should not shutdown VM", func() {
		vmi := libvmifact.NewAlpine(
			libvmi.WithEvictionStrategy(v1.EvictionStrategyExternal),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)

		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

		ctx := context.Background()
		_, cancel := context.WithCancel(ctx)
		defer cancel()
		errChan := make(chan error)
		errors := make(chan error, 5)

		go func() {
			virtClient := kubevirt.Client()

			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(500 * time.Millisecond):
				}

				err := virtClient.CoreV1().Pods(pod.Namespace).EvictV1(ctx, &policy.Eviction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pod.Name,
						Namespace: pod.Namespace,
					},
					DeleteOptions: &metav1.DeleteOptions{
						Preconditions: &metav1.Preconditions{
							UID: &pod.UID,
						},
						GracePeriodSeconds: pointer.P(int64(0)),
					},
				})
				if err != nil && !k8serrors.IsTooManyRequests(err) {
					errChan <- err
					return
				}
				select {
				case errors <- err:
				default:
				}

			}

		}()
		Eventually(matcher.ThisVMI(vmi)).WithTimeout(time.Minute).WithPolling(20 * time.Second).Should(
			gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"EvacuationNodeName": Not(BeEmpty()),
				}),
			})),
		)
		Eventually(errors).WithTimeout(3 * time.Second).WithPolling(time.Second).
			Should(Receive(MatchError(ContainSubstring("Eviction triggered evacuation of VMI"))))
		for i := 0; i < 3; i++ {
			Eventually(errors).WithTimeout(3*time.Second).WithPolling(time.Second).
				Should(Receive(MatchError(ContainSubstring("Evacuation in progress"))), fmt.Sprintf("Failed in iteration %d", i+1))

		}

		Expect(matcher.ThisVMI(vmi)()).To(matcher.BeRunning())
		Expect(console.LoginToAlpine(vmi)).To(Succeed())
		Expect(errChan).To(BeEmpty())
	})
}))
