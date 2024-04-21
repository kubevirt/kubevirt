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

package tests

import (
	"archive/tar"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"

	util2 "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kutil "kubevirt.io/kubevirt/pkg/util"
	launcherApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

const (
	BinBash                = "/bin/bash"
	waitingVMInstanceStart = "Waiting until the VirtualMachineInstance will start"
	EchoLastReturnValue    = "echo $?\n"
	CustomHostPath         = "custom-host-path"
	DiskAlpineHostPath     = "disk-alpine-host-path"
	DiskWindowsSysprep     = "disk-windows-sysprep"
	DiskCustomHostPath     = "disk-custom-host-path"
)

func CreateSecret(name, namespace string, data map[string]string) {
	virtCli := kubevirt.Client()

	_, err := virtCli.CoreV1().Secrets(namespace).Create(context.Background(), &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		StringData: data,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util2.PanicOnError(err)
	}
}

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By(waitingVMInstanceStart)
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
		libwait.WithTimeout(timeout),
	)
}

func RunVMIAndExpectLaunchWithDataVolume(vmi *v1.VirtualMachineInstance, dv *cdiv1.DataVolume, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By("Waiting until the DataVolume is ready")
	libstorage.EventuallyDV(dv, timeout, HaveSucceeded())
	By(waitingVMInstanceStart)
	warningsIgnoreList := []string{"didn't find PVC", "unable to find datavolume"}
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
		libwait.WithWarningsIgnoreList(warningsIgnoreList),
		libwait.WithTimeout(timeout),
	)
}

func RunVMIAndExpectLaunchIgnoreWarnings(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By(waitingVMInstanceStart)
	return libwait.WaitForSuccessfulVMIStart(vmi,
		libwait.WithFailOnWarnings(false),
		libwait.WithTimeout(timeout),
	)
}

func RunVMIAndExpectScheduling(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return RunVMIAndExpectSchedulingWithWarningPolicy(vmi, timeout, wp)
}

func RunVMIAndExpectSchedulingWithWarningPolicy(vmi *v1.VirtualMachineInstance, timeout int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By("Waiting until the VirtualMachineInstance will be scheduled")
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running},
		libwait.WithWarningsPolicy(&wp),
		libwait.WithTimeout(timeout),
	)
}

func getRunningPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) (*k8sv1.Pod, error) {
	virtCli := kubevirt.Client()

	var err error
	vmi, err = virtCli.VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return libpod.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, namespace, vmi.Status.NodeName)
}

// NewRandomVirtualMachineInstanceWithDisk
//
// Deprecated: Use libvmi directly
func NewRandomVirtualMachineInstanceWithDisk(imageUrl, namespace, sc string, accessMode k8sv1.PersistentVolumeAccessMode, volMode k8sv1.PersistentVolumeMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
	virtCli := kubevirt.Client()

	dv := libdv.NewDataVolume(
		libdv.WithRegistryURLSourceAndPullMethod(imageUrl, cdiv1.RegistryPullNode),
		libdv.WithPVC(
			libdv.PVCWithStorageClass(sc),
			libdv.PVCWithVolumeSize(cd.ContainerDiskSizeBySourceURL(imageUrl)),
			libdv.PVCWithAccessMode(accessMode),
			libdv.PVCWithVolumeMode(volMode),
		),
	)

	var err error
	dv, err = virtCli.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	libstorage.EventuallyDV(dv, 240, Or(HaveSucceeded(), WaitForFirstConsumer()))
	return libvmi.New(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithDataVolume("disk0", dv.Name),
		libvmi.WithResourceMemory("1Gi"),
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
	), dv
}

// NewRandomVirtualMachineInstanceWithFileDisk
//
// Deprecated: Use libvmi directly
func NewRandomVirtualMachineInstanceWithFileDisk(imageUrl, namespace string, accessMode k8sv1.PersistentVolumeAccessMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
	if !libstorage.HasCDI() {
		Skip("Skip DataVolume tests when CDI is not present")
	}
	sc, foundSC := libstorage.GetRWOFileSystemStorageClass()
	if accessMode == k8sv1.ReadWriteMany {
		sc, foundSC = libstorage.GetRWXFileSystemStorageClass()
	}
	if !foundSC {
		Skip("Skip test when Filesystem storage is not present")
	}

	return NewRandomVirtualMachineInstanceWithDisk(imageUrl, namespace, sc, accessMode, k8sv1.PersistentVolumeFilesystem)
}

