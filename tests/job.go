package tests

import (
	"fmt"
	"strconv"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	. "github.com/onsi/gomega"
)

func WaitForJobToSucceed(virtClient *kubecli.KubevirtClient, job *batchv1.Job, timeoutSec time.Duration) {
	EventuallyWithOffset(1, func() bool {
		job, err := (*virtClient).BatchV1().Jobs(job.Namespace).Get(job.Name, metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		Expect(job.Status.Failed).NotTo(Equal(*job.Spec.BackoffLimit), "Job was expected to succeed but failed")
		return job.Status.Succeeded > 0
	}, timeoutSec*time.Second, 1*time.Second).Should(BeTrue(), "Job should succeed")

}

func WaitForJobToFail(virtClient *kubecli.KubevirtClient, job *batchv1.Job, timeoutSec time.Duration) {
	EventuallyWithOffset(1, func() int32 {
		job, err := (*virtClient).BatchV1().Jobs(job.Namespace).Get(job.Name, metav1.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		Expect(job.Status.Succeeded).ToNot(BeNumerically(">", 0), "Job should not succeed")
		return job.Status.Failed
	}, timeoutSec*time.Second, 1*time.Second).Should(BeNumerically(">", 0), "Job should fail")
}

func RenderJob(name string, cmd, args []string) *batchv1.Job {
	pod := RenderPod(name, cmd, args)
	job := batchv1.Job{
		ObjectMeta: pod.ObjectMeta,
		Spec: batchv1.JobSpec{
			BackoffLimit:            NewInt32(3),
			TTLSecondsAfterFinished: NewInt32(60),
			ActiveDeadlineSeconds:   NewInt64(480),
			Template: k8sv1.PodTemplateSpec{
				ObjectMeta: pod.ObjectMeta,
				Spec:       pod.Spec,
			},
		},
	}
	return &job
}

// NewHelloWorldJob takes a DNS entry or an IP and a port which it will use to create a job
// which tries to contact the host on the provided port.
// It expects to receive "Hello World!" to succeed.
func NewHelloWorldJob(host string, port string) *batchv1.Job {
	check := []string{fmt.Sprintf(`set -x; x="$(head -n 1 < <(nc %s %s -i 3 -w 3))"; echo "$x" ; if [ "$x" = "Hello World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`, host, port)}
	job := RenderJob("netcat", []string{"/bin/bash", "-c"}, check)
	return job
}

// NewHelloWorldJobUDP takes a DNS entry or an IP and a port which it will use create a pod
// which tries to contact the host on the provided port. It expects to receive "Hello World!" to succeed.
// Note that in case of UDP, the server will not see the connection unless something is sent over it
// However, netcat does not work well with UDP and closes before the answer arrives, for that another netcat call is needed,
// this time as a UDP listener
func NewHelloWorldJobUDP(host string, port string) *batchv1.Job {
	localPort, err := strconv.Atoi(port)
	if err != nil {
		return nil
	}
	// local port is used to catch the reply - any number can be used
	// we make it different than the port to be safe if both are running on the same machine
	localPort--
	check := []string{fmt.Sprintf(`set -x; trap "kill 0" EXIT; x="$(head -n 1 < <(echo | nc -up %d %s %s -i 3 -w 3 & nc -ul %d))"; echo "$x" ; if [ "$x" = "Hello UDP World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`,
		localPort, host, port, localPort)}
	job := RenderJob("netcat", []string{"/bin/bash", "-c"}, check)

	return job
}

// NewHelloWorldJobHTTP gets an IP address and a port, which it uses to create a pod.
// This pod tries to contact the host on the provided port, over HTTP.
// On success - it expects to receive "Hello World!".
func NewHelloWorldJobHTTP(host string, port string) *batchv1.Job {
	check := []string{fmt.Sprintf(`set -x; x="$(head -n 1 < <(curl %s:%s))"; echo "$x" ; if [ "$x" = "Hello World!" ]; then echo "succeeded"; exit 0; else echo "failed"; exit 1; fi`, FormatIPForURL(host), port)}
	job := RenderJob("curl", []string{"/bin/bash", "-c"}, check)

	return job
}
