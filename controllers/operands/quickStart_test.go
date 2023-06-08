package operands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"k8s.io/client-go/tools/reference"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
)

var _ = Describe("QuickStart tests", func() {

	schemeForTest := commontestutils.GetScheme()

	var (
		logger            = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("quickstart_test")
		testFilesLocation = getTestFilesLocation() + "/quickstarts"
		hco               = commontestutils.NewHco()
	)

	Context("test checkCrdExists", func() {
		It("should return not-stop-processing if not exists", func() {
			cli := commontestutils.InitClient([]client.Object{})

			err := checkCrdExists(context.TODO(), cli, logger)
			Expect(err).Should(HaveOccurred())
			Expect(errors.Unwrap(err)).ToNot(HaveOccurred())
		})

		It("should return true if CRD exists, with no error", func() {
			cli := commontestutils.InitClient([]client.Object{qsCrd})

			Expect(checkCrdExists(context.TODO(), cli, logger)).To(Succeed())
		})
	})

	Context("test getQuickStartHandlers", func() {
		It("should use env var to override the yaml locations", func() {
			// create temp folder for the test
			dir := path.Join(os.TempDir(), fmt.Sprint(time.Now().UTC().Unix()))
			_ = os.Setenv(quickStartManifestLocationVarName, dir)
			By("CRD is not deployed", func() {
				cli := commontestutils.InitClient([]client.Object{})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			By("folder not exists", func() {
				cli := commontestutils.InitClient([]client.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			Expect(os.Mkdir(dir, 0744)).To(Succeed())
			defer os.RemoveAll(dir)

			By("folder is empty", func() {
				cli := commontestutils.InitClient([]client.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			nonYaml, err := os.OpenFile(path.Join(dir, "for_test.txt"), os.O_CREATE|os.O_WRONLY, 0644)
			Expect(err).ToNot(HaveOccurred())
			defer os.Remove(nonYaml.Name())

			_, err = fmt.Fprintln(nonYaml, `some text`)
			Expect(err).ToNot(HaveOccurred())
			_ = nonYaml.Close()

			By("no yaml files", func() {
				cli := commontestutils.InitClient([]client.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			Expect(commontestutils.CopyFile(path.Join(dir, "quickStart.yaml"), path.Join(testFilesLocation, "quickstart.yaml"))).To(Succeed())

			By("yaml file exists", func() {
				cli := commontestutils.InitClient([]client.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
				Expect(quickstartNames).To(ContainElements("test-quick-start"))
			})
		})

		It("should return error if quickstart path is not a directory", func() {
			filePath := "/testFiles/quickstarts/quickstart.yaml"
			const currentDir = "/controllers/operands"
			wd, _ := os.Getwd()
			if !strings.HasSuffix(wd, currentDir) {
				filePath = wd + currentDir + filePath
			} else {
				filePath = wd + filePath
			}

			// quickstart directory path of a file
			_ = os.Setenv(quickStartManifestLocationVarName, filePath)
			By("check that getQuickStartHandlers returns error", func() {
				cli := commontestutils.InitClient([]client.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)

				Expect(err).Should(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})
		})
	})

	Context("test quickStartHandler", func() {

		var exists *consolev1.ConsoleQuickStart = nil
		It("should create the ConsoleQuickStart resource if not exists", func() {
			_ = os.Setenv(quickStartManifestLocationVarName, testFilesLocation)

			cli := commontestutils.InitClient([]client.Object{qsCrd})
			handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(quickstartNames).To(ContainElement("test-quick-start"))

			{ // for the next test
				handler, ok := handlers[0].(*genericOperand)
				Expect(ok).To(BeTrue())
				hooks, ok := handler.hooks.(*qsHooks)
				Expect(ok).To(BeTrue())
				exists = hooks.required.DeepCopy()
			}

			hco := commontestutils.NewHco()
			By("apply the quickstart CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				quickstartObjects := &consolev1.ConsoleQuickStartList{}
				Expect(cli.List(context.TODO(), quickstartObjects)).To(Succeed())
				Expect(quickstartObjects.Items).To(HaveLen(1))
				Expect(quickstartObjects.Items[0].Name).Should(Equal("test-quick-start"))
			})
		})

		It("should update the ConsoleQuickStart resource if not not equal to the expected one", func() {

			Expect(exists).ToNot(BeNil(), "Must run the previous test first")
			exists.Spec.DurationMinutes = exists.Spec.DurationMinutes * 2

			_ = os.Setenv(quickStartManifestLocationVarName, testFilesLocation)

			cli := commontestutils.InitClient([]client.Object{qsCrd, exists})
			handlers, err := getQuickStartHandlers(logger, cli, schemeForTest, hco)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))
			Expect(quickstartNames).To(ContainElement("test-quick-start"))

			hco := commontestutils.NewHco()
			By("apply the quickstart CRs", func() {
				req := commontestutils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				quickstartObjects := &consolev1.ConsoleQuickStartList{}
				Expect(cli.List(context.TODO(), quickstartObjects)).To(Succeed())
				Expect(quickstartObjects.Items).To(HaveLen(1))
				Expect(quickstartObjects.Items[0].Name).Should(Equal("test-quick-start"))
				// check that the existing object was reconciled
				Expect(quickstartObjects.Items[0].Spec.DurationMinutes).Should(Equal(20))

				// ObjectReference should have been updated
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRefOutdated, err := reference.GetReference(schemeForTest, exists)
				Expect(err).ToNot(HaveOccurred())
				objectRefFound, err := reference.GetReference(schemeForTest, &quickstartObjects.Items[0])
				Expect(err).ToNot(HaveOccurred())
				Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
			})
		})
	})
})
