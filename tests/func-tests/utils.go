package tests

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/tests"
)

const (
	KubevirtCfgMap = "kubevirt-config"
)

var KubeVirtStorageClassLocal string

func init() {
	flag.StringVar(&KubeVirtStorageClassLocal, "storage-class-local", "local", "Storage provider to use for tests which want local storage")
}

//GetJobTypeEnvVar returns "JOB_TYPE" enviroment varibale
func GetJobTypeEnvVar() string {
	return (os.Getenv("JOB_TYPE"))
}

func FlagParse() {
	flag.Parse()
}

func ForwardPortsFromService(service *k8sv1.Service, ports []string, stop chan struct{}, readyTimeout time.Duration) error {
	selector := labels.FormatLabels(service.Spec.Selector)

	targetPorts := []string{}
	for _, p := range ports {
		split := strings.Split(p, ":")
		if len(split) != 2 {
			return fmt.Errorf("invalid port mapping for %s", p)
		}
		found := false
		for _, servicePort := range service.Spec.Ports {
			if split[1] == strconv.Itoa(int(servicePort.Port)) {
				targetPorts = append(targetPorts, split[0]+":"+servicePort.TargetPort.String())
				found = true
				break
			}
		}
		if found == false {
			return fmt.Errorf("Port %s not found on service", split[1])
		}
	}
	cli, err := kubecli.GetKubevirtClient()
	if err != nil {
		return err
	}

	pods, err := cli.CoreV1().Pods(service.Namespace).List(v1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		return err
	}

	var targetPod *k8sv1.Pod
ForLoop:
	for _, pod := range pods.Items {
		if pod.Status.Phase != k8sv1.PodRunning {
			continue
		}
		for _, conditions := range pod.Status.Conditions {
			if conditions.Type == k8sv1.PodReady && conditions.Status == k8sv1.ConditionTrue {
				targetPod = &pod
				break ForLoop
			}
		}
	}

	if targetPod == nil {
		return fmt.Errorf("No ready pod listening on the service.")
	}

	return tests.ForwardPorts(targetPod, targetPorts, stop, readyTimeout)
}

func IsOpenShift() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	isOpenShift, err := cluster.IsOnOpenShift(virtClient)
	if err != nil {
		fmt.Printf("ERROR: Can not determine cluster type %v\n", err)
		panic(err)
	}

	return isOpenShift
}

func SkipIfNotOpenShift(message string) {
	if !IsOpenShift() {
		ginkgo.Skip("Not running on openshift: " + message)
	}
}

func BeforeEach() {
	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	tests.PanicOnError(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachines").Do().Error())
	tests.PanicOnError(virtClient.RestClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("virtualmachineinstances").Do().Error())
	tests.PanicOnError(virtClient.CoreV1().RESTClient().Delete().Namespace(tests.NamespaceTestDefault).Resource("persistentvolumeclaims").Do().Error())
}
