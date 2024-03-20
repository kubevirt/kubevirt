package tests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	openshiftConsoleNamespace = "openshift-console"
)

var _ = Describe("kubevirt console plugin", func() {
	var (
		cli                               kubecli.KubevirtClient
		ctx                               context.Context
		expectedKubevirtConsolePluginName = "kubevirt-plugin"
		consoleGVR                        = schema.GroupVersionResource{
			Group:    "console.openshift.io",
			Version:  "v1",
			Resource: "consoleplugins",
		}
	)

	tests.FlagParse()

	BeforeEach(func() {
		var err error
		cli, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		tests.SkipIfNotOpenShift(cli, "kubevirt console plugin")
		ctx = context.Background()

		hco := tests.GetHCO(ctx, cli)
		originalInfra := hco.Spec.Infra
		DeferCleanup(func() {
			hco.Spec.Infra = originalInfra
			tests.UpdateHCORetry(ctx, cli, hco)
		})
	})

	It("console should reach kubevirt-plugin manifests", func() {
		unstructured, err := cli.DynamicClient().Resource(consoleGVR).Get(ctx, expectedKubevirtConsolePluginName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		kubevirtPlugin := &consolev1.ConsolePlugin{}
		Expect(runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, kubevirtPlugin)).To(Succeed())

		pluginServiceName := kubevirtPlugin.Spec.Backend.Service.Name
		pluginServicePort := kubevirtPlugin.Spec.Backend.Service.Port

		consolePodsLabelSelector := "app=console,component=ui"

		consolePods, err := cli.CoreV1().Pods(openshiftConsoleNamespace).List(ctx, metav1.ListOptions{
			LabelSelector: consolePodsLabelSelector,
		})
		Expect(err).ToNot(HaveOccurred())
		Expect(consolePods.Items).ToNot(BeEmpty())

		testConsolePod := consolePods.Items[0]
		command := fmt.Sprintf(`curl -ks https://%s.%s.svc:%d/plugin-manifest.json`,
			pluginServiceName, flags.KubeVirtInstallNamespace, pluginServicePort)

		stdout, stderr, err := executeCommandOnPod(ctx, cli, &testConsolePod, command)
		Expect(err).ToNot(HaveOccurred())
		Expect(stdout).ToNot(BeEmpty())
		Expect(stderr).To(BeEmpty())

		var pluginManifests map[string]interface{}
		err = json.Unmarshal([]byte(stdout), &pluginManifests)
		Expect(err).ToNot(HaveOccurred())

		pluginName := pluginManifests["name"]
		Expect(pluginName).To(Equal(expectedKubevirtConsolePluginName))
	})

	It("nodePlacement should be propagated from HyperConverged CR to console-plugin and apiserver-proxy Deployments", Serial, func() {

		expectedNodeSelector := map[string]string{
			"foo": "bar",
		}
		expectedNodeSelectorBytes, err := json.Marshal(expectedNodeSelector)
		Expect(err).ToNot(HaveOccurred())
		expectedNodeSelectorStr := string(expectedNodeSelectorBytes)
		addNodeSelectorPatch := []byte(fmt.Sprintf(`[{"op": "add", "path": "/spec/infra", "value": {"nodePlacement": {"nodeSelector": %s}}}]`, expectedNodeSelectorStr))

		Eventually(func() error {
			err = tests.PatchHCO(ctx, cli, addNodeSelectorPatch)
			return err
		}).WithTimeout(1 * time.Minute).
			WithPolling(1 * time.Millisecond).
			Should(Succeed())

		Eventually(func(g Gomega) {
			consoleUIDeployment, err := cli.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(ctx, string(hcoutil.AppComponentUIPlugin), metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(consoleUIDeployment.Spec.Template.Spec.NodeSelector).To(Equal(expectedNodeSelector))
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())

		Eventually(func(g Gomega) {
			proxyUIDeployment, err := cli.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(ctx, string(hcoutil.AppComponentUIProxy), metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(proxyUIDeployment.Spec.Template.Spec.NodeSelector).To(Equal(expectedNodeSelector))
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())

		// clear node placement from HyperConverged CR and verify the nodeSelector has been cleared as well from the UI Deployments
		removeNodeSelectorPatch := []byte(`[{"op": "replace", "path": "/spec/infra", "value": {}}]`)
		Eventually(func() error {
			err = tests.PatchHCO(ctx, cli, removeNodeSelectorPatch)
			return err
		}).WithTimeout(1 * time.Minute).
			WithPolling(1 * time.Millisecond).
			Should(Succeed())

		Eventually(func(g Gomega) {
			consoleUIDeployment, err := cli.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(ctx, string(hcoutil.AppComponentUIPlugin), metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(consoleUIDeployment.Spec.Template.Spec.NodeSelector).To(BeEmpty())
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())

		Eventually(func(g Gomega) {
			proxyUIDeployment, err := cli.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(ctx, string(hcoutil.AppComponentUIProxy), metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(proxyUIDeployment.Spec.Template.Spec.NodeSelector).To(BeEmpty())
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())
	})
})

func executeCommandOnPod(ctx context.Context, cli kubecli.KubevirtClient, pod *v1.Pod, command string) (string, string, error) {
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := cli.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: []string{"/bin/sh", "-c", command},
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(cli.Config(), "POST", request.URL())
	if err != nil {
		return "", "", fmt.Errorf("%w: failed to create pod executor for %v/%v", err, pod.Namespace, pod.Name)
	}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})
	if err != nil {
		return "", "", fmt.Errorf("%w Failed executing command %s on %v/%v", err, command, pod.Namespace, pod.Name)
	}
	return buf.String(), errBuf.String(), nil
}
