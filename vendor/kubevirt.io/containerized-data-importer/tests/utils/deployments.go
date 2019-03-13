package utils

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// WaitForDeploymentReplicasReadyOrDie adds the ability to fatal out if the replicas don't become ready
func WaitForDeploymentReplicasReadyOrDie(c *kubernetes.Clientset, namespace, name string) {
	if err := WaitForDeploymentReplicasReady(c, namespace, name); err != nil {
		glog.Fatal(errors.Wrapf(err, "Failed waiting for deployment \"%s/%s\" replicas to become Ready", namespace, name))
	}
}

// WaitForDeploymentReplicasReady will wait for replicase to become ready and return an error if they do not
func WaitForDeploymentReplicasReady(c *kubernetes.Clientset, namespace, name string) error {
	return wait.PollImmediate(defaultPollInterval, defaultPollPeriod, func() (done bool, err error) {
		dep, err := c.ExtensionsV1beta1().Deployments(namespace).Get(name, metav1.GetOptions{})
		// Fail if deployment not found, ignore other (possibly intermittent) API errors
		if apierrs.IsNotFound(err) {
			return true, err
		}
		// Log non-fatal errors
		if err != nil {
			glog.Error(errors.Wrapf(err, "Error getting deployment \"%s/%s\"", namespace, name))
		}
		// All replicas not ready, continue wait
		if dep.Status.ReadyReplicas != *dep.Spec.Replicas {
			return false, nil
		}
		// Replicas ready, done
		return true, nil
	})
}
