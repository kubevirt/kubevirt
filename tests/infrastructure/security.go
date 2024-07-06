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
 * Copyright The KubeVirt Authors
 *
 */

package infrastructure

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
)

var _ = DescribeInfra("Node Restriction", decorators.RequiresTwoSchedulableNodes, decorators.Kubernetes130, func() {

	var (
		virtClient kubecli.KubevirtClient
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		tests.EnableFeatureGate(virtconfig.NodeRestrictionGate)
	})

	It("Should disallow to modify VMs on different node", func() {
		nodes := libnode.GetAllSchedulableNodes(virtClient).Items
		if len(nodes) < 2 {
			Fail("Requires multiple nodes with virt-handler running")
		}

		vmi := libvmifact.NewAlpine()
		vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

		node := vmi.Status.NodeName

		differentNode := ""
		for _, n := range nodes {
			if node != n.Name {
				differentNode = n.Name
				break
			}
		}
		pod, err := libnode.GetVirtHandlerPod(virtClient, differentNode)
		Expect(err).ToNot(HaveOccurred())

		token, err := exec.ExecuteCommandOnPod(
			pod,
			"virt-handler",
			[]string{"cat",
				"/var/run/secrets/kubernetes.io/serviceaccount/token",
			},
		)
		Expect(err).ToNot(HaveOccurred())

		handlerClient, err := kubecli.GetKubevirtClientFromRESTConfig(&rest.Config{
			Host: virtClient.Config().Host,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
			BearerToken: string(token),
		})
		Expect(err).ToNot(HaveOccurred())

		// We cannot use patch as handler doesn't have RBAC
		// Therefore we need to use Eventually with Update
		Eventually(func(g Gomega) error {
			vmiScoped, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.TODO(), vmi.Name, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			vmiScoped.Labels["allowed.io"] = "value"
			_, err = handlerClient.VirtualMachineInstance(vmi.Namespace).Update(context.TODO(), vmiScoped, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, time.Second).Should(MatchError(
			ContainSubstring("Node restriction, virt-handler is only allowed to modify VMIs it owns"),
		))
	})

})
