//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"bufio"
	"encoding/base64"
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	cdicluster "kubevirt.io/containerized-data-importer/pkg/operator/resources/cluster"
	"kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
	cdinamespaced "kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
	cdioperator "kubevirt.io/containerized-data-importer/pkg/operator/resources/operator"
	"kubevirt.io/containerized-data-importer/tools/marketplace/helper"
	"kubevirt.io/containerized-data-importer/tools/util"
)

type templateData struct {
	DockerRepo             string
	DockerTag              string
	CsvVersion             string
	ReplacesCsvVersion     string
	QuayNamespace          string
	QuayRepository         string
	OperatorRules          string
	OperatorDeploymentSpec string
	CDILogo                string
	DeployClusterResources string
	OperatorImage          string
	ControllerImage        string
	ImporterImage          string
	ClonerImage            string
	APIServerImage         string
	UploadProxyImage       string
	UploadServerImage      string
	Verbosity              string
	PullPolicy             string
	Namespace              string
	GeneratedManifests     map[string]string
}

var (
	dockerRepo             = flag.String("docker-repo", "", "")
	dockertag              = flag.String("docker-tag", "", "")
	csvVersion             = flag.String("csv-version", "", "")
	cdiLogoPath            = flag.String("cdi-logo-path", "", "")
	genManifestsPath       = flag.String("generated-manifests-path", "", "")
	bundleOut              = flag.String("olm-bundle-dir", "", "")
	quayNamespace          = flag.String("quay-namespace", "", "")
	quayRepository         = flag.String("quay-repository", "", "")
	deployClusterResources = flag.String("deploy-cluster-resources", "", "")
	operatorImage          = flag.String("operator-image", "", "")
	controllerImage        = flag.String("controller-image", "", "")
	importerImage          = flag.String("importer-image", "", "")
	clonerImage            = flag.String("cloner-image", "", "")
	apiServerImage         = flag.String("apiserver-image", "", "")
	uploadProxyImage       = flag.String("uploadproxy-image", "", "")
	uploadServerImage      = flag.String("uploadserver-image", "", "")
	verbosity              = flag.String("verbosity", "1", "")
	pullPolicy             = flag.String("pull-policy", "", "")
	namespace              = flag.String("namespace", "", "")
)

