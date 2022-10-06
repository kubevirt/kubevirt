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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"

	utiltype "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libstorage"

	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/tests/libnode"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/network/dns"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	windowsDisk        = "windows-disk"
	windowsFirmware    = "5d307ca9-b3ef-428c-8861-06e72d69f223"
	windowsVMIUser     = "Administrator"
	windowsVMIPassword = "Heslo123"
)

const (
	winrmCli    = "winrmcli"
	winrmCliCmd = "winrm-cli"
)

func getWindowsVMISpec() v1.VirtualMachineInstanceSpec {
	gracePeriod := int64(0)
	spinlocks := uint32(8191)
	firmware := types.UID(windowsFirmware)
	_false := false
	return v1.VirtualMachineInstanceSpec{
		TerminationGracePeriodSeconds: &gracePeriod,
		Domain: v1.DomainSpec{
			CPU: &v1.CPU{Cores: 2},
			Features: &v1.Features{
				ACPI: v1.FeatureState{},
				APIC: &v1.FeatureAPIC{},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:    &v1.FeatureState{},
					SyNICTimer: &v1.SyNICTimer{Direct: &v1.FeatureState{}},
					VAPIC:      &v1.FeatureState{},
					Spinlocks:  &v1.FeatureSpinlocks{Retries: &spinlocks},
				},
			},
			Clock: &v1.Clock{
				ClockOffset: v1.ClockOffset{UTC: &v1.ClockOffsetUTC{}},
				Timer: &v1.Timer{
					HPET:   &v1.HPETTimer{Enabled: &_false},
					PIT:    &v1.PITTimer{TickPolicy: v1.PITTickPolicyDelay},
					RTC:    &v1.RTCTimer{TickPolicy: v1.RTCTickPolicyCatchup},
					Hyperv: &v1.HypervTimer{},
				},
			},
			Firmware: &v1.Firmware{UUID: firmware},
			Resources: v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("2048Mi"),
				},
			},
			Devices: v1.Devices{
				Disks: []v1.Disk{
					{
						Name: windowsDisk,
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusSATA,
							},
						},
					},
				},
			},
		},
		Volumes: []v1.Volume{
			{
				Name: windowsDisk,
				VolumeSource: v1.VolumeSource{
					Ephemeral: &v1.EphemeralVolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: tests.DiskWindows,
						},
					},
				},
			},
		},
	}

}

