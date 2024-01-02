package libpod

import (
	"context"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/exec"
)

func AddKubernetesApiBlackhole(pods *v1.PodList, containerName string) {
	kubernetesApiServiceBlackhole(pods, containerName, true)
}

func DeleteKubernetesApiBlackhole(pods *v1.PodList, containerName string) {
	kubernetesApiServiceBlackhole(pods, containerName, false)
}

func kubernetesApiServiceBlackhole(pods *v1.PodList, containerName string, present bool) {
	virtCli, err := kubecli.GetKubevirtClient()
	Expect(err).NotTo(HaveOccurred())

	serviceIp := getKubernetesApiServiceIp(virtCli)

	var addOrDel string
	if present {
		addOrDel = "add"
	} else {
		addOrDel = "del"
	}

	for _, pod := range pods.Items {
		_, err = exec.ExecuteCommandOnPod(virtCli, &pod, containerName, []string{"ip", "route", addOrDel, "blackhole", serviceIp})
		Expect(err).NotTo(HaveOccurred())
	}
}

func getKubernetesApiServiceIp(virtClient kubecli.KubevirtClient) string {
	const kubernetesServiceName = "kubernetes"
	const kubernetesServiceNamespace = "default"

	kubernetesService, err := virtClient.CoreV1().Services(kubernetesServiceNamespace).Get(context.Background(), kubernetesServiceName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())

	return kubernetesService.Spec.ClusterIP
}
