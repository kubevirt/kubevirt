/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package operator

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	jsonpatch "gopkg.in/evanphx/json-patch.v4"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libconfigmap"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	virtApiDepName                = "virt-api"
	virtControllerDepName         = "virt-controller"
	virtTemplateApiserverDepName  = "virt-template-apiserver"
	virtTemplateControllerDepName = "virt-template-controller"
	secondaryNetworkName          = "secondarynet"
)

func patchCRD(orig *extv1.CustomResourceDefinition, modified *extv1.CustomResourceDefinition) []byte {
	origCRDByte, err := json.Marshal(orig)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	crdByte, err := json.Marshal(modified)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	patch, err := jsonpatch.CreateMergePatch(origCRDByte, crdByte)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return patch
}

// prometheusRuleEnabled returns true if the PrometheusRule CRD is enabled
// and false otherwise.
func prometheusRuleEnabled() bool {
	virtClient := kubevirt.Client()

	prometheusRuleEnabled, err := util.IsPrometheusRuleEnabled(virtClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Unable to verify PrometheusRule CRD")

	return prometheusRuleEnabled
}

func serviceMonitorEnabled() bool {
	virtClient := kubevirt.Client()

	serviceMonitorEnabled, err := util.IsServiceMonitorEnabled(virtClient)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Unable to verify ServiceMonitor CRD")

	return serviceMonitorEnabled
}

// verifyOperatorWebhookCertificate can be used when inside tests doing reinstalls of kubevirt, to ensure that virt-operator already got the new certificate.
// This is necessary, since it can take up to a minute to get the fresh certificates when secrets are updated.
func verifyOperatorWebhookCertificate() {
	caBundle, _ := libinfra.GetBundleFromConfigMap(context.Background(), components.KubeVirtCASecretName)
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(caBundle)
	// ensure that the state is fully restored before each test
	Eventually(func() error {
		currentCert, err := libpod.GetCertsForPods(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-operator"), flags.KubeVirtInstallNamespace, "8444")
		Expect(err).ToNot(HaveOccurred())
		crt, err := x509.ParseCertificate(currentCert[0])
		Expect(err).ToNot(HaveOccurred())
		_, err = crt.Verify(x509.VerifyOptions{
			Roots: certPool,
		})
		return err
	}, 90*time.Second, 1*time.Second).Should(Not(HaveOccurred()), "bundle and certificate are still not in sync after 90 seconds")
	// we got the first pod with the new certificate, now let's wait until every pod sees it
	// this can take additional time since nodes are not synchronizing at the same moment
	libinfra.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-operator"), flags.KubeVirtInstallNamespace, "8444")
}

func atLeastOnePendingPodExistInDeployment(deploymentName string) bool {
	virtClient := kubevirt.Client()
	pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("kubevirt.io=%s", deploymentName),
			FieldSelector: fields.ParseSelectorOrDie("status.phase=Pending").String(),
		})
	Expect(err).ShouldNot(HaveOccurred())
	if len(pods.Items) == 0 {
		return false
	}
	return true
}

func nodeSelectorExistInDeployment(deploymentName string, labelKey string, labelValue string) bool {
	virtClient := kubevirt.Client()
	deployment, err := virtClient.AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if deployment.Spec.Template.Spec.NodeSelector == nil || deployment.Spec.Template.Spec.NodeSelector[labelKey] != labelValue {
		return false
	}
	return true
}

