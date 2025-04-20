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

package checks

import (
	"fmt"

	"kubevirt.io/kubevirt/tests/libnode"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"
)

const (
	DescriptionTwoWorkerNodesWCPUManagerRequired = "at least two worker nodes with cpumanager are required for migration"
)

// ExpectAtLeastTwoWorkerNodesWithCPUManager uses gomega.Expect to verify that the node list returned by
// libnode.GetWorkerNodesWithCPUManagerEnabled contains at least two elements.
// DescriptionTwoWorkerNodesWCPUManagerRequired is added to the default description via ginkgo.By
func ExpectAtLeastTwoWorkerNodesWithCPUManager(virtClient kubecli.KubevirtClient) {
	ginkgo.By(fmt.Sprintf("expect at least 2 nodes - %s", DescriptionTwoWorkerNodesWCPUManagerRequired))
	_ = gomega.Expect(len(libnode.GetWorkerNodesWithCPUManagerEnabled(virtClient))).To(gomega.BeNumerically(">=", 2), DescriptionTwoWorkerNodesWCPUManagerRequired)
}
