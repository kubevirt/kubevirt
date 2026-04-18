/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package multus_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/multus"
)

var _ = Describe("NetAttachDefNamespacedName", func() {
	It("should return vmi namespace when namespace is implicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		nadNamespacedName := multus.NetAttachDefNamespacedName(vmi.Namespace, "testnet")
		Expect(nadNamespacedName).To(Equal(types.NamespacedName{Namespace: "testns", Name: "testnet"}))
	})

	It("should return namespace from networkName when namespace is explicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		nadNamespacedName := multus.NetAttachDefNamespacedName(vmi.Namespace, "otherns/testnet")
		Expect(nadNamespacedName).To(Equal(types.NamespacedName{Namespace: "otherns", Name: "testnet"}))
	})
})