func expectDeploymentsToExist(deployments ...string) {
	Eventually(func() error {
		for _, deployment := range deployments {
			virtClient := kubevirt.Client()
			namespace := flags.KubeVirtInstallNamespace
			_, err := virtClient.AppsV1().Deployments(namespace).Get(context.Background(), deployment, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	}).WithTimeout(15 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
}

func sanityCheckDeploymentsExist() {
	expectDeploymentsToExist(virtApiDepName, virtControllerDepName)
}

func sanityCheckVirtTemplateDeploymentsExist() {
	expectDeploymentsToExist(virtTemplateApiserverDepName, virtTemplateControllerDepName)
}

func eventuallyVirtTemplateDeploymentsNotFound() {
	eventuallyDeploymentNotFound(virtTemplateApiserverDepName)
	eventuallyDeploymentNotFound(virtTemplateControllerDepName)
}

func checkVirtComponents(namespace string, imagePullSecrets []k8sv1.LocalObjectReference) {
	GinkgoHelper()
	virtClient := kubevirt.Client()
	vc, err := virtClient.AppsV1().Deployments(namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(vc.Spec.Template.Spec.ImagePullSecrets).To(Equal(imagePullSecrets))

	va, err := virtClient.AppsV1().Deployments(namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(va.Spec.Template.Spec.ImagePullSecrets).To(Equal(imagePullSecrets))

	vh, err := virtClient.AppsV1().DaemonSets(namespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	Expect(vh.Spec.Template.Spec.ImagePullSecrets).To(Equal(imagePullSecrets))

	if len(imagePullSecrets) == 0 {
		Expect(vh.Spec.Template.Spec.Containers).To(HaveLen(1))
	} else {
		Expect(vh.Spec.Template.Spec.Containers).To(HaveLen(2))
		Expect(vh.Spec.Template.Spec.Containers[1].Name).To(Equal("virt-launcher-image-holder"))
	}
}

func checkVirtLauncherPod(vmi *v1.VirtualMachineInstance) {
	GinkgoHelper()
	virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).NotTo(HaveOccurred())

	serviceAccount, err := kubevirt.Client().CoreV1().ServiceAccounts(vmi.Namespace).Get(context.Background(), virtLauncherPod.Spec.ServiceAccountName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	Expect(equality.Semantic.DeepEqual(virtLauncherPod.Spec.ImagePullSecrets, serviceAccount.ImagePullSecrets)).To(BeTrue())
}

func reinstallKubeVirt(kv *v1.KubeVirt, timeout time.Duration) {
	GinkgoHelper()
	deleteAllKvAndWait(false, kv.Name)
	eventuallyDeploymentNotFound(virtApiDepName)
	eventuallyDeploymentNotFound(virtControllerDepName)
	createKv(kv)
	testsuite.EnsureKubevirtReadyWithTimeout(kv, timeout)
	allKvInfraPodsAreReady(kv)
	sanityCheckDeploymentsExist()
}

func updateConfigAndWait(config v1.KubeVirtConfiguration) {
	GinkgoHelper()
	kvconfig.UpdateKubeVirtConfigValueAndWait(config)
	testsuite.EnsureKubevirtReady()
}

func defaultDeployment() {
	GinkgoHelper()
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	kv.Spec.Configuration.CommonInstancetypesDeployment = nil
	updateConfigAndWait(kv.Spec.Configuration)
}

func enableDeployment() {
	GinkgoHelper()
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	kv.Spec.Configuration.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
		Enabled: pointer.P(true),
	}
	updateConfigAndWait(kv.Spec.Configuration)
}

func disableDeployment() {
	GinkgoHelper()
	kv := libkubevirt.GetCurrentKv(kubevirt.Client())
	kv.Spec.Configuration.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
		Enabled: pointer.P(false),
	}
	updateConfigAndWait(kv.Spec.Configuration)
}

func expectResourcesToExist(labelSelector string) {
	GinkgoHelper()
	virtClient := kubevirt.Client()
	instancetypes, err := virtClient.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(instancetypes.Items).ToNot(BeEmpty())

	preferences, err := virtClient.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(preferences.Items).ToNot(BeEmpty())
}

func expectResourcesToNotExist(labelSelector string) {
	GinkgoHelper()
	virtClient := kubevirt.Client()
	instancetypes, err := virtClient.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(instancetypes.Items).To(BeEmpty())

	preferences, err := virtClient.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).ToNot(HaveOccurred())
	Expect(preferences.Items).To(BeEmpty())
}

func copyOriginalKv(originalKv *v1.KubeVirt) *v1.KubeVirt {
	newKv := &v1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:        originalKv.Name,
			Namespace:   originalKv.Namespace,
			Labels:      originalKv.ObjectMeta.Labels,
			Annotations: originalKv.ObjectMeta.Annotations,
		},
		Spec: *originalKv.Spec.DeepCopy(),
	}

	return newKv
}

func createKv(newKv *v1.KubeVirt) {
	Eventually(func() error {
		_, err := kubevirt.Client().KubeVirt(newKv.Namespace).Create(context.Background(), newKv, metav1.CreateOptions{})
		return err
	}).WithTimeout(10 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
}

func eventuallyDeploymentNotFound(name string) {
	Eventually(func() error {
		_, err := kubevirt.Client().AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
		return err
	}).WithTimeout(15 * time.Second).WithPolling(1 * time.Second).Should(MatchError(errors.IsNotFound, "not found error"))
}

func allKvInfraPodsAreReady(kv *v1.KubeVirt) {
	Eventually(func(g Gomega) error {

		curKv, err := kubevirt.Client().KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if curKv.Status.TargetDeploymentID != curKv.Status.ObservedDeploymentID {
			return fmt.Errorf("target and observed id don't match")
		}

		foundReadyAndOwnedPod := false
		pods, err := kubevirt.Client().CoreV1().Pods(curKv.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io,app.kubernetes.io/managed-by in (virt-operator, kubevirt-operator)"})
		if err != nil {
			return err
		}

		for _, pod := range pods.Items {
			if pod.Status.Phase != k8sv1.PodRunning {
				return fmt.Errorf("waiting for pod %s with phase %s to reach Running phase", pod.Name, pod.Status.Phase)
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				if !containerStatus.Ready {
					return fmt.Errorf("waiting for pod %s to have all containers in Ready state", pod.Name)
				}
			}

			id, ok := pod.Annotations[v1.InstallStrategyIdentifierAnnotation]
			if !ok {
				return fmt.Errorf("pod %s is owned by operator but has no id annotation", pod.Name)
			}

			expectedID := curKv.Status.ObservedDeploymentID
			if id != expectedID {
				return fmt.Errorf("pod %s is of version %s when we expected id %s", pod.Name, id, expectedID)
			}
			foundReadyAndOwnedPod = true
		}

		// this just sanity checks that at least one pod was found and verified.
		// false would indicate our labeling was incorrect.
		g.Expect(foundReadyAndOwnedPod).To(BeTrue(), "no ready and owned pod was found. Check if the labeling was incorrect")

		return nil
	}).WithTimeout(300 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
}

func waitForUpdateCondition(kv *v1.KubeVirt) {
	Eventually(func(g Gomega) *v1.KubeVirt {
		foundKV, err := kubevirt.Client().KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())

		return foundKV
	}).WithTimeout(120 * time.Second).WithPolling(1 * time.Second).Should(
		SatisfyAll(
			matcher.HaveConditionTrue(v1.KubeVirtConditionAvailable),
			matcher.HaveConditionTrue(v1.KubeVirtConditionProgressing),
			matcher.HaveConditionTrue(v1.KubeVirtConditionDegraded),
		),
	)
}

func patchKV(name string, patches *patch.PatchSet) {
	data, err := patches.GeneratePayload()
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		_, err := kubevirt.Client().KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), name, types.JSONPatchType, data, metav1.PatchOptions{})

		return err
	}).WithTimeout(10 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
}

