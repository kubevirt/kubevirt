package tests

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/util"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

func NewHTTPServerPod(port int) *corev1.Pod {
	serverCommand := fmt.Sprintf("nc -klp %d --sh-exec 'echo -e \"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"'", port)
	return RenderPrivilegedPod("http-hello-world-server", []string{"/bin/bash"}, []string{"-c", serverCommand})
}

func NewTCPServerPod(port int) *corev1.Pod {
	serverCommand := fmt.Sprintf("nc -klp %d --sh-exec 'echo \"Hello World!\"'", port)
	return RenderPrivilegedPod("tcp-hello-world-server", []string{"/bin/bash"}, []string{"-c", serverCommand})
}

func CreatePodAndWaitUntil(pod *corev1.Pod, phaseToWait corev1.PodPhase) *corev1.Pod {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	pod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Create(context.Background(), pod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred(), "should succeed creating pod")

	getStatus := func() corev1.PodPhase {
		pod, err = virtClient.CoreV1().Pods(util.NamespaceTestDefault).Get(context.Background(), pod.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return pod.Status.Phase
	}
	Eventually(getStatus, 30, 1).Should(Equal(phaseToWait), "should reach %s phase", phaseToWait)
	return pod
}

func StartTCPServerPod(port int) *corev1.Pod {
	By(fmt.Sprintf("Start TCP Server pod at port %d", port))
	return CreatePodAndWaitUntil(NewTCPServerPod(port), corev1.PodRunning)
}

func StartHTTPServerPod(port int) *corev1.Pod {
	By(fmt.Sprintf("Start HTTP Server pod at port %d", port))
	return CreatePodAndWaitUntil(NewHTTPServerPod(port), corev1.PodRunning)
}
