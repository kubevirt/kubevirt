package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gertd/go-pluralize"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kvv1 "kubevirt.io/api/core/v1"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Check that all the sub-resources have the required labels", Label("labels"), func() {
	tests.FlagParse()
	var (
		cli    client.Client
		cliSet *kubernetes.Clientset
		ctx    context.Context
	)

	BeforeEach(func() {
		cli = tests.GetControllerRuntimeClient()
		cliSet = tests.GetK8sClientSet()

		ctx = context.Background()
	})

	It("should have all the required labels in all the controlled resources", func() {
		hc := tests.GetHCO(ctx, cli)
		plural := pluralize.NewClient()
		const kvName = "kubevirt-kubevirt-hyperconverged"

		By("removing one of the managed labels and wait for it to be added back")
		kv := &kvv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kvName,
				Namespace: hc.Namespace,
			},
		}

		Expect(cli.Get(ctx, client.ObjectKeyFromObject(kv), kv)).To(Succeed())
		expectedVersion := kv.Labels[hcoutil.AppLabelVersion]

		patchBytes := []byte(`[{"op": "remove", "path": "/metadata/labels/app.kubernetes.io~1version"}]`)
		patch := client.RawPatch(types.JSONPatchType, patchBytes)

		Eventually(func() error {
			return cli.Patch(ctx, kv, patch)
		}).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(kv), kv)).To(Succeed())
			g.Expect(kv.Labels).To(HaveKeyWithValue(hcoutil.AppLabelVersion, expectedVersion))
		}).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(Succeed())

		By("checking all the labels")
		for _, resource := range hc.Status.RelatedObjects {
			By(fmt.Sprintf("checking labels for %s/%s", resource.Kind, resource.Name))
			parts := strings.Split(resource.APIVersion, "/")
			if len(parts) == 1 {
				switch resource.Kind {
				case "ConfigMap":
					cm, err := cliSet.CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					checkLabels(cm.GetLabels())

				case "Service":
					svc, err := cliSet.CoreV1().Services(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					checkLabels(svc.GetLabels())
				default:
					GinkgoWriter.Printf("Missed corev1 resource to check the labels for; %s/%s\n", resource.Kind, resource.Name)
				}
			} else {
				dynamicClient, err := dynamic.NewForConfig(tests.GetClientConfig())
				Expect(err).ToNot(HaveOccurred())

				kind := plural.Plural(strings.ToLower(resource.Kind))
				gvr := schema.GroupVersionResource{
					Group:    parts[0],
					Version:  parts[1],
					Resource: kind,
				}
				if len(resource.Namespace) == 0 {
					rc, err := dynamicClient.Resource(gvr).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					checkLabels(rc.GetLabels())
				} else {
					rc, err := dynamicClient.Resource(gvr).Namespace(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					checkLabels(rc.GetLabels())
				}
			}
		}
	})
})

func checkLabels(labels map[string]string) {
	ExpectWithOffset(1, labels).Should(HaveKey("app.kubernetes.io/component"))
	ExpectWithOffset(1, labels).Should(HaveKey("app.kubernetes.io/version"))
	ExpectWithOffset(1, labels).Should(HaveKeyWithValue("app", "kubevirt-hyperconverged"))
	ExpectWithOffset(1, labels).Should(HaveKeyWithValue("app.kubernetes.io/part-of", "hyperconverged-cluster"))
	ExpectWithOffset(1, labels).Should(HaveKeyWithValue("app.kubernetes.io/managed-by", "hco-operator"))
}
