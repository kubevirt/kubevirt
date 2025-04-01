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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"dario.cat/mergo"
	"github.com/blang/semver/v4"
	"github.com/ghodss/yaml"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"
)

const (
	operatorName            = "kubevirt-hyperconverged-operator"
	CSVMode                 = "CSV"
	CRDMode                 = "CRDs"
	almExamplesAnnotation   = "alm-examples"
	validOutputModes        = CSVMode + "|" + CRDMode
	supported               = "supported"
	operatorFrameworkPrefix = "operatorframework.io/"
	mgImageAnnotation       = "operators.openshift.io/must-gather-image"
)

var (
	supportedArches = []string{"arch.amd64"}
	supportedOS     = []string{"os.linux"}
)

type EnvVarFlags []corev1.EnvVar

func (i *EnvVarFlags) String() string {
	es := make([]string, 0)
	for _, ev := range *i {
		es = append(es, fmt.Sprintf("%s=%s", ev.Name, ev.Value))
	}
	return strings.Join(es, ",")
}

func (i *EnvVarFlags) Set(value string) error {
	kv := strings.Split(value, "=")
	*i = append(*i, corev1.EnvVar{
		Name:  kv[0],
		Value: kv[1],
	})
	return nil
}

var (
	cwd, _              = os.Getwd()
	outputMode          = flag.String("output-mode", CSVMode, "Working mode: "+validOutputModes)
	cnaCsv              = flag.String("cna-csv", "", "Cluster Network Addons CSV string")
	virtCsv             = flag.String("virt-csv", "", "KubeVirt CSV string")
	sspCsv              = flag.String("ssp-csv", "", "Scheduling Scale Performance CSV string")
	cdiCsv              = flag.String("cdi-csv", "", "Containerized Data Importer CSV String")
	hppCsv              = flag.String("hpp-csv", "", "HostPath Provisioner Operator CSV String")
	aaqCsv              = flag.String("aaq-csv", "", "Applications Aware Quota Operator CSV String")
	operatorImage       = flag.String("operator-image-name", "", "HyperConverged Cluster Operator image")
	webhookImage        = flag.String("webhook-image-name", "", "HyperConverged Cluster Webhook image")
	cliDownloadsImage   = flag.String("cli-downloads-image-name", "", "Downloads Server image")
	kvUIPluginImage     = flag.String("kubevirt-consoleplugin-image-name", "", "KubeVirt Console Plugin image")
	kvUIProxyImage      = flag.String("kubevirt-consoleproxy-image-name", "", "KubeVirt Console Proxy image")
	kvVirtIOWinImage    = flag.String("kv-virtiowin-image-name", "", "KubeVirt VirtIO Win image")
	_                   = flag.String("primary-udn-binding-image-name", "", "deprecated. This flag is ignored")
	smbios              = flag.String("smbios", "", "Custom SMBIOS string for KubeVirt ConfigMap")
	machinetype         = flag.String("machinetype", "", "Custom MACHINETYPE string for KubeVirt ConfigMap (Deprecated, use amd64-machinetype)")
	amd64MachineType    = flag.String("amd64-machinetype", "", "Custom AMD64_MACHINETYPE string for KubeVirt ConfigMap")
	arm64MachineType    = flag.String("arm64-machinetype", "", "Custom ARM64_MACHINETYPE string for KubeVirt ConfigMap")
	csvVersion          = flag.String("csv-version", "", "CSV version")
	replacesCsvVersion  = flag.String("replaces-csv-version", "", "CSV version to replace")
	metadataDescription = flag.String("metadata-description", "", "One-Liner Description")
	specDescription     = flag.String("spec-description", "", "Description")
	specDisplayName     = flag.String("spec-displayname", "", "Display Name")
	namespace           = flag.String("namespace", "kubevirt-hyperconverged", "Namespace")
	crdDisplay          = flag.String("crd-display", "KubeVirt HyperConverged Cluster", "Label show in OLM UI about the primary CRD")
	csvOverrides        = flag.String("csv-overrides", "", "CSV like string with punctual changes that will be recursively applied (if possible)")
	visibleCRDList      = flag.String("visible-crds-list", "hyperconvergeds.hco.kubevirt.io,hostpathprovisioners.hostpathprovisioner.kubevirt.io",
		"Comma separated list of all the CRDs that should be visible in OLM console")
	relatedImagesList = flag.String("related-images-list", "",
		"Comma separated list of all the images referred in the CSV (just the image pull URLs or eventually a set of 'image|name' collations)")
	ignoreComponentsRelatedImages = flag.Bool("ignore-component-related-image", false, "Ignore relatedImages from components CSVs")
	crdDir                        = flag.String("crds-dir", "", "the directory containing the CRDs for apigroup validation. The validation will be performed if and only if the value is non-empty.")
	hcoKvIoVersion                = flag.String("hco-kv-io-version", "", "KubeVirt version")
	kubevirtVersion               = flag.String("kubevirt-version", "", "Kubevirt operator version")
	kvVirtLauncherOSVersion       = flag.String("virt-launcher-os-version", "", "Virt launcher OS version")
	cdiVersion                    = flag.String("cdi-version", "", "CDI operator version")
	cnaoVersion                   = flag.String("cnao-version", "", "CNA operator version")
	sspVersion                    = flag.String("ssp-version", "", "SSP operator version")
	hppoVersion                   = flag.String("hppo-version", "", "HPP operator version")
	aaqVersion                    = flag.String("aaq-version", "", "AAQ operator version")
	apiSources                    = flag.String("api-sources", cwd+"/...", "Project sources")
	enableUniqueSemver            = flag.Bool("enable-unique-version", false, "Insert a skipRange annotation to support unique semver in the CSV")
	skipsList                     = flag.String("skips-list", "",
		"Comma separated list of CSVs that can be skipped (read replaced) by this version")
	olmSkipRange = flag.String("olm-skip-range", "",
		"Semver range expression for CSVs that can be skipped (read replaced) by this version")
	mgImage        = flag.String("mg-image", "quay.io/kubevirt/must-gather", "Operator suggested must-gather image")
	testImagesNVRs = flag.String("test-images-nvrs", "", "Test Images NVRs")

	envVars EnvVarFlags
)

func IOReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := os.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, filepath.Join(root, file.Name()))
	}
	return files, nil
}

func validateNoAPIOverlap(crdDir string) error {
	crdFiles, err := IOReadDir(crdDir)
	if err != nil {
		return err
	}

	// crdMap is populated with operator names as keys and a slice of associated api groups as values.
	crdMap := getCrdMap(crdFiles)

	overlapsMap := detectAPIOverlap(crdMap)

	return checkAPIOverlapMap(overlapsMap)
}

func checkAPIOverlapMap(overlapsMap map[string]sets.Set[string]) error {
	// if at least one overlap found - emit an error.
	if len(overlapsMap) != 0 {
		var sb strings.Builder
		// WriteString always returns error=nil. no point to check it.
		_, _ = sb.WriteString("ERROR: Overlapping API Groups were found between different operators.\n")
		for apiGroup := range overlapsMap {
			_, _ = sb.WriteString(fmt.Sprintf("The API Group %s is being used by these operators: %s\n", apiGroup, strings.Join(overlapsMap[apiGroup].UnsortedList(), ", ")))
		}
		return errors.New(sb.String())
	}
	return nil
}

func detectAPIOverlap(crdMap map[string][]string) map[string]sets.Set[string] {
	// overlapsMap is populated with collisions found - API Groups as keys,
	// and slice containing operators using them, as values.
	overlapsMap := make(map[string]sets.Set[string])
	for operator, groups := range crdMap {
		for _, apiGroup := range groups {
			compareMapWithEntry(crdMap, operator, apiGroup, overlapsMap)
		}
	}
	return overlapsMap
}

func compareMapWithEntry(crdMap map[string][]string, operator string, apigroup string, overlapsMap map[string]sets.Set[string]) {
	for comparedOperator := range crdMap {
		if operator == comparedOperator { // don't check self
			continue
		}

		if slices.Contains(crdMap[comparedOperator], apigroup) {
			overlapsMap[apigroup].Insert(operator)
			overlapsMap[apigroup].Insert(comparedOperator)
		}
	}
}

func getCrdMap(crdFiles []string) map[string][]string {
	crdMap := make(map[string][]string)

	for _, crdFilePath := range crdFiles {
		file, err := os.Open(crdFilePath)
		panicOnError(err)

		content, err := io.ReadAll(file)
		panicOnError(err)

		err = file.Close()
		panicOnError(err)

		crdFileName := filepath.Base(crdFilePath)
		reg := regexp.MustCompile(`([^\d]+)`)
		operator := reg.FindString(crdFileName)

		var crd apiextensions.CustomResourceDefinition
		err = yaml.Unmarshal(content, &crd)
		panicOnError(err)

		if !slices.Contains(crdMap[operator], crd.Spec.Group) {
			crdMap[operator] = append(crdMap[operator], crd.Spec.Group)
		}
	}
	return crdMap
}