// NewRandomVirtualMachineInstanceWithBlockDisk
//
// Deprecated: Use libvmi directly
func NewRandomVirtualMachineInstanceWithBlockDisk(imageUrl, namespace string, accessMode k8sv1.PersistentVolumeAccessMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
	if !libstorage.HasCDI() {
		Skip("Skip DataVolume tests when CDI is not present")
	}
	sc, exists := libstorage.GetRWOBlockStorageClass()
	if accessMode == k8sv1.ReadWriteMany {
		sc, exists = libstorage.GetRWXBlockStorageClass()
	}
	if !exists {
		Skip("Skip test when Block storage is not present")
	}

	return NewRandomVirtualMachineInstanceWithDisk(imageUrl, namespace, sc, accessMode, k8sv1.PersistentVolumeBlock)
}

// NewRandomVMWithDataVolumeWithRegistryImport
//
// Deprecated: Use libvmi directly
func NewRandomVMWithDataVolumeWithRegistryImport(imageUrl, namespace, storageClass string, accessMode k8sv1.PersistentVolumeAccessMode) *v1.VirtualMachine {
	dataVolume := libdv.NewDataVolume(
		libdv.WithRegistryURLSourceAndPullMethod(imageUrl, cdiv1.RegistryPullNode),
		libdv.WithPVC(
			libdv.PVCWithStorageClass(storageClass),
			libdv.PVCWithVolumeSize(cd.ContainerDiskSizeBySourceURL(imageUrl)),
			libdv.PVCWithAccessMode(accessMode),
		),
	)

	vmi := libvmi.New(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithDataVolume("disk0", dataVolume.Name),
		libvmi.WithResourceMemory("1Gi"),
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
	)
	vm := libvmi.NewVirtualMachine(vmi)

	libstorage.AddDataVolumeTemplate(vm, dataVolume)
	return vm
}

func cirrosMemory() string {
	// Cirros image need 256M to boot on ARM64,
	// this issue is traced in https://github.com/kubevirt/kubevirt/issues/6363
	if checks.IsARM64(testsuite.Arch) {
		return "256Mi"
	}
	return "128Mi"
}

// NewRandomVMIWithEphemeralDisk
//
// Deprecated: Use libvmi directly
func NewRandomVMIWithEphemeralDisk(containerImage string) *v1.VirtualMachineInstance {
	opts := []libvmi.Option{
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
		libvmi.WithResourceMemory(cirrosMemory()),
		libvmi.WithContainerDisk("disk0", containerImage),
	}
	if containerImage == cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling) {
		opts = append(
			[]libvmi.Option{libvmi.WithRng()},
			opts...)
	}
	return libvmi.New(opts...)
}

// AddEphemeralDisk
//
// Deprecated: Use libvmi
func AddEphemeralDisk(vmi *v1.VirtualMachineInstance, name string, bus v1.DiskBus, image string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image: image,
			},
		},
	})

	return vmi
}

func GetFedoraToolsGuestAgentBlacklistUserData(commands string) string {
	return fmt.Sprintf(`#!/bin/bash
            echo -e "\n\nBLACKLIST_RPC=%s" | sudo tee -a /etc/sysconfig/qemu-ga
`, commands)
}

// NewRandomVMIWithEphemeralDiskAndUserdata
//
// Deprecated: Use libvmi directly
func NewRandomVMIWithEphemeralDiskAndUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddUserData(vmi, "disk1", userData)
	return vmi
}

// NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata
//
// Deprecated: Use libvmi directly
func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitConfigDriveData(vmi, "disk1", userData, "", false)
	return vmi
}

// NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData
//
// Deprecated: Use libvmi directly
func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitConfigDriveData(vmi, "disk1", userData, networkData, b64encode)
	return vmi
}

// AddUserData
//
// Deprecated: Use libvmi
func AddUserData(vmi *v1.VirtualMachineInstance, name string, userData string) {
	AddCloudInitNoCloudData(vmi, name, userData, "", true)
}

