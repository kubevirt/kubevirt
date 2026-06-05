package checks

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
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
		ginkgo.Fail(fmt.Sprintf("Skip tests that requires PV %s", imageName))
	} else if windowsPv.Status.Phase == k8sv1.VolumeReleased {
		windowsPv.Spec.ClaimRef = nil
		_, err = virtClient.CoreV1().PersistentVolumes().Update(context.Background(), windowsPv, metav1.UpdateOptions{})
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}
}
