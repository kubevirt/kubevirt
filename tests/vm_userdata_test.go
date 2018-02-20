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
	"fmt"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("CloudInit UserData", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	LaunchVM := func(vm *v1.VirtualMachine) runtime.Object {
		By("Starting a VM")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachines").Namespace(tests.NamespaceTestDefault).Body(vm).Do().Get()
		Expect(err).To(BeNil())
		return obj
	}

	VerifyUserDataVM := func(vm *v1.VirtualMachine, obj runtime.Object, magicStr string) {
		_, ok := obj.(*v1.VirtualMachine)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VM")
		tests.WaitForSuccessfulVMStart(obj)

		By("Expecting the VM console")
		expecter, _, err := tests.NewConsoleExpecter(virtClient, vm, "serial0", 10*time.Second)
		defer expecter.Close()
		Expect(err).ToNot(HaveOccurred())

		By("Checking that the console output equals to expected one")
		_, err = expecter.ExpectBatch([]expect.Batcher{
			&expect.BExp{R: magicStr},
		}, 120*time.Second)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("A new VM", func() {
		Context("with cloudInitNoCloud source", func() {
			It("should have cloud-init data", func(done Done) {
				magicStr := "printed from cloud-init userdata"
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", magicStr)

				vm := tests.NewRandomVMWithEphemeralDiskAndUserdata("kubevirt/cirros-registry-disk-demo:devel", userData)
				obj := LaunchVM(vm)
				VerifyUserDataVM(vm, obj, magicStr)
				close(done)
			}, 180)
		})

		It("should take user-data from k8s secret", func(done Done) {
			magicStr := "printed from cloud-init userdata"
			userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", magicStr)
			vm := tests.NewRandomVMWithEphemeralDiskAndUserdata("kubevirt/cirros-registry-disk-demo:devel", userData)

			for _, volume := range vm.Spec.Volumes {
				if volume.CloudInitNoCloud == nil {
					continue
				}

				secretID := fmt.Sprintf("%s-test-secret", vm.Name)
				spec := volume.CloudInitNoCloud
				spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}
				userData64 := spec.UserDataBase64
				spec.UserDataBase64 = ""

				// Store userdata as k8s secret
				By("Creating a user-data secret")
				secret := kubev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretID,
						Namespace: vm.GetObjectMeta().GetNamespace(),
					},
					Type: "Opaque",
					Data: map[string][]byte{
						"userdata": []byte(userData64),
					},
				}
				_, err := virtClient.CoreV1().Secrets(vm.GetObjectMeta().GetNamespace()).Create(&secret)
				Expect(err).To(BeNil())
				break
			}
			obj := LaunchVM(vm)
			VerifyUserDataVM(vm, obj, magicStr)

			close(done)
		}, 180)
	})
})
