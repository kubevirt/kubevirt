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

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("VMIlifecycle", func() {

	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(tests.RegistryDiskFor(tests.RegistryDiskAlpine))
	})

	AfterEach(func() {
		// Not every test causes virt-handler to restart, but a few different contexts do.
		// This check is fast and non-intrusive if virt-handler is already running.
	})

	Describe("Creating a VirtualMachineInstance", func() {
		It("should success", func() {
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil())
		})

		It("should start it", func() {
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMIStart(vmi)
		})

		It("should attach virt-launcher to it", func() {
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil())
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("Found PID for"))
		})

		It("should reject POST if schema is invalid", func() {
			jsonBytes, err := json.Marshal(vmi)
			Expect(err).To(BeNil())

			// change the name of a required field (like domain) so validation will fail
			jsonString := strings.Replace(string(jsonBytes), "domain", "not-a-domain", -1)

			result := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body([]byte(jsonString)).SetHeader("Content-Type", "application/json").Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))
		})
		It("should reject POST if validation webhook deems the spec invalid", func() {

			// Add a disk that doesn't map to a volume.
			// This should get rejected which tells us the webhook validator is working.
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk",
				VolumeName: "testvolume",
			})
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name:       "testdisk2",
				VolumeName: "testvolume2",
			})

			result := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))

			reviewResponse := &metav1.Status{}
			body, _ := result.Raw()
			err = json.Unmarshal(body, reviewResponse)
			Expect(err).To(BeNil())

			Expect(len(reviewResponse.Details.Causes)).To(Equal(2))
			Expect(reviewResponse.Details.Causes[0].Field).To(Equal("spec.domain.devices.disks[1].volumeName"))
			Expect(reviewResponse.Details.Causes[1].Field).To(Equal("spec.domain.devices.disks[2].volumeName"))
		})

		It("should reject PATCH if schema is invalid", func() {
			err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
			Expect(err).To(BeNil())

			// Add a disk without a volume reference (this is in valid)
			patchStr := "{\"apiVersion\":\"kubevirt.io/v1alpha2\",\"kind\":\"VirtualMachineInstance\",\"spec\":{\"domain\":{\"devices\":{\"disks\":[{\"disk\":{\"bus\":\"virtio\"},\"name\":\"fakedisk\"}]}}}}"

			result := virtClient.RestClient().Patch(types.MergePatchType).Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Name(vmi.Name).Body([]byte(patchStr)).Do()

			// Verify validation failed.
			statusCode := 0
			result.StatusCode(&statusCode)
			Expect(statusCode).To(Equal(http.StatusUnprocessableEntity))
		})

		Context("when it already exist", func() {
			It("should be rejected", func() {
				By("Creating a VirtualMachineInstance")
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
				Expect(err).To(BeNil())
				By("Creating the same VirtualMachineInstance second time")
				b, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).DoRaw()
				Expect(err).ToNot(BeNil())
				By("Checking that POST return status equals to 409")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil())
				Expect(status.Code).To(Equal(int32(http.StatusConflict)))
			})
		})

		Context("with boot order", func() {
			table.DescribeTable("should be able to boot from selected disk", func(alpineBootOrder uint, cirrosBootOrder uint, consoleText string, wait int) {
				By("defining a VirtualMachineInstance with an Alpine disk")
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(tests.RegistryDiskFor(tests.RegistryDiskAlpine), "#!/bin/sh\n\necho 'hi'\n")
				By("adding a Cirros Disk")
				tests.AddEphemeralDisk(vmi, "disk2", "virtio", tests.RegistryDiskFor(tests.RegistryDiskCirros))

				By("setting boot order")
				vmi = tests.AddBootOrderToDisk(vmi, "disk0", &alpineBootOrder)
				vmi = tests.AddBootOrderToDisk(vmi, "disk2", &cirrosBootOrder)

				By("starting VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil())

				By("Waiting the VirtualMachineInstance start")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Checking console text")
				err = tests.CheckForTextExpecter(vmi, consoleText, wait)
				Expect(err).ToNot(HaveOccurred())
			},
				table.Entry("Alpine as first boot", uint(1), uint(2), "Welcome to Alpine", 90),
				table.Entry("Cirros as first boot", uint(2), uint(1), "cirros", 90),
			)
		})

		Context("with user-data", func() {
			Context("without k8s secret", func() {
				It("should retry starting the VirtualMachineInstance", func() {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), userData)

					for _, volume := range vmi.Spec.Volumes {
						if volume.CloudInitNoCloud != nil {
							spec := volume.CloudInitNoCloud
							spec.UserDataBase64 = ""
							spec.UserDataSecretRef = &k8sv1.LocalObjectReference{Name: "nonexistent"}
							break
						}
					}
					By("Starting a VirtualMachineInstance")
					obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
					Expect(err).To(BeNil())

					By("Checking that VirtualMachineInstance was restarted twice")
					retryCount := 0
					tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(60 * time.Second).Watch(func(event *k8sv1.Event) bool {
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
					vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), userData)

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
					Expect(err).To(BeNil())

					// Wait until we see that starting the VirtualMachineInstance is failing
					By("Checking that VirtualMachineInstance start failed")
					event := tests.NewObjectEventWatcher(createdVMI).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.SyncFailed)
					Expect(event.Message).To(ContainSubstring("nonexistent"))

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
					Expect(err).ToNot(HaveOccurred())

					// Wait for the VirtualMachineInstance to be started, allow warning events to occur
					By("Checking that VirtualMachineInstance start succeeded")
					tests.NewObjectEventWatcher(createdVMI).SinceWatchedObjectResourceVersion().Timeout(30*time.Second).WaitFor(tests.NormalEvent, v1.Started)
				})
			})
		})

		Context("when virt-launcher crashes", func() {
			It("should be stopped and have Failed phase", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil())

				nodeName := tests.WaitForSuccessfulVMIStart(vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Crashing the virt-launcher")
				err = pkillAllLaunchers(virtClient, nodeName)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(tests.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Expect(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase
				}()).To(Equal(v1.Failed))
			})
		})

		Context("when virt-handler crashes", func() {
			It("should recover and continue management", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil())

				// Start a VirtualMachineInstance
				nodeName := tests.WaitForSuccessfulVMIStart(vmi)
				Expect(err).ToNot(HaveOccurred())

				// Kill virt-handler on the node the VirtualMachineInstance is active on.
				By("Crashing the virt-handler")
				err = pkillHandler(virtClient, nodeName)
				Expect(err).To(BeNil())

				// Crash the VirtualMachineInstance and verify a recovered version of virt-handler processes the crash
				By("Killing the VirtualMachineInstance")
				err = pkillAllVMIs(virtClient, nodeName)
				Expect(err).To(BeNil())

				tests.NewObjectEventWatcher(vmi).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Expect(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmi.Status.Phase
				}()).To(Equal(v1.Failed))
			})
		})

		Context("when virt-handler is responsive", func() {
			It("should indicate that a node is ready for vmis", func() {

				By("adding a heartbeat annotation and a schedulable label to the node")
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{LabelSelector: v1.NodeSchedulable + "=" + "true"})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				for _, node := range nodes.Items {
					Expect(node.Annotations[v1.VirtHandlerHeartbeat]).ToNot(HaveLen(0))
				}

				node := &nodes.Items[0]
				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType, []byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "false"}}}`, v1.NodeSchedulable)))
				Expect(err).ToNot(HaveOccurred())
				timestamp := node.Annotations[v1.VirtHandlerHeartbeat]

				By("setting the schedulable label back to true")
				Eventually(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return n.Labels[v1.NodeSchedulable]
				}, 2*time.Minute, 2*time.Second).Should(Equal("true"))
				By("updating the heartbeat roughly every minute")
				Expect(func() string {
					n, err := virtClient.CoreV1().Nodes().Get(node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return n.Labels[v1.VirtHandlerHeartbeat]
				}()).ShouldNot(Equal(timestamp))
			})
		})

		Context("when virt-handler is not responsive", func() {

			var vmi *v1.VirtualMachineInstance
			var nodeName string
			var virtHandler *k8sv1.Pod
			var virtHandlerAvailablePods int32

			BeforeEach(func() {
				// Schedule a vmi and make sure that virt-handler gets evicted from the node where the vmi was started
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(tests.RegistryDiskFor(tests.RegistryDiskCirros), "echo hi!")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				nodeName = tests.WaitForSuccessfulVMIStart(vmi)
				virtHandler, err = kubecli.NewVirtHandlerClient(virtClient).ForNode(nodeName).Pod()
				Expect(err).ToNot(HaveOccurred())
				ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get("virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
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
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool {
					_, err := virtClient.CoreV1().Pods(virtHandler.Namespace).Get(virtHandler.Name, metav1.GetOptions{})
					return errors.IsNotFound(err)
				}, 90*time.Second, 1*time.Second).Should(BeTrue())
			})

			It("the node controller should react", func() {

				// Update virt-handler heartbeat, to trigger a timeout
				data := []byte(fmt.Sprintf(`{"metadata": { "annotations": {"%s": "%s"}}}`, v1.VirtHandlerHeartbeat, nowAsJSONWithOffset(-10*time.Minute)))
				_, err = virtClient.CoreV1().Nodes().Patch(nodeName, types.StrategicMergePatchType, data)
				Expect(err).ToNot(HaveOccurred())

				// Delete vmi pod
				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(pods.Items).To(HaveLen(1))
				Expect(virtClient.CoreV1().Pods(vmi.Namespace).Delete(pods.Items[0].Name, &metav1.DeleteOptions{})).To(Succeed())

				// it will take at least 45 seconds until the vmi is gone, check the schedulable state in the meantime
				By("marking the node as not schedulable")
				Eventually(func() string {
					node, err := virtClient.CoreV1().Nodes().Get(nodeName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return node.Labels[v1.NodeSchedulable]
				}, 20*time.Second, 1*time.Second).Should(Equal("false"))

				By("moving stuck vmis to failed state")
				Eventually(func() v1.VirtualMachineInstancePhase {
					failedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return failedVMI.Status.Phase
				}, 180*time.Second, 1*time.Second).Should(Equal(v1.Failed))
			})

			AfterEach(func() {
				// Restore virt-handler daemonset
				ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get("virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				ds.Spec.Template.Spec.Affinity = nil
				_, err = virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Update(ds)
				Expect(err).ToNot(HaveOccurred())

				// Wait until virt-handler ds will have expected number of pods
				Eventually(func() bool {
					ds, err := virtClient.AppsV1().DaemonSets(virtHandler.Namespace).Get("virt-handler", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return ds.Status.NumberAvailable == virtHandlerAvailablePods &&
						ds.Status.CurrentNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.DesiredNumberScheduled == virtHandlerAvailablePods &&
						ds.Status.NumberReady == virtHandlerAvailablePods &&
						ds.Status.UpdatedNumberScheduled == virtHandlerAvailablePods
				}, 60*time.Second, 1*time.Second).Should(Equal(true))
			})
		})

		Context("with non default namespace", func() {
			table.DescribeTable("should log libvirt start and stop lifecycle events of the domain", func(namespace string) {

				_, exists := os.LookupEnv("JENKINS_HOME")
				if exists {
					Skip("Skip log query tests for JENKINS ci test environment")
				}
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				node := nodes.Items[0].Name

				By("Creating a VirtualMachineInstance with different namespace")
				vmi = tests.NewRandomVMIWithNS(namespace)
				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(node).Pod()
				Expect(err).ToNot(HaveOccurred())

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(120)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})

				// Make sure we schedule the VirtualMachineInstance to master
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node}

				// Start the VirtualMachineInstance and wait for the confirmation of the start
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)

				// Check if the start event was logged
				By("Checking that virt-handler logs VirtualMachineInstance creation")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain is in state Running)", vmi.GetObjectMeta().GetName()))
				// Check the VirtualMachineInstance Namespace
				Expect(vmi.GetObjectMeta().GetNamespace()).To(Equal(namespace))

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				_, err = virtClient.RestClient().Delete().Resource("virtualmachineinstances").Namespace(vmi.GetObjectMeta().GetNamespace()).Name(vmi.GetObjectMeta().GetName()).Do().Get()
				Expect(err).To(BeNil())
				tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)
				tests.WaitForVirtualMachineToDisappearWithTimeout(vmi, 120)

				// Check if the stop event was logged
				By("Checking that virt-handler logs VirtualMachineInstance deletion")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(MatchRegexp("(name=%s)[^\n]+(kind=Domain)[^\n]+(Domain deleted)", vmi.GetObjectMeta().GetName()))

			},
				table.Entry(tests.NamespaceTestDefault, tests.NamespaceTestDefault),
				table.Entry(tests.NamespaceTestAlternative, tests.NamespaceTestAlternative),
			)
		})

		Context("VirtualMachineInstance Emulation Mode", func() {
			BeforeEach(func() {
				allowEmuation := false
				options := metav1.GetOptions{}
				cfgMap, err := virtClient.CoreV1().ConfigMaps("kube-system").Get("kubevirt-config", options)
				if err == nil {
					val, ok := cfgMap.Data["debug.allowEmulation"]
					allowEmuation = ok && (val == "true")
				} else {
					// If the cfgMap is missing, default to allowEmulation=false
					// no other error is expected
					if !errors.IsNotFound(err) {
						Expect(err).ToNot(HaveOccurred())
					}
				}
				if !allowEmuation {
					Skip("Software emulation is not enabled on this cluster")
				}
			})

			It("should enable emulation in virt-launcher", func() {
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
				Expect(err).To(BeNil())

				listOptions := metav1.ListOptions{}
				var pod k8sv1.Pod

				Eventually(func() error {
					podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
					Expect(err).ToNot(HaveOccurred())
					for _, item := range podList.Items {
						if strings.HasPrefix(item.Name, vmi.ObjectMeta.GenerateName) {
							pod = item
							return nil
						}
					}
					return fmt.Errorf("Associated pod for VirtualMachineInstance '%s' not found", vmi.Name)
				}, 75, 0.5).Should(Succeed())

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

			It("should be reflected in domain XML", func() {
				err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Error()
				Expect(err).To(BeNil())

				listOptions := metav1.ListOptions{}

				Eventually(func() int {
					podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
					Expect(err).ToNot(HaveOccurred())
					return len(podList.Items)
				}, 75, 0.5).Should(Equal(1))

				Eventually(func() error {
					podList, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(listOptions)
					Expect(err).ToNot(HaveOccurred())
					for _, item := range podList.Items {
						if strings.HasPrefix(item.Name, vmi.ObjectMeta.GenerateName) {
							return nil
						}
					}
					return fmt.Errorf("Associated pod for VirtualMachineInstance '%s' not found", vmi.Name)
				}, 75, 0.5).Should(Succeed())

				getOptions := metav1.GetOptions{}
				var newVMI *v1.VirtualMachineInstance

				newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Get(vmi.Name, &getOptions)
				Expect(err).ToNot(HaveOccurred())

				domain := &api.Domain{}
				context := &api.ConverterContext{
					AllowEmulation: true,
					VirtualMachine: newVMI,
				}
				api.Convert_v1_VirtualMachine_To_api_Domain(newVMI, domain, context)

				expectedType := ""
				if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
					expectedType = "qemu"
				}

				Expect(domain.Spec.Type).To(Equal(expectedType))
			})
		})
	})

	Describe("Get a VirtualMachineInstance", func() {
		Context("when that not exist", func() {
			It("should return 404", func() {
				b, err := virtClient.RestClient().Get().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Name("nonexistnt").DoRaw()
				Expect(err).ToNot(BeNil())
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil())
				Expect(status.Code).To(Equal(int32(http.StatusNotFound)))
			})
		})
	})

	Describe("Delete a VirtualMachineInstance's Pod", func() {
		It("should result in the VirtualMachineInstance moving to a finalized state", func() {
			By("Creating the VirtualMachineInstance")
			obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(obj)

			By("Verifying VirtualMachineInstance's pod is active")
			pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(vmi))
			Expect(err).ToNot(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			pod := pods.Items[0]

			// Delete the Pod
			By("Deleting the VirtualMachineInstance's pod")
			Eventually(func() error {
				return virtClient.CoreV1().Pods(pod.Namespace).Delete(pod.Name, &metav1.DeleteOptions{})
			}, 10*time.Second, 1*time.Second).Should(Succeed())

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
			}, 60*time.Second, 1*time.Second).Should(Succeed())
		})
	})
	Describe("Delete a VirtualMachineInstance", func() {
		Context("with an active pod.", func() {
			It("should result in pod being terminated", func() {

				By("Creating the VirtualMachineInstance")
				obj, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(obj)

				By("Verifying VirtualMachineInstance's pod is active")
				pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(vmi))
				Expect(err).ToNot(HaveOccurred())
				Expect(len(pods.Items)).To(Equal(1))

				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed())

				By("Verifying VirtualMachineInstance's pod terminates")
				Eventually(func() int {
					pods, err := virtClient.CoreV1().Pods(tests.NamespaceTestDefault).List(tests.UnfinishedVMIPodSelector(vmi))
					Expect(err).ToNot(HaveOccurred())
					return len(pods.Items)
				}, 75, 0.5).Should(Equal(0))
			})
		})
		Context("with grace period greater than 0", func() {
			It("should run graceful shutdown", func() {
				nodes, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodes.Items).ToNot(BeEmpty())
				node := nodes.Items[0].Name

				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).ForNode(node).Pod()
				Expect(err).ToNot(HaveOccurred())

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
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(obj)

				// Delete the VirtualMachineInstance and wait for the confirmation of the delete
				By("Deleting the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Delete(obj.Name, &metav1.DeleteOptions{})).To(Succeed())
				tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().WaitFor(tests.NormalEvent, v1.Deleted)

				// Check if the graceful shutdown was logged
				By("Checking that virt-handler logs VirtualMachineInstance graceful shutdown")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("Signaled graceful shutdown for %s", vmi.GetObjectMeta().GetName())))

				// Verify VirtualMachineInstance is killed after grace period expires
				By("Checking that the VirtualMachineInstance does not exist after grace period")
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred())
					return string(data)
				}, 30, 0.5).Should(ContainSubstring(fmt.Sprintf("grace period expired, killing deleted VirtualMachineInstance %s", vmi.GetObjectMeta().GetName())))
			})
		})
	})

	Describe("Killed VirtualMachineInstance", func() {
		It("should be in Failed phase", func() {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
			Expect(err).To(BeNil())

			nodeName := tests.WaitForSuccessfulVMIStart(obj)
			_, ok := obj.(*v1.VirtualMachineInstance)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
			Expect(err).ToNot(HaveOccurred())

			By("Killing the VirtualMachineInstance")
			time.Sleep(10 * time.Second)
			err = pkillAllVMIs(virtClient, nodeName)
			Expect(err).To(BeNil())

			tests.NewObjectEventWatcher(obj).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

			By("Checking that the VirtualMachineInstance has 'Failed' phase")
			Expect(func() v1.VirtualMachineInstancePhase {
				vmi := &v1.VirtualMachineInstance{}
				err := virtClient.RestClient().Get().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Name(obj.(*v1.VirtualMachineInstance).ObjectMeta.Name).Do().Into(vmi)
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase
			}()).To(Equal(v1.Failed))

		})

		It("should be left alone by virt-handler", func() {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Body(vmi).Do().Get()
			Expect(err).To(BeNil())

			nodeName := tests.WaitForSuccessfulVMIStart(obj)
			_, ok := obj.(*v1.VirtualMachineInstance)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
			Expect(err).ToNot(HaveOccurred())

			By("Killing the VirtualMachineInstance")
			err = pkillAllVMIs(virtClient, nodeName)
			Expect(err).To(BeNil())

			// Wait for stop event of the VirtualMachineInstance
			tests.NewObjectEventWatcher(obj).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.Stopped)

			// Wait for some time and see if a sync event happens on the stopped VirtualMachineInstance
			By("Checking that virt-handler does not try to sync stopped VirtualMachineInstance")
			event := tests.NewObjectEventWatcher(obj).SinceWatchedObjectResourceVersion().Timeout(5*time.Second).
				SinceWatchedObjectResourceVersion().WaitFor(tests.WarningEvent, v1.SyncFailed)
			Expect(event).To(BeNil(), "virt-handler tried to sync on a VirtualMachineInstance in final state")
		})
	})
})

func renderPkillAllJob(processName string) *k8sv1.Pod {
	return tests.RenderJob("vmi-killer", []string{"pkill"}, []string{"-9", processName})
}

func getVirtLauncherLogs(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	uid := vmi.GetObjectMeta().GetUID()

	labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))

	pods, err := virtCli.CoreV1().Pods(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())

	podName := ""
	for _, pod := range pods.Items {
		if pod.ObjectMeta.DeletionTimestamp == nil {
			podName = pod.ObjectMeta.Name
			break
		}
	}
	Expect(podName).ToNot(BeEmpty())

	var tailLines int64 = 100
	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			TailLines: &tailLines,
			Container: "compute",
		}).
		DoRaw()
	Expect(err).To(BeNil())

	return string(logsRaw)
}

func pkillHandler(virtCli kubecli.KubevirtClient, node string) error {
	job := renderPkillAllJob("virt-handler")
	job.Spec.NodeName = node
	pod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(job)
	Expect(err).ToNot(HaveOccurred())

	getStatus := func() k8sv1.PodPhase {
		pod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Get(pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}

	Eventually(getStatus, 30, 0.5).Should(Equal(k8sv1.PodSucceeded))

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
	Expect(err).ToNot(HaveOccurred())
	return strings.Trim(string(data), `"`)
}
