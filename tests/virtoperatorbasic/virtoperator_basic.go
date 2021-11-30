package virtoperatorbasic

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
)

var _ = Describe("virt-operator basic tests", func() {

	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		checks.SkipTestIfNoCPUManager()
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeTestCleanup()
	})

	Context("[sig-compute][serial]Obsolete ConfigMap", func() {
		ctx := context.Background()
		virtOpLabelSelector := metav1.ListOptions{
			LabelSelector: "kubevirt.io=virt-operator",
		}

		AfterEach(func() {
			// cleanup the obsolete configMap
			_ = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Delete(ctx, "kubevirt-config", metav1.DeleteOptions{})

			// make sure virt-operators are up before leaving
			Eventually(func() bool {
				pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(ctx, virtOpLabelSelector)
				if err != nil {
					fmt.Fprintln(GinkgoWriter, "can't get the virt-operator pods")
					return false
				}

				for _, virtOpPod := range pods.Items {
					for _, containerStatus := range virtOpPod.Status.ContainerStatuses {
						if !containerStatus.Ready {
							fmt.Fprintf(GinkgoWriter, "The %s container in pod %s is not ready yet\n", containerStatus.Name, virtOpPod.Name)
							return false
						}
					}
				}

				return true
			}, time.Minute*5, time.Second*2).Should(BeTrue())

		})

		It("should emit event if the obsolete kubevirt-config configMap still exists", func() {
			cm := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt-config",
					Namespace: flags.KubeVirtInstallNamespace,
				},
				Data: map[string]string{
					"test": "data",
				},
			}

			_, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Create(ctx, cm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			err = virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).DeleteCollection(ctx, metav1.DeleteOptions{}, virtOpLabelSelector)
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() int {
				events, err := virtClient.CoreV1().Events(flags.KubeVirtInstallNamespace).List(
					context.Background(),
					metav1.ListOptions{
						FieldSelector: "involvedObject.kind=ConfigMap,involvedObject.name=kubevirt-config,type=Warning,reason=ObsoleteConfigMapExists",
					},
				)
				Expect(err).ToNot(HaveOccurred())
				return events.Size()
			}, time.Minute*5, time.Second*10).ShouldNot(BeZero())
		})
	})

})