func main() {
	templFile := flag.String("template", "", "")
	codeGroup := flag.String("code-group", "everything", "")
	flag.Parse()

	klogFlags := flag.NewFlagSet("klog", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	flag.CommandLine.VisitAll(func(f1 *flag.Flag) {
		f2 := klogFlags.Lookup(f1.Name)
		if f2 != nil {
			value := f1.Value.String()
			f2.Value.Set(value)
		}
	})

	if *templFile != "" {
		generateFromFile(*templFile)
		return
	}

	generateFromCode(*codeGroup)
}

func getOperatorRules() string {
	rules := *cdioperator.GetOperatorClusterRules()

	writer := strings.Builder{}
	for _, rule := range rules {
		err := util.MarshallObject(rule, &writer)
		if err != nil {
			panic(err)
		}
	}
	return fixResourceString(writer.String(), 14)
}

func getOperatorDeploymentSpec() string {
	args := &cdioperator.FactoryArgs{
		Verbosity:              *verbosity,
		DockerRepo:             *dockerRepo,
		DockerTag:              *dockertag,
		DeployClusterResources: *deployClusterResources,
		OperatorImage:          *operatorImage,
		ControllerImage:        *controllerImage,
		ImporterImage:          *importerImage,
		ClonerImage:            *clonerImage,
		APIServerImage:         *apiServerImage,
		UploadProxyImage:       *uploadProxyImage,
		UploadServerImage:      *uploadServerImage,
		PullPolicy:             *pullPolicy,
		Namespace:              *namespace,

		CsvVersion: *csvVersion,
		CDILogo:    getCdiLogo(*cdiLogoPath),
	}

	spec := cdioperator.GetOperatorDeploymentSpec(args)

	writer := strings.Builder{}

	err := util.MarshallObject(spec, &writer)
	if err != nil {
		panic(err)
	}

	return fixResourceString(writer.String(), 14)
}

func fixResourceString(in string, indention int) string {
	out := strings.Builder{}
	scanner := bufio.NewScanner(strings.NewReader(in))
	for scanner.Scan() {
		line := scanner.Text()
		// remove separator lines
		if !strings.HasPrefix(line, "---") {
			// indent so that it fits into the manifest
			// spaces is is indention - 2, because we want to have 2 spaces less for being able to start an array
			spaces := strings.Repeat(" ", indention-2)
			if strings.HasPrefix(line, "apiGroups") {
				// spaces + array start
				out.WriteString(spaces + "- " + line + "\n")
			} else {
				// 2 more spaces
				out.WriteString(spaces + "  " + line + "\n")
			}
		}
	}
	return out.String()
}

func getReplacesVersion(csvVersion, quayNamespace, quayRepository string) string {
	bundleHelper, err := helper.NewBundleHelper(quayRepository, quayNamespace)
	if err != nil {
		klog.Fatalf("Failed to access quay namespace %s, repo %s, %v\n", quayNamespace, quayRepository, err)
	}
	if !bundleHelper.VerifyNotPublishedCSVVersion(csvVersion) {
		klog.Fatalf("CSV version %s is already published!", csvVersion)
	}
	return bundleHelper.GetLatestPublishedCSVVersion()
}

func evalOlmCsvUpdateVersion(inFile, csvVersion, bundleOutDir, quayNamespace, quayRepository string) string {
	latestVersion := ""
	if strings.Contains(inFile, ".csv.yaml") && bundleOutDir != "" {
		bundleHelper, err := helper.NewBundleHelper(quayRepository, quayNamespace)
		if err != nil {
			klog.Fatalf("Failed to access quay namespace %s, repo %s, %v\n", quayNamespace, quayRepository, err)
		}
		if !bundleHelper.VerifyNotPublishedCSVVersion(csvVersion) {
			klog.Fatalf("CSV version %s is already published!", csvVersion)
		}
		latestVersion := bundleHelper.GetLatestPublishedCSVVersion()
		if latestVersion != "" {
			// prevent generating the same version again
			if strings.HasSuffix(latestVersion, csvVersion) {
				klog.Fatalf("CSV version %s is already published!", csvVersion)
			}
			// also copy old manifests to out dir
			if *bundleOut != "" {
				bundleHelper.AddOldManifests(bundleOutDir, csvVersion)
			}
		}
	}
	return latestVersion
}

func generateFromFile(templFile string) {
	data := &templateData{
		Verbosity:              *verbosity,
		DockerRepo:             *dockerRepo,
		DockerTag:              *dockertag,
		CsvVersion:             *csvVersion,
		DeployClusterResources: *deployClusterResources,
		OperatorImage:          *operatorImage,
		ControllerImage:        *controllerImage,
		ImporterImage:          *importerImage,
		ClonerImage:            *clonerImage,
		APIServerImage:         *apiServerImage,
		UploadProxyImage:       *uploadProxyImage,
		UploadServerImage:      *uploadServerImage,
		PullPolicy:             *pullPolicy,
		Namespace:              *namespace,
	}

	file, err := os.OpenFile(templFile, os.O_RDONLY, 0)
	if err != nil {
		klog.Fatalf("Failed to open file %s: %v\n", templFile, err)
	}
	defer file.Close()

	data.ReplacesCsvVersion = evalOlmCsvUpdateVersion(templFile, *csvVersion, *bundleOut, *quayNamespace, *quayRepository)
	data.QuayRepository = *quayRepository
	data.QuayNamespace = *quayNamespace
	data.OperatorRules = getOperatorRules()
	data.OperatorDeploymentSpec = getOperatorDeploymentSpec()
	data.CDILogo = getCdiLogo(*cdiLogoPath)

	// Read generated manifests and populate templated manifest
	genDir := *genManifestsPath
	data.GeneratedManifests = make(map[string]string)
	manifests, err := ioutil.ReadDir(genDir)
	if err != nil {
		klog.Fatalf("Failed to read directory %s: %v\n", genDir, err)
	}

	for _, manifest := range manifests {
		if manifest.IsDir() {
			continue
		}
		b, err := ioutil.ReadFile(filepath.Join(genDir, manifest.Name()))
		if err != nil {
			klog.Fatalf("Failed to read file %s: %v\n", templFile, err)
		}

		data.GeneratedManifests[manifest.Name()] = string(b)
	}

	tmpl := template.Must(template.ParseFiles(templFile))
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		klog.Fatalf("Error executing template: %v\n", err)
	}
}

func getCdiLogo(path string) string {
	file, err := os.Open(path)
	if err != nil {
		klog.Fatalf("Error retrieving cdi logo file: %s, %v\n", path, err)
	}

	// Read entire file into byte slice.
	reader := bufio.NewReader(file)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		klog.Fatalf("Error reading cdi logo file: %v\n", err)
	}

	// Encode as base64.
	encoded := base64.StdEncoding.EncodeToString(content)
	return encoded
}

const (
	//ClusterResource - cluster resources
	ClusterResource string = "cluster"
	//OperatorResource - operator resources
	OperatorResource string = "operator"
	//NamespaceResource - namespace resources
	NamespaceResource string = "namespaces"
)

