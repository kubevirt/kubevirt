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
	"flag"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("RegistryDisk", func() {

	flag.Parse()

	virtCli, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	restClient, err := kubecli.GetRESTClient()
	tests.PanicOnError(err)

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	LaunchVM := func(vm *v1.VM) runtime.Object {
		obj, err := restClient.Post().Resource("vms").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())
		return obj
	}

	VerifyRegistryDiskVM := func(vm *v1.VM, obj runtime.Object) {
		_, ok := obj.(*v1.VM)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
		tests.WaitForSuccessfulVMStart(obj)

		// Verify Registry Disks are Online
		pods, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).List(services.UnfinishedVMPodSelector(vm))
		Expect(err).To(BeNil())
		disksFound := 0
		for _, pod := range pods.Items {
			if pod.ObjectMeta.DeletionTimestamp != nil {
				continue
			}
			for _, containerStatus := range pod.Status.ContainerStatuses {
				if strings.Contains(containerStatus.Name, "disk") == false {
					// only check readiness of disk containers
					continue
				}
				if containerStatus.Ready == true {
					disksFound++
				}
			}
			break
		}
		Expect(disksFound).To(Equal(1))
	}

	Context("Ephemeral RegistryDisk", func() {
		It("should launch multiple VMs using ephemeral registry disks", func(done Done) {
			num := 5
			vms := make([]*v1.VM, 0, num)
			objs := make([]runtime.Object, 0, num)
			for i := 0; i < num; i++ {
				vm := tests.NewRandomVMWithEphemeralDisk("kubevirt/cirros-registry-disk-demo:devel")
				obj := LaunchVM(vm)
				vms = append(vms, vm)
				objs = append(objs, obj)
			}

			for idx, vm := range vms {
				VerifyRegistryDiskVM(vm, objs[idx])
			}

			close(done)
		}, 60) // Timeout is long because this test involves multiple parallel VM launches.
	})
})
