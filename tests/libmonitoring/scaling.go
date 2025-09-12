package libmonitoring

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/flags"
)

type Scaling struct {
	k8sClient kubernetes.Interface
	scales    map[string]int32
}

func NewScaling(k8sClient kubernetes.Interface, deployments []string) *Scaling {
	s := &Scaling{
		k8sClient: k8sClient,
		scales:    make(map[string]int32, len(deployments)),
	}

	for _, operatorName := range deployments {
		s.BackupScale(operatorName)
	}

	return s
}

func (s *Scaling) BackupScale(operatorName string) {
	By("Backing up scale for " + operatorName)
	Eventually(func() error {
		virtOperatorCurrentScale, err := s.k8sClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).GetScale(context.TODO(), operatorName, metav1.GetOptions{})
		if err == nil {
			s.scales[operatorName] = virtOperatorCurrentScale.Spec.Replicas
		}
		return err
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}

func (s *Scaling) GetScale(operatorName string) int32 {
	return s.scales[operatorName]
}

func (s *Scaling) UpdateScale(operatorName string, replicas int32) {
	By(fmt.Sprintf("Updating scale for %s to %d", operatorName, replicas))

	Eventually(func() error {
		scale, err := s.k8sClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).GetScale(context.TODO(), operatorName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		scale.Spec.Replicas = replicas

		_, err = s.k8sClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).UpdateScale(context.TODO(), operatorName, scale, metav1.UpdateOptions{})
		return err
	}, 30*time.Second, 1*time.Second).Should(Succeed())

	Eventually(func() int32 {
		deployment, err := s.k8sClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.TODO(), operatorName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return deployment.Status.ReadyReplicas
	}, 2*time.Minute, 10*time.Second).Should(Equal(replicas), "failed to verify updated replicas for %s", operatorName)
}

func (s *Scaling) RestoreAllScales() {
	for operatorName := range s.scales {
		s.RestoreScale(operatorName)
	}
}

func (s *Scaling) RestoreScale(operatorName string) {
	revert := s.scales[operatorName]
	s.UpdateScale(operatorName, revert)
}