func main() {
	flag.Var(&envVars, "env-var", "HCO environment variable (key=value), may be used multiple times")

	flag.Parse()

	if webhookImage == nil || *webhookImage == "" {
		*webhookImage = *operatorImage
	}

	if *crdDir != "" {
		panicOnError(validateNoAPIOverlap(*crdDir))
		os.Exit(0)
	}

	if *enableUniqueSemver && *olmSkipRange != "" {
		panicOnError(errors.New("enable-unique-version and olm-skip-range cannot be used and the same time"))
		os.Exit(1)
	}

	switch *outputMode {
	case CRDMode:
		panicOnError(util.MarshallObject(components.GetOperatorCRD(*apiSources), os.Stdout))
	case CSVMode:
		getHcoCsv()

	default:
		panic("Unsupported output mode: " + *outputMode)
	}

}

func getHcoCsv() {
	if *specDisplayName == "" || *specDescription == "" {
		panic(errors.New("must specify spec-displayname and spec-description"))
	}

	componentsWithCsvs := getInitialCsvList()

	version := semver.MustParse(*csvVersion)
	replaces := getReplacesVersion()

	csvParams := getCsvBaseParams(replaces, version)

	// This is the basic CSV without an InstallStrategy defined
	csvBase := components.GetCSVBase(csvParams)

	if *enableUniqueSemver {
		csvBase.Annotations["olm.skipRange"] = fmt.Sprintf("<%v", version.String())
	} else if *olmSkipRange != "" {
		csvBase.Annotations["olm.skipRange"] = *olmSkipRange
	}

	params := getDeploymentParams()
	// This is the base deployment + rbac for the HCO CSV
	installStrategyBase := components.GetInstallStrategyBase(params)

	overwriteDeploymentSpecLabels(installStrategyBase.DeploymentSpecs, hcoutil.AppComponentDeployment)

	relatedImages := getRelatedImages()

	processCsvs(componentsWithCsvs, installStrategyBase, csvBase, &relatedImages)

	csvBase.Spec.RelatedImages = relatedImages

	if *skipsList != "" {
		csvBase.Spec.Skips = strings.Split(*skipsList, ",")
	}

	hiddenCRDsJ, err := getHiddenCrds(*csvBase)
	panicOnError(err)

	csvBase.Annotations["operators.operatorframework.io/internal-objects"] = hiddenCRDsJ

	// Update csv strategy.
	csvBase.Spec.InstallStrategy.StrategyName = "deployment"
	csvBase.Spec.InstallStrategy.StrategySpec = *installStrategyBase

	if *metadataDescription != "" {
		csvBase.Annotations["description"] = *metadataDescription
	}
	if *specDescription != "" {
		csvBase.Spec.Description = *specDescription
	}
	if *specDisplayName != "" {
		csvBase.Spec.DisplayName = *specDisplayName
	}
	if *mgImage != "" {
		csvBase.Annotations[mgImageAnnotation] = *mgImage
	}
	if *testImagesNVRs != "" {
		csvBase.Annotations["test-images-nvrs"] = *testImagesNVRs
	}

	setSupported(csvBase)

	applyOverrides(csvBase)

	csvBase.Spec.RelatedImages = sortRelatedImages(csvBase.Spec.RelatedImages)
	panicOnError(util.MarshallObject(csvBase, os.Stdout))
}

func getHiddenCrds(csvBase csvv1alpha1.ClusterServiceVersion) (string, error) {
	hiddenCrds := make([]string, 0)
	visibleCrds := strings.Split(*visibleCRDList, ",")
	for _, owned := range csvBase.Spec.CustomResourceDefinitions.Owned {
		if !slices.Contains(visibleCrds, owned.Name) {
			hiddenCrds = append(
				hiddenCrds,
				owned.Name,
			)
		}
	}

	hiddenCrdsJ, err := json.Marshal(hiddenCrds)
	if err != nil {
		return "", err
	}
	return string(hiddenCrdsJ), nil
}

func processCsvs(componentsWithCsvs []util.CsvWithComponent, installStrategyBase *csvv1alpha1.StrategyDetailsDeployment, csvBase *csvv1alpha1.ClusterServiceVersion, ris *[]csvv1alpha1.RelatedImage) {
	for _, c := range componentsWithCsvs {
		processOneCsv(c, installStrategyBase, csvBase, ris)
	}
}

