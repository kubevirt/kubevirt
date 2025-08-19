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

package tests_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	k8sv1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/rand"
	k8sWatch "k8s.io/apimachinery/pkg/watch"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	kvcorev1 "kubevirt.io/client-go/kubevirt/typed/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
)

var _ = Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]VMIlifecycle", decorators.SigCompute, decorators.VMIlifecycle, decorators.WgArm64, func() {

	const fakeLibvirtLogFilters = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"
	const startupTimeout = 45

	Context("when virt-handler is deleted", Serial, decorators.WgS390x, func() {
		It("[test_id:4716]should label the node with kubevirt.io/schedulable=false", func() {
			pods, err := kubevirt.Client().CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).ToNot(BeEmpty())

			pod := pods.Items[0]
			handlerNamespace := pod.GetNamespace()

			By("setting up a watch on Nodes")
			nodeWatch, err := kubevirt.Client().CoreV1().Nodes().Watch(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = kubevirt.Client().CoreV1().Pods(handlerNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(nodeWatch.ResultChan(), 120*time.Second).Should(Receive(WithTransform(func(e k8sWatch.Event) metav1.ObjectMeta {
				node, ok := e.Object.(*k8sv1.Node)
				Expect(ok).To(BeTrue())
				return node.ObjectMeta
			}, MatchFields(IgnoreExtras, Fields{
				"Name":   Equal(pod.Spec.NodeName),
				"Labels": HaveKeyWithValue(v1.NodeSchedulable, "false"),
			}))), "Failed to observe change in schedulable label")
		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Creating a VirtualMachineInstance", func() {

		It("[test_id:6095]should start in paused state if start strategy set to paused", decorators.WgS390x, decorators.Conformance, func() {
			vmi := libvmifact.NewAlpine(libvmi.WithStartStrategy(v1.StartStrategyPaused))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

			By("Unpausing VMI")
			err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
		})

		It("[test_id:1621]should attach virt-launcher to it", decorators.WgS390x, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(), startupTimeout)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(kubevirt.Client(), vmi) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("Found PID for"))
		})

		It("[test_id:3196]should carry kubernetes and kubevirt annotations to pod", decorators.WgS390x, decorators.Conformance, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(
				libvmi.WithAnnotation("kubevirt.io/test", "test"),
				libvmi.WithAnnotation("kubernetes.io/test", "test"),
				libvmi.WithAnnotation("testannotation", "annotation from vmi")),
				startupTimeout)

			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(pod.Annotations).To(HaveKey("kubevirt.io/test"), "kubevirt annotation should not be carried to the pod")
			Expect(pod.Annotations).To(HaveKey("kubernetes.io/test"), "kubernetes annotation should not be carried to the pod")
			Expect(pod.Annotations).To(HaveKeyWithValue("testannotation", "annotation from vmi"), "annotation should be carried to the pod")

		})

		It("Should prevent eviction when EvictionStratgy: External", decorators.WgS390x, decorators.Conformance, func() {
			vmi := libvmifact.NewAlpine(libvmi.WithEvictionStrategy(v1.EvictionStrategyExternal))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)

			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			By("calling evict on VMI's pod")
			err = kubevirt.Client().CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
			// The "too many requests" err is what get's returned when an
			// eviction would invalidate a pdb. This is what we want to see here.
			Expect(err).To(MatchError(k8serrors.IsTooManyRequests, "too many requests should be returned as way of blocking eviction"))
			Expect(err).To(MatchError(ContainSubstring("Eviction triggered evacuation of VMI")))

			By("should have evacuation node name set on vmi status")
			vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.EvacuationNodeName).To(Equal(pod.Spec.NodeName), "Should have evacuation node name set to where the Pod is running")

			By("should not delete the Pod")
			Consistently(func() *metav1.Time {
				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				return pod.DeletionTimestamp
			}, 10*time.Second, 1*time.Second).Should(BeNil(), "Should not delete the Pod")
		})

		It("[test_id:1622]should log libvirtd logs", decorators.WgS390x, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(), startupTimeout)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(kubevirt.Client(), vmi) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("libvirt version: "))
			Eventually(logs,
				2*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring(`"subcomponent":"libvirt"`))
		})

		DescribeTable("log libvirtd debug logs should be", func(vmiLabels, vmiAnnotations map[string]string, expectDebugLogs bool) {
			options := []libvmi.Option{libvmi.WithMemoryRequest("32Mi")}
			for k, v := range vmiLabels {
				options = append(options, libvmi.WithLabel(k, v))
			}

			for k, v := range vmiAnnotations {
				options = append(options, libvmi.WithAnnotation(k, v))
			}
			vmi := libvmi.New(options...)

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(kubevirt.Client(), vmi) }

			const totalTestTime = 2 * time.Second
			const checkIntervalTime = 500 * time.Millisecond
			const logEntryToSearch = "QEMU_MONITOR_SEND_MSG"
			const subcomponent = `"subcomponent":"libvirt"`

			// There are plenty of strings we can use to identify the debug logs.
			// Here we use something we deeply care about when in debug mode.
			if expectDebugLogs {
				Eventually(logs,
					totalTestTime,
					checkIntervalTime).
					Should(And(ContainSubstring(logEntryToSearch), ContainSubstring(subcomponent)))
			} else {
				Consistently(logs,
					totalTestTime,
					checkIntervalTime).
					ShouldNot(And(ContainSubstring(logEntryToSearch), ContainSubstring(subcomponent)))
			}

		},
			Entry("[test_id:3197]enabled when debugLogs label defined", map[string]string{"debugLogs": "true"}, nil, true),
			Entry("[test_id:8530]enabled when customLogFilters defined", nil, map[string]string{v1.CustomLibvirtLogFiltersAnnotation: fakeLibvirtLogFilters}, true),
			Entry("[test_id:8531]enabled when log verbosity is high", map[string]string{"logVerbosity": "10"}, nil, true),
			Entry("[test_id:8532]disabled when log verbosity is low", map[string]string{"logVerbosity": "2"}, nil, false),
			Entry("[test_id:8533]disabled when log verbosity, debug logs and customLogFilters are not defined", nil, nil, false),
		)

		It("[test_id:1623]should reject POST if validation webhook deems the spec invalid", decorators.WgS390x, func() {
			vmi := libvmifact.NewAlpine()
			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
			})

			result := kubevirt.Client().RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background())

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "VMI should be rejected as unprocessable")

			reviewResponse := &metav1.Status{}
			body, _ := result.Raw()
			Expect(json.Unmarshal(body, reviewResponse)).To(Succeed())

			Expect(reviewResponse.Details.Causes).To(HaveLen(2), "There should be 2 thing wrong in response")
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[1].name"))
			Expect(reviewResponse.Details.Causes[1].Field).To(Equal("spec.domain.devices.disks[2].name"))
		})

		It("[test_id:1624]should reject PATCH if schema is invalid", decorators.WgS390x, func() {
			vmi := libvmifact.NewAlpine()
			err := kubevirt.Client().RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred(), "Send POST successfully")

			// Add a disk without a volume reference (this is in valid)
			patchStr := fmt.Sprintf("{\"apiVersion\":\"kubevirt.io/%s\",\"kind\":\"VirtualMachineInstance\",\"spec\":{\"domain\":{\"devices\":{\"disks\":[{\"disk\":{\"bus\":\"virtio\"},\"name\":\"fakedisk\"}]}}}}", v1.ApiLatestVersion)

			result := kubevirt.Client().RestClient().Patch(types.MergePatchType).Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Name(vmi.Name).Body([]byte(patchStr)).Do(context.Background())

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "The entity should be unprocessable")
		})

		Context("when name is longer than 63 characters", decorators.WgS390x, func() {
			It("[test_id:1625]should start it", func() {
				By("Creating a VirtualMachineInstance with a long name")
				vmi := libvmifact.NewAlpine(libvmi.WithName("testvmi" + rand.String(63)))
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "cannot create VirtualMachineInstance %q: %v", vmi.Name, err)
				Expect(len(vmi.Name)).To(BeNumerically(">", 63), "VirtualMachineInstance %q name is not longer than 63 characters", vmi.Name)

				By("Waiting until it starts")
				libwait.WaitForSuccessfulVMIStart(vmi)
				vmi, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "cannot fetch VirtualMachineInstance %q: %v", vmi.Name, err)

				By("Obtaining serial console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed(), "VirtualMachineInstance %q console is not accessible: %v", vmi.Name, err)
			})
		})

		bootOrderToDisk := func(bootOrder uint) func(disk *v1.Disk) {
			return func(disk *v1.Disk) {
				disk.BootOrder = &bootOrder
			}
		}

		Context("with boot order", func() {
			DescribeTable("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]should be able to boot from selected disk", func(disk1, disk2 libvmi.Option, expectedConsoleText string) {
				By("defining a VirtualMachineInstance with an Alpine disk")
				vmi := libvmi.New(disk1, disk2, libvmi.WithMemoryRequest("256Mi"))

				By("starting VMI")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 2*startupTimeout)

				By("Checking console text")
				err := console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedConsoleText},
				}, 90)
				Expect(err).ToNot(HaveOccurred(), "Should match the console in VMI")
			},
				Entry("[test_id:1627]Alpine as first boot",
					libvmi.WithContainerDisk("disk1", cd.ContainerDiskFor(cd.ContainerDiskAlpine), bootOrderToDisk(1)), libvmi.WithContainerDisk("disk2", cd.ContainerDiskFor(cd.ContainerDiskCirros), bootOrderToDisk(2)),
					"Welcome to Alpine"),
				Entry("[test_id:1628]Cirros as first boot",
					libvmi.WithContainerDisk("disk1", cd.ContainerDiskFor(cd.ContainerDiskAlpine), bootOrderToDisk(2)), libvmi.WithContainerDisk("disk2", cd.ContainerDiskFor(cd.ContainerDiskCirros), bootOrderToDisk(1)),
					"cirros"),
			)
		})

		Context("with user-data", func() {

			Context("without k8s secret", func() {

				It("[test_id:1630]should log warning and proceed once the secret is there", func() {
					userData64 := ""
					vmi := libvmifact.NewCirros()

					for _, volume := range vmi.Spec.Volumes {
						if volume.CloudInitNoCloud != nil {
							spec := volume.CloudInitNoCloud
							userData64 = spec.UserDataBase64
							spec.UserDataBase64 = ""
							spec.UserDataSecretRef = &k8sv1.LocalObjectReference{Name: "nonexistent"}
							break
						}
					}
					By("Starting a VirtualMachineInstance")
					createdVMI := libvmops.RunVMIAndExpectScheduling(vmi, 30)
					launcher, err := libpod.GetPodByVirtualMachineInstance(createdVMI, createdVMI.Namespace)
					Expect(err).ToNot(HaveOccurred())
					// Wait until we see that starting the VirtualMachineInstance is failing
					By(fmt.Sprintf("Checking that VirtualMachineInstance start failed: starting at %v", time.Now()))
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					event := watcher.New(launcher).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.WarningEvent, "FailedMount")
					Expect(event.Message).To(SatisfyAny(
						ContainSubstring(`secret "nonexistent" not found`),
						ContainSubstring(`secrets "nonexistent" not found`), // for k8s 1.11.x
					), "VMI should not be started")

					// Creat nonexistent secret, so that the VirtualMachineInstance can recover
					By("Creating a user-data secret")
					secret := libsecret.New("nonexistent", libsecret.DataString{"userdata": userData64})
					_, err = kubevirt.Client().CoreV1().Secrets(createdVMI.Namespace).Create(context.Background(), secret, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should create secret successfully")

					// Wait for the VirtualMachineInstance to be started, allow warning events to occur
					By("Checking that VirtualMachineInstance start succeeded")
					watcher.New(createdVMI).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(ctx, watcher.NormalEvent, v1.Started)
				})
			})
		})

		Context("with nodeselector", func() {
			It("[test_id:5760]should check if vm's with non existing nodeselector is not running and node selector is not updated", func() {
				vmi := libvmifact.NewCirros()
				By("setting nodeselector with non-existing-os label")
				vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelOSStable: "not-existing-os"}
				vmi = libvmops.RunVMIAndExpectScheduling(vmi, 30)

				pods, err := kubevirt.Client().CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, pod := range pods.Items {
					for _, owner := range pod.GetOwnerReferences() {
						if owner.Name == vmi.Name {
							break
						}
					}
					Expect(pod.Spec.NodeSelector[k8sv1.LabelOSStable]).To(Equal("not-existing-os"), "pod should have node selector")
					Expect(pod.Status.Phase).To(Equal(k8sv1.PodPending), "pod has to be in pending state")
					for _, condition := range pod.Status.Conditions {
						if condition.Type == k8sv1.PodScheduled {
							Expect(condition.Reason).To(Equal(k8sv1.PodReasonUnschedulable), "condition reason has to be unschedulable")
						}
					}
				}
			})

			It("[test_id:5761]should check if vm with valid node selector is scheduled and running and node selector is not updated", func() {
				vmi := libvmifact.NewCirros()
				vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelOSStable: "linux"}
				libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)

				pods, err := kubevirt.Client().CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, pod := range pods.Items {
					for _, owner := range pod.GetOwnerReferences() {
						if owner.Name == vmi.Name {
							break
						}
					}
					Expect(pod.Spec.NodeSelector[k8sv1.LabelOSStable]).To(Equal("linux"), "pod should have node selector")
					Expect(pod.Status.Phase).To(Equal(k8sv1.PodRunning), "pod has to be in running state")
					for _, condition := range pod.Status.Conditions {
						if condition.Type == k8sv1.ContainersReady {
							Expect(condition.Reason).To(Equal(""), "condition reason has to be empty")
						}
					}
				}
			})

			Context("for Machine Type", func() {

				DescribeTable("should prevent scheduling of a pod for a VMI with an unsupported machine type", func(unsupportedMachineType string) {
					virtClient := kubevirt.Client()
					vmi := libvmifact.NewGuestless()
					vmi.Namespace = testsuite.GetTestNamespace(vmi)
					vmi.Spec.Domain.Machine = &v1.Machine{Type: unsupportedMachineType}

					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.BeInPhase(v1.Scheduling))

					virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).ToNot(HaveOccurred())
					Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKey(ContainSubstring(v1.SupportedMachineTypeLabel + unsupportedMachineType)))

					var scheduledCond *v1.VirtualMachineInstanceCondition
					Eventually(func() *v1.VirtualMachineInstanceCondition {
						curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						scheduledCond = controller.NewVirtualMachineInstanceConditionManager().
							GetCondition(curVMI, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
						return scheduledCond
					}, 10*time.Second, 1*time.Second).ShouldNot(BeNil(), "The PodScheduled condition should eventually appear")

					Expect(scheduledCond.Status).To(BeEquivalentTo(k8sv1.ConditionFalse))
					Expect(scheduledCond.Reason).To(BeEquivalentTo(k8sv1.PodReasonUnschedulable))
					Expect(scheduledCond.Message).To(ContainSubstring("node(s) didn't match Pod's node affinity/selector"))
				},
					Entry("amd64", "pc-q35-test-1.2.3", decorators.RequiresAMD64),
					Entry("arm64", "virt-test-1.2.3", decorators.RequiresARM64),
					Entry("s390x", "s390-ccw-virtio-test-1.2.3", decorators.RequiresS390X),
				)
			})
		})

		Context("when guest crashes", Serial, decorators.VMIlifecycle, func() {
			BeforeEach(func() {
				kvconfig.EnableFeatureGate(featuregate.PanicDevicesGate)
			})

			DescribeTable("should be stopped and have Failed phase when a PanicDevice is provided", func(device v1.PanicDeviceModel) {
				vmi := libvmifact.NewFedora(libvmi.WithPanicDevice(device))
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("Crashing the vm guest")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "sudo su -\n"},
					&expect.BExp{R: "#"},
					&expect.BSnd{S: `echo c > /proc/sysrq-trigger` + "\n"},
					&expect.BExp{R: "sysrq triggered crash"},
				}, 10)).To(Succeed())

				By("Waiting for the vm to be stopped")
				event := watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(15*time.Second).WaitFor(ctx, watcher.WarningEvent, v1.Stopped)
				Expect(event.Message).To(ContainSubstring(`The VirtualMachineInstance crashed`), "VMI should be stopped because of a guest crash")

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(matcher.ThisVMI(vmi)).WithTimeout(10 * time.Second).WithPolling(time.Second).Should(matcher.BeInPhase(v1.Failed))
			},
				Entry("amd64", v1.Isa, decorators.RequiresAMD64),
				Entry("arm64", v1.Pvpanic, decorators.RequiresARM64),
			)
		})

		Context("when virt-launcher crashes", decorators.WgS390x, func() {
			It("[test_id:1631]should be stopped and have Failed phase", Serial, func() {
				vmi := libvmifact.NewAlpine()
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")

				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("Crashing the virt-launcher")
				vmiKiller, err := pkillAllLaunchers(kubevirt.Client(), nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should create vmi-killer pod to kill virt-launcher successfully")
				watcher.New(vmiKiller).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(ctx, watcher.NormalEvent, v1.Started)

				By("Waiting for the vm to be stopped")
				watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}, 10, 1).Should(Equal(v1.Failed), "VMI should be failed")
			})
		})

		Context("when virt-handler crashes", Serial, func() {
			// FIXME: This test has the issues that it tests a lot of different timing scenarios in an intransparent way:
			// e.g. virt-handler can die before or after virt-launcher. If we wait until virt-handler is dead before we
			// kill virt-launcher then we don't know if virt-handler already restarted.
			// Also the virt-handler crash-loop plays a role here. We could also change the daemon-set but then we would not check the crash behaviour.
			It("[test_id:1632]should recover and continue management", func() {
				vmi := libvmifact.NewAlpine()
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				vmi = libwait.WaitForSuccessfulVMIStart(vmi)
				nodeName := vmi.Status.NodeName

				// Kill virt-handler on the node the VirtualMachineInstance is active on.
				By("Crashing the virt-handler")
				err = pkillHandler(kubevirt.Client(), nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should kill virt-handler successfully")

				// Crash the VirtualMachineInstance and verify a recovered version of virt-handler processes the crash
				By("Killing the VirtualMachineInstance")
				err = pkillVMI(kubevirt.Client(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should kill VMI successfully")

				// Give virt-handler some time. It can greatly vary when virt-handler will be ready again
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(matcher.ThisVMI(vmi), 240*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Failed), "VMI should be failed")

				By("Waiting for the vmi to be stopped")
				watcher.New(vmi).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

				By("checking that it can still start VMIs")
				newVMI := libvmifact.NewCirros(libvmi.WithNodeSelectorFor(nodeName))
				newVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), newVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(newVMI)
			})
		})

		Context("when virt-handler is responsive", Serial, func() {
			It("[test_id:1633]should indicate that a node is ready for vmis", decorators.WgS390x, func() {

				By("adding a heartbeat annotation and a schedulable label to the node")
				nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				for _, node := range nodes.Items {
					Expect(node.Annotations[v1.VirtHandlerHeartbeat]).ToNot(BeEmpty(), "Nodes should have be ready for VMI")
				}

				node := &nodes.Items[0]
				node, err := kubevirt.Client().CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, v1.NodeSchedulable)), metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")
				timestamp := node.Annotations[v1.VirtHandlerHeartbeat]

				By("setting the schedulable label back to true")
				Eventually(func() string {
					n, err := kubevirt.Client().CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get nodes successfully")
					return n.Labels[v1.NodeSchedulable]
				}, 5*time.Minute, 2*time.Second).Should(Equal("true"), "Nodes should be schedulable")
				By("updating the heartbeat roughly every minute")
				Expect(func() string {
					n, err := kubevirt.Client().CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get nodes successfully")
					return n.Labels[v1.VirtHandlerHeartbeat]
				}()).ShouldNot(Equal(timestamp), "Should not have old vmi heartbeat")
			})

			It("[test_id:3198]device plugins should re-register if the kubelet restarts", func() {

				By("starting a VMI on a node")
				vmi := libvmifact.NewAlpine()
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				By("triggering a device plugin re-registration on that node")
				pod, err := libnode.GetVirtHandlerPod(kubevirt.Client(), nodeName)
				Expect(err).ToNot(HaveOccurred())

				_, _, err = exec.ExecuteCommandOnPodWithResults(pod,
					"virt-handler",
					[]string{
						"rm",
						// We want to fail if the file does not exist, but don't want to be asked
						// if we really want to remove write-protected files
						"--interactive=never",
						device_manager.SocketPath("kvm"),
					})
				Expect(err).ToNot(HaveOccurred())

				By("checking if we see the device plugin restart in the logs")
				virtHandlerPod, err := libnode.GetVirtHandlerPod(kubevirt.Client(), nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(10)
				logsQuery := kubevirt.Client().CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})
				Eventually(func() string {
					data, err := logsQuery.DoRaw(context.Background())
					Expect(err).ToNot(HaveOccurred(), "Should get logs")
					return string(data)
				}, 60, 1).Should(
					ContainSubstring(
						fmt.Sprintf("device socket file for device %s was removed, kubelet probably restarted.", "kvm"),
					), "Should log device plugin restart")

				// This is a little bit arbitrar
				// Background is that new pods go into a crash loop if the devices are still report but virt-handler
				// re-registers exactly during that moment. This is not too bad, since normally kubelet itself deletes
				// the socket and knows that the devices are not there. However we have to wait in this test a little bit.
				time.Sleep(10 * time.Second)

				By("starting another VMI on the same node, to verify devices still work")
				newVMI := libvmifact.NewCirros()
				newVMI.Spec.NodeSelector = map[string]string{k8sv1.LabelHostname: nodeName}
				newVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), newVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(newVMI)
			})
		})

		Context("when virt-handler is not responsive", Serial, func() {

			var vmi *v1.VirtualMachineInstance
			var nodeName string
			var virtHandler *k8sv1.Pod
			var virtHandlerAvailablePods int32

			BeforeEach(func() {
				// Schedule a vmi and make sure that virt-handler gets evicted from the node where the vmi was started
				// Note: we want VMI without any container
				vmi = libvmifact.NewGuestless(libvmi.WithLogSerialConsole(false))
				var err error
				vmi, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")

				// Ensure that the VMI is running. This is necessary to ensure that virt-handler is fully responsible for
				// the VMI. Otherwise virt-controller may move the VMI to failed instead of the node controller.
				nodeName = libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				).Status.NodeName

				virtHandler, err = libnode.GetVirtHandlerPod(kubevirt.Client(), nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client")

				ds, err := kubevirt.Client().AppsV1().DaemonSets(virtHandler.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler daemonset")
				// Save virt-handler number of desired pods
				virtHandlerAvailablePods = ds.Status.DesiredNumberScheduled

				kv := libkubevirt.GetCurrentKv(kubevirt.Client())
				kv.Spec.Workloads = &v1.ComponentConfig{
					NodePlacement: &v1.NodePlacement{
						Affinity: &k8sv1.Affinity{
							NodeAffinity: &k8sv1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
									NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
										{MatchExpressions: []k8sv1.NodeSelectorRequirement{
											{Key: k8sv1.LabelHostname, Operator: "NotIn", Values: []string{nodeName}},
										}},
									},
								},
							},
						},
					},
				}
				_, err = kubevirt.Client().KubeVirt(kv.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should update kubevirt infra placement")

				Eventually(func() error {
					_, err := kubevirt.Client().CoreV1().Pods(virtHandler.Namespace).Get(context.Background(), virtHandler.Name, metav1.GetOptions{})
					return err
				}, 120*time.Second, 1*time.Second).Should(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"), "The virt-handler pod should be gone")
			})

			It("[test_id:1634]the node controller should mark the node as unschedulable when the virt-handler heartbeat has timedout", func() {
				// Update virt-handler heartbeat, to trigger a timeout
				node, err := kubevirt.Client().CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get node successfully")

				Expect(node.Annotations).To(HaveKey(v1.VirtHandlerHeartbeat))
				lastHeartBeat := node.Annotations[v1.VirtHandlerHeartbeat]
				timestamp := &metav1.Time{}
				Expect(json.Unmarshal([]byte(`"`+lastHeartBeat+`"`), &timestamp)).To(Succeed())
				timeToSet := metav1.NewTime(timestamp.Add(-10 * time.Minute))
				patchBytes, err := patch.New(
					patch.WithReplace(fmt.Sprintf("/metadata/labels/%s", patch.EscapeJSONPointer(v1.NodeSchedulable)), "true"),
					patch.WithReplace(fmt.Sprintf("/metadata/annotations/%s", patch.EscapeJSONPointer(v1.VirtHandlerHeartbeat)), timeToSet),
				).GeneratePayload()
				Expect(err).ToNot(HaveOccurred(), "Should generate patches")

				_, err = kubevirt.Client().CoreV1().Nodes().Patch(context.Background(), nodeName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				// Note we cannot remove the Pod as the vmi controller also moves the VMI to Failed if Pod disappears
				// This leads to race condition, killing the process gives us exactly what we want
				By("killing the virt-launcher")
				vmiKiller, err := pkillAllLaunchers(kubevirt.Client(), nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should create vmi-killer pod to kill virt-launcher successfully")
				watcher.New(vmiKiller).SinceWatchedObjectResourceVersion().Timeout(20*time.Second).WaitFor(context.Background(), watcher.NormalEvent, v1.Started)

				// it will take at least 45 seconds until the vmi is gone, check the schedulable state in the meantime
				By("marking the node as not schedulable")
				Eventually(func() map[string]string {
					node, err := kubevirt.Client().CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get node successfully")
					return node.Labels
				}, 10*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.NodeSchedulable, "false"), "The node should not be schedulable")

				By("moving stuck vmis to failed state")
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Failed))

				failedVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get vmi successfully")
				Expect(failedVMI.Status.Reason).To(Equal("NodeUnresponsive"))

				err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				testsuite.RestoreKubeVirtResource()

				// Wait until virt-handler ds will have expected number of pods
				Eventually(func() bool {
					ds, err := kubevirt.Client().AppsV1().DaemonSets(virtHandler.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get virthandler successfully")

					return ds.Status.NumberAvailable == virtHandlerAvailablePods &&
						ds.Status.CurrentNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.DesiredNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.NumberReady == virtHandlerAvailablePods &&
						ds.Status.UpdatedNumberScheduled == virtHandlerAvailablePods
				}, 180*time.Second, 1*time.Second).Should(BeTrue(), "Virthandler should be ready to work")
			})
		})

		Context("with node tainted", Serial, func() {
			var nodes *k8sv1.NodeList
			BeforeEach(func() {
				Eventually(func() []k8sv1.Node {
					nodes = libnode.GetAllSchedulableNodes(kubevirt.Client())
					return nodes.Items
				}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")

				// Taint first node with "NoSchedule"
				data := []byte(`{"spec":{"taints":[{"effect":"NoSchedule","key":"test","timeAdded":null,"value":"123"}]}}`)
				_, err := kubevirt.Client().CoreV1().Nodes().Patch(context.Background(), nodes.Items[0].Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node")

			})

			AfterEach(func() {
				// Untaint first node
				data := []byte(`{"spec":{"taints":[]}}`)
				_, err := kubevirt.Client().CoreV1().Nodes().Patch(context.Background(), nodes.Items[0].Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node")
			})

			It("[test_id:1635]the vmi with tolerations should be scheduled", func() {
				vmi := libvmifact.NewCirros(libvmi.WithNodeAffinityFor(nodes.Items[0].Name))
				vmi.Spec.Tolerations = []k8sv1.Toleration{{Key: "test", Value: "123"}}
				libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)
			})

			It("[test_id:1636]the vmi without tolerations should not be scheduled", func() {
				vmi := libvmifact.NewCirros(libvmi.WithNodeAffinityFor(nodes.Items[0].Name))
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					scheduledCond := controller.NewVirtualMachineInstanceConditionManager().
						GetCondition(curVMI, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					if scheduledCond != nil {
						return scheduledCond.Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal(k8sv1.PodReasonUnschedulable), "VMI should be unschedulable")
			})
		})

		Context("with affinity", func() {
			var node *k8sv1.Node

			BeforeEach(func() {
				nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node = nodes.Items[0].DeepCopy()
			})

			It("[test_id:1637]the vmi with node affinity and no conflicts should be scheduled", decorators.Conformance, func() {
				vmi := libvmifact.NewCirros(libvmi.WithNodeAffinityFor(node.Name))

				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).To(Not(HaveOccurred()))
				vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Scheduled, v1.Running}, libwait.WithTimeout(startupTimeout))

				By("Asserting that VMI is scheduled on the pre-picked node")
				Expect(vmi.Status.NodeName).To(Equal(node.Name), "Updated VMI name run on the same node")

			})

			It("[test_id:1638]the vmi with node affinity and anti-pod affinity should not be scheduled", decorators.Conformance, func() {
				vmi := libvmifact.NewCirros(libvmi.WithNodeAffinityFor(node.Name))
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).To(Not(HaveOccurred()))
				vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Scheduled, v1.Running}, libwait.WithTimeout(startupTimeout))

				secondVMI := libvmifact.NewCirros(libvmi.WithNodeAffinityFor(node.Name))

				secondVMI.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{Key: v1.CreatedByLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{string(vmi.GetUID())}},
								},
							},
							TopologyKey: k8sv1.LabelHostname,
						},
					},
				}

				secondVMI, err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(secondVMI)).Create(context.Background(), secondVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMIB")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVmiB, err := kubevirt.Client().VirtualMachineInstance(secondVMI.Namespace).Get(context.Background(), secondVMI.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMIB")
					scheduledCond := controller.NewVirtualMachineInstanceConditionManager().
						GetCondition(curVmiB, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					if scheduledCond != nil {
						return scheduledCond.Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal(k8sv1.PodReasonUnschedulable), "VMI should be unchedulable")

			})

		})

		Context("with default cpu model", Serial, decorators.WgS390x, decorators.CPUModel, func() {
			var originalConfig v1.KubeVirtConfiguration
			var supportedCpuModels []string
			var defaultCPUModel string
			var vmiCPUModel string

			//store old kubevirt-config
			BeforeEach(func() {
				nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				supportedCpuModels = libnode.GetSupportedCPUModels(*nodes)
				if len(supportedCpuModels) < 2 {
					Fail("need at least 2 supported cpu models for this test")
				}
				defaultCPUModel = supportedCpuModels[0]
				vmiCPUModel = supportedCpuModels[1]
				kv := libkubevirt.GetCurrentKv(kubevirt.Client())
				originalConfig = kv.Spec.Configuration
			})

			//replace new kubevirt-config with old config
			AfterEach(func() {
				kvconfig.UpdateKubeVirtConfigValueAndWait(originalConfig)
			})

			It("[test_id:3199]should set default cpu model when vmi doesn't have it set", func() {
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

				vmi := libvmifact.NewAlpine()
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)
				curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(defaultCPUModel), "Expected default CPU model")

			})

			It("[test_id:3200]should not set default cpu model when vmi has it set", func() {
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

				vmi := libvmifact.NewAlpine()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Model: vmiCPUModel,
				}
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)

				curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(vmiCPUModel), "Expected vmi CPU model")

			})

			It("[sig-compute][test_id:3201]should set cpu model to default when vmi does not have it set and default cpu model is not set", func() {
				vmi := libvmifact.NewAlpine()
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)

				curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(v1.DefaultCPUModel),
					fmt.Sprintf("Expected CPU model to equal to the default (%v)", v1.DefaultCPUModel),
				)
			})
			It("should add node selector to virt-launcher when setting default cpuModel in kubevirtCR", func() {
				if len(supportedCpuModels) < 1 {
					Fail("Must have at least one supported cpu model for this test")
				}
				defaultCPUModel := supportedCpuModels[0]
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

				newVMI := libvmifact.NewAlpine()
				newVMI = libvmops.RunVMIAndExpectLaunch(newVMI, libvmops.StartupTimeoutSecondsMedium)
				By("Fetching virt-launcher pod")
				virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(newVMI, newVMI.Namespace)
				Expect(err).NotTo(HaveOccurred())
				Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKey(ContainSubstring(defaultCPUModel)), "Node selector for the cpuModel in vmi spec should appear in virt-launcher pod")

			})

			It("should prefer node selector of the vmi if cpuModel field is set in kubevirtCR and in the vmi", func() {
				if len(supportedCpuModels) < 2 {
					Fail("Must have at least two supported cpuModel for this test")
				}
				vmiCPUModel := supportedCpuModels[1]
				defaultCPUModel := supportedCpuModels[0]
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)

				newVMI := libvmifact.NewAlpine()
				newVMI.Spec.Domain.CPU = &v1.CPU{
					Model: vmiCPUModel,
				}
				newVMI = libvmops.RunVMIAndExpectLaunch(newVMI, libvmops.StartupTimeoutSecondsMedium)
				By("Fetching virt-launcher pod")
				virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(newVMI, newVMI.Namespace)
				Expect(err).NotTo(HaveOccurred())
				Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKey(ContainSubstring(vmiCPUModel)), "Node selector for the cpuModel in kubevirtCR should appear in virt-launcher pod")

			})
		})

		Context("with node feature discovery", Serial, decorators.CPUModel, func() {
			var node *k8sv1.Node
			var supportedCPU string
			var supportedCPUs []string
			var supportedFeatures []string
			var nodes *k8sv1.NodeList
			var supportedKVMInfoFeature []string

			BeforeEach(func() {
				nodes = libnode.GetAllSchedulableNodes(kubevirt.Client())
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

				node = &nodes.Items[0]
				supportedCPUs = libnode.GetSupportedCPUModels(*nodes)
				Expect(supportedCPUs).ToNot(BeEmpty(), "There should be some supported cpu models")

				supportedCPU = supportedCPUs[0]

				supportedFeatures = libnode.GetSupportedCPUFeatures(*nodes)
				Expect(len(supportedFeatures)).To(BeNumerically(">=", 2), "There should be at least 2 supported cpu features")

				for key := range node.Labels {
					if strings.Contains(key, v1.HypervLabel) &&
						!strings.Contains(key, "tlbflush") &&
						!strings.Contains(key, "ipi") &&
						!strings.Contains(key, "synictimer") {
						supportedKVMInfoFeature = append(supportedKVMInfoFeature, strings.TrimPrefix(key, v1.HypervLabel))
					}

				}

				kvconfig.EnableFeatureGate(featuregate.HypervStrictCheckGate)
			})

			It("[test_id:1639]the vmi with cpu.model matching a nfd label on a node should be scheduled", func() {
				vmi := libvmifact.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: supportedCPU,
				}
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)

				By("Verifying VirtualMachineInstance's status is Succeeded")
				Eventually(func() v1.VirtualMachineInstancePhase {
					currVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					return currVMI.Status.Phase
				}, 120, 0.5).Should(Equal(v1.Running), "VMI should be succeeded")
			})

			It("[test_id:1640]the vmi with cpu.model that cannot match an nfd label on node should not be scheduled", func() {
				vmi := libvmifact.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: "486",
				}

				//Make sure the vmi should try to be scheduled only on master node
				vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelHostname: node.Name}

				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi")
					scheduledCond := controller.NewVirtualMachineInstanceConditionManager().
						GetCondition(curVMI, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					if scheduledCond != nil {
						return scheduledCond.Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal(k8sv1.PodReasonUnschedulable), "VMI should be unchedulable")
			})

			It("[test_id:3202]the vmi with cpu.features matching nfd labels on a node should be scheduled", func() {

				By("adding a node-feature-discovery CPU model label to a node")
				vmi := libvmifact.NewCirros()
				const featureToDisable = "fpu"

				featureToRequire := supportedFeatures[0]

				if featureToRequire == featureToDisable {
					// Picking another feature since this one is going to be disabled
					featureToRequire = supportedFeatures[1]
				}

				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   featureToRequire,
							Policy: "require",
						},
						{
							Name:   "fpu",
							Policy: "disable",
						},
					},
				}
				libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)
			})

			It("[test_id:3203]the vmi with cpu.features that cannot match nfd labels on a node should not be scheduled", func() {
				var featureDenyList = map[string]struct{}{
					"svm": {},
				}
				appendFeatureFromFeatureLabel := func(supportedFeatures []string, label string) []string {
					if strings.Contains(label, v1.CPUFeatureLabel) {
						feature := strings.TrimPrefix(label, v1.CPUFeatureLabel)
						if _, exist := featureDenyList[feature]; !exist {
							return append(supportedFeatures, feature)
						}
					}
					return supportedFeatures
				}

				removeDups := func(elements []string) (intersection []string) {
					found := make(map[string]struct{})
					for _, element := range elements {
						if _, exist := found[element]; !exist {
							intersection = append(intersection, element)
							found[element] = struct{}{}
						}
					}
					return intersection
				}

				setIntersection := func(firstSet, secondSet []string) []string {
					firstSetMap := make(map[string]struct{})
					var setOfFeaturesWithDups []string

					for _, element := range firstSet {
						firstSetMap[element] = struct{}{}
					}

					for _, element := range secondSet {
						if _, exist := firstSetMap[element]; exist {
							setOfFeaturesWithDups = append(setOfFeaturesWithDups, element)
						}
					}
					return removeDups(setOfFeaturesWithDups)
				}

				GetSupportedCPUFeaturesFromNodes := func(nodes k8sv1.NodeList) []string {
					var supportedFeatures []string
					for label := range nodes.Items[0].Labels {
						supportedFeatures = appendFeatureFromFeatureLabel(supportedFeatures, label)
					}

					for _, node := range nodes.Items {
						var currFeatures []string
						for label := range node.Labels {
							currFeatures = appendFeatureFromFeatureLabel(currFeatures, label)
						}
						supportedFeatures = setIntersection(supportedFeatures, currFeatures)
					}

					return supportedFeatures
				}

				supportedFeaturesAmongAllNodes := GetSupportedCPUFeaturesFromNodes(*nodes)
				vmi := libvmifact.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   supportedFeaturesAmongAllNodes[0],
							Policy: "require",
						},
						{
							Name:   supportedFeaturesAmongAllNodes[1],
							Policy: "forbid",
						},
					},
				}

				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi")
					scheduledCond := controller.NewVirtualMachineInstanceConditionManager().
						GetCondition(curVMI, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					if scheduledCond != nil {
						return scheduledCond.Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal(k8sv1.PodReasonUnschedulable), "VMI should be unchedulable")
			})

			It("[test_id:3204]the vmi with cpu.feature policy 'forbid' should not be scheduled on a node with that cpu feature label", func() {

				// Add node affinity first to test later on that although there is node affinity to
				// the specific node - the feature policy 'forbid' will deny scheduling on that node.
				vmi := libvmifact.NewCirros(libvmi.WithNodeAffinityFor(nodes.Items[0].Name))
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   supportedFeatures[0],
							Policy: "forbid",
						},
					},
				}

				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi")
					scheduledCond := controller.NewVirtualMachineInstanceConditionManager().
						GetCondition(curVMI, v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled))
					if scheduledCond != nil {
						return scheduledCond.Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal(k8sv1.PodReasonUnschedulable), "VMI should be unschedulable")
			})

		})

		Context("with non default namespace", func() {
			DescribeTable("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]should log libvirt start and stop lifecycle events of the domain", func(alternativeNamespace bool) {
				namespace := testsuite.GetTestNamespace(nil)
				if alternativeNamespace {
					namespace = testsuite.NamespaceTestAlternative
				}

				nodes := libnode.GetAllSchedulableNodes(kubevirt.Client())
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				By("Creating a VirtualMachineInstance with different namespace")
				vmi := libvmi.New(
					libvmi.WithMemoryRequest("1Mi"),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				)
				virtHandlerPod, err := libnode.GetVirtHandlerPod(kubevirt.Client(), node)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := kubevirt.Client().CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				// Make sure we schedule the VirtualMachineInstance to master
				vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelHostname: node}

				// Start the VirtualMachineInstance and wait for the confirmation of the start
				vmi, err = kubevirt.Client().VirtualMachineInstance(namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)

				// Check if the start event was logged
				By("Checking that virt-handler logs VirtualMachineInstance creation")
				Eventually(func() string {
					data, err := logsQuery.DoRaw(context.Background())
					Expect(err).ToNot(HaveOccurred(), "Should get logs from virthandler")
					return string(data)
				}, 30, 0.5).Should(MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain is in state Running reason Unknown","name":"%s"`, vmi.GetObjectMeta().GetName()), "Should verify from logs that domain is running")
				// Check the VirtualMachineInstance Namespace
				Expect(vmi.GetObjectMeta().GetNamespace()).To(Equal(namespace), "VMI should run in the right namespace")

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				_, err = kubevirt.Client().RestClient().Delete().Resource("virtualmachineinstances").Namespace(vmi.GetObjectMeta().GetNamespace()).Name(vmi.GetObjectMeta().GetName()).Do(context.Background()).Get()
				Expect(err).ToNot(HaveOccurred())
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				watcher.New(vmi).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.NormalEvent, v1.Deleted)
				libwait.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				// Check if the stop event was logged
				By("Checking that virt-handler logs VirtualMachineInstance deletion")
				/*
						Since we deleted the VMI object, there are two possible outcomes and both are expected:
						1. virt-controller kicks in, registers a deletion request on the launcher pod and K8s deletes the pod
					       before virt-handler had a chance to set or check the deletion timestamp on the domain.
						2. virt-handler detects the deletion timestamp on the domain and removes it.

						TODO: https://github.com/kubevirt/kubevirt/issues/3764
				*/
				Eventually(func() string {
					data, err := logsQuery.DoRaw(context.Background())
					Expect(err).ToNot(HaveOccurred(), "Should get the virthandler logs")
					return string(data)
				}, 30, 0.5).Should(SatisfyAny(
					MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain is marked for deletion","name":"%s"`, vmi.GetObjectMeta().GetName()),               // Domain was deleted by virt-handler
					MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain is in state Shutoff reason Destroyed","name":"%s"`, vmi.GetObjectMeta().GetName()), // Domain was destroyed because the launcher pod is gone
				), "Logs should confirm pod deletion")
			},
				Entry("[test_id:1641]Default test namespace", false),
				Entry("[test_id:1642]Alternative test namespace", true),
			)
		})

		Context("VM Accelerated Mode", decorators.WgS390x, func() {

			It("[test_id:1648]Should provide KVM via plugin framework", func() {
				nodeList := libnode.GetAllSchedulableNodes(kubevirt.Client())

				if len(nodeList.Items) == 0 {
					Fail("There are no compute nodes in cluster")
				}
				node := nodeList.Items[0]

				_, ok := node.Status.Allocatable[services.KvmDevice]
				Expect(ok).To(BeTrue(), "KVM devices not allocatable on node: %s", node.Name)

				_, ok = node.Status.Capacity[services.KvmDevice]
				Expect(ok).To(BeTrue(), "No Capacity for KVM devices on node: %s", node.Name)
			})
		})
	})

	Describe("Freeze/Unfreeze a VirtualMachineInstance", func() {
		It("[test_id:7476][test_id:7477]should fail without guest agent", func() {
			vmi := libvmifact.NewCirros()
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(180))

			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Freeze(context.Background(), vmi.Name, 0)
			Expect(err).To(MatchError(MatchRegexp("Internal error occurred:.*command Freeze failed:.*QEMU guest agent is not connected")))
			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Unfreeze(context.Background(), vmi.Name)
			Expect(err).To(MatchError(MatchRegexp("Internal error occurred:.*command Unfreeze failed:.*QEMU guest agent is not connected")))
		})

		waitVMIFSFreezeStatus := func(ns, name, expectedStatus string) {
			Eventually(func() string {
				vmi, err := kubevirt.Client().VirtualMachineInstance(ns).Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.FSFreezeStatus
			}, 30*time.Second, 2*time.Second).Should(Equal(expectedStatus))
		}

		It("[test_id:7479] should succeed", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(180))
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			By("Freezing VMI")
			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Freeze(context.Background(), vmi.Name, 0)
			Expect(err).ToNot(HaveOccurred())

			waitVMIFSFreezeStatus(vmi.Namespace, vmi.Name, "frozen")

			By("Unfreezing VMI")
			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Unfreeze(context.Background(), vmi.Name)
			Expect(err).ToNot(HaveOccurred())

			waitVMIFSFreezeStatus(vmi.Namespace, vmi.Name, "")
		})

		It("[test_id:7480] should succeed multiple times", decorators.Conformance, func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(180))
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			By("Freezing VMI")
			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Freeze(context.Background(), vmi.Name, 0)
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < 5; i++ {
				By("Freezing VMI")
				err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Freeze(context.Background(), vmi.Name, 0)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus(vmi.Namespace, vmi.Name, "frozen")
			}

			By("Unfreezing VMI")
			for i := 0; i < 5; i++ {
				err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Unfreeze(context.Background(), vmi.Name)
				Expect(err).ToNot(HaveOccurred())

				waitVMIFSFreezeStatus(vmi.Namespace, vmi.Name, "")
			}
		})

		It("Freeze without Unfreeze should trigger unfreeze after timeout", decorators.Conformance, func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(180))
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Freezing VMI")
			unfreezeTimeout := 10 * time.Second
			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Freeze(context.Background(), vmi.Name, unfreezeTimeout)
			Expect(err).ToNot(HaveOccurred())

			waitVMIFSFreezeStatus(vmi.Namespace, vmi.Name, "frozen")

			By("Wait Unfreeze VMI to be triggered")
			waitVMIFSFreezeStatus(vmi.Namespace, vmi.Name, "")
		})
	})

	Describe("Softreboot a VirtualMachineInstance", decorators.ACPI, func() {
		const vmiLaunchTimeout = 360

		It("soft reboot vmi with agent connected should succeed", decorators.Conformance, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(withoutACPI()), vmiLaunchTimeout)

			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
			Expect(err).ToNot(HaveOccurred())

			waitForVMIRebooted(vmi, console.LoginToFedora)
		})

		It("soft reboot vmi with ACPI feature enabled should succeed", decorators.Conformance, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), vmiLaunchTimeout)

			Expect(console.LoginToCirros(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

			err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
			Expect(err).ToNot(HaveOccurred())

			waitForVMIRebooted(vmi, console.LoginToCirros)
		})

		It("soft reboot vmi neither have the agent connected nor the ACPI feature enabled should fail", decorators.Conformance, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(withoutACPI()), vmiLaunchTimeout)

			Expect(console.LoginToCirros(vmi)).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceAgentConnected))

			err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
			Expect(err).To(MatchError(ContainSubstring("VMI neither have the agent connected nor the ACPI feature enabled")))
		})

		It("soft reboot vmi should fail to soft reboot a paused vmi", func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewFedora(), vmiLaunchTimeout)
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
			Expect(err).To(MatchError(ContainSubstring("VMI is paused")))

			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).SoftReboot(context.Background(), vmi.Name)
			Expect(err).ToNot(HaveOccurred())

			waitForVMIRebooted(vmi, console.LoginToFedora)
		})
	})

	Describe("Pausing/Unpausing a VirtualMachineInstance", func() {
		It("[test_id:4597]should signal paused state with condition", decorators.Conformance, func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), libvmops.StartupTimeoutSecondsMedium)
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))

			By("Pausing VMI")
			err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

			By("Unpausing VMI")
			err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceReady))
		})

		It("[test_id:3083][test_id:3084]should be able to connect to serial console and VNC", func() {
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(libvmi.WithAutoattachGraphicsDevice(true)), libvmops.StartupTimeoutSecondsMedium)

			By("Pausing the VMI")
			err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstanceReady))

			By("Trying to console into the VMI")
			_, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).SerialConsole(vmi.Name, &kvcorev1.SerialConsoleOptions{ConnectionTimeout: 30 * time.Second})
			Expect(err).ToNot(HaveOccurred())

			By("Trying to vnc into the VMI")
			_, err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).VNC(vmi.Name)
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:3090]should result in a difference in the uptime after pause", func() {
			const (
				sleepTimeSeconds = 10
				deviation        = 4
			)

			grepGuestUptime := func(vmi *v1.VirtualMachineInstance) float64 {
				res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
					&expect.BSnd{S: `cat /proc/uptime | awk '{print $1;}'` + "\n"},
					&expect.BExp{R: console.RetValue("[0-9\\.]+")}, // guest uptime
				}, 15)
				Expect(err).ToNot(HaveOccurred())
				re := regexp.MustCompile("\r\n[0-9\\.]+\r\n")
				guestUptime, err := strconv.ParseFloat(strings.TrimSpace(re.FindString(res[0].Match[0])), 64)
				Expect(err).ToNot(HaveOccurred(), "should be able to parse uptime to float")
				return guestUptime
			}

			hostUptime := func(startTime time.Time) float64 {
				return time.Since(startTime).Seconds()
			}
			startTime := time.Now()
			By("Starting a Cirros VMI")
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewCirros(), libvmops.StartupTimeoutSecondsMedium)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			By("checking uptime difference between guest and host")
			uptimeDiffBeforePausing := hostUptime(startTime) - grepGuestUptime(vmi)

			By("Pausing the VMI")
			err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))
			time.Sleep(sleepTimeSeconds * time.Second) // sleep to increase uptime diff

			By("Unpausing the VMI")
			err = kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

			By("Verifying VMI was indeed Paused")
			uptimeDiffAfterPausing := hostUptime(startTime) - grepGuestUptime(vmi)

			// We subtract from the sleep time the deviation due to the low resolution of `uptime` (seconds).
			// If you capture the uptime when it is at the beginning of that second or at the end of that second,
			// the value comes out the same even though in fact a whole second has almost passed.
			// In extreme cases, as we take 4 readings (2 initially and 2 after the unpause), the deviation could be up to just under 4 seconds.
			// This fact does not invalidate the purpose of the test, which is to prove that during the pause the vmi is actually paused.
			Expect(uptimeDiffAfterPausing-uptimeDiffBeforePausing).To(BeNumerically(">=", sleepTimeSeconds-deviation), fmt.Sprintf("guest should be paused for at least %d seconds", sleepTimeSeconds-deviation))
		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Delete a VirtualMachineInstance's Pod (API)", decorators.WgS390x, func() {
		It("[test_id:1650]should result in the VirtualMachineInstance moving to a finalized state", decorators.Conformance, func() {
			By("Creating the VirtualMachineInstance")
			vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(), startupTimeout)

			By("Verifying VirtualMachineInstance's pod is active")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			err = kubevirt.Client().CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			// TODD By("Verifying VirtualMachineInstance's pod terminates")

			// Wait for VirtualMachineInstance to finalize
			By("Waiting for the VirtualMachineInstance to move to a finalized state")
			Eventually(matcher.ThisVMI(vmi)).WithTimeout(time.Minute).WithPolling(time.Second).
				Should(Or(matcher.BeInPhase(v1.Succeeded), matcher.BeInPhase(v1.Failed)))
		})
	})
	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Delete a VirtualMachineInstance", func() {
		Context("with an active pod.", decorators.WgS390x, func() {
			It("[test_id:1651]should result in pod being terminated", func() {
				By("Creating the VirtualMachineInstance")
				vmi := libvmops.RunVMIAndExpectLaunch(libvmifact.NewAlpine(), startupTimeout)

				By("Verifying VirtualMachineInstance's pod is active")
				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				By("Deleting the VirtualMachineInstance")
				Expect(kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's pod terminates")
				Eventually(func() error {
					_, err := kubevirt.Client().CoreV1().Pods(pod.Namespace).Get(context.Background(), pod.Name, metav1.GetOptions{})
					return err
				}, 60*time.Second, 5*time.Second).Should(MatchError(k8serrors.IsNotFound, "k8serrors.IsNotFound"))
			})
		})
		Context("with ACPI and some grace period seconds", decorators.WgS390x, func() {

			withoutTerminationGracePeriodSeconds := func(vmi *v1.VirtualMachineInstance) {
				vmi.Spec.TerminationGracePeriodSeconds = nil
			}

			DescribeTable("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]should result in vmi status succeeded", func(option libvmi.Option, gracePeriodSeconds int64) {
				vmi := libvmifact.NewFedora(option)

				By("Creating the VirtualMachineInstance")
				vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Wait for the login")
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				By("Deleting the VirtualMachineInstance")
				Expect(kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's status is Succeeded")
				Eventually(matcher.ThisVMI(vmi)).WithTimeout(time.Duration(gracePeriodSeconds) * time.Second).WithPolling(time.Second).Should(
					matcher.BeInPhase(v1.Succeeded))
			},
				Entry("[test_id:1653]with set grace period seconds", decorators.Conformance, libvmi.WithTerminationGracePeriod(10), int64(10)),
				Entry("[test_id:1654]with default grace period seconds", decorators.Conformance, withoutTerminationGracePeriodSeconds, v1.DefaultGracePeriodSeconds),
			)
		})
		Context("with grace period greater than 0", func() {
			It("[test_id:1655]should run graceful shutdown", decorators.Conformance, func() {
				By("Setting a VirtualMachineInstance termination grace period to 5")
				// Give the VirtualMachineInstance a custom grace period
				vmi := libvmifact.NewAlpine(libvmi.WithTerminationGracePeriod(5))

				By("Creating the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTimeout)

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				Expect(kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI gracefully")

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				// Check if the graceful shutdown was logged
				By("Checking that virt-handler logs VirtualMachineInstance graceful shutdown")
				event := watcher.New(vmi).Timeout(30*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.NormalEvent, "ShuttingDown")
				Expect(event).ToNot(BeNil(), "There should be a graceful shutdown")

				// Verify VirtualMachineInstance is killed after grace period expires
				// 5 seconds is grace period, doubling to prevent flakiness
				By("Checking that the VirtualMachineInstance does not exist after grace period")
				event = watcher.New(vmi).Timeout(10*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.NormalEvent, "Deleted")
				Expect(event).ToNot(BeNil(), "There should be a graceful shutdown")

				Eventually(matcher.ThisVMI(vmi)).WithTimeout(15 * time.Second).WithPolling(time.Second).Should(matcher.BeGone())
			})
		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Killed VirtualMachineInstance", Serial, decorators.WgS390x, func() {
		It("[test_id:1656]should be in Failed phase", func() {
			By("Starting a VirtualMachineInstance")
			vmi := libvmifact.NewAlpine()
			vmi, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			vmi = libwait.WaitForSuccessfulVMIStart(vmi)

			By("Killing the VirtualMachineInstance")
			Expect(pkillVMI(kubevirt.Client(), vmi)).To(Succeed(), "Should deploy helper pod to kill VMI")

			// Wait for stop event of the VirtualMachineInstance
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			objectEventWatcher := watcher.New(vmi).Timeout(60 * time.Second).SinceWatchedObjectResourceVersion()
			wp := watcher.WarningsPolicy{FailOnWarnings: true, WarningsIgnoreList: []string{
				"server error. command SyncVMI failed",
				"The VirtualMachineInstance crashed",
				"cannot detect vm",
				"Can not update a VirtualMachineInstance with unresponsive command server",
			},
			}
			objectEventWatcher.SetWarningsPolicy(wp)
			objectEventWatcher.WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

			// Wait for some time and see if a sync event happens on the stopped VirtualMachineInstance
			By("Checking that virt-handler does not try to sync stopped VirtualMachineInstance")
			stoppedVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should refresh VMI to get its current resourceVersion")
			// This is not optimal as we always spend 10 seconds here. Optimally we should wait for the Failed test and verify that no warning were fired.
			event := watcher.New(stoppedVMI).Timeout(10*time.Second).SinceWatchedObjectResourceVersion().WaitNotFor(ctx, watcher.WarningEvent, v1.SyncFailed)
			Expect(event).To(BeNil(), "virt-handler tried to sync on a VirtualMachineInstance in final state")

			By("Checking that the VirtualMachineInstance has 'Failed' phase")
			Expect(matcher.ThisVMI(stoppedVMI)()).To(matcher.BeInPhase(v1.Failed))
		})
	})
})

func getVirtLauncherLogs(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()

	labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

	pods, err := virtCli.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred(), "Should list pods")

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty(), "Should find pod not scheduled for deletion")

	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			Container: "compute",
		}).
		DoRaw(context.Background())
	Expect(err).ToNot(HaveOccurred(), "Should get virt-launcher pod logs")

	return string(logsRaw)
}

