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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package libinfra

import (
	"github.com/onsi/ginkgo/v2"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

func WakeNodeLabellerUp(virtClient kubecli.KubevirtClient) {
	const fakeModel = "fake-model-1423"

	ginkgo.By("Updating Kubevirt CR to wake node-labeller up")
	kvConfig := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
	if kvConfig.ObsoleteCPUModels == nil {
		kvConfig.ObsoleteCPUModels = make(map[string]bool)
	}
	kvConfig.ObsoleteCPUModels[fakeModel] = true
	tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
	delete(kvConfig.ObsoleteCPUModels, fakeModel)
	tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
}
