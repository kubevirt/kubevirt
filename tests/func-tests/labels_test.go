package tests_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/gertd/go-pluralize"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Check that all the sub-resources have the required labels", Label("labels"), func() {
	tests.FlagParse()
	var (
		cli kubecli.KubevirtClient
		ctx context.Context
	)

	BeforeEach(func() {
		var err error
		cli, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()
	})

	It("should have all the required labels in all the controlled resources", func() {
		hc := tests.GetHCO(ctx, cli)
		plural := pluralize.NewClient()

		for _, resource := range hc.Status.RelatedObjects {
			By(fmt.Sprintf("checking labels for %s/%s", resource.Kind, resource.Name))
			parts := strings.Split(resource.APIVersion, "/")
			if len(parts) == 1 {
				switch resource.Kind {
				case "ConfigMap":
					cm, err := cli.CoreV1().ConfigMaps(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					checkLabels(cm.GetLabels())

				case "Service":
					svc, err := cli.CoreV1().Services(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					checkLabels(svc.GetLabels())
				default:
					GinkgoWriter.Printf("Missed corev1 resource to check the labels for; %s/%s\n", resource.Kind, resource.Name)
				}
			} else {
				kind := plural.Plural(strings.ToLower(resource.Kind))
				gvr := schema.GroupVersionResource{
					Group:    parts[0],
					Version:  parts[1],
					Resource: kind,
				}
				if len(resource.Namespace) == 0 {
					rc, err := cli.DynamicClient().Resource(gvr).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					checkLabels(rc.GetLabels())
				} else {
					rc, err := cli.DynamicClient().Resource(gvr).Namespace(resource.Namespace).Get(ctx, resource.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
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
