package libreplicaset

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	autov1 "k8s.io/api/autoscaling/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/util"
)

func DoScaleWithScaleSubresource(virtClient kubecli.KubevirtClient, name string, scale int32) {
	// Status updates can conflict with our desire to change the spec
	By(fmt.Sprintf("Scaling to %d", scale))
	var s *autov1.Scale
	err := tests.RetryIfModified(func() error {
		s, err := virtClient.ReplicaSet(util.NamespaceTestDefault).GetScale(name, v12.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		s.Spec.Replicas = scale
		s, err = virtClient.ReplicaSet(util.NamespaceTestDefault).UpdateScale(name, s)
		return err
	})

	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	By("Checking the number of replicas")
	EventuallyWithOffset(1, func() int32 {
		s, err = virtClient.ReplicaSet(util.NamespaceTestDefault).GetScale(name, v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return s.Status.Replicas
	}, 90*time.Second, time.Second).Should(Equal(scale))

	vmis, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).List(&v12.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, tests.NotDeleted(vmis)).To(HaveLen(int(scale)))
}
