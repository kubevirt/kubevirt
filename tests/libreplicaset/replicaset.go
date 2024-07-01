package libreplicaset

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	autov1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tests/testsuite"
)

func DoScaleWithScaleSubresource(virtClient kubecli.KubevirtClient, name string, scale int32) {
	// Status updates can conflict with our desire to change the spec
	By(fmt.Sprintf("Scaling to %d", scale))
	var s *autov1.Scale
	err := retryIfModified(func() error {
		s, err := virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).GetScale(context.Background(), name, v12.GetOptions{})
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		s.Spec.Replicas = scale
		s, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).UpdateScale(context.Background(), name, s)
		return err
	})

	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	By("Checking the number of replicas")
	EventuallyWithOffset(1, func() int32 {
		s, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).GetScale(context.Background(), name, v12.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		return s.Status.Replicas
	}, 90*time.Second, time.Second).Should(Equal(scale))

	vmis, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).List(context.Background(), v12.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, FilterNotDeletedVMIs(vmis)).To(HaveLen(int(scale)))
}

func FilterNotDeletedVMIs(vmis *v1.VirtualMachineInstanceList) []v1.VirtualMachineInstance {
	var notDeleted []v1.VirtualMachineInstance
	for _, vmi := range vmis.Items {
		if vmi.DeletionTimestamp == nil {
			notDeleted = append(notDeleted, vmi)
		}
	}
	return notDeleted
}

func retryIfModified(do func() error) (err error) {
	retries := 0
	for err = do(); errors.IsConflict(err); err = do() {
		if retries >= 10 {
			return fmt.Errorf("object seems to be permanently modified, failing after 10 retries: %v", err)
		}
		retries++
		log.DefaultLogger().Reason(err).Infof("Object got modified, will retry.")
	}
	return err
}
