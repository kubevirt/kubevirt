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

package vm_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

var _ = Describe("Add volume command", func() {
	var vmInterface *kubecli.MockVirtualMachineInterface
	var vmiInterface *kubecli.MockVirtualMachineInstanceInterface
	var ctrl *gomock.Controller

	var cdiClient *cdifake.Clientset
	var coreClient *fake.Clientset

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		vmInterface = kubecli.NewMockVirtualMachineInterface(ctrl)
		vmiInterface = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		cdiClient = cdifake.NewSimpleClientset()
		coreClient = fake.NewSimpleClientset()
	})

	createTestDataVolume := func() *v1beta1.DataVolume {
		return &v1beta1.DataVolume{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "testvolume",
			},
		}
	}

	createTestPVC := func() *k8sv1.PersistentVolumeClaim {
		return &k8sv1.PersistentVolumeClaim{
			ObjectMeta: k8smetav1.ObjectMeta{
				Name: "testvolume",
			},
		}
	}

	verifyVolumeSource := func(volumeOptions interface{}, useDv bool) {
		if useDv {
			Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.DataVolume).ToNot(BeNil())
			Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.PersistentVolumeClaim).To(BeNil())
		} else {
			Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.DataVolume).To(BeNil())
			Expect(volumeOptions.(*v1.AddVolumeOptions).VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
		}
	}

	expectVMIEndpointAddVolume := func(vmiName, volumeName string, useDv bool) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().AddVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.AddVolumeOptions).Name).To(Equal(volumeName))
			verifyVolumeSource(arg1, useDv)
			return nil
		})
	}

	expectVMEndpointAddVolume := func(vmiName, volumeName string, useDv bool) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().AddVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			Expect(arg1.(*v1.AddVolumeOptions).Name).To(Equal(volumeName))
			verifyVolumeSource(arg1, useDv)
			return nil
		})
	}

	DescribeTable("should fail with missing required or invalid parameters", func(commandName, errorString string, args ...string) {
		commandAndArgs := append([]string{commandName}, args...)
		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(errorString))
	},
		Entry("addvolume no args", "addvolume", "argument validation failed"),
		Entry("addvolume name, missing required volume-name", "addvolume", "required flag(s)", "testvmi"),
		Entry("addvolume name, invalid extra parameter", "addvolume", "unknown flag", "testvmi", "--volume-name=blah", "--invalid=test"),
	)

	It("should fail when trying to add volume with invalid disk type", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
		coreClient.CoreV1().PersistentVolumeClaims(k8smetav1.NamespaceDefault).Create(context.Background(), createTestPVC(), k8smetav1.CreateOptions{})
		commandAndArgs := []string{"addvolume", "testvmi", "--volume-name=testvolume", "--disk-type=cdrom"}

		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Invalid disk type"))
	})

	DescribeTable("should fail addvolume when no source is found according to option", func(isDryRun bool) {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
		commandAndArgs := []string{"addvolume", "testvmi", "--volume-name=testvolume"}

		if isDryRun {
			commandAndArgs = append(commandAndArgs, "--dry-run")
		}
		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Volume testvolume is not a DataVolume or PersistentVolumeClaim"))
	},
		Entry("with default", false),
		Entry("with dry-run arg", true),
	)

	DescribeTable("should call correct endpoint", func(commandName, vmiName, volumeName string, useDv bool, expectFunc func(vmiName, volumeName string, useDv bool), args ...string) {
		if commandName == "addvolume" {
			kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
			if useDv {
				cdiClient.CdiV1beta1().DataVolumes(k8smetav1.NamespaceDefault).Create(context.Background(), createTestDataVolume(), k8smetav1.CreateOptions{})
			} else {
				kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
				coreClient.CoreV1().PersistentVolumeClaims(k8smetav1.NamespaceDefault).Create(context.Background(), createTestPVC(), k8smetav1.CreateOptions{})
			}
		}
		expectFunc(vmiName, volumeName, useDv)
		commandAndArgs := append([]string{commandName, vmiName, fmt.Sprintf("--volume-name=%s", volumeName)}, args...)
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)

		Expect(cmd.Execute()).To(Succeed())
	},
		Entry("addvolume dv, no persist should call VMI endpoint", "addvolume", "testvmi", "testvolume", true, expectVMIEndpointAddVolume),
		Entry("addvolume pvc, no persist should call VMI endpoint", "addvolume", "testvmi", "testvolume", false, expectVMIEndpointAddVolume),
		Entry("addvolume dv, with persist should call VM endpoint", "addvolume", "testvmi", "testvolume", true, expectVMEndpointAddVolume, "--persist"),
		Entry("addvolume pvc, with persist should call VM endpoint", "addvolume", "testvmi", "testvolume", false, expectVMEndpointAddVolume, "--persist"),

		Entry("addvolume dv, no persist with dry-run should call VMI endpoint", "addvolume", "testvmi", "testvolume", true, expectVMIEndpointAddVolume, "--dry-run"),
		Entry("addvolume pvc, no persist with dry-run should call VMI endpoint", "addvolume", "testvmi", "testvolume", false, expectVMIEndpointAddVolume, "--dry-run"),
		Entry("addvolume dv, with persist with dry-run should call VM endpoint", "addvolume", "testvmi", "testvolume", true, expectVMEndpointAddVolume, "--persist", "--dry-run"),
		Entry("addvolume pvc, with persist with dry-run should call VM endpoint", "addvolume", "testvmi", "testvolume", false, expectVMEndpointAddVolume, "--persist", "--dry-run"),

		Entry("addvolume pvc, with persist and LUN-type disk should call VM endpoint", "addvolume", "testvmi", "testvolume", false, expectVMEndpointAddVolume, "--persist", "--disk-type", "lun"),
		Entry("addvolume dv, with persist and LUN-type disk should call VM endpoint", "addvolume", "testvmi", "testvolume", true, expectVMEndpointAddVolume, "--persist", "--disk-type", "lun"),
		Entry("addvolume pvc, with LUN-type disk should call VMI endpoint", "addvolume", "testvmi", "testvolume", false, expectVMIEndpointAddVolume, "--disk-type", "lun"),
		Entry("addvolume dv, with LUN-type disk should call VMI endpoint", "addvolume", "testvmi", "testvolume", true, expectVMIEndpointAddVolume, "--disk-type", "lun"),
	)
})
