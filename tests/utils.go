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
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/transport/spdy"
	netutils "k8s.io/utils/net"

	"kubevirt.io/kubevirt/tests/framework/checks"

	util2 "kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/tests/framework/cleanup"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"

	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	kutil "kubevirt.io/kubevirt/pkg/util"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	launcherApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

const (
	BinBash                      = "/bin/bash"
	StartingVMInstance           = "Starting a VirtualMachineInstance"
	WaitingVMInstanceStart       = "Waiting until the VirtualMachineInstance will start"
	CouldNotFindComputeContainer = "could not find compute container for pod"
	EchoLastReturnValue          = "echo $?\n"
)

const (
	CustomHostPath = "custom-host-path"
)

const (
	DiskAlpineHostPath = "disk-alpine-host-path"
	DiskWindows        = "disk-windows"
	DiskWindowsSysprep = "disk-windows-sysprep"
	DiskCustomHostPath = "disk-custom-host-path"
)

const (
	defaultDiskSize = "1Gi"
)

const MigrationWaitTime = 240
const ContainerCompletionWaitTime = 60

func TestCleanup() {
	GinkgoWriter.Println("Global test cleanup started.")
	testsuite.CleanNamespaces()
	libnode.CleanNodes()
	resetToDefaultConfig()
	testsuite.EnsureKubevirtInfra()
	SetupAlpineHostPath()
	GinkgoWriter.Println("Global test cleanup ended.")
}

func SetupAlpineHostPath() {
	const osAlpineHostPath = "alpine-host-path"
	libstorage.CreateHostPathPv(osAlpineHostPath, testsuite.HostPathAlpine)
	libstorage.CreateHostPathPVC(osAlpineHostPath, defaultDiskSize)
}

func GetSupportedCPUFeatures(nodes k8sv1.NodeList) []string {
	var featureDenyList = map[string]bool{
		"svm": true,
	}
	featuresMap := make(map[string]bool)
	for _, node := range nodes.Items {
		for key := range node.Labels {
			if strings.Contains(key, services.NFD_CPU_FEATURE_PREFIX) {
				feature := strings.TrimPrefix(key, services.NFD_CPU_FEATURE_PREFIX)
				if _, ok := featureDenyList[feature]; !ok {
					featuresMap[feature] = true
				}
			}
		}
	}

	features := make([]string, 0)
	for feature := range featuresMap {
		features = append(features, feature)
	}
	return features
}

func GetSupportedCPUModels(nodes k8sv1.NodeList) []string {
	var cpuDenyList = map[string]bool{
		"qemu64":     true,
		"Opteron_G2": true,
	}
	cpuMap := make(map[string]bool)
	for _, node := range nodes.Items {
		for key := range node.Labels {
			if strings.Contains(key, services.NFD_CPU_MODEL_PREFIX) {
				cpu := strings.TrimPrefix(key, services.NFD_CPU_MODEL_PREFIX)
				if _, ok := cpuDenyList[cpu]; !ok {
					cpuMap[cpu] = true
				}
			}
		}
	}

	cpus := make([]string, 0)
	for model := range cpuMap {
		cpus = append(cpus, model)
	}
	return cpus
}

func CreateConfigMap(name string, data map[string]string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)
	_, err = virtCli.CoreV1().ConfigMaps(util2.NamespaceTestDefault).Create(context.Background(), &k8sv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Data:       data,
	}, metav1.CreateOptions{})

	if !errors.IsAlreadyExists(err) {
		util2.PanicOnError(err)
	}
}

func CreateSecret(name string, data map[string]string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	_, err = virtCli.CoreV1().Secrets(util2.NamespaceTestDefault).Create(context.Background(), &k8sv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		StringData: data,
	}, metav1.CreateOptions{})
	if !errors.IsAlreadyExists(err) {
		util2.PanicOnError(err)
	}
}

func ServiceMonitorEnabled() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	serviceMonitorEnabled, err := util.IsServiceMonitorEnabled(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can't verify ServiceMonitor CRD %v\n", err)
		panic(err)
	}

	return serviceMonitorEnabled
}

// PrometheusRuleEnabled returns true if the PrometheusRule CRD is enabled
// and false otherwise.
func PrometheusRuleEnabled() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	prometheusRuleEnabled, err := util.IsPrometheusRuleEnabled(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can't verify PrometheusRule CRD %v\n", err)
		panic(err)
	}

	return prometheusRuleEnabled
}

func DeleteConfigMap(name string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	err = virtCli.CoreV1().ConfigMaps(util2.NamespaceTestDefault).Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util2.PanicOnError(err)
	}
}

func DeleteSecret(name string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	err = virtCli.CoreV1().Secrets(util2.NamespaceTestDefault).Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util2.PanicOnError(err)
	}
}
func RunVMI(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	By(StartingVMInstance)
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	var obj *v1.VirtualMachineInstance
	Eventually(func() error {
		obj, err = virtCli.VirtualMachineInstance(util2.NamespaceTestDefault).Create(vmi)
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	return obj
}

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By(WaitingVMInstanceStart)
	return WaitForSuccessfulVMIStartWithTimeout(obj, timeout)
}

func RunVMIAndExpectLaunchWithDataVolume(vmi *v1.VirtualMachineInstance, dv *cdiv1.DataVolume, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By("Waiting until the DataVolume is ready")
	libstorage.EventuallyDV(dv, timeout, HaveSucceeded())
	By(WaitingVMInstanceStart)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	warningsIgnoreList := []string{"didn't find PVC"}
	wp := watcher.WarningsPolicy{FailOnWarnings: true, WarningsIgnoreList: warningsIgnoreList}
	return waitForVMIStart(ctx, obj, timeout, wp)
}

func RunVMIAndExpectLaunchIgnoreWarnings(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By(WaitingVMInstanceStart)
	return WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(obj, timeout)
}

func RunVMIAndExpectScheduling(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	obj := RunVMI(vmi, timeout)
	By("Waiting until the VirtualMachineInstance will be scheduled")
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return waitForVMIScheduling(obj, timeout, wp)
}

func getRunningPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) (*k8sv1.Pod, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	vmi, err = virtCli.VirtualMachineInstance(namespace).Get(vmi.Name, &metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, namespace, vmi.Status.NodeName)
}

func GetRunningPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance, namespace string) *k8sv1.Pod {
	pod, err := getRunningPodByVirtualMachineInstance(vmi, namespace)
	util2.PanicOnError(err)
	return pod
}

func GetPodByVirtualMachineInstance(vmi *v1.VirtualMachineInstance) *k8sv1.Pod {
	pods, err := getPodsByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace)
	util2.PanicOnError(err)

	if len(pods.Items) != 1 {
		util2.PanicOnError(fmt.Errorf("found wrong number of pods for VMI '%v', count: %d", vmi, len(pods.Items)))
	}

	return &pods.Items[0]
}

func getPodsByLabel(label, labelType, namespace string) (*k8sv1.PodList, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	labelSelector := fmt.Sprintf("%s=%s", labelType, label)

	pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector},
	)
	if err != nil {
		return nil, err
	}

	return pods, nil
}

