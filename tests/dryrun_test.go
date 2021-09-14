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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package tests_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute]VMI Dry-Run requests", func() {
	var err error
	var virtClient kubecli.KubevirtClient
	var restClient *rest.RESTClient
	var vmi *v1.VirtualMachineInstance

	resource := "virtualmachineinstances"

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		restClient = virtClient.RestClient()

		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	})

	It("create a VirtualMachineInstance", func() {
		By("Make a Dry-Run request to create a Virtual Machine")
		err = tests.DryRunCreate(restClient, resource, vmi.Namespace, vmi, nil)
		Expect(err).To(BeNil())

		By("Check that no Virtual Machine was actually created")
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("delete a VirtualMachineInstance", func() {
		By("Create a VirtualMachineInstance")
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
		Expect(err).To(BeNil())

		By("Make a Dry-Run request to delete a Virtual Machine")
		deletePolicy := metav1.DeletePropagationForeground
		opts := metav1.DeleteOptions{
			DryRun:            []string{metav1.DryRunAll},
			PropagationPolicy: &deletePolicy,
		}
		err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &opts)
		Expect(err).To(BeNil())

		By("Check that no Virtual Machine was actually deleted")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	It("update a VirtualMachineInstance", func() {
		By("Create a VirtualMachineInstance")
		_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
		Expect(err).To(BeNil())

		By("Make a Dry-Run request to update a Virtual Machine")
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			if err != nil {
				return err
			}
			vmi.Labels = map[string]string{
				"key": "42",
			}
			return tests.DryRunUpdate(restClient, resource, vmi.Name, vmi.Namespace, vmi, nil)
		})

		By("Check that no update actually took place")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Labels["key"]).ToNot(Equal("42"))
	})

	It("patch a VirtualMachineInstance", func() {
		By("Create a VirtualMachineInstance")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
		Expect(err).To(BeNil())

		By("Make a Dry-Run request to patch a Virtual Machine")
		patch := []byte(`{"metadata": {"labels": {"key": "42"}}}`)
		err = tests.DryRunPatch(restClient, resource, vmi.Name, vmi.Namespace, types.MergePatchType, patch, nil)
		Expect(err).To(BeNil())

		By("Check that no update actually took place")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Labels["key"]).ToNot(Equal("42"))
	})
})