var (
	imageTagRegEx = regexp.MustCompile(`^(.+)/(.+)(:.+)$`)
)

func parseImage(image string) (registry, imageName, version string) {
	var getVersion func(matches [][]string) string
	var imageRegEx *regexp.Regexp

	imageRegEx = imageTagRegEx
	getVersion = func(matches [][]string) string { return matches[0][3] }

	matches := imageRegEx.FindAllStringSubmatch(image, 1)
	Expect(matches).To(HaveLen(1))
	registry = matches[0][1]
	imageName = matches[0][2]
	version = getVersion(matches)

	return
}

func getDaemonsetImage(name string) string {
	var err error
	daemonSet, err := kubevirt.Client().AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	image := daemonSet.Spec.Template.Spec.Containers[0].Image
	imageRegEx := regexp.MustCompile(fmt.Sprintf("%s%s%s", `^(.*)/(.*)`, name, `([@:].*)?$`))
	matches := imageRegEx.FindAllStringSubmatch(image, 1)
	Expect(matches).To(HaveLen(1))
	Expect(matches[0]).To(HaveLen(4))

	return matches[0][2]
}

func getComponentConfigPatchOption(toChange *v1.ComponentConfig, orig *v1.ComponentConfig, path string) patch.PatchOption {
	var opt patch.PatchOption
	if toChange == nil {
		opt = patch.WithRemove(path)
	} else {
		if orig != nil {
			opt = patch.WithReplace(path, toChange)
		} else {
			opt = patch.WithAdd(path, toChange)
		}
	}
	return opt
}

