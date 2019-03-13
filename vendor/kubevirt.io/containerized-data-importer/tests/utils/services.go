package utils

import (
	"github.com/golang/glog"
	"github.com/pkg/errors"

	"k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// GetServiceInNamespaceOrDie attempts to get the service in namespace `ns` by `name`.  Returns pointer to service on
// success.  Panics on error.
func GetServiceInNamespaceOrDie(c *kubernetes.Clientset, ns, name string) *v1.Service {
	svc, err := GetServiceInNamespace(c, ns, name)
	if err != nil || svc == nil {
		glog.Fatal(err)
	}
	return svc
}

// GetServiceInNamespace retries get on service `name` in namespace `ns` until timeout or IsNotFound error.  Ignores
// api errors that may be intermittent.  Returns pointer to service (nil on error) or an error (nil on success)
func GetServiceInNamespace(c *kubernetes.Clientset, ns, name string) (*v1.Service, error) {
	var svc *v1.Service
	err := wait.PollImmediate(defaultPollInterval, defaultPollPeriod, func() (done bool, err error) {
		svc, err = c.CoreV1().Services(ns).Get(name, metav1.GetOptions{})
		// success
		if err == nil {
			return true, nil
		}
		// fail if the service does not exist
		if apierrs.IsNotFound(err) {
			return false, errors.Wrap(err, "Service not found")
		}
		// log non-fatal errors
		glog.Error(errors.Wrapf(err, "Encountered non-fatal error getting service \"%s/%s\". retrying", ns, name))
		return false, nil
	})
	return svc, err
}

// GetServicesInNamespaceByLabel retries get of services in namespace `ns` by `labelSelector` until timeout.  Ignores
// api errors that may be intermittent.  Returns pointer to ServiceList (nil on error) and an error (nil on success)
func GetServicesInNamespaceByLabel(c *kubernetes.Clientset, ns, labelSelector string) (*v1.ServiceList, error) {
	var svcList *v1.ServiceList
	err := wait.PollImmediate(defaultPollInterval, defaultPollPeriod, func() (done bool, err error) {
		svcList, err = c.CoreV1().Services(ns).List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		// success
		if err == nil {
			return true, nil
		}
		// log non-fatal errors
		glog.Error(errors.Wrapf(err, "Encountered non-fatal error getting service list in namespace %s", ns))
		return false, nil
	})
	return svcList, err
}

// GetServicePortByName scans a service's ports for names matching `name`.  Returns integer port value (0 on error) and
// error (nil on success).
func GetServicePortByName(svc *v1.Service, name string) (int, error) {
	if svc == nil {
		return 0, errors.New("nil service")
	}

	var port int
	for _, p := range svc.Spec.Ports {
		if p.Name == name {
			port = int(p.Port)
			break
		}
	}
	if port == 0 {
		return 0, errors.Errorf("port %q not found", name)
	}
	return port, nil
}

// GetServiceNodePortByName scans a service's nodePorts for a name matching the `name` parameter and returns
// the associated port integer or an error if not match is found.
func GetServiceNodePortByName(svc *v1.Service, name string) (int, error) {
	if svc == nil {
		return 0, errors.New("nil service")
	}

	var port int
	for _, p := range svc.Spec.Ports {
		if p.Name == name {
			port = int(p.NodePort)
			break
		}
	}
	if port == 0 {
		return 0, errors.Errorf("port %q not found", name)
	}
	return port, nil
}