// AddCloudInitNoCloudData
//
// Deprecated: Use libvmi
func AddCloudInitNoCloudData(vmi *v1.VirtualMachineInstance, name, userData, networkData string, b64encode bool) {
	cloudInitNoCloudSource := v1.CloudInitNoCloudSource{}
	if b64encode {
		cloudInitNoCloudSource.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userData))
		if networkData != "" {
			cloudInitNoCloudSource.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(networkData))
		}
	} else {
		cloudInitNoCloudSource.UserData = userData
		if networkData != "" {
			cloudInitNoCloudSource.NetworkData = networkData
		}
	}
	addCloudInitDiskAndVolume(vmi, name, v1.VolumeSource{CloudInitNoCloud: &cloudInitNoCloudSource})
}

// AddCloudInitConfigDriveData
//
// Deprecated: Use libvmi
func AddCloudInitConfigDriveData(vmi *v1.VirtualMachineInstance, name, userData, networkData string, b64encode bool) {
	cloudInitConfigDriveSource := v1.CloudInitConfigDriveSource{}
	if b64encode {
		cloudInitConfigDriveSource.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(userData))
		if networkData != "" {
			cloudInitConfigDriveSource.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(networkData))
		}
	} else {
		cloudInitConfigDriveSource.UserData = userData
		if networkData != "" {
			cloudInitConfigDriveSource.NetworkData = networkData
		}
	}
	addCloudInitDiskAndVolume(vmi, name, v1.VolumeSource{CloudInitConfigDrive: &cloudInitConfigDriveSource})
}

func addCloudInitDiskAndVolume(vmi *v1.VirtualMachineInstance, name string, volumeSource v1.VolumeSource) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusVirtio,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name:         name,
		VolumeSource: volumeSource,
	})
}

func NewRandomReplicaSetFromVMI(vmi *v1.VirtualMachineInstance, replicas int32) *v1.VirtualMachineInstanceReplicaSet {
	name := "replicaset" + rand.String(5)
	rs := &v1.VirtualMachineInstanceReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "replicaset" + rand.String(5)},
		Spec: v1.VirtualMachineInstanceReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"name": name},
			},
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"name": name},
					Name:   vmi.ObjectMeta.Name,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return rs
}

func RunPodInNamespace(pod *k8sv1.Pod, namespace string) *k8sv1.Pod {
	virtClient := kubevirt.Client()

	var err error
	pod, err = virtClient.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisPod(pod), 180).Should(BeInPhase(k8sv1.PodRunning))

	pod, err = ThisPod(pod)()
	Expect(err).ToNot(HaveOccurred())
	return pod
}

func RunPod(pod *k8sv1.Pod) *k8sv1.Pod {
	return RunPodInNamespace(pod, testsuite.GetTestNamespace(pod))
}

func RunPodAndExpectCompletion(pod *k8sv1.Pod) *k8sv1.Pod {
	virtClient := kubevirt.Client()

	var err error
	pod, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisPod(pod), 120).Should(BeInPhase(k8sv1.PodSucceeded))

	pod, err = ThisPod(pod)()
	Expect(err).ToNot(HaveOccurred())
	return pod
}

func ChangeImgFilePermissionsToNonQEMU(pvc *k8sv1.PersistentVolumeClaim) {
	args := []string{fmt.Sprintf(`chmod 640 %s && chown root:root %s && sync`, filepath.Join(libstorage.DefaultPvcMountPath, "disk.img"), filepath.Join(libstorage.DefaultPvcMountPath, "disk.img"))}

	By("changing disk.img permissions to non qemu")
	pod := libstorage.RenderPodWithPVC("change-permissions-disk-img-pod", []string{"/bin/bash", "-c"}, args, pvc)

	// overwrite securityContext
	rootUser := int64(0)
	pod.Spec.Containers[0].SecurityContext = &k8sv1.SecurityContext{
		Capabilities: &k8sv1.Capabilities{
			Drop: []k8sv1.Capability{"ALL"},
		},
		Privileged:   pointer.P(true),
		RunAsUser:    &rootUser,
		RunAsNonRoot: pointer.P(false),
	}

	RunPodAndExpectCompletion(pod)
}

