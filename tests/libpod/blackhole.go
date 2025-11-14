package libpod

import (
	"context"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/k8s"
)

func AddKubernetesAPIBlackhole(pods *v1.PodList, containerName string) {
	kubernetesAPIServiceBlackhole(pods, containerName, true)
}

func DeleteKubernetesAPIBlackhole(pods *v1.PodList, containerName string) {
	kubernetesAPIServiceBlackhole(pods, containerName, false)
}

func kubernetesAPIServiceBlackhole(pods *v1.PodList, containerName string, present bool) {
	serviceIP := getKubernetesAPIServiceIP()

	var addOrDel string
	if present {
		addOrDel = "add"
	} else {
		addOrDel = "del"
	}

	for idx := range pods.Items {
		_, err := exec.ExecuteCommandOnPod(&pods.Items[idx], containerName, []string{"ip", "route", addOrDel, "blackhole", serviceIP})
		Expect(err).NotTo(HaveOccurred())
	}
}

func getKubernetesAPIServiceIP() string {
	const serviceName = "kubernetes"
	const serviceNamespace = "default"

	kubernetesService, err := k8s.Client().CoreV1().Services(serviceNamespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return kubernetesService.Spec.ClusterIP
}
