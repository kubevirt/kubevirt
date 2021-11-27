package operands

import (
	"context"
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

var _ = Describe("Test operandHandler", func() {
	Context("Test operandHandler", func() {
		testFileLocation := getTestFilesLocation()

		_ = os.Setenv(quickStartManifestLocationVarName, testFileLocation+"/quickstarts")
		_ = os.Setenv(dashboardManifestLocationVarName, testFileLocation+"/dashboards")

		It("should create all objects are created", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)

			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())
			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created PriorityClass kubevirt-cluster-critical",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created KubeVirt kubevirt-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created CDI cdi-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created ConfigMap kubevirt-storage-class-defaults",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created NetworkAddonsConfig cluster",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created SSP ssp-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created Service kubevirt-hyperconverged-operator-metrics",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created ServiceMonitor kubevirt-hyperconverged-operator-metrics",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created PrometheusRule kubevirt-hyperconverged-prometheus-rule",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created ConsoleQuickStart test-quick-start",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created ConfigMap grafana-dashboard-kubevirt-top-consumers",
				},
			}
			Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeTrue())

			By("make sure the KV object created", func() {
				// Read back KV
				kvList := kubevirtcorev1.KubeVirtList{}
				err := cli.List(req.Ctx, &kvList)
				Expect(err).ToNot(HaveOccurred())
				Expect(kvList).ToNot(BeNil())
				Expect(kvList.Items).To(HaveLen(1))
				Expect(kvList.Items[0].Name).Should(Equal("kubevirt-kubevirt-hyperconverged"))
			})

			By("make sure the CNA object created", func() {
				// Read back CNA
				cnaList := networkaddonsv1.NetworkAddonsConfigList{}
				err := cli.List(req.Ctx, &cnaList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnaList).ToNot(BeNil())
				Expect(cnaList.Items).To(HaveLen(1))
				Expect(cnaList.Items[0].Name).Should(Equal("cluster"))
			})

			By("make sure the CDI object created", func() {
				// Read back CDI
				cdiList := cdiv1beta1.CDIList{}
				err := cli.List(req.Ctx, &cdiList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdiList).ToNot(BeNil())
				Expect(cdiList.Items).To(HaveLen(1))
				Expect(cdiList.Items[0].Name).Should(Equal("cdi-kubevirt-hyperconverged"))
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

			By("make sure the Dashboard confimap created", func() {
				cmList := corev1.ConfigMapList{}
				err := cli.List(req.Ctx, &cmList, &client.ListOptions{Namespace: "openshift-config-managed"})
				Expect(err).ToNot(HaveOccurred())
				Expect(cmList).ToNot(BeNil())
				Expect(cmList.Items).To(HaveLen(1))
				Expect(cmList.Items[0].Name).Should(Equal("grafana-dashboard-kubevirt-top-consumers"))
			})
		})

		It("should handle errors on ensure loop", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)

			// fail to create CDI
			fakeError := fmt.Errorf("fake create CDI error")
			cli.InitiateCreateErrors(func(obj client.Object) error {
				if _, ok := obj.(*cdiv1beta1.CDI); ok {
					return fakeError
				}

				return nil
			})

			err := handler.Ensure(req)
			Expect(err).To(HaveOccurred())
			Expect(err).Should(Equal(fakeError))

			Expect(req.ComponentUpgradeInProgress).To(BeFalse())
			cond := req.Conditions[hcov1beta1.ConditionReconcileComplete]
			Expect(cond).ToNot(BeNil())
			Expect(cond.Status).Should(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).Should(Equal(reconcileFailed))
			Expect(cond.Message).Should(Equal(fmt.Sprintf("Error while reconciling: %v", fakeError)))

			By("make sure the CDI object not created", func() {
				// Read back CDI
				cdiList := cdiv1beta1.CDIList{}
				err := cli.List(req.Ctx, &cdiList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdiList).ToNot(BeNil())
				Expect(cdiList.Items).To(BeEmpty())
			})
		})

		It("make sure the all objects are deleted", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()
			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			eventEmitter.Reset()
			err = handler.EnsureDeleted(req)
			Expect(err).ToNot(HaveOccurred())

			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed ConsoleCLIDownload virtctl-clidownloads-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed NetworkAddonsConfig cluster",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed CDI cdi-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed ConsoleQuickStart test-quick-start",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed SSP ssp-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed KubeVirt kubevirt-kubevirt-hyperconverged",
				},
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed ConfigMap grafana-dashboard-kubevirt-top-consumers",
				},
			}
			Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeTrue())

			By("check that KV is deleted", func() {
				// Read back KV
				kvList := kubevirtcorev1.KubeVirtList{}
				err = cli.List(req.Ctx, &kvList)
				Expect(err).ToNot(HaveOccurred())
				Expect(kvList).ToNot(BeNil())
				Expect(kvList.Items).To(BeEmpty())
			})

			By("make sure the CNA object deleted", func() {
				// Read back CNA
				cnaList := networkaddonsv1.NetworkAddonsConfigList{}
				err := cli.List(req.Ctx, &cnaList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnaList).ToNot(BeNil())
				Expect(cnaList.Items).To(BeEmpty())
			})

			By("make sure the CDI object deleted", func() {
				// Read back CDI
				cdiList := cdiv1beta1.CDIList{}
				err := cli.List(req.Ctx, &cdiList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdiList).ToNot(BeNil())
				Expect(cdiList.Items).To(BeEmpty())
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

		It("delete KV error handling", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			fakeError := fmt.Errorf("fake KV deletion error")
			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "KubeVirt" {
						return fakeError
					}
				}
				return nil
			})

			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    ErrVirtUninstall,
					Msg:       uninstallVirtErrorMsg + fakeError.Error(),
				},
			}
			eventEmitter.Reset()
			err = handler.EnsureDeleted(req)
			Expect(err).Should(Equal(fakeError))

			By("Check that event was emitted", func() {
				Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeTrue())
			})

			By("check that KV still exists", func() {
				// Read back KV
				kvList := kubevirtcorev1.KubeVirtList{}
				err := cli.List(req.Ctx, &kvList)
				Expect(err).ToNot(HaveOccurred())
				Expect(kvList).ToNot(BeNil())
				Expect(kvList.Items).To(HaveLen(1))
				Expect(kvList.Items[0].Name).Should(Equal("kubevirt-kubevirt-hyperconverged"))
			})
		})

		It("delete CDI error handling", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()
			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			fakeError := fmt.Errorf("fake CDI deletion error")
			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "CDI" {
						return fakeError
					}
				}
				return nil
			})

			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    ErrCDIUninstall,
					Msg:       uninstallCDIErrorMsg + fakeError.Error(),
				},
			}

			eventEmitter.Reset()
			err = handler.EnsureDeleted(req)
			Expect(err).Should(Equal(fakeError))

			By("Check that event was emitted", func() {
				Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeTrue())
			})

			By("make sure the CDI object still exists", func() {
				// Read back KV
				cdiList := cdiv1beta1.CDIList{}
				err := cli.List(req.Ctx, &cdiList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdiList).ToNot(BeNil())
				Expect(cdiList.Items).To(HaveLen(1))
				Expect(cdiList.Items[0].Name).Should(Equal("cdi-kubevirt-hyperconverged"))
			})
		})

		It("default delete error handling", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			fakeError := fmt.Errorf("fake CNA deletion error")
			eventEmitter := commonTestUtils.NewEventEmitterMock()

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "NetworkAddonsConfig" {
						return fakeError
					}
				}
				return nil
			})

			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    ErrHCOUninstall,
					Msg:       uninstallHCOErrorMsg,
				},
			}

			eventEmitter.Reset()
			err = handler.EnsureDeleted(req)
			Expect(err).Should(Equal(fakeError))

			By("Check that event was emitted", func() {
				Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeTrue())
			})

			By("make sure the CNA object still exists", func() {
				// Read back CNA
				cnaList := networkaddonsv1.NetworkAddonsConfigList{}
				err := cli.List(req.Ctx, &cnaList)
				Expect(err).ToNot(HaveOccurred())
				Expect(cnaList).ToNot(BeNil())
				Expect(cnaList.Items).To(HaveLen(1))
				Expect(cnaList.Items[0].Name).Should(Equal("cluster"))
			})
		})

		It("delete timeout error handling", func() {
			hco := commonTestUtils.NewHco()
			cli := commonTestUtils.InitClient([]runtime.Object{qsCrd, hco})

			eventEmitter := commonTestUtils.NewEventEmitterMock()

			handler := NewOperandHandler(cli, commonTestUtils.GetScheme(), true, eventEmitter)
			handler.FirstUseInitiation(commonTestUtils.GetScheme(), true, hco)

			req := commonTestUtils.NewReq(hco)
			err := handler.Ensure(req)
			Expect(err).ToNot(HaveOccurred())

			cli.InitiateDeleteErrors(func(obj client.Object) error {
				if unstructed, ok := obj.(runtime.Unstructured); ok {
					kind := unstructed.GetObjectKind()
					if kind.GroupVersionKind().Kind == "NetworkAddonsConfig" {
						time.Sleep(time.Millisecond * 500)
					}
				}
				return nil
			})

			eventEmitter.Reset()
			ctx, cancelFunc := context.WithTimeout(req.Ctx, time.Millisecond*300)
			defer cancelFunc()
			req.Ctx = ctx
			err = handler.EnsureDeleted(req)
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(Equal("context deadline exceeded"))

			expectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed NetworkAddonsConfig cluster",
				},
			}

			By("Check that event was *not* emitted", func() {
				Expect(eventEmitter.CheckEvents(expectedEvents)).To(BeFalse())
			})
		})
	})
})