func CopyAlpineWithNonQEMUPermissions() (dstPath, nodeName string) {
	dstPath = testsuite.HostPathAlpine + "-nopriv"
	args := []string{fmt.Sprintf(`mkdir -p %[1]s-nopriv && cp %[1]s/disk.img %[1]s-nopriv/ && chmod 640 %[1]s-nopriv/disk.img  && chown root:root %[1]s-nopriv/disk.img`, testsuite.HostPathAlpine)}

	By("creating an image with without qemu permissions")
	pod := libpod.RenderHostPathPod("tmp-image-create-job", testsuite.HostPathBase, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationNone, []string{BinBash, "-c"}, args)

	nodeName = RunPodAndExpectCompletion(pod).Spec.NodeName
	return
}

func DeleteAlpineWithNonQEMUPermissions() {
	nonQemuAlpinePath := testsuite.HostPathAlpine + "-nopriv"
	args := []string{fmt.Sprintf(`rm -rf %s`, nonQemuAlpinePath)}

	pod := libpod.RenderHostPathPod("remove-tmp-image-job", testsuite.HostPathBase, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationNone, []string{BinBash, "-c"}, args)

	RunPodAndExpectCompletion(pod)
}

func GetRunningVirtualMachineInstanceDomainXML(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (string, error) {
	vmiPod, err := getRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
	if err != nil {
		return "", err
	}

	// get current vmi
	freshVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get vmi, %s", err)
	}

	command := []string{"virsh"}
	if kutil.IsNonRootVMI(freshVMI) {
		command = append(command, "-c")
		command = append(command, "qemu+unix:///session?socket=/var/run/libvirt/virtqemud-sock")
	}
	command = append(command, []string{"dumpxml", vmi.Namespace + "_" + vmi.Name}...)

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
		vmiPod,
		libpod.LookupComputeContainer(vmiPod).Name,
		command,
	)
	if err != nil {
		return "", fmt.Errorf("could not dump libvirt domxml (remotely on pod %s): %v: %s, %s", vmiPod.Name, err, stdout, stderr)
	}
	return stdout, err
}

func GenerateVMJson(vm *v1.VirtualMachine, generateDirectory string) (string, error) {
	data, err := json.Marshal(vm)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for vm %s", vm.Name)
	}

	jsonFile := filepath.Join(generateDirectory, fmt.Sprintf("%s.json", vm.Name))
	err = os.WriteFile(jsonFile, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write json file %s", jsonFile)
	}
	return jsonFile, nil
}

func NotDeleted(vmis *v1.VirtualMachineInstanceList) (notDeleted []v1.VirtualMachineInstance) {
	for _, vmi := range vmis.Items {
		if vmi.DeletionTimestamp == nil {
			notDeleted = append(notDeleted, vmi)
		}
	}
	return
}

func NotDeletedVMs(vms *v1.VirtualMachineList) (notDeleted []v1.VirtualMachine) {
	for _, vm := range vms.Items {
		if vm.DeletionTimestamp == nil {
			notDeleted = append(notDeleted, vm)
		}
	}
	return
}

func UnfinishedVMIPodSelector(vmi *v1.VirtualMachineInstance) metav1.ListOptions {
	virtClient := kubevirt.Client()

	var err error
	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	fieldSelectorStr := "status.phase!=" + string(k8sv1.PodFailed) +
		",status.phase!=" + string(k8sv1.PodSucceeded)

	if vmi.Status.NodeName != "" {
		fieldSelectorStr = fieldSelectorStr +
			",spec.nodeName=" + vmi.Status.NodeName
	}

	fieldSelector := fields.ParseSelectorOrDie(fieldSelectorStr)
	labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel + "=virt-launcher," + v1.CreatedByLabel + "=" + string(vmi.GetUID())))
	if err != nil {
		panic(err)
	}
	return metav1.ListOptions{FieldSelector: fieldSelector.String(), LabelSelector: labelSelector.String()}
}

func RemoveHostDiskImage(diskPath string, nodeName string) {
	virtClient := kubevirt.Client()
	procPath := filepath.Join("/proc/1/root", diskPath)
	virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
	Expect(err).ToNot(HaveOccurred())
	_, _, err = exec.ExecuteCommandOnPodWithResults(virtHandlerPod, "virt-handler", []string{"rm", "-rf", procPath})
	Expect(err).ToNot(HaveOccurred())
}

