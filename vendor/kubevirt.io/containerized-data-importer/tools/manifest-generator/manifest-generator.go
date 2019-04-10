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
	"flag"
	"os"
	"text/template"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	cdicluster "kubevirt.io/containerized-data-importer/pkg/operator/resources/cluster"
	cdinamespaced "kubevirt.io/containerized-data-importer/pkg/operator/resources/namespaced"
)

type templateData struct {
	DockerRepo        string
	DockerTag         string
	ControllerImage   string
	ImporterImage     string
	ClonerImage       string
	APIServerImage    string
	UploadProxyImage  string
	UploadServerImage string
	OperatorImage     string
	Verbosity         string
	PullPolicy        string
	Namespace         string
}

var (
	dockerRepo        = flag.String("docker-repo", "", "")
	dockertag         = flag.String("docker-tag", "", "")
	controllerImage   = flag.String("controller-image", "", "")
	importerImage     = flag.String("importer-image", "", "")
	clonerImage       = flag.String("cloner-image", "", "")
	apiServerImage    = flag.String("apiserver-image", "", "")
	uploadProxyImage  = flag.String("uploadproxy-image", "", "")
	uploadServerImage = flag.String("uploadserver-image", "", "")
	operatorImage     = flag.String("operator-image", "", "")
	verbosity         = flag.String("verbosity", "1", "")
	pullPolicy        = flag.String("pull-policy", "", "")
	namespace         = flag.String("namespace", "", "")
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

func generateFromFile(templFile string) {
	data := &templateData{
		Verbosity:         *verbosity,
		DockerRepo:        *dockerRepo,
		DockerTag:         *dockertag,
		ControllerImage:   *controllerImage,
		ImporterImage:     *importerImage,
		ClonerImage:       *clonerImage,
		APIServerImage:    *apiServerImage,
		UploadProxyImage:  *uploadProxyImage,
		UploadServerImage: *uploadServerImage,
		OperatorImage:     *operatorImage,
		PullPolicy:        *pullPolicy,
		Namespace:         *namespace,
	}

	file, err := os.OpenFile(templFile, os.O_RDONLY, 0)
	if err != nil {
		klog.Fatalf("Failed to open file %s: %v\n", templFile, err)
	}
	defer file.Close()

	tmpl := template.Must(template.ParseFiles(templFile))
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		klog.Fatalf("Error executing template: %v\n", err)
	}
}

func generateFromCode(codeGroup string) {
	var resources []runtime.Object

	crs, err := getClusterResources(codeGroup)
	if err != nil {
		klog.Fatalf("Error getting cluster resources: %v\n", err)
	}

	resources = append(resources, crs...)

	nsrs, err := getNamespacedResources(codeGroup)
	if err != nil {
		klog.Fatalf("Error getting namespaced resources: %v\n", err)
	}

	resources = append(resources, nsrs...)

	for _, resource := range resources {
		err = MarshallObject(resource, os.Stdout)
		if err != nil {
			klog.Fatalf("Error marshalling resource: %v\n", err)
		}
	}
}

func getClusterResources(codeGroup string) ([]runtime.Object, error) {
	args := &cdicluster.FactoryArgs{
		Namespace: *namespace,
	}

	if codeGroup == "everything" {
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

	if codeGroup == "everything" {
		return cdinamespaced.CreateAllResources(args)
	}

	return cdinamespaced.CreateResourceGroup(codeGroup, args)
}