func patchKVInfra(origKV *v1.KubeVirt, toChange *v1.ComponentConfig) error {
	return patchKVComponentConfig(origKV.Name, toChange, origKV.Spec.Infra, "/spec/infra")
}

func patchKVWorkloads(origKV *v1.KubeVirt, toChange *v1.ComponentConfig) error {
	return patchKVComponentConfig(origKV.Name, toChange, origKV.Spec.Workloads, "/spec/workloads")
}

func patchKVComponentConfig(kvName string, toChange, origField *v1.ComponentConfig, path string) error {
	opt := getComponentConfigPatchOption(toChange, origField, path)
	patches := patch.New(opt)
	data, err := patches.GeneratePayload()
	Expect(err).ToNot(HaveOccurred())

	_, err = kubevirt.Client().KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kvName, types.JSONPatchType, data, metav1.PatchOptions{})

	return err
}

func patchKvCertConfig(name string, certConfig *v1.KubeVirtSelfSignConfiguration) error {
	certRotationStrategy := v1.KubeVirtCertificateRotateStrategy{
		SelfSigned: certConfig,
	}

	data, err := patch.New(patch.WithReplace("/spec/certificateRotateStrategy", certRotationStrategy)).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())

	_, err = kubevirt.Client().KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), name, types.JSONPatchType, data, metav1.PatchOptions{})
	return err
}

func patchOperator(newImageName, version *string) bool {
	operator, oldImage, registry, oldImageName, oldVersion := parseOperatorImage()
	if newImageName == nil {
		// keep old prefix
		newImageName = &oldImageName
	}
	if version == nil {
		// keep old version
		version = &oldVersion
	} else {
		newVersion := components.AddVersionSeparatorPrefix(*version)
		version = &newVersion
	}
	newImage := fmt.Sprintf("%s/%s%s", registry, *newImageName, *version)

	if oldImage == newImage {
		return false
	}

	operator.Spec.Template.Spec.Containers[0].Image = newImage
	idx := -1
	var env k8sv1.EnvVar
	for idx, env = range operator.Spec.Template.Spec.Containers[0].Env {
		if env.Name == util.VirtOperatorImageEnvName {
			break
		}
	}
	Expect(idx).To(BeNumerically(">=", 0), "virt-operator image name environment variable is not found")

	path := fmt.Sprintf("/spec/template/spec/containers/0/env/%d/value", idx)
	op, err := patch.New(
		patch.WithReplace("/spec/template/spec/containers/0/image", newImage),
		patch.WithReplace(path, newImage),
	).GeneratePayload()
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		_, err = kubevirt.Client().AppsV1().Deployments(flags.KubeVirtInstallNamespace).Patch(context.Background(), "virt-operator", types.JSONPatchType, op, metav1.PatchOptions{})

		return err
	}).WithTimeout(15 * time.Second).WithPolling(time.Second).Should(Succeed())

	return true
}

