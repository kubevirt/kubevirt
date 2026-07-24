package envtest_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/envtest/framework"
)

var _ = Describe("Virt Operator", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.WithVirtOperator())
		f.Start()
	})

	AfterEach(func() {
		f.Stop()
	})

	It("should install KubeVirt and reach Deployed phase", func() {
		kv := &virtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
		}
		var err error
		kv, err = f.VirtClient().KubeVirt("kubevirt").Create(ctx, kv, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the operator to transition to Deploying")
		Eventually(func() virtv1.KubeVirtPhase {
			kv, err = f.VirtClient().KubeVirt("kubevirt").Get(ctx, "kubevirt", metav1.GetOptions{})
			if err != nil {
				return ""
			}
			return kv.Status.Phase
		}, 30*time.Second, 500*time.Millisecond).Should(Equal(virtv1.KubeVirtPhaseDeploying))

		By("waiting for the operator to reach Deployed phase")
		Eventually(func(g Gomega) {
			kv, err = f.VirtClient().KubeVirt("kubevirt").Get(ctx, "kubevirt", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			depls, _ := f.K8sClient().AppsV1().Deployments("kubevirt").List(ctx, metav1.ListOptions{})
			ds, _ := f.K8sClient().AppsV1().DaemonSets("kubevirt").List(ctx, metav1.ListOptions{})
			pods, _ := f.K8sClient().CoreV1().Pods("kubevirt").List(ctx, metav1.ListOptions{})

			var deplInfo []string
			for _, d := range depls.Items {
				deplInfo = append(deplInfo, fmt.Sprintf("%s(ready=%d/%d)", d.Name, d.Status.ReadyReplicas, d.Status.Replicas))
			}
			var dsInfo []string
			for _, d := range ds.Items {
				dsInfo = append(dsInfo, fmt.Sprintf("%s(ready=%d/%d)", d.Name, d.Status.NumberReady, d.Status.DesiredNumberScheduled))
			}

			fmt.Printf("[state] phase=%s depls=%v ds=%v pods=%d\n", kv.Status.Phase, deplInfo, dsInfo, len(pods.Items))
			g.Expect(kv.Status.Phase).To(Equal(virtv1.KubeVirtPhaseDeployed))
		}, 120*time.Second, 5*time.Second).Should(Succeed())

		By("verifying deployment IDs match")
		Expect(kv.Status.ObservedDeploymentID).NotTo(BeEmpty())
		Expect(kv.Status.ObservedDeploymentID).To(Equal(kv.Status.TargetDeploymentID))
	})
})
