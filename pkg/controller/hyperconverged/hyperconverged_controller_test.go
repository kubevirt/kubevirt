package hyperconverged

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	realClient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	kubevirt "kubevirt.io/client-go/api/v1"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

const (
	namespace = "kubevirt-hyperconverged"
)

type args struct {
	hco        *hcov1alpha1.HyperConverged
	scc        *secv1.SecurityContextConstraints
	client     client.Client
	reconciler *ReconcileHyperConverged
}

func init() {
	hcov1alpha1.AddToScheme(scheme.Scheme)
	secv1.Install(scheme.Scheme)
	sspv1.SchemeBuilder.AddToScheme(scheme.Scheme)
	networkaddons.SchemeBuilder.AddToScheme(scheme.Scheme)
	cdi.AddToScheme(scheme.Scheme)
	kubevirt.AddToScheme(scheme.Scheme)
}

var _ = Describe("HyperconvergedController", func() {
	Describe("CR metadata and spec creation functions", func() {
		instance := &hcov1alpha1.HyperConverged{}
		instance.Name = "hyperconverged-cluster"
		appLabel := map[string]string{
			"app": instance.Name,
		}

		Context("KubeVirt Config CR", func() {
			It("should have metadata", func() {
				cr := newKubeVirtConfigForCR(instance, namespace)
				checkMetadata(cr.ObjectMeta, "kubevirt-config", appLabel, namespace)
			})
		})

		Context("KubeVirt CR", func() {
			It("should have metadata", func() {
				cr := newKubeVirtForCR(instance, namespace)
				checkMetadata(cr.ObjectMeta, "kubevirt-"+instance.Name, appLabel, namespace)
			})
		})

		Context("CDI CR", func() {
			It("should have metadata", func() {
				cr := newCDIForCR(instance, namespace)
				checkMetadata(cr.ObjectMeta, "cdi-"+instance.Name, appLabel, namespace)
			})
		})

		Context("Network Addons CR", func() {
			It("should have metadata and spec and namespace should be unspecified", func() {
				cr := newNetworkAddonsForCR(instance, "")
				checkMetadata(cr.ObjectMeta, networkaddonsnames.OPERATOR_CONFIG, appLabel, "")
				Expect(cr.Spec.Multus).To(Equal(&networkaddons.Multus{}))
				Expect(cr.Spec.LinuxBridge).To(Equal(&networkaddons.LinuxBridge{}))
				Expect(cr.Spec.KubeMacPool).To(Equal(&networkaddons.KubeMacPool{}))
			})
		})

		Context("KubeVirt Common Template Bundle CR", func() {
			It("should have metadata and namespace should be openshift", func() {
				cr := newKubeVirtCommonTemplateBundleForCR(instance, "openshift")
				checkMetadata(cr.ObjectMeta, "common-templates-"+instance.Name, appLabel, "openshift")
			})
		})

		Context("KubeVirt Node Labeller Bundle CR", func() {
			It("should have metadata", func() {
				cr := newKubeVirtNodeLabellerBundleForCR(instance, namespace)
				checkMetadata(cr.ObjectMeta, "node-labeller-"+instance.Name, appLabel, namespace)
			})
		})

		Context("KubeVirt Template Validator CR", func() {
			It("should have metadata", func() {
				cr := newKubeVirtTemplateValidatorForCR(instance, namespace)
				checkMetadata(cr.ObjectMeta, "template-validator-"+instance.Name, appLabel, namespace)
			})
		})

	})
	Describe("Deploying HCO", func() {
		Context("HCO Lifecycle", func() {
			It("should get deployed", func() {
				args := createArgs()
				doReconcile(args)

				// TODO: should we be tracking these versions?
				// Expect(args.hco.Status.OperatorVersion).Should(Equal(version))
				// Expect(args.hco.Status.TargetVersion).Should(Equal(version))
				// Expect(args.hco.Status.ObservedVersion).Should(Equal(version))
			})

			It("should create all resources", func() {
				args := createArgs()
				doReconcile(args)

				expectedResources := args.reconciler.getAllResources(args.hco, reconcileRequest(namespace))
				for _, r := range expectedResources {
					_, err := getObject(args.client, r)
					Expect(err).ToNot(HaveOccurred())
				}
			})

			It("should be able to detect resources that aren't in hco", func() {
				args := createArgs()
				doReconcile(args)

				r := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: "should-not-exist",
					}}
				_, err := getObject(args.client, r)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})

func checkMetadata(metadata metav1.ObjectMeta, expectedName string, expectedLabel map[string]string, expectedNamespace string) {
	Expect(metadata.Name).To(Equal(expectedName))
	Expect(metadata.Labels).To(Equal(expectedLabel))
	Expect(metadata.Namespace).To(Equal(expectedNamespace))
}

func getObject(client realClient.Client, obj runtime.Object) (runtime.Object, error) {
	metaObj := obj.(metav1.Object)
	key := realClient.ObjectKey{Namespace: metaObj.GetNamespace(), Name: metaObj.GetName()}

	typ := reflect.ValueOf(obj).Elem().Type()
	result := reflect.New(typ).Interface().(runtime.Object)

	if err := client.Get(context.TODO(), key, result); err != nil {
		return nil, err
	}

	return result, nil
}

func getHyperConverged(client realClient.Client, hco *hcov1alpha1.HyperConverged) (*hcov1alpha1.HyperConverged, error) {
	result, err := getObject(client, hco)
	if err != nil {
		return nil, err
	}
	return result.(*hcov1alpha1.HyperConverged), nil
}

func reconcileRequest(namespace string) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Name: namespace}}
}

func createClient(objs ...runtime.Object) realClient.Client {
	return fakeClient.NewFakeClientWithScheme(scheme.Scheme, objs...)
}

func createHyperConverged(name, uid string) *hcov1alpha1.HyperConverged {
	return &hcov1alpha1.HyperConverged{ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(uid)}}
}

func createSCC() *secv1.SecurityContextConstraints {
	return &secv1.SecurityContextConstraints{ObjectMeta: metav1.ObjectMeta{Name: "privileged"}}
}

func createArgs() *args {
	hco := createHyperConverged("hco", "good uid")
	scc := createSCC()
	client := createClient(hco, scc)
	reconciler := createReconciler(client)

	return &args{
		hco:        hco,
		scc:        scc,
		client:     client,
		reconciler: reconciler,
	}
}

func createReconciler(client realClient.Client) *ReconcileHyperConverged {
	return &ReconcileHyperConverged{
		client: client,
		scheme: scheme.Scheme,
	}
}

func doReconcile(args *args) {
	result, err := args.reconciler.Reconcile(reconcileRequest(args.hco.Name))
	Expect(err).ToNot(HaveOccurred())
	Expect(result.Requeue).To(BeFalse())

	args.hco, err = getHyperConverged(args.client, args.hco)
	Expect(err).ToNot(HaveOccurred())
}