func parseOperatorImage() (*appsv1.Deployment, string, string, string, string) {
	return parseDeployment("virt-operator")
}

func parseDeployment(name string) (*appsv1.Deployment, string, string, string, string) {
	var (
		err        error
		deployment *appsv1.Deployment
	)

	deployment, err = kubevirt.Client().AppsV1().Deployments(flags.KubeVirtInstallNamespace).Get(context.Background(), name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	image := deployment.Spec.Template.Spec.Containers[0].Image
	registry, imageName, version := parseImage(image)

	return deployment, image, registry, imageName, version
}

func createRunningVMIs(vmis []*v1.VirtualMachineInstance) []*v1.VirtualMachineInstance {
	newVMIs := make([]*v1.VirtualMachineInstance, len(vmis))
	for i, vmi := range vmis {
		var err error
		newVMIs[i], err = kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred(), "Create VMI successfully")
	}

	for i, vmi := range newVMIs {
		newVMIs[i] = libwait.WaitForSuccessfulVMIStart(vmi)
	}

	return newVMIs
}

func deleteVMIs(vmis []*v1.VirtualMachineInstance) {
	for _, vmi := range vmis {
		err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred(), "Delete VMI successfully")
	}
}

func deleteAllKvAndWait(ignoreOriginal bool, originalKvName string) {
	GinkgoHelper()

	virtClient := kubevirt.Client()
	Eventually(func(g Gomega) {
		kvs := libkubevirt.GetKvList(virtClient)

		deleteCount := 0
		for _, kv := range kvs {
			if ignoreOriginal && kv.Name == originalKvName {
				continue
			}
			deleteCount++
			if kv.DeletionTimestamp == nil {
				GinkgoLogr.Info("deleting the kv object", "namespace", kv.Namespace, "name", kv.Name)
				g.Expect(
					virtClient.KubeVirt(kv.Namespace).Delete(context.Background(), kv.Name, metav1.DeleteOptions{}),
				).To(Succeed())
			}
		}

		g.Expect(deleteCount).To(BeZero(), "still waiting on %d kvs to delete", deleteCount)
	}).WithTimeout(240 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
}

func verifyVMIsEvicted(vmis []*v1.VirtualMachineInstance) {
	Eventually(func() error {
		for _, vmi := range vmis {
			foundVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			if err == nil && !foundVMI.IsFinal() {
				return fmt.Errorf("waiting for vmi %s/%s to shutdown as part of update", foundVMI.Namespace, foundVMI.Name)
			} else if !errors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}, 320, 1).Should(Succeed(), "All VMIs should delete automatically")

}

func generateMigratableVMIs(num int) ([]*v1.VirtualMachineInstance, error) {
	virtClient := kubevirt.Client()

	var vmis []*v1.VirtualMachineInstance
	for range num {
		configMapName := "configmap-" + rand.String(5)
		secretName := "secret-" + rand.String(5)
		downwardAPIName := "downwardapi-" + rand.String(5)

		configData := map[string]string{
			"config1": "value1",
			"config2": "value2",
		}

		var err error
		cm := libconfigmap.New(configMapName, configData)
		_, err = virtClient.CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}

		secret := libsecret.New(secretName, libsecret.DataString{"user": "admin", "password": "community"})
		_, err = kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return nil, err
		}

		vmi := libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithConfigMapDisk(configMapName, configMapName),
			libvmi.WithSecretDisk(secretName, secretName),
			libvmi.WithServiceAccountDisk("default"),
			libvmi.WithDownwardAPIDisk(downwardAPIName),
			libvmi.WithWatchdog(v1.WatchdogActionPoweroff, libnode.GetArch()),
		)
		// In case there are no existing labels add labels to add some data to the downwardAPI disk
		if vmi.ObjectMeta.Labels == nil {
			vmi.ObjectMeta.Labels = map[string]string{"downwardTestLabelKey": "downwardTestLabelVal"}
		}

		vmis = append(vmis, vmi)
	}

	addSecondaryNetworkToLastVMI(vmis, secondaryNetworkName, "tenant-blue")

	return vmis, nil
}

