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
	"fmt"
	"net/http"
	"os"
	"strconv"
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

	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
)

const kubevirtConfig = "kubevirt-config"

func newCirrosVMI() *v1.VirtualMachineInstance {
	return tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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

var _ = Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]VMIlifecycle", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var vmi *v1.VirtualMachineInstance

	var useEmulation *bool

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
	})

	AfterEach(func() {
		// Not every test causes virt-handler to restart, but a few different contexts do.
		// This check is fast and non-intrusive if virt-handler is already running.
		tests.EnsureKVMPresent()
	})

	Context("when virt-handler is deleted", func() {
		It("[Serial][test_id:4716]should label the node with kubevirt.io/schedulable=false", func() {
			pods, err := virtClient.CoreV1().Pods("").List(metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(pods.Items).ToNot(BeEmpty())

			pod := pods.Items[0]
			handlerNamespace := pod.GetNamespace()
			err = virtClient.CoreV1().Pods(handlerNamespace).Delete(pod.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() string {
				n, err := virtClient.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return n.Labels[v1.NodeSchedulable]
			}, 20*time.Second, 1*time.Second).Should(Equal("false"))

		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Creating a VirtualMachineInstance", func() {
		It("[test_id:1619]should success", func() {
			_, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
		})

		It("[test_id:1620]should start it", func() {
			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)
		})

		It("[test_id:1621]should attach virt-launcher to it", func() {
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

		It("[test_id:3195]should carry annotations to pod", func() {
			vmi.Annotations = map[string]string{
				"testannotation": "annotation from vmi",
			}

			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)

			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(pod).NotTo(BeNil())

			Expect(pod.Annotations).To(HaveKeyWithValue("testannotation", "annotation from vmi"), "annotation should be carried to the pod")
		})

		It("[test_id:3196]should carry kubernetes and kubevirt annotations to pod", func() {
			vmi.Annotations = map[string]string{
				"kubevirt.io/test":   "test",
				"kubernetes.io/test": "test",
			}

			vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)

			pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(pod).NotTo(BeNil())

			Expect(pod.Annotations).To(HaveKey("kubevirt.io/test"), "kubevirt annotation should not be carried to the pod")
			Expect(pod.Annotations).To(HaveKey("kubernetes.io/test"), "kubernetes annotation should not be carried to the pod")
		})

		It("[test_id:1622]should log libvirtd logs", func() {
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
				Should(And(ContainSubstring("At least one cgroup controller is required: No such device or address"), ContainSubstring(`"subcomponent":"libvirt"`)))
		})

		It("[test_id:3197]should log libvirtd debug logs when enabled", func() {
			var err error
			vmi := tests.NewRandomVMI()
			vmi.Labels = map[string]string{
				"debugLogs": "true",
			}

			vmi, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
			Expect(err).To(BeNil(), "Create VMI successfully")
			tests.WaitForSuccessfulVMIStart(vmi)

			By("Getting virt-launcher logs")
			logs := func() string { return getVirtLauncherLogs(virtClient, vmi) }

			// there are plenty of strings we can use to identify the debug logs. Here we use something easy to see...
			Eventually(logs,
				11*time.Second,
				500*time.Millisecond).
				Should(ContainSubstring("OBJECT_REF"))
			// ...and something we deeply care about when in debug mode.
			Eventually(logs,
				2*time.Second,
				500*time.Millisecond).
				Should(And(ContainSubstring("QEMU_MONITOR_SEND_MSG"), ContainSubstring(`"subcomponent":"libvirt"`)))
		})

		It("[test_id:1623]should reject POST if validation webhook deems the spec invalid", func() {

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

		It("[test_id:1624]should reject PATCH if schema is invalid", func() {
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
				vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Name = "testvmi" + rand.String(63)
			})
			It("[test_id:1625]should start it", func() {
				By("Creating a VirtualMachineInstance with a long name")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "cannot create VirtualMachineInstance %q: %v", vmi.Name, err)
				Expect(len(vmi.Name)).To(BeNumerically(">", 63), "VirtualMachineInstance %q name is not longer than 63 characters", vmi.Name)

				By("Waiting until it starts")
				tests.WaitForSuccessfulVMIStart(vmi)
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "cannot fetch VirtualMachineInstance %q: %v", vmi.Name, err)

				By("Obtaining serial console")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred(), "VirtualMachineInstance %q console is not accessible: %v", vmi.Name, err)
				expecter.Close()
			})
		})

		Context("when it already exist", func() {
			It("[test_id:1626]should be rejected", func() {
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
			table.DescribeTable("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]should be able to boot from selected disk", func(alpineBootOrder uint, cirrosBootOrder uint, consoleText string, wait int) {
				By("defining a VirtualMachineInstance with an Alpine disk")
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(cd.ContainerDiskFor(cd.ContainerDiskAlpine), "#!/bin/sh\n\necho 'hi'\n")
				By("adding a Cirros Disk")
				tests.AddEphemeralDisk(vmi, "disk2", "virtio", cd.ContainerDiskFor(cd.ContainerDiskCirros))

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
					&expect.BExp{R: consoleText},
				}, wait)
				Expect(err).ToNot(HaveOccurred(), "Should match the console in VMI")
			},
				table.Entry("[test_id:1627]Alpine as first boot", uint(1), uint(2), "Welcome to Alpine", 90),
				table.Entry("[test_id:1628]Cirros as first boot", uint(2), uint(1), "cirros", 90),
			)
		})

		Context("with user-data", func() {

			Context("without k8s secret", func() {
				It("[test_id:1629][posneg:negative]should not be able to start virt-launcher pod", func() {
					userData := fmt.Sprintf("#!/bin/sh\n\necho 'hi'\n")
					vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

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
					stopChan := make(chan struct{})
					defer close(stopChan)
					launcher := tests.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
					tests.NewObjectEventWatcher(launcher).
						SinceWatchedObjectResourceVersion().
						Timeout(60*time.Second).
						Watch(stopChan, func(event *k8sv1.Event) bool {
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
					launcher := tests.GetPodByVirtualMachineInstance(createdVMI, createdVMI.Namespace)
					// Wait until we see that starting the VirtualMachineInstance is failing
					By("Checking that VirtualMachineInstance start failed")
					stopChan := make(chan struct{})
					defer close(stopChan)
					event := tests.NewObjectEventWatcher(launcher).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.WarningEvent, "FailedMount")
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
					tests.NewObjectEventWatcher(createdVMI).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(stopChan, tests.NormalEvent, v1.Started)
				})
			})
		})

		Context("when virt-launcher crashes", func() {
			It("[Serial][test_id:1631]should be stopped and have Failed phase", func() {
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Should create VMI successfully")

				nodeName := tests.WaitForSuccessfulVMIStart(vmi)

				stopChan := make(chan struct{})
				defer close(stopChan)

				By("Crashing the virt-launcher")
				vmiKiller, err := pkillAllLaunchers(virtClient, nodeName)
				Expect(err).To(BeNil(), "Should create vmi-killer pod to kill virt-launcher successfully")
				tests.NewObjectEventWatcher(vmiKiller).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(stopChan, tests.NormalEvent, v1.Started)

				By("Waiting for the vm to be stopped")
				tests.NewObjectEventWatcher(vmi).SinceWatchedObjectResourceVersion().Timeout(60*time.Second).WaitFor(stopChan, tests.WarningEvent, v1.Stopped)

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}, 10, 1).Should(Equal(v1.Failed), "VMI should be failed")
			})
		})

		Context("[Serial]when virt-handler crashes", func() {
			// FIXME: This test has the issues that it tests a lot of different timing scenarios in an intransparent way:
			// e.g. virt-handler can die before or after virt-launcher. If we wait until virt-handler is dead before we
			// kill virt-launcher then we don't know if virt-handler already restarted.
			// Also the virt-handler crash-loop plays a role here. We could also change the daemon-set but then we would not check the crash behaviour.
			It("[test_id:1632]should recover and continue management", func() {

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

				By("Checking that VirtualMachineInstance has 'Failed' phase")
				Eventually(func() v1.VirtualMachineInstancePhase {
					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should get VMI successfully")
					return vmi.Status.Phase
				}, 240*time.Second, 1*time.Second).Should(Equal(v1.Failed), "VMI should be failed")

				By(fmt.Sprintf("Waiting for %q %q event after the resource version %q", tests.WarningEvent, v1.Stopped, vmi.ResourceVersion))
				tests.NewObjectEventWatcher(vmi).Timeout(60*time.Second).SinceWatchedObjectResourceVersion().WaitFor(stopChan, tests.WarningEvent, v1.Stopped)

				By("checking that it can still start VMIs")
				newVMI := newCirrosVMI()
				newVMI.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(newVMI)
				Expect(err).To(BeNil())

				tests.WaitForSuccessfulVMIStart(newVMI)
			})
		})

		Context("[Serial]when virt-handler is responsive", func() {
			It("[test_id:1633]should indicate that a node is ready for vmis", func() {

				By("adding a heartbeat annotation and a schedulable label to the node")
				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
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

			It("[test_ids:3198]device plugins should re-register if the kubelet restarts", func() {

				By("starting a VMI on a node")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Should submit VMI successfully")

				// Start a VirtualMachineInstance
				nodeName := tests.WaitForSuccessfulVMIStart(vmi)

				By("triggering a device plugin re-registration on that node")
				pod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
				Expect(err).ToNot(HaveOccurred())

				_, _, err = tests.ExecuteCommandOnPodV2(virtClient, pod,
					"virt-handler",
					[]string{
						"rm",
						// We want to fail if the file does not exist, but don't want to be asked
						// if we really want to remove write-protected files
						"--interactive=never",
						device_manager.SocketPath(device_manager.KVMName),
					})
				Expect(err).ToNot(HaveOccurred())

				By("checking if we see the device plugin restart in the logs")
				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
				Expect(err).ToNot(HaveOccurred(), "Should get virthandler client for node")

				handlerName := virtHandlerPod.GetObjectMeta().GetName()
				handlerNamespace := virtHandlerPod.GetObjectMeta().GetNamespace()
				seconds := int64(10)
				logsQuery := virtClient.CoreV1().Pods(handlerNamespace).GetLogs(handlerName, &k8sv1.PodLogOptions{SinceSeconds: &seconds, Container: "virt-handler"})
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
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
				newVMI := newCirrosVMI()
				newVMI.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": nodeName}
				newVMI, err = virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(newVMI)
				Expect(err).To(BeNil())

				tests.WaitForSuccessfulVMIStart(newVMI)
			})
		})

		Context("[Serial]when virt-handler is not responsive", func() {

			var vmi *v1.VirtualMachineInstance
			var nodeName string
			var virtHandler *k8sv1.Pod
			var virtHandlerAvailablePods int32

			BeforeEach(func() {

				// Schedule a vmi and make sure that virt-handler gets evicted from the node where the vmi was started
				vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "echo hi!")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI successfully")
				nodeName = tests.WaitForSuccessfulVMIStart(vmi)
				virtHandler, err = kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
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

			It("[test_id:1634]the node controller should mark the node as unschedulable when the virt-handler heartbeat has timedout", func() {

				// Update virt-handler heartbeat, to trigger a timeout
				data := []byte(fmt.Sprintf(`{"metadata": { "labels": { "%s": "true" }, "annotations": {"%s": "%s"}}}`, v1.NodeSchedulable, v1.VirtHandlerHeartbeat, nowAsJSONWithOffset(-10*time.Minute)))
				_, err = virtClient.CoreV1().Nodes().Patch(nodeName, types.StrategicMergePatchType, data)
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				// Delete vmi pod
				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
				Expect(pods.Items).To(HaveLen(1), "There should be only one VMI pod")
				var gracePeriod int64 = 0
				Expect(virtClient.CoreV1().Pods(vmi.Namespace).Delete(pods.Items[0].Name, &metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriod,
				})).To(Succeed(), "The vmi pod should be deleted successfully")

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
				}, 180*time.Second, 1*time.Second).Should(BeTrue(), "Virthandler should be ready to work")
			})
		})

		Context("[Serial]with node tainted", func() {
			var nodes *k8sv1.NodeList
			var err error
			BeforeEach(func() {
				nodes = tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

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

			It("[test_id:1635]the vmi with tolerations should be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Tolerations = []k8sv1.Toleration{{Key: "test", Value: "123"}}
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:1636]the vmi without tolerations should not be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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
				nodes = tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
			})

			It("[test_id:1637]the vmi with node affinity and no conflicts should be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "Updated VMI name run on the same node")

			})

			It("[test_id:1638]the vmi with node affinity and anti-pod affinity should not be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				addNodeAffinityToVMI(vmi, nodes.Items[0].Name)
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(nodes.Items[0].Name), "VMI should run on the same node")

				vmiB := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
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

		Context("[Serial]with default cpu model", func() {
			var cfgMap *k8sv1.ConfigMap
			var originalData map[string]string
			var options metav1.GetOptions
			var defaultCPUModelKey = "default-cpu-model"
			var defaultCPUModel = "Nehalem"
			var vmiCPUModel = "SandyBridge"

			//store old kubevirt-config
			BeforeEach(func() {
				cfgMap, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				originalData = cfgMap.Data
			})

			//replace new kubevirt-config with old config
			AfterEach(func() {
				cfgMap, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
				Expect(err).ToNot(HaveOccurred())
				cfgMap.Data = originalData
				_, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Update(cfgMap)
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(5 * time.Second)
			})

			It("[test_id:3199]should set default cpu model when vmi doesn't have it set", func() {
				tests.UpdateClusterConfigValueAndWait(defaultCPUModelKey, defaultCPUModel)

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)
				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal("Nehalem"), "Expected default CPU model")

			})

			It("[test_id:3200]should not set default cpu model when vmi has it set", func() {
				tests.UpdateClusterConfigValueAndWait(defaultCPUModelKey, defaultCPUModel)

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Model: vmiCPUModel,
				}
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU.Model).To(Equal(vmiCPUModel), "Expected vmi CPU model")

			})

			It("[test_id:3201]should not set cpu model when vmi does not have it set and default cpu model is not set", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")

				tests.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Spec.Domain.CPU).To(BeNil(), "Expected CPU to be nil")
			})
		})

		Context("[Serial]with node feature discovery", func() {

			var node *k8sv1.Node
			var originalLabels map[string]string

			BeforeEach(func() {
				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")

				node = &nodes.Items[0]
				originalLabels = node.GetObjectMeta().GetLabels()

				tests.UpdateClusterConfigValueAndWait(virtconfig.FeatureGatesKey, virtconfig.CPUNodeDiscoveryGate)
			})

			AfterEach(func() {
				Expect(err).ToNot(HaveOccurred())
				labelBytes, err := json.Marshal(originalLabels)
				Expect(err).ToNot(HaveOccurred())

				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": %s}}`, labelBytes)))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				time.Sleep(5 * time.Second)
			})

			It("[test_id:1639]the vmi with cpu.model matching a nfd label on a node should be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: "Conroe",
				}

				cpuModelLabel, err := services.CPUModelLabelFromCPUModel(vmi)
				Expect(err).ToNot(HaveOccurred(), "CPU model label should have been retrieved successfully")

				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, cpuModelLabel)))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(node.Name), "VMI should run on a node with matching NFD CPU label")

			})

			It("[test_id:1640]the vmi with cpu.model that cannot match an nfd label on node should not be scheduled", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Model: "Conroe",
				}

				cpuModelLabel, err := services.CPUModelLabelFromCPUModel(vmi)
				Expect(err).ToNot(HaveOccurred(), "CPU model label should have been retrieved successfully")

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

			It("[test_id:3202]the vmi with cpu.features matching nfd labels on a node should be scheduled", func() {

				By("adding a node-feature-discovery CPU model label to a node")
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   "lahf_lm",
							Policy: "require",
						},
						{
							Name:   "mmx",
							Policy: "disable",
						},
					},
				}

				labels := "{"
				featureLabels := services.CPUFeatureLabelsFromCPUFeatures(vmi)
				labels += `"` + featureLabels[0] + `"` + ":\"true\""
				for _, featurelabel := range featureLabels[1:] {
					labels += `,"` + featurelabel + `"` + ":\"true\""
				}
				labels += "}"

				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": %s }}`, labels)))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI")
				tests.WaitForSuccessfulVMIStart(vmi)

				curVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI")
				Expect(curVMI.Status.NodeName).To(Equal(node.Name), "VMI should run on a node with matching NFD CPU features labels")

			})

			It("[test_id:3203]the vmi with cpu.features that cannot match nfd labels on a node should not be scheduled", func() {

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   "lahf_lm",
							Policy: "require",
						},
						{
							Name:   "mmx",
							Policy: "disable",
						},
					},
				}

				labels := "{"
				featureLabels := services.CPUFeatureLabelsFromCPUFeatures(vmi)
				labels += `"` + featureLabels[0] + `"` + ":\"false\""
				for _, featurelabel := range featureLabels[1:] {
					labels += `,"` + featurelabel + `"` + ":\"false\""
				}
				labels += "}"

				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": %s }}`, labels)))
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

			It("[test_id:3204]the vmi with cpu.feature policy 'forbid' should not be scheduled on a node with that cpu feature label", func() {

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 1,
					Features: []v1.CPUFeature{
						{
							Name:   "monitor",
							Policy: "forbid",
						},
					},
				}

				// Add node affinity first to test later on that although there is node affinity to
				// the specific node - the feature policy 'forbid' will deny shceduling on that node.
				addNodeAffinityToVMI(vmi, node.Name)

				node, err = virtClient.CoreV1().Nodes().Patch(node.Name, types.StrategicMergePatchType,
					[]byte(fmt.Sprintf(`{"metadata": { "labels": {"%s": "true"}}}`, services.NFD_CPU_FEATURE_PREFIX+"monitor")))
				Expect(err).ToNot(HaveOccurred(), "Should patch node successfully")

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
				}, 60*time.Second, 1*time.Second).Should(Equal("Unschedulable"), "VMI should be unschedulable")
			})

		})

		Context("with non default namespace", func() {
			table.DescribeTable("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]should log libvirt start and stop lifecycle events of the domain", func(namespace *string) {

				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				By("Creating a VirtualMachineInstance with different namespace")
				vmi = tests.NewRandomVMIWithNS(*namespace)
				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
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
				Expect(vmi.GetObjectMeta().GetNamespace()).To(Equal(*namespace), "VMI should run in the right namespace")

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
				/*
						Since we deleted the VMI object, there are two possible outcomes and both are expected:
						1. virt-controller kicks in, registers a deletion request on the launcher pod and K8s deletes the pod
					       before virt-handler had a chance to set or check the deletion timestamp on the domain.
						2. virt-handler detects the deletion timestamp on the domain and removes it.

						TODO: https://github.com/kubevirt/kubevirt/issues/3764
				*/
				Eventually(func() string {
					data, err := logsQuery.DoRaw()
					Expect(err).ToNot(HaveOccurred(), "Should get the virthandler logs")
					return string(data)
				}, 30, 0.5).Should(SatisfyAny(
					MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain is marked for deletion","name":"%s"`, vmi.GetObjectMeta().GetName()),               // Domain was deleted by virt-handler
					MatchRegexp(`"kind":"Domain","level":"info","msg":"Domain is in state Shutoff reason Destroyed","name":"%s"`, vmi.GetObjectMeta().GetName()), // Domain was destroyed because the launcher pod is gone
				), "Logs should confirm pod deletion")
			},
				table.Entry("[test_id:1641]"+tests.NamespaceTestDefault, &tests.NamespaceTestDefault),
				table.Entry("[test_id:1642]"+tests.NamespaceTestAlternative, &tests.NamespaceTestAlternative),
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

			It("[test_id:1643]should enable emulation in virt-launcher", func() {
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

			It("[test_id:1644]should be reflected in domain XML", func() {
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

			It("[test_id:1645]should request a TUN device but not KVM", func() {
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

			It("[test_id:1646]should request a KVM and TUN device", func() {
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

			It("[test_id:1647]should not enable emulation in virt-launcher", func() {
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

			It("[test_id:1648]Should provide KVM via plugin framework", func() {
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

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Get a VirtualMachineInstance", func() {
		Context("when that not exist", func() {
			It("[test_id:1649]should return 404", func() {
				b, err := virtClient.RestClient().Get().Resource("virtualmachineinstances").Namespace(tests.NamespaceTestDefault).Name("nonexistnt").DoRaw()
				Expect(err).ToNot(BeNil(), "Should get VMIs")
				status := metav1.Status{}
				err = json.Unmarshal(b, &status)
				Expect(err).To(BeNil(), "Unmarshal without error")
				Expect(status.Code).To(Equal(int32(http.StatusNotFound)), "There should not be and VMI")
			})
		})
	})

	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Delete a VirtualMachineInstance's Pod", func() {
		It("[test_id:1650]should result in the VirtualMachineInstance moving to a finalized state", func() {
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
	Describe("[rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Delete a VirtualMachineInstance", func() {
		Context("with an active pod.", func() {
			It("[test_id:1651]should result in pod being terminated", func() {

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
			It("[test_id:1652]should result in vmi status failed", func() {

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
			table.DescribeTable("[rfe_id:273][crit:medium][vendor:cnv-qe@redhat.com][level:component]should result in vmi status succeeded", func(gracePeriod int64) {
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
				table.Entry("[test_id:1653]with set grace period seconds", int64(10)),
				table.Entry("[test_id:1654]with default grace period seconds", int64(-1)),
			)
		})
		Context("with grace period greater than 0", func() {
			It("[test_id:1655]should run graceful shutdown", func() {
				nodes := tests.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				node := nodes.Items[0].Name

				virtHandlerPod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(node).Pod()
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

	Describe("[Serial][rfe_id:273][crit:high][vendor:cnv-qe@redhat.com][level:component]Killed VirtualMachineInstance", func() {
		It("[test_id:1656]should be in Failed phase", func() {
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

		It("[test_id:1657]should be left alone by virt-handler", func() {
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

	Describe("Defaults", func() {
		Context("FSGroup", func() {
			It("[test_id:4120]Should run with qemu as supplemental group", func() {
				By("Starting VirtualMachineInstance")
				vmi, err := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(vmi)
				Expect(err).To(BeNil(), "Create VMI successfully")
				tests.WaitForSuccessfulVMIStart(vmi)

				By("Checking supplemental groups of PID 1")
				pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(pod).NotTo(BeNil())
				output, err := tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					pod.Spec.Containers[0].Name,
					[]string{"ps", "-o", "supgrp", "1"},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(output).To(ContainSubstring("qemu"))

				By("Looking up qemu's UID")
				output, err = tests.ExecuteCommandOnPod(
					virtClient,
					pod,
					pod.Spec.Containers[0].Name,
					[]string{"id", "-g", "qemu"},
				)
				Expect(err).ToNot(HaveOccurred())

				qemuGroup, err := strconv.Atoi(strings.TrimSpace(output))
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.SecurityContext.FSGroup).ToNot(BeNil())
				Expect(int(*pod.Spec.SecurityContext.FSGroup)).To(Equal(qemuGroup))
			})
		})
	})
})

func shouldUseEmulation(virtClient kubecli.KubevirtClient) bool {
	useEmulation := false
	options := metav1.GetOptions{}
	cfgMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(kubevirtConfig, options)
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

func renderPkillAllPod(processName string) *k8sv1.Pod {
	return tests.RenderPod("vmi-killer", []string{"pkill"}, []string{"-9", processName})
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

	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			Container: "compute",
		}).
		DoRaw()
	Expect(err).To(BeNil(), "Should get virt-launcher pod logs")

	return string(logsRaw)
}

func pkillHandler(virtCli kubecli.KubevirtClient, node string) error {
	pod := renderPkillAllPod("virt-handler")
	pod.Spec.NodeName = node
	createdPod, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(pod)
	Expect(err).ToNot(HaveOccurred(), "Should create helper pod")

	getStatus := func() k8sv1.PodPhase {
		podG, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Get(createdPod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred(), "Should return current status")
		return podG.Status.Phase
	}

	Eventually(getStatus, 30, 0.5).Should(Equal(k8sv1.PodSucceeded), "Pod should end itself")

	return err
}

func pkillAllLaunchers(virtCli kubecli.KubevirtClient, node string) (*k8sv1.Pod, error) {
	pod := renderPkillAllPod("virt-launcher")
	pod.Spec.NodeName = node
	return virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(pod)
}

func pkillAllVMIs(virtCli kubecli.KubevirtClient, node string) error {
	pod := renderPkillAllPod("qemu")
	pod.Spec.NodeName = node
	_, err := virtCli.CoreV1().Pods(tests.NamespaceTestDefault).Create(pod)

	return err
}

func nowAsJSONWithOffset(offset time.Duration) string {
	now := metav1.Now()
	now = metav1.NewTime(now.Add(offset))

	data, err := json.Marshal(now)
	Expect(err).ToNot(HaveOccurred(), "Should marshal to json")
	return strings.Trim(string(data), `"`)
}
