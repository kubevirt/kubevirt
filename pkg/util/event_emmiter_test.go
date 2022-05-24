package util

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
)

var _ = Describe("", func() {
	Context("test UpdateClient", func() {
		const (
			rsName    = "hco-operator"
			podName   = rsName + "-12345"
			namespace = "kubevirt-hyperconverged"
		)

		recorder := newEventRecorderMock()
		ee := eventEmitter{
			pod: nil,
			csv: nil,
		}

		It("should not update pod if the pod not found", func() {
			justACmForTest := &corev1.ConfigMap{
				TypeMeta:   metav1.TypeMeta{Kind: "ConfigMap", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{Name: "justACmForTest", Namespace: namespace},
			}

			ee.Init(nil, nil, recorder)
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
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterServiceVersion",
					APIVersion: "operators.coreos.com/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      rsName,
					Namespace: namespace,
				},
			}

			rs := &appsv1.ReplicaSet{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ReplicaSet",
					APIVersion: "apps/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      rsName,
					Namespace: namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
							Name:       rsName,
							Controller: pointer.BoolPtr(true),
						},
					},
				},
			}

			pod := &corev1.Pod{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      podName,
					Namespace: namespace,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: "apps/v1",
							Kind:       "ReplicaSet",
							Name:       rsName,
							Controller: pointer.BoolPtr(true),
						},
					},
				},
			}

			ee.Init(pod, csv, recorder)

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
