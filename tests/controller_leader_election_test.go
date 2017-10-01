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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"encoding/json"
	"flag"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("LeaderElection", func() {
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("Controller pod destroyed", func() {
		It("should success to start VM", func() {
			leaderPodName := getLeader()
			Expect(virtClient.CoreV1().Pods(leaderelectionconfig.DefaultNamespace).Delete(leaderPodName, &metav1.DeleteOptions{})).To(BeNil())

			Eventually(getLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

			Eventually(func() k8sv1.ConditionStatus {
				leaderPod, err := virtClient.CoreV1().Pods(leaderelectionconfig.DefaultNamespace).Get(getLeader(), metav1.GetOptions{})
				Expect(err).To(BeNil())
				for _, condition := range leaderPod.Status.Conditions {
					if condition.Type == k8sv1.PodReady {
						return condition.Status
					}
				}
				return k8sv1.ConditionUnknown
			},
				30*time.Second,
				5*time.Second).Should(Equal(k8sv1.ConditionTrue))

			vm := tests.NewRandomVM()
			obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMStart(obj)
		}, 70)
	})
})

func getLeader() string {
	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	controllerEndpoint, err := virtClient.CoreV1().Endpoints(leaderelectionconfig.DefaultNamespace).Get(leaderelectionconfig.DefaultEndpointName, metav1.GetOptions{})
	tests.PanicOnError(err)

	var record resourcelock.LeaderElectionRecord
	if recordBytes, found := controllerEndpoint.Annotations[resourcelock.LeaderElectionRecordAnnotationKey]; found {
		err := json.Unmarshal([]byte(recordBytes), &record)
		tests.PanicOnError(err)
	}
	return record.HolderIdentity
}
