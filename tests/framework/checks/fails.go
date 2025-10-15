package checks

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

func FailTestIfNoFeatureGate(featureGate string) {
	if !HasFeature(featureGate) {
		ginkgo.Fail(fmt.Sprintf("the %v feature gate is not enabled.", featureGate))
	}
}

func RecycleImageOrFail(virtClient kubecli.KubevirtClient, imageName string) {
	windowsPv, err := virtClient.CoreV1().PersistentVolumes().Get(context.Background(), imageName, metav1.GetOptions{})
	if err != nil || windowsPv.Status.Phase == k8sv1.VolumePending || windowsPv.Status.Phase == k8sv1.VolumeFailed {
		ginkgo.Fail(fmt.Sprintf("Fail tests that requires PV %s", imageName))
	} else if windowsPv.Status.Phase == k8sv1.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), windowsPv, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func FailTestIfPrometheusRuleIsNotEnabled(virtClient kubecli.KubevirtClient) {
	ext, err := clientset.NewForConfig(virtClient.Config())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "prometheusrules.monitoring.coreos.com", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		ginkgo.Fail("PrometheusRule CRD is not available in the cluster")
	} else if err != nil {
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}

func FailTestIfRunningOnKindInfra(message string) {
	if IsRunningOnKindInfra() {
		ginkgo.Fail("Test cannot run on kind infra: " + message)
	}
}