func GetPodCPUSet(pod *k8sv1.Pod) (output string, err error) {
	const (
		cgroupV1cpusetPath = "/sys/fs/cgroup/cpuset/cpuset.cpus"
		cgroupV2cpusetPath = "/sys/fs/cgroup/cpuset.cpus.effective"
	)

	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return
	}
	output, err = ExecuteCommandOnPod(
		virtClient,
		pod,
		"compute",
		[]string{"cat", cgroupV2cpusetPath},
	)

	if err == nil {
		return
	}

	output, err = ExecuteCommandOnPod(
		virtClient,
		pod,
		"compute",
		[]string{"cat", cgroupV1cpusetPath},
	)

	return
}

func GetRunningPodByLabel(label string, labelType string, namespace string, node string) (*k8sv1.Pod, error) {
	virtCli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	labelSelector := fmt.Sprintf("%s=%s", labelType, label)
	var fieldSelector string
	if node != "" {
		fieldSelector = fmt.Sprintf("status.phase==%s,spec.nodeName==%s", k8sv1.PodRunning, node)
	} else {
		fieldSelector = fmt.Sprintf("status.phase==%s", k8sv1.PodRunning)
	}
	pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector, FieldSelector: fieldSelector},
	)
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("failed to find pod with the label %s", label)
	}

	var readyPod *k8sv1.Pod
	for _, pod := range pods.Items {
		ready := true
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == "kubevirt-infra" {
				ready = status.Ready
				break
			}
		}
		if ready {
			readyPod = &pod
			break
		}
	}
	if readyPod == nil {
		return nil, fmt.Errorf("no ready pods with the label %s", label)
	}

	return readyPod, nil
}

func GetComputeContainerOfPod(pod *k8sv1.Pod) *k8sv1.Container {
	return GetContainerOfPod(pod, "compute")
}

func GetContainerOfPod(pod *k8sv1.Pod, containerName string) *k8sv1.Container {
	var computeContainer *k8sv1.Container
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			computeContainer = &container
			break
		}
	}
	if computeContainer == nil {
		util2.PanicOnError(fmt.Errorf("could not find the %s container", containerName))
	}
	return computeContainer
}

func NewRandomVirtualMachineInstanceWithDisk(imageUrl, namespace, sc string, accessMode k8sv1.PersistentVolumeAccessMode, volMode k8sv1.PersistentVolumeMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	dv := libstorage.NewDataVolumeWithRegistryImportInStorageClass(imageUrl, namespace, sc, accessMode, volMode)
	_, err = virtCli.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	libstorage.EventuallyDV(dv, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))
	return NewRandomVMIWithDataVolume(dv.Name), dv
}

func NewRandomVirtualMachineInstanceWithFileDisk(imageUrl, namespace string, accessMode k8sv1.PersistentVolumeAccessMode) (*v1.VirtualMachineInstance, *cdiv1.DataVolume) {
	if !libstorage.HasCDI() {
		Skip("Skip DataVolume tests when CDI is not present")
	}
	sc, exists := libstorage.GetRWOFileSystemStorageClass()
	if accessMode == k8sv1.ReadWriteMany {
		sc, exists = libstorage.GetRWXFileSystemStorageClass()
	}
	if !exists {
		Skip("Skip test when Filesystem storage is not present")
	}

	return NewRandomVirtualMachineInstanceWithDisk(imageUrl, namespace, sc, accessMode, k8sv1.PersistentVolumeFilesystem)
}

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

func NewRandomVMI() *v1.VirtualMachineInstance {
	// To avoid mac address issue in the tests change the pod interface binding to masquerade
	// https://github.com/kubevirt/kubevirt/issues/1494
	vmi := libvmi.New(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
	)
	vmi.ObjectMeta.Namespace = util2.NamespaceTestDefault
	vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{}

	if checks.IsARM64(testsuite.Arch) {
		// Cirros image need 256M to boot on ARM64,
		// this issue is traced in https://github.com/kubevirt/kubevirt/issues/6363
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("256Mi")
	} else {
		vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
	}

	return vmi
}

func NewRandomVMIWithDataVolume(dataVolumeName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	diskName := "disk0"

	vmi = libstorage.AddDataVolumeDisk(vmi, diskName, dataVolumeName)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")
	return vmi
}

func NewRandomVMWithEphemeralDisk(containerImage string) *v1.VirtualMachine {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	vm := NewRandomVirtualMachine(vmi, false)

	return vm
}

func NewRandomVMWithDataVolumeWithRegistryImport(imageUrl, namespace, storageClass string, accessMode k8sv1.PersistentVolumeAccessMode) *v1.VirtualMachine {
	dataVolume := libstorage.NewDataVolumeWithRegistryImportInStorageClass(imageUrl, namespace, storageClass, accessMode, k8sv1.PersistentVolumeFilesystem)
	dataVolume.Spec.PVC.Resources.Requests[k8sv1.ResourceStorage] = resource.MustParse("6Gi")
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	vm := NewRandomVirtualMachine(vmi, false)

	libstorage.AddDataVolumeTemplate(vm, dataVolume)
	return vm
}

func NewRandomVMWithDataVolume(imageUrl string, namespace string) *v1.VirtualMachine {
	dataVolume := libstorage.NewDataVolumeWithRegistryImport(imageUrl, namespace, k8sv1.ReadWriteOnce)
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	vm := NewRandomVirtualMachine(vmi, false)

	libstorage.AddDataVolumeTemplate(vm, dataVolume)
	return vm
}

func NewRandomVMWithDataVolumeAndUserData(dataVolume *cdiv1.DataVolume, userData string) *v1.VirtualMachine {
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	AddUserData(vmi, "cloud-init", userData)
	vm := NewRandomVirtualMachine(vmi, false)

	libstorage.AddDataVolumeTemplate(vm, dataVolume)
	return vm
}

func NewRandomVMWithDataVolumeAndUserDataInStorageClass(imageUrl, namespace, userData, storageClass string) *v1.VirtualMachine {
	dataVolume := libstorage.NewDataVolumeWithRegistryImportInStorageClass(imageUrl, namespace, storageClass, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
	return NewRandomVMWithDataVolumeAndUserData(dataVolume, userData)
}

func NewRandomVMWithCloneDataVolume(sourceNamespace, sourceName, targetNamespace string) *v1.VirtualMachine {
	dataVolume := libstorage.NewDataVolumeWithPVCSource(sourceNamespace, sourceName, targetNamespace, k8sv1.ReadWriteOnce)
	vmi := NewRandomVMIWithDataVolume(dataVolume.Name)
	vmi.Namespace = targetNamespace
	vm := NewRandomVirtualMachine(vmi, false)

	libstorage.AddDataVolumeTemplate(vm, dataVolume)
	return vm
}

func NewRandomVMIWithEphemeralDiskHighMemory(containerImage string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, userData)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataHighMemory(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(containerImage, userData)

	vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512M")
	return vmi
}

func NewRandomMigration(vmiName string, namespace string) *v1.VirtualMachineInstanceMigration {
	return &v1.VirtualMachineInstanceMigration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstanceMigration",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "test-migration-",
			Namespace:    namespace,
		},
		Spec: v1.VirtualMachineInstanceMigrationSpec{
			VMIName: vmiName,
		},
	}
}

func NewRandomVMIWithEphemeralDisk(containerImage string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	AddEphemeralDisk(vmi, "disk0", v1.DiskBusVirtio, containerImage)
	if containerImage == cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling) {
		vmi.Spec.Domain.Devices.Rng = &v1.Rng{} // newer fedora kernels may require hardware RNG to boot
	}
	return vmi
}

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

