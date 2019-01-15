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
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/controller"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
)

func newCirrosVMI() *v1.VirtualMachineInstance {
	return tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
}

func addNodeAffinityToVMI(vmi *v1.VirtualMachineInstance, nodeName string) {
	vmi.Spec.Affinity = &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{Key: "kubernetes.io/hostname", Operator: k8sv1.NodeSelectorOpIn, Values: []string{nodeName}},
						},
					},
				},
			},
		},
	}
}

var _ = Describe("VMIlifecycle", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

	var useEmulation *bool

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
	})

	AfterEach(func() {
		// Not every test causes virt-handler to restart, but a few different contexts do.
		// This check is fast and non-intrusive if virt-handler is already running.
		tests.EnsureKVMPresent()
	})

	Describe("Creating a VirtualMachineInstance", func() {
		It("should success", func() {
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
		})

		It("should start it", func() {
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)
		})

		It("should attach virt-launcher to it", func() {
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("Found PID for"))
		})

		It("should log libvirtd logs", func() {
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("libvirt version: "))
			Eventually(logs,
				2*time.Second,
				500*time.Millisecond).
				Should(And(ContainSubstring("internal error: Cannot probe for supported suspend types"), ContainSubstring(`"subcomponent":"libvirt"`)))
		})
		It("should reject POST if validation webhook deems the spec invalid", func() {

			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: "testdisk2",
			})

			result := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "VMI should be rejected as unprocessable")

			reviewResponse := &metav1.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).To(BeNil(), "Result should be unmarshallable")

			Expect(len(reviewResponse.Details.Causes)).To(Equal(2), "There should be 2 thing wrong in response")
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[1].name"))
			Expect(reviewResponse.Details.Causes[1].Field).To(Equal("spec.domain.devices.disks[2].name"))
		})

		It("should reject PATCH if schema is invalid", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
			Expect(err).To(BeNil(), "Send POST successfully")

			// Add a disk without a volume reference (this is in valid)
			patchStr := "{\"apiVersion\":\"kubevirt.io/v1alpha3\",\"kind\":\"VirtualMachineInstance\",\"spec\":{\"domain\":{\"devices\":{\"disks\":[{\"disk\":{\"bus\":\"virtio\"},\"name\":\"fakedisk\"}]}}}}"

			result := virtClient.RestClient().Patch(types.MergePatchType).Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Name(vmi.Name).Body([]byte(patchStr)).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity), "The entity should be unprocessable")
		})

		Context("when name is longer than 63 characters", func() {
			BeforeEach(func() {
				vmi = tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskAlpine))
				vmi.Name = "testvmi" + rand.String(63)
			})
			It("should start it", func() {
				By("Creating a VirtualMachineInstance with a long name")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "cannot create VirtualMachineInstance %q: %v", vmi.Name, err)
				Expect(len(vmi.Name)).To(BeNumerically(">", 63), "VirtualMachineInstance %q name is not longer than 63 characters", vmi.Name)

				By("Waiting until it starts")
				tests.WaitForSuccessfulVMIStart(vmi)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "cannot fetch VirtualMachineInstance %q: %v", vmi.Name, err)
				Eventually(controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi, v1.VirtualMachineInstanceConditionType(k8sv1.PodReady), k8sv1.ConditionTrue), 2*time.Second, 100*time.Millisecond).Should(BeTrue(), "VirtualMachineInstance %q is not ready: %v", vmi.Name, vmi.Status.Phase)

				By("Obtaining serial console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "VirtualMachineInstance %q console is not accessible: %v", vmi.Name, err)
				expecter.Close()
			})
		})

		Context("when it already exist", func() {
			It("should be rejected", func() {
				By("Creating a VirtualMachineInstance")
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
				Expect(err).To(BeNil(), "Should create VMI successfully")
				By("Creating the same VirtualMachineInstance second time")
				b, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).DoRaw()
				Expect(err).ToNot(BeNil(), "Second VMI should be rejected")
				By("Checking that POST return status equals to 409")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil(), "Response should be decoded successfully from json")
				Expect(status.Code).To(Equal(int32(http.StatusConflict)), "There should be conflict with existing VMI")
			})
		})

		Context("with boot order", func() {
			table.DescribeTable("should be able to boot from selected disk", func(alpineBootOrder uint, cirrosBootOrder uint, consoleText string, wait int) {
				By("defining a VirtualMachineInstance with an Alpine disk")
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(tests.ContainerDiskFor(tests.ContainerDiskAlpine), "#!/bin/sh\n\necho 'hi'\n")
				By("adding a Cirros Disk")
				tests.AddEphemeralDisk(vmi, "disk2", "virtio", tests.ContainerDiskFor(tests.ContainerDiskCirros))

				By("setting boot order")
				vmi = tests.AddBootOrderToDisk(vmi, "disk0", &alpineBootOrder)
				vmi = tests.AddBootOrderToDisk(vmi, "disk2", &cirrosBootOrder)

				By("starting VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "VMI should be created successfully")

				By("Waiting the VirtualMachineInstance start")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Checking console text")
				err = tests.CheckForTextExpecter(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: consoleText},
				}, wait)
				Expect(err).ToNot(HaveOccurred(), "Should match the console in VMI")
			},
				table.Entry("Alpine as first boot", uint(1), uint(2), "Welcome to Alpine", 90),
				table.Entry("Cirros as first boot", uint(2), uint(1), "cirros", 90),
			)
		})

		Context("with user-data", func() {
			Context("without k8s secret", func() {
				It("should retry starting the VirtualMachineInstance", func() {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), userData)

					for _, volume := range vmi.Spec.Volumes {
						if volume.CloudInitNoCloud != nil {
							spec := volume.CloudInitNoCloud
							spec.UserDataBase64 = ""
							spec.UserDataSecretRef = &k8sv1.LocalObjectReference{Name: "nonexistent"}
							break
						}
					}
					By("Starting a VirtualMachineInstance")
					obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					Expect(err).To(BeNil(), "Should create VMI successfully")

					By("Checking that VirtualMachineInstance was restarted twice")
					retryCount := 0
					stopChan := make(chan struct{})
					defer close(stopChan)
					tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).Watch(stopChan, func(event *k8sv1.Event) bool {
						if event.Type == "Warning" && event.Reason == v1.SyncFailed.String() {
							retryCount++
							if retryCount >= 2 {
								// Done, two retries is enough
								return true
							}
						}
						return false
					})
				})

				It("should log warning and proceed once the secret is there", func() {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					userData64 := ""
					vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), userData)

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
					createdVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
					Expect(err).To(BeNil(), "Should create VMI successfully")

					// Wait until we see that starting the VirtualMachineInstance is failing
					By("Checking that VirtualMachineInstance start failed")
					stopChan := make(chan struct{})
					defer close(stopChan)
					event := tests.NewObjectEventWatcher(createdVMI).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.WarningEvent, v1.SyncFailed)
					Expect(event.Message).To(ContainSubstring("nonexistent"), "VMI should not be started")

					// Creat nonexistent secret, so that the VirtualMachineInstance can recover
					By("Creating a user-data secret")
					secret := k8sv1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "nonexistent",
							Namespace: vmi.Namespace,
							Labels: map[string]string{
								tests.SecretLabel: "nonexistent",
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							"userdata": []byte(userData64),
						},
					}
					_, err = virtClient.CoreV1().Secrets(vmi.Namespace).Create(&secret)
					Expect(err).ToNot(HaveOccurred(), "Should create secret successfully")

					// Wait for the VirtualMachineInstance to be started, allow warning events to occur
					By("Checking that VirtualMachineInstance start succeeded")
					tests.NewObjectEventWatcher(createdVMI).SinceWatchedObjectResourceVersion().Timeout(30*time.Second).WaitFor(stopChan, tests.NormalEvent, v1.Started)
				})
			})
		})

		Context("when virt-launcher crashes", func() {
			It("should be stopped and have Failed phase", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Should create VMI successfully")

				nodeName := tests.WaitForSuccessfulVMIStart(vmi)

				By("Crashing the virt-launcher")
				err = pkillAllLaunchers(virtClient, nodeName)
				Expect(err).To(BeNil(), "Should kill virt-launcher successfully")

				By("Waiting for the vm to be stopped")
				stopChan := make(chan struct{})
				defer close(stopChan)
				tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(stopChan, tests.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}, 10, 1).Should(Equal(v1.Failed), "VMI should be failed")
			})
		})

		Context("when virt-handler crashes", func() {
			// FIXME: This test has the issues that it tests a lot of different timing scenarios in an intransparent way:
			// e.g. virt-handler can die before or after virt-launcher. If we wait until virt-handler is dead before we
			// kill virt-launcher then we don't know if virt-handler already restarted.
			// Also the virt-handler crash-loop plays a role here. We could also change the daemon-set but then we would not check the crash behaviour.
			It("should recover and continue management", func() {

				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := tests.WaitForSuccessfulVMIStart(vmi)

				// Kill virt-handler on the node the VirtualMachineInstance is active on.
				By("Crashing the virt-handler")
				err = pkillHandler(virtClient, nodeName)
				Expect(err).To(BeNil(), "Should kill virt-handler successfully")

				// Crash the VirtualMachineInstance and verify a recovered version of virt-handler processes the crash
				By("Killing the VirtualMachineInstance")
				err = pkillAllVMIs(virtClient, nodeName)
				Expect(err).To(BeNil(), "Should kill VMI successfully")

				// Give virt-handler some time. It can greatly vary when virt-handler will be ready again
				stopChan := make(chan struct{})
				defer close(stopChan)
				tests.NewObjectEventWatcher(vmi).Timeout(120*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}(), 10, 1).Should(Equal(v1.Failed), "VMI should be failed")
			})
		})

		Context("when virt-handler is responsive", func() {
			It("should indicate that a node is ready for vmis", func() {

				By("adding a heartbeat annotation and a schedulable label to the node")
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true"})
				Expect(err).ToNot(HaveOccurred(), "Should list nodes successfully")
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some nodes")
				for _, node := range nodes.Items {
					Expect(node.Annotations[v1.VirtHandlerHeartbeat]).ToNot(HaveLen(0), "Nodes should have be ready for VMI")
				}

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, v1.NodeSchedulable)))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")
				timestamp := node.Annotations[v1.VirtHandlerHeartbeat]

				By("setting the schedulable label back to true")
				Eventually(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get nodes successfully")
					return n.Labels[v1.NodeSchedulable]
				}, 2*time.Minute, 2*time.Second).Should(Equal("true"), "Nodes should be schedulable")
				By("updating the heartbeat roughly every minute")
				Expect(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get nodes successfully")
					return n.Labels[v1.VirtHandlerHeartbeat]
				}()).ShouldNot(Equal(timestamp), "Should not have old vmi heartbeat")
			})
		})

		Context("when virt-handler is not responsive", func() {

			var vmi *v1.VirtualMachineInstance
			var nodeName string
			var virtHandler *k8sv1.Pod
			var virtHandlerAvailablePods int32

			BeforeEach(func() {

				// Schedule a vmi and make sure that virt-handler gets evicted from the node where the vmi was started
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "echo hi!")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				nodeName = tests.WaitForSuccessfulVMIStart(vmi)
				virtHandler, err = kubecli.NewVirtHandlerClient(virtClient).ForNode(nodeName).Pod()
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client")
				ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get("virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler daemonset")
				// Save virt-handler number of desired pods
				virtHandlerAvailablePods = ds.Status.DesiredNumberScheduled
				ds.Spec.Template.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: &k8sv1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
							NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
								{MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{Key: "kubernetes.io/hostname", Operator: "NotIn", Values: []string{nodeName}},
								}},
							},
						},
					},
				}
				_, err = virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Update(ds)
				Expect(err).ToNot(HaveOccurred(), "Should update virthandler daemonset")
				Eventually(func() bool {
					_, err := virtClient.CoreV1().Pods(virtHandler.Namespace).Get(virtHandler.Name, metav1.GetOptions{})
					return errors.IsNotFound(err)
				}, 90*time.Second, 1*time.Second).Should(BeTrue(), "The virthandler pod should be gone")
			})

			It("the node controller should react", func() {

				// Update virt-handler heartbeat, to trigger a timeout
				data := []byte(fmt.Sprintf(`{"metadata": { "annotations": {"%s": "%s"}}}`, v1.VirtHandlerHeartbeat, nowAsJSONWithOffset(-10*time.Minute)))
				_, err = virtClient.CoreV1().Nodes().Patch(nodeName, types.StrategicMergePatchType, data)
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				// Delete vmi pod
				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
				Expect(pods.Items).To(HaveLen(1), "There should be only one VMI pod")
				Expect(virtClient.CoreV1().Pods(vmi.Namespace).Delete(pods.Items[0].Name, &metav1.DeleteOptions{})).To(Succeed(), "The vmi pod should be deleted successfully")

				// it will take at least 45 seconds until the vmi is gone, check the schedulable state in the meantime
				By("marking the node as not schedulable")
				Eventually(func() string {
					node, err := virtClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get node successfully")
					return node.Labels[v1.NodeSchedulable]
				}, 20*time.Second, 1*time.Second).Should(Equal("false"), "The node should not be schedulable")

				By("moving stuck vmis to failed state")
				Eventually(func() v1.VirtualMachineInstancePhase {
					failedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi successfully")
					return failedVMI.Status.Phase
				}, 180*time.Second, 1*time.Second).Should(Equal(v1.Failed))
				failedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(failedVMI.Status.Reason).To(Equal(watch.NodeUnresponsiveReason))
			})

			AfterEach(func() {
				// Restore virt-handler daemonset
				ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get("virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler successfully")
				ds.Spec.Template.Spec.Affinity = nil
				_, err = virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Update(ds)
				Expect(err).ToNot(HaveOccurred(), "Should update virthandler successfully")

				// Wait until virt-handler ds will have expected number of pods
				Eventually(func() bool {
					ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get("virt-handler", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get virthandler successfully")

					return ds.Status.NumberAvailable == virtHandlerAvailablePods &&
						ds.Status.CurrentNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.DesiredNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.NumberReady == virtHandlerAvailablePods &&
						ds.Status.UpdatedNumberScheduled == virtHandlerAvailablePods
				}, 60*time.Second, 1*time.Second).Should(Equal(true), "Virthandler should be ready to work")
			})
		})

		Context("with node tainted", func() {
			var nodes *k8sv1.NodeList
			var err error
			BeforeEach(func() {
				nodes, err = virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should list nodes")
				Expect(nodes.Items).NotTo(BeEmpty(), "There should be some node")

				// Taint first node with "NoSchedule"
				data := []byte(`{"spec":{"taints":[{"effect":"NoSchedule","key":"test","timeAdded":null,"value":"123"}]}}`)
				_, err = virtClient.CoreV1().Nodes().Patch(nodes.Items[0].Name, types.StrategicMergePatchType, data)
				Expect(err).ToNot(HaveOccurred(), "Should patch node")

			})

			AfterEach(func() {
				// Untaint first node
				data := []byte(`{"spec":{"taints":[]}}`)
				_, err = virtClient.CoreV1().Nodes().Patch(nodes.Items[0].Name, types.StrategicMergePatchType, data)
				Expect(err).ToNot(HaveOccurred(), "Should patch node")
			})

			It("the vmi with tolerations should be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Tolerations = []k8sv1.Toleration{{Key: "test", Value: "123"}}
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
			})

			It("the vmi without tolerations should not be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					if curVMI.Status.Conditions != nil {
						return curVMI.Status.Conditions[0].Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal("Unschedulable"), "VMI should be unschedulable")
			})
		})

		Context("with affinity", func() {
			var nodes *k8sv1.NodeList
			var err error

			BeforeEach(func() {
				nodes, err = virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should list nodes")
				Expect(nodes.Items).NotTo(BeEmpty(), "There should be some node")
			})

			It("the vmi with node affinity and no conflicts should be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "Updated VMI name run on the same node")

			})

			It("the vmi with node affinity and anti-pod affinity should not be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "VMI should run on the same node")

				vmiB := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				addNodeAffinityToVMI(vmiB, nodes.Items[0].Name)

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

				_, err = virtClient.VirtualMachineInstance(vmiB.Namespace).Create(vmiB)
				Expect(err).ToNot(HaveOccurred(), "Should create VMIB")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVmiB, err := virtClient.VirtualMachineInstance(vmiB.Namespace).Get(vmiB.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMIB")
					if curVmiB.Status.Conditions != nil {
						return curVmiB.Status.Conditions[0].Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal("Unschedulable"), "VMI should be unchedulable")

			})

		})

		Context("with node feature discovery", func() {

			var options metav1.GetOptions
			var cfgMap *k8sv1.ConfigMap
			var originalData string

			BeforeEach(func() {
				options = metav1.GetOptions{}
				cfgMap, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Get("kubevirt-config", options)
				Expect(err).ToNot(HaveOccurred())
				originalData = cfgMap.Data[virtconfig.FeatureGatesKey]
				cfgMap.Data[virtconfig.FeatureGatesKey] = virtconfig.CPUNodeDiscoveryGate
				_, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Update(cfgMap)
				Expect(err).ToNot(HaveOccurred())
				//FIXME improve the detection if virt-controller already received the config-map change
				time.Sleep(time.Millisecond * 500)

			})

			AfterEach(func() {
				cfgMap, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Get("kubevirt-config", options)
				Expect(err).ToNot(HaveOccurred())
				cfgMap.Data[virtconfig.FeatureGatesKey] = originalData
				_, err = virtClient.CoreV1().ConfigMaps(namespaceKubevirt).Update(cfgMap)
				Expect(err).ToNot(HaveOccurred())
				//FIXME improve the detection if virt-controller already received the config-map change
				time.Sleep(time.Millisecond * 500)
			})

			It("the vmi with cpu.model matching a nfd label on a node should be scheduled", func() {

				By("adding a node-feature-discovery CPU model label to a node")
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should list nodes")
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some nodes")

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: "Conroe",
				}

				cpuModelLabel, err := services.CPUModelLabelFromCPUModel(vmi)
				Expect(err).ToNot(HaveOccurred(), "CPU model label should have been retrieved successfully")

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, cpuModelLabel)))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "Updated VMI name run on the node with matching NFD CPU label")

			})

			It("the vmi with cpu.model that cannot match an nfd label on node should not be scheduled", func() {

				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should list nodes")
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some nodes")

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.ContainerDiskFor(tests.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: "Conroe",
				}

				cpuModelLabel, err := services.CPUModelLabelFromCPUModel(vmi)
				Expect(err).ToNot(HaveOccurred(), "CPU model label should have been retrieved successfully")

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, cpuModelLabel)))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				//Make sure the vmi should try to be scheduled only on master node
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node.Name}

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				By("Waiting for the VirtualMachineInstance to be unschedulable")
				Eventually(func() string {
					curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get vmi")
					if curVMI.Status.Conditions != nil {
						return curVMI.Status.Conditions[0].Reason
					}
					return ""
				}, 60*time.Second, 1*time.Second).Should(Equal("Unschedulable"), "VMI should be unchedulable")
			})

		})

		Context("with non default namespace", func() {
			table.DescribeTable("should log libvirt start and stop lifecycle events of the domain", func(namespace string) {

				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				By("Creating a VirtualMachineInstance with different namespace")
				vmi = tests.NewRandomVMIWithNS(namespace)
				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(node).Pod()
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				// Make sure we schedule the VirtualMachineInstance to master
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				// Start the VirtualMachineInstance and wait for the confirmation of the start
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)

				// Check if the start event was logged
				By("Checking that virt-handler logs VirtualMachineInstance creation")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred(), "Should get logs from virthandler")
					return string(data)
				}, 30, 0.5).Should(MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain is in state Running reason Unknown","name":"%s"`, vmi.GetObjectMeta().GetName()), "Should verify from logs that domain is running")
				// Check the VirtualMachineInstance Namespace
				Expect(vmi.GetObjectMeta().GetNamespace()).To(Equal(namespace), "VMI should run in the right namespace")

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				_, err = virtClient.RestClient().Delete().Resource("virtualmachineinstances").Namespace(vmi.GetObjectMeta().GetNamespace()).Name(vmi.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())
				stopChan := make(chan struct{})
				defer close(stopChan)
				tests.NewObjectEventWatcher(vmi).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.NormalEvent, v1.Deleted)
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				// Check if the stop event was logged
				By("Checking that virt-handler logs VirtualMachineInstance deletion")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred(), "Should get the virthandler logs")
					return string(data)
				}, 30, 0.5).Should(MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain deleted","name":"%s"`, vmi.GetObjectMeta().GetName()), "Logs should confirm pod deletion")

			},
				table.Entry(tests.NamespaceTestDefault, tests.NamespaceTestDefault),
				table.Entry(tests.NamespaceTestAlternative, tests.NamespaceTestAlternative),
			)
		})

		Context("VirtualMachineInstance Emulation Mode", func() {
			BeforeEach(func() {
				// useEmulation won't change in a test suite run, so cache it
				if useEmulation == nil {
					emulation := shouldUseEmulation(virtClient)
					useEmulation = &emulation
				}
				if !(*useEmulation) {
					Skip("Software emulation is not enabled on this cluster")
				}
			})

			It("should enable emulation in virt-launcher", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				tests.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())

				emulationFlagFound := false
				computeContainerFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "compute" {
						computeContainerFound = true
						for _, cmd := range container.Command {
							By(cmd)
							if cmd == "--use-emulation" {
								emulationFlagFound = true
							}
						}
					}
				}

				Expect(computeContainerFound).To(BeTrue(), "Compute container was not found in pod")
				Expect(emulationFlagFound).To(BeTrue(), "Expected VirtualMachineInstance pod to have '--use-emulation' flag")
			})

			It("should be reflected in domain XML", func() {
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
				Expect(err).To(BeNil(), "Should post the VMI")

				listOptions := metav1.ListOptions{}

				Eventually(func() int {
					podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
					Expect(err).ToNot(HaveOccurred(), "Should list the pods")
					return len(podList.Items)
				}, 75, 0.5).Should(Equal(1), "There should be only one pod")

				Eventually(func() error {
					podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
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

				newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")

				domain := &api.Domain{}
				context := &api.ConverterContext{
					VirtualMachine: newVMI,
					UseEmulation:   true,
				}
				api.Convert_v1_VirtualMachine_To_api_Domain(newVMI, domain, context)

				expectedType := ""
				if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
					expectedType = "qemu"
				}

				Expect(domain.Spec.Type).To(Equal(expectedType), "VMI domain type should be of expectedType")
			})

			It("should request a TUN device but not KVM", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				tests.WaitForSuccessfulVMIStart(vmi)

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
				// useEmulation won't change in a test suite run, so cache it
				if useEmulation == nil {
					emulation := shouldUseEmulation(virtClient)
					useEmulation = &emulation
				}
				if *useEmulation {
					Skip("Software emulation is enabled on this cluster")
				}
			})

			It("should request a KVM and TUN device", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				tests.WaitForSuccessfulVMIStart(vmi)

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

			It("should not enable emulation in virt-launcher", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())

				tests.WaitForSuccessfulVMIStart(vmi)

				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())

				emulationFlagFound := false
				computeContainerFound := false
				for _, container := range pod.Spec.Containers {
					if container.Name == "compute" {
						computeContainerFound = true
						for _, cmd := range container.Command {
							By(cmd)
							if cmd == "--use-emulation" {
								emulationFlagFound = true
							}
						}
					}
				}

				Expect(computeContainerFound).To(BeTrue(), "Compute container was not found in pod")
				Expect(emulationFlagFound).To(BeFalse(), "Expected VM pod not to have '--use-emulation' flag")
			})

			It("Should provide KVM via plugin framework", func() {
				nodeList := tests.GetAllSchedulableNodes(virtClient)

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

	Describe("Get a VirtualMachineInstance", func() {
		Context("when that not exist", func() {
			It("should return 404", func() {
				b, err := virtClient.RestClient().Get().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Name("nonexistnt").DoRaw()
				Expect(err).ToNot(BeNil(), "Should get VMIs")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil(), "Unmarshal without error")
				Expect(status.Code).To(Equal(int32(http.StatusNotFound)), "There should not be and VMI")
			})
		})
	})

	Describe("Delete a VirtualMachineInstance's Pod", func() {
		It("should result in the VirtualMachineInstance moving to a finalized state", func() {
			By("Creating the VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")
			tests.WaitForSuccessfulVMIStart(obj)

			By("Verifying VirtualMachineInstance's pod is active")
			pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(vmi))
			Expect(err).ToNot(HaveOccurred(), "Should list pods")
			Expect(len(pods.Items)).To(Equal(1), "There should be only one pod")
			pod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			}, 10*time.Second, 1*time.Second).Should(Succeed(), "Should delete VMI pod")

			// Wait for VirtualMachineInstance to finalize
			By("Waiting for the VirtualMachineInstance to move to a finalized state")
			Eventually(func() error {
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				if err != nil {
					return err
				} else if !curVMI.IsFinal() {
					return fmt.Errorf("VirtualMachineInstance has not reached a finalized state yet")
				}
				return nil
			}, 60*time.Second, 1*time.Second).Should(Succeed(), "VMI reached finalized state")
		})
	})
	Describe("Delete a VirtualMachineInstance", func() {
		Context("with an active pod.", func() {
			It("should result in pod being terminated", func() {

				By("Creating the VirtualMachineInstance")
				obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(obj)

				podSelector := tests.UnfinishedVMIPodSelector(vmi)
				By("Verifying VirtualMachineInstance's pod is active")
				pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(podSelector)
				Expect(err).ToNot(HaveOccurred(), "Should list pods")
				Expect(len(pods.Items)).To(Equal(1), "There should be only one pod")

				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's pod terminates")
				Eventually(func() int {
					pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(podSelector)
					Expect(err).ToNot(HaveOccurred(), "Should list pods")
					return len(pods.Items)
				}, 75, 0.5).Should(Equal(0), "There should be no pods")

			})
		})
		Context("with ACPI and 0 grace period seconds", func() {
			It("should result in vmi status failed", func() {

				vmi = newCirrosVMI()
				gracePeriod := int64(0)
				vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod

				By("Creating the VirtualMachineInstance")
				obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				// wait until booted
				vmi = tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's status is Failed")
				Eventually(func() v1.VirtualMachineInstancePhase {
					currVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					return currVMI.Status.Phase
				}, 5, 0.5).Should(Equal(v1.Failed), "VMI should be failed")
			})
		})
		Context("with ACPI and some grace period seconds", func() {
			table.DescribeTable("should result in vmi status succeeded", func(gracePeriod int64) {
				vmi = newCirrosVMI()

				if gracePeriod >= 0 {
					vmi.Spec.TerminationGracePeriodSeconds = &gracePeriod
				} else {
					gracePeriod = v1.DefaultGracePeriodSeconds
					vmi.Spec.TerminationGracePeriodSeconds = nil
				}

				By("Creating the VirtualMachineInstance")
				obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				// wait until booted
				vmi = tests.WaitUntilVMIReady(vmi, tests.LoggedInCirrosExpecter)

				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI")

				By("Verifying VirtualMachineInstance's status is Succeeded")
				Eventually(func() v1.VirtualMachineInstancePhase {
					currVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI")
					return currVMI.Status.Phase
				}, gracePeriod+5, 0.5).Should(Equal(v1.Succeeded), "VMI should be succeeded")
			},
				table.Entry("with set grace period seconds", int64(10)),
				table.Entry("with default grace period seconds", int64(-1)),
			)
		})
		Context("with grace period greater than 0", func() {
			It("should run graceful shutdown", func() {
				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(node).Pod()
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
				obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(obj)

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed(), "Should delete VMI gracefully")
				stopChan := make(chan struct{})
				defer close(stopChan)
				event := tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(75*time.Second).WaitFor(stopChan, tests.NormalEvent, v1.Deleted)
				Expect(event).ToNot(BeNil(), "There should be a delete event")

				// Check if the graceful shutdown was logged
				By("Checking that virt-handler logs VirtualMachineInstance graceful shutdown")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred(), "Should get the logs")
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())), "Should log gracefull shutdown")

				// Verify VirtualMachineInstance is killed after grace period expires
				By("Checking that the VirtualMachineInstance does not exist after grace period")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred(), "Should get logs")
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())), "Should log gracefull kill")
			})
		})
	})

	Describe("Killed VirtualMachineInstance", func() {
		It("should be in Failed phase", func() {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Should create VMI")

			nodeName := tests.WaitForSuccessfulVMIStart(obj)

			By("Killing the VirtualMachineInstance")
			time.Sleep(10 * time.Second)
			err = pkillAllVMIs(virtClient, nodeName)
			Expect(err).To(BeNil(), "Should deploy helper pod to kill VMI")

			stopChan := make(chan struct{})
			defer close(stopChan)
			tests.NewObjectEventWatcher(obj).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.WarningEvent, v1.Stopped)

			By("Checking that the VirtualMachineInstance has 'Failed' phase")
			Eventually(func() v1.VirtualMachineInstancePhase {
				failedVMI, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				return failedVMI.Status.Phase
			}, 10, 1).Should(Equal(v1.Failed), "VMI should be failed")

		})

		It("should be left alone by virt-handler", func() {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Should create VMI")

			nodeName := tests.WaitForSuccessfulVMIStart(obj)

			By("Killing the VirtualMachineInstance")
			err = pkillAllVMIs(virtClient, nodeName)
			Expect(err).To(BeNil(), "Should create kill pod to kill all VMs")

			// Wait for stop event of the VirtualMachineInstance
			stopChan := make(chan struct{})
			defer close(stopChan)
			tests.NewObjectEventWatcher(obj).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.WarningEvent, v1.Stopped)

			// Wait for some time and see if a sync event happens on the stopped VirtualMachineInstance
			By("Checking that virt-handler does not try to sync stopped VirtualMachineInstance")
			event := tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(10*time.Second).WaitNotFor(stopChan, tests.WarningEvent, v1.SyncFailed)
			Expect(event).To(BeNil(), "virt-handler tried to sync on a VirtualMachineInstance in final state")
		})
	})
})

