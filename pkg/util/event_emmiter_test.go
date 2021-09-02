package util

import (
	"context"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var _ = Describe("", func() {
	var (
		logger                   = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("eventEmmiter_test")
		ctx                      = context.TODO()
		origGetOperatorNamespace = GetOperatorNamespace
		origPodName              = os.Getenv(PodNameEnvVar)
		controllerTrue           = true
	)

	Context("test UpdateClient", func() {
		const (
			rsName    = "hco-operator"
			podName   = rsName + "-12345"
			namespace = "kubevirt-hyperconverged"
		)

		origGetClusterInfo := GetClusterInfo

		BeforeEach(func() {
			GetOperatorNamespace = func(_ logr.Logger) (string, error) {
				return namespace, nil
			}

			os.Setenv(PodNameEnvVar, podName)

			GetClusterInfo = func() ClusterInfo {
				return &ClusterInfoImp{
					runningInOpenshift: true,
					managedByOLM:       true,
					runningLocally:     false,
				}
			}
		})

		AfterEach(func() {
			GetOperatorNamespace = origGetOperatorNamespace
			os.Setenv(PodNameEnvVar, origPodName)
			GetClusterInfo = origGetClusterInfo
		})

		recorder := newEventRecorderMock()
		ee := eventEmitter{
			pod: nil,
			csv: nil,
		}

		testScheme := scheme.Scheme
		err := csvv1alpha1.AddToScheme(testScheme)
		Expect(err).ToNot(HaveOccurred())

		It("should not update pod if the pod not found", func() {
			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				Build()

			justACmForTest := &corev1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "justACmForTest", Namespace: namespace},
			}

			ee.Init(ctx, cl, recorder, logger)
			Expect(ee.pod).To(BeNil())
			Expect(ee.csv).To(BeNil())

			By("should emmit event for all three resources", func() {
				// we'll use the replica set as object, because we just need one. Originally we would use the HyperConverged
				// resource, but this is not accessible (cyclic import)
				expectedEvent := eventMock{
					eventType: corev1.EventTypeNormal,
					reason:    "justTesting",
					message:   "this is a test message",
				}

				ee.EmitEvent(justACmForTest, corev1.EventTypeNormal, "justTesting", "this is a test message")
				mock := ee.recorder.(*EventRecorderMock)

				rsEvent, found := mock.events["ConfigMap"]
				Expect(found).To(BeTrue())
				Expect(rsEvent).Should(Equal(expectedEvent))

				_, found = mock.events["Pod"]
				Expect(found).To(BeFalse())

				_, found = mock.events["ClusterServiceVersion"]
				Expect(found).To(BeFalse())
			})
		})

		It("should update pod and csv if they are found", func() {
			csv := &csvv1alpha1.ClusterServiceVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rsName,
					Namespace: namespace,
				},
			}

			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rsName,
					Namespace: namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "operators.coreos.com/v1alpha1",
							Kind:       csvv1alpha1.ClusterServiceVersionKind,
							Name:       rsName,
							Controller: &controllerTrue,
						},
					},
				},
			}

			rs := &appsv1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      rsName,
					Namespace: namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       rsName,
							Controller: &controllerTrue,
						},
					},
				},
			}

			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       rsName,
							Controller: &controllerTrue,
						},
					},
				},
			}

			cl := fake.NewClientBuilder().
				WithScheme(testScheme).
				WithRuntimeObjects(csv, dep, rs, pod).
				Build()

			Expect(GetClusterInfo().IsOpenshift()).To(BeTrue())
			ee.Init(ctx, cl, recorder, logger)

			Expect(ee.pod).ToNot(BeNil())
			Expect(ee.csv).ToNot(BeNil())

			By("should emmit event for all three resources", func() {
				// we'll use the replica set as object, because we just need one. Originally we would use the HyperConverged
				// resource, but this is not accessible (cyclic import)
				expectedEvent := eventMock{
					eventType: corev1.EventTypeNormal,
					reason:    "justTesting",
					message:   "this is a test message",
				}

				ee.EmitEvent(rs, corev1.EventTypeNormal, "justTesting", "this is a test message")
				mock := ee.recorder.(*EventRecorderMock)

				rsEvent, found := mock.events["ReplicaSet"]
				Expect(found).To(BeTrue())
				Expect(rsEvent).Should(Equal(expectedEvent))

				rsEvent, found = mock.events["Pod"]
				Expect(found).To(BeTrue())
				Expect(rsEvent).Should(Equal(expectedEvent))

				rsEvent, found = mock.events["ClusterServiceVersion"]
				Expect(found).To(BeTrue())
				Expect(rsEvent).Should(Equal(expectedEvent))
			})

		})
	})
})

type eventMock struct {
	eventType string
	reason    string
	message   string
}

type EventRecorderMock struct {
	events map[string]eventMock
}

func newEventRecorderMock() *EventRecorderMock {
	return &EventRecorderMock{
		events: make(map[string]eventMock),
	}
}

func (mock EventRecorderMock) Event(object runtime.Object, eventType, reason, message string) {
	kind := object.GetObjectKind().GroupVersionKind().Kind
	mock.events[kind] = eventMock{eventType: eventType, reason: reason, message: message}
}
func (mock EventRecorderMock) Eventf(_ runtime.Object, _, _, _ string, _ ...interface{}) {
	/* not implemented */
}
func (mock EventRecorderMock) AnnotatedEventf(_ runtime.Object, _ map[string]string, _, _, _ string, _ ...interface{}) {
	/* not implemented */
}