func AddBootOrderToDisk(vmi *v1.VirtualMachineInstance, diskName string, bootorder *uint) *v1.VirtualMachineInstance {
	for i, d := range vmi.Spec.Domain.Devices.Disks {
		if d.Name == diskName {
			vmi.Spec.Domain.Devices.Disks[i].BootOrder = bootorder
			return vmi
		}
	}
	return vmi
}

func AddPVCDisk(vmi *v1.VirtualMachineInstance, name string, bus v1.DiskBus, claimName string) *v1.VirtualMachineInstance {
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
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		},
	})

	return vmi
}

func AddEphemeralCdrom(vmi *v1.VirtualMachineInstance, name string, bus v1.DiskBus, image string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			CDRom: &v1.CDRomTarget{
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

func NewRandomFedoraVMI() *v1.VirtualMachineInstance {
	networkData, err := libnet.CreateDefaultCloudInitNetworkData()
	Expect(err).NotTo(HaveOccurred())

	return libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloudNetworkData(networkData),
	)
}

func NewRandomFedoraVMIWithGuestAgent() *v1.VirtualMachineInstance {
	networkData, err := libnet.CreateDefaultCloudInitNetworkData()
	Expect(err).NotTo(HaveOccurred())

	return libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloudNetworkData(networkData),
	)
}

func NewRandomFedoraVMIWithBlacklistGuestAgent(commands string) *v1.VirtualMachineInstance {
	networkData, err := libnet.CreateDefaultCloudInitNetworkData()
	Expect(err).NotTo(HaveOccurred())

	return libvmi.NewFedora(
		libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithCloudInitNoCloudUserData(GetFedoraToolsGuestAgentBlacklistUserData(commands), false),
		libvmi.WithCloudInitNoCloudNetworkData(networkData),
	)
}

func AddPVCFS(vmi *v1.VirtualMachineInstance, name string, claimName string) *v1.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
		Name:     name,
		Virtiofs: &v1.FilesystemVirtiofs{},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		},
	})

	return vmi
}

func NewRandomVMIWithFSFromDataVolume(dataVolumeName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()
	containerImage := cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)
	AddEphemeralDisk(vmi, "disk0", v1.DiskBusVirtio, containerImage)
	vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, v1.Filesystem{
		Name:     "disk1",
		Virtiofs: &v1.FilesystemVirtiofs{},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk1",
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	})
	return vmi
}

func NewRandomVMIWithPVCFS(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	containerImage := cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)
	AddEphemeralDisk(vmi, "disk0", v1.DiskBusVirtio, containerImage)
	vmi = AddPVCFS(vmi, "disk1", claimName)
	return vmi
}

func NewRandomFedoraVMIWithDmidecode() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling))
	return vmi
}

func NewRandomFedoraVMIWithVirtWhatCpuidHelper() *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDiskHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling))
	return vmi
}

func GetFedoraToolsGuestAgentBlacklistUserData(commands string) string {
	return fmt.Sprintf(`#!/bin/bash
            echo -e "\n\nBLACKLIST_RPC=%s" | sudo tee -a /etc/sysconfig/qemu-ga
`, commands)
}

func NewRandomVMIWithEphemeralDiskAndUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddUserData(vmi, "disk1", userData)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitConfigDriveData(vmi, "disk1", userData, "", false)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitNoCloudData(vmi, "disk1", userData, networkData, b64encode)
	return vmi
}

func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	AddCloudInitConfigDriveData(vmi, "disk1", userData, networkData, b64encode)
	return vmi
}

func AddUserData(vmi *v1.VirtualMachineInstance, name string, userData string) {
	AddCloudInitNoCloudData(vmi, name, userData, "", true)
}

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

func NewRandomVMIWithPVC(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi = AddPVCDisk(vmi, "disk0", v1.DiskBusVirtio, claimName)
	return vmi
}

func NewRandomVMIWithPVCAndUserData(claimName, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi = AddPVCDisk(vmi, "disk0", v1.DiskBusVirtio, claimName)
	AddUserData(vmi, "disk1", userData)
	return vmi
}

func DeletePvAndPvc(name string) {
	virtCli, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	err = virtCli.CoreV1().PersistentVolumes().Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util2.PanicOnError(err)
	}

	err = virtCli.CoreV1().PersistentVolumeClaims(util2.NamespaceTestDefault).Delete(context.Background(), name, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		util2.PanicOnError(err)
	}
}

func NewRandomVMIWithCDRom(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "disk0",
		DiskDevice: v1.DiskDevice{
			CDRom: &v1.CDRomTarget{
				// Do not specify ReadOnly flag so that
				// default behavior can be tested
				Bus: v1.DiskBusSATA,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk0",
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
				ClaimName: claimName,
			}},
		},
	})
	return vmi
}

func NewRandomVMIWithEphemeralPVC(claimName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: "disk0",
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusSATA,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "disk0",

		VolumeSource: v1.VolumeSource{
			Ephemeral: &v1.EphemeralVolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		},
	})
	return vmi
}

func AddHostDisk(vmi *v1.VirtualMachineInstance, path string, diskType v1.HostDiskType, name string) {
	hostDisk := v1.HostDisk{
		Path: path,
		Type: diskType,
	}
	if diskType == v1.HostDiskExistsOrCreate {
		hostDisk.Capacity = resource.MustParse(defaultDiskSize)
	}

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusVirtio,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			HostDisk: &hostDisk,
		},
	})
}

func NewRandomVMIWithHostDisk(diskPath string, diskType v1.HostDiskType, nodeName string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMI()
	AddHostDisk(vmi, diskPath, diskType, "host-disk")
	if nodeName != "" {
		vmi.Spec.Affinity = &k8sv1.Affinity{
			NodeAffinity: &k8sv1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
					NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
						{
							MatchExpressions: []k8sv1.NodeSelectorRequirement{
								{
									Key:      util2.KubernetesIoHostName,
									Operator: k8sv1.NodeSelectorOpIn,
									Values:   []string{nodeName},
								},
							},
						},
					},
				},
			},
		}
	}
	return vmi
}

func AddConfigMapDisk(vmi *v1.VirtualMachineInstance, configMapName string, volumeName string) {
	AddConfigMapDiskWithCustomLabel(vmi, configMapName, volumeName, "")

}
func AddConfigMapDiskWithCustomLabel(vmi *v1.VirtualMachineInstance, configMapName string, volumeName string, volumeLabel string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: k8sv1.LocalObjectReference{
					Name: configMapName,
				},
				VolumeLabel: volumeLabel,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: volumeName,
	})
}

func AddSecretDisk(vmi *v1.VirtualMachineInstance, secretName string, volumeName string) {
	AddSecretDiskWithCustomLabel(vmi, secretName, volumeName, "")
}

func AddSecretDiskWithCustomLabel(vmi *v1.VirtualMachineInstance, secretName string, volumeName string, volumeLabel string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			Secret: &v1.SecretVolumeSource{
				SecretName:  secretName,
				VolumeLabel: volumeLabel,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: volumeName,
	})
}

func AddLabelDownwardAPIVolume(vmi *v1.VirtualMachineInstance, volumeName string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			DownwardAPI: &v1.DownwardAPIVolumeSource{
				Fields: []k8sv1.DownwardAPIVolumeFile{
					{
						Path: "labels",
						FieldRef: &k8sv1.ObjectFieldSelector{
							FieldPath: "metadata.labels",
						},
					},
				},
				VolumeLabel: "",
			},
		},
	})

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: volumeName,
	})
}

