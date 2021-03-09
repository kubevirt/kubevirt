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
	"github.com/blang/semver"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/tools/util"
	"io/ioutil"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	operatorName          = "kubevirt-hyperconverged-operator"
	CSVMode               = "CSV"
	CRDMode               = "CRDs"
	almExamplesAnnotation = "alm-examples"
	validOutputModes      = CSVMode + "|" + CRDMode
)

type ClusterServiceVersionSpecExtended struct {
	csvv1alpha1.ClusterServiceVersionSpec
	RelatedImages []relatedImage `json:"relatedImages,omitempty"`
}

type ClusterServiceVersionExtended struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterServiceVersionSpecExtended       `json:"spec"`
	Status csvv1alpha1.ClusterServiceVersionStatus `json:"status"`
}

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
	nmoCsv              = flag.String("nmo-csv", "", "Node Maintenance Operator CSV String")
	hppCsv              = flag.String("hpp-csv", "", "HostPath Provisioner Operator CSV String")
	vmImportCsv         = flag.String("vmimport-csv", "", "Virtual Machine Import Operator CSV String")
	operatorImage       = flag.String("operator-image-name", "", "HyperConverged Cluster Operator image")
	webhookImage        = flag.String("webhook-image-name", "", "HyperConverged Cluster Webhook image")
	imsConversionImage  = flag.String("ims-conversion-image-name", "", "IMS conversion image")
	imsVMWareImage      = flag.String("ims-vmware-image-name", "", "IMS VMWare image")
	smbios              = flag.String("smbios", "", "Custom SMBIOS string for KubeVirt ConfigMap")
	machinetype         = flag.String("machinetype", "", "Custom MACHINETYPE string for KubeVirt ConfigMap")
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
	cdiVersion                    = flag.String("cdi-version", "", "CDI operator version")
	cnaoVersion                   = flag.String("cnao-version", "", "CNA operator version")
	sspVersion                    = flag.String("ssp-version", "", "SSP operator version")
	nmoVersion                    = flag.String("nmo-version", "", "NM operator version")
	hppoVersion                   = flag.String("hppo-version", "", "HPP operator version")
	vmImportVersion               = flag.String("vm-import-version", "", "VM-Import operator version")
	apiSources                    = flag.String("api-sources", cwd+"/...", "Project sources")
	enableUniqueSemver            = flag.Bool("enable-unique-version", false, "Insert a skipRange annotation to support unique semver in the CSV")
	envVars                       EnvVarFlags
)

func genHcoCrds() error {
	// Write out CRDs and CR
	if err := util.MarshallObject(components.GetOperatorCRD(*apiSources), os.Stdout); err != nil {
		return err
	}

	if err := util.MarshallObject(components.GetV2VCRD(), os.Stdout); err != nil {
		return err
	}

	if err := util.MarshallObject(components.GetV2VOvirtProviderCRD(), os.Stdout); err != nil {
		return err
	}

	return nil
}

