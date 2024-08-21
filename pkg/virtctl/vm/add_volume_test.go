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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/golang/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	kvtesting "kubevirt.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/generated/containerized-data-importer/clientset/versioned/fake"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/clientcmd"
)

type verifyFn func(*v1.AddVolumeOptions)

var _ = Describe("Add volume command", func() {
	const (
		vmiName    = "testvmi"
		volumeName = "testvolume"
	)

	var cdiClient *cdifake.Clientset
	var coreClient *k8sfake.Clientset
	var virtClient *kubevirtfake.Clientset

	BeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		cdiClient = cdifake.NewSimpleClientset()
		coreClient = k8sfake.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()
	})

	DescribeTable("should fail with missing required or invalid parameters", func(expected string, extraArgs ...string) {
		args := append([]string{"addvolume"}, extraArgs...)
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring(expected)))
	},
		Entry("addvolume no args", "accepts 1 arg(s), received 0"),
		Entry("addvolume name, missing required volume-name", "required flag(s)", vmiName),
		Entry("addvolume name, invalid extra parameter", "unknown flag", vmiName, "--volume-name=blah", "--invalid=test"),
	)

	It("should fail when trying to add volume with invalid disk type", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())

		_, err := coreClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		cmd := clientcmd.NewRepeatableVirtctlCommand("addvolume", vmiName, "--volume-name="+volumeName, "--disk-type=cdrom")
		Expect(cmd()).To(MatchError(ContainSubstring("Invalid disk type")))
	})

	DescribeTable("should fail addvolume when no source is found according to option", func(dryRun bool) {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
		kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())

		args := []string{"addvolume", vmiName, "--volume-name=" + volumeName}
		if dryRun {
			args = append(args, "--dry-run")
		}
		cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring("Volume " + volumeName + " is not a DataVolume or PersistentVolumeClaim")))
	},
		Entry("with default", false),
		Entry("with dry-run arg", true),
	)

	Context("addvolume cmd", func() {
		expectVMIEndpointAddVolume := func(verifyFns []verifyFn) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(metav1.NamespaceDefault).
				Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).
				Times(1)
			virtClient.PrependReactor("put", "virtualmachineinstances/addvolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				switch action := action.(type) {
				case kvtesting.PutAction[*v1.AddVolumeOptions]:
					volumeOptions := action.GetOptions()
					Expect(volumeOptions).ToNot(BeNil())
					Expect(volumeOptions.Name).To(Equal(volumeName))
					Expect(volumeOptions.VolumeSource).ToNot(BeNil())
					for _, verifyFn := range verifyFns {
						verifyFn(volumeOptions)
					}
					return true, nil, nil
				default:
					Fail("unexpected action type on addvolume")
					return false, nil, nil
				}
			})
		}

		expectVMEndpointAddVolume := func(verifyFns []verifyFn) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(metav1.NamespaceDefault).
				Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).
				Times(1)
			virtClient.PrependReactor("put", "virtualmachines/addvolume", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
				switch action := action.(type) {
				case kvtesting.PutAction[*v1.AddVolumeOptions]:
					volumeOptions := action.GetOptions()
					Expect(volumeOptions).ToNot(BeNil())
					Expect(volumeOptions.Name).To(Equal(volumeName))
					Expect(volumeOptions.VolumeSource).ToNot(BeNil())
					for _, verifyFn := range verifyFns {
						verifyFn(volumeOptions)
					}
					return true, nil, nil
				default:
					Fail("unexpected action type on addvolume")
					return false, nil, nil
				}
			})
		}

		runCmd := func(persist bool, extraArg string) {
			args := []string{"addvolume", vmiName, "--volume-name=" + volumeName}
			if persist {
				args = append(args, "--persist")
			}
			if extraArg != "" {
				args = append(args, extraArg)
			}
			cmd := clientcmd.NewRepeatableVirtctlCommand(args...)
			Expect(cmd()).To(Succeed())
		}

		Context("with DataVolume", func() {
			BeforeEach(func() {
				kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
				_, err := cdiClient.CdiV1beta1().DataVolumes(metav1.NamespaceDefault).Create(
					context.Background(),
					&v1beta1.DataVolume{
						ObjectMeta: metav1.ObjectMeta{
							Name: volumeName,
						},
					},
					metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("should call VMI endpoint without persist and with", func(arg string, verifyFns ...verifyFn) {
				verifyFns = append(verifyFns, verifyDVVolumeSource)
				expectVMIEndpointAddVolume(verifyFns)
				runCmd(false, arg)
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachineinstances", "addvolume")).To(HaveLen(1))
			},
				Entry("no args", "", verifyDiskSerial(volumeName)),
				Entry("dry-run", "--dry-run", verifyDiskSerial(volumeName), verifyDryRun),
				Entry("disk-type disk", "--disk-type=disk", verifyDiskSerial(volumeName), verifyDiskTypeDisk),
				Entry("disk-type lun", "--disk-type=lun", verifyDiskSerial(volumeName), verifyDiskTypeLun),
				Entry("serial", "--serial=test", verifyDiskSerial("test")),
				Entry("cache none", "--cache=none", verifyDiskSerial(volumeName), verifyCache(v1.CacheNone)),
				Entry("cache writethrough", "--cache=writethrough", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteThrough)),
				Entry("cache writeback", "--cache=writeback", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteBack)),
			)

			DescribeTable("should call VM endpoint with persist and", func(arg string, verifyFns ...verifyFn) {
				verifyFns = append(verifyFns, verifyDVVolumeSource)
				expectVMEndpointAddVolume(verifyFns)
				runCmd(true, arg)
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "addvolume")).To(HaveLen(1))
			},
				Entry("no args", "", verifyDiskSerial(volumeName)),
				Entry("dry-run", "--dry-run", verifyDiskSerial(volumeName), verifyDryRun),
				Entry("disk-type disk", "--disk-type=disk", verifyDiskSerial(volumeName), verifyDiskTypeDisk),
				Entry("disk-type lun", "--disk-type=lun", verifyDiskSerial(volumeName), verifyDiskTypeLun),
				Entry("serial", "--serial=test", verifyDiskSerial("test")),
				Entry("cache none", "--cache=none", verifyDiskSerial(volumeName), verifyCache(v1.CacheNone)),
				Entry("cache writethrough", "--cache=writethrough", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteThrough)),
				Entry("cache writeback", "--cache=writeback", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteBack)),
			)
		})

		Context("with PVC", func() {
			BeforeEach(func() {
				kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
				kubecli.MockKubevirtClientInstance.EXPECT().CoreV1().Return(coreClient.CoreV1())
				_, err := coreClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(
					context.Background(),
					&k8sv1.PersistentVolumeClaim{
						ObjectMeta: metav1.ObjectMeta{
							Name: volumeName,
						},
					},
					metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			DescribeTable("should call VMI endpoint without persist and with", func(arg string, verifyFns ...verifyFn) {
				verifyFns = append(verifyFns, verifyPVCVolumeSource)
				expectVMIEndpointAddVolume(verifyFns)
				runCmd(false, arg)
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachineinstances", "addvolume")).To(HaveLen(1))
			},
				Entry("no args", "", verifyDiskSerial(volumeName)),
				Entry("dry-run", "--dry-run", verifyDiskSerial(volumeName), verifyDryRun),
				Entry("disk-type disk", "--disk-type=disk", verifyDiskSerial(volumeName), verifyDiskTypeDisk),
				Entry("disk-type lun", "--disk-type=lun", verifyDiskSerial(volumeName), verifyDiskTypeLun),
				Entry("serial", "--serial=test", verifyDiskSerial("test")),
				Entry("cache none", "--cache=none", verifyDiskSerial(volumeName), verifyCache(v1.CacheNone)),
				Entry("cache writethrough", "--cache=writethrough", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteThrough)),
				Entry("cache writeback", "--cache=writeback", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteBack)),
			)

			DescribeTable("should call VM endpoint with persist and", func(arg string, verifyFns ...verifyFn) {
				verifyFns = append(verifyFns, verifyPVCVolumeSource)
				expectVMEndpointAddVolume(verifyFns)
				runCmd(true, arg)
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "addvolume")).To(HaveLen(1))
			},
				Entry("no args", "", verifyDiskSerial(volumeName)),
				Entry("dry-run", "--dry-run", verifyDiskSerial(volumeName), verifyDryRun),
				Entry("disk-type disk", "--disk-type=disk", verifyDiskSerial(volumeName), verifyDiskTypeDisk),
				Entry("disk-type lun", "--disk-type=lun", verifyDiskSerial(volumeName), verifyDiskTypeLun),
				Entry("serial", "--serial=test", verifyDiskSerial("test")),
				Entry("cache none", "--cache=none", verifyDiskSerial(volumeName), verifyCache(v1.CacheNone)),
				Entry("cache writethrough", "--cache=writethrough", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteThrough)),
				Entry("cache writeback", "--cache=writeback", verifyDiskSerial(volumeName), verifyCache(v1.CacheWriteBack)),
			)
		})
	})
})

