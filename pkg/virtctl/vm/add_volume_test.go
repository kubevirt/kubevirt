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

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/libdv"
)

const (
	vmiName    = "testvmi"
	volumeName = "testvolume"
)

type VerifyFunc func(*v1.AddVolumeOptions)

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

	expectVMIEndpointAddVolume := func(vmiName, volumeName string, verifyFunc VerifyFunc) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachineInstance(k8smetav1.NamespaceDefault).
			Return(vmiInterface).
			Times(1)
		vmiInterface.EXPECT().AddVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			volumeOptions, ok := arg1.(*v1.AddVolumeOptions)
			Expect(ok).To(BeTrue())
			Expect(volumeOptions).ToNot(BeNil())
			Expect(volumeOptions.Name).To(Equal(volumeName))
			verifyFunc(volumeOptions)
			return nil
		})
	}

	expectVMEndpointAddVolume := func(vmiName, volumeName string, verifyFunc VerifyFunc) {
		kubecli.MockKubevirtClientInstance.
			EXPECT().
			VirtualMachine(k8smetav1.NamespaceDefault).
			Return(vmInterface).
			Times(1)
		vmInterface.EXPECT().AddVolume(context.Background(), vmiName, gomock.Any()).DoAndReturn(func(ctx context.Context, arg0, arg1 interface{}) interface{} {
			volumeOptions, ok := arg1.(*v1.AddVolumeOptions)
			Expect(ok).To(BeTrue())
			Expect(volumeOptions).ToNot(BeNil())
			Expect(volumeOptions.Name).To(Equal(volumeName))
			verifyFunc(volumeOptions)
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
		Entry("addvolume name, missing required volume-name", "addvolume", "required flag(s)", vmiName),
		Entry("addvolume name, invalid extra parameter", "addvolume", "unknown flag", vmiName, "--volume-name=blah", "--invalid=test"),
	)

	It("should fail when trying to add volume with invalid disk type", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
		coreClient.CoreV1().PersistentVolumeClaims(k8smetav1.NamespaceDefault).Create(context.Background(), createTestPVC(volumeName), k8smetav1.CreateOptions{})
		commandAndArgs := []string{"addvolume", vmiName, "--volume-name=" + volumeName, "--disk-type=cdrom"}

		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Invalid disk type"))
	})

	DescribeTable("should fail addvolume when no source is found according to option", func(isDryRun bool) {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
		commandAndArgs := []string{"addvolume", vmiName, "--volume-name=" + volumeName}

		if isDryRun {
			commandAndArgs = append(commandAndArgs, "--dry-run")
		}
		cmdAdd := clientcmd.NewRepeatableVirtctlCommand(commandAndArgs...)
		err := cmdAdd()

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Volume " + volumeName + " is not a DataVolume or PersistentVolumeClaim"))
	},
		Entry("with default", false),
		Entry("with dry-run arg", true),
	)

	DescribeTable("with DataVolume addvolume cmd, should call correct endpoint", func(expectFunc func(vmiName, volumeName string, verifyFunc VerifyFunc), args ...string) {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		cdiClient.CdiV1beta1().DataVolumes(k8smetav1.NamespaceDefault).Create(
			context.Background(),
			libdv.NewDataVolume(libdv.WithName(volumeName)),
			k8smetav1.CreateOptions{})

		verifyDVVolumeSource := func(volumeOptions *v1.AddVolumeOptions) {
			Expect(volumeOptions.VolumeSource).ToNot(BeNil())
			Expect(volumeOptions.VolumeSource.DataVolume).ToNot(BeNil())
			Expect(volumeOptions.VolumeSource.PersistentVolumeClaim).To(BeNil())
		}

		expectFunc(vmiName, volumeName, verifyDVVolumeSource)
		commandAndArgs := append([]string{"addvolume", vmiName, fmt.Sprintf("--volume-name=%s", volumeName)}, args...)
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)

		Expect(cmd.Execute()).To(Succeed())
	},
		Entry("no persist should call VMI endpoint", expectVMIEndpointAddVolume),
		Entry("with persist should call VM endpoint", expectVMEndpointAddVolume, "--persist"),
		Entry("no persist with dry-run should call VMI endpoint", expectVMIEndpointAddVolume, "--dry-run"),
		Entry("with persist with dry-run should call VM endpoint", expectVMEndpointAddVolume, "--persist", "--dry-run"),
		Entry("with persist and LUN-type disk should call VM endpoint", expectVMEndpointAddVolume, "--persist", "--disk-type", "lun"),
		Entry("with LUN-type disk should call VMI endpoint", expectVMIEndpointAddVolume, "--disk-type", "lun"),
	)

	DescribeTable("with PVC addvolume cmd, should call correct endpoint", func(expectFunc func(vmiName, volumeName string, verifyFunc VerifyFunc), args ...string) {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
		coreClient.CoreV1().PersistentVolumeClaims(k8smetav1.NamespaceDefault).Create(
			context.Background(),
			createTestPVC(volumeName),
			k8smetav1.CreateOptions{})

		verifyPVCVolumeSource := func(volumeOptions *v1.AddVolumeOptions) {
			Expect(volumeOptions.VolumeSource).ToNot(BeNil())
			Expect(volumeOptions.VolumeSource.DataVolume).To(BeNil())
			Expect(volumeOptions.VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
		}

		expectFunc(vmiName, volumeName, verifyPVCVolumeSource)
		commandAndArgs := append([]string{"addvolume", vmiName, fmt.Sprintf("--volume-name=%s", volumeName)}, args...)
		cmd := clientcmd.NewVirtctlCommand(commandAndArgs...)

		Expect(cmd.Execute()).To(Succeed())
	},
		Entry("no persist should call VMI endpoint", expectVMIEndpointAddVolume),
		Entry("with persist should call VM endpoint", expectVMEndpointAddVolume, "--persist"),
		Entry("no persist with dry-run should call VMI endpoint", expectVMIEndpointAddVolume, "--dry-run"),
		Entry("with persist with dry-run should call VM endpoint", expectVMEndpointAddVolume, "--persist", "--dry-run"),
		Entry("with persist and LUN-type disk should call VM endpoint", expectVMEndpointAddVolume, "--persist", "--disk-type", "lun"),
		Entry("with LUN-type disk should call VMI endpoint", expectVMIEndpointAddVolume, "--disk-type", "lun"),
	)
})

func createTestPVC(name string) *k8sv1.PersistentVolumeClaim {
	return &k8sv1.PersistentVolumeClaim{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: name,
		},
	}
}
