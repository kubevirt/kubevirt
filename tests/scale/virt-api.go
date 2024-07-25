package scale

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
)

var _ = Describe("[sig-compute] virt-api scaling", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
	numberOfNodes := 0

	setccs := func(ccs v1.CustomizeComponents, kvNamespace string, kvName string) error {
		patchPayload, err := patch.New(patch.WithReplace("/spec/customizeComponents", ccs)).GeneratePayload()
		if err != nil {
			return err
		}
		_, err = virtClient.KubeVirt(kvNamespace).Patch(context.Background(), kvName, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
		return err
	}

	getApiReplicas := func() int32 {
		By("Finding out virt-api replica number")
		apiDeployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), components.VirtAPIName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(apiDeployment.Spec.Replicas).ToNot(BeNil(), "The number of replicas in the virt-api deployment should not be nil")

		return *apiDeployment.Spec.Replicas
	}
	BeforeEach(func() {
		virtClient = kubevirt.Client()
		if numberOfNodes == 0 {
			By("Finding out nodes count")
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
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
		Eventually(func() int32 {
			return getApiReplicas()
		}, 1*time.Minute, 5*time.Second).Should(Equal(calcExpectedReplicas(numberOfNodes)), "number of virt API should be as expected")
	})

	It("[Serial]virt-api replicas should be determined by patch if exist", Serial, func() {
		originalKv := libkubevirt.GetCurrentKv(virtClient)
		expectedResult := calcExpectedReplicas(numberOfNodes)
		expectedResult += 1
		ccs := v1.CustomizeComponents{
			Patches: []v1.CustomizeComponentsPatch{
				{
					ResourceName: components.VirtAPIName,
					ResourceType: "Deployment",
					Patch:        `[{"op":"replace","path":"/spec/replicas","value":` + strconv.Itoa(int(expectedResult)) + `}]`,
					Type:         v1.JSONPatchType,
				},
			},
		}
		err := setccs(ccs, originalKv.Namespace, originalKv.Name)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(setccs, originalKv.Spec.CustomizeComponents, originalKv.Namespace, originalKv.Name)

		Eventually(func() int32 {
			return getApiReplicas()
		}, 1*time.Minute, 5*time.Second).Should(Equal(expectedResult), "number of virt API should be as expected")
	})

})