func AddDownwardMetricsVolume(vmi *v1.VirtualMachineInstance, volumeName string) {
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			DownwardMetrics: &v1.DownwardMetricsVolumeSource{},
		}})

	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: volumeName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusVirtio,
			},
		},
	})
}

func AddServiceAccountDisk(vmi *v1.VirtualMachineInstance, serviceAccountName string) {
	volumeName := serviceAccountName + "-disk"
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			ServiceAccount: &v1.ServiceAccountVolumeSource{
				ServiceAccountName: serviceAccountName,
			},
		},
	})
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: serviceAccountName + "-disk",
	})
}

func AddExplicitPodNetworkInterface(vmi *v1.VirtualMachineInstance) {
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
}

// WaitForVMIStartOrFailed blocks until the specified VirtualMachineInstance reached either Failed or Running states
func WaitForVMIStartOrFailed(obj runtime.Object, seconds int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return waitForVMIPhase(ctx, []v1.VirtualMachineInstancePhase{v1.Running, v1.Failed}, obj, seconds, wp, true)
}

// Block until the specified VirtualMachineInstance started and return the target node name.
func waitForVMIStart(ctx context.Context, obj runtime.Object, seconds int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	return waitForVMIPhase(ctx, []v1.VirtualMachineInstancePhase{v1.Running}, obj, seconds, wp, false)
}

// Block until the specified VirtualMachineInstance scheduled and return the target node name.
func waitForVMIScheduling(obj runtime.Object, seconds int, wp watcher.WarningsPolicy) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return waitForVMIPhase(ctx, []v1.VirtualMachineInstancePhase{v1.Scheduling, v1.Scheduled, v1.Running}, obj, seconds, wp, false)
}

func waitForVMIPhase(ctx context.Context, phases []v1.VirtualMachineInstancePhase, obj runtime.Object, seconds int, wp watcher.WarningsPolicy, waitForFail bool) *v1.VirtualMachineInstance {
	vmi, ok := obj.(*v1.VirtualMachineInstance)
	ExpectWithOffset(1, ok).To(BeTrue(), "Object is not of type *v1.VMI")

	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Fetch the VirtualMachineInstance, to make sure we have a resourceVersion as a starting point for the watch
	// FIXME: This may start watching too late and we may miss some warnings
	if vmi.ResourceVersion == "" {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
	}

	objectEventWatcher := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(time.Duration(seconds+2) * time.Second)
	if wp.FailOnWarnings == true {
		objectEventWatcher.SetWarningsPolicy(wp)
	}

	go func() {
		defer GinkgoRecover()
		objectEventWatcher.WaitFor(ctx, watcher.NormalEvent, v1.Started)
	}()

	timeoutMsg := fmt.Sprintf("Timed out waiting for VMI %s to enter %s phase(s)", vmi.Name, phases)
	// FIXME the event order is wrong. First the document should be updated
	EventuallyWithOffset(1, func() v1.VirtualMachineInstancePhase {
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())

		Expect(vmi.Status.Phase == v1.Succeeded).To(BeFalse(), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		// May need to wait for Failed state
		if !waitForFail {
			Expect(vmi.Status.Phase == v1.Failed).To(BeFalse(), "VMI %s unexpectedly stopped. State: %s", vmi.Name, vmi.Status.Phase)
		}
		return vmi.Status.Phase
	}, time.Duration(seconds)*time.Second, 1*time.Second).Should(BeElementOf(phases), timeoutMsg)

	return vmi
}

func WaitForSuccessfulVMIStartIgnoreWarnings(vmi runtime.Object) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp := watcher.WarningsPolicy{FailOnWarnings: false}
	return waitForVMIStart(ctx, vmi, 180, wp)
}

func WaitForSuccessfulVMIStartWithTimeout(vmi runtime.Object, seconds int) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return waitForVMIStart(ctx, vmi, seconds, wp)
}

func WaitForSuccessfulVMIStartWithTimeoutIgnoreWarnings(vmi runtime.Object, seconds int) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wp := watcher.WarningsPolicy{FailOnWarnings: false}
	return waitForVMIStart(ctx, vmi, seconds, wp)
}

func WaitForVirtualMachineToDisappearWithTimeout(vmi *v1.VirtualMachineInstance, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() error {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		return err
	}, seconds, 1*time.Second).Should(SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())), "The VMI should be gone within the given timeout")
}

func WaitForMigrationToDisappearWithTimeout(migration *v1.VirtualMachineInstanceMigration, seconds int) {
	virtClient, err := kubecli.GetKubevirtClient()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	EventuallyWithOffset(1, func() bool {
		_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
		return errors.IsNotFound(err)
	}, seconds, 1*time.Second).Should(BeTrue(), fmt.Sprintf("migration %s was expected to dissapear after %d seconds, but it did not", migration.Name, seconds))
}

func WaitForSuccessfulVMIStart(vmi runtime.Object) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return WaitForSuccessfulVMIStartWithContext(ctx, vmi)
}

func WaitForSuccessfulVMIStartWithContext(ctx context.Context, vmi runtime.Object) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true}
	return waitForVMIStart(ctx, vmi, 360, wp)
}

func WaitForSuccessfulVMIStartWithContextIgnoreSelectedWarnings(ctx context.Context, vmi runtime.Object, warningsIgnoreList []string) *v1.VirtualMachineInstance {
	wp := watcher.WarningsPolicy{FailOnWarnings: true, WarningsIgnoreList: warningsIgnoreList}
	return waitForVMIStart(ctx, vmi, 360, wp)
}

func WaitUntilVMIReady(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return WaitUntilVMIReadyWithContext(ctx, vmi, loginTo)
}

func WaitUntilVMIReadyIgnoreSelectedWarnings(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction, warningsIgnoreList []string) *v1.VirtualMachineInstance {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return WaitUntilVMIReadyWithContextIgnoreSelectedWarnings(ctx, vmi, loginTo, warningsIgnoreList)
}

func WaitUntilVMIReadyWithContext(ctx context.Context, vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	WaitForSuccessfulVMIStartWithContext(ctx, vmi)
	return LoginToVM(vmi, loginTo)
}

func WaitUntilVMIReadyWithContextIgnoreSelectedWarnings(ctx context.Context, vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction, warningsIgnoreList []string) *v1.VirtualMachineInstance {
	// Wait for VirtualMachineInstance start
	WaitForSuccessfulVMIStartWithContextIgnoreSelectedWarnings(ctx, vmi, warningsIgnoreList)
	return LoginToVM(vmi, loginTo)
}

func LoginToVM(vmi *v1.VirtualMachineInstance, loginTo console.LoginToFunction) *v1.VirtualMachineInstance {
	// Fetch the new VirtualMachineInstance with updated status
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	// Lets make sure that the OS is up by waiting until we can login

	ExpectWithOffset(1, loginTo(vmi)).To(Succeed())

	return vmi
}

