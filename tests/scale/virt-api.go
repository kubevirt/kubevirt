package scale

import (
	"context"
	"strconv"
	"time"

	v12 "kubevirt.io/api/core/v1"

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
	numberOfNodes := 0

	setccs := func(ccs v12.CustomizeComponents) (oldcss v12.CustomizeComponents) {
		originalKv := util.GetCurrentKv(virtClient)
		kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		oldcss = kv.Spec.CustomizeComponents
		kv.Spec.CustomizeComponents = ccs
		EventuallyWithOffset(1, func() error {
			_, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			return err
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

		return oldcss
	}

	restorescc := func(ccs v12.CustomizeComponents) {
		originalKv := util.GetCurrentKv(virtClient)
		kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(originalKv.Name, &v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		kv.Spec.CustomizeComponents = v12.CustomizeComponents{}

		EventuallyWithOffset(1, func() error {
			_, err = virtClient.KubeVirt(originalKv.Namespace).Update(kv)
			return err
		}, 30*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	}

	getApiReplicas := func(virtClient kubecli.KubevirtClient, expectedResult int32) int32 {
		By("Finding out virt-api replica number")
		kv := util.GetCurrentKv(virtClient)
		Expect(kv).ToNot(BeNil())

		apiDeployment, err := virtClient.AppsV1().Deployments(kv.GetNamespace()).Get(context.Background(), components.VirtAPIName, v1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Expecting number of replicas to be as expected")
		Expect(apiDeployment.Spec.Replicas).ToNot(BeNil())

		return *apiDeployment.Spec.Replicas
	}
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		if numberOfNodes == 0 {
			By("Finding out nodes count")
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), v1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())

			numberOfNodes = len(nodes.Items)
		}
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
		expectedResult := calcExpectedReplicas(numberOfNodes)
		Eventually(func() int32 {
			return getApiReplicas(virtClient, expectedResult)
		}, 1*time.Minute, 5*time.Second).Should(Equal(calcExpectedReplicas(numberOfNodes)), "number of virt API should be as expected")
	})

	It("[Serial]virt-api replicas should be determined by patch if exist", Serial, func() {
		expectedResult := calcExpectedReplicas(numberOfNodes)
		expectedResult += 1
		ccs := v12.CustomizeComponents{
			Patches: []v12.CustomizeComponentsPatch{
				{
					ResourceName: components.VirtAPIName,
					ResourceType: "Deployment",
					Patch:        `[{"op":"replace","path":"/spec/replicas","value":` + strconv.Itoa(int(expectedResult)) + `}]`,
					Type:         v12.JSONPatchType,
				},
			},
		}
		oldcss := setccs(ccs)
		DeferCleanup(restorescc, oldcss)

		Eventually(func() int32 {
			return getApiReplicas(virtClient, expectedResult)
		}, 1*time.Minute, 5*time.Second).Should(Equal(expectedResult), "number of virt API should be as expected")
	})

})