func GetVmiPod(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), UnfinishedVMIPodSelector(vmi))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pods.Items).NotTo(BeEmpty())
	vmiPod := pods.Items[0]

	return &vmiPod
}

// RunCommandOnVmiPod runs specified command on the virt-launcher pod
func RunCommandOnVmiPod(vmi *v1.VirtualMachineInstance, command []string) string {
	virtClient := kubevirt.Client()
	vmiPod := GetVmiPod(virtClient, vmi)

	output, err := exec.ExecuteCommandOnPod(
		vmiPod,
		"compute",
		command,
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return output
}

func StopVirtualMachineWithTimeout(vm *v1.VirtualMachine, timeout time.Duration) *v1.VirtualMachine {
	By("Stopping the VirtualMachineInstance")
	virtClient := kubevirt.Client()

	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = pointer.P(false)
		updatedVM.Spec.RunStrategy = nil
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(context.Background(), updatedVM, metav1.UpdateOptions{})
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	// Observe the VirtualMachineInstance deleted
	Eventually(func() error {
		_, err = virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, metav1.GetOptions{})
		return err
	}, timeout, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "The vmi did not disappear")
	By("VM has not the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, timeout, 1*time.Second).Should(BeFalse())
	return updatedVM
}

func StopVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	return StopVirtualMachineWithTimeout(vm, time.Second*300)
}

func StartVirtualMachine(vm *v1.VirtualMachine) *v1.VirtualMachine {
	By("Starting the VirtualMachineInstance")
	virtClient := kubevirt.Client()

	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = pointer.P(true)
		updatedVM.Spec.RunStrategy = nil
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(context.Background(), updatedVM, metav1.UpdateOptions{})
		return err
	}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	// Observe the VirtualMachineInstance created
	Eventually(func() error {
		_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, metav1.GetOptions{})
		return err
	}, 300*time.Second, 1*time.Second).Should(Succeed())
	By("VMI has the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeTrue())
	return updatedVM
}

func DisableFeatureGate(feature string) {
	if !checks.HasFeature(feature) {
		return
	}
	virtClient := kubevirt.Client()

	kv := util2.GetCurrentKv(virtClient)
	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{
			FeatureGates: []string{},
		}
	}

	var newArray []string
	featureGates := kv.Spec.Configuration.DeveloperConfiguration.FeatureGates
	for _, fg := range featureGates {
		if fg == feature {
			continue
		}

		newArray = append(newArray, fg)
	}

	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = newArray
	if checks.RequireFeatureGateVirtHandlerRestart(feature) {
		updateKubeVirtConfigValueAndWaitHandlerRedeploymnet(kv.Spec.Configuration)
		return
	}

	UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
}

func EnableFeatureGate(feature string) *v1.KubeVirt {
	virtClient := kubevirt.Client()

	kv := util2.GetCurrentKv(virtClient)
	if checks.HasFeature(feature) {
		return kv
	}

	if kv.Spec.Configuration.DeveloperConfiguration == nil {
		kv.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{
			FeatureGates: []string{},
		}
	}

	kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates, feature)

	if checks.RequireFeatureGateVirtHandlerRestart(feature) {
		return updateKubeVirtConfigValueAndWaitHandlerRedeploymnet(kv.Spec.Configuration)
	}

	return UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
}

func GetVmPodName(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()
	labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

	pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty())

	return podName
}

func GetRunningVMIDomainSpec(vmi *v1.VirtualMachineInstance) (*launcherApi.DomainSpec, error) {
	runningVMISpec := launcherApi.DomainSpec{}
	cli := kubevirt.Client()

	domXML, err := GetRunningVirtualMachineInstanceDomainXML(cli, vmi)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal([]byte(domXML), &runningVMISpec)
	return &runningVMISpec, err
}

func GetRunningVMIEmulator(vmi *v1.VirtualMachineInstance) (string, error) {
	domSpec, err := GetRunningVMIDomainSpec(vmi)
	if err != nil {
		return "", err
	}
	return domSpec.Devices.Emulator, nil
}

