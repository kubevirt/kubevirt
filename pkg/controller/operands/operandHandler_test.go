package operands

import (
	"os"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	"github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

var _ = Describe("Test operandHandler", func() {
	Context("Test operandHandler", func() {
		testFileLocation := getTestFilesLocation()

		_ = os.Setenv("CONVERSION_CONTAINER", "just-a-value:version")
		_ = os.Setenv("VMWARE_CONTAINER", "just-a-value:version")

		It("should create all objects are created", func() {
			err := os.Setenv(manifestLocationVarName, testFileLocation)
			Expect(err).ToNot(HaveOccurred())
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, commonTestUtils.EventEmitterMock{})
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)

			err = handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			By("make sure the KV object created", func() {
				// Read back KV
				kvList := kubevirtv1.KubeVirtList{}
				err := cli.List(req.Ctx, &kvList)
				Expect(err).ToNot(HaveOccurred())
				Expect(kvList).ToNot(BeNil())
				Expect(kvList.Items).To(HaveLen(1))
				Expect(kvList.Items[0].Name).Should(Equal("kubevirt-kubevirt-hyperconverged"))
			})

			By("make sure the CNA object created", func() {
				// Read back KV
				cnaList := networkaddonsv1.NetworkAddonsConfigList{}
				err := cli.List(req.Ctx, &cnaList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnaList).ToNot(BeNil())
				Expect(cnaList.Items).To(HaveLen(1))
				Expect(cnaList.Items[0].Name).Should(Equal("cluster"))
			})

			By("make sure the CDI object created", func() {
				// Read back KV
				cdiList := cdiv1beta1.CDIList{}
				err := cli.List(req.Ctx, &cdiList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdiList).ToNot(BeNil())
				Expect(cdiList.Items).To(HaveLen(1))
				Expect(cdiList.Items[0].Name).Should(Equal("cdi-kubevirt-hyperconverged"))
			})

			By("make sure the VM-Import object created", func() {
				// Read back KV
				vmImportList := v1beta1.VMImportConfigList{}
				err := cli.List(req.Ctx, &vmImportList)
				Expect(err).ToNot(HaveOccurred())
				Expect(vmImportList).ToNot(BeNil())
				Expect(vmImportList.Items).To(HaveLen(1))
				Expect(vmImportList.Items[0].Name).Should(Equal("vmimport-kubevirt-hyperconverged"))
			})

			By("make sure the ConsoleQuickStart object created", func() {
				// Read back the ConsoleQuickStart
				qsList := consolev1.ConsoleQuickStartList{}
				err := cli.List(req.Ctx, &qsList)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsList).ToNot(BeNil())
				Expect(qsList.Items).To(HaveLen(1))
				Expect(qsList.Items[0].Name).Should(Equal("test-quick-start"))
			})
		})

		It("make sure the all objects are deleted", func() {
			err := os.Setenv(manifestLocationVarName, testFileLocation)
			Expect(err).ToNot(HaveOccurred())
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, commonTestUtils.EventEmitterMock{})
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err = handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			err = handler.EnsureDeleted(req)
			Expect(err).ToNot(HaveOccurred())

			By("check that KV is deleted", func() {
				// Read back KV
				kvList := kubevirtv1.KubeVirtList{}
				err = cli.List(req.Ctx, &kvList)
				Expect(err).ToNot(HaveOccurred())
				Expect(kvList).ToNot(BeNil())
				Expect(kvList.Items).To(BeEmpty())
			})

			By("make sure the CNA object deleted", func() {
				// Read back KV
				cnaList := networkaddonsv1.NetworkAddonsConfigList{}
				err := cli.List(req.Ctx, &cnaList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnaList).ToNot(BeNil())
				Expect(cnaList.Items).To(BeEmpty())
			})

			By("make sure the CDI object deleted", func() {
				// Read back KV
				cdiList := cdiv1beta1.CDIList{}
				err := cli.List(req.Ctx, &cdiList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdiList).ToNot(BeNil())
				Expect(cdiList.Items).To(BeEmpty())
			})

			By("make sure the VM-Import object deleted", func() {
				// Read back KV
				vmImportList := v1beta1.VMImportConfigList{}
				err := cli.List(req.Ctx, &vmImportList)
				Expect(err).ToNot(HaveOccurred())
				Expect(vmImportList).ToNot(BeNil())
				Expect(vmImportList.Items).To(BeEmpty())
			})

			By("check that ConsoleQuickStart is deleted", func() {
				// Read back the ConsoleQuickStart
				qsList := consolev1.ConsoleQuickStartList{}
				err = cli.List(req.Ctx, &qsList)
				Expect(err).ToNot(HaveOccurred())
				Expect(qsList).ToNot(BeNil())
				Expect(qsList.Items).To(BeEmpty())
			})
		})
	})
})
