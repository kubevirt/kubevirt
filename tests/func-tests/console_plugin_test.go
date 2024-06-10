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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	openshiftConsoleNamespace         = "openshift-console"
	expectedKubevirtConsolePluginName = "kubevirt-plugin"
)

var _ = Describe("kubevirt console plugin", Label(tests.OpenshiftLabel, "consolePlugin"), func() {

	var (
		cli          client.Client
		k8sClientSet *kubernetes.Clientset
		ctx          context.Context
	)

	tests.FlagParse()

	BeforeEach(func() {
		cli = tests.GetControllerRuntimeClient()

		ctx = context.Background()
		tests.FailIfNotOpenShift(ctx, cli, "kubevirt console plugin")

		hco := tests.GetHCO(ctx, cli)
		originalInfra := hco.Spec.Infra

		k8sClientSet = tests.GetK8sClientSet()

		DeferCleanup(func() {
			hco.Spec.Infra = originalInfra
			tests.UpdateHCORetry(ctx, cli, hco)
		})

	})

	It("console should reach kubevirt-plugin manifests", func() {
		kubevirtPlugin := &consolev1.ConsolePlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name: expectedKubevirtConsolePluginName,
			},
		}

		Expect(cli.Get(ctx, client.ObjectKeyFromObject(kubevirtPlugin), kubevirtPlugin)).To(Succeed())

		pluginServiceName := kubevirtPlugin.Spec.Backend.Service.Name
		pluginServicePort := kubevirtPlugin.Spec.Backend.Service.Port

		consolePods := &corev1.PodList{}
		Expect(cli.List(ctx, consolePods, client.MatchingLabels{
			"app":       "console",
			"component": "ui",
		}, client.InNamespace(openshiftConsoleNamespace))).To(Succeed())

		Expect(consolePods.Items).ToNot(BeEmpty())

		testConsolePod := consolePods.Items[0]
		command := fmt.Sprintf(`curl -ks https://%s.%s.svc:%d/plugin-manifest.json`,
			pluginServiceName, tests.InstallNamespace, pluginServicePort)

		stdout, stderr, err := executeCommandOnPod(ctx, k8sClientSet, &testConsolePod, command)
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
			consoleUIDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(hcoutil.AppComponentUIPlugin),
					Namespace: tests.InstallNamespace,
				},
			}

			g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(consoleUIDeployment), consoleUIDeployment)).To(Succeed())

			g.Expect(consoleUIDeployment.Spec.Template.Spec.NodeSelector).To(Equal(expectedNodeSelector))
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())

		Eventually(func(g Gomega) {
			proxyUIDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(hcoutil.AppComponentUIProxy),
					Namespace: tests.InstallNamespace,
				},
			}
			g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(proxyUIDeployment), proxyUIDeployment)).To(Succeed())
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
			consoleUIDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(hcoutil.AppComponentUIPlugin),
					Namespace: tests.InstallNamespace,
				},
			}

			g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(consoleUIDeployment), consoleUIDeployment)).To(Succeed())
			g.Expect(consoleUIDeployment.Spec.Template.Spec.NodeSelector).To(BeEmpty())
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())

		Eventually(func(g Gomega) {
			proxyUIDeployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      string(hcoutil.AppComponentUIProxy),
					Namespace: tests.InstallNamespace,
				},
			}
			g.Expect(cli.Get(ctx, client.ObjectKeyFromObject(proxyUIDeployment), proxyUIDeployment)).To(Succeed())
			g.Expect(proxyUIDeployment.Spec.Template.Spec.NodeSelector).To(BeEmpty())
		}).WithTimeout(1 * time.Minute).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())
	})
})

func executeCommandOnPod(ctx context.Context, k8scli *kubernetes.Clientset, pod *corev1.Pod, command string) (string, string, error) {
	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := k8scli.CoreV1().RESTClient().
		Post().
		Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command: []string{"/bin/sh", "-c", command},
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(tests.GetClientConfig(), "POST", request.URL())
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