func shouldUseEmulation(virtClient kubecli.KubevirtClient) bool {
	useEmulation := false
	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps(tests.KubeVirtInstallNamespace).Get("kubevirt-config", options)
	if err == nil {
		val, ok := cfgMap.Data["debug.useEmulation"]
		useEmulation = ok && (val == "true")
	} else {
		// If the cfgMap is missing, default to useEmulation=false
		// no other error is expected
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	return useEmulation
}

func renderPkillAllJob(processName string) *k8sv1.Pod {
	return tests.RenderJob("vmi-killer", []string{"pkill"}, []string{"-9", processName})
}

func getVirtLauncherLogs(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()

	labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

	pods, err := virtCli.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred(), "Should list pods")

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty(), "Should find pod not scheduled for deletion")

	var tailLines int64 = 100
	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			TailLines: &tailLines,
			Container: "compute",
		}).
		DoRaw()
	Expect(err).To(BeNil(), "Should get virt-launcher pod logs")

	return string(logsRaw)
}

func pkillHandler(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("virt-handler")
	job.Spec.NodeName = node
	pod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
	Expect(err).ToNot(HaveOccurred(), "Should create helper pod")

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Get(pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "Should return current status")
		return pod.Status.Phase
	}

	Eventually(getStatus, 30, 0.5).Should(Equal(k8sv1.PodSucceeded), "Pod should end itself")

	return err
}

func pkillAllLaunchers(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("virt-launcher")
	job.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)

	return err
}

func pkillAllVMIs(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("qemu")
	job.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)

	return err
}

func nowAsJSONWithOffset(offset time.Duration) string {
	now := metav1.Now()
	now = metav1.NewTime(now.Add(offset))

	data, err := json.Marshal(now)
	Expect(err).ToNot(HaveOccurred(), "Should marshal to json")
	return strings.Trim(string(data), `"`)
}
