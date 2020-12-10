package operands

import (
	"context"
	"fmt"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	"io"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"path"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"strings"
	"time"
)

const (
	pkgDirectory = "pkg/controller/operands"
	testFilesLoc = "testFiles"
)

var _ = Describe("QuickStart tests", func() {

	schemeForTest := commonTestUtils.GetScheme()

	var (
		logger            = logf.ZapLoggerTo(GinkgoWriter, true).WithName("quickstart_test")
		testFilesLocation string
	)

	wd, err := os.Getwd()
	Expect(err).ToNot(HaveOccurred())
	if strings.HasSuffix(wd, pkgDirectory) {
		testFilesLocation = testFilesLoc
	} else {
		testFilesLocation = path.Join(pkgDirectory, testFilesLoc)
	}

	qsCrd := &extv1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			Kind: "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: consoleQuickStartCrdName,
		},
	}

	Context("test checkCrdExists", func() {
		It("should return false if not exists, with no error", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{})

			supported, err := checkCrdExists(context.TODO(), cli, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(supported).To(BeFalse())
		})

		It("should return true if CRD exists, with no error", func() {
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd})

			supported, err := checkCrdExists(context.TODO(), cli, logger)
			Expect(err).ToNot(HaveOccurred())
			Expect(supported).To(BeTrue())
		})
	})

	Context("test getQuickStartHandlers", func() {
		// create temp folder for the test
		dir := path.Join(os.TempDir(), fmt.Sprint(time.Now().UTC().Unix()))
		_ = os.Setenv(manifestLocationVarName, dir)

		It("should use env var to override the yaml locations", func() {
			By("CRD is not deployed", func() {
				cli := commonTestUtils.InitClient([]runtime.Object{})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			By("folder not exists", func() {
				cli := commonTestUtils.InitClient([]runtime.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			err := os.Mkdir(dir, 0744)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			By("folder is empty", func() {
				cli := commonTestUtils.InitClient([]runtime.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)

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
				cli := commonTestUtils.InitClient([]runtime.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(BeEmpty())
			})

			err = copyFile(path.Join(dir, "quickStart.yaml"), path.Join(testFilesLocation, "quickstart.yaml"))
			Expect(err).ToNot(HaveOccurred())

			By("yaml file exists", func() {
				cli := commonTestUtils.InitClient([]runtime.Object{qsCrd})
				handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)

				Expect(err).ToNot(HaveOccurred())
				Expect(handlers).To(HaveLen(1))
			})
		})
	})

	Context("test quickStartHandler", func() {

		var exists *consolev1.ConsoleQuickStart = nil
		It("should create the ConsoleQuickStart resource if not exists", func() {
			dir := path.Join(os.TempDir(), fmt.Sprint(time.Now().UTC().Unix()))
			_ = os.Setenv(manifestLocationVarName, dir)
			err := os.Mkdir(dir, 0744)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = copyFile(path.Join(dir, "quickStart.yaml"), path.Join(testFilesLocation, "quickstart.yaml"))
			Expect(err).ToNot(HaveOccurred())
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd})
			handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))

			{ // for the next test
				handler, ok := handlers[0].(*genericOperand)
				Expect(ok).To(BeTrue())
				hooks, ok := handler.hooks.(*qsHooks)
				Expect(ok).To(BeTrue())
				exists = hooks.required.DeepCopy()
			}

			hco := commonTestUtils.NewHco()
			By("apply the quickstart CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Created).To(BeTrue())

				quickstartObjects := &consolev1.ConsoleQuickStartList{}
				err := cli.List(context.TODO(), quickstartObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(quickstartObjects.Items).To(HaveLen(1))
				Expect(quickstartObjects.Items[0].Name).Should(Equal("test-quick-start"))
			})
		})

		It("should update the ConsoleQuickStart resource if not not equal to the expected one", func() {

			Expect(exists).ToNot(BeNil(), "Must run the previous test first")
			exists.Spec.DurationMinutes = exists.Spec.DurationMinutes * 2

			dir := path.Join(os.TempDir(), fmt.Sprint(time.Now().UTC().Unix()))
			_ = os.Setenv(manifestLocationVarName, dir)
			err := os.Mkdir(dir, 0744)
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(dir)

			err = copyFile(path.Join(dir, "quickStart.yaml"), path.Join(testFilesLocation, "quickstart.yaml"))
			Expect(err).ToNot(HaveOccurred())

			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, exists})
			handlers, err := getQuickStartHandlers(logger, cli, schemeForTest)
			Expect(err).ToNot(HaveOccurred())
			Expect(handlers).To(HaveLen(1))

			hco := commonTestUtils.NewHco()
			By("apply the quickstart CRs", func() {
				req := commonTestUtils.NewReq(hco)
				res := handlers[0].ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())

				quickstartObjects := &consolev1.ConsoleQuickStartList{}
				err := cli.List(context.TODO(), quickstartObjects)
				Expect(err).ToNot(HaveOccurred())
				Expect(quickstartObjects.Items).To(HaveLen(1))
				Expect(quickstartObjects.Items[0].Name).Should(Equal("test-quick-start"))
				// check that the existing object was reconciled
				Expect(quickstartObjects.Items[0].Spec.DurationMinutes).Should(Equal(20))
			})
		})
	})
})

func copyFile(dest, src string) error {
	fin, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fin.Close()

	fout, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer fout.Close()

	_, err = io.Copy(fout, fin)

	if err != nil {
		return err
	}
	return nil
}