var _ = Describe("[Serial][sig-compute]Windows VirtualMachineInstance", func() {
	var err error
	var virtClient kubecli.KubevirtClient

	var windowsVMI *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
		tests.BeforeTestCleanup()
		checks.SkipIfMissingRequiredImage(virtClient, tests.DiskWindows)
		tests.CreatePVC(tests.OSWindows, "30Gi", libstorage.Config.StorageClassWindows, true)
		windowsVMI = tests.NewRandomVMI()
		windowsVMI.Spec = getWindowsVMISpec()
		tests.AddExplicitPodNetworkInterface(windowsVMI)
		windowsVMI.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
	})

	It("[test_id:487]should succeed to start a vmi", func() {
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
		Expect(err).To(BeNil())
		tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 360)
	})

	It("[test_id:488]should succeed to stop a running vmi", func() {
		By("Starting the vmi")
		vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
		Expect(err).To(BeNil())
		tests.WaitForSuccessfulVMIStartWithTimeout(vmi, 360)

		By("Stopping the vmi")
		err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &metav1.DeleteOptions{})
		Expect(err).To(BeNil())
	})

	Context("VMI with HyperV reenlightenment enabled", func() {

		type mapType string

		const (
			label      mapType = "label"
			annotation mapType = "annotation"
		)

		type mapAction string

		const (
			add    mapAction = "add"
			remove mapAction = "remove"
		)

		patchLabelAnnotationHelper := func(virtCli kubecli.KubevirtClient, nodeName string, newMap, oldMap map[string]string, mapType mapType) (*k8sv1.Node, error) {
			p := []utiltype.PatchOperation{
				{
					Op:    "test",
					Path:  "/metadata/" + string(mapType) + "s",
					Value: oldMap,
				},
				{
					Op:    "replace",
					Path:  "/metadata/" + string(mapType) + "s",
					Value: newMap,
				},
			}

			patchBytes, err := json.Marshal(p)
			Expect(err).ToNot(HaveOccurred())

			patchedNode, err := virtCli.CoreV1().Nodes().Patch(context.Background(), nodeName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			return patchedNode, err
		}

		// Adds or removes a label or annotation from a node. When removing a label/annotation, the "value" parameter
		// is ignored.
		addRemoveLabelAnnotationHelper := func(nodeName, key, value string, mapType mapType, mapAction mapAction) *k8sv1.Node {
			var fetchMap func(node *k8sv1.Node) map[string]string
			var mutateMap func(key, val string, m map[string]string) map[string]string

			switch mapType {
			case label:
				fetchMap = func(node *k8sv1.Node) map[string]string { return node.Labels }
			case annotation:
				fetchMap = func(node *k8sv1.Node) map[string]string { return node.Annotations }
			}

			switch mapAction {
			case add:
				mutateMap = func(key, val string, m map[string]string) map[string]string {
					m[key] = val
					return m
				}
			case remove:
				mutateMap = func(key, val string, m map[string]string) map[string]string {
					delete(m, key)
					return m
				}
			}

			Expect(fetchMap).ToNot(BeNil())
			Expect(mutateMap).ToNot(BeNil())

			virtCli, err := kubecli.GetKubevirtClient()
			Expect(err).ToNot(HaveOccurred())

			var nodeToReturn *k8sv1.Node
			EventuallyWithOffset(2, func() error {
				origNode, err := virtCli.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				originalMap := fetchMap(origNode)
				expectedMap := make(map[string]string, len(originalMap))
				for k, v := range originalMap {
					expectedMap[k] = v
				}

				expectedMap = mutateMap(key, value, expectedMap)

				if equality.Semantic.DeepEqual(originalMap, expectedMap) {
					// key and value already exist in node
					nodeToReturn = origNode
					return nil
				}

				patchedNode, err := patchLabelAnnotationHelper(virtCli, nodeName, expectedMap, originalMap, mapType)
				if err != nil {
					return err
				}

				resultMap := fetchMap(patchedNode)

				const errPattern = "adding %s (key: %s. value: %s) to node %s failed. Expected %ss: %v, actual: %v"
				if !equality.Semantic.DeepEqual(resultMap, expectedMap) {
					return fmt.Errorf(errPattern, string(mapType), key, value, nodeName, string(mapType), expectedMap, resultMap)
				}

				nodeToReturn = patchedNode
				return nil
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			return nodeToReturn
		}

		addAnnotationToNode := func(nodeName, key, value string) *k8sv1.Node {
			return addRemoveLabelAnnotationHelper(nodeName, key, value, annotation, add)
		}

		removeAnnotationFromNode := func(nodeName string, key string) *k8sv1.Node {
			return addRemoveLabelAnnotationHelper(nodeName, key, "", annotation, remove)
		}

		wakeNodeLabellerUp := func(virtClient kubecli.KubevirtClient) {
			const fakeModel = "fake-model-1423"

			By("Updating Kubevirt CR to wake node-labeller up")
			kvConfig := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
			if kvConfig.ObsoleteCPUModels == nil {
				kvConfig.ObsoleteCPUModels = make(map[string]bool)
			}
			kvConfig.ObsoleteCPUModels[fakeModel] = true
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
			delete(kvConfig.ObsoleteCPUModels, fakeModel)
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
		}

		stopNodeLabeller := func(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
			var err error
			var node *k8sv1.Node

			suiteConfig, _ := GinkgoConfiguration()
			Expect(suiteConfig.ParallelTotal).To(Equal(1), "stopping / resuming node-labeller is supported for serial tests only")

			By(fmt.Sprintf("Patching node to %s include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
			key, value := v1.LabellerSkipNodeAnnotation, "true"
			addAnnotationToNode(nodeName, key, value)

			By(fmt.Sprintf("Expecting node %s to include %s label", nodeName, v1.LabellerSkipNodeAnnotation))
			Eventually(func() bool {
				node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				value, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
				return exists && value == "true"
			}, 30*time.Second, time.Second).Should(BeTrue(), fmt.Sprintf("node %s is expected to have annotation %s", nodeName, v1.LabellerSkipNodeAnnotation))

			return node
		}

		resumeNodeLabeller := func(nodeName string, virtClient kubecli.KubevirtClient) *k8sv1.Node {
			var err error
			var node *k8sv1.Node

			node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			if _, isNodeLabellerStopped := node.Annotations[v1.LabellerSkipNodeAnnotation]; !isNodeLabellerStopped {
				// Nothing left to do
				return node
			}

			By(fmt.Sprintf("Patching node to %s not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
			removeAnnotationFromNode(nodeName, v1.LabellerSkipNodeAnnotation)

			// In order to make sure node-labeller has updated the node, the host-model label (which node-labeller
			// makes sure always resides on any node) will be removed. After node-labeller is enabled again, the
			// host model label would be expected to show up again on the node.
			By(fmt.Sprintf("Removing host model label %s from node %s (so we can later expect it to return)", v1.HostModelCPULabel, nodeName))
			for _, label := range node.Labels {
				if strings.HasPrefix(label, v1.HostModelCPULabel) {
					libnode.RemoveLabelFromNode(nodeName, label)
				}
			}

			wakeNodeLabellerUp(virtClient)

			By(fmt.Sprintf("Expecting node %s to not include %s annotation", nodeName, v1.LabellerSkipNodeAnnotation))
			Eventually(func() error {
				node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())

				_, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
				if exists {
					return fmt.Errorf("node %s is expected to not have annotation %s", node.Name, v1.LabellerSkipNodeAnnotation)
				}

				foundHostModelLabel := false
				for labelKey := range node.Labels {
					if strings.HasPrefix(labelKey, v1.HostModelCPULabel) {
						foundHostModelLabel = true
						break
					}
				}
				if !foundHostModelLabel {
					return fmt.Errorf("node %s is expected to have a label with %s prefix. this means node-labeller is not enabled for the node", nodeName, v1.HostModelCPULabel)
				}

				return nil
			}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())

			return node
		}

		BeforeEach(func() {
			windowsVMI.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: pointer.Bool(true)}
		})

		When("TSC frequency is exposed on the cluster", func() {
			It("should be able to migrate", func() {
				if !isTSCFrequencyExposed(virtClient) {
					Skip("TSC frequency is not exposed on the cluster")
				}

				var err error
				By("Creating a windows VM")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

				By("Migrating the VM")
				migration := tests.NewRandomMigration(windowsVMI.Name, windowsVMI.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				By("Checking VMI, confirm migration state")
				tests.ConfirmVMIPostMigration(virtClient, windowsVMI, migrationUID)
			})
		})

		When("TSC frequency is not exposed on the cluster", func() {

			BeforeEach(func() {
				if isTSCFrequencyExposed(virtClient) {
					nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					for _, node := range nodeList.Items {
						stopNodeLabeller(node.Name, virtClient)
						removeTSCFrequencyFromNode(node)
					}
				}
			})

			AfterEach(func() {
				nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, node := range nodeList.Items {
					_, isNodeLabellerStopped := node.Annotations[v1.LabellerSkipNodeAnnotation]
					Expect(isNodeLabellerStopped).To(BeTrue())

					updatedNode := resumeNodeLabeller(node.Name, virtClient)
					_, isNodeLabellerStopped = updatedNode.Annotations[v1.LabellerSkipNodeAnnotation]
					Expect(isNodeLabellerStopped).To(BeFalse(), "after node labeller is resumed, %s annotation is expected to disappear from node", v1.LabellerSkipNodeAnnotation)
				}
			})

			It("should be able to start successfully", func() {
				var err error
				By("Creating a windows VM")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)
				winrnLoginCommand(virtClient, windowsVMI)
			})

			It("should be marked as non-migratable", func() {
				var err error
				By("Creating a windows VM")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

				conditionManager := controller.NewVirtualMachineInstanceConditionManager()
				isNonMigratable := func() error {
					windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(windowsVMI.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					cond := conditionManager.GetCondition(windowsVMI, v1.VirtualMachineInstanceIsMigratable)
					const errFmt = "condition " + string(v1.VirtualMachineInstanceIsMigratable) + " is expected to be %s %s"

					if statusFalse := k8sv1.ConditionFalse; cond.Status != statusFalse {
						return fmt.Errorf(errFmt, "of status", string(statusFalse))
					}
					if notMigratableNoTscReason := v1.VirtualMachineInstanceReasonNoTSCFrequencyMigratable; cond.Reason != notMigratableNoTscReason {
						return fmt.Errorf(errFmt, "of reason", notMigratableNoTscReason)
					}
					if !strings.Contains(cond.Message, "HyperV Reenlightenment") {
						return fmt.Errorf(errFmt, "with message that contains", "HyperV Reenlightenment")
					}
					return nil
				}

				Eventually(isNonMigratable, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
				Consistently(isNonMigratable, 15*time.Second, 3*time.Second).ShouldNot(HaveOccurred())
			})
		})
	})

	Context("with winrm connection", func() {
		var winrmcliPod *k8sv1.Pod
		var cli []string
		var output string

		BeforeEach(func() {
			By("Creating winrm-cli pod for the future use")
			winrmcliPod = winRMCliPod()

			var err error
			winrmcliPod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), winrmcliPod, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("[ref_id:139]VMI is created", func() {

			BeforeEach(func() {
				By("Starting the windows VirtualMachineInstance")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).To(BeNil())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("[test_id:240]should have correct UUID", func() {
				command := append(cli, "wmic csproduct get \"UUID\"")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					output, err = tests.ExecuteCommandOnPod(
						virtClient,
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())
				By("Checking that the Windows VirtualMachineInstance has expected UUID")
				Expect(output).Should(ContainSubstring(strings.ToUpper(windowsFirmware)))
			})

			It("[test_id:3159]should have default masquerade IP", func() {
				command := append(cli, "ipconfig /all")
				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					output, err = tests.ExecuteCommandOnPod(
						virtClient,
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

				By("Checking that the Windows VirtualMachineInstance has expected IP address")
				Expect(output).Should(ContainSubstring("10.0.2.2"))
			})

			It("[test_id:3160]should have the domain set properly", func() {
				searchDomain := getPodSearchDomain(windowsVMI)
				Expect(searchDomain).To(HavePrefix(windowsVMI.Namespace), "should contain a searchdomain with the namespace of the VMI")

				runCommandAndExpectOutput(virtClient,
					winrmcliPod,
					cli,
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+searchDomain+`[\n\r\t ]+`)
			})
		})

		Context("VMI with subdomain is created", func() {
			BeforeEach(func() {
				windowsVMI.Spec.Subdomain = "subdomain"

				By("Starting the windows VirtualMachineInstance with subdomain")
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("should have the domain set properly with subdomain", func() {
				searchDomain := getPodSearchDomain(windowsVMI)
				Expect(searchDomain).To(HavePrefix(windowsVMI.Namespace), "should contain a searchdomain with the namespace of the VMI")

				expectedSearchDomain := windowsVMI.Spec.Subdomain + "." + searchDomain
				runCommandAndExpectOutput(virtClient,
					winrmcliPod,
					cli,
					"wmic nicconfig get dnsdomain",
					`DNSDomain[\n\r\t ]+`+expectedSearchDomain+`[\n\r\t ]+`)
			})
		})

		Context("with bridge binding", func() {
			BeforeEach(func() {
				By("Starting Windows VirtualMachineInstance with bridge binding")
				windowsVMI.Spec.Domain.Devices.Interfaces = []v1.Interface{libvmi.InterfaceDeviceWithBridgeBinding(libvmi.DefaultInterfaceName)}
				windowsVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(windowsVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 420)

				cli = winrnLoginCommand(virtClient, windowsVMI)
			})

			It("should be recognized by other pods in cluster", func() {

				By("Pinging virt-handler Pod from Windows VMI")

				windowsVMI, err = virtClient.VirtualMachineInstance(windowsVMI.Namespace).Get(windowsVMI.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				getVirtHandlerPod := func() (*k8sv1.Pod, error) {
					winVmiPod := tests.GetRunningPodByVirtualMachineInstance(windowsVMI, windowsVMI.Namespace)
					nodeName := winVmiPod.Spec.NodeName

					pod, err := kubecli.NewVirtHandlerClient(virtClient).Namespace(flags.KubeVirtInstallNamespace).ForNode(nodeName).Pod()
					if err != nil {
						return nil, fmt.Errorf("failed to get virt-handler pod on node %s: %v", nodeName, err)
					}
					return pod, nil
				}

				virtHandlerPod, err := getVirtHandlerPod()
				Expect(err).ToNot(HaveOccurred())

				virtHandlerPodIP := libnet.GetPodIpByFamily(virtHandlerPod, k8sv1.IPv4Protocol)

				command := append(cli, fmt.Sprintf("ping %s", virtHandlerPodIP))

				By(fmt.Sprintf("Running \"%s\" command via winrm-cli", command))
				Eventually(func() error {
					_, err = tests.ExecuteCommandOnPod(
						virtClient,
						winrmcliPod,
						winrmcliPod.Spec.Containers[0].Name,
						command,
					)
					return err
				}, time.Minute*1, time.Second*15).Should(Succeed())
			})
		})
	})

	Context("[ref_id:142]with kubectl command", func() {
		var yamlFile string
		BeforeEach(func() {
			clientcmd.SkipIfNoCmd("kubectl")
			yamlFile, err = tests.GenerateVMIJson(windowsVMI, GinkgoT().TempDir())
			Expect(err).ToNot(HaveOccurred())
		})

		It("[test_id:223]should succeed to start a vmi", func() {
			By("Starting the vmi via kubectl command")
			_, _, err = clientcmd.RunCommand("kubectl", "create", "-f", yamlFile)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)
		})

		It("[test_id:239]should succeed to stop a vmi", func() {
			By("Starting the vmi via kubectl command")
			_, _, err = clientcmd.RunCommand("kubectl", "create", "-f", yamlFile)
			Expect(err).ToNot(HaveOccurred())

			tests.WaitForSuccessfulVMIStartWithTimeout(windowsVMI, 360)

			podSelector := tests.UnfinishedVMIPodSelector(windowsVMI)
			By("Deleting the vmi via kubectl command")
			_, _, err = clientcmd.RunCommand("kubectl", "delete", "-f", yamlFile)
			Expect(err).ToNot(HaveOccurred())

			By("Checking that the vmi does not exist anymore")
			result := virtClient.RestClient().Get().Resource(tests.VMIResource).Namespace(k8sv1.NamespaceDefault).Name(windowsVMI.Name).Do(context.Background())
			Expect(result).To(testutils.HaveStatusCode(http.StatusNotFound))

			By("Checking that the vmi pod terminated")
			Eventually(func() int {
				pods, err := virtClient.CoreV1().Pods(util.NamespaceTestDefault).List(context.Background(), podSelector)
				Expect(err).ToNot(HaveOccurred())
				return len(pods.Items)
			}, 75, 0.5).Should(Equal(0))
		})
	})
})

func winrnLoginCommand(virtClient kubecli.KubevirtClient, windowsVMI *v1.VirtualMachineInstance) []string {
	var err error
	windowsVMI, err = virtClient.VirtualMachineInstance(windowsVMI.Namespace).Get(windowsVMI.Name, &metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	vmiIp := windowsVMI.Status.Interfaces[0].IP
	cli := []string{
		winrmCliCmd,
		"-hostname",
		vmiIp,
		"-username",
		windowsVMIUser,
		"-password",
		windowsVMIPassword,
	}

	return cli
}

func getPodSearchDomain(windowsVMI *v1.VirtualMachineInstance) string {
	By("fetching /etc/resolv.conf from the VMI Pod")
	resolvConf := tests.RunCommandOnVmiPod(windowsVMI, []string{"cat", "/etc/resolv.conf"})

	By("extracting the search domain of the VMI")
	searchDomains, err := dns.ParseSearchDomains(resolvConf)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	searchDomain := ""
	for _, s := range searchDomains {
		if len(searchDomain) < len(s) {
			searchDomain = s
		}
	}

	return searchDomain
}

func runCommandAndExpectOutput(virtClient kubecli.KubevirtClient, winrmcliPod *k8sv1.Pod, cli []string, command, expectedOutputRegex string) {
	cliCmd := append(cli, command)
	By(fmt.Sprintf("Running \"%s\" command via winrm-cli", cliCmd))
	By("first making sure that we can execute VMI commands")
	EventuallyWithOffset(1, func() error {
		_, err := tests.ExecuteCommandOnPod(
			virtClient,
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		return err
	}, time.Minute*5, time.Second*15).ShouldNot(HaveOccurred())

	By("repeatedly trying to get the search domain, since it may take some time until the domain is set")
	EventuallyWithOffset(1, func() string {
		output, err := tests.ExecuteCommandOnPod(
			virtClient,
			winrmcliPod,
			winrmcliPod.Spec.Containers[0].Name,
			cliCmd,
		)
		Expect(err).ToNot(HaveOccurred())
		return output
	}, time.Minute*1, time.Second*10).Should(MatchRegexp(expectedOutputRegex))
}

func isTSCFrequencyExposed(virtClient kubecli.KubevirtClient) bool {
	nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	for _, node := range nodeList.Items {
		if _, isExposed := node.Labels[topology.TSCFrequencyLabel]; isExposed {
			return true
		}
	}

	return false
}

func removeTSCFrequencyFromNode(node k8sv1.Node) {
	for _, baseLabelToRemove := range []string{topology.TSCFrequencyLabel, topology.TSCFrequencySchedulingLabel} {
		for key, _ := range node.Labels {
			if strings.HasPrefix(key, baseLabelToRemove) {
				libnode.RemoveLabelFromNode(node.Name, key)
			}
		}
	}
}
