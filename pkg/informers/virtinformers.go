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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package informers

import (
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/kubecli"

	kubev1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

type newSharedInformer func() cache.SharedIndexInformer

type KubeInformerFactory interface {
	// Starts any informers that have not been started yet
	// This function is thread safe and idempotent
	Start(stopCh <-chan struct{})

	// Watches for vm objects
	VM() cache.SharedIndexInformer
	// Watches for migration objects
	Migration() cache.SharedIndexInformer
	// Watches for pods related only to kubevirt
	KubeVirtPod() cache.SharedIndexInformer
}

type kubeInformerFactory struct {
	restClient    *rest.RESTClient
	clientSet     *kubernetes.Clientset
	lock          sync.Mutex
	defaultResync time.Duration

	informers        map[string]cache.SharedIndexInformer
	startedInformers map[string]bool
}

func NewKubeInformerFactory(restClient *rest.RESTClient, clientSet *kubernetes.Clientset) KubeInformerFactory {
	return &kubeInformerFactory{
		restClient:       restClient,
		clientSet:        clientSet,
		defaultResync:    0,
		informers:        make(map[string]cache.SharedIndexInformer),
		startedInformers: make(map[string]bool),
	}
}

// Start can be called from multiple controllers in different go routines safely.
// Only informers that have not started are triggered by this function.
// Multiple calls to this function are idempotent.
func (f *kubeInformerFactory) Start(stopCh <-chan struct{}) {
	f.lock.Lock()
	defer f.lock.Unlock()

	for name, informer := range f.informers {
		if f.startedInformers[name] {
			// skip informers that have already started.
			continue
		}
		logging.DefaultLogger().Info().Msgf("STARTING informer %s", name)
		go informer.Run(stopCh)
		f.startedInformers[name] = true
	}
}

// internal function used to retrieve an already created informer
// or create a new informer if one does not already exist.
// Thread safe
func (f *kubeInformerFactory) getInformer(key string, newFunc newSharedInformer) cache.SharedIndexInformer {
	f.lock.Lock()
	defer f.lock.Unlock()

	informer, exists := f.informers[key]
	if exists {
		return informer
	}
	informer = newFunc()
	f.informers[key] = informer

	return informer
}

func (f *kubeInformerFactory) VM() cache.SharedIndexInformer {
	return f.getInformer("vmInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "vms", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.VM{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) Migration() cache.SharedIndexInformer {
	return f.getInformer("migrationInformer", func() cache.SharedIndexInformer {
		lw := cache.NewListWatchFromClient(f.restClient, "migrations", k8sv1.NamespaceAll, fields.Everything())
		return cache.NewSharedIndexInformer(lw, &kubev1.Migration{}, f.defaultResync, cache.Indexers{})
	})
}

func (f *kubeInformerFactory) KubeVirtPod() cache.SharedIndexInformer {
	return f.getInformer("kubeVirtPodInformer", func() cache.SharedIndexInformer {
		// Watch all pods with the kubevirt app label
		labelSelector, err := labels.Parse(kubev1.AppLabel)
		if err != nil {
			panic(err)
		}

		lw := kubecli.NewListWatchFromClient(f.clientSet.CoreV1().RESTClient(), "pods", k8sv1.NamespaceAll, fields.Everything(), labelSelector)
		return cache.NewSharedIndexInformer(lw, &k8sv1.Pod{}, f.defaultResync, cache.Indexers{})
	})
}
