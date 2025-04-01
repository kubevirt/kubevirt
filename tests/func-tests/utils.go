package tests

import (
	"context"
	"errors"
	"flag"
	"sync"
	"time"

	. "github.com/onsi/gomega" //nolint dot-imports
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var (
	KubeVirtStorageClassLocal string
	InstallNamespace          string
	cdiNS                     string
)

// labels
const (
	SingleNodeLabel             = "SINGLE_NODE_ONLY"
	HighlyAvailableClusterLabel = "HIGHLY_AVAILABLE_CLUSTER"
	OpenshiftLabel              = "OpenShift"

	TestNamespace = "hco-test-default"
)

func init() {
	flag.StringVar(&KubeVirtStorageClassLocal, "storage-class-local", "local", "Storage provider to use for tests which want local storage")
	flag.StringVar(&InstallNamespace, "installed-namespace", "", "Set the namespace KubeVirt is installed in")
	flag.StringVar(&cdiNS, "cdi-namespace", "", "ignored")
}

func FlagParse() {
	flag.Parse()
}

func BeforeEach(ctx context.Context) {
	cli := GetK8sClientSet().RESTClient()

	deleteAllResources(ctx, cli, "virtualmachines")
	deleteAllResources(ctx, cli, "virtualmachineinstances")
	deleteAllResources(ctx, cli, "persistentvolumeclaims")
}

func FailIfNotOpenShift(ctx context.Context, cli client.Client, testName string) {
	isOpenShift := false
	Eventually(func(ctx context.Context) error {
		var err error
		isOpenShift, err = IsOpenShift(ctx, cli)
		return err
	}).WithTimeout(10*time.Second).WithPolling(time.Second).WithContext(ctx).Should(Succeed(), "failed to check if running on an openshift cluster")

	ExpectWithOffset(1, isOpenShift).To(BeTrue(), `the %q test must run on openshift cluster. Use the "!%s" label filter in order to skip this test`, testName, OpenshiftLabel)
}

func FailIfSingleNodeCluster(singleWorkerCluster bool) {
	ExpectWithOffset(1, singleWorkerCluster).To(BeFalse(), `this test requires a highly available cluster; use the "!%s" label filter to skip this test`, HighlyAvailableClusterLabel)
}

func FailIfHighAvailableCluster(singleWorkerCluster bool) {
	ExpectWithOffset(1, singleWorkerCluster).To(BeTrue(), `this test requires a single worker cluster; use the "!%s" label filter to skip this test`, SingleNodeLabel)
}

type cacheIsOpenShift struct {
	isOpenShift bool
	hasSet      bool
	lock        sync.Mutex
}

func (c *cacheIsOpenShift) IsOpenShift(ctx context.Context, cli client.Client) (bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.hasSet {
		return c.isOpenShift, nil
	}

	err := openshiftconfigv1.AddToScheme(cli.Scheme())
	if err != nil {
		panic("can't register scheme; " + err.Error())
	}

	clusterVersion := &openshiftconfigv1.ClusterVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name: "version",
		},
	}

	err = cli.Get(ctx, client.ObjectKeyFromObject(clusterVersion), clusterVersion)
	if err == nil {
		c.isOpenShift = true
		c.hasSet = true
		return c.isOpenShift, nil
	}

	discoveryErr := &discovery.ErrGroupDiscoveryFailed{}
	if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) || errors.As(err, &discoveryErr) {
		c.isOpenShift = false
		c.hasSet = true
		return c.isOpenShift, nil
	}

	return false, err
}

var isOpenShiftCache cacheIsOpenShift

func IsOpenShift(ctx context.Context, cli client.Client) (bool, error) {
	return isOpenShiftCache.IsOpenShift(ctx, cli)
}

// GetHCO reads the HCO CR from the APIServer with a DynamicClient
func GetHCO(ctx context.Context, cli client.Client) *v1beta1.HyperConverged {
	Expect(v1beta1.AddToScheme(cli.Scheme())).To(Succeed())
	hco := &v1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcoutil.HyperConvergedName,
			Namespace: InstallNamespace,
		},
	}

	Expect(cli.Get(ctx, client.ObjectKeyFromObject(hco), hco)).To(Succeed())

	return hco
}

// UpdateHCORetry updates the HCO CR in a safe way internally calling UpdateHCO_old
// UpdateHCORetry internally uses an async Eventually block refreshing the in-memory
// object if needed and setting there Spec, Annotations, Finalizers and Labels from the
// input object.
// UpdateHCORetry should be preferred over UpdateHCO_old to reduce test flakiness due to
// inevitable concurrency conflicts
func UpdateHCORetry(ctx context.Context, cli client.Client, input *v1beta1.HyperConverged) *v1beta1.HyperConverged {
	var output *v1beta1.HyperConverged
	var err error

	Eventually(func(ctx context.Context) error {
		hco := GetHCO(ctx, cli)
		input.Spec.DeepCopyInto(&hco.Spec)
		hco.ObjectMeta.Annotations = input.ObjectMeta.Annotations
		hco.ObjectMeta.Finalizers = input.ObjectMeta.Finalizers
		hco.ObjectMeta.Labels = input.ObjectMeta.Labels

		output, err = UpdateHCO(ctx, cli, hco)
		return err
	}).WithTimeout(10 * time.Second).WithPolling(time.Second).WithContext(ctx).Should(Succeed())

	return output
}

// UpdateHCO updates the HCO CR using a DynamicClient, it can return errors on failures
func UpdateHCO(ctx context.Context, cli client.Client, input *v1beta1.HyperConverged) (*v1beta1.HyperConverged, error) {
	err := v1beta1.AddToScheme(cli.Scheme())
	if err != nil {
		return nil, err
	}

	hco := GetHCO(ctx, cli)
	input.Spec.DeepCopyInto(&hco.Spec)
	hco.Annotations = input.Annotations
	hco.Finalizers = input.Finalizers
	hco.Labels = input.Labels
	hco.Status = v1beta1.HyperConvergedStatus{} // to silence warning about unknown fields.

	err = cli.Update(ctx, hco)
	if err != nil {
		return nil, err
	}

	hco = GetHCO(ctx, cli)
	return hco, nil
}

// PatchHCO updates the HCO CR using a DynamicClient, it can return errors on failures
func PatchHCO(ctx context.Context, cli client.Client, patchBytes []byte) error {
	patch := client.RawPatch(types.JSONPatchType, patchBytes)
	hco := &v1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hcoutil.HyperConvergedName,
			Namespace: InstallNamespace,
		},
	}

	return cli.Patch(ctx, hco, patch)
}

func RestoreDefaults(ctx context.Context, cli client.Client) {
	Eventually(func(ctx context.Context) error {
		return PatchHCO(ctx, cli, []byte(`[{"op": "replace", "path": "/spec", "value": {}}]`))
	}).
		WithOffset(1).
		WithTimeout(time.Second * 5).
		WithPolling(time.Millisecond * 100).
		WithContext(ctx).
		Should(Succeed())
}

func deleteAllResources(ctx context.Context, restClient rest.Interface, resourceName string) {
	Eventually(func() bool {
		err := restClient.Delete().Namespace(TestNamespace).Resource(resourceName).Do(ctx).Error()
		return err == nil || apierrors.IsNotFound(err)
	}).WithTimeout(time.Minute).
		WithPolling(time.Second).
		Should(BeTrue())
}
