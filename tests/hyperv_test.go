package tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"

	utiltype "kubevirt.io/kubevirt/pkg/util/types"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[Serial][sig-compute] Hyper-V enlightenments", func() {

	var (
		virtClient kubecli.KubevirtClient
		err        error
	)
	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
	})

	Context("VMI with HyperV re-enlightenment enabled", func() {
		var reEnlightenmentVMI *v1.VirtualMachineInstance

		withReEnlightenment := func(vmi *v1.VirtualMachineInstance) {
			if vmi.Spec.Domain.Features == nil {
				vmi.Spec.Domain.Features = &v1.Features{}
			}
			if vmi.Spec.Domain.Features.Hyperv == nil {
				vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
			}

			vmi.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: pointer.Bool(true)}
		}

		vmiWithReEnlightenment := func() *v1.VirtualMachineInstance {
			options := libvmi.WithMasqueradeNetworking()
			options = append(options, withReEnlightenment)
			return libvmi.NewAlpine(options...)
		}

		BeforeEach(func() {
			reEnlightenmentVMI = vmiWithReEnlightenment()
		})

		When("TSC frequency is exposed on the cluster", func() {
			BeforeEach(func() {
				if !isTSCFrequencyExposed(virtClient) {
					Skip("TSC frequency is not exposed on the cluster")
				}
			})

			It("should be able to migrate", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				reEnlightenmentVMI = tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)

				By("Migrating the VM")
				migration := tests.NewRandomMigration(reEnlightenmentVMI.Name, reEnlightenmentVMI.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				By("Checking VMI, confirm migration state")
				tests.ConfirmVMIPostMigration(virtClient, reEnlightenmentVMI, migrationUID)
			})

			It("should have TSC frequency set up in label and domain", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				reEnlightenmentVMI = tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)

				virtLauncherPod := tests.GetPodByVirtualMachineInstance(reEnlightenmentVMI)

				foundNodeSelector := false
				for key, _ := range virtLauncherPod.Spec.NodeSelector {
					if strings.HasPrefix(key, topology.TSCFrequencySchedulingLabel+"-") {
						foundNodeSelector = true
						break
					}
				}
				Expect(foundNodeSelector).To(BeTrue(), "wasn't able to find a node selector key with prefix ", topology.TSCFrequencySchedulingLabel)

				domainSpec, err := tests.GetRunningVMIDomainSpec(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())

				foundTscTimer := false
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						foundTscTimer = true
						break
					}
				}
				Expect(foundTscTimer).To(BeTrue(), "wasn't able to find tsc timer in domain spec")
			})
		})

		When("TSC frequency is not exposed on the cluster", func() {

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
					if !isNodeLabellerStopped {
						continue
					}

					updatedNode := resumeNodeLabeller(node.Name, virtClient)
					_, isNodeLabellerStopped = updatedNode.Annotations[v1.LabellerSkipNodeAnnotation]
					Expect(isNodeLabellerStopped).To(BeFalse(), "after node labeller is resumed, %s annotation is expected to disappear from node", v1.LabellerSkipNodeAnnotation)
				}
			})

			It("should be able to start successfully", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)
				Expect(console.LoginToAlpine(reEnlightenmentVMI)).To(Succeed())
			})

			It("should be marked as non-migratable", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)

				conditionManager := controller.NewVirtualMachineInstanceConditionManager()
				isNonMigratable := func() error {
					reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(reEnlightenmentVMI.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					cond := conditionManager.GetCondition(reEnlightenmentVMI, v1.VirtualMachineInstanceIsMigratable)
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

		It("the vmi with HyperV feature matching a nfd label on a node should be scheduled", func() {
			enableHyperVInVMI := func(label string) v1.FeatureHyperv {
				features := v1.FeatureHyperv{}
				trueV := true
				switch label {
				case "vpindex":
					features.VPIndex = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "runtime":
					features.Runtime = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "reset":
					features.Reset = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "synic":
					features.SyNIC = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "frequencies":
					features.Frequencies = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "reenlightenment":
					features.Reenlightenment = &v1.FeatureState{
						Enabled: &trueV,
					}
				}

				return features
			}
			var supportedKVMInfoFeature []string
			checks.SkipIfARM64(testsuite.Arch, "arm64 does not support cpu model")
			nodes := libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
			node := &nodes.Items[0]
			supportedCPUs := tests.GetSupportedCPUModels(*nodes)
			Expect(supportedCPUs).ToNot(BeEmpty(), "There should be some supported cpu models")

			for key := range node.Labels {
				if strings.Contains(key, services.NFD_KVM_INFO_PREFIX) &&
					!strings.Contains(key, "tlbflush") &&
					!strings.Contains(key, "ipi") &&
					!strings.Contains(key, "synictimer") {
					supportedKVMInfoFeature = append(supportedKVMInfoFeature, strings.TrimPrefix(key, services.NFD_KVM_INFO_PREFIX))
				}
			}

			for _, label := range supportedKVMInfoFeature {
				vmi := libvmi.NewCirros()
				features := enableHyperVInVMI(label)
				vmi.Spec.Domain.Features = &v1.Features{
					Hyperv: &features,
				}

				vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI when using %v", label)
				tests.WaitForSuccessfulVMIStart(vmi)

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI when using %v", label)
			}
		})

		DescribeTable("the vmi with EVMCS HyperV feature should have correct HyperV and cpu features auto filled", func(featureState *v1.FeatureState) {
			vmi := libvmi.NewCirros()
			vmi.Spec.Domain.Features = &v1.Features{
				Hyperv: &v1.FeatureHyperv{
					EVMCS: featureState,
				},
			}

			vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred(), "Should create VMI")

			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should get VMI")
			Expect(vmi.Spec.Domain.Features.Hyperv.EVMCS).ToNot(BeNil(), "evmcs should not be nil")
			Expect(vmi.Spec.Domain.CPU).ToNot(BeNil(), "cpu topology can't be nil")
			if featureState.Enabled == nil || *featureState.Enabled == true {
				Expect(vmi.Spec.Domain.Features.Hyperv.VAPIC).ToNot(BeNil(), "vapic should not be nil")
				Expect(vmi.Spec.Domain.CPU.Features).To(HaveLen(1), "cpu topology has to contain 1 feature")
				Expect(vmi.Spec.Domain.CPU.Features[0].Name).To(Equal(nodelabellerutil.VmxFeature), "vmx cpu feature should be requested")
			} else {
				Expect(vmi.Spec.Domain.Features.Hyperv.VAPIC).To(BeNil(), "vapic should be nil")
				Expect(vmi.Spec.Domain.CPU.Features).To(BeEmpty())
			}

		},
			Entry("hyperv and cpu features should be auto filled when EVMCS is enabled", &v1.FeatureState{Enabled: pointer.BoolPtr(true)}),
			Entry("EVMCS should be enabled when vmi.Spec.Domain.Features.Hyperv.EVMCS is set but the EVMCS.Enabled field is nil ", &v1.FeatureState{Enabled: nil}),
			Entry("Verify that features aren't applied when enabled is false", &v1.FeatureState{Enabled: pointer.BoolPtr(false)}),
		)
	})
})
