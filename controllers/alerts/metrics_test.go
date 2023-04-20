package alerts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

func TestAlerts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Alerts Suite")
}

var _ = Describe("alert tests", func() {
	var (
		ci            = commonTestUtils.ClusterInfoMock{}
		ee            = commonTestUtils.NewEventEmitterMock()
		ns            *corev1.Namespace
		req           *common.HcoRequest
		currentMetric float64
	)

	BeforeEach(func() {
		ee.Reset()
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: commonTestUtils.Namespace,
			},
		}

		req = commonTestUtils.NewReq(nil)
	})

	Context("test reconciler", func() {

		expectedEvents := []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Created",
				Msg:       "Created PrometheusRule " + ruleName,
			},
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Created",
				Msg:       "Created Role " + roleName,
			},
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Created",
				Msg:       "Created RoleBinding " + roleName,
			},
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Created",
				Msg:       "Created Service " + serviceName,
			},
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Created",
				Msg:       "Created ServiceMonitor " + serviceName,
			},
		}

		It("should create all the resources if missing", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{ns})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())

			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())

			hco := commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
			Expect(r.UpdateRelatedObjects(req)).Should(Succeed())
			Expect(req.StatusDirty).To(BeTrue())
			Expect(hco.Status.RelatedObjects).To(HaveLen(5))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
		})

		It("should fail on error", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{ns})
			fakeError := fmt.Errorf("fake error")
			cl.InitiateCreateErrors(func(obj client.Object) error {
				if obj.GetObjectKind().GroupVersionKind().Kind == "Service" {
					return fakeError
				}
				return nil
			})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			err := r.Reconcile(req, false)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(fakeError))
		})
	})

	Context("test PrometheusRule", func() {
		BeforeEach(func() {
			currentMetric, _ = metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)
		})

		AfterEach(func() {
			os.Unsetenv(runbookURLTemplateEnv)
		})

		expectedEvents := []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Updated",
				Msg:       "Updated PrometheusRule " + ruleName,
			},
		}

		It("should update the labels if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)
			existRule.Labels = map[string]string{
				"wrongKey1": "wrongValue1",
				"wrongKey2": "wrongValue2",
				"wrongKey3": "wrongValue3",
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())

			Expect(pr.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should add the labels if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)
			existRule.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())

			Expect(pr.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified", func() {
			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(pr.OwnerReferences).Should(HaveLen(1))
			Expect(pr.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(pr.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(pr.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(pr.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified; not HCO triggered", func() {

			req.HCOTriggered = false
			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(pr.OwnerReferences).Should(HaveLen(1))
			Expect(pr.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(pr.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(pr.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(pr.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			overrideExpectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    "Overwritten",
					Msg:       "Overwritten PrometheusRule " + ruleName,
				},
			}

			Expect(ee.CheckEvents(overrideExpectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric + 1))
		})

		It("should update the referenceOwner if missing", func() {
			owner := metav1.OwnerReference{}
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)
			existRule.OwnerReferences = nil
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(pr.OwnerReferences).Should(HaveLen(1))
			Expect(pr.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(pr.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(pr.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(pr.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the spec if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)

			existRule.Spec.Groups[0].Rules = []monitoringv1.Rule{
				existRule.Spec.Groups[0].Rules[0],
				existRule.Spec.Groups[0].Rules[2],
				existRule.Spec.Groups[0].Rules[3],
			}
			// modify the first rule
			existRule.Spec.Groups[0].Rules[0].Alert = "modified alert"

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())
			Expect(pr.Spec).Should(Equal(*NewPrometheusRuleSpec()))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the spec if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)

			existRule.Spec = monitoringv1.PrometheusRuleSpec{}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())
			Expect(pr.Spec).Should(Equal(*NewPrometheusRuleSpec()))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should use the default runbook URL template when no ENV Variable is set", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			promRule := newPrometheusRule(commonTestUtils.Namespace, owner)

			for _, group := range promRule.Spec.Groups {
				for _, rule := range group.Rules {
					if rule.Alert != "" {
						if rule.Annotations["runbook_url"] != "" {
							Expect(rule.Annotations["runbook_url"]).To(Equal(fmt.Sprintf(defaultRunbookURLTemplate, rule.Alert)))
						}
					}
				}
			}
		})

		It("should use the desired runbook URL template when its ENV Variable is set", func() {
			desiredRunbookURLTemplate := "desired/runbookURL/template/%s"
			os.Setenv(runbookURLTemplateEnv, desiredRunbookURLTemplate)

			owner := getDeploymentReference(ci.GetDeployment())
			promRule := newPrometheusRule(commonTestUtils.Namespace, owner)

			for _, group := range promRule.Spec.Groups {
				for _, rule := range group.Rules {
					if rule.Alert != "" {
						if rule.Annotations["runbook_url"] != "" {
							Expect(rule.Annotations["runbook_url"]).To(Equal(fmt.Sprintf(desiredRunbookURLTemplate, rule.Alert)))
						}
					}
				}
			}
		})

		DescribeTable("test the OverwrittenModificationsCount", func(hcoTriggered, upgradeMode, firstLoop bool, expectedCountDelta float64) {
			req.HCOTriggered = hcoTriggered
			req.UpgradeMode = upgradeMode

			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRule := newPrometheusRule(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRule})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, firstLoop)).Should(Succeed())
			pr := &monitoringv1.PrometheusRule{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: ruleName}, pr)).Should(Succeed())

			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount(monitoringv1.PrometheusRuleKind, ruleName)).Should(BeEquivalentTo(currentMetric + expectedCountDelta))
		},
			Entry("should not increase the counter if it HCO triggered, in upgrade mode and in the first loop", true, true, true, float64(0)), // can't really happen
			Entry("should not increase the counter if it HCO triggered, not in upgrade mode but in the first loop", true, false, true, float64(0)),
			Entry("should not increase the counter if it HCO triggered, in upgrade mode but not in the first loop", true, true, false, float64(0)),
			Entry("should not increase the counter if it HCO triggered, not in upgrade mode and not in the first loop", true, false, false, float64(0)),

			Entry("should not increase the counter if it not HCO triggered, in upgrade mode and in the first loop", false, true, true, float64(0)), // can't really happen
			Entry("should not increase the counter if it not HCO triggered, not in upgrade mode but in the first loop", false, false, true, float64(0)),
			Entry("should not increase the counter if it not HCO triggered, in upgrade mode and not in the first loop", false, true, false, float64(0)),
			Entry("should increase the counter if it not HCO triggered, not in upgrade mode and not in the first loop", false, false, false, float64(1)),
		)
	})

	Context("test Role", func() {
		BeforeEach(func() {
			currentMetric, _ = metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)
		})

		expectedEvents := []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Updated",
				Msg:       "Updated Role " + roleName,
			},
		}

		It("should update the labels if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRole := newRole(owner, commonTestUtils.Namespace)
			existRole.Labels = map[string]string{
				"wrongKey1": "wrongValue1",
				"wrongKey2": "wrongValue2",
				"wrongKey3": "wrongValue3",
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())

			Expect(role.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the labels if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRole := newRole(owner, commonTestUtils.Namespace)
			existRole.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())

			Expect(role.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified", func() {
			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRole := newRole(owner, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(role.OwnerReferences).Should(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(role.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(role.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(role.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified; not HCO triggered", func() {
			req.HCOTriggered = false

			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRole := newRole(owner, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(role.OwnerReferences).Should(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(role.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(role.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(role.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			overrideExpectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    "Overwritten",
					Msg:       "Overwritten Role " + roleName,
				},
			}

			Expect(ee.CheckEvents(overrideExpectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric + 1))
		})

		It("should update the referenceOwner if missing", func() {
			owner := metav1.OwnerReference{}
			existRole := newRole(owner, commonTestUtils.Namespace)
			existRole.OwnerReferences = nil
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(role.OwnerReferences).Should(HaveLen(1))
			Expect(role.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(role.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(role.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(role.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Rules if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRole := newRole(owner, commonTestUtils.Namespace)

			existRole.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"wrongGroup1"},
					Resources: []string{"wrongResource1", "wrongResource2", "wrongResource3", "wrongResource4"},
					Verbs:     []string{"list", "update"},
				},
				{
					APIGroups: []string{"wrongGroup2"},
					Verbs:     []string{"list", "update", "help"},
				},
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())
			Expect(role.Rules).Should(HaveLen(1))
			Expect(role.Rules[0].APIGroups).Should(Equal([]string{""}))
			Expect(role.Rules[0].Resources).Should(Equal([]string{"services", "endpoints", "pods"}))
			Expect(role.Rules[0].Verbs).Should(Equal([]string{"get", "list", "watch"}))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Rules if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRole := newRole(owner, commonTestUtils.Namespace)

			existRole.Rules = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRole})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			role := &rbacv1.Role{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, role)).Should(Succeed())
			Expect(role.Rules).Should(HaveLen(1))
			Expect(role.Rules[0].APIGroups).Should(Equal([]string{""}))
			Expect(role.Rules[0].Resources).Should(Equal([]string{"services", "endpoints", "pods"}))
			Expect(role.Rules[0].Verbs).Should(Equal([]string{"get", "list", "watch"}))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Role", roleName)).Should(BeEquivalentTo(currentMetric))
		})
	})

	Context("test RoleBinding", func() {
		BeforeEach(func() {
			currentMetric, _ = metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)
		})

		expectedEvents := []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Updated",
				Msg:       "Updated RoleBinding " + roleName,
			},
		}

		It("should update the labels if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)
			existRB.Labels = map[string]string{
				"wrongKey1": "wrongValue1",
				"wrongKey2": "wrongValue2",
				"wrongKey3": "wrongValue3",
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())

			Expect(rb.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the labels if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)
			existRB.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())

			Expect(rb.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified", func() {
			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(rb.OwnerReferences).Should(HaveLen(1))
			Expect(rb.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(rb.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(rb.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(rb.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified; not HCO triggered", func() {
			req.HCOTriggered = false

			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(rb.OwnerReferences).Should(HaveLen(1))
			Expect(rb.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(rb.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(rb.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(rb.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			overrideExpectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    "Overwritten",
					Msg:       "Overwritten RoleBinding " + roleName,
				},
			}

			Expect(ee.CheckEvents(overrideExpectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric + 1))
		})

		It("should update the referenceOwner if missing", func() {
			owner := metav1.OwnerReference{}
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)
			existRB.OwnerReferences = nil
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(rb.OwnerReferences).Should(HaveLen(1))
			Expect(rb.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(rb.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(rb.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(rb.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the RoleRef if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)

			existRB.RoleRef = rbacv1.RoleRef{
				APIGroup: "wrongAPIGroup",
				Kind:     "wrongKind",
				Name:     "wrongName",
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())
			Expect(rb.RoleRef.APIGroup).Should(Equal(rbacv1.GroupName))
			Expect(rb.RoleRef.Kind).Should(Equal("Role"))
			Expect(rb.RoleRef.Name).Should(Equal(roleName))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the RoleRef if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)

			existRB.RoleRef = rbacv1.RoleRef{}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())
			Expect(rb.RoleRef.APIGroup).Should(Equal(rbacv1.GroupName))
			Expect(rb.RoleRef.Kind).Should(Equal("Role"))
			Expect(rb.RoleRef.Name).Should(Equal(roleName))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Subjects if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)

			existRB.Subjects = []rbacv1.Subject{
				{
					Kind:      "wrongKind1",
					Name:      "wrongName1",
					Namespace: "wrongNamespace1",
				},
				{
					Kind:      "wrongKind2",
					Name:      "wrongName2",
					Namespace: "wrongNamespace2",
				},
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())
			Expect(rb.Subjects).Should(HaveLen(1))
			Expect(rb.Subjects[0].Kind).Should(Equal(rbacv1.ServiceAccountKind))
			Expect(rb.Subjects[0].Name).Should(Equal("prometheus-k8s"))
			Expect(rb.Subjects[0].Namespace).Should(Equal(monitoringNamespace))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Subjects if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existRB := newRoleBinding(owner, commonTestUtils.Namespace)

			existRB.Subjects = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existRB})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())

			rb := &rbacv1.RoleBinding{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: roleName}, rb)).Should(Succeed())
			Expect(rb.Subjects).Should(HaveLen(1))
			Expect(rb.Subjects[0].Kind).Should(Equal(rbacv1.ServiceAccountKind))
			Expect(rb.Subjects[0].Name).Should(Equal("prometheus-k8s"))
			Expect(rb.Subjects[0].Namespace).Should(Equal(monitoringNamespace))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("RoleBinding", roleName)).Should(BeEquivalentTo(currentMetric))
		})
	})

	Context("test Service", func() {
		BeforeEach(func() {
			currentMetric, _ = metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)
		})

		expectedEvents := []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Updated",
				Msg:       "Updated Service " + serviceName,
			},
		}

		It("should update the labels if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)
			existSM.Labels = map[string]string{
				"wrongKey1": "wrongValue1",
				"wrongKey2": "wrongValue2",
				"wrongKey3": "wrongValue3",
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())

			Expect(svc.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the labels if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)
			existSM.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())

			Expect(svc.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified", func() {
			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(svc.OwnerReferences).Should(HaveLen(1))
			Expect(svc.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(svc.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(svc.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(svc.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified; No HCO triggered", func() {
			req.HCOTriggered = false

			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(svc.OwnerReferences).Should(HaveLen(1))
			Expect(svc.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(svc.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(svc.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(svc.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			overrideExpectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    "Overwritten",
					Msg:       "Overwritten Service " + serviceName,
				},
			}

			Expect(ee.CheckEvents(overrideExpectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric + 1))
		})

		It("should update the referenceOwner if missing", func() {
			owner := metav1.OwnerReference{}
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)
			existSM.OwnerReferences = nil
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(svc.OwnerReferences).Should(HaveLen(1))
			Expect(svc.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(svc.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(svc.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(svc.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Spec if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)

			existSM.Spec = corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port:     1234,
						Name:     "wrongName",
						Protocol: corev1.ProtocolUDP,
						TargetPort: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: 1234,
						},
					},
				},
				Selector: map[string]string{
					"wrongKey1": "wrongValue1",
					"wrongKey2": "wrongValue2",
				},
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())
			Expect(svc.Spec.Ports).Should(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(hcoutil.MetricsPort))
			Expect(svc.Spec.Ports[0].Name).Should(Equal(operatorPortName))
			Expect(svc.Spec.Ports[0].Protocol).Should(Equal(corev1.ProtocolTCP))
			Expect(svc.Spec.Ports[0].TargetPort).Should(Equal(intstr.IntOrString{Type: intstr.Int, IntVal: hcoutil.MetricsPort}))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Spec if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewMetricsService(commonTestUtils.Namespace, owner)

			existSM.Spec = corev1.ServiceSpec{}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			svc := &corev1.Service{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, svc)).Should(Succeed())
			Expect(svc.Spec.Ports).Should(HaveLen(1))
			Expect(svc.Spec.Ports[0].Port).Should(Equal(hcoutil.MetricsPort))
			Expect(svc.Spec.Ports[0].Name).Should(Equal(operatorPortName))
			Expect(svc.Spec.Ports[0].Protocol).Should(Equal(corev1.ProtocolTCP))
			Expect(svc.Spec.Ports[0].TargetPort).Should(Equal(intstr.IntOrString{Type: intstr.Int, IntVal: hcoutil.MetricsPort}))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("Service", serviceName)).Should(BeEquivalentTo(currentMetric))
		})
	})

	Context("test ServiceMonitor", func() {
		BeforeEach(func() {
			currentMetric, _ = metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)
		})

		expectedEvents := []commonTestUtils.MockEvent{
			{
				EventType: corev1.EventTypeNormal,
				Reason:    "Updated",
				Msg:       "Updated ServiceMonitor " + serviceName,
			},
		}

		It("should update the labels if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)
			existSM.Labels = map[string]string{
				"wrongKey1": "wrongValue1",
				"wrongKey2": "wrongValue2",
				"wrongKey3": "wrongValue3",
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())

			Expect(sm.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the labels if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)
			existSM.Labels = nil

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())

			Expect(sm.Labels).Should(Equal(hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)))
			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified", func() {
			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(sm.OwnerReferences).Should(HaveLen(1))
			Expect(sm.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(sm.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(sm.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(sm.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the referenceOwner if modified; no HCO triggered", func() {
			req.HCOTriggered = false

			owner := metav1.OwnerReference{
				APIVersion:         "wrongAPIVersion",
				Kind:               "wrongKind",
				Name:               "wrongName",
				Controller:         pointer.Bool(true),
				BlockOwnerDeletion: pointer.Bool(true),
				UID:                "0987654321",
			}
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(sm.OwnerReferences).Should(HaveLen(1))
			Expect(sm.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(sm.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(sm.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(sm.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			overrideExpectedEvents := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeWarning,
					Reason:    "Overwritten",
					Msg:       "Overwritten ServiceMonitor " + serviceName,
				},
			}

			Expect(ee.CheckEvents(overrideExpectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric + 1))
		})

		It("should update the referenceOwner if missing", func() {
			owner := metav1.OwnerReference{}
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)
			existSM.OwnerReferences = nil
			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())

			deployment := ci.GetDeployment()

			Expect(sm.OwnerReferences).Should(HaveLen(1))
			Expect(sm.OwnerReferences[0].Name).Should(Equal(deployment.Name))
			Expect(sm.OwnerReferences[0].Kind).Should(Equal("Deployment"))
			Expect(sm.OwnerReferences[0].APIVersion).Should(Equal(appsv1.GroupName + "/v1"))
			Expect(sm.OwnerReferences[0].UID).Should(Equal(deployment.UID))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Spec if modified", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)

			existSM.Spec = monitoringv1.ServiceMonitorSpec{
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"wrongKey1": "wrongValue1",
						"wrongKey2": "wrongValue2",
					},
				},
				Endpoints: []monitoringv1.Endpoint{{Port: "wrongPort", Path: "/metrics"}},
			}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())
			Expect(sm.Spec.Selector).Should(Equal(metav1.LabelSelector{MatchLabels: hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)}))
			Expect(sm.Spec.Endpoints[0].Port).Should(Equal(operatorPortName))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric))
		})

		It("should update the Spec if it's missing", func() {
			owner := getDeploymentReference(ci.GetDeployment())
			existSM := NewServiceMonitor(commonTestUtils.Namespace, owner)

			existSM.Spec = monitoringv1.ServiceMonitorSpec{}

			cl := commonTestUtils.InitClient([]runtime.Object{ns, existSM})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())
			sm := &monitoringv1.ServiceMonitor{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Namespace: r.namespace, Name: serviceName}, sm)).Should(Succeed())
			Expect(sm.Spec.Selector).Should(Equal(metav1.LabelSelector{MatchLabels: hcoutil.GetLabels(hcoutil.HyperConvergedName, hcoutil.AppComponentMonitoring)}))
			Expect(sm.Spec.Endpoints[0].Port).Should(Equal(operatorPortName))

			Expect(ee.CheckEvents(expectedEvents)).To(BeTrue())
			Expect(metrics.HcoMetrics.GetOverwrittenModificationsCount("ServiceMonitor", serviceName)).Should(BeEquivalentTo(currentMetric))
		})
	})

	Context("test Namespace", func() {

		DescribeTable("validate the annotation and the label", func(nsGenerator func() *corev1.Namespace) {
			cl := commonTestUtils.InitClient([]runtime.Object{nsGenerator()})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())

			foundNS := &corev1.Namespace{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: r.namespace}, foundNS)).Should(Succeed())
			Expect(foundNS.Annotations).ShouldNot(BeEmpty())
			annotation, ok := foundNS.Annotations[hcoutil.OpenshiftNodeSelectorAnn]
			Expect(ok).To(BeTrue())
			Expect(annotation).To(BeEmpty())

			label, ok := foundNS.Labels[hcoutil.PrometheusNSLabel]
			Expect(ok).To(BeTrue())
			Expect(label).To(Equal("true"))
		},
			Entry("when the annotations and the labels fields are nil", func() *corev1.Namespace { return ns }),
			Entry("when the annotations and the labels fields are empty", func() *corev1.Namespace {
				ns.Annotations = map[string]string{}
				ns.Labels = map[string]string{}
				return ns
			}),
			Entry("when the annotation is not empty", func() *corev1.Namespace {
				ns.Annotations = map[string]string{hcoutil.OpenshiftNodeSelectorAnn: "notEmpty"}
				return ns
			}),
			Entry("when the label is empty", func() *corev1.Namespace {
				ns.Labels = map[string]string{hcoutil.PrometheusNSLabel: ""}
				return ns
			}),
			Entry("when the label is false", func() *corev1.Namespace {
				ns.Labels = map[string]string{hcoutil.PrometheusNSLabel: "false"}
				return ns
			}),
			Entry("when the label is wrong", func() *corev1.Namespace {
				ns.Labels = map[string]string{hcoutil.PrometheusNSLabel: "wrong"}
				return ns
			}),
		)

		It("should not modify other labels", func() {
			ns.Labels = map[string]string{"aaa": "AAA", "bbb": "BBB"}
			cl := commonTestUtils.InitClient([]runtime.Object{ns})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())

			foundNS := &corev1.Namespace{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: r.namespace}, foundNS)).Should(Succeed())

			Expect(foundNS.Labels).Should(HaveLen(3))
			Expect(foundNS.Labels).Should(HaveKeyWithValue("aaa", "AAA"))
			Expect(foundNS.Labels).Should(HaveKeyWithValue("bbb", "BBB"))
			Expect(foundNS.Labels).Should(HaveKeyWithValue(hcoutil.PrometheusNSLabel, "true"))
		})

		It("should not modify other annotations", func() {
			ns.Annotations = map[string]string{"aaa": "AAA", "bbb": "BBB"}
			cl := commonTestUtils.InitClient([]runtime.Object{ns})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(Succeed())

			foundNS := &corev1.Namespace{}
			Expect(cl.Get(context.Background(), client.ObjectKey{Name: r.namespace}, foundNS)).Should(Succeed())

			Expect(foundNS.Annotations).Should(HaveLen(3))
			Expect(foundNS.Annotations).Should(HaveKeyWithValue("aaa", "AAA"))
			Expect(foundNS.Annotations).Should(HaveKeyWithValue("bbb", "BBB"))
			Expect(foundNS.Annotations).Should(HaveKeyWithValue(hcoutil.OpenshiftNodeSelectorAnn, ""))
		})

		It("should return error if can't read the namespace", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			Expect(r.Reconcile(req, false)).Should(HaveOccurred())
		})

		It("should return error if failed to read the namespace", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{ns})
			err := errors.New("fake error")
			cl.InitiateGetErrors(func(_ client.ObjectKey) error {
				return err
			})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			retErr := r.Reconcile(req, false)
			Expect(retErr).Should(HaveOccurred())
			Expect(retErr).Should(MatchError(err))
		})

		It("should return error if can't update the namespace", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{ns})
			err := errors.New("fake error")
			cl.InitiateUpdateErrors(func(_ client.Object) error {
				return err
			})
			r := NewMonitoringReconciler(ci, cl, ee, commonTestUtils.GetScheme())

			retErr := r.Reconcile(req, false)
			Expect(retErr).Should(HaveOccurred())
			Expect(retErr).Should(MatchError(err))
		})
	})
})
