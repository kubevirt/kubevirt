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
	"kubevirt.io/kubevirt/tests/libnode"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
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
			It("should be able to migrate", func() {
				if !isTSCFrequencyExposed(virtClient) {
					Skip("TSC frequency is not exposed on the cluster")
				}

				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)

				By("Migrating the VM")
				migration := tests.NewRandomMigration(reEnlightenmentVMI.Name, reEnlightenmentVMI.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				By("Checking VMI, confirm migration state")
				tests.ConfirmVMIPostMigration(virtClient, reEnlightenmentVMI, migrationUID)
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
					Expect(isNodeLabellerStopped).To(BeTrue())

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
	})

})