func NewInt32(x int32) *int32 {
	return &x
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

func NewBool(x bool) *bool {
	return &x
}

func RenderPrivilegedPod(name string, cmd []string, args []string) *k8sv1.Pod {
	pod := RenderPod(name, cmd, args)
	pod.Spec.HostPID = true
	pod.Spec.SecurityContext = &k8sv1.PodSecurityContext{
		RunAsUser: new(int64),
	}
	pod.Spec.Containers = []k8sv1.Container{
		renderPrivilegedContainerSpec(
			fmt.Sprintf("%s/vm-killer:%s", flags.KubeVirtUtilityRepoPrefix, flags.KubeVirtUtilityVersionTag),
			name,
			cmd,
			args),
	}

	return pod
}

func RenderPod(name string, cmd []string, args []string) *k8sv1.Pod {
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Namespace:    util2.NamespaceTestDefault,
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				renderContainerSpec(
					fmt.Sprintf("%s/vm-killer:%s", flags.KubeVirtUtilityRepoPrefix, flags.KubeVirtUtilityVersionTag),
					name,
					cmd,
					args),
			},
		},
	}

	return &pod
}

// CreateExecutorPodWithPVC creates a Pod with the passed in PVC mounted under /pvc. You can then use the executor utilities to
// run commands against the PVC through this Pod.
func CreateExecutorPodWithPVC(podName string, pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
	return RunPod(newExecutorPodWithPVC(podName, pvc))
}

func newExecutorPodWithPVC(podName string, pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
	return libstorage.RenderPodWithPVC(podName, []string{"/bin/bash", "-c", "while true; do echo hello; sleep 2;done"}, nil, pvc)
}

func RunPod(pod *k8sv1.Pod) *k8sv1.Pod {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	pod, err = virtClient.CoreV1().Pods(util2.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(ThisPod(pod), 180).Should(BeInPhase(k8sv1.PodRunning))

	pod, err = ThisPod(pod)()
	Expect(err).ToNot(HaveOccurred())
	return pod
}

func RunPodAndExpectCompletion(pod *k8sv1.Pod) *k8sv1.Pod {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	pod, err = virtClient.CoreV1().Pods(util2.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
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

	RunPodAndExpectCompletion(pod)
}

func CopyAlpineWithNonQEMUPermissions() (dstPath, nodeName string) {
	dstPath = testsuite.HostPathAlpine + "-nopriv"
	args := []string{fmt.Sprintf(`mkdir -p %[1]s-nopriv && cp %[1]s/disk.img %[1]s-nopriv/ && chmod 640 %[1]s-nopriv/disk.img  && chown root:root %[1]s-nopriv/disk.img`, testsuite.HostPathAlpine)}

	By("creating an image with without qemu permissions")
	pod := RenderHostPathPod("tmp-image-create-job", testsuite.HostPathBase, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationNone, []string{BinBash, "-c"}, args)

	nodeName = RunPodAndExpectCompletion(pod).Spec.NodeName
	return
}

func DeleteAlpineWithNonQEMUPermissions() {
	nonQemuAlpinePath := testsuite.HostPathAlpine + "-nopriv"
	args := []string{fmt.Sprintf(`rm -rf %s`, nonQemuAlpinePath)}

	pod := RenderHostPathPod("remove-tmp-image-job", testsuite.HostPathBase, k8sv1.HostPathDirectoryOrCreate, k8sv1.MountPropagationNone, []string{BinBash, "-c"}, args)

	RunPodAndExpectCompletion(pod)
}

func renderContainerSpec(imgPath string, name string, cmd []string, args []string) k8sv1.Container {
	return k8sv1.Container{
		Name:    name,
		Image:   imgPath,
		Command: cmd,
		Args:    args,
	}
}

func renderPrivilegedContainerSpec(imgPath string, name string, cmd []string, args []string) k8sv1.Container {
	return k8sv1.Container{
		Name:    name,
		Image:   imgPath,
		Command: cmd,
		Args:    args,
		SecurityContext: &k8sv1.SecurityContext{
			Privileged: NewBool(true),
			RunAsUser:  new(int64),
		},
	}
}

func ExecuteCommandOnPod(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string) (string, error) {
	stdout, stderr, err := ExecuteCommandOnPodV2(virtCli, pod, containerName, command)

	if err != nil {
		return "", fmt.Errorf("failed executing command on pod: %v: stderr %v: stdout: %v", err, stderr, stdout)
	}

	if len(stderr) > 0 {
		return "", fmt.Errorf("stderr: %v", stderr)
	}

	return stdout, nil
}

func CopyFromPod(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName, sourceFile, targetFile string) (stderr string, err error) {
	var (
		stderrBuf bytes.Buffer
	)
	file, err := os.Create(targetFile)
	Expect(err).ToNot(HaveOccurred())
	defer file.Close()

	options := remotecommand.StreamOptions{
		Stdout: file,
		Stderr: &stderrBuf,
		Tty:    false,
	}
	err = execCommandOnPod(virtCli, pod, containerName, []string{"cat", sourceFile}, options)
	return stderrBuf.String(), err
}

func ExecuteCommandOnPodV2(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string) (stdout, stderr string, err error) {
	var (
		stdoutBuf bytes.Buffer
		stderrBuf bytes.Buffer
	)
	options := remotecommand.StreamOptions{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Tty:    false,
	}
	err = execCommandOnPod(virtCli, pod, containerName, command, options)
	return stdoutBuf.String(), stderrBuf.String(), err
}

func execCommandOnPod(virtCli kubecli.KubevirtClient, pod *k8sv1.Pod, containerName string, command []string, options remotecommand.StreamOptions) error {

	req := virtCli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", containerName)

	req.VersionedParams(&k8sv1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	virtConfig, err := kubecli.GetKubevirtClientConfig()
	if err != nil {
		return err
	}

	executor, err := remotecommand.NewSPDYExecutor(virtConfig, "POST", req.URL())
	if err != nil {
		return err
	}

	return executor.Stream(options)
}

func GetRunningVirtualMachineInstanceDomainXML(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (string, error) {
	vmiPod, err := getRunningPodByVirtualMachineInstance(vmi, util2.NamespaceTestDefault)
	if err != nil {
		return "", err
	}

	// get current vmi
	freshVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get vmi, %s", err)
	}

	command := []string{"virsh"}
	if kutil.IsNonRootVMI(freshVMI) {
		command = append(command, "-c")
		command = append(command, "qemu+unix:///session?socket=/var/run/libvirt/libvirt-sock")
	}
	command = append(command, []string{"dumpxml", vmi.Namespace + "_" + vmi.Name}...)

	stdout, stderr, err := ExecuteCommandOnPodV2(
		virtClient,
		vmiPod,
		GetComputeContainerOfPod(vmiPod).Name,
		command,
	)
	if err != nil {
		return "", fmt.Errorf("could not dump libvirt domxml (remotely on pod): %v: %s", err, stderr)
	}
	return stdout, err
}

func LibvirtDomainIsPaused(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (bool, error) {
	vmiPod, err := getRunningPodByVirtualMachineInstance(vmi, util2.NamespaceTestDefault)
	if err != nil {
		return false, err
	}

	stdout, stderr, err := ExecuteCommandOnPodV2(
		virtClient,
		vmiPod,
		GetComputeContainerOfPod(vmiPod).Name,
		[]string{"virsh", "--quiet", "domstate", vmi.Namespace + "_" + vmi.Name},
	)
	if err != nil {
		return false, fmt.Errorf("could not get libvirt domstate (remotely on pod): %v: %s", err, stderr)
	}
	return strings.Contains(stdout, "paused"), nil
}

// Deprecated: DeprecatedBeforeAll must not be used. Tests need to be self-contained to allow sane cleanup, accurate reporting and
// parallel execution.
func DeprecatedBeforeAll(fn func()) {
	first := true
	BeforeEach(func() {
		if first {
			fn()
			first = false
		}
	})
}

func GenerateVMJson(vm *v1.VirtualMachine, generateDirectory string) (string, error) {
	data, err := json.Marshal(vm)
	if err != nil {
		return "", fmt.Errorf("failed to generate json for vm %s", vm.Name)
	}

	jsonFile := filepath.Join(generateDirectory, fmt.Sprintf("%s.json", vm.Name))
	err = ioutil.WriteFile(jsonFile, data, 0644)
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

func Running(vmis *v1.VirtualMachineInstanceList) (running []v1.VirtualMachineInstance) {
	for _, vmi := range vmis.Items {
		if vmi.DeletionTimestamp == nil && vmi.Status.Phase == v1.Running {
			running = append(running, vmi)
		}
	}
	return
}

func UnfinishedVMIPodSelector(vmi *v1.VirtualMachineInstance) metav1.ListOptions {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
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
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)
	procPath := filepath.Join("/proc/1/root", diskPath)
	virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
	Expect(err).ToNot(HaveOccurred())
	_, _, err = ExecuteCommandOnPodV2(virtClient, virtHandlerPod, "virt-handler", []string{"rm", "-rf", procPath})
	Expect(err).ToNot(HaveOccurred())
}

func CreateHostDiskImage(diskPath string) *k8sv1.Pod {
	hostPathType := k8sv1.HostPathDirectoryOrCreate
	dir := filepath.Dir(diskPath)

	command := fmt.Sprintf(`dd if=/dev/zero of=%s bs=1 count=0 seek=1G && ls -l %s`, diskPath, dir)
	if checks.HasFeature(virtconfig.NonRoot) {
		command = command + fmt.Sprintf(" && chown 107:107 %s", diskPath)
	}

	args := []string{command}
	pod := RenderHostPathPod("hostdisk-create-job", dir, hostPathType, k8sv1.MountPropagationNone, []string{BinBash, "-c"}, args)

	return pod
}

func RenderHostPathPod(podName string, dir string, hostPathType k8sv1.HostPathType, mountPropagation k8sv1.MountPropagationMode, cmd []string, args []string) *k8sv1.Pod {
	pod := RenderPrivilegedPod(podName, cmd, args)
	pod.Spec.Containers[0].VolumeMounts = append(pod.Spec.Containers[0].VolumeMounts, k8sv1.VolumeMount{
		Name:             "hostpath-mount",
		MountPropagation: &mountPropagation,
		MountPath:        dir,
	})
	pod.Spec.Volumes = append(pod.Spec.Volumes, k8sv1.Volume{
		Name: "hostpath-mount",
		VolumeSource: k8sv1.VolumeSource{
			HostPath: &k8sv1.HostPathVolumeSource{
				Path: dir,
				Type: &hostPathType,
			},
		},
	})

	return pod
}

// CreateVmiOnNodeLabeled creates a VMI a node that has a give label set to a given value
func CreateVmiOnNodeLabeled(vmi *v1.VirtualMachineInstance, nodeLabel, labelValue string) *v1.VirtualMachineInstance {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	vmi.Spec.Affinity = &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{Key: nodeLabel, Operator: k8sv1.NodeSelectorOpIn, Values: []string{labelValue}},
						},
					},
				},
			},
		},
	}

	vmi, err = virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Create(vmi)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return vmi
}

