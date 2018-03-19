/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2018 Red Hat, Inc.
 *
 */

package kubecli

import (
	"fmt"

	flag "github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var (
	kubeconfig string
	master     string
	server     string
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "Deprecated: use server instead")
	flag.StringVarP(&server, "server", "s", "", "The address and port of the Kubernetes API server")
}

func GetKubevirtClientFromConfig(master string, kubeconfig string) (KubevirtClient, error) {
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		return nil, err
	}

	config.GroupVersion = &v1.GroupVersion
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	coreClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &kubevirt{master, kubeconfig, restClient, coreClient}, nil
}

func GetKubevirtClientFromFlags(flags *flag.FlagSet) (KubevirtClient, error) {
	flagServer, _ := flags.GetString("server")
	flagConfig, _ := flags.GetString("kubeconfig")
	return GetKubevirtClientFromConfig(flagServer, flagConfig)
}

// the "master" command line flag is deprecated. Once it is removed, this entire
// function should be deleted (just use server).
func getServer() (string, error) {
	if master != "" {
		if server == "" {
			return master, nil
		} else {
			return "", fmt.Errorf("'master' command line flag is deprecated, use 'server' instead.")
		}
	}
	return server, nil
}

func GetKubevirtClient() (KubevirtClient, error) {
	kubeServer, err := getServer()
	if err != nil {
		return nil, err
	}
	return GetKubevirtClientFromConfig(kubeServer, kubeconfig)
}

func GetKubevirtClientConfig() (*rest.Config, error) {
	kubeServer, err := getServer()
	if err != nil {
		return nil, err
	}
	config, err := clientcmd.BuildConfigFromFlags(kubeServer, kubeconfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}
