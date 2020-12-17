package tests_test

import (
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	testscore "kubevirt.io/kubevirt/tests"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"kubevirt.io/kubevirt/tests/flags"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const timeout = 360 * time.Second
const pollingInterval = 5 * time.Second

var _ = Describe("Virtual Machines", func() {
	tests.FlagParse()
	client, err := kubecli.GetKubevirtClient()
	testscore.PanicOnError(err)

	var (
		workloadsNode      *corev1.Node
		checkNodePlacement = false
	)
	workloadsNodes, err := client.CoreV1().Nodes().List(k8smetav1.ListOptions{
		LabelSelector: "node.kubernetes.io/hco-test-node-type==workloads",
	})

	if err == nil && workloadsNodes != nil && len(workloadsNodes.Items) == 1 {
		checkNodePlacement = true
		workloadsNode = &workloadsNodes.Items[0]

		fmt.Fprintf(GinkgoWriter, "Found Workloads Node. Node name: %s; node labels:\n", workloadsNode.Name)
		w := json.NewEncoder(GinkgoWriter)
		w.SetIndent("", "  ")

		w.Encode(workloadsNode.Labels)

		Context("validate node placement in workloads nodes", func() {
			expectedWorkloadsPods := map[string]bool{
				"bridge-marker":   false,
				"cni-plugins":     false,
				//"kube-multus":     false,
				"nmstate-handler": false,
				"ovs-cni-marker":  false,
				"virt-handler":    false,
			}

			var cnaoCR networkaddonsv1.NetworkAddonsConfig

			s := scheme.Scheme
			_ = apis.AddToScheme(s)
			s.AddKnownTypes(networkaddonsv1.SchemeGroupVersion)
			opts := k8smetav1.GetOptions{}
			err = client.RestClient().Get().
				Resource("networkaddonsconfigs").
				Name("cluster").
				VersionedParams(&opts, scheme.ParameterCodec).
				Timeout(10 * time.Second).
				Do().Into(&cnaoCR)

			if cnaoCR.Spec.Ovs == nil {
				delete(expectedWorkloadsPods, "ovs-cni-marker")
			}

			pods, err := client.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(k8smetav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.nodeName=%s", workloadsNode.Name),
			})
			It("should read 'workloads' node's pods", func() {
				Expect(err).ToNot(HaveOccurred())
			})

			for _, pod := range pods.Items {
				podName := pod.Spec.Containers[0].Name
				fmt.Fprintf(GinkgoWriter, "Found %s pod '%s' in the 'workloads' node %s\n", podName, pod.Name, workloadsNode.Name)
				if found, ok := expectedWorkloadsPods[podName]; ok {
					if !found {
						expectedWorkloadsPods[podName] = true
					}
				}
			}

			It("all expected 'workloads' pod must be on infra node", func() {
				Expect(expectedWorkloadsPods).ToNot(ContainElement(false))
			})
		})

		Context("validate node placement on infra nodes", func() {
			infraNodes, err := client.CoreV1().Nodes().List(k8smetav1.ListOptions{
				LabelSelector: "node.kubernetes.io/hco-test-node-type==infra",
			})

			It("should get infra nodes", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})

			expectedInfraPods := map[string]bool{
				"cdi-apiserver":        false,
				"cdi-controller":       false,
				"cdi-uploadproxy":      false,
				"manager":              false,
				"nmstate-webhook":      false,
				"virt-api":             false,
				"virt-controller":      false,
				"vm-import-controller": false,
			}

			for _, node := range infraNodes.Items {
				pods, err := client.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(k8smetav1.ListOptions{
					FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
				})
				It("should read 'infra' node's pods", func() {
					Expect(err).ToNot(HaveOccurred())
				})

				for _, pod := range pods.Items {
					podName := pod.Spec.Containers[0].Name
					fmt.Fprintf(GinkgoWriter, "Found %s pod '%s' in the 'infra' node %s\n", podName, pod.Name, node.Name)
					if found, ok := expectedInfraPods[podName]; ok {
						if !found {
							expectedInfraPods[podName] = true
						}
					}
				}
			}

			It("all expected 'infra' pod must be on infra node", func() {
				Expect(expectedInfraPods).ToNot(ContainElement(false))
			})
		})
	}

	BeforeEach(func() {
		tests.BeforeEach()
	})

	Context("vmi testing", func() {
		for i := 0; i < 20; i++ {
			It(fmt.Sprintf("should create, verify and delete a vmi; run #%d", i), func() {
				vmi := testscore.NewRandomVMI()
				vmiName := vmi.Name
				Eventually(func() error {
					_, err := client.VirtualMachineInstance(testscore.NamespaceTestDefault).Create(vmi)
					return err
				}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to create a vmi")
				Eventually(func() bool {
					vmi, err = client.VirtualMachineInstance(testscore.NamespaceTestDefault).Get(vmiName, &k8smetav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if vmi.Status.Phase == kubevirtv1.Running {
						if checkNodePlacement {
							Expect(vmi.Labels["kubevirt.io/nodeName"]).Should(Equal(workloadsNode.Name))
							fmt.Fprintf(GinkgoWriter, "The VMI is running on the right node: %s\n", workloadsNode.Name)
						}
						return true
					}
					return false
				}, timeout, pollingInterval).Should(BeTrue(), "failed to get the vmi Running")
				Eventually(func() error {
					err := client.VirtualMachineInstance(testscore.NamespaceTestDefault).Delete(vmiName, &k8smetav1.DeleteOptions{})
					return err
				}, timeout, pollingInterval).Should(Not(HaveOccurred()), "failed to delete a vmi")
			})
		}
	})
})
