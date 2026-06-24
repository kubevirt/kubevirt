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
 * Copyright the KubeVirt Authors.
 *
 */

package checks

import (
	"context"
	"fmt"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libnode"
)

const cpuManagerNodeWaitTimeout = 360 * time.Second

// EnforceDecoratedCPUManagerRequirements waits for virt-handler to label worker nodes
// when the current spec carries a CPU manager requirement decorator.
func EnforceDecoratedCPUManagerRequirements(virtClient kubecli.KubevirtClient) {
	switch {
	case specMatchesLabel(decorators.RequiresTwoWorkerNodesWithCPUManager):
		waitForWorkerNodesWithCPUManager(virtClient, 2, "at least two worker nodes with cpumanager are required for migration")
	case specMatchesLabel(decorators.RequiresNodeWithCPUManager):
		waitForWorkerNodesWithCPUManager(virtClient, 1, "at least one worker node with cpumanager is required")
	}
}

func specMatchesLabel(label ginkgo.Labels) bool {
	if len(label) == 0 {
		return false
	}
	matched, err := ginkgo.CurrentSpecReport().MatchesLabelFilter(label[0])
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("failed to match label filter %q", label[0]))
	return matched
}

func waitForWorkerNodesWithCPUManager(virtClient kubecli.KubevirtClient, minNodes int, description string) {
	ginkgo.By(fmt.Sprintf("expect at least %d worker nodes with cpumanager - %s", minNodes, description))

	workerNodes, err := virtClient.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	if len(workerNodes.Items) < minNodes {
		ginkgo.Fail(fmt.Sprintf("not enough worker nodes: need at least %d to run this test but cluster has %d", minNodes, len(workerNodes.Items)))
	}

	gomega.Eventually(func() int {
		return len(libnode.GetWorkerNodesWithCPUManagerEnabled(virtClient))
	}, cpuManagerNodeWaitTimeout, time.Second).Should(gomega.BeNumerically(">=", minNodes), description)
}
