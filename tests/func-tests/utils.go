package tests

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	"github.com/onsi/ginkgo/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"kubevirt.io/kubevirt/tests/flags"

	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"

	"kubevirt.io/client-go/kubecli"
	kvtutil "kubevirt.io/kubevirt/tests/util"
)

var KubeVirtStorageClassLocal string

const resource = "hyperconvergeds"

func init() {
	flag.StringVar(&KubeVirtStorageClassLocal, "storage-class-local", "local", "Storage provider to use for tests which want local storage")
}

// GetJobTypeEnvVar returns "JOB_TYPE" environment variable
func GetJobTypeEnvVar() string {
	return (os.Getenv("JOB_TYPE"))
}

func FlagParse() {
	flag.Parse()
}

func BeforeEach() {
	virtClient, err := kubecli.GetKubevirtClient()
	kvtutil.PanicOnError(err)

	kvtutil.PanicOnError(virtClient.RestClient().Delete().Namespace(kvtutil.NamespaceTestDefault).Resource("virtualmachines").Do(context.TODO()).Error())
	kvtutil.PanicOnError(virtClient.RestClient().Delete().Namespace(kvtutil.NamespaceTestDefault).Resource("virtualmachineinstances").Do(context.TODO()).Error())
	kvtutil.PanicOnError(virtClient.CoreV1().RESTClient().Delete().Namespace(kvtutil.NamespaceTestDefault).Resource("persistentvolumeclaims").Do(context.TODO()).Error())
}

func SkipIfNotOpenShift(cli kubecli.KubevirtClient, testName string) {
	isOpenShift, err := IsOpenShift(cli)
	kvtutil.PanicOnError(err)

	if !isOpenShift {
		ginkgo.Skip(fmt.Sprintf("Skipping %s tests when the cluster is not OpenShift", testName))
	}
}

type cacheIsOpenShift struct {
	isOpenShift bool
	hasSet      bool
	lock        sync.Mutex
}

func (c *cacheIsOpenShift) IsOpenShift(cli kubecli.KubevirtClient) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.hasSet {
		return c.isOpenShift, nil
	}

	s := scheme.Scheme
	_ = openshiftconfigv1.Install(s)
	s.AddKnownTypes(openshiftconfigv1.GroupVersion)

	clusterVersion := &openshiftconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}

	err := cli.RestClient().Get().
		Resource("clusterversions").
		Name("version").
		AbsPath("/apis", openshiftconfigv1.GroupVersion.Group, openshiftconfigv1.GroupVersion.Version).
		Timeout(10 * time.Second).
		Do(context.TODO()).Into(clusterVersion)

	if err == nil {
		c.isOpenShift = true
		c.hasSet = true
		return c.isOpenShift, nil
	}

	if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
		c.isOpenShift = false
		c.hasSet = true
		return c.isOpenShift, nil
	}

	return false, err
}

var isOpenShiftCache cacheIsOpenShift

func IsOpenShift(cli kubecli.KubevirtClient) (bool, error) {
	return isOpenShiftCache.IsOpenShift(cli)
}

// GetHCO reads the HCO CR from the APIServer with a DynamicClient
func GetHCO(ctx context.Context, client kubecli.KubevirtClient) *v1beta1.HyperConverged {
	hco := &v1beta1.HyperConverged{}

	hcoGVR := schema.GroupVersionResource{Group: v1beta1.SchemeGroupVersion.Group, Version: v1beta1.SchemeGroupVersion.Version, Resource: resource}

	unstHco, err := client.DynamicClient().Resource(hcoGVR).Namespace(flags.KubeVirtInstallNamespace).Get(ctx, hcoutil.HyperConvergedName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstHco.Object, hco)
	Expect(err).ToNot(HaveOccurred())

	return hco
}

// UpdateHCORetry updates the HCO CR in a safe way internally calling UpdateHCO
// UpdateHCORetry internally uses an async Eventually block refreshing the in-memory
// object if needed and setting there Spec, Annotations, Finalizers and Labels from the
// input object.
// UpdateHCORetry should be preferred over UpdateHCO to reduce test flakiness due to
// inevitable concurrency conflicts
func UpdateHCORetry(ctx context.Context, client kubecli.KubevirtClient, input *v1beta1.HyperConverged) *v1beta1.HyperConverged {
	var output *v1beta1.HyperConverged
	var err error

	Eventually(func() error {
		hco := GetHCO(ctx, client)
		input.Spec.DeepCopyInto(&hco.Spec)
		hco.ObjectMeta.Annotations = input.ObjectMeta.Annotations
		hco.ObjectMeta.Finalizers = input.ObjectMeta.Finalizers
		hco.ObjectMeta.Labels = input.ObjectMeta.Labels

		output, err = UpdateHCO(ctx, client, hco)
		return err
	}, 10*time.Second, time.Second).Should(Succeed())

	return output
}

// UpdateHCO updates the HCO CR using a DynamicClient, it can return errors on failures
func UpdateHCO(ctx context.Context, client kubecli.KubevirtClient, input *v1beta1.HyperConverged) (*v1beta1.HyperConverged, error) {
	hcoGVR := schema.GroupVersionResource{Group: input.GroupVersionKind().Group, Version: input.GroupVersionKind().Version, Resource: resource}
	hcoNamespace := input.Namespace

	unstructuredHco := &unstructured.Unstructured{}

	hco := GetHCO(ctx, client)
	input.Spec.DeepCopyInto(&hco.Spec)
	hco.ObjectMeta.Annotations = input.ObjectMeta.Annotations
	hco.ObjectMeta.Finalizers = input.ObjectMeta.Finalizers
	hco.ObjectMeta.Labels = input.ObjectMeta.Labels

	object, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&hco)
	if err != nil {
		return nil, err
	}
	unstructuredHco = &unstructured.Unstructured{Object: object}

	unstructuredHco, err = client.DynamicClient().Resource(hcoGVR).Namespace(hcoNamespace).Update(ctx, unstructuredHco, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	output := &v1beta1.HyperConverged{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredHco.Object, output)
	if err != nil {
		return nil, err
	}
	return output, nil
}
