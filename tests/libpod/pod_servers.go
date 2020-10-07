package libpod

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

func NewHTTPServerPod(port int) *corev1.Pod {
	serverCommand := fmt.Sprintf("nc -klp %d --sh-exec 'echo -e \"HTTP/1.1 200 OK\\nContent-Length: 12\\n\\nHello World!\"'", port)
	return RenderPod("http-hello-world-server", []string{"/bin/bash"}, []string{"-c", serverCommand})
}

func NewTCPServerPod(port int) *corev1.Pod {
	serverCommand := fmt.Sprintf("nc -klp %d --sh-exec 'echo \"Hello World!\"'", port)
	return RenderPod("tcp-hello-world-server", []string{"/bin/bash"}, []string{"-c", serverCommand})
}

func StartTCPServerPod(port int) (*corev1.Pod, error) {
	return CreatePodAndWaitUntil(NewTCPServerPod(port), corev1.PodRunning)
}

func StartHTTPServerPod(port int) (*corev1.Pod, error) {
	return CreatePodAndWaitUntil(NewTCPServerPod(port), corev1.PodRunning)
}
