package util

import (
	"context"
	"os"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Test OwnResources", func() {
	const (
		rsName    = "hco-operator"
		podName   = rsName + "-12345"
		namespace = "kubevirt-hyperconverged"
	)

	var (
		origGetOperatorNamespace = GetOperatorNamespace
		origPodName              = os.Getenv(PodNameEnvVar)
		origGetClusterInfo       = GetClusterInfo
	)

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

	testScheme := scheme.Scheme
	_ = csvv1alpha1.AddToScheme(testScheme)

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

		dep := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: "apps/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      rsName,
				Namespace: namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "operators.coreos.com/v1alpha1",
						Kind:       csvv1alpha1.ClusterServiceVersionKind,
						Name:       rsName,
						Controller: pointer.BoolPtr(true),
					},
				},
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

		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithRuntimeObjects(csv, dep, rs, pod).
			Build()

		or := findOwnResources(context.Background(), cl, logger)
		Expect(*or.GetPod()).Should(Equal(*pod))
		Expect(*or.GetDeployment()).Should(Equal(*dep))
		Expect(*or.GetCSV()).Should(Equal(*csv))
	})

})
