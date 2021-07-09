package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/ghodss/yaml"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func main() {

	dirname := flag.String("crdDir", "staging/src/kubevirt.io/client-go/config/crd/", "path to directory with crds from where validation field will be parsed")
	outputdir := flag.String("outputDir", "pkg/virt-operator/resource/generate/components/", "path to dir where go file will be generated")

	flag.Parse()

	files, err := os.ReadDir(*dirname)
	if err != nil {
		panic(fmt.Errorf("Error occurred reading directory, %v", err))
	}

	if len(files) == 0 {
		panic("Povided crdDir is empty")
	}

	validations := make(map[string]*extv1.CustomResourceValidation)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		if strings.HasSuffix(filename, ".yaml") {
			crdname, validation := getValidation(*dirname + filename)
			if validation != nil {
				validations[crdname] = validation
			}
		}

	}
	generateGoFile(*outputdir, validations)
}

var variable = " \"%s\" : `%s`,\n"

func generateGoFile(outputDir string, validations map[string]*extv1.CustomResourceValidation) {
	filepath := fmt.Sprintf("%svalidations_generated.go", outputDir)
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		panic(fmt.Errorf("Failed to create go file %v, %v", filepath, err))
	}
	// w := bufio.NewWriter(file)
	file.WriteString("package components\n\n")
	file.WriteString("var CRDsValidation map[string]string = map[string]string{\n")

	crds := make([]string, 0, 0)
	for k := range validations {
		crds = append(crds, k)
	}

	sort.Strings(crds)

	for _, crdname := range crds {
		crd := validations[crdname]
		b, _ := yaml.Marshal(crd)
		file.WriteString(fmt.Sprintf(variable, crdname, string(b)))
	}
	file.WriteString("}\n")

}

func getValidation(filename string) (string, *extv1.CustomResourceValidation) {
	file, err := os.Open(filename)
	if err != nil {
		panic(fmt.Errorf("Failed to read file %v, %v", filename, err))
	}
	defer file.Close()

	crd := extv1.CustomResourceDefinition{}
	err = k8syaml.NewYAMLToJSONDecoder(file).Decode(&crd)
	if err != nil {
		panic(fmt.Errorf("Failed to parse crd from file %v, %v", filename, err))
	}
	return crd.Spec.Names.Singular, crd.Spec.Versions[0].Schema
}
