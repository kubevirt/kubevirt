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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package flags

import (
	"flag"
	"os"

	"kubevirt.io/client-go/kubecli"
)

var KubeVirtUtilityVersionTag = ""
var KubeVirtVersionTag = "latest"
var KubeVirtVersionTagAlt = ""
var KubeVirtUtilityRepoPrefix = ""
var KubeVirtRepoPrefix = "quay.io/kubevirt"
var ImagePrefixAlt = ""
var ContainerizedDataImporterNamespace = "cdi"
var KubeVirtKubectlPath = ""
var KubeVirtOcPath = ""
var KubeVirtVirtctlPath = ""
var KubeVirtGoCliPath = ""
var KubeVirtInstallNamespace string
var PreviousReleaseTag = ""
var PreviousReleaseRegistry = ""
var ConfigFile = ""
var SkipShasumCheck bool
var SkipDualStackTests bool
var IPV4ConnectivityCheckAddress = ""
var IPV6ConnectivityCheckAddress = ""
var ConnectivityCheckDNS = ""
var ArtifactsDir string
var ApplyDefaulte2eConfiguration bool

var DeployTestingInfrastructureFlag = false
var PathToTestingInfrastrucureManifests = ""

func init() {
	kubecli.Init()
	flag.StringVar(&KubeVirtUtilityVersionTag, "utility-container-tag", "", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtVersionTag, "container-tag", "latest", "Set the image tag or digest to use")
	flag.StringVar(&KubeVirtVersionTagAlt, "container-tag-alt", "", "An alternate tag that can be used to test operator deployments")
	flag.StringVar(&KubeVirtUtilityRepoPrefix, "utility-container-prefix", "", "Set the repository prefix for all images")
	flag.StringVar(&KubeVirtRepoPrefix, "container-prefix", KubeVirtRepoPrefix, "Set the repository prefix for all images")
	flag.StringVar(&ImagePrefixAlt, "image-prefix-alt", "", "Optional prefix for virt-* image names for additional imagePrefix operator test")
	flag.StringVar(&ContainerizedDataImporterNamespace, "cdi-namespace", "cdi", "Set the repository prefix for CDI components")
	flag.StringVar(&KubeVirtKubectlPath, "kubectl-path", "", "Set path to kubectl binary")
	flag.StringVar(&KubeVirtOcPath, "oc-path", "", "Set path to oc binary")
	flag.StringVar(&KubeVirtVirtctlPath, "virtctl-path", "", "Set path to virtctl binary")
	flag.StringVar(&KubeVirtGoCliPath, "gocli-path", "", "Set path to gocli binary")
	flag.StringVar(&KubeVirtInstallNamespace, "installed-namespace", "", "Set the namespace KubeVirt is installed in")
	flag.BoolVar(&DeployTestingInfrastructureFlag, "deploy-testing-infra", false, "Deploy testing infrastructure if set")
	flag.StringVar(&PathToTestingInfrastrucureManifests, "path-to-testing-infra-manifests", "manifests/testing", "Set path to testing infrastructure manifests")
	flag.StringVar(&PreviousReleaseTag, "previous-release-tag", "", "Set tag of the release to test updating from")
	flag.StringVar(&PreviousReleaseRegistry, "previous-release-registry", "quay.io/kubevirt", "Set registry of the release to test updating from")
	flag.StringVar(&ConfigFile, "config", "tests/default-config.json", "Path to a JSON formatted file from which the test suite will load its configuration. The path may be absolute or relative; relative paths start at the current working directory.")
	flag.StringVar(&ArtifactsDir, "artifacts", os.Getenv("ARTIFACTS"), "Directory for storing reporter artifacts like junit files or logs")
	flag.BoolVar(&SkipShasumCheck, "skip-shasums-check", false, "Skip tests with sha sums.")
	flag.BoolVar(&SkipDualStackTests, "skip-dual-stack-test", false, "Skip test that actively checks for the presence of IPv6 address in the cluster pods.")
	flag.StringVar(&IPV4ConnectivityCheckAddress, "conn-check-ipv4-address", "", "Address that is used for testing IPV4 connectivity to the outside world")
	flag.StringVar(&IPV6ConnectivityCheckAddress, "conn-check-ipv6-address", "", "Address that is used for testing IPV6 connectivity to the outside world")
	flag.StringVar(&ConnectivityCheckDNS, "conn-check-dns", "", "dns that is used for testing connectivity to the outside world")
	flag.BoolVar(&ApplyDefaulte2eConfiguration, "apply-default-e2e-configuration", false, "Apply the default e2e test configuration (feature gates, selinux contexts, ...)")
}

func NormalizeFlags() {
	// When the flags are not provided, copy the values from normal version tag and prefix
	if KubeVirtUtilityVersionTag == "" {
		KubeVirtUtilityVersionTag = KubeVirtVersionTag
	}

	if KubeVirtUtilityRepoPrefix == "" {
		KubeVirtUtilityRepoPrefix = KubeVirtRepoPrefix
	}
}
