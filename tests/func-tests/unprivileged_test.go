package tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"kubevirt.io/client-go/kubecli"
	flags "kubevirt.io/kubevirt/tests/flags"
	kvtutil "kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[rfe_id:393][crit:medium][vendor:cnv-qe@redhat.com][level:system]Unprivileged tests", func() {
	virtClient, err := kubecli.GetKubevirtClient()
	kvtutil.PanicOnError(err)

	// don't break other tests
	cfg := rest.CopyConfig(virtClient.Config())
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: "non-existent-user",
		Groups:   []string{"system:authenticated"},
	}

	unprivClient, err := kubecli.GetKubevirtClientFromRESTConfig(cfg)
	kvtutil.PanicOnError(err)

	It("[test_id:5676]should be able to read kubevirt-storage-class-defaults ConfigMap", func() {

		// Sanity check: can't read an arbitrary configmap (nonexistent)
		_, err = unprivClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.TODO(), "non-existent-configmap", metav1.GetOptions{})
		Expect(apierrors.IsForbidden(err)).To(BeTrue())

		configmap, err := unprivClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.TODO(), "kubevirt-storage-class-defaults", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(configmap.Data["local-sc.volumeMode"]).To(Equal("Filesystem"))
		Expect(configmap.Data["local-sc.accessMode"]).To(Equal("ReadWriteOnce"))
	})
})