func ForwardPorts(pod *k8sv1.Pod, ports []string, stop chan struct{}, readyTimeout time.Duration) error {
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	go func() {
		cli := kubevirt.Client()

		req := cli.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(pod.Namespace).
			Name(pod.Name).
			SubResource("portforward")

		kubevirtClientConfig, err := kubecli.GetKubevirtClientConfig()
		if err != nil {
			errChan <- err
			return
		}
		transport, upgrader, err := spdy.RoundTripperFor(kubevirtClientConfig)
		if err != nil {
			errChan <- err
			return
		}
		dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
		forwarder, err := portforward.New(dialer, ports, stop, readyChan, GinkgoWriter, GinkgoWriter)
		if err != nil {
			errChan <- err
			return
		}
		err = forwarder.ForwardPorts()
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-readyChan:
		return nil
	case <-time.After(readyTimeout):
		return fmt.Errorf("failed to forward ports, timed out")
	}
}

func GenerateHelloWorldServer(vmi *v1.VirtualMachineInstance, testPort int, protocol string, loginTo console.LoginToFunction, sudoNeeded bool) {
	Expect(loginTo(vmi)).To(Succeed())

	sudoPrefix := ""
	if sudoNeeded {
		sudoPrefix = "sudo "
	}

	serverCommand := fmt.Sprintf("%snc -klp %d -e echo -e 'Hello World!'&\n", sudoPrefix, testPort)
	if protocol == "udp" {
		// nc has to be in a while loop in case of UDP, since it exists after one message
		serverCommand = fmt.Sprintf("%ssh -c \"while true; do nc -uklp %d -e echo -e 'Hello UDP World!';done\"&\n", sudoPrefix, testPort)
	}
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: serverCommand},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: EchoLastReturnValue},
		&expect.BExp{R: console.RetValue("0")},
	}, 60)).To(Succeed())
}

func updateKubeVirtConfigValueAndWaitHandlerRedeploymnet(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {
	virtClient := kubevirt.Client()
	ds, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.TODO(), "virt-handler", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	currentGen := ds.Status.ObservedGeneration
	kv := testsuite.UpdateKubeVirtConfigValue(kvConfig)
	Eventually(func() bool {
		ds, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.TODO(), "virt-handler", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		gen := ds.Status.ObservedGeneration
		if gen > currentGen {
			return true
		}
		return false

	}, 90*time.Second, 1*time.Second).Should(BeTrue())

	waitForConfigToBePropagated(kv.ResourceVersion)
	log.DefaultLogger().Infof("system is in sync with kubevirt config resource version %s", kv.ResourceVersion)

	return kv
}

// UpdateKubeVirtConfigValueAndWait updates the given configuration in the kubevirt custom resource
// and then waits  to allow the configuration events to be propagated to the consumers.
func UpdateKubeVirtConfigValueAndWait(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {
	kv := testsuite.UpdateKubeVirtConfigValue(kvConfig)

	waitForConfigToBePropagated(kv.ResourceVersion)
	log.DefaultLogger().Infof("system is in sync with kubevirt config resource version %s", kv.ResourceVersion)

	return kv
}

type compare func(string, string) bool

func ExpectResourceVersionToBeLessEqualThanConfigVersion(resourceVersion, configVersion string) bool {
	rv, err := strconv.ParseInt(resourceVersion, 10, 32)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Resource version is unable to be parsed")
		return false
	}

	crv, err := strconv.ParseInt(configVersion, 10, 32)
	if err != nil {
		log.DefaultLogger().Reason(err).Errorf("Config resource version is unable to be parsed")
		return false
	}

	if rv > crv {
		log.DefaultLogger().Errorf("Config is not in sync. Expected %s or greater, Got %s", resourceVersion, configVersion)
		return false
	}

	return true
}

func waitForConfigToBePropagated(resourceVersion string) {
	WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-controller", resourceVersion, ExpectResourceVersionToBeLessEqualThanConfigVersion, 10*time.Second)
	WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-api", resourceVersion, ExpectResourceVersionToBeLessEqualThanConfigVersion, 10*time.Second)
	WaitForConfigToBePropagatedToComponent("kubevirt.io=virt-handler", resourceVersion, ExpectResourceVersionToBeLessEqualThanConfigVersion, 10*time.Second)
}

func WaitForConfigToBePropagatedToComponent(podLabel string, resourceVersion string, compareResourceVersions compare, duration time.Duration) {
	virtClient := kubevirt.Client()

	errComponentInfo := fmt.Sprintf("component: \"%s\"", strings.TrimPrefix(podLabel, "kubevirt.io="))

	EventuallyWithOffset(3, func() error {
		pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: podLabel})

		if err != nil {
			return fmt.Errorf("failed to fetch pods: %v, %s", err, errComponentInfo)
		}
		for _, pod := range pods.Items {
			errAdditionalInfo := errComponentInfo + fmt.Sprintf(", pod: \"%s\"", pod.Name)

			if pod.DeletionTimestamp != nil {
				continue
			}

			body, err := callUrlOnPod(&pod, "8443", "/healthz")
			if err != nil {
				return fmt.Errorf("failed to call healthz endpoint: %v, %s", err, errAdditionalInfo)
			}
			result := map[string]interface{}{}
			err = json.Unmarshal(body, &result)
			if err != nil {
				return fmt.Errorf("failed to parse response from healthz endpoint: %v, %s", err, errAdditionalInfo)
			}

			if configVersion := result["config-resource-version"].(string); !compareResourceVersions(resourceVersion, configVersion) {
				return fmt.Errorf("resource & config versions (%s and %s respectively) are not as expected. %s ",
					resourceVersion, configVersion, errAdditionalInfo)
			}
		}
		return nil
	}, duration, 1*time.Second).ShouldNot(HaveOccurred())
}