func pkillHandler(virtCli kubecli.KubevirtClient, node string) error {
	pod := libpod.RenderPrivilegedPod("vmi-killer", []string{"pkill"}, []string{"-9", "virt-handler"})
	pod.Spec.NodeName = node
	createdPod, err := virtCli.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "Should create helper pod")

	getStatus := func() k8sv1.PodPhase {
		podG, err := virtCli.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Get(context.Background(), createdPod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "Should return current status")
		return podG.Status.Phase
	}

	Eventually(getStatus, 30, 0.5).Should(Equal(k8sv1.PodSucceeded), "Pod should end itself")

	return err
}

func pkillAllLaunchers(virtCli kubecli.KubevirtClient, node string) (*k8sv1.Pod, error) {
	pod := libpod.RenderPrivilegedPod("vmi-killer", []string{"pkill"}, []string{"-9", "virt-launcher"})
	pod.Spec.NodeName = node
	return virtCli.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
}

func pkillVMI(client kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) error {
	node := vmi.Status.NodeName
	if node == "" {
		return fmt.Errorf("VMI %s was not scheduled yet", node)
	}
	pod := libpod.RenderPrivilegedPod("vmi-killer", []string{"pkill"}, []string{"-9", "-f", string(vmi.UID)})
	pod.Spec.NodeName = node
	_, err := client.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
	return err
}

func waitForVMIRebooted(vmi *v1.VirtualMachineInstance, login console.LoginToFunction) {
	By(fmt.Sprintf("Waiting for vmi %s rebooted", vmi.Name))
	Expect(login(vmi)).To(Succeed())
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "last reboot | grep reboot | wc -l\n"},
		&expect.BExp{R: "2"},
	}, 300)).To(Succeed(), "expected reboot record")
}

func withoutACPI() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.Features = &v1.Features{
			ACPI: v1.FeatureState{Enabled: pointer.P(false)},
		}
	}
}
