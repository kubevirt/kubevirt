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
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/coreos/go-semver/semver"
	"github.com/google/go-github/v83/github"

	k8sv1 "k8s.io/api/core/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/operator/resourcefiles"
	"kubevirt.io/kubevirt/tests/testsuite"
)

type vmSnapshotDef struct {
	vmSnapshotName  string
	yamlFile        string
	restoreName     string
	restoreYamlFile string
}

type vmYamlDefinition struct {
	apiVersion  string
	vmName      string
	yamlFile    string
	vmSnapshots []vmSnapshotDef
}

var _ = Describe(SIGSerial("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]should update kubevirt", decorators.Upgrade, func() {
	var (
		virtClient              kubecli.KubevirtClient
		originalKv              *v1.KubeVirt
		originalOperatorVersion string
		workDir                 string
		err                     error
	)

	runStrategyHalted := v1.RunStrategyHalted

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv = libkubevirt.GetCurrentKv(virtClient)
		_, _, _, _, version := parseOperatorImage()
		const prefix = ":"
		Expect(strings.HasPrefix(version, prefix)).To(BeTrue(), "version %s is expected to start with %s", version, prefix)
		originalOperatorVersion = strings.TrimPrefix(version, prefix)
		workDir = GinkgoT().TempDir()
		verifyOperatorWebhookCertificate()
	})

	AfterEach(func() {
		deleteAllKvAndWait(true, originalKv.Name)

		kvs := libkubevirt.GetKvList(virtClient)
		if len(kvs) == 0 {
			By("Re-creating the original KV to stabilize")
			createKv(copyOriginalKv(originalKv))
		}

		modified := patchOperator(nil, &originalOperatorVersion)
		if modified {
			waitForUpdateCondition(originalKv)
		}

		By("Waiting for original KV to stabilize")
		testsuite.EnsureKubevirtReadyWithTimeout(originalKv, 420*time.Second)
		allKvInfraPodsAreReady(originalKv)

		verifyOperatorWebhookCertificate()

		_, err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "disks-images-provider", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
	})

	// This test is installing a previous release of KubeVirt
	// running a VM/VMI using that previous release
	// Updating KubeVirt to the target tested code
	// Ensuring VM/VMI is still operational after the update from previous release.
	DescribeTable("[release-blocker][test_id:3145]from previous release to target tested release", func(updateOperator bool) {
		if !libstorage.HasCDI() {
			Fail("Fail update test when CDI is not present")
		}

		if updateOperator && flags.OperatorManifestPath == "" {
			Fail("operator manifest path must be configured for update tests")
		}

		// This test should run fine on single-node setups as long as no VM is created pre-update
		createVMs := true
		if !checks.HasAtLeastTwoNodes() {
			createVMs = false
		}

		var migratableVMIs []*v1.VirtualMachineInstance
		if createVMs {
			migratableVMIs, err = generateMigratableVMIs(2)
			Expect(err).NotTo(HaveOccurred())
		}
		previousImageTag := flags.PreviousReleaseTag
		previousImageRegistry := flags.PreviousReleaseRegistry
		if previousImageTag == "" {
			previousImageTag, err = detectLatestUpstreamOfficialTag()
			Expect(err).ToNot(HaveOccurred())
			By(fmt.Sprintf("By Using detected tag %s for previous kubevirt", previousImageTag))
		} else {
			By(fmt.Sprintf("By Using user defined tag %s for previous kubevirt", previousImageTag))
		}

		curVersion := originalKv.Status.ObservedKubeVirtVersion
		curRegistry := originalKv.Status.ObservedKubeVirtRegistry

		allKvInfraPodsAreReady(originalKv)
		sanityCheckDeploymentsExist()

		// Delete current KubeVirt install so we can install previous release.
		By("Deleting KubeVirt object")
		deleteAllKvAndWait(false, originalKv.Name)

		By("Verifying all infra pods have terminated")
		expectVirtOperatorPodsToTerminate(originalKv)

		By("Sanity Checking Deployments infrastructure is deleted")
		eventuallyDeploymentNotFound(virtApiDepName)
		eventuallyDeploymentNotFound(virtControllerDepName)

		if updateOperator {
			By("Deleting testing manifests")
			_, stderr, err := clientcmd.RunCommand(metav1.NamespaceNone, "kubectl", "delete", "-f", flags.TestingManifestPath)
			Expect(err).ToNot(HaveOccurred(), "failed to delete testing manifests: "+stderr)

			By("Deleting virt-operator installation")
			_, stderr, err = clientcmd.RunCommand(metav1.NamespaceNone, "kubectl", "delete", "-f", flags.OperatorManifestPath)
			Expect(err).ToNot(HaveOccurred(), "failed to delete virt-operator installation: "+stderr)

			By("Installing previous release of virt-operator")
			manifestURL := getUpstreamReleaseAssetURL(previousImageTag, "kubevirt-operator.yaml")
			installOperator(manifestURL)
		}

		// Install previous release of KubeVirt
		By("Creating KubeVirt object")
		kv := copyOriginalKv(originalKv)
		kv.Name = "kubevirt-release-install"
		kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate}

		// If updating via the KubeVirt CR, explicitly specify the desired release.
		if !updateOperator {
			kv.Spec.ImageTag = previousImageTag
			kv.Spec.ImageRegistry = previousImageRegistry
		}
		// For now disable upgrading the synchronization controller since no previous released versions
		// exist.
		updatedFeatureGates := make([]string, 0)
		featureGates := kv.Spec.Configuration.DeveloperConfiguration.FeatureGates
		for _, fg := range featureGates {
			if fg != featuregate.DecentralizedLiveMigration {
				updatedFeatureGates = append(updatedFeatureGates, fg)
			}
		}
		// Old releases don't support Beta-on-by-default, so Snapshot must be
		// explicitly listed for the previous release's webhook to accept
		// snapshot creation.
		updatedFeatureGates = append(updatedFeatureGates, featuregate.SnapshotGate)
		kv.Spec.Configuration.DeveloperConfiguration.FeatureGates = updatedFeatureGates

		k8sVersion, err := checks.GetKubernetesVersion()
		Expect(err).ToNot(HaveOccurred())
		if semver.New(k8sVersion).LessThan(*semver.New("1.35.0")) {
			kv.Spec.Configuration.DeveloperConfiguration.DisabledFeatureGates = append(
				kv.Spec.Configuration.DeveloperConfiguration.DisabledFeatureGates,
				featuregate.ImageVolume,
			)
		}

		kv.Spec.Configuration.DeveloperConfiguration.DisabledFeatureGates = append(
			kv.Spec.Configuration.DeveloperConfiguration.DisabledFeatureGates,
			featuregate.ExternalNetResourceInjection,
		)

		createKv(kv)

		// Wait for previous release to come online
		// wait 7 minutes because this test involves pulling containers
		// over the internet related to the latest kubevirt release
		By("Waiting for KV to stabilize")
		testsuite.EnsureKubevirtReadyWithTimeout(kv, 420*time.Second)
		//previousImageTag
		// TODO: find way to verify strategy job version as well
		pods, err := kubevirt.Client().CoreV1().Pods(kv.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io,app.kubernetes.io/managed-by=virt-operator"})
		Expect(err).ToNot(HaveOccurred())
		Expect(pods.Items).ToNot(BeEmpty())
		for _, pod := range pods.Items {
			Expect(pod.Spec.Containers[0].Image).To(ContainSubstring(previousImageTag))
			fmt.Println(pod.Spec.Containers[0].Image)
		}

		By("Verifying infrastructure is Ready")
		allKvInfraPodsAreReady(kv)
		sanityCheckDeploymentsExist()

		// kubectl API discovery cache only refreshes every 10 minutes
		// Since we're likely dealing with api additions/removals here, we
		// need to ensure we're using a different cache directory after
		// the update from the previous release occurs.
		oldClientCacheDir := filepath.Join(workDir, "oldclient")
		err = os.MkdirAll(oldClientCacheDir, 0755)
		Expect(err).ToNot(HaveOccurred())
		newClientCacheDir := filepath.Join(workDir, "newclient")
		err = os.MkdirAll(newClientCacheDir, 0755)
		Expect(err).ToNot(HaveOccurred())

		// Create VM on previous release using a specific API.
		// NOTE: we are testing with yaml here and explicitly _NOT_ generating
		// this vm using the latest api code. We want to guarantee there are no
		// surprises when it comes to backwards compatibility with previous
		// virt apis.  As we progress our api from v1alpha3 -> v1 there
		// needs to be a VM created for every api. This is how we will ensure
		// our api remains upgradable and supportable from previous release.

		var vmYamls map[string]*vmYamlDefinition
		if createVMs {
			vmYamls, err = generatePreviousVersionVmYamls(workDir)
			Expect(err).ToNot(HaveOccurred())
			Expect(generatePreviousVersionVmsnapshotYamls(vmYamls, workDir)).To(Succeed())
		}
		for _, vmYaml := range vmYamls {
			By(fmt.Sprintf("Creating VM with %s api", vmYaml.vmName))
			// NOTE: using kubectl to post yaml directly
			_, stderr, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), "kubectl", "create", "-f", vmYaml.yamlFile, "--cache-dir", oldClientCacheDir)
			Expect(err).ToNot(HaveOccurred(), stderr)

			for _, vmSnapshot := range vmYaml.vmSnapshots {
				By(fmt.Sprintf("Creating VM snapshot %s for vm %s", vmSnapshot.vmSnapshotName, vmYaml.vmName))
				_, stderr, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), "kubectl", "create", "-f", vmSnapshot.yamlFile, "--cache-dir", oldClientCacheDir)
				Expect(err).ToNot(HaveOccurred(), stderr)
			}

			By("Starting VM")
			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Start(context.Background(), vmYaml.vmName, &v1.StartOptions{})
			Expect(err).ToNot(HaveOccurred())

			By(fmt.Sprintf("Waiting for VM with %s api to become ready", vmYaml.apiVersion))

			Eventually(func() bool {
				virtualMachine, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if virtualMachine.Status.Ready {
					return true
				}
				return false
			}, 180*time.Second, 1*time.Second).Should(BeTrue())
		}

		netAttachDef := libnet.NewBridgeNetAttachDef(secondaryNetworkName, secondaryNetworkName)
		_, err = libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(migratableVMIs[0]), netAttachDef)
		Expect(err).NotTo(HaveOccurred())

		By("Starting multiple migratable VMIs before performing update")
		migratableVMIs = createRunningVMIs(migratableVMIs)

		// Update KubeVirt from the previous release to the testing target release.
		if updateOperator {
			By("Updating virt-operator installation")
			installOperator(flags.OperatorManifestPath)

			By("Re-installing testing manifests")
			_, stderr, err := clientcmd.RunCommand(metav1.NamespaceNone, "kubectl", "apply", "-f", flags.TestingManifestPath)
			Expect(err).ToNot(HaveOccurred(), "failed to re-install the testing manifests: "+stderr)
		} else {
			By("Updating KubeVirt object With current tag")
			patches := patch.New(
				patch.WithReplace("/spec/imageTag", curVersion),
				patch.WithReplace("/spec/imageRegistry", curRegistry),
			)

			patchKV(kv.Name, patches)
		}

		By("Wait for Updating Condition")
		waitForUpdateCondition(kv)

		By("Waiting for KV to stabilize")
		testsuite.EnsureKubevirtReadyWithTimeout(kv, 420*time.Second)

		By("Verifying infrastructure Is Updated")
		allKvInfraPodsAreReady(kv)

		// Verify console connectivity to VMI still works and stop VM
		for _, vmYaml := range vmYamls {
			By(fmt.Sprintf("Ensuring vm %s is ready and latest API annotation is set", vmYaml.apiVersion))
			Eventually(func() bool {
				// We are using our internal client here on purpose to ensure we can interact
				// with previously created objects that may have been created using a different
				// api version from the latest one our client uses.
				virtualMachine, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if !virtualMachine.Status.Ready {
					return false
				}

				if !controller.ObservedLatestApiVersionAnnotation(virtualMachine) {
					return false
				}

				return true
			}, 180*time.Second, 1*time.Second).Should(BeTrue())

			By(fmt.Sprintf("Ensure vm %s vmsnapshots exist and ready ", vmYaml.vmName))
			for _, snapshot := range vmYaml.vmSnapshots {
				Eventually(func() bool {
					vmSnapshot, err := virtClient.VirtualMachineSnapshot(testsuite.GetTestNamespace(nil)).Get(context.Background(), snapshot.vmSnapshotName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					if !(vmSnapshot.Status != nil && vmSnapshot.Status.ReadyToUse != nil && *vmSnapshot.Status.ReadyToUse) {
						return false
					}

					if vmSnapshot.Status.Phase != snapshotv1.Succeeded {
						return false
					}

					return true
				}, 120*time.Second, 3*time.Second).Should(BeTrue())
			}

			By(fmt.Sprintf("Connecting to %s's console", vmYaml.vmName))
			// This is in an eventually loop because it's possible for the
			// subresource endpoint routing to fail temporarily right after a deployment
			// completes while we wait for the kubernetes apiserver to detect our
			// subresource api server is online and ready to serve requests.
			Eventually(func() error {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return console.LoginToAlpine(vmi)
			}, 60*time.Second, 1*time.Second).Should(Succeed())

			By(fmt.Sprintf("Verifying firmware UUID for vm %s", vmYaml.vmName))
			Eventually(func(g Gomega) {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())

				vmFirmwareUUID := vm.Spec.Template.Spec.Domain.Firmware.UUID
				vmiFirmwareUUID := vmi.Spec.Domain.Firmware.UUID

				g.Expect(vmFirmwareUUID).ToNot(BeEmpty(), "expected firmware UUID in VM spec to be populated, but it's empty")
				g.Expect(vmFirmwareUUID).To(Equal(vmiFirmwareUUID), "firmware UUID mismatch: VM spec UUID does not match VMI UUID")
			}, 60*time.Second, 2*time.Second).Should(Succeed())

			By("Stopping VM")
			err = virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Stop(context.Background(), vmYaml.vmName, &v1.StopOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to stop")
			Eventually(func() error {
				_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				return err
			}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

			By("Ensuring we can Modify the VM Spec")
			Eventually(func() error {
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				if err != nil {
					return err
				}

				// by making a change to the VM, we ensure that writing the object is possible.
				// This ensures VMs created previously before the update are still compatible with our validation webhooks
				ops, err := patch.New(patch.WithAdd("/metadata/annotations/some-annotation", "some-val")).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, ops, metav1.PatchOptions{})
				return err
			}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

			By("Changing run strategy to halted to be able to restore the vm")
			patchBytes, err := patch.New(patch.WithAdd("/spec/runStrategy", &runStrategyHalted)).GeneratePayload()
			vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Patch(context.Background(), vmYaml.vmName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(*vm.Spec.RunStrategy).To(Equal(runStrategyHalted))

			By(fmt.Sprintf("Ensure vm %s can be restored from vmsnapshots", vmYaml.vmName))
			for _, snapshot := range vmYaml.vmSnapshots {
				_, stderr, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), "kubectl", "create", "-f", snapshot.restoreYamlFile, "--cache-dir", newClientCacheDir)
				Expect(err).ToNot(HaveOccurred(), stderr)
				Eventually(func() bool {
					r, err := virtClient.VirtualMachineRestore(testsuite.GetTestNamespace(nil)).Get(context.Background(), snapshot.restoreName, metav1.GetOptions{})
					if err != nil {
						return false
					}
					return r.Status != nil && r.Status.Complete != nil && *r.Status.Complete
				}, 180*time.Second, 3*time.Second).Should(BeTrue())
			}

			By(fmt.Sprintf("Deleting VM with %s api", vmYaml.apiVersion))
			_, stderr, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), "kubectl", "delete", "-f", vmYaml.yamlFile, "--cache-dir", newClientCacheDir)
			Expect(err).ToNot(HaveOccurred(), stderr)

			By("Waiting for VM to be removed")
			Eventually(func() error {
				_, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(nil)).Get(context.Background(), vmYaml.vmName, metav1.GetOptions{})
				return err
			}, 90*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
		}

		By("Verifying all migratable vmi workloads are updated via live migration")
		verifyVMIsUpdated(migratableVMIs)

		if len(migratableVMIs) > 0 {
			By("Verifying that a once migrated VMI after an update can be migrated again")
			vmi := migratableVMIs[0]
			migration, err := virtClient.VirtualMachineInstanceMigration(testsuite.GetTestNamespace(vmi)).Create(context.Background(), libmigration.New(vmi.Name, vmi.Namespace), metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisMigration(migration), 180).Should(HaveSucceeded())
		}

		By("Deleting migratable VMIs")
		deleteVMIs(migratableVMIs)

		By("Deleting KubeVirt object")
		deleteAllKvAndWait(false, originalKv.Name)
	},
		Entry("by patching KubeVirt CR", false),
		Entry("by updating virt-operator", true),
	)
}))