func RetryWithMetadataIfModified(objectMeta metav1.ObjectMeta, do func(objectMeta metav1.ObjectMeta) error) (err error) {
	return RetryIfModified(func() error {
		return do(objectMeta)
	})
}

func RetryIfModified(do func() error) (err error) {
	retries := 0
	for err = do(); errors.IsConflict(err); err = do() {
		if retries >= 10 {
			return fmt.Errorf("object seems to be permanently modified, failing after 10 retries: %v", err)
		}
		retries++
		log.DefaultLogger().Reason(err).Infof("Object got modified, will retry.")
	}
	return err
}

func getCert(pod *k8sv1.Pod, port string) []byte {
	randPort := strconv.Itoa(4321 + rand.Intn(6000))
	var rawCert []byte
	mutex := &sync.Mutex{}
	conf := &tls.Config{
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			mutex.Lock()
			defer mutex.Unlock()
			rawCert = rawCerts[0]
			return nil
		},
	}

	var certificate []byte
	EventuallyWithOffset(2, func() []byte {
		stopChan := make(chan struct{})
		defer close(stopChan)
		err := ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 10*time.Second)
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		conn, err := tls.Dial("tcp4", fmt.Sprintf("localhost:%s", randPort), conf)
		if err == nil {
			defer conn.Close()
		}
		mutex.Lock()
		defer mutex.Unlock()
		certificate = make([]byte, len(rawCert))
		copy(certificate, rawCert)
		return certificate
	}, 40*time.Second, 1*time.Second).Should(Not(BeEmpty()))

	return certificate
}

func callUrlOnPod(pod *k8sv1.Pod, port string, url string) ([]byte, error) {
	randPort := strconv.Itoa(4321 + rand.Intn(6000))
	stopChan := make(chan struct{})
	defer close(stopChan)
	err := ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 5*time.Second)
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true, VerifyPeerCertificate: func(_ [][]byte, _ [][]*x509.Certificate) error {
			return nil
		}},
	}
	client := &http.Client{Transport: tr}
	resp, err := client.Get(fmt.Sprintf("https://localhost:%s/%s", randPort, strings.TrimSuffix(url, "/")))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// GetCertsForPods returns the used certificates for all pods matching  the label selector