// CreateVmiOnNode creates a VMI on the specified node
func CreateVmiOnNode(vmi *v1.VirtualMachineInstance, nodeName string) *v1.VirtualMachineInstance {
	return CreateVmiOnNodeLabeled(vmi, util2.KubernetesIoHostName, nodeName)
}

// RunCommandOnVmiPod runs specified command on the virt-launcher pod
func RunCommandOnVmiPod(vmi *v1.VirtualMachineInstance, command []string) string {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	pods, err := virtClient.CoreV1().Pods(util2.NamespaceTestDefault).List(context.Background(), UnfinishedVMIPodSelector(vmi))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pods.Items).NotTo(BeEmpty())
	vmiPod := pods.Items[0]

	output, err := ExecuteCommandOnPod(
		virtClient,
		&vmiPod,
		"compute",
		command,
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return output
}

// RunCommandOnVmiTargetPod runs specified command on the target virt-launcher pod of a migration
func RunCommandOnVmiTargetPod(vmi *v1.VirtualMachineInstance, command []string) (string, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	pods, err := virtClient.CoreV1().Pods(util2.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pods.Items).NotTo(BeEmpty())
	var vmiPod *k8sv1.Pod
	for _, pod := range pods.Items {
		if pod.Name == vmi.Status.MigrationState.TargetPod {
			vmiPod = &pod
			break
		}
	}
	if vmiPod == nil {
		return "", fmt.Errorf("failed to find migration target pod")
	}

	output, err := ExecuteCommandOnPod(
		virtClient,
		vmiPod,
		"compute",
		command,
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return output, nil
}

func NewRandomVirtualMachine(vmi *v1.VirtualMachineInstance, running bool) *v1.VirtualMachine {
	name := vmi.Name
	namespace := vmi.Namespace
	vmLabels := map[string]string{"name": name}
	for k, v := range vmi.Labels {
		vmLabels[k] = v
	}
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.VirtualMachineSpec{
			Running: &running,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    vmLabels,
					Name:      name + "makeitinteresting", // this name should have no effect
					Namespace: namespace,
				},
				Spec: vmi.Spec,
			},
		},
	}
	vm.SetGroupVersionKind(schema.GroupVersionKind{Group: v1.GroupVersion.Group, Kind: "VirtualMachine", Version: v1.GroupVersion.Version})
	return vm
}

func StopVirtualMachineWithTimeout(vm *v1.VirtualMachine, timeout time.Duration) *v1.VirtualMachine {
	By("Stopping the VirtualMachineInstance")
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)
	running := false
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = &running
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
		return err
	}, timeout, 1*time.Second).ShouldNot(HaveOccurred())
	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	// Observe the VirtualMachineInstance deleted
	Eventually(func() bool {
		_, err = virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return true
		}
		return false
	}, timeout, 1*time.Second).Should(BeTrue(), "The vmi did not disappear")
	By("VM has not the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
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
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)
	running := true
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = &running
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
		return err
	}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	// Observe the VirtualMachineInstance created
	Eventually(func() error {
		_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		return err
	}, 300*time.Second, 1*time.Second).Should(Succeed())
	By("VMI has the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeTrue())
	return updatedVM
}

func DisableFeatureGate(feature string) {
	if !checks.HasFeature(feature) {
		return
	}
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

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

	UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
}

func EnableFeatureGate(feature string) *v1.KubeVirt {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())

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

	return UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
}

func SetDataVolumeForceBindAnnotation(dv *cdiv1.DataVolume) {
	annotations := dv.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["cdi.kubevirt.io/storage.bind.immediate.requested"] = "true"
	dv.SetAnnotations(annotations)
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

func AppendEmptyDisk(vmi *v1.VirtualMachineInstance, diskName string, busName v1.DiskBus, diskSize string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name: diskName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: busName,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: diskName,
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: resource.MustParse(diskSize),
			},
		},
	})
}

func GetRunningVMIDomainSpec(vmi *v1.VirtualMachineInstance) (*launcherApi.DomainSpec, error) {
	runningVMISpec := launcherApi.DomainSpec{}
	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	domXML, err := GetRunningVirtualMachineInstanceDomainXML(cli, vmi)
	if err != nil {
		return nil, err
	}

	err = xml.Unmarshal([]byte(domXML), &runningVMISpec)
	return &runningVMISpec, err
}