func getUpstreamReleaseAssetURL(tag string, assetName string) string {
	client := github.NewClient(&http.Client{
		Timeout: 5 * time.Second,
	})
	var err error
	var release *github.RepositoryRelease

	Eventually(func() error {
		release, _, err = client.Repositories.GetReleaseByTag(context.Background(), "kubevirt", "kubevirt", tag)

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	for _, asset := range release.Assets {
		if asset.GetName() == assetName {
			return asset.GetBrowserDownloadURL()
		}
	}

	Fail(fmt.Sprintf("Asset %s not found in release %s of kubevirt upstream repo", assetName, tag))
	return ""
}

func detectLatestUpstreamOfficialTag() (string, error) {
	client := github.NewClient(&http.Client{
		Timeout: 5 * time.Second,
	})

	var err error
	var releases []*github.RepositoryRelease

	Eventually(func() error {
		releases, _, err = client.Repositories.ListReleases(context.Background(), "kubevirt", "kubevirt", &github.ListOptions{PerPage: 10000})

		return err
	}, 10*time.Second, 1*time.Second).ShouldNot(HaveOccurred())

	var vs []*semver.Version

	for _, release := range releases {
		if *release.Draft ||
			*release.Prerelease ||
			len(release.Assets) == 0 {

			continue
		}
		tagName := strings.TrimPrefix(*release.TagName, "v")
		v, err := semver.NewVersion(tagName)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to parse latest release tag")
		vs = append(vs, v)
	}

	if len(vs) == 0 {
		return "", fmt.Errorf("no kubevirt releases found")
	}

	// descending order from most recent.
	sort.Sort(sort.Reverse(semver.Versions(vs)))

	// most recent tag
	tag := fmt.Sprintf("v%v", vs[0])

	// tag hint gives us information about the most recent tag in the current branch
	// this is executing in. We want to make sure we are using the previous most
	// recent official release from the branch we're in if possible. Note that this is
	// all best effort. If a tag hint can't be detected, we move on with the most
	// recent release from master.
	tagHint := strings.TrimPrefix(getTagHint(), "v")
	hint, err := semver.NewVersion(tagHint)

	if tagHint != "" && err == nil {
		for _, v := range vs {
			if v.LessThan(*hint) || v.Equal(*hint) {
				tag = fmt.Sprintf("v%v", v)
				By(fmt.Sprintf("Choosing tag %s influenced by tag hint %s", tag, tagHint))
				break
			}
		}
	}

	By(fmt.Sprintf("By detecting latest upstream official tag %s for current branch", tag))
	return tag, nil
}

func getTagHint() string {
	//git describe --tags --abbrev=0 "$(git rev-parse HEAD)"
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmdOutput, err := cmd.Output()
	if err != nil {
		return ""
	}

	cmd = exec.Command("git", "describe", "--tags", "--abbrev=0", strings.TrimSpace(string(cmdOutput)))
	cmdOutput, err = cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(strings.Split(string(cmdOutput), "-rc")[0])
}

func expectVirtOperatorPodsToTerminate(kv *v1.KubeVirt) {
	Eventually(func(g Gomega) {
		pods, err := kubevirt.Client().CoreV1().Pods(kv.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io,app.kubernetes.io/managed-by=virt-operator"})
		g.Expect(err).ToNot(HaveOccurred())

		for _, pod := range pods.Items {
			g.Expect(pod.Status.Phase).To(BeElementOf(k8sv1.PodFailed, k8sv1.PodSucceeded), "waiting for pod %s with phase %s to reach final phase", pod.Name, pod.Status.Phase)
		}
	}).WithTimeout(120 * time.Second).WithPolling(1 * time.Second).Should(Succeed())
}

func installOperator(manifestPath string) {
	// namespace is already hardcoded within the manifests
	_, stderr, err := clientcmd.RunCommand(metav1.NamespaceNone, "kubectl", "apply", "-f", manifestPath)
	Expect(err).ToNot(HaveOccurred(), stderr)

	By("Waiting for KubeVirt CRD to be created")
	ext, err := extclient.NewForConfig(kubevirt.Client().Config())
	Expect(err).ToNot(HaveOccurred())

	Eventually(func() error {
		_, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "kubevirts.kubevirt.io", metav1.GetOptions{})
		return err
	}).WithTimeout(60 * time.Second).WithPolling(1 * time.Second).ShouldNot(HaveOccurred())
}

func generatePreviousVersionVmYamls(workDir string) (map[string]*vmYamlDefinition, error) {
	virtClient := kubevirt.Client()
	vmYamls := make(map[string]*vmYamlDefinition)
	ext, err := extclient.NewForConfig(virtClient.Config())
	if err != nil {
		return nil, err
	}

	crd, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachines.kubevirt.io", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Generate a vm Yaml for every version supported in the currently deployed KubeVirt
	var supportedVersions []string
	for _, version := range crd.Spec.Versions {
		supportedVersions = append(supportedVersions, version.Name)
	}

	for i, version := range supportedVersions {
		yamlFileName := filepath.Join(workDir, fmt.Sprintf("vm-%s.yaml", version))

		err = resourcefiles.WriteFile(yamlFileName, resourcefiles.VMInfo{
			Version:   version,
			Index:     i,
			ImageName: cd.ContainerDiskFor(cd.ContainerDiskAlpine),
		})
		if err != nil {
			return nil, err
		}

		vmYamls[version] = &vmYamlDefinition{
			apiVersion: version,
			vmName:     "vm-" + version,
			yamlFile:   yamlFileName,
		}
	}

	return vmYamls, nil
}

func generatePreviousVersionVmsnapshotYamls(vmYamls map[string]*vmYamlDefinition, workDir string) error {
	virtClient := kubevirt.Client()
	ext, err := extclient.NewForConfig(virtClient.Config())
	if err != nil {
		return err
	}

	crd, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachinesnapshots.snapshot.kubevirt.io", metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Generate a vmsnapshot Yaml for every version
	// supported in the currently deployed KubeVirt
	// For every vm version
	var supportedVersions []string
	for _, version := range crd.Spec.Versions {
		supportedVersions = append(supportedVersions, version.Name)
	}

	for _, vmYaml := range vmYamls {
		var vmSnapshots []vmSnapshotDef
		for _, version := range supportedVersions {
			vmSnapshots, err = generateSnapshotsForVersion(vmYaml, version, workDir, vmSnapshots)
			if err != nil {
				return err
			}
		}

		vmYaml.vmSnapshots = vmSnapshots
	}

	return nil
}

func generateSnapshotsForVersion(vmYaml *vmYamlDefinition, version string, workDir string, vmSnapshots []vmSnapshotDef) ([]vmSnapshotDef, error) {
	snapshotName := fmt.Sprintf("vm-%s-snapshot-%s", vmYaml.apiVersion, version)
	snapshotYamlFileName := filepath.Join(workDir, fmt.Sprintf("%s.yaml", snapshotName))

	err := resourcefiles.WriteFile(
		snapshotYamlFileName,
		resourcefiles.SnapshotInfo{
			Version: version,
			Name:    snapshotName,
			VMName:  vmYaml.vmName,
		})
	if err != nil {
		return nil, err
	}

	restoreName := fmt.Sprintf("vm-%s-restore-%s", vmYaml.apiVersion, version)
	restoreYamlFileName := filepath.Join(workDir, fmt.Sprintf("%s.yaml", restoreName))
	err = resourcefiles.WriteFile(
		restoreYamlFileName,
		resourcefiles.RestoreInfo{
			Version:      version,
			Name:         restoreName,
			VMName:       vmYaml.vmName,
			SnapshotName: snapshotName,
		})
	if err != nil {
		return nil, err
	}

	vmSnapshots = append(vmSnapshots, vmSnapshotDef{
		vmSnapshotName:  snapshotName,
		yamlFile:        snapshotYamlFileName,
		restoreName:     restoreName,
		restoreYamlFile: restoreYamlFileName,
	})

	return vmSnapshots, nil
}
