package integration_test

import (
	"context"
	goerrors "errors"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("JSON patch", func() {
	// virt-api's isJSONPatchApplyError recognizes a JSON patch whose test
	// operation fails by this error alone and returns a retryable conflict for
	// it. If this test fails, the API server reports the failure differently
	// and isJSONPatchApplyError in pkg/virt-api/rest must change with it.
	DescribeTable("should report a failed test operation as a generic 422 Invalid", func(ctx SpecContext, patchVM func(context.Context, string, string, []byte) error) {
		ns := f.CreateNamespace(ctx)

		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyHalted),
		)
		vm, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		patchBytes, err := patch.New(patch.WithTest("/spec/runStrategy", virtv1.RunStrategyAlways)).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())

		err = patchVM(ctx, ns, vm.Name, patchBytes)

		var statusErr *errors.StatusError
		Expect(goerrors.As(err, &statusErr)).To(BeTrue(), "expected a status error, got %v", err)
		Expect(statusErr.ErrStatus.Code).To(Equal(int32(http.StatusUnprocessableEntity)))
		Expect(statusErr.ErrStatus.Reason).To(Equal(metav1.StatusReasonInvalid))
		Expect(statusErr.ErrStatus.Message).To(Equal("the server rejected our request due to an error in our request"))
	},
		Entry("on the VM", func(ctx context.Context, ns, name string, patchBytes []byte) error {
			_, err := f.VirtClient().VirtualMachine(ns).Patch(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			return err
		}),
		Entry("on the VM status", func(ctx context.Context, ns, name string, patchBytes []byte) error {
			_, err := f.VirtClient().VirtualMachine(ns).PatchStatus(ctx, name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			return err
		}),
	)
})