func verifyDVVolumeSource(volumeOptions *v1.AddVolumeOptions) {
	Expect(volumeOptions.VolumeSource.DataVolume).ToNot(BeNil())
	Expect(volumeOptions.VolumeSource.PersistentVolumeClaim).To(BeNil())
}

func verifyPVCVolumeSource(volumeOptions *v1.AddVolumeOptions) {
	Expect(volumeOptions.VolumeSource.DataVolume).To(BeNil())
	Expect(volumeOptions.VolumeSource.PersistentVolumeClaim).ToNot(BeNil())
}

func verifyDryRun(volumeOptions *v1.AddVolumeOptions) {
	Expect(volumeOptions.DryRun).To(Equal([]string{metav1.DryRunAll}))
}

func verifyDiskTypeDisk(volumeOptions *v1.AddVolumeOptions) {
	Expect(volumeOptions.Disk.DiskDevice.Disk).ToNot(BeNil())
	Expect(volumeOptions.Disk.DiskDevice.Disk.Bus).To(Equal(v1.DiskBusSCSI))
	Expect(volumeOptions.Disk.DiskDevice.LUN).To(BeNil())
}

func verifyDiskTypeLun(volumeOptions *v1.AddVolumeOptions) {
	Expect(volumeOptions.Disk.DiskDevice.Disk).To(BeNil())
	Expect(volumeOptions.Disk.DiskDevice.LUN).ToNot(BeNil())
	Expect(volumeOptions.Disk.DiskDevice.LUN.Bus).To(Equal(v1.DiskBusSCSI))
}

func verifyDiskSerial(serial string) verifyFn {
	return func(volumeOptions *v1.AddVolumeOptions) {
		Expect(volumeOptions.Disk.Serial).To(Equal(serial))
	}
}

func verifyCache(cache v1.DriverCache) verifyFn {
	return func(volumeOptions *v1.AddVolumeOptions) {
		Expect(volumeOptions.Disk.Cache).To(Equal(cache))
	}
}
