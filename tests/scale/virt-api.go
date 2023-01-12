package scale

import (
	"context"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute] virt-api scaling", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	calcExpectedReplicas := func(nodesCount int) (expectedReplicas int32) {
		// Please note that this logic is temporary. For more information take a look on the comment in
		// getDesiredApiReplicas() function in pkg/virt-operator/resource/apply/apps.go.
		//
		// When the logic is replaced for getDesiredApiReplicas(), it needs to be replaced here as well.

		if nodesCount == 1 {
			return 1
		}

		const minReplicas = 2

		expectedReplicas = int32(nodesCount) / 10
		if expectedReplicas < minReplicas {
			expectedReplicas = minReplicas
		}

		return expectedReplicas
	}

	It("virt-api replicas should be scaled as expected", func() {
		By("Finding out nodes count")
		nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())

		numberOfNodes := len(nodes.Items)

		Eventually(func() int32 {
			By("Finding out virt-api replica number")
			kv := util.GetCurrentKv(virtClient)
			Expect(kv).ToNot(BeNil())

			apiDeployment, err := virtClient.AppsV1().Deployments(kv.GetNamespace()).Get(context.Background(), components.VirtAPIName, v1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Expecting number of replicas to be as expected")
			Expect(apiDeployment.Spec.Replicas).ToNot(BeNil())

			return *apiDeployment.Spec.Replicas
		}, 1*time.Minute, 5*time.Second).Should(Equal(calcExpectedReplicas(numberOfNodes)), "number of virt API should be as expected")
	})

})
