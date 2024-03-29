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
 * Copyright The KubeVirt Authors
 *
 */

package virt_controller

import (
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"kubevirt.io/client-go/kubecli"
)

type ControllerKey string

const (
	KeyVM                 ControllerKey = "vm"
	KeyVMI                ControllerKey = "vmi"
	KeyVMPool             ControllerKey = "vmpool"
	KeyVMIReplicaset      ControllerKey = "vmireplicaset"
	KeyMigration          ControllerKey = "migration"
	KeyVMClone            ControllerKey = "vmclone"
	KeyVMSnapshot         ControllerKey = "vmsnapshot"
	KeyVMRestore          ControllerKey = "vmrestore"
	KeyVMExport           ControllerKey = "vmexport"
	KeyExportService      ControllerKey = "vmexportservice"
	KeyVMSnapshotContent  ControllerKey = "vmsnapshotcontent"
	KeyIT                 ControllerKey = "instancetype"
	KeyClusterIT          ControllerKey = "clusterinstancetype"
	KeyPref               ControllerKey = "preference"
	KeyClusterPref        ControllerKey = "clusterpreference"
	KeyCR                 ControllerKey = "cr"
	KeyCRD                ControllerKey = "crd"
	KeyRQ                 ControllerKey = "rq"
	KeyPod                ControllerKey = "pod"
	KeyPDB                ControllerKey = "pdb"
	KeyMigrationPolicy    ControllerKey = "migrationpolicy"
	KeyKubeVirt           ControllerKey = "kubevirt"
	KeyKubeVirtPod        ControllerKey = "kubevirtpod"
	KeyDV                 ControllerKey = "dv"
	KeyDS                 ControllerKey = "ds"
	KeyNS                 ControllerKey = "ns"
	KeyNode               ControllerKey = "node"
	KeySC                 ControllerKey = "sc"
	KeyPVC                ControllerKey = "pvc"
	KeyConfigMap          ControllerKey = "configmap"
	KeyRouteConfigMap     ControllerKey = "routeconfigmap"
	KeySecret             ControllerKey = "secret"
	KeyControllerRevision ControllerKey = "controllerrevision"
	KeyIngress            ControllerKey = "ingress"
	KeyRoute              ControllerKey = "route"
	KeyCDI                ControllerKey = "cdi"
	KeyCDIConfig          ControllerKey = "cdiconfig"
)

type ControllerInterface interface {
	Execute() bool
	Run(threadiness int, stopCh <-chan struct{}) error
	WaitForCacheSync(stopCh <-chan struct{}) bool

	Queue() workqueue.RateLimitingInterface
	SetQueue(workqueue.RateLimitingInterface)

	Clientset() kubecli.KubevirtClient
	SetClientset(kubecli.KubevirtClient)

	Informers() map[ControllerKey]cache.SharedIndexInformer
	SetInformers(map[ControllerKey]cache.SharedIndexInformer)
	Informer(ControllerKey) cache.SharedIndexInformer
	SetInformer(ControllerKey, cache.SharedIndexInformer)

	Stores() map[ControllerKey]cache.Store
	SetStores(map[ControllerKey]cache.Store)
	Store(ControllerKey) cache.Store
	SetStore(ControllerKey, cache.Store)

	Indexers() map[ControllerKey]cache.Indexer
	SetIndexers(map[ControllerKey]cache.Indexer)
	Indexer(ControllerKey) cache.Indexer
	SetIndexer(ControllerKey, cache.Indexer)
}

type Controller struct {
	queue     workqueue.RateLimitingInterface
	clientset kubecli.KubevirtClient
	informers map[ControllerKey]cache.SharedIndexInformer
	stores    map[ControllerKey]cache.Store
	indexers  map[ControllerKey]cache.Indexer
}

func NewController(queue workqueue.RateLimitingInterface, clientset kubecli.KubevirtClient) *Controller {
	return &Controller{
		queue:     queue,
		clientset: clientset,
		informers: make(map[ControllerKey]cache.SharedIndexInformer),
		stores:    make(map[ControllerKey]cache.Store),
		indexers:  make(map[ControllerKey]cache.Indexer),
	}
}

func (c *Controller) WaitForCacheSync(stopCh <-chan struct{}) bool {
	var synceds []cache.InformerSynced
	for _, value := range c.Informers() {
		synceds = append(synceds, value.HasSynced)
	}
	return cache.WaitForCacheSync(stopCh, synceds...)
}

func (c *Controller) Queue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c *Controller) SetQueue(queue workqueue.RateLimitingInterface) {
	c.queue = queue
}

func (c *Controller) Clientset() kubecli.KubevirtClient {
	return c.clientset
}

func (c *Controller) SetClientset(clientset kubecli.KubevirtClient) {
	c.clientset = clientset
}

func (c *Controller) Informers() map[ControllerKey]cache.SharedIndexInformer {
	return c.informers
}

func (c *Controller) SetInformers(informers map[ControllerKey]cache.SharedIndexInformer) {
	c.informers = informers
}

func (c *Controller) Informer(key ControllerKey) cache.SharedIndexInformer {
	return c.informers[key]
}

func (c *Controller) SetInformer(key ControllerKey, informer cache.SharedIndexInformer) {
	c.informers[key] = informer
}

func (c *Controller) Stores() map[ControllerKey]cache.Store {
	return c.stores
}

func (c *Controller) SetStores(stores map[ControllerKey]cache.Store) {
	c.stores = stores
}

func (c *Controller) Store(key ControllerKey) cache.Store {
	return c.stores[key]
}

func (c *Controller) SetStore(key ControllerKey, store cache.Store) {
	c.stores[key] = store
}

func (c *Controller) Indexers() map[ControllerKey]cache.Indexer {
	return c.indexers
}

func (c *Controller) SetIndexers(indexers map[ControllerKey]cache.Indexer) {
	c.indexers = indexers
}

func (c *Controller) Indexer(key ControllerKey) cache.Indexer {
	return c.indexers[key]
}

func (c *Controller) SetIndexer(key ControllerKey, indexer cache.Indexer) {
	c.indexers[key] = indexer
}