type resourceGet func(string) ([]runtime.Object, error)
type resourcetype func(string) bool
type resourceTuple struct {
	resourcetype resourcetype
	resourceGet  resourceGet
}

var resourcesTable = map[string]resourceTuple{
	ClusterResource:   {cdicluster.IsFactoryResource, getClusterResources},
	NamespaceResource: {namespaced.IsFactoryResource, getNamespacedResources},
	OperatorResource:  {cdioperator.IsFactoryResource, getOperatorClusterResources},
}

func generateFromCode(codeGroup string) {
	var resources []runtime.Object

	for r, dispatch := range resourcesTable {
		if dispatch.resourcetype(codeGroup) {
			crs, err := dispatch.resourceGet(codeGroup)
			if err != nil {
				klog.Fatalf("Error getting %s resources: %v\n", r, err)
			}
			resources = append(resources, crs...)
		} //of codeGroup matches resource then get it
	} //iterate through all resources

	for _, resource := range resources {
		err := util.MarshallObject(resource, os.Stdout)
		if err != nil {
			klog.Fatalf("Error marshalling resource: %v\n", err)
		}
	}
}

const (
	//ClusterResourcesCodeGroupEverything - generate all cluster resources
	ClusterResourcesCodeGroupEverything string = "cluster-everything"
	//NamespaceResourcesCodeGroupEverything - generate all namespace resources
	NamespaceResourcesCodeGroupEverything string = "namespace-everything"
	//ClusterResourcesCodeOperatorGroupEverything - generate all operator resources
	ClusterResourcesCodeOperatorGroupEverything string = "operator-everything"
)

func getOperatorClusterResources(codeGroup string) ([]runtime.Object, error) {
	replacesCsvVersion := ""
	if codeGroup == cdioperator.OperatorCSV || codeGroup == ClusterResourcesCodeOperatorGroupEverything {
		replacesCsvVersion = getReplacesVersion(*csvVersion, *quayNamespace, *quayRepository)
	}

	args := &cdioperator.FactoryArgs{
		Verbosity:              *verbosity,
		DockerRepo:             *dockerRepo,
		DockerTag:              *dockertag,
		DeployClusterResources: *deployClusterResources,
		OperatorImage:          *operatorImage,
		ControllerImage:        *controllerImage,
		ImporterImage:          *importerImage,
		ClonerImage:            *clonerImage,
		APIServerImage:         *apiServerImage,
		UploadProxyImage:       *uploadProxyImage,
		UploadServerImage:      *uploadServerImage,
		PullPolicy:             *pullPolicy,
		Namespace:              *namespace,

		CsvVersion:         *csvVersion,
		ReplacesCsvVersion: replacesCsvVersion,
		CDILogo:            getCdiLogo(*cdiLogoPath),
	}

	if codeGroup == ClusterResourcesCodeOperatorGroupEverything {
		return cdioperator.CreateAllOperatorResources(args)
	}

	return cdioperator.CreateOperatorResourceGroup(codeGroup, args)
}

func getClusterResources(codeGroup string) ([]runtime.Object, error) {
	args := &cdicluster.FactoryArgs{
		Verbosity:              *verbosity,
		DockerRepo:             *dockerRepo,
		DockerTag:              *dockertag,
		DeployClusterResources: *deployClusterResources,
		ControllerImage:        *controllerImage,
		ImporterImage:          *importerImage,
		ClonerImage:            *clonerImage,
		APIServerImage:         *apiServerImage,
		UploadProxyImage:       *uploadProxyImage,
		UploadServerImage:      *uploadServerImage,
		PullPolicy:             *pullPolicy,
		Namespace:              *namespace,
	}

	if codeGroup == ClusterResourcesCodeGroupEverything {
		return cdicluster.CreateAllResources(args)
	}

	return cdicluster.CreateResourceGroup(codeGroup, args)
}

func getNamespacedResources(codeGroup string) ([]runtime.Object, error) {
	args := &cdinamespaced.FactoryArgs{
		Verbosity:         *verbosity,
		DockerRepo:        *dockerRepo,
		DockerTag:         *dockertag,
		ControllerImage:   *controllerImage,
		ImporterImage:     *importerImage,
		ClonerImage:       *clonerImage,
		APIServerImage:    *apiServerImage,
		UploadProxyImage:  *uploadProxyImage,
		UploadServerImage: *uploadServerImage,
		PullPolicy:        *pullPolicy,
		Namespace:         *namespace,
	}

	if codeGroup == NamespaceResourcesCodeGroupEverything {
		return cdinamespaced.CreateAllResources(args)
	}

	return cdinamespaced.CreateResourceGroup(codeGroup, args)
}