func IOReadDir(root string) ([]string, error) {
	var files []string
	fileInfo, err := ioutil.ReadDir(root)
	if err != nil {
		return files, err
	}

	for _, file := range fileInfo {
		files = append(files, filepath.Join(root, file.Name()))
	}
	return files, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
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

func checkAPIOverlapMap(overlapsMap map[string][]string) error {
	// if at least one overlap found - emit an error.
	if len(overlapsMap) != 0 {
		var sb strings.Builder
		// WriteString always returns error=nil. no point to check it.
		_, _ = sb.WriteString("ERROR: Overlapping API Groups were found between different operators.\n")
		for apiGroup := range overlapsMap {
			_, _ = sb.WriteString(fmt.Sprintf("The API Group %s is being used by these operators: %s\n", apiGroup, strings.Join(overlapsMap[apiGroup], ", ")))
		}
		return errors.New(sb.String())
	}
	return nil
}

func detectAPIOverlap(crdMap map[string][]string) map[string][]string {
	// overlapsMap is populated with collisions found - API Groups as keys,
	// and slice containing operators using them, as values.
	overlapsMap := make(map[string][]string)
	for operator, groups := range crdMap {
		for _, apiGroup := range groups {
			// We work on replacement for current v2v. Remove this check when vmware import is removed
			if apiGroup == "v2v.kubevirt.io" {
				continue
			}

			compareMapWithEntry(crdMap, operator, apiGroup, overlapsMap)
		}
	}
	return overlapsMap
}

func compareMapWithEntry(crdMap map[string][]string, operator string, apigroup string, overlapsMap map[string][]string) {
	for comparedOperator := range crdMap {
		if operator == comparedOperator { // don't check self
			continue
		}

		if stringInSlice(apigroup, crdMap[comparedOperator]) {
			appendOnce(overlapsMap[apigroup], operator)
			appendOnce(overlapsMap[apigroup], comparedOperator)
		}
	}
}

func getCrdMap(crdFiles []string) map[string][]string {
	crdMap := make(map[string][]string)

	for _, crdFilePath := range crdFiles {
		file, err := os.Open(crdFilePath)
		panicOnError(err)

		content, err := ioutil.ReadAll(file)
		panicOnError(err)

		err = file.Close()
		panicOnError(err)

		crdFileName := filepath.Base(crdFilePath)
		reg := regexp.MustCompile(`([^\d]+)`)
		operator := reg.FindString(crdFileName)

		var crd apiextensions.CustomResourceDefinition
		err = yaml.Unmarshal(content, &crd)
		panicOnError(err)

		if !stringInSlice(crd.Spec.Group, crdMap[operator]) {
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

	switch *outputMode {
	case CRDMode:
		panicOnError(genHcoCrds())
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
		csvBase.ObjectMeta.Annotations["olm.skipRange"] = fmt.Sprintf(">=%v-1 <%v", strings.Split(version.String(), "-")[0], version.String())
	}

	csvExtended := ClusterServiceVersionExtended{
		TypeMeta:   csvBase.TypeMeta,
		ObjectMeta: csvBase.ObjectMeta,
		Spec:       ClusterServiceVersionSpecExtended{ClusterServiceVersionSpec: csvBase.Spec},
		Status:     csvBase.Status}

	params := getDeploymentParams()
	// This is the base deployment + rbac for the HCO CSV
	installStrategyBase := components.GetInstallStrategyBase(params)

	overwriteDeploymentSpecLabels(installStrategyBase.DeploymentSpecs, hcoutil.AppComponentDeployment)

	relatedImageSet := getRelatedImages()

	processCsvs(componentsWithCsvs, installStrategyBase, &csvExtended, relatedImageSet)

	csvExtended.Spec.RelatedImages = relatedImageSet.dump()

	hiddenCRDsJ, err := getHiddenCrds(csvExtended)
	panicOnError(err)

	csvExtended.Annotations["operators.operatorframework.io/internal-objects"] = hiddenCRDsJ

	// Update csv strategy.
	csvExtended.Spec.InstallStrategy.StrategyName = "deployment"
	csvExtended.Spec.InstallStrategy.StrategySpec = *installStrategyBase

	if *metadataDescription != "" {
		csvExtended.Annotations["description"] = *metadataDescription
	}
	if *specDescription != "" {
		csvExtended.Spec.Description = *specDescription
	}
	if *specDisplayName != "" {
		csvExtended.Spec.DisplayName = *specDisplayName
	}

	applyOverrides(&csvExtended)

	util.MarshallObject(csvExtended, os.Stdout)
}

func getHiddenCrds(csvExtended ClusterServiceVersionExtended) (string, error) {
	hiddenCrds := make([]string, 0)
	visibleCrds := strings.Split(*visibleCRDList, ",")
	for _, owned := range csvExtended.Spec.CustomResourceDefinitions.Owned {
		if !stringInSlice(owned.Name, visibleCrds) {
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

func processCsvs(componentsWithCsvs []util.CsvWithComponent, installStrategyBase *csvv1alpha1.StrategyDetailsDeployment, csvExtended *ClusterServiceVersionExtended, relatedImageSet RelatedImageSet) {
	for i, c := range componentsWithCsvs {
		processOneCsv(c, i, installStrategyBase, csvExtended, relatedImageSet)
	}
}

func processOneCsv(c util.CsvWithComponent, i int, installStrategyBase *csvv1alpha1.StrategyDetailsDeployment, csvExtended *ClusterServiceVersionExtended, relatedImageSet RelatedImageSet) {
	if c.Csv == "" {
		csvNames := []string{"CNA", "KubeVirt", "SSP", "CDI", "NMO", "HPP", "VM Import"}
		log.Panicf("ERROR: the %s CSV was empty", csvNames[i])
	}
	csvBytes := []byte(c.Csv)

	csvStruct := &ClusterServiceVersionExtended{}

	panicOnError(yaml.Unmarshal(csvBytes, csvStruct))

	strategySpec := csvStruct.Spec.InstallStrategy.StrategySpec

	// temporary workaround for https://bugzilla.redhat.com/1907381
	// a custom .spec.template.annotations["description"] it's causing a failure on OLM
	// TODO: remove this once fixed on OLM side
	for _, deployment := range strategySpec.DeploymentSpecs {
		delete(deployment.Spec.Template.Annotations, "description")
	}

	overwriteDeploymentSpecLabels(strategySpec.DeploymentSpecs, c.Component)
	installStrategyBase.DeploymentSpecs = append(installStrategyBase.DeploymentSpecs, strategySpec.DeploymentSpecs...)

	installStrategyBase.ClusterPermissions = append(installStrategyBase.ClusterPermissions, strategySpec.ClusterPermissions...)
	installStrategyBase.Permissions = append(installStrategyBase.Permissions, strategySpec.Permissions...)

	csvExtended.Spec.WebhookDefinitions = append(csvExtended.Spec.WebhookDefinitions, csvStruct.Spec.WebhookDefinitions...)

	for _, owned := range csvStruct.Spec.CustomResourceDefinitions.Owned {
		csvExtended.Spec.CustomResourceDefinitions.Owned = append(
			csvExtended.Spec.CustomResourceDefinitions.Owned,
			newCRDDescription(owned),
		)
	}
	csvBaseAlmString := csvExtended.Annotations[almExamplesAnnotation]
	csvStructAlmString := csvStruct.Annotations[almExamplesAnnotation]
	var baseAlmcrs []interface{}
	var structAlmcrs []interface{}

	if !strings.HasPrefix(csvBaseAlmString, "[") {
		csvBaseAlmString = "[" + csvBaseAlmString + "]"
	}

	panicOnError(json.Unmarshal([]byte(csvBaseAlmString), &baseAlmcrs))
	panicOnError(json.Unmarshal([]byte(csvStructAlmString), &structAlmcrs))

	baseAlmcrs = append(baseAlmcrs, structAlmcrs...)
	almB, err := json.Marshal(baseAlmcrs)
	panicOnError(err)
	csvExtended.Annotations[almExamplesAnnotation] = string(almB)

	if !*ignoreComponentsRelatedImages {
		for _, image := range csvStruct.Spec.RelatedImages {
			relatedImageSet.add(image.Ref)
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

func applyOverrides(csvExtended *ClusterServiceVersionExtended) {
	if *csvOverrides != "" {
		csvOBytes := []byte(*csvOverrides)

		csvO := &ClusterServiceVersionExtended{}

		panicOnError(yaml.Unmarshal(csvOBytes, csvO))

		panicOnError(mergo.Merge(csvExtended, csvO, mergo.WithOverride))
	}
}

func getInitialCsvList() []util.CsvWithComponent {
	return []util.CsvWithComponent{
		{
			Csv:       *cnaCsv,
			Component: hcoutil.AppComponentNetwork,
		},
		{
			Csv:       *virtCsv,
			Component: hcoutil.AppComponentCompute,
		},
		{
			Csv:       *sspCsv,
			Component: hcoutil.AppComponentSchedule,
		},
		{
			Csv:       *cdiCsv,
			Component: hcoutil.AppComponentStorage,
		},
		{
			Csv:       *nmoCsv,
			Component: hcoutil.AppComponentNetwork,
		},
		{
			Csv:       *hppCsv,
			Component: hcoutil.AppComponentStorage,
		},
		{
			Csv:       *vmImportCsv,
			Component: hcoutil.AppComponentImport,
		},
	}
}

func getReplacesVersion() string {
	if *replacesCsvVersion != "" {
		return fmt.Sprintf("%v.v%v", operatorName, semver.MustParse(*replacesCsvVersion).String())
	}
	return ""
}

func getRelatedImages() RelatedImageSet {
	relatedImageSet := newRelatedImageSet()

	for _, image := range strings.Split(*relatedImagesList, ",") {
		if image != "" {
			relatedImageSet.add(image)
		}
	}
	return relatedImageSet
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
		Namespace:           *namespace,
		Image:               *operatorImage,
		WebhookImage:        *webhookImage,
		ImagePullPolicy:     "IfNotPresent",
		ConversionContainer: *imsConversionImage,
		VmwareContainer:     *imsVMWareImage,
		Smbios:              *smbios,
		Machinetype:         *machinetype,
		HcoKvIoVersion:      *hcoKvIoVersion,
		KubevirtVersion:     *kubevirtVersion,
		CdiVersion:          *cdiVersion,
		CnaoVersion:         *cnaoVersion,
		SspVersion:          *sspVersion,
		NmoVersion:          *nmoVersion,
		HppoVersion:         *hppoVersion,
		VMImportVersion:     *vmImportVersion,
		Env:                 envVars,
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

// TODO: get rid of this once RelatedImageSet officially
// appears in github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators
type relatedImage struct {
	Name string `json:"name"`
	Ref  string `json:"image"`
}

// RelatedImageSet is a set that makes sure that each image appears only once.
type RelatedImageSet map[string]string

// constructor
func newRelatedImageSet() RelatedImageSet {
	return make(map[string]string)
}

// add image to the set. Ignore if the image already exists in the set
func (ri *RelatedImageSet) add(image string) {
	name := ""
	if strings.Contains(image, "|") {
		imageS := strings.Split(image, "|")
		image = imageS[0]
		name = imageS[1]
	} else {
		names := strings.Split(strings.Split(image, "@")[0], "/")
		name = names[len(names)-1]
	}

	(*ri)[name] = image
}

// return the related image set as a sorted slice
func (ri RelatedImageSet) dump() []relatedImage {
	images := make([]relatedImage, 0, len(ri))

	for name, image := range ri {
		images = append(images, relatedImage{Name: name, Ref: image})
	}

	sort.Sort(relatedImageSortable(images))
	return images
}

// implement sort.Interface for relatedImage slice. Sort by RelatedImage.Name
type relatedImageSortable []relatedImage

func (ris relatedImageSortable) Len() int {
	return len(ris)
}

func (ris relatedImageSortable) Less(i, j int) bool {
	return ris[i].Name < ris[j].Name
}

func (ris relatedImageSortable) Swap(i, j int) {
	ris[i], ris[j] = ris[j], ris[i]
}

func panicOnError(err error) {
	if err != nil {
		log.Println("Error!", err)
		panic(err)
	}
}

func appendOnce(slice []string, item string) []string {
	if stringInSlice(item, slice) {
		return slice
	}

	return append(slice, item)
}
