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
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

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

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"

	util2 "kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	kutil "kubevirt.io/kubevirt/pkg/util"
	launcherApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

const (
	BinBash                = "/bin/bash"
	waitingVMInstanceStart = "Waiting until the VirtualMachineInstance will start"

	CustomHostPath     = "custom-host-path"
	DiskAlpineHostPath = "disk-alpine-host-path"
	DiskWindowsSysprep = "disk-windows-sysprep"
	DiskCustomHostPath = "disk-custom-host-path"
)

func RunVMIAndExpectLaunch(vmi *v1.VirtualMachineInstance, timeout int) *v1.VirtualMachineInstance {
	vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	By(waitingVMInstanceStart)
	return libwait.WaitForVMIPhase(vmi,
		[]v1.VirtualMachineInstancePhase{v1.Running},
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

// NewRandomVMIWithEphemeralDiskAndUserdata
//
// Deprecated: Use libvmi directly
func NewRandomVMIWithEphemeralDiskAndUserdata(containerImage string, userData string) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	cloudInitNoCloudEncodedUserData := libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(userData))
	cloudInitNoCloudEncodedUserData(vmi)
	return vmi
}

// NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData
//
// Deprecated: Use libvmi directly
func NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(containerImage, userData, networkData string, b64encode bool) *v1.VirtualMachineInstance {
	vmi := NewRandomVMIWithEphemeralDisk(containerImage)
	if b64encode {
		cloudInitConfigDriveData := libvmi.WithCloudInitConfigDrive(libvmici.WithConfigDriveEncodedUserData(userData), libvmici.WithConfigDriveEncodedNetworkData(networkData))
		cloudInitConfigDriveData(vmi)
	} else {
		cloudInitConfigDriveData := libvmi.WithCloudInitConfigDrive(libvmici.WithConfigDriveUserData(userData), libvmici.WithConfigDriveNetworkData(networkData))
		cloudInitConfigDriveData(vmi)
	}
	return vmi
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

// RunCommandOnVmiPod runs specified command on the virt-launcher pod
func RunCommandOnVmiPod(vmi *v1.VirtualMachineInstance, command []string) string {
	virtClient := kubevirt.Client()
	pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), UnfinishedVMIPodSelector(vmi))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pods.Items).NotTo(BeEmpty())
	vmiPod := pods.Items[0]

	output, err := exec.ExecuteCommandOnPod(
		&vmiPod,
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
	Eventually(ThisVM(updatedVM)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())
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

func callUrlOnPod(pod *k8sv1.Pod, port string, url string) ([]byte, error) {
	randPort := strconv.Itoa(4321 + rand.Intn(6000))
	stopChan := make(chan struct{})
	defer close(stopChan)
	err := libpod.ForwardPorts(pod, []string{fmt.Sprintf("%s:%s", randPort, port)}, stopChan, 5*time.Second)
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
			&expect.BSnd{S: console.EchoLastReturnValue},
			&expect.BExp{R: console.RetValue("0")},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}
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
	Eventually(ThisVM(updatedVM)).WithTimeout(300 * time.Second).WithPolling(time.Second).Should(BeReady())

	return updatedVM
}
