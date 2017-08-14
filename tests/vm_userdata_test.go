/*
 * This file is part of the kubevirt project
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

var _ = Describe("CloudInit UserData", func() {

	flag.Parse()

	coreClient, err := kubecli.Get()
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

	VerifyUserDataVM := func(vm *v1.VM, obj runtime.Object) {
		_, ok := obj.(*v1.VM)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
		tests.WaitForSuccessfulVMStart(obj)

		// Verify Registry Disks are Online
		pods, err := coreClient.CoreV1().Pods(tests.NamespaceTestDefault).List(services.UnfinishedVMPodSelector(vm))
		Expect(err).To(BeNil())
		// TODO verify nocloud datasource somehow here.
		//disksFound := 0
		//for _, pod := range pods.Items {
		//	if pod.ObjectMeta.DeletionTimestamp != nil {
		//		continue
		//	}
		//	for _, containerStatus := range pod.Status.ContainerStatuses {
		//		if strings.Contains(containerStatus.Name, "cloud-init-no-cloud") == false {
		//			// only check readiness of cloud-init-no-cloud containers
		//			continue
		//		}
		//		if containerStatus.Ready == true {
		//			disksFound++
		//		}
		//	}
		//	break
		//}
		//Expect(disksFound).To(Equal(1))
	}

	Context("CloudInit Data Source NoCloud", func() {
		It("should launch multiple VMs with cloud-init data source NoCloud", func(done Done) {
			num := 2
			vms := make([]*v1.VM, 0, num)
			objs := make([]runtime.Object, 0, num)
			for i := 0; i < num; i++ {
				vm, err := tests.NewRandomVMWithUserData("kubevirt/cirros-registry-disk-demo:devel", "noCloud")
				Expect(err).ToNot(HaveOccurred())
				obj := LaunchVM(vm)
				vms = append(vms, vm)
				objs = append(objs, obj)
			}

			for idx, vm := range vms {
				VerifyUserDataVM(vm, objs[idx])
			}

			close(done)
		}, 45)
	})
})
