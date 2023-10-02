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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"

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

	"kubevirt.io/kubevirt/pkg/controller"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
	"kubevirt.io/kubevirt/tests/watcher"
)

var _ = Describe("[rfe_id:273][crit:high][arm64][vendor:cnv-qe@redhat.com][level:component][sig-compute]VMIlifecycle", decorators.SigCompute, decorators.VMIlifecycle, func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var vmi *v1.VirtualMachineInstance

	var allowEmulation *bool

	const fakeLibvirtLogFilters = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	})

	AfterEach(func() {
		// Not every test causes virt-handler to restart, but a few different contexts do.
		// This check is fast and non-intrusive if virt-handler is already running.
		testsuite.EnsureKVMPresent()
	})

	Context("when virt-handler is deleted", Serial, func() {
		It("[Serial][test_id:4716]should label the node with kubevirt.io/schedulable=false", func() {
			pods, err := virtClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).ToNot(BeEmpty())

			pod := pods.Items[0]
			handlerNamespace := pod.GetNamespace()

			By("setting up a watch on Nodes")
			nodeWatch, err := virtClient.CoreV1().Nodes().Watch(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = virtClient.CoreV1().Pods(handlerNamespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
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
		It("[test_id:1619]should success", func() {
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
		})

		It("[test_id:1620]should start it", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)
		})

		It("[test_id:6095]should start in paused state if start strategy set to paused", func() {
			strategy := v1.StartStrategyPaused
			vmi.Spec.StartStrategy = &strategy
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

			By("Unpausing VMI")
			command := clientcmd.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", testsuite.GetTestNamespace(vmi), vmi.Name)
			Expect(command()).To(Succeed())
			Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))
		})

		It("[test_id:1621]should attach virt-launcher to it", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("Found PID for"))
		})

		It("[test_id:3195]should carry annotations to pod", func() {
			vmi.Annotations = map[string]string{
				"testannotation": "annotation from vmi",
			}

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(pod).NotTo(BeNil())

			Expect(pod.Annotations).To(HaveKeyWithValue("testannotation", "annotation from vmi"), "annotation should be carried to the pod")
		})

		It("[test_id:3196]should carry kubernetes and kubevirt annotations to pod", func() {
			vmi.Annotations = map[string]string{
				"kubevirt.io/test":   "test",
				"kubernetes.io/test": "test",
			}

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(pod).NotTo(BeNil())

			Expect(pod.Annotations).To(HaveKey("kubevirt.io/test"), "kubevirt annotation should not be carried to the pod")
			Expect(pod.Annotations).To(HaveKey("kubernetes.io/test"), "kubernetes annotation should not be carried to the pod")
		})

		It("Should prevent eviction when EvictionStratgy: External", func() {
			strategy := v1.EvictionStrategyExternal
			vmi.Spec.EvictionStrategy = &strategy

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(pod).ToNot(BeNil())

			By("calling evict on VMI's pod")
			err = virtClient.CoreV1().Pods(vmi.Namespace).EvictV1beta1(context.Background(), &policyv1beta1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name}})
			// The "too many requests" err is what get's returned when an
			// eviction would invalidate a pdb. This is what we want to see here.
			Expect(k8serrors.IsTooManyRequests(err)).To(BeTrue())

			By("should have evacuation node name set on vmi status")
			Consistently(func() error {

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).ToNot(BeNil())
				Expect(pod.DeletionTimestamp).To(BeNil())

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Status.EvacuationNodeName).ToNot(Equal(""))
				return nil
			}, 20*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		})

		It("[test_id:1622]should log libvirtd logs", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }
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
			var err error
			options := []libvmi.Option{libvmi.WithResourceMemory("32Mi")}
			for k, v := range vmiLabels {
				options = append(options, libvmi.WithLabel(k, v))
			}

			for k, v := range vmiAnnotations {
				options = append(options, libvmi.WithAnnotation(k, v))
			}

			vmi := libvmi.New(options...)

			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }

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

		It("[test_id:1623]should reject POST if validation webhook deems the spec invalid", func() {

			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
			})

			result := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background())

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "VMI should be rejected as unprocessable")

			reviewResponse := &metav1.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).ToNot(HaveOccurred(), "Result should be unmarshallable")

			Expect(reviewResponse.Details.Causes).To(HaveLen(2), "There should be 2 thing wrong in response")
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[1].name"))
			Expect(reviewResponse.Details.Causes[1].Field).To(Equal("spec.domain.devices.disks[2].name"))
		})

		It("[test_id:1624]should reject PATCH if schema is invalid", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Error()
			Expect(err).ToNot(HaveOccurred(), "Send POST successfully")

			// Add a disk without a volume reference (this is in valid)
			patchStr := fmt.Sprintf("{\"apiVersion\":\"kubevirt.io/%s\",\"kind\":\"VirtualMachineInstance\",\"spec\":{\"domain\":{\"devices\":{\"disks\":[{\"disk\":{\"bus\":\"virtio\"},\"name\":\"fakedisk\"}]}}}}", v1.ApiLatestVersion)

			result := virtClient.RestClient().Patch(types.MergePatchType).Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Name(vmi.Name).Body([]byte(patchStr)).Do(context.Background())

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "The entity should be unprocessable")
		})

		Context("when name is longer than 63 characters", func() {
			BeforeEach(func() {
				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Name = "testvmi" + rand.String(63)
			})
			It("[test_id:1625]should start it", func() {
				By("Creating a VirtualMachineInstance with a long name")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "cannot create VirtualMachineInstance %q: %v", vmi.Name, err)
				Expect(len(vmi.Name)).To(BeNumerically(">", 63), "VirtualMachineInstance %q name is not longer than 63 characters", vmi.Name)

				By("Waiting until it starts")
				libwait.WaitForSuccessfulVMIStart(vmi)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "cannot fetch VirtualMachineInstance %q: %v", vmi.Name, err)

				By("Obtaining serial console")
				Expect(console.LoginToAlpine(vmi)).To(Succeed(), "VirtualMachineInstance %q console is not accessible: %v", vmi.Name, err)
			})
		})

		Context("when it already exist", func() {
			It("[test_id:1626]should be rejected", func() {
				By("Creating a VirtualMachineInstance")
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Error()
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				By("Creating the same VirtualMachineInstance second time")
				b, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).DoRaw(context.Background())
				Expect(err).To(HaveOccurred(), "Second VMI should be rejected")
				By("Checking that POST return status equals to 409")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).ToNot(HaveOccurred(), "Response should be decoded successfully from json")
				Expect(status.Code).To(Equal(int32(http.StatusConflict)), "There should be conflict with existing VMI")
			})
		})

		Context("with boot order", func() {
			DescribeTable("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]should be able to boot from selected disk", func(alpineBootOrder uint, cirrosBootOrder uint, consoleText string, wait int) {
				By("defining a VirtualMachineInstance with an Alpine disk")
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/sh\n\necho 'hi'\n")
				By("adding a Cirros Disk")
				tests.AddEphemeralDisk(vmi, "disk2", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))

				By("setting boot order")
				vmi = tests.AddBootOrderToDisk(vmi, "disk0", &alpineBootOrder)
				vmi = tests.AddBootOrderToDisk(vmi, "disk2", &cirrosBootOrder)

				By("starting VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "VMI should be created successfully")

				By("Waiting the VirtualMachineInstance start")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Checking console text")
				err = console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: consoleText},
				}, wait)
				Expect(err).ToNot(HaveOccurred(), "Should match the console in VMI")
			},
				Entry("[test_id:1627]Alpine as first boot", uint(1), uint(2), "Welcome to Alpine", 90),
				Entry("[test_id:1628]Cirros as first boot", uint(2), uint(1), "cirros", 90),
			)
		})

		Context("with user-data", func() {

			Context("without k8s secret", func() {
				It("[test_id:1629][posneg:negative]should not be able to start virt-launcher pod", func() {
					vmi = libvmi.NewCirros()

					for _, volume := range vmi.Spec.Volumes {
						if volume.CloudInitNoCloud != nil {
							spec := volume.CloudInitNoCloud
							spec.UserDataBase64 = ""
							spec.UserDataSecretRef = &k8sv1.LocalObjectReference{Name: "nonexistent"}
							break
						}
					}
					By("Starting a VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectScheduling(vmi, 30)
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					launcher, err := libvmi.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					Expect(err).ToNot(HaveOccurred())
					watcher.New(launcher).
						SinceWatchedObjectResourceVersion().
						Timeout(60*time.Second).
						Watch(ctx, func(event *k8sv1.Event) bool {
							if event.Type == "Warning" && event.Reason == "FailedMount" {
								return true
							}
							return false
						},
							"event of type Warning, reason = FailedMount")
				})

				It("[test_id:1630]should log warning and proceed once the secret is there", func() {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					userData64 := ""
					vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

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
					createdVMI := tests.RunVMIAndExpectScheduling(vmi, 30)
					launcher, err := libvmi.GetPodByVirtualMachineInstance(createdVMI, createdVMI.Namespace)
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
					secret := k8sv1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "nonexistent",
							Namespace: vmi.Namespace,
							Labels: map[string]string{
								util.SecretLabel: "nonexistent",
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							"userdata": []byte(userData64),
						},
					}
					_, err = virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should create secret successfully")

					// Wait for the VirtualMachineInstance to be started, allow warning events to occur
					By("Checking that VirtualMachineInstance start succeeded")
					watcher.New(createdVMI).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(ctx, watcher.NormalEvent, v1.Started)
				})
			})
		})

		Context("with nodeselector", func() {
			It("[test_id:5760]should check if vm's with non existing nodeselector is not running and node selector is not updated", func() {
				vmi := libvmi.NewCirros()
				By("setting nodeselector with non-existing-os label")
				vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelOSStable: "not-existing-os"}
				vmi = tests.RunVMIAndExpectScheduling(vmi, 30)

				pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{})
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
				vmi := libvmi.NewCirros()
				vmi.Spec.NodeSelector = map[string]string{k8sv1.LabelOSStable: "linux"}
				tests.RunVMIAndExpectLaunch(vmi, 60)

				pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{})
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
		})

		Context("when virt-launcher crashes", func() {
			It("[Serial][test_id:1631]should be stopped and have Failed phase", Serial, func() {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")

				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("Crashing the virt-launcher")
				vmiKiller, err := pkillAllLaunchers(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should create vmi-killer pod to kill virt-launcher successfully")
				watcher.New(vmiKiller).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(ctx, watcher.NormalEvent, v1.Started)

				By("Waiting for the vm to be stopped")
				watcher.New(vmi).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}, 10, 1).Should(Equal(v1.Failed), "VMI should be failed")
			})
		})

		Context("[Serial]when virt-handler crashes", Serial, func() {
			// FIXME: This test has the issues that it tests a lot of different timing scenarios in an intransparent way:
			// e.g. virt-handler can die before or after virt-launcher. If we wait until virt-handler is dead before we
			// kill virt-launcher then we don't know if virt-handler already restarted.
			// Also the virt-handler crash-loop plays a role here. We could also change the daemon-set but then we would not check the crash behaviour.
			It("[test_id:1632]should recover and continue management", func() {

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				// Kill virt-handler on the node the VirtualMachineInstance is active on.
				By("Crashing the virt-handler")
				err = pkillHandler(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should kill virt-handler successfully")

				// Crash the VirtualMachineInstance and verify a recovered version of virt-handler processes the crash
				By("Killing the VirtualMachineInstance")
				err = pkillAllVMIs(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should kill VMI successfully")

				// Give virt-handler some time. It can greatly vary when virt-handler will be ready again
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}, 240*time.Second, 1*time.Second).Should(Equal(v1.Failed), "VMI should be failed")

				By(fmt.Sprintf("Waiting for %q %q event after the resource version %q", watcher.WarningEvent, v1.Stopped, vmi.ResourceVersion))
				watcher.New(vmi).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

				By("checking that it can still start VMIs")
				newVMI := libvmi.NewCirros()
				newVMI.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), newVMI)
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(newVMI)
			})
		})

		Context("[Serial]when virt-handler is responsive", Serial, func() {
			It("[test_id:1633]should indicate that a node is ready for vmis", func() {

				By("adding a heartbeat annotation and a schedulable label to the node")
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				for _, node := range nodes.Items {
					Expect(node.Annotations[v1.VirtHandlerHeartbeat]).ToNot(BeEmpty(), "Nodes should have be ready for VMI")
				}

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, v1.NodeSchedulable)), metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")
				timestamp := node.Annotations[v1.VirtHandlerHeartbeat]

				By("setting the schedulable label back to true")
				Eventually(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get nodes successfully")
					return n.Labels[v1.NodeSchedulable]
				}, 5*time.Minute, 2*time.Second).Should(Equal("true"), "Nodes should be schedulable")
				By("updating the heartbeat roughly every minute")
				Expect(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get nodes successfully")
					return n.Labels[v1.VirtHandlerHeartbeat]
				}()).ShouldNot(Equal(timestamp), "Should not have old vmi heartbeat")
			})

			It("[test_id:3198]device plugins should re-register if the kubelet restarts", func() {

				By("starting a VMI on a node")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName

				By("triggering a device plugin re-registration on that node")
				pod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred())

				_, _, err = exec.ExecuteCommandOnPodWithResults(virtClient, pod,
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
				virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(10)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})
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
				newVMI := libvmi.NewCirros()
				newVMI.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), newVMI)
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(newVMI)
			})
		})

		Context("[Serial]when virt-handler is not responsive", Serial, func() {

			var vmi *v1.VirtualMachineInstance
			var nodeName string
			var virtHandler *k8sv1.Pod
			var virtHandlerAvailablePods int32

			BeforeEach(func() {

				// Schedule a vmi and make sure that virt-handler gets evicted from the node where the vmi was started
				vmi = libvmi.NewCirros()
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")

				// Ensure that the VMI is running. This is necessary to ensure that virt-handler is fully responsible for
				// the VMI. Otherwise virt-controller may move the VMI to failed instead of the node controller.
				nodeName = libwait.WaitForSuccessfulVMIStart(vmi,
					libwait.WithFailOnWarnings(false),
					libwait.WithTimeout(180),
				).Status.NodeName

				virtHandler, err = libnode.GetVirtHandlerPod(virtClient, nodeName)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client")

				ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler daemonset")
				// Save virt-handler number of desired pods
				virtHandlerAvailablePods = ds.Status.DesiredNumberScheduled

				kv := util.GetCurrentKv(virtClient)
				kv.Spec.Workloads = &v1.ComponentConfig{
					NodePlacement: &v1.NodePlacement{
						Affinity: &k8sv1.Affinity{
							NodeAffinity: &k8sv1.NodeAffinity{
								RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
									NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
										{MatchExpressions: []k8sv1.NodeSelectorRequirement{
											{Key: "kubernetes.io/hostname", Operator: "NotIn", Values: []string{nodeName}},
										}},
									},
								},
							},
						},
					},
				}
				_, err = virtClient.KubeVirt(kv.Namespace).Update(kv)
				Expect(err).ToNot(HaveOccurred(), "Should update kubevirt infra placement")

				Eventually(func() bool {
					_, err := virtClient.CoreV1().Pods(virtHandler.Namespace).Get(context.Background(), virtHandler.Name, metav1.GetOptions{})
					return k8serrors.IsNotFound(err)
				}, 120*time.Second, 1*time.Second).Should(BeTrue(), "The virthandler pod should be gone")
			})

			It("[test_id:1634]the node controller should mark the node as unschedulable when the virt-handler heartbeat has timedout", func() {
				// Update virt-handler heartbeat, to trigger a timeout
				data := []byte(fmt.Sprintf(`{"metadata": { "labels": { "%s": "true" }, "annotations": {"%s": "%s"}}}`, v1.NodeSchedulable, v1.VirtHandlerHeartbeat, nowAsJSONWithOffset(-10*time.Minute)))
				_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				// Delete vmi pod
				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
				Expect(pods.Items).To(HaveLen(1), "There should be only one VMI pod")
				var gracePeriod int64 = 0
				Expect(virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), pods.Items[0].Name, metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriod,
				})).To(Succeed(), "The vmi pod should be deleted successfully")

				// it will take at least 45 seconds until the vmi is gone, check the schedulable state in the meantime
				By("marking the node as not schedulable")
				Eventually(func() string {
					node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get node successfully")
					return node.Labels[v1.NodeSchedulable]
				}, 20*time.Second, 1*time.Second).Should(Equal("false"), "The node should not be schedulable")

				By("moving stuck vmis to failed state")
				Eventually(func() v1.VirtualMachineInstancePhase {
					failedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi successfully")
					return failedVMI.Status.Phase
				}, 180*time.Second, 1*time.Second).Should(Equal(v1.Failed))

				Eventually(func() string {
					failedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi successfully")
					return failedVMI.Status.Reason
				}, 180*time.Second, 1*time.Second).Should(Equal(watch.NodeUnresponsiveReason))

				err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				testsuite.RestoreKubeVirtResource()

				// Wait until virt-handler ds will have expected number of pods
				Eventually(func() bool {
					ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get virthandler successfully")

					return ds.Status.NumberAvailable == virtHandlerAvailablePods &&
						ds.Status.CurrentNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.DesiredNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.NumberReady == virtHandlerAvailablePods &&
						ds.Status.UpdatedNumberScheduled == virtHandlerAvailablePods
				}, 180*time.Second, 1*time.Second).Should(BeTrue(), "Virthandler should be ready to work")
			})
		})

		Context("[Serial]with node tainted", Serial, func() {
			var nodes *k8sv1.NodeList
			var err error
			BeforeEach(func() {
				Eventually(func() []k8sv1.Node {
					nodes = libnode.GetAllSchedulableNodes(virtClient)
					return nodes.Items
				}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")

				// Taint first node with "NoSchedule"
				data := []byte(`{"spec":{"taints":[{"effect":"NoSchedule","key":"test","timeAdded":null,"value":"123"}]}}`)
				_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodes.Items[0].Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node")

			})

			AfterEach(func() {
				// Untaint first node
				data := []byte(`{"spec":{"taints":[]}}`)
				_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodes.Items[0].Name, types.StrategicMergePatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should patch node")
			})

			It("[test_id:1635]the vmi with tolerations should be scheduled", func() {
				vmi := libvmi.NewCirros(libvmi.WithNodeAffinityFor(&nodes.Items[0]))
				vmi.Spec.Tolerations = []k8sv1.Toleration{{Key: "test", Value: "123"}}
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:1636]the vmi without tolerations should not be scheduled", func() {
				vmi := libvmi.NewCirros(libvmi.WithNodeAffinityFor(&nodes.Items[0]))
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
			var nodes *k8sv1.NodeList
			var err error

			BeforeEach(func() {
				nodes = libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
			})

			It("[test_id:1637]the vmi with node affinity and no conflicts should be scheduled", func() {
				vmi := libvmi.NewCirros(libvmi.WithNodeAffinityFor(&nodes.Items[0]))
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "Updated VMI name run on the same node")

			})

			It("[test_id:1638]the vmi with node affinity and anti-pod affinity should not be scheduled", func() {
				vmi := libvmi.NewCirros(libvmi.WithNodeAffinityFor(&nodes.Items[0]))
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "VMI should run on the same node")

				vmiB := libvmi.NewCirros(libvmi.WithNodeAffinityFor(&nodes.Items[0]))

				vmiB.Spec.Affinity.PodAntiAffinity = &k8sv1.PodAntiAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: []k8sv1.PodAffinityTerm{
						{
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{Key: v1.CreatedByLabel, Operator: metav1.LabelSelectorOpIn, Values: []string{string(curVMI.GetUID())}},
								},
							},
							TopologyKey: "kubernetes.io/hostname",
						},
					},
				}

				vmiB, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmiB)).Create(context.Background(), vmiB)
				Expect(err).ToNot(HaveOccurred(), "Should create VMIB")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVmiB, err := virtClient.VirtualMachineInstance(vmiB.Namespace).Get(context.Background(), vmiB.Name, &metav1.GetOptions{})
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

		Context("[Serial]with default cpu model", Serial, func() {
			var originalConfig v1.KubeVirtConfiguration
			var supportedCpuModels []string
			var defaultCPUModel string
			var vmiCPUModel string

			//store old kubevirt-config
			BeforeEach(func() {
				// arm64 does not support cpu model
				checks.SkipIfARM64(testsuite.Arch, "arm64 does not support cpu model")
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				supportedCpuModels = tests.GetSupportedCPUModels(*nodes)
				if len(supportedCpuModels) < 2 {
					Skip("need at least 2 supported cpuModels for this test")
				}
				defaultCPUModel = supportedCpuModels[0]
				vmiCPUModel = supportedCpuModels[1]
				kv := util.GetCurrentKv(virtClient)
				originalConfig = kv.Spec.Configuration
			})

			//replace new kubevirt-config with old config
			AfterEach(func() {
				tests.UpdateKubeVirtConfigValueAndWait(originalConfig)
			})

			It("[test_id:3199]should set default cpu model when vmi doesn't have it set", func() {
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi := libvmi.NewCirros()

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(defaultCPUModel), "Expected default CPU model")

			})

			It("[test_id:3200]should not set default cpu model when vmi has it set", func() {
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				vmi := libvmi.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Model: vmiCPUModel,
				}
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(vmiCPUModel), "Expected vmi CPU model")

			})

			It("[sig-compute][test_id:3201]should set cpu model to default when vmi does not have it set and default cpu model is not set", func() {
				vmi := libvmi.NewCirros()
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				libwait.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(v1.DefaultCPUModel),
					fmt.Sprintf("Expected CPU model to equal to the default (%v)", v1.DefaultCPUModel),
				)
			})
			It("should add node selector to virt-launcher when setting default cpuModel in kubevirtCR", func() {
				if len(supportedCpuModels) < 1 {
					Skip("Must have at least one supported cpuModel for this test")
				}
				defaultCPUModel := supportedCpuModels[0]
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				newVMI := libvmi.NewCirros()
				newVMI = tests.RunVMIAndExpectLaunch(newVMI, 90)
				By("Fetching virt-launcher pod")
				virtLauncherPod := tests.GetRunningPodByVirtualMachineInstance(newVMI, newVMI.Namespace)
				Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKey(ContainSubstring(defaultCPUModel)), "Node selector for the cpuModel in vmi spec should appear in virt-launcher pod")

			})

			It("should prefer node selector of the vmi if cpuModel field is set in kubevirtCR and in the vmi", func() {
				if len(supportedCpuModels) < 2 {
					Skip("Must have at least one supported cpuModel for this test")
				}
				vmiCPUModel := supportedCpuModels[1]
				defaultCPUModel := supportedCpuModels[0]
				config := originalConfig.DeepCopy()
				config.CPUModel = defaultCPUModel
				tests.UpdateKubeVirtConfigValueAndWait(*config)

				newVMI := libvmi.NewCirros()
				newVMI.Spec.Domain.CPU = &v1.CPU{
					Model: vmiCPUModel,
				}
				newVMI = tests.RunVMIAndExpectLaunch(newVMI, 90)
				By("Fetching virt-launcher pod")
				virtLauncherPod := tests.GetRunningPodByVirtualMachineInstance(newVMI, newVMI.Namespace)
				Expect(virtLauncherPod.Spec.NodeSelector).To(HaveKey(ContainSubstring(vmiCPUModel)), "Node selector for the cpuModel in kubevirtCR should appear in virt-launcher pod")

			})
		})

		Context("[Serial]with node feature discovery", Serial, func() {
			var node *k8sv1.Node
			var supportedCPU string
			var supportedCPUs []string
			var supportedFeatures []string
			var nodes *k8sv1.NodeList
			var supportedKVMInfoFeature []string

			BeforeEach(func() {
				// arm64 does not support cpu model
				checks.SkipIfARM64(testsuite.Arch, "arm64 does not support cpu model")
				nodes = libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

				node = &nodes.Items[0]
				supportedCPUs = tests.GetSupportedCPUModels(*nodes)
				Expect(supportedCPUs).ToNot(BeEmpty(), "There should be some supported cpu models")

				supportedCPU = supportedCPUs[0]

				supportedFeatures = tests.GetSupportedCPUFeatures(*nodes)
				Expect(len(supportedFeatures)).To(BeNumerically(">=", 2), "There should be at least 2 supported cpu features")

				for key := range node.Labels {
					if strings.Contains(key, services.NFD_KVM_INFO_PREFIX) &&
						!strings.Contains(key, "tlbflush") &&
						!strings.Contains(key, "ipi") &&
						!strings.Contains(key, "synictimer") {
						supportedKVMInfoFeature = append(supportedKVMInfoFeature, strings.TrimPrefix(key, services.NFD_KVM_INFO_PREFIX))
					}

				}

				tests.EnableFeatureGate(virtconfig.HypervStrictCheckGate)
			})

			It("[test_id:1639]the vmi with cpu.model matching a nfd label on a node should be scheduled", func() {
				vmi := libvmi.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: supportedCPU,
				}

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)

				By("Verifying VirtualMachineInstance's status is Succeeded")
				Eventually(func() v1.VirtualMachineInstancePhase {
					currVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					return currVMI.Status.Phase
				}, 120, 0.5).Should(Equal(v1.Running), "VMI should be succeeded")
			})

			It("[test_id:1640]the vmi with cpu.model that cannot match an nfd label on node should not be scheduled", func() {
				vmi := libvmi.NewCirros()
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: "486",
				}

				//Make sure the vmi should try to be scheduled only on master node
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node.Name}

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
				vmi := libvmi.NewCirros()
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

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:3203]the vmi with cpu.features that cannot match nfd labels on a node should not be scheduled", func() {
				var featureDenyList = map[string]struct{}{
					"svm": {},
				}
				appendFeatureFromFeatureLabel := func(supportedFeatures []string, label string) []string {
					if strings.Contains(label, services.NFD_CPU_FEATURE_PREFIX) {
						feature := strings.TrimPrefix(label, services.NFD_CPU_FEATURE_PREFIX)
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
				vmi := libvmi.NewCirros()
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

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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
				// the specific node - the feature policy 'forbid' will deny shceduling on that node.
				vmi := libvmi.NewCirros(libvmi.WithNodeAffinityFor(&nodes.Items[0]))
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   supportedFeatures[0],
							Policy: "forbid",
						},
					},
				}

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
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

				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				By("Creating a VirtualMachineInstance with different namespace")
				vmi = libvmi.New(
					libvmi.WithResourceMemory("1Mi"),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				)
				virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, node)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				// Make sure we schedule the VirtualMachineInstance to master
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				// Start the VirtualMachineInstance and wait for the confirmation of the start
				vmi, err = virtClient.VirtualMachineInstance(namespace).Create(context.Background(), vmi)
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
				_, err = virtClient.RestClient().Delete().Resource("virtualmachineinstances").Namespace(vmi.GetObjectMeta().GetNamespace()).Name(vmi.GetObjectMeta().GetName()).Do(context.Background()).Get()
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

		Context("VirtualMachineInstance Emulation Mode", func() {
			BeforeEach(func() {
				// allowEmulation won't change in a test suite run, so cache it
				if allowEmulation == nil {
					emulation := testsuite.ShouldAllowEmulation(virtClient)
					allowEmulation = &emulation
				}
				if !(*allowEmulation) {
					Skip("Software emulation is not enabled on this cluster")
				}
			})

			It("[test_id:1643]should enable emulation in virt-launcher", func() {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())

				emulationFlagFound := false
				computeContainerFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "compute" {
						computeContainerFound = true
						for _, cmd := range container.Command {
							By(cmd)
							if cmd == "--allow-emulation" {
								emulationFlagFound = true
							}
						}
					}
				}

				Expect(computeContainerFound).To(BeTrue(), "Compute container was not found in pod")
				Expect(emulationFlagFound).To(BeTrue(), "Expected VirtualMachineInstance pod to have '--allow-emulation' flag")
			})

			It("[test_id:1644]should be reflected in domain XML", func() {
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Error()
				Expect(err).ToNot(HaveOccurred(), "Should post the VMI")

				listOptions := metav1.ListOptions{}

				Eventually(func() int {
					podList, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), listOptions)
					Expect(err).ToNot(HaveOccurred(), "Should list the pods")
					return len(podList.Items)
				}, 75, 0.5).Should(Equal(1), "There should be only one pod")

				Eventually(func() error {
					podList, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), listOptions)
					Expect(err).ToNot(HaveOccurred(), "Should list the pods")
					for _, item := range podList.Items {
						if strings.HasPrefix(item.Name, vmi.ObjectMeta.GenerateName) {
							return nil
						}
					}
					return fmt.Errorf("Associated pod for VirtualMachineInstance '%s' not found", vmi.Name)
				}, 75, 0.5).Should(Succeed(), "Should find the VMI pod")

				getOptions := metav1.GetOptions{}
				var newVMI *v1.VirtualMachineInstance

				newVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(newVMI)).Get(context.Background(), vmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")

				domain := &api.Domain{}
				context := &converter.ConverterContext{
					VirtualMachine: newVMI,
					AllowEmulation: true,
				}
				converter.Convert_v1_VirtualMachineInstance_To_api_Domain(newVMI, domain, context)

				expectedType := ""
				if _, err := os.Stat("/dev/kvm"); errors.Is(err, os.ErrNotExist) {
					expectedType = "qemu"
				}

				Expect(domain.Spec.Type).To(Equal(expectedType), "VMI domain type should be of expectedType")
			})

			It("[test_id:1645]should request a TUN device but not KVM", func() {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())

				computeContainerFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "compute" {
						computeContainerFound = true

						_, ok := container.Resources.Limits[services.KvmDevice]
						Expect(ok).To(BeFalse(), "Container should not have requested KVM device")

						_, ok = container.Resources.Limits[services.TunDevice]
						Expect(ok).To(BeTrue(), "Container should have requested TUN device")
					}
				}

				Expect(computeContainerFound).To(BeTrue(), "Compute container was not found in pod")
			})
		})

		Context("VM Accelerated Mode", func() {
			BeforeEach(func() {
				// allowEmulation won't change in a test suite run, so cache it
				if allowEmulation == nil {
					emulation := testsuite.ShouldAllowEmulation(virtClient)
					allowEmulation = &emulation
				}
				if *allowEmulation {
					Skip("Software emulation is enabled on this cluster")
				}
			})

			It("[test_id:1646]should request a KVM and TUN device", func() {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())

				computeContainerFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "compute" {
						computeContainerFound = true

						_, ok := container.Resources.Limits[services.KvmDevice]
						Expect(ok).To(BeTrue(), "Container should have requested KVM device")

						_, ok = container.Resources.Limits[services.TunDevice]
						Expect(ok).To(BeTrue(), "Container should have requested TUN device")
					}
				}

				Expect(computeContainerFound).To(BeTrue(), "Compute container was not found in pod")
			})

			It("[test_id:1647]should not enable emulation in virt-launcher", func() {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				libwait.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())

				emulationFlagFound := false
				computeContainerFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "compute" {
						computeContainerFound = true
						for _, cmd := range container.Command {
							By(cmd)
							if cmd == "--allow-emulation" {
								emulationFlagFound = true
							}
						}
					}
				}

				Expect(computeContainerFound).To(BeTrue(), "Compute container was not found in pod")
				Expect(emulationFlagFound).To(BeFalse(), "Expected VM pod not to have '--allow-emulation' flag")
			})

			It("[test_id:1648]Should provide KVM via plugin framework", func() {
				nodeList := libnode.GetAllSchedulableNodes(virtClient)

				if len(nodeList.Items) == 0 {
					Skip("There are no compute nodes in cluster")
				}
				node := nodeList.Items[0]

				_, ok := node.Status.Allocatable[services.KvmDevice]
				Expect(ok).To(BeTrue(), "KVM devices not allocatable on node: %s", node.Name)

				_, ok = node.Status.Capacity[services.KvmDevice]
				Expect(ok).To(BeTrue(), "No Capacity for KVM devices on node: %s", node.Name)
			})
		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Get a VirtualMachineInstance", func() {
		Context("when that not exist", func() {
			It("[test_id:1649]should return 404", func() {
				b, err := virtClient.RestClient().Get().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Name("nonexistnt").DoRaw(context.Background())
				Expect(err).To(HaveOccurred(), "Should get VMIs")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).ToNot(HaveOccurred(), "Unmarshal without error")
				Expect(status.Code).To(Equal(int32(http.StatusNotFound)), "There should not be and VMI")
			})
		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Delete a VirtualMachineInstance's Pod", func() {
		It("[test_id:1650]should result in the VirtualMachineInstance moving to a finalized state", func() {
			By("Creating the VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")
			libwait.WaitForSuccessfulVMIStart(obj)

			By("Verifying VirtualMachineInstance's pod is active")
			pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), tests.UnfinishedVMIPodSelector(vmi))
			Expect(err).ToNot(HaveOccurred(), "Should list pods")
			Expect(pods.Items).To(HaveLen(1), "There should be only one pod")
			pod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "Should delete VMI pod")

			// Wait for VirtualMachineInstance to finalize
			By("Waiting for the VirtualMachineInstance to move to a finalized state")
			Eventually(func() error {
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				} else if !curVMI.IsFinal() {
					return fmt.Errorf("VirtualMachineInstance has not reached a finalized state yet")
				}
				return nil
			}, 60*time.Second, 1*time.Second).Should(Succeed(), "VMI reached finalized state")
		})
	})
	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Delete a VirtualMachineInstance", func() {
		Context("with an active pod.", func() {
			It("[test_id:1651]should result in pod being terminated", func() {

				By("Creating the VirtualMachineInstance")
				obj, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(obj)

				podSelector := tests.UnfinishedVMIPodSelector(vmi)
				By("Verifying VirtualMachineInstance's pod is active")
				pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), podSelector)
				Expect(err).ToNot(HaveOccurred(), "Should list pods")
				Expect(pods.Items).To(HaveLen(1), "There should be only one pod")

				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), obj.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's pod terminates")
				Eventually(func() int {
					pods, err := virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), podSelector)
					Expect(err).ToNot(HaveOccurred(), "Should list pods")
					return len(pods.Items)
				}, 75, 0.5).Should(Equal(0), "There should be no pods")

			})
		})
		Context("with ACPI and some grace period seconds", func() {
			DescribeTable("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]should result in vmi status succeeded", func(gracePeriod int64) {
				vmi := libvmi.NewCirros()

				if gracePeriod >= 0 {
					vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
				} else {
					gracePeriod = v1.DefaultGracePeriodSeconds
					vmi.Spec.TerminationGracePeriodSeconds = nil
				}

				By("Creating the VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				// wait until booted
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's status is Succeeded")
				Eventually(func() v1.VirtualMachineInstancePhase {
					currVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					return currVMI.Status.Phase
				}, gracePeriod+5, 0.5).Should(Equal(v1.Succeeded), "VMI should be succeeded")
			},
				Entry("[test_id:1653]with set grace period seconds", int64(10)),
				Entry("[test_id:1654]with default grace period seconds", int64(-1)),
			)
		})
		Context("with grace period greater than 0", func() {
			It("[test_id:1655]should run graceful shutdown", func() {
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				virtHandlerPod, err := libnode.GetVirtHandlerPod(virtClient, node)
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				By("Setting a VirtualMachineInstance termination grace period to 5")
				var gracePeriod int64
				gracePeriod = int64(5)
				// Give the VirtualMachineInstance a custom grace period
				vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
				// Make sure we schedule the VirtualMachineInstance to master
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				By("Creating the VirtualMachineInstance")
				obj, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				libwait.WaitForSuccessfulVMIStart(obj)

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), obj.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI gracefully")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				event := watcher.New(obj).SinceWatchedObjectResourceVersion().Timeout(75*time.Second).WaitFor(ctx, watcher.NormalEvent, v1.Deleted)
				Expect(event).ToNot(BeNil(), "There should be a delete event")

				// Check if the graceful shutdown was logged
				By("Checking that virt-handler logs VirtualMachineInstance graceful shutdown")
				Eventually(func() string {
					data, err := logsQuery.DoRaw(context.Background())
					Expect(err).ToNot(HaveOccurred(), "Should get the logs")
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())), "Should log graceful shutdown")

				// Verify VirtualMachineInstance is killed after grace period expires
				By("Checking that the VirtualMachineInstance does not exist after grace period")
				Eventually(func() string {
					data, err := logsQuery.DoRaw(context.Background())
					Expect(err).ToNot(HaveOccurred(), "Should get logs")
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())), "Should log graceful kill")
			})
		})
	})

	Describe("[Serial][rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Killed VirtualMachineInstance", Serial, func() {
		It("[test_id:1656]should be in Failed phase", func() {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			nodeName := libwait.WaitForSuccessfulVMIStart(obj).Status.NodeName

			By("Killing the VirtualMachineInstance")
			time.Sleep(10 * time.Second)
			err = pkillAllVMIs(virtClient, nodeName)
			Expect(err).ToNot(HaveOccurred(), "Should deploy helper pod to kill VMI")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			watcher.New(obj).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

			By("Checking that the VirtualMachineInstance has 'Failed' phase")
			Eventually(func() v1.VirtualMachineInstancePhase {
				failedVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				return failedVMI.Status.Phase
			}, 10, 1).Should(Equal(v1.Failed), "VMI should be failed")

		})

		It("[test_id:1657]should be left alone by virt-handler", func() {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			nodeName := libwait.WaitForSuccessfulVMIStart(obj).Status.NodeName

			By("Killing the VirtualMachineInstance")
			err = pkillAllVMIs(virtClient, nodeName)
			Expect(err).ToNot(HaveOccurred(), "Should create kill pod to kill all VMs")

			// Wait for stop event of the VirtualMachineInstance
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			objectEventWatcher := watcher.New(obj).Timeout(60 * time.Second).SinceWatchedObjectResourceVersion()
			wp := watcher.WarningsPolicy{FailOnWarnings: true, WarningsIgnoreList: []string{"server error. command SyncVMI failed", "The VirtualMachineInstance crashed"}}
			objectEventWatcher.SetWarningsPolicy(wp)
			objectEventWatcher.WaitFor(ctx, watcher.WarningEvent, v1.Stopped)

			// Wait for some time and see if a sync event happens on the stopped VirtualMachineInstance
			By("Checking that virt-handler does not try to sync stopped VirtualMachineInstance")
			stoppedVMI, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should refresh VMI to get its current resourceVersion")
			event := watcher.New(stoppedVMI).Timeout(10*time.Second).SinceWatchedObjectResourceVersion().WaitNotFor(ctx, watcher.WarningEvent, v1.SyncFailed)
			Expect(event).To(BeNil(), "virt-handler tried to sync on a VirtualMachineInstance in final state")
		})
	})

	Context("replicaset with topology spread constraints", func() {
		It("Replicas should be spread across nodes", func() {
			nodes := libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some schedulable nodes")
			numNodes := len(nodes.Items)
			if numNodes < 2 {
				Skip("Skipping spec if test environment has less than two schedulable nodes")
			}
			vmLabelKey := "test" + rand.String(5)
			vmLabelValue := "test" + rand.String(5)
			vmi := tests.NewRandomVMI()
			vmi.Spec.TopologySpreadConstraints = []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "kubernetes.io/hostname",
					WhenUnsatisfiable: "DoNotSchedule",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							vmLabelKey: vmLabelValue,
						},
					},
				},
			}

			By("Creating a VirtualMachineInstanceReplicaSet")
			replicas := int32(numNodes)
			// limit the number of replicas launched for this test
			if replicas > 10 {
				replicas = 10
			}
			rs := &v1.VirtualMachineInstanceReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: "replicaset" + rand.String(5)},
				Spec: v1.VirtualMachineInstanceReplicaSetSpec{
					Replicas: &replicas,
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{vmLabelKey: vmLabelValue},
					},
					Template: &v1.VirtualMachineInstanceTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{vmLabelKey: vmLabelValue},
							Name:   vmi.ObjectMeta.Name,
						},
						Spec: vmi.Spec,
					},
				},
			}
			createdRs, err := virtClient.ReplicaSet(vmi.Namespace).Create(rs)
			Expect(err).ToNot(HaveOccurred(), "Should create replicaset")

			By("Ensuring that all VMIs are ready")
			Eventually(func() int32 {
				rs, err := virtClient.ReplicaSet(vmi.Namespace).Get(createdRs.ObjectMeta.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return rs.Status.ReadyReplicas
			}, 120*time.Second, 1*time.Second).Should(Equal(replicas))

			By("Ensuring that VMI replicas are scheduled to seperate nodes")
			vmiSet, err := virtClient.VirtualMachineInstance(vmi.Namespace).List(context.Background(), &metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", vmLabelKey, vmLabelValue),
			})
			Expect(err).ToNot(HaveOccurred(), "Should find VMIs by label")
			Expect(vmiSet.Items).To(HaveLen(int(replicas)), "Should get expected number of VMIs")

			nodeNames := make(map[string]bool)
			for _, vmi := range vmiSet.Items {
				nodeName := vmi.Status.NodeName
				_, nodeHasReplica := nodeNames[nodeName]
				Expect(nodeHasReplica).To(BeFalse(), "Multiple replicas should not be scheduled to the same node")
				nodeNames[nodeName] = true
			}
		})
	})
})

func renderPkillAllPod(processName string) *k8sv1.Pod {
	return tests.RenderPrivilegedPod("vmi-killer", []string{"pkill"}, []string{"-9", processName})
}

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
	pod := renderPkillAllPod("virt-handler")
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
	pod := renderPkillAllPod("virt-launcher")
	pod.Spec.NodeName = node
	return virtCli.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})
}

func pkillAllVMIs(virtCli kubecli.KubevirtClient, node string) error {
	pod := renderPkillAllPod("qemu")
	pod.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(testsuite.GetTestNamespace(pod)).Create(context.Background(), pod, metav1.CreateOptions{})

	return err
}

func nowAsJSONWithOffset(offset time.Duration) string {
	now := metav1.Now()
	now = metav1.NewTime(now.Add(offset))

	data, err := json.Marshal(now)
	Expect(err).ToNot(HaveOccurred(), "Should marshal to json")
	return strings.Trim(string(data), `"`)
}
