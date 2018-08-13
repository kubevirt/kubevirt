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

package config

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/log"
)

var _ = Describe("ConfigMap", func() {

	log.Log.SetIOWriter(GinkgoWriter)

	var cmListWatch *cache.ListWatch
	var cmInformer cache.SharedIndexInformer
	var stopChan chan struct{}

	BeforeEach(func() {
		stopChan = make(chan struct{})
	})

	AfterEach(func() {
		close(stopChan)
	})

	It("Should return false if configmap is not present", func() {
		cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{})
		cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		go cmInformer.Run(stopChan)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
		clusterConfig := NewClusterConfig(cmInformer.GetStore())
		result, err := clusterConfig.IsUseEmulation()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Should return false if configmap doesn't have useEmulation set", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{},
		}
		cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
		cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		go cmInformer.Run(stopChan)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
		clusterConfig := NewClusterConfig(cmInformer.GetStore())
		result, err := clusterConfig.IsUseEmulation()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("Should return true if useEmulation = true", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{"debug.useEmulation": "true"},
		}
		cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
		cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		go cmInformer.Run(stopChan)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)
		clusterConfig := NewClusterConfig(cmInformer.GetStore())
		result, err := clusterConfig.IsUseEmulation()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("Should return IfNotPresent if configmap doesn't have imagePullPolicy set", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{},
		}
		cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
		cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		go cmInformer.Run(stopChan)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

		result, err := NewClusterConfig(cmInformer.GetStore()).GetImagePullPolicy()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(kubev1.PullIfNotPresent))
	})

	It("Should return Always if imagePullPolicy = Always", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{imagePullPolicyKey: "Always"},
		}
		cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
		cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		go cmInformer.Run(stopChan)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

		result, err := NewClusterConfig(cmInformer.GetStore()).GetImagePullPolicy()
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(kubev1.PullAlways))
	})

	It("Should return an error if imagePullPolicy is not valid", func() {
		cfgMap := kubev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "kube-system",
				Name:      "kubevirt-config",
			},
			Data: map[string]string{imagePullPolicyKey: "IHaveNoStrongFeelingsOneWayOrTheOther"},
		}
		cmListWatch = MakeFakeConfigMapWatcher([]kubev1.ConfigMap{cfgMap})
		cmInformer = cache.NewSharedIndexInformer(cmListWatch, &v1.VirtualMachineInstance{}, time.Second, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc})
		go cmInformer.Run(stopChan)
		cache.WaitForCacheSync(stopChan, cmInformer.HasSynced)

		_, err := NewClusterConfig(cmInformer.GetStore()).GetImagePullPolicy()
		Expect(err).To(HaveOccurred())
	})
})

func MakeFakeConfigMapWatcher(configMaps []kubev1.ConfigMap) *cache.ListWatch {
	cmListWatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return &kubev1.ConfigMapList{Items: configMaps}, nil
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			fakeWatch := watch.NewFake()
			for _, cfgMap := range configMaps {
				fakeWatch.Add(&cfgMap)
			}
			return watch.NewFake(), nil
		},
	}
	return cmListWatch
}
