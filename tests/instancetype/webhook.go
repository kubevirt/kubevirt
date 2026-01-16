//nolint:lll
package instancetype

import (
	"context"
	goerrors "errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute] Instancetype and Preferences admission", decorators.SigCompute, decorators.SigComputeInstancetype, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Instancetype validation", func() {
		It("[test_id:CNV-9082] should allow valid instancetype", func() {
			instancetype := builder.NewInstancetypeFromVMI(nil)
			_, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		DescribeTable("[test_id:CNV-9083] should reject invalid instancetype", func(instancetype instancetypev1beta1.VirtualMachineInstancetype) {
			_, err := virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(nil)).
				Create(context.Background(), &instancetype, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			var apiStatus errors.APIStatus
			Expect(goerrors.As(err, &apiStatus)).To(BeTrue(), "error should be type APIStatus")
			cause := apiStatus.Status().Details.Causes[0]
			Expect(cause.Type).To(Equal(metav1.CauseTypeFieldValueRequired))
		},
			Entry("without CPU defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				},
			}),
			Entry("without CPU.Guest defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{},
					Memory: instancetypev1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128M"),
					},
				},
			}),
			Entry("without Memory defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: 1,
					},
				},
			}),
			Entry("without Memory.Guest defined", instancetypev1beta1.VirtualMachineInstancetype{
				Spec: instancetypev1beta1.VirtualMachineInstancetypeSpec{
					CPU: instancetypev1beta1.CPUInstancetype{
						Guest: 1,
					},
					Memory: instancetypev1beta1.MemoryInstancetype{},
				},
			}),
		)
	})

	Context("Preference validation", func() {
		It("[test_id:CNV-9084] should allow valid preference", func() {
			preference := builder.NewPreference()
			_, err := virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
