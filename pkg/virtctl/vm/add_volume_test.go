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

package vm_test

import (
	"context"
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	cdifake "kubevirt.io/client-go/containerizeddataimporter/fake"
	"kubevirt.io/client-go/kubecli"
	kubevirtfake "kubevirt.io/client-go/kubevirt/fake"
	kvtesting "kubevirt.io/client-go/testing"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/virtctl/testing"
)

type verifyFn func(*v1.AddVolumeOptions)

const (
	concurrentErrorAdd = "the server rejected our request due to an error in our request"
)

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
		coreClient = k8sfake.NewSimpleClientset()
		kubecli.GetKubevirtClientFromClientConfig = kubecli.GetMockKubevirtClientFromClientConfig
		kubecli.MockKubevirtClientInstance = kubecli.NewMockKubevirtClient(ctrl)
		kubecli.GetK8sClientFromClientConfig = kubecli.GetMockK8sClientFromClientConfig
		kubecli.MockK8sClientInstance = coreClient
		cdiClient = cdifake.NewSimpleClientset()
		virtClient = kubevirtfake.NewSimpleClientset()
	})

	DescribeTable("should fail with missing required or invalid parameters", func(expected string, extraArgs ...string) {
		args := append([]string{"addvolume"}, extraArgs...)
		cmd := testing.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring(expected)))
	},
		Entry("addvolume no args", "accepts 1 arg(s), received 0"),
		Entry("addvolume name, missing required volume-name", "required flag(s)", vmiName),
		Entry("addvolume name, invalid extra parameter", "unknown flag", vmiName, "--volume-name=blah", "--invalid=test"),
	)

	It("should fail when trying to add volume with invalid disk type", func() {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)

		_, err := coreClient.CoreV1().PersistentVolumeClaims(metav1.NamespaceDefault).Create(context.Background(), &k8sv1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: volumeName,
			},
		}, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		cmd := testing.NewRepeatableVirtctlCommand("addvolume", vmiName, "--volume-name="+volumeName, "--disk-type=cdrom")
		Expect(cmd()).To(MatchError(ContainSubstring("Invalid disk type")))
	})

	DescribeTable("should fail addvolume when no source is found according to option", func(dryRun bool) {
		kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)

		args := []string{"addvolume", vmiName, "--volume-name=" + volumeName}
		if dryRun {
			args = append(args, "--dry-run")
		}
		cmd := testing.NewRepeatableVirtctlCommand(args...)
		Expect(cmd()).To(MatchError(ContainSubstring("Volume " + volumeName + " is not a DataVolume or PersistentVolumeClaim")))
	},
		Entry("with default", false),
		Entry("with dry-run arg", true),
	)

	Context("addvolume cmd", func() {
		expectVMIEndpointAddVolume := func(verifyFns ...verifyFn) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachineInstance(metav1.NamespaceDefault).
				Return(virtClient.KubevirtV1().VirtualMachineInstances(metav1.NamespaceDefault)).
				Times(1)
			virtClient.PrependReactor("put", "virtualmachineinstances/addvolume", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
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

		expectVMEndpointAddVolumeErrorFunc := func(errFunc func() error, verifyFns ...verifyFn) {
			kubecli.MockKubevirtClientInstance.
				EXPECT().
				VirtualMachine(metav1.NamespaceDefault).
				Return(virtClient.KubevirtV1().VirtualMachines(metav1.NamespaceDefault)).
				AnyTimes()
			virtClient.PrependReactor("put", "virtualmachines/addvolume", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				switch action := action.(type) {
				case kvtesting.PutAction[*v1.AddVolumeOptions]:
					volumeOptions := action.GetOptions()
					Expect(volumeOptions).ToNot(BeNil())
					Expect(volumeOptions.Name).To(Equal(volumeName))
					Expect(volumeOptions.VolumeSource).ToNot(BeNil())
					for _, verifyFn := range verifyFns {
						verifyFn(volumeOptions)
					}
					if errFunc == nil {
						return true, nil, nil
					}
					return true, nil, errFunc()
				default:
					Fail("unexpected action type on addvolume")
					return false, nil, nil
				}
			})
		}

		expectVMEndpointAddVolume := func(verifyFns ...verifyFn) {
			expectVMEndpointAddVolumeErrorFunc(nil, verifyFns...)
		}

		runCmd := func(persist bool, extraArg string) error {
			args := []string{"addvolume", vmiName, "--volume-name=" + volumeName}
			if persist {
				args = append(args, "--persist")
			}
			args = append(args, strings.Fields(extraArg)...)
			cmd := testing.NewRepeatableVirtctlCommand(args...)
			return cmd()
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
				expectVMIEndpointAddVolume(verifyFns...)
				Expect(runCmd(false, arg)).To(Succeed())
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
				Entry("virtio bus", "--bus=virtio", verifyDiskSerial(volumeName), verifyBus(v1.DiskBusVirtio)),
			)

			DescribeTable("should call VM endpoint with persist and", func(arg string, verifyFns ...verifyFn) {
				verifyFns = append(verifyFns, verifyDVVolumeSource)
				expectVMEndpointAddVolume(verifyFns...)
				Expect(runCmd(true, arg)).To(Succeed())
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
				Entry("virtio bus", "--bus=virtio", verifyDiskSerial(volumeName), verifyBus(v1.DiskBusVirtio)),
			)

			It("should fail immediately on non concurrent error", func() {
				expectVMEndpointAddVolumeErrorFunc(func() error { return fmt.Errorf("fatal error") }, verifyDVVolumeSource)
				Expect(runCmd(true, "")).To(MatchError(ContainSubstring("fatal error")))
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "addvolume")).To(HaveLen(1))
			})

			It("should retry on error", func() {
				count := 0
				expectVMEndpointAddVolumeErrorFunc(func() error {
					if count == 0 {
						count++
						return errors.New(concurrentErrorAdd)
					} else {
						return nil
					}
				}, verifyDVVolumeSource)
				Expect(runCmd(true, "")).To(Succeed())
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "addvolume")).To(HaveLen(2))
			})

			It("should fail after 15 retries", func() {
				expectVMEndpointAddVolumeErrorFunc(func() error {
					return errors.New(concurrentErrorAdd)
				}, verifyDVVolumeSource)
				Expect(runCmd(true, "")).To(MatchError(ContainSubstring("error adding volume after 15 retries")))
				Expect(kvtesting.FilterActions(&virtClient.Fake, "put", "virtualmachines", "addvolume")).To(HaveLen(15))
			})

			DescribeTable("should fail addvolume with LUN and virtio bus", func(persist bool) {
				Expect(runCmd(persist, "--disk-type=lun --bus=virtio")).To(
					MatchError(ContainSubstring("Invalid bus type 'virtio' for LUN disk. Only 'scsi' bus is supported.")))
			},
				Entry("without persist", false),
				Entry("with persist", true),
			)
		})

		Context("with PVC", func() {
			BeforeEach(func() {
				kubecli.MockKubevirtClientInstance.EXPECT().CdiClient().Return(cdiClient)
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
				expectVMIEndpointAddVolume(verifyFns...)
				Expect(runCmd(false, arg)).To(Succeed())
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
				Entry("virtio bus", "--bus=virtio", verifyDiskSerial(volumeName), verifyBus(v1.DiskBusVirtio)),
			)

			DescribeTable("should call VM endpoint with persist and", func(arg string, verifyFns ...verifyFn) {
				verifyFns = append(verifyFns, verifyPVCVolumeSource)
				expectVMEndpointAddVolume(verifyFns...)
				Expect(runCmd(true, arg)).To(Succeed())
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
				Entry("virtio bus", "--bus=virtio", verifyDiskSerial(volumeName), verifyBus(v1.DiskBusVirtio)),
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
	Expect(volumeOptions.Disk.DiskDevice.Disk.Bus).To(Or(Equal(v1.DiskBusSCSI), Equal(v1.DiskBusVirtio)))
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

func verifyBus(bus v1.DiskBus) verifyFn {
	return func(volumeOptions *v1.AddVolumeOptions) {
		Expect(volumeOptions.Disk.Disk.Bus).To(Equal(bus))
	}
}