func processOneCsv(c util.CsvWithComponent, installStrategyBase *csvv1alpha1.StrategyDetailsDeployment, csvBase *csvv1alpha1.ClusterServiceVersion, ris *[]csvv1alpha1.RelatedImage) {
	if c.Csv == "" {
		log.Panicf("ERROR: the %s CSV was empty", c.Name)
	}
	csvBytes := []byte(c.Csv)

	csvStruct := &csvv1alpha1.ClusterServiceVersion{}

	panicOnError(yaml.Unmarshal(csvBytes, csvStruct), "failed to unmarshal the CSV for", c.Name)

	strategySpec := csvStruct.Spec.InstallStrategy.StrategySpec

	overwriteDeploymentSpecLabels(strategySpec.DeploymentSpecs, c.Component)
	installStrategyBase.DeploymentSpecs = append(installStrategyBase.DeploymentSpecs, strategySpec.DeploymentSpecs...)

	installStrategyBase.ClusterPermissions = append(installStrategyBase.ClusterPermissions, strategySpec.ClusterPermissions...)
	installStrategyBase.Permissions = append(installStrategyBase.Permissions, strategySpec.Permissions...)

	csvBase.Spec.WebhookDefinitions = append(csvBase.Spec.WebhookDefinitions, csvStruct.Spec.WebhookDefinitions...)

	for _, owned := range csvStruct.Spec.CustomResourceDefinitions.Owned {
		csvBase.Spec.CustomResourceDefinitions.Owned = append(
			csvBase.Spec.CustomResourceDefinitions.Owned,
			newCRDDescription(owned),
		)
	}
	csvBaseAlmString := csvBase.Annotations[almExamplesAnnotation]
	csvStructAlmString := csvStruct.Annotations[almExamplesAnnotation]
	var baseAlmcrs []interface{}
	var structAlmcrs []interface{}

	if !strings.HasPrefix(csvBaseAlmString, "[") {
		csvBaseAlmString = "[" + csvBaseAlmString + "]"
	}

	panicOnError(json.Unmarshal([]byte(csvBaseAlmString), &baseAlmcrs), "failed to unmarshal the example from base from base csv for", c.Name, "csvBaseAlmString:", csvBaseAlmString)
	panicOnError(json.Unmarshal([]byte(csvStructAlmString), &structAlmcrs), "failed to unmarshal the example from base from struct csv for", c.Name, "csvStructAlmString:", csvStructAlmString)

	baseAlmcrs = append(baseAlmcrs, structAlmcrs...)
	almB, err := json.Marshal(baseAlmcrs)
	panicOnError(err, "failed to marshal the combined example for", c.Name)
	csvBase.Annotations[almExamplesAnnotation] = string(almB)

	if !*ignoreComponentsRelatedImages {
		for _, image := range csvStruct.Spec.RelatedImages {
			*ris = appendRelatedImageIfMissing(*ris, image)
		}
	}
}

func newCRDDescription(owned csvv1alpha1.CRDDescription) csvv1alpha1.CRDDescription {
	return csvv1alpha1.CRDDescription{
		Name:        owned.Name,
		Version:     owned.Version,
		Kind:        owned.Kind,
		Description: owned.Description,
		DisplayName: owned.DisplayName,
	}
}

func applyOverrides(csvBase *csvv1alpha1.ClusterServiceVersion) {
	if *csvOverrides != "" {
		csvOBytes := []byte(*csvOverrides)

		csvO := &csvv1alpha1.ClusterServiceVersion{}

		panicOnError(yaml.Unmarshal(csvOBytes, csvO))

		panicOnError(mergo.Merge(csvBase, csvO, mergo.WithOverride))
	}
}

func setSupported(csvBase *csvv1alpha1.ClusterServiceVersion) {
	if csvBase.Labels == nil {
		csvBase.Labels = make(map[string]string)
	}
	for _, ele := range supportedArches {
		csvBase.Labels[operatorFrameworkPrefix+ele] = supported
	}
	for _, ele := range supportedOS {
		csvBase.Labels[operatorFrameworkPrefix+ele] = supported
	}
}

func getInitialCsvList() []util.CsvWithComponent {
	return []util.CsvWithComponent{
		{
			Name:      "CNA",
			Csv:       *cnaCsv,
			Component: hcoutil.AppComponentNetwork,
		},
		{
			Name:      "KubeVirt",
			Csv:       *virtCsv,
			Component: hcoutil.AppComponentCompute,
		},
		{
			Name:      "SSP",
			Csv:       *sspCsv,
			Component: hcoutil.AppComponentSchedule,
		},
		{
			Name:      "CDI",
			Csv:       *cdiCsv,
			Component: hcoutil.AppComponentStorage,
		},
		{
			Name:      "HPP",
			Csv:       *hppCsv,
			Component: hcoutil.AppComponentStorage,
		},
		{
			Name:      "AAQ",
			Csv:       *aaqCsv,
			Component: hcoutil.AppComponentQuotaMngt,
		},
	}
}