func addSecondaryNetworkToLastVMI(vmis []*v1.VirtualMachineInstance, nadName, networkName string) {
	lastVMIIndex := len(vmis) - 1
	vmi := vmis[lastVMIIndex]

	libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName))(vmi)
	libvmi.WithNetwork(libvmi.MultusNetwork(networkName, nadName))(vmi)
}

func verifyVMIsUpdated(vmis []*v1.VirtualMachineInstance) {
	Eventually(func(g Gomega) {
		for _, vmi := range vmis {
			foundVMI, err := kubevirt.Client().VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())

			g.Expect(foundVMI.Status.MigrationState).ToNot(BeNil(), "waiting for vmi %s/%s to migrate as part of update", foundVMI.Namespace, foundVMI.Name)
			g.Expect(foundVMI.Status.MigrationState.Completed).To(BeTrue(), func() string {
				var startTime time.Time
				var endTime time.Time
				now := time.Now()

				if foundVMI.Status.MigrationState.StartTimestamp != nil {
					startTime = foundVMI.Status.MigrationState.StartTimestamp.Time
				}
				if foundVMI.Status.MigrationState.EndTimestamp != nil {
					endTime = foundVMI.Status.MigrationState.EndTimestamp.Time
				}

				return fmt.Sprintf("waiting for migration %s to complete for vmi %s/%s. Source Node [%s], Target Node [%s], Start Time [%s], End Time [%s], Now [%s], Failed: %t",
					string(foundVMI.Status.MigrationState.MigrationUID),
					foundVMI.Namespace,
					foundVMI.Name,
					foundVMI.Status.MigrationState.SourceNode,
					foundVMI.Status.MigrationState.TargetNode,
					startTime.String(),
					endTime.String(),
					now.String(),
					foundVMI.Status.MigrationState.Failed,
				)
			})

			g.Expect(foundVMI.Labels).ToNot(HaveKey(v1.OutdatedLauncherImageLabel),
				"waiting for vmi %s/%s to have update launcher image in status", foundVMI.Namespace, foundVMI.Name)
		}
	}).WithTimeout(500*time.Second).WithPolling(time.Second).Should(Succeed(), "All VMIs should update via live migration")

	// this is put in an eventually loop because it's possible for the VMI to complete
	// migrating and for the migration object to briefly lag behind in reporting
	// the results
	Eventually(func(g Gomega) {
		By("Verifying only a single successful migration took place for each vmi")
		migrationList, err := kubevirt.Client().VirtualMachineInstanceMigration(testsuite.GetTestNamespace(nil)).List(context.Background(), metav1.ListOptions{})
		g.Expect(err).ToNot(HaveOccurred(), "retrieving migrations")
		for _, vmi := range vmis {
			count := 0
			for _, migration := range migrationList.Items {
				if migration.Spec.VMIName == vmi.Name && migration.Status.Phase == v1.MigrationSucceeded {
					count++
				}
			}
			g.Expect(count).To(Equal(1), "vmi [%s] returned %d successful migrations", vmi.Name, count)
		}
	}).WithTimeout(10*time.Second).WithPolling(time.Second).Should(Succeed(), "Expects only a single successful migration per workload update")
}

func SIGSerial(text string, args ...interface{}) (extendedText string, newArgs []interface{}) {
	return decorators.SIG("[sig-operator]", "Operator "+text, decorators.SigOperator, Serial, args)
}
