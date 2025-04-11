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
 * Copyright The KubeVirt Authors.
 *
 */

package tls

import (
	"crypto/x509"
	"fmt"
	"sync"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/cert"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
)

type ClientCAManager interface {
	GetCurrent() (*x509.CertPool, error)
	GetCurrentRaw() ([]byte, error)
}

type manager struct {
	store        cache.Store
	lock         *sync.Mutex
	lastRevision string
	namespace    string
	name         string
	secretKey    string

	lastPool *x509.CertPool
	lastRaw  []byte
}

func NewKubernetesClientCAManager(configMapCache cache.Store) ClientCAManager {
	return &manager{
		store:        configMapCache,
		lock:         &sync.Mutex{},
		namespace:    metav1.NamespaceSystem,
		name:         util.ExtensionAPIServerAuthenticationConfigMap,
		secretKey:    util.RequestHeaderClientCAFileKey,
		lastRevision: "-1",
	}
}

func NewCAManager(configMapCache cache.Store, namespace string, configMapName string) ClientCAManager {
	return &manager{
		store:        configMapCache,
		lock:         &sync.Mutex{},
		namespace:    namespace,
		name:         configMapName,
		secretKey:    components.CABundleKey,
		lastRevision: "-1",
	}
}

func (m *manager) GetCurrentRaw() ([]byte, error) {
	_, err := m.GetCurrent()
	if err != nil {
		return nil, err
	}
	return m.lastRaw, nil
}

func (m *manager) GetCurrent() (*x509.CertPool, error) {
	m.lock.Lock()
	defer m.lock.Unlock()

	obj, exists, err := m.store.GetByKey(m.namespace + "/" + m.name)

	if err != nil {
		return nil, err
	} else if !exists {
		if m.lastPool != nil {
			return m.lastPool, nil
		}

		return nil, fmt.Errorf("configmap %s not found. Unable to detect request header CA", m.name)
	}

	configMap := obj.(*k8sv1.ConfigMap)

	// no change detected.
	if m.lastRevision == configMap.ResourceVersion {
		return m.lastPool, nil
	}
	log.DefaultLogger().Infof("CA update in configmap %s/%s detected. Updating from resource version %v to %v", m.namespace, m.name, m.lastRevision, configMap.ResourceVersion)

	requestHeaderClientCA, ok := configMap.Data[m.secretKey]
	if !ok {
		return nil, fmt.Errorf("requestheader-client-ca-file not found in configmap %s/%s", m.namespace, m.name)
	}

	certs, err := cert.ParseCertsPEM([]byte(requestHeaderClientCA))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}

	m.lastRevision = configMap.ResourceVersion
	m.lastPool = pool
	m.lastRaw = []byte(requestHeaderClientCA)

	return pool, nil
}
