package operands

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutils "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("Test kubeVirtCmHandler", func() {
	var (
		hco            *hcov1beta1.HyperConverged
		req            *common.HcoRequest
		expectedEvents = []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Killing",
				Msg:       fmt.Sprintf("Removed ConfigMap %s", kvCmName),
			},
		}
		emitter = commonTestUtils.NewEventEmitterMock()
	)

	BeforeEach(func() {
		hco = commonTestUtils.NewHco()
		req = commonTestUtils.NewReq(hco)

		req.SetUpgradeMode(false)

		emitter.Reset()
	})

	It("should do nothing if the configMap does not exist", func() {
		c := commonTestUtils.InitClient([]runtime.Object{hco})
		handler := newKubeVirtCmHandler(c, emitter)

		res := handler.ensure(req)

		Expect(res.Err).ToNot(HaveOccurred())
		Expect(res.UpgradeDone).To(BeTrue())

		Expect(emitter.CheckEvents(expectedEvents)).To(BeFalse())
	})

	It("should remove the configMap if it exists", func() {
		cm := newKvConfigMap()

		c := commonTestUtils.InitClient([]runtime.Object{hco, cm})
		handler := newKubeVirtCmHandler(c, emitter)

		res := handler.ensure(req)

		Expect(res.Err).ToNot(HaveOccurred())
		Expect(res.UpgradeDone).To(BeTrue())

		Expect(emitter.CheckEvents(expectedEvents)).To(BeTrue())

		resCm := newEmptyKvConfigMap()

		err := hcoutils.GetRuntimeObject(context.TODO(), c, resCm, req.Logger)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsNotFound(err)).To(BeTrue())
	})

	It("should not remove the configMap during upgrade", func() {
		req.SetUpgradeMode(true)

		cm := newKvConfigMap()

		c := commonTestUtils.InitClient([]runtime.Object{hco, cm})
		handler := newKubeVirtCmHandler(c, emitter)

		res := handler.ensure(req)

		Expect(res.Err).ToNot(HaveOccurred())
		Expect(res.UpgradeDone).To(BeTrue())

		Expect(emitter.CheckEvents(expectedEvents)).To(BeFalse())

		resCm := newEmptyKvConfigMap()

		err := hcoutils.GetRuntimeObject(context.TODO(), c, resCm, req.Logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(resCm.Data).ToNot(BeEmpty())
		Expect(resCm.Data["fakeKey"]).Should(Equal("fakeValue"))
	})

	It("should return error if failed to read the configMap", func() {
		cm := newKvConfigMap()
		c := commonTestUtils.InitClient([]runtime.Object{hco, cm})
		fakeError := fmt.Errorf("fake get error")
		c.InitiateGetErrors(func(key client.ObjectKey) error {
			if key.Name == kvCmName {
				return fakeError
			}
			return nil
		})

		handler := newKubeVirtCmHandler(c, emitter)

		res := handler.ensure(req)

		Expect(res.Err).To(HaveOccurred())
		Expect(res.Err).To(MatchError(fakeError))
		Expect(res.UpgradeDone).To(BeTrue())

		Expect(emitter.CheckEvents(expectedEvents)).To(BeFalse())

		resCm := newEmptyKvConfigMap()

		c.InitiateGetErrors(nil) // cancel fake error
		err := hcoutils.GetRuntimeObject(context.TODO(), c, resCm, req.Logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(resCm.Data).ToNot(BeEmpty())
		Expect(resCm.Data["fakeKey"]).Should(Equal("fakeValue"))
	})

	It("should return error if failed to delete the configMap", func() {
		cm := newKvConfigMap()
		c := commonTestUtils.InitClient([]runtime.Object{hco, cm})
		fakeError := fmt.Errorf("fake delete error")
		c.InitiateDeleteErrors(func(obj client.Object) error {
			if unstructured, ok := obj.(runtime.Unstructured); ok {
				kind := unstructured.GetObjectKind()
				if kind.GroupVersionKind().Kind == "ConfigMap" && obj.GetName() == kvCmName {
					return fakeError
				}
			}
			return nil
		})

		handler := newKubeVirtCmHandler(c, emitter)

		res := handler.ensure(req)

		Expect(res.Err).To(HaveOccurred())
		Expect(res.Err).To(MatchError(fakeError))
		Expect(res.UpgradeDone).To(BeTrue())

		Expect(emitter.CheckEvents(expectedEvents)).To(BeFalse())

		resCm := newEmptyKvConfigMap()

		err := hcoutils.GetRuntimeObject(context.TODO(), c, resCm, req.Logger)
		Expect(err).ToNot(HaveOccurred())
		Expect(resCm.Data).ToNot(BeEmpty())
		Expect(resCm.Data["fakeKey"]).Should(Equal("fakeValue"))
	})
})

func newEmptyKvConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kvCmName,
			Namespace: commonTestUtils.Namespace,
		},
	}
}

func newKvConfigMap() *corev1.ConfigMap {
	cm := newEmptyKvConfigMap()
	cm.Data = map[string]string{
		"fakeKey": "fakeValue",
	}

	return cm
}
