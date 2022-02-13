package virtoperatorbasic

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/flags"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe("virt-operator basic tests", func() {

	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		var err error
		virtClient, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())
		tests.BeforeTestCleanup()
	})

	Context("[sig-compute][Serial]Obsolete ConfigMap", func() {
		ctx := context.Background()
		virtOpLabelSelector := metav1.ListOptions{
			LabelSelector: "kubevirt.io=virt-operator",
		}

		AfterEach(func() {
			// cleanup the obsolete configMap
			_ = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Delete(ctx, "kubevirt-config", metav1.DeleteOptions{})

			// make sure virt-operators are up before leaving
			Eventually(ThisDeploymentWith(flags.KubeVirtInstallNamespace, components.VirtOperatorName), 180*time.Second, 1*time.Second).Should(HaveReadyReplicasNumerically(">", 0))
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
			Eventually(ThisDeploymentWith(flags.KubeVirtInstallNamespace, components.VirtOperatorName), 180*time.Second, 1*time.Second).Should(HaveReadyReplicasNumerically("==", 0))

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
