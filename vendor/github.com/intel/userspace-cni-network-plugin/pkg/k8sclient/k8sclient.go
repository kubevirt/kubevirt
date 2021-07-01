// Copyright 2017-2020 Intel Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package k8sclient

import (
	"context"
	"net"
	"os"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	"github.com/containernetworking/cni/pkg/skel"
	cnitypes "github.com/containernetworking/cni/pkg/types"

	"github.com/intel/userspace-cni-network-plugin/logging"
)

// k8sArgs is the valid CNI_ARGS used for Kubernetes
type k8sArgs struct {
	cnitypes.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               cnitypes.UnmarshallableString
	K8S_POD_NAMESPACE          cnitypes.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID cnitypes.UnmarshallableString
}

func getK8sArgs(args *skel.CmdArgs) (*k8sArgs, error) {
	k8sArgs := &k8sArgs{}

	logging.Verbosef("getK8sArgs: %v", args)

	if args == nil {
		return nil, logging.Errorf("getK8sArgs: failed to get k8s args for CmdArgs set to %v", args)
	}
	err := cnitypes.LoadArgs(args.Args, k8sArgs)
	if err != nil {
		return nil, err
	}

	return k8sArgs, nil
}

func getK8sClient(kubeClient kubernetes.Interface, kubeConfig string) (kubernetes.Interface, error) {
	logging.Verbosef("getK8sClient: %s, %v", kubeClient, kubeConfig)

	// If we get a valid kubeClient (eg from testcases) just return that
	// one.
	if kubeClient != nil {
		return kubeClient, nil
	}

	var err error
	var config *rest.Config

	// Otherwise try to create a kubeClient from a given kubeConfig
	if kubeConfig != "" {
		// uses the current context in kubeConfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
		if err != nil {
			return nil, logging.Errorf("getK8sClient: failed to get context for the kubeConfig %v, refer Multus README.md for the usage guide: %v", kubeConfig, err)
		}
	} else if os.Getenv("KUBERNETES_SERVICE_HOST") != "" && os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		// Try in-cluster config where multus might be running in a kubernetes pod
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, logging.Errorf("createK8sClient: failed to get context for in-cluster kube config, refer Multus README.md for the usage guide: %v", err)
		}
	} else {
		// No kubernetes config; assume we shouldn't talk to Kube at all
		return nil, nil
	}

	// Specify that we use gRPC
	config.AcceptContentTypes = "application/vnd.kubernetes.protobuf,application/json"
	config.ContentType = "application/vnd.kubernetes.protobuf"

	// Create a new clientset (Interface)
	return kubernetes.NewForConfig(config)
}

func GetPod(args *skel.CmdArgs,
			kubeClient kubernetes.Interface,
			kubeConfig string) (*v1.Pod, kubernetes.Interface, error) {
	var err error

	logging.Verbosef("GetPod: ENTER - %v, %v, %v", args, kubeClient, kubeConfig)

	// Get k8sArgs
	k8sArgs, err := getK8sArgs(args)
	if err != nil {
		logging.Errorf("GetPod: Err in getting k8s args: %v", err)
		return nil, kubeClient, err
	}

	// Get kubeClient. If passed in, GetK8sClient() will just return it back.
	kubeClient, err = getK8sClient(kubeClient, kubeConfig)
	if err != nil {
		logging.Errorf("GetPod: Err in getting kubeClient: %v", err)
		return nil, kubeClient, err
	}

	if kubeClient == nil {
		return nil, nil, logging.Errorf("GetPod: No kubeClient: %v", err)
	}

	// Get the pod info. If cannot get it, we use cached delegates
	//pod, err := kubeClient.GetPod(string(k8sArgs.K8S_POD_NAMESPACE), string(k8sArgs.K8S_POD_NAME))
	pod, err := kubeClient.CoreV1().Pods(string(k8sArgs.K8S_POD_NAMESPACE)).Get(context.TODO(), string(k8sArgs.K8S_POD_NAME), metav1.GetOptions{})

	if err != nil {
		logging.Debugf("GetPod: Err in loading K8s cluster default network from pod annotation: %v, use cached delegates", err)
		return nil, kubeClient, err
	}

	logging.Verbosef("pod.Annotations: %v", pod.Annotations)

	return pod, kubeClient, err
}

func WritePodAnnotation(kubeClient kubernetes.Interface, pod *v1.Pod) (*v1.Pod, error) {
	var err error

	if kubeClient == nil {
		return pod, logging.Errorf("WritePodAnnotation: No kubeClient: %v", err)
	}
	if pod == nil {
		return pod, logging.Errorf("WritePodAnnotation: No pod: %v", err)
	}

	// Keep original pod info for log message in case of failure
	origPod := pod
	// Update the pod
	pod = pod.DeepCopy()
	if resultErr := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err != nil {
			// Re-get the pod unless it's the first attempt to update
			pod, err = kubeClient.CoreV1().Pods(pod.Namespace).Get(context.TODO(), pod.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}

		pod, err = kubeClient.CoreV1().Pods(pod.Namespace).UpdateStatus(context.TODO(), pod, metav1.UpdateOptions{})
		return err
	}); resultErr != nil {
		return nil, logging.Errorf("status update failed for pod %s/%s: %v", origPod.Namespace, origPod.Name, resultErr)
	}
	return pod, nil
}
