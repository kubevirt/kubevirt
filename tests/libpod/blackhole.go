package libpod

import (
	"context"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/exec"
)

func AddKubernetesAPIBlackhole(pods *v1.PodList, containerName string) {
	kubernetesAPIServiceBlackhole(pods, containerName, true)
}

func DeleteKubernetesAPIBlackhole(pods *v1.PodList, containerName string) {
	kubernetesAPIServiceBlackhole(pods, containerName, false)
}

func kubernetesAPIServiceBlackhole(pods *v1.PodList, containerName string, present bool) {
	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).NotTo(HaveOccurred())

	serviceIP := getKubernetesAPIServiceIP(virtCli)

	var addOrDel string
	if present {
		addOrDel = "add"
	} else {
		addOrDel = "del"
	}

	for idx := range pods.Items {
		_, err = exec.ExecuteCommandOnPod(virtCli, &pods.Items[idx], containerName, []string{"ip", "route", addOrDel, "blackhole", serviceIP})
		Expect(err).NotTo(HaveOccurred())
	}
}

func getKubernetesAPIServiceIP(virtClient kubecli.KubevirtClient) string {
	const serviceName = "kubernetes"
	const serviceNamespace = "default"

	kubernetesService, err := virtClient.CoreV1().Services(serviceNamespace).Get(context.Background(), serviceName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return kubernetesService.Spec.ClusterIP
}
