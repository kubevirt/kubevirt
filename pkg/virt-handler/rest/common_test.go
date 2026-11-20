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

package rest

import (
	"fmt"
	"net/http"

	"github.com/emicklei/go-restful/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
)

// errorStore is a cache.Store whose lookups always fail. Only GetByKey is
// implemented, the embedded nil Store panics on any other call.
type errorStore struct {
	cache.Store
	err error
}

func (s *errorStore) GetByKey(string) (any, bool, error) {
	return nil, false, s.err
}

var _ = Describe("getVMI", func() {
	newRequestFor := func(namespace, name string) *restful.Request {
		httpRequest, err := http.NewRequest(http.MethodGet, "/", nil)
		Expect(err).ToNot(HaveOccurred())
		request := restful.NewRequest(httpRequest)
		params := request.PathParameters()
		params["namespace"] = namespace
		params["name"] = name
		return request
	}

	It("should return the VMI stored under namespace/name", func() {
		vmiStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "testvmi"},
		}
		Expect(vmiStore.Add(vmi)).To(Succeed())

		found, code, err := getVMI(newRequestFor("default", "testvmi"), vmiStore)

		Expect(err).ToNot(HaveOccurred())
		Expect(code).To(Equal(0))
		Expect(found).To(Equal(vmi))
	})

	It("should report not found for a VMI that is not in the store", func() {
		vmiStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

		found, code, err := getVMI(newRequestFor("default", "missing"), vmiStore)

		Expect(err).To(MatchError(ContainSubstring("default/missing does not exist")))
		Expect(code).To(Equal(http.StatusNotFound))
		Expect(found).To(BeNil())
	})

	It("should report an internal error when the store lookup fails", func() {
		vmiStore := &errorStore{err: fmt.Errorf("store is broken")}

		found, code, err := getVMI(newRequestFor("default", "testvmi"), vmiStore)

		Expect(err).To(MatchError(ContainSubstring("store is broken")))
		Expect(code).To(Equal(http.StatusInternalServerError))
		Expect(found).To(BeNil())
	})

	It("should not confuse VMIs with the same name in different namespaces", func() {
		vmiStore := cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)
		other := &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Namespace: "other", Name: "testvmi"},
		}
		Expect(vmiStore.Add(other)).To(Succeed())

		_, code, err := getVMI(newRequestFor("default", "testvmi"), vmiStore)

		Expect(err).To(HaveOccurred())
		Expect(code).To(Equal(http.StatusNotFound))
	})
})
