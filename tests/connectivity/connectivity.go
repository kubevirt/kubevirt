package connectivity

import (
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
)

type JobCreationFunction func(virtClient kubecli.KubevirtClient, host, port, namespace string) (*batchv1.Job, error)

func RunJobOnNode(virtClient kubecli.KubevirtClient, host, port, namespace string, function JobCreationFunction, nodeName string, op k8sv1.NodeSelectorOperator) (*batchv1.Job, error) {
	job, err := function(virtClient, host, port, namespace)
	return pinJobOnNode(job, nodeName, op), err
}

func pinJobOnNode(job *batchv1.Job, nodeName string, op k8sv1.NodeSelectorOperator) *batchv1.Job {
	job.Spec.Template.Spec.Affinity = &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
				NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
					{
						MatchExpressions: []k8sv1.NodeSelectorRequirement{
							{Key: "kubernetes.io/hostname", Operator: op, Values: []string{nodeName}},
						},
					},
				},
			},
		},
	}
	return job
}

func SetHostNetwork(job *batchv1.Job) *batchv1.Job {
	job.Spec.Template.Spec.HostNetwork = true
	return job
}

func RunHelloWorldJob(virtClient kubecli.KubevirtClient, host, port, namespace string) (*batchv1.Job, error) {
	job := NewHelloWorldJob(host, port)
	return virtClient.BatchV1().Jobs(namespace).Create(job)
}

func RunHelloWorldJobUDP(virtClient kubecli.KubevirtClient, host, port, namespace string) (*batchv1.Job, error) {
	job := NewHelloWorldJobUDP(host, port)
	return virtClient.BatchV1().Jobs(namespace).Create(job)
}

func RunHelloWorldJobHttp(virtClient kubecli.KubevirtClient, host, port, namespace string) (*batchv1.Job, error) {
	job := NewHelloWorldJobHTTP(host, port)
	return virtClient.BatchV1().Jobs(namespace).Create(job)
}

func AssertConnectivityToServiceByIP(virtClient kubecli.KubevirtClient, jobCreationFunc JobCreationFunction, host, namespace string, servicePort int) (func() error, error) {
	job, deleteCallback, err := startJobAndReturnDeleteCallback(virtClient, jobCreationFunc, host, namespace, fmt.Sprintf("%d", servicePort))
	if err != nil {
		return nil, err
	}

	err = WaitForJobToSucceed(job, 90*time.Second)
	return deleteCallback, err
}

func AssertNoConnectivityToServiceByIP(virtClient kubecli.KubevirtClient, jobCreationFunc JobCreationFunction, host, namespace string, servicePort int) (func() error, error) {
	job, deleteCallback, err := startJobAndReturnDeleteCallback(virtClient, jobCreationFunc, host, namespace, fmt.Sprintf("%d", servicePort))
	if err != nil {
		return nil, err
	}

	err = WaitForJobToFail(job, 90*time.Second)
	return deleteCallback, err
}

func AssertConnectivityToService(virtClient kubecli.KubevirtClient, jobCreationFunc JobCreationFunction, serviceName, namespace string, servicePort int) (func() error, error) {
	serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)
	job, deleteCallback, err := startJobAndReturnDeleteCallback(virtClient, jobCreationFunc, serviceFQDN, namespace, fmt.Sprintf("%d", servicePort))
	if err != nil {
		return nil, err
	}

	err = WaitForJobToSucceed(job, 90*time.Second)
	return deleteCallback, err
}

func AssertNoConnectivityToService(virtClient kubecli.KubevirtClient, jobCreationFunc JobCreationFunction, serviceName, namespace string, servicePort int) (func() error, error) {
	serviceFQDN := fmt.Sprintf("%s.%s", serviceName, namespace)
	job, deleteCallback, err := startJobAndReturnDeleteCallback(virtClient, jobCreationFunc, serviceFQDN, namespace, fmt.Sprintf("%d", servicePort))
	if err != nil {
		return nil, err
	}

	err = WaitForJobToFail(job, 90*time.Second)
	return deleteCallback, err
}

func startJobAndReturnDeleteCallback(virtClient kubecli.KubevirtClient, jobCreationFunc JobCreationFunction, host string, namespace string, port string) (*batchv1.Job, func() error, error) {
	job, err := jobCreationFunc(virtClient, host, port, namespace)

	return job, func() error {
		return virtClient.BatchV1().Jobs(job.GetNamespace()).Delete(job.GetName(), &k8smetav1.DeleteOptions{})
	}, err
}
