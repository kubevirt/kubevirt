package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

func main() {

	dirname := flag.String("crdDir", "./manifests/generated/", "path to directory with crds from where validation field will be parsed")
	outputdir := flag.String("outputDir", "./pkg/virt-operator/install-strategy/", "path to dir where go file will be generated")

	flag.Parse()

	files, err := ioutil.ReadDir(*dirname)
	if err != nil {
		panic(fmt.Errorf("Error occured reading directory, %v", err))
	}
	validations := make(map[string]*extv1beta1.CustomResourceValidation)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		filename := file.Name()
		if strings.HasSuffix(filename, "-resource.yaml") {
			crdname, validation := getValidation(*dirname + filename)
			if validation != nil {
				validations[crdname] = validation
			}
		}

	}
	generateGoFile(*outputdir, &validations)
}

var variable = " \"%s\" : `%s`,\n"

func generateGoFile(outputDir string, validations *map[string]*extv1beta1.CustomResourceValidation) {
	filepath := fmt.Sprintf("%svalidations_generated.go", outputDir)
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	if err != nil {
		panic(fmt.Errorf("Failed to create go file %v, %v", filepath, err))
	}
	// w := bufio.NewWriter(file)
	file.WriteString("package installstrategy\n\n")
	file.WriteString("var resources map[string]string = map[string]string{\n")
	for k, v := range *validations {
		b, _ := yaml.Marshal(v)
		file.WriteString(fmt.Sprintf(variable, k, string(b)))
	}
	file.WriteString("}\n")

}

func getValidation(filename string) (string, *extv1beta1.CustomResourceValidation) {
	file, err := os.Open(filename)
	if err != nil {
		panic(fmt.Errorf("Failed to read file %v, %v", filename, err))
	}
	defer file.Close()

	crd := extv1beta1.CustomResourceDefinition{}
	err = k8syaml.NewYAMLToJSONDecoder(file).Decode(&crd)
	if err != nil {
		panic(fmt.Errorf("Failed to parse crd from file %v, %v", filename, err))
	}
	return crd.Spec.Names.ShortNames[0], crd.Spec.Validation
}