func GetCertsForPods(labelSelector string, namespace string, port string) ([][]byte, error) {
	cli := kubevirt.Client()
	pods, err := cli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(pods.Items).ToNot(BeEmpty())

	var certs [][]byte

	for _, pod := range pods.Items {
		err := func() error {
			certs = append(certs, getCert(&pod, port))
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}
	return certs, nil
}

// EnsurePodsCertIsSynced waits until new certificates are rolled out  to all pods which are matching the specified labelselector.
// Once all certificates are in sync, the final secret is returned
func EnsurePodsCertIsSynced(labelSelector string, namespace string, port string) []byte {
	var certs [][]byte
	EventuallyWithOffset(1, func() bool {
		var err error
		certs, err = GetCertsForPods(labelSelector, namespace, port)
		Expect(err).ToNot(HaveOccurred())
		if len(certs) == 0 {
			return true
		}
		for _, crt := range certs {
			if !reflect.DeepEqual(certs[0], crt) {
				return false
			}
		}
		return true
	}, 90*time.Second, 1*time.Second).Should(BeTrue(), "certificates across '%s' pods are not in sync", labelSelector)
	if len(certs) > 0 {
		return certs[0]
	}
	return nil
}

// GetPodsCertIfSynced returns the certificate for all matching pods once all of them use the same certificate
func GetPodsCertIfSynced(labelSelector string, namespace string, port string) (cert []byte, synced bool, err error) {
	certs, err := GetCertsForPods(labelSelector, namespace, port)
	if err != nil {
		return nil, false, err
	}
	if len(certs) == 0 {
		return nil, true, nil
	}
	for _, crt := range certs {
		if !reflect.DeepEqual(certs[0], crt) {
			return nil, false, nil
		}
	}
	return certs[0], true, nil
}

func GetBundleFromConfigMap(configMapName string) ([]byte, []*x509.Certificate) {
	virtClient := kubevirt.Client()
	configMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if rawBundle, ok := configMap.Data[components.CABundleKey]; ok {
		crts, err := cert.ParseCertsPEM([]byte(rawBundle))
		Expect(err).ToNot(HaveOccurred())
		return []byte(rawBundle), crts
	}
	return nil, nil
}

func RandTmpDir() string {
	const tmpPath = "/var/provision/kubevirt.io/tests"
	return filepath.Join(tmpPath, rand.String(10))
}

func CheckCloudInitMetaData(vmi *v1.VirtualMachineInstance, testFile, testData string) {
	cmdCheck := "cat " + filepath.Join("/mnt", testFile) + "\n"
	res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
		&expect.BSnd{S: "sudo su -\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: testData},
	}, 15)
	if err != nil {
		Expect(res[1].Output).To(ContainSubstring(testData))
	}
}

func MountCloudInitFunc(devName string) func(*v1.VirtualMachineInstance) {
	return func(vmi *v1.VirtualMachineInstance) {
		cmdCheck := fmt.Sprintf("mount $(blkid  -L %s) /mnt/\n", devName)
		err := console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo su -\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: cmdCheck},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}
}

func ArchiveToFile(tgtFile *os.File, sourceFilesNames ...string) {
	w := tar.NewWriter(tgtFile)
	defer w.Close()

	for _, src := range sourceFilesNames {
		srcFile, err := os.Open(src)
		Expect(err).ToNot(HaveOccurred())
		defer srcFile.Close()

		srcFileInfo, err := srcFile.Stat()
		Expect(err).ToNot(HaveOccurred())

		hdr, err := tar.FileInfoHeader(srcFileInfo, "")
		Expect(err).ToNot(HaveOccurred())

		err = w.WriteHeader(hdr)
		Expect(err).ToNot(HaveOccurred())

		_, err = io.Copy(w, srcFile)
		Expect(err).ToNot(HaveOccurred())
	}
}

func ExecuteCommandOnNodeThroughVirtHandler(virtCli kubecli.KubevirtClient, nodeName string, command []string) (stdout string, stderr string, err error) {
	virtHandlerPod, err := libnode.GetVirtHandlerPod(virtCli, nodeName)
	if err != nil {
		return "", "", err
	}
	return exec.ExecuteCommandOnPodWithResults(virtHandlerPod, components.VirtHandlerName, command)
}

func RunVMAndExpectLaunchWithRunStrategy(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, runStrategy v1.VirtualMachineRunStrategy) *v1.VirtualMachine {
	By("Starting the VirtualMachine")

	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = nil
		updatedVM.Spec.RunStrategy = &runStrategy
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(context.Background(), updatedVM, metav1.UpdateOptions{})
		return err
	}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	// Observe the VirtualMachineInstance created
	Eventually(func() error {
		_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, metav1.GetOptions{})
		return err
	}, 300*time.Second, 1*time.Second).Should(Succeed())

	By("VMI has the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(context.Background(), updatedVM.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeTrue())

	return updatedVM
}
