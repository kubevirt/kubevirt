/*
 * This file is part of the CDI project
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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package apiserver

import (
	"crypto/x509"
	"encoding/json"
	"sync"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"

	"kubevirt.io/containerized-data-importer/pkg/common"
)

const (
	configMapName = "extension-apiserver-authentication"
)

// AuthConfig contains extension-apiserver-authentication data
type AuthConfig struct {
	AllowedNames       []string
	UserHeaders        []string
	GroupHeaders       []string
	ExtraPrefixHeaders []string

	ClientCABytes              []byte
	RequestheaderClientCABytes []byte

	CertPool *x509.CertPool
}

// AuthConfigWatcher is the interface of authConfigWatcher
type AuthConfigWatcher interface {
	GetAuthConfig() *AuthConfig
}

type authConfigWatcher struct {
	// keep this around for tests
	informer cache.SharedIndexInformer

	config *AuthConfig
	mutex  sync.RWMutex
}

// NewAuthConfigWatcher crates a new authConfigWatcher
func NewAuthConfigWatcher(client kubernetes.Interface, stopCh <-chan struct{}) AuthConfigWatcher {
	informerFactory := informers.NewFilteredSharedInformerFactory(client,
		common.DefaultResyncPeriod,
		metav1.NamespaceSystem,
		func(options *metav1.ListOptions) {
			options.FieldSelector = "metadata.name=" + configMapName
		},
	)

	configMapInformer := informerFactory.Core().V1().ConfigMaps().Informer()

	acw := &authConfigWatcher{
		informer: configMapInformer,
	}

	configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			klog.V(3).Infof("configMapInformer add callback: %+v", obj)
			acw.updateConfig(obj.(*corev1.ConfigMap))
		},
		UpdateFunc: func(_, obj interface{}) {
			klog.V(3).Infof("configMapInformer update callback: %+v", obj)
			acw.updateConfig(obj.(*corev1.ConfigMap))
		},
		DeleteFunc: func(obj interface{}) {
			cm := obj.(*corev1.ConfigMap)
			klog.Errorf("Configmap %s deleted", cm.Name)
		},
	})

	go informerFactory.Start(stopCh)

	klog.V(3).Infoln("Waiting for cache sync")
	cache.WaitForCacheSync(stopCh, configMapInformer.HasSynced)
	klog.V(3).Infoln("Cache sync complete")

	return acw
}

func (acw *authConfigWatcher) GetAuthConfig() *AuthConfig {
	acw.mutex.RLock()
	defer acw.mutex.RUnlock()
	return acw.config
}

func deserializeStringSlice(in string) []string {
	if len(in) == 0 {
		return nil
	}
	var ret []string
	if err := json.Unmarshal([]byte(in), &ret); err != nil {
		klog.Errorf("Error decoding %q", in)
		return nil
	}
	return ret
}

func (acw *authConfigWatcher) updateConfig(cm *corev1.ConfigMap) {
	newConfig := &AuthConfig{}
	pool := x509.NewCertPool()

	s, ok := cm.Data["client-ca-file"]
	if ok {
		newConfig.ClientCABytes = []byte(s)
		// TODO don't think we've done enough testing to support this path (direct access to the apiserver)
		// Have to write code to get user/groups/etc from cert
		/*
			if ok = pool.AppendCertsFromPEM(newConfig.ClientCABytes); !ok {
				klog.Errorf("Error adding ClientCABytes to client cert pool")
			}
		*/
	}

	s, ok = cm.Data["requestheader-client-ca-file"]
	if ok {
		newConfig.RequestheaderClientCABytes = []byte(s)
		if ok = pool.AppendCertsFromPEM(newConfig.RequestheaderClientCABytes); !ok {
			klog.Errorf("Error adding RequestheaderClientCABytes to client cert pool")
		}
	}

	newConfig.CertPool = pool

	newConfig.AllowedNames = deserializeStringSlice(cm.Data["requestheader-allowed-names"])
	newConfig.UserHeaders = deserializeStringSlice(cm.Data["requestheader-username-headers"])
	newConfig.GroupHeaders = deserializeStringSlice(cm.Data["requestheader-group-headers"])
	newConfig.ExtraPrefixHeaders = deserializeStringSlice(cm.Data["requestheader-extra-headers-prefix"])

	acw.mutex.Lock()
	defer acw.mutex.Unlock()
	acw.config = newConfig
}