func getReplacesVersion() string {
	if *replacesCsvVersion != "" {
		return fmt.Sprintf("%v.v%v", operatorName, semver.MustParse(*replacesCsvVersion).String())
	}
	return ""
}

func getRelatedImages() []csvv1alpha1.RelatedImage {
	var ris []csvv1alpha1.RelatedImage

	for _, image := range strings.Split(*relatedImagesList, ",") {
		if image != "" {
			ris = addRelatedImage(ris, image)
		}
	}
	return ris
}

func getCsvBaseParams(replaces string, version semver.Version) *components.CSVBaseParams {
	return &components.CSVBaseParams{
		Name:            operatorName,
		Namespace:       *namespace,
		DisplayName:     *specDisplayName,
		MetaDescription: *metadataDescription,
		Description:     *specDescription,
		Image:           *operatorImage,
		Replaces:        replaces,
		Version:         version,
		CrdDisplay:      *crdDisplay,
	}
}

func getDeploymentParams() *components.DeploymentOperatorParams {
	return &components.DeploymentOperatorParams{
		Namespace:              *namespace,
		Image:                  *operatorImage,
		WebhookImage:           *webhookImage,
		CliDownloadsImage:      *cliDownloadsImage,
		KVUIPluginImage:        *kvUIPluginImage,
		KVUIProxyImage:         *kvUIProxyImage,
		ImagePullPolicy:        "IfNotPresent",
		VirtIOWinContainer:     *kvVirtIOWinImage,
		Smbios:                 *smbios,
		Machinetype:            *machinetype,
		Amd64MachineType:       *amd64MachineType,
		Arm64MachineType:       *arm64MachineType,
		HcoKvIoVersion:         *hcoKvIoVersion,
		KubevirtVersion:        *kubevirtVersion,
		KvVirtLancherOsVersion: *kvVirtLauncherOSVersion,
		CdiVersion:             *cdiVersion,
		CnaoVersion:            *cnaoVersion,
		SspVersion:             *sspVersion,
		HppoVersion:            *hppoVersion,
		AaqVersion:             *aaqVersion,
		Env:                    envVars,
	}
}

func overwriteDeploymentSpecLabels(specs []csvv1alpha1.StrategyDeploymentSpec, component hcoutil.AppComponent) {
	for i := range specs {
		if specs[i].Label == nil {
			specs[i].Label = make(map[string]string)
		}
		if specs[i].Spec.Template.Labels == nil {
			specs[i].Spec.Template.Labels = make(map[string]string)
		}
		overwriteWithStandardLabels(specs[i].Spec.Template.Labels, *hcoKvIoVersion, component)
		overwriteWithStandardLabels(specs[i].Label, *hcoKvIoVersion, component)
	}

}

func overwriteWithStandardLabels(labels map[string]string, version string, component hcoutil.AppComponent) {
	labels[hcoutil.AppLabelManagedBy] = "olm"
	labels[hcoutil.AppLabelVersion] = version
	labels[hcoutil.AppLabelPartOf] = hcoutil.HyperConvergedCluster
	labels[hcoutil.AppLabelComponent] = string(component)
}

// add image to the slice. Ignore if the image already exists in the slice
func addRelatedImage(images []csvv1alpha1.RelatedImage, image string) []csvv1alpha1.RelatedImage {
	var ri csvv1alpha1.RelatedImage
	if strings.Contains(image, "|") {
		imageS := strings.Split(image, "|")
		ri.Image = imageS[0]
		ri.Name = imageS[1]
	} else {
		names := strings.Split(strings.Split(image, "@")[0], "/")
		ri.Name = names[len(names)-1]
		ri.Image = image
	}

	return appendRelatedImageIfMissing(images, ri)
}

func panicOnError(err error, info ...string) {
	if err != nil {
		moreInfo := ""
		if len(info) > 0 {
			moreInfo = strings.Join(info, " ")
		}

		log.Println("Error!", err, moreInfo)
		panic(err)
	}
}

func appendRelatedImageIfMissing(slice []csvv1alpha1.RelatedImage, ri csvv1alpha1.RelatedImage) []csvv1alpha1.RelatedImage {
	for _, ele := range slice {
		if ele.Image == ri.Image {
			return slice
		}
	}
	return append(slice, ri)
}

func sortRelatedImages(slice []csvv1alpha1.RelatedImage) []csvv1alpha1.RelatedImage {
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Name < slice[j].Name
	})
	return slice
}