func ForwardPorts(pod *k8sv1.Pod, ports []string, stop chan struct{}, readyTimeout time.Duration) error {
	errChan := make(chan error, 1)
	readyChan := make(chan struct{})
	go func() {
		cli, err := kubecli.GetKubevirtClient()
		if err != nil {
			errChan <- err
			return
		}

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

// UpdateKubeVirtConfigValueAndWait updates the given configuration in the kubevirt custom resource
// and then waits  to allow the configuration events to be propagated to the consumers.
func UpdateKubeVirtConfigValueAndWait(kvConfig v1.KubeVirtConfiguration) *v1.KubeVirt {
	kv := testsuite.UpdateKubeVirtConfigValue(kvConfig)

	waitForConfigToBePropagated(kv.ResourceVersion)
	log.DefaultLogger().Infof("system is in sync with kubevirt config resource version %s", kv.ResourceVersion)

	return kv
}

// resetToDefaultConfig resets the config to the state found when the test suite started. It will wait for the config to
// be propagated to all components before it returns. It will only update the configuration and wait for it to be
// propagated if the current config in use does not match the original one.
func resetToDefaultConfig() {
	suiteConfig, _ := GinkgoConfiguration()
	if suiteConfig.ParallelTotal > 1 {
		// Tests which alter the global kubevirt config must be run serial, therefor, if we run in parallel
		// we can just skip the restore step.
		return
	}

	UpdateKubeVirtConfigValueAndWait(testsuite.KubeVirtDefaultConfig)
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
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

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

func WaitAgentConnected(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	WaitForVMICondition(virtClient, vmi, v1.VirtualMachineInstanceAgentConnected, 12*60)
}

func WaitAgentDisconnected(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) {
	WaitForVMIConditionRemovedOrFalse(virtClient, vmi, v1.VirtualMachineInstanceAgentConnected, 30)
}

func WaitForVMICondition(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVmi, err := virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVmi.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeTrue(), fmt.Sprintf("Should have %s condition", conditionType))
}

func WaitForVMIConditionRemovedOrFalse(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition removed or false", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVmi, err := virtClient.VirtualMachineInstance(util2.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVmi.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeFalse(), fmt.Sprintf("Should have no or false %s condition", conditionType))
}

func WaitForVMCondition(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, conditionType v1.VirtualMachineConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVm, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVm.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeTrue(), fmt.Sprintf("Should have %s condition", conditionType))
}

func WaitForVMConditionRemovedOrFalse(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, conditionType v1.VirtualMachineConditionType, timeoutSec int) {
	By(fmt.Sprintf("Waiting for %s condition removed or false", conditionType))
	EventuallyWithOffset(1, func() bool {
		updatedVm, err := virtClient.VirtualMachine(util2.NamespaceTestDefault).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		for _, condition := range updatedVm.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8sv1.ConditionTrue {
				return true
			}
		}
		return false
	}, time.Duration(timeoutSec)*time.Second, 2).Should(BeFalse(), fmt.Sprintf("Should have no or false %s condition", conditionType))
}

// GeneratePrivateKey creates a RSA Private Key of specified byte size
func GeneratePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// GeneratePublicKey will return in the format "ssh-rsa ..."
func GeneratePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return publicKeyBytes, nil
}

// EncodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func EncodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privateBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privateBlock)

	return privatePEM
}

func PodReady(pod *k8sv1.Pod) k8sv1.ConditionStatus {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == k8sv1.PodReady {
			return cond.Status
		}
	}
	return k8sv1.ConditionFalse
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
	err := ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 10*time.Second)
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
	return ioutil.ReadAll(resp.Body)
}

// GetCertsForPods returns the used certificates for all pods matching  the label selector
func GetCertsForPods(labelSelector string, namespace string, port string) ([][]byte, error) {
	cli, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
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

func GetCertFromSecret(secretName string) []byte {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if rawBundle, ok := secret.Data[bootstrap.CertBytesValue]; ok {
		return rawBundle
	}
	return nil
}

func GetBundleFromConfigMap(configMapName string) ([]byte, []*x509.Certificate) {
	virtClient, err := kubecli.GetKubevirtClient()
	Expect(err).ToNot(HaveOccurred())
	configMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), configMapName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if rawBundle, ok := configMap.Data[components.CABundleKey]; ok {
		crts, err := cert.ParseCertsPEM([]byte(rawBundle))
		Expect(err).ToNot(HaveOccurred())
		return []byte(rawBundle), crts
	}
	return nil, nil
}

func FormatIPForURL(ip string) string {
	if netutils.IsIPv6String(ip) {
		return "[" + ip + "]"
	}
	return ip
}

func IsRunningOnKindInfra() bool {
	provider := os.Getenv("KUBEVIRT_PROVIDER")
	return strings.HasPrefix(provider, "kind")
}

func RandTmpDir() string {
	const tmpPath = "/var/provision/kubevirt.io/tests"
	return filepath.Join(tmpPath, rand.String(10))
}

// VMILauncherIgnoreWarnings waiting for the VMI to be up but ignoring warnings like a disconnected guest-agent
func VMILauncherIgnoreWarnings(virtClient kubecli.KubevirtClient) func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
	return func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
		By(StartingVMInstance)
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(util2.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
		Expect(err).ToNot(HaveOccurred())

		By("Waiting the VirtualMachineInstance start")
		vmi, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		// Warnings are okay. We'll receive a warning that the agent isn't connected
		// during bootup, but that is transient
		Expect(WaitForSuccessfulVMIStartIgnoreWarnings(obj).Status.NodeName).ToNot(BeEmpty())
		return vmi
	}
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

func DryRunCreate(client *rest.RESTClient, resource, namespace string, obj interface{}, result runtime.Object) error {
	opts := metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}}
	return client.Post().
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(obj).
		Do(context.Background()).
		Into(result)
}

func DryRunUpdate(client *rest.RESTClient, resource, name, namespace string, obj interface{}, result runtime.Object) error {
	opts := metav1.UpdateOptions{DryRun: []string{metav1.DryRunAll}}
	return client.Put().
		Name(name).
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(obj).
		Do(context.Background()).
		Into(result)
}

