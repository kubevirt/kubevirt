package libpod

import (
	"fmt"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"
)

func CreatePodAndWaitUntil(pod *k8sv1.Pod, phaseToWait k8sv1.PodPhase) (*k8sv1.Pod, error) {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		return nil, err
	}

	pod, err = virtClient.CoreV1().Pods(pod.GetNamespace()).Create(pod)
	if err != nil {
		return nil, err
	}

	getStatus := func(podName string, podNamespace string) (k8sv1.PodPhase, error) {
		pod, err = virtClient.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			return k8sv1.PodUnknown, err
		}
		return pod.Status.Phase, nil
	}

	timeout := 30 * time.Second
	err = wait.PollImmediate(time.Second, timeout, func() (bool, error) {
		podStatus, err := getStatus(pod.GetName(), pod.GetNamespace())
		return podStatus == phaseToWait, err
	})
	return pod, err
}

func RenderPod(name string, cmd []string, args []string) *k8sv1.Pod {
	privilegedContainer := true
	pod := k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: name,
			Labels: map[string]string{
				v1.AppLabel: "test",
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    name,
					Image:   fmt.Sprintf("%s/vm-killer:%s", flags.KubeVirtUtilityRepoPrefix, flags.KubeVirtUtilityVersionTag),
					Command: cmd,
					Args:    args,
					SecurityContext: &k8sv1.SecurityContext{
						Privileged: &privilegedContainer,
						RunAsUser:  new(int64),
					},
				},
			},
			HostPID: true,
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsUser: new(int64),
			},
		},
	}

	return &pod
}