func DryRunPatch(client *rest.RESTClient, resource, name, namespace string, pt types.PatchType, data []byte, result runtime.Object) error {
	opts := metav1.PatchOptions{DryRun: []string{metav1.DryRunAll}}
	return client.Patch(pt).
		Name(name).
		Namespace(namespace).
		Resource(resource).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(context.Background()).
		Into(result)
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

func GetPolicyMatchedToVmi(name string, vmi *v1.VirtualMachineInstance, namespace *k8sv1.Namespace, matchingVmiLabels, matchingNSLabels int) *migrationsv1.MigrationPolicy {
	Expect(vmi).ToNot(BeNil())
	Expect(namespace).ToNot(BeNil())
	Expect(name).ToNot(BeEmpty())

	policy := kubecli.NewMinimalMigrationPolicy(name)
	if policy.Labels == nil {
		policy.Labels = map[string]string{}
	}
	policy.Labels[cleanup.TestLabelForNamespace(util2.NamespaceTestDefault)] = ""

	if vmi.Labels == nil {
		vmi.Labels = make(map[string]string)
	}
	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}

	if policy.Spec.Selectors == nil {
		policy.Spec.Selectors = &migrationsv1.Selectors{
			VirtualMachineInstanceSelector: migrationsv1.LabelSelector{}, //&metav1.LabelSelector{MatchLabels: map[string]string{}},
			NamespaceSelector:              migrationsv1.LabelSelector{},
		}
	} else if policy.Spec.Selectors.VirtualMachineInstanceSelector == nil {
		policy.Spec.Selectors.VirtualMachineInstanceSelector = migrationsv1.LabelSelector{}
	} else if policy.Spec.Selectors.NamespaceSelector == nil {
		policy.Spec.Selectors.NamespaceSelector = migrationsv1.LabelSelector{}
	}

	labelKeyPattern := "mp-key-%d"
	labelValuePattern := "mp-value-%d"

	applyLabels := func(policyLabels, vmiOrNSLabels map[string]string, labelCount int) {
		for i := 0; i < labelCount; i++ {
			labelKey := fmt.Sprintf(labelKeyPattern, i)
			labelValue := fmt.Sprintf(labelValuePattern, i)

			vmiOrNSLabels[labelKey] = labelValue
			policyLabels[labelKey] = labelValue
		}
	}

	applyLabels(policy.Spec.Selectors.VirtualMachineInstanceSelector, vmi.Labels, matchingVmiLabels)
	applyLabels(policy.Spec.Selectors.NamespaceSelector, namespace.Labels, matchingNSLabels)

	return policy
}

func GetIdOfLauncher(vmi *v1.VirtualMachineInstance) string {
	virtClient, err := kubecli.GetKubevirtClient()
	util2.PanicOnError(err)

	vmiPod := GetRunningPodByVirtualMachineInstance(vmi, util2.NamespaceTestDefault)
	podOutput, err := ExecuteCommandOnPod(
		virtClient,
		vmiPod,
		vmiPod.Spec.Containers[0].Name,
		[]string{"id", "-u"},
	)
	Expect(err).NotTo(HaveOccurred())

	return strings.TrimSpace(podOutput)
}

func ExecuteCommandOnNodeThroughVirtHandler(virtCli kubecli.KubevirtClient, nodeName string, command []string) (stdout string, stderr string, err error) {
	virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtCli).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
	if err != nil {
		return "", "", err
	}
	return ExecuteCommandOnPodV2(virtCli, virtHandlerPod, components.VirtHandlerName, command)
}

func GetKubevirtVMMetricsFunc(virtClient *kubecli.KubevirtClient, pod *k8sv1.Pod) func(string) string {
	return func(ip string) string {
		metricsURL := PrepareMetricsURL(ip, 8443)
		stdout, _, err := ExecuteCommandOnPodV2(*virtClient,
			pod,
			"virt-handler",
			[]string{
				"curl",
				"-L",
				"-k",
				metricsURL,
			})
		Expect(err).ToNot(HaveOccurred())
		return stdout
	}
}

func PrepareMetricsURL(ip string, port int) string {
	return fmt.Sprintf("https://%s/metrics", net.JoinHostPort(ip, strconv.Itoa(port)))
}

func StartVMAndExpectRunning(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine) *v1.VirtualMachine {
	runStrategyAlways := v1.RunStrategyAlways
	By("Starting the VirtualMachine")

	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		updatedVM.Spec.Running = nil
		updatedVM.Spec.RunStrategy = &runStrategyAlways
		_, err = virtClient.VirtualMachine(updatedVM.Namespace).Update(updatedVM)
		return err
	}, 300*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	// Observe the VirtualMachineInstance created
	Eventually(func() error {
		_, err := virtClient.VirtualMachineInstance(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		return err
	}, 300*time.Second, 1*time.Second).Should(Succeed())

	By("VMI has the running condition")
	Eventually(func() bool {
		vm, err := virtClient.VirtualMachine(updatedVM.Namespace).Get(updatedVM.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return vm.Status.Ready
	}, 300*time.Second, 1*time.Second).Should(BeTrue())

	return updatedVM
}
func GetNodeHostModel(node *k8sv1.Node) (hostModel string) {
	for key, _ := range node.Labels {
		if strings.HasPrefix(key, v1.HostModelCPULabel) {
			hostModel = strings.TrimPrefix(key, v1.HostModelCPULabel)
			break
		}
	}
	return hostModel
}
func GetValidSourceNodeAndTargetNodeForHostModelMigration(virtCli kubecli.KubevirtClient) (sourceNode *k8sv1.Node, targetNode *k8sv1.Node, err error) {
	getNodeHostRequiredFeatures := func(node *k8sv1.Node) (features []string) {
		for key, _ := range node.Labels {
			if strings.HasPrefix(key, v1.HostModelRequiredFeaturesLabel) {
				features = append(features, strings.TrimPrefix(key, v1.HostModelRequiredFeaturesLabel))
			}
		}
		return features
	}
	doesFeaturesSupportedOnNode := func(node *k8sv1.Node, features []string) bool {
		isFeatureSupported := func(feature string) bool {
			for key, _ := range node.Labels {
				if strings.HasPrefix(key, v1.CPUFeatureLabel) && strings.Contains(key, feature) {
					return true
				}
			}
			return false
		}
		for _, feature := range features {
			if !isFeatureSupported(feature) {
				return false
			}
		}

		return true
	}

	var sourceHostCpuModel string

	nodes := libnode.GetAllSchedulableNodes(virtCli)
	Expect(err).ToNot(HaveOccurred(), "Should list compute nodes")
	for _, potentialSourceNode := range nodes.Items {
		for _, potentialTargetNode := range nodes.Items {
			if potentialSourceNode.Name == potentialTargetNode.Name {
				continue
			}

			sourceHostCpuModel = GetNodeHostModel(&potentialSourceNode)
			if sourceHostCpuModel == "" {
				continue
			}
			supportedInTarget := false
			for key, _ := range potentialTargetNode.Labels {
				if strings.HasPrefix(key, v1.SupportedHostModelMigrationCPU) && strings.Contains(key, sourceHostCpuModel) {
					supportedInTarget = true
					break
				}
			}

			if supportedInTarget == false {
				continue
			}
			sourceNodeHostModelRequiredFeatures := getNodeHostRequiredFeatures(&potentialSourceNode)
			if doesFeaturesSupportedOnNode(&potentialTargetNode, sourceNodeHostModelRequiredFeatures) == false {
				continue
			}
			return &potentialSourceNode, &potentialTargetNode, nil
		}
	}
	return nil, nil, fmt.Errorf("couldn't find valid nodes for host-model migration")
}

func AffinityToMigrateFromSourceToTargetAndBack(sourceNode *k8sv1.Node, targetNode *k8sv1.Node) (nodefiinity *k8sv1.NodeAffinity, err error) {
	if sourceNode == nil || targetNode == nil {
		return nil, fmt.Errorf("couldn't find valid nodes for host-model migration")
	}
	return &k8sv1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
			NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
				{
					MatchExpressions: []k8sv1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: k8sv1.NodeSelectorOpIn,
							Values:   []string{sourceNode.Name, targetNode.Name},
						},
					},
				},
			},
		},
		PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.PreferredSchedulingTerm{
			{
				Preference: k8sv1.NodeSelectorTerm{
					MatchExpressions: []k8sv1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/hostname",
							Operator: k8sv1.NodeSelectorOpIn,
							Values:   []string{sourceNode.Name},
						},
					},
				},
				Weight: 1,
			},
		},
	}, nil
}
