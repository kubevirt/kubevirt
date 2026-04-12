package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
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
		crd.OpenAPIV3Schema = sanitizeSchema(crd.OpenAPIV3Schema)
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

// sanitizeSchema traverses the given JSON-Schema object and:
// - replaces backticks (`) in Description fields with single quotes
// - normalizes checksum field schemas to the expected uint32 range represented as int64
func sanitizeSchema(inSchema *extv1.JSONSchemaProps) *extv1.JSONSchemaProps {
	return sanitizeSchemaWithPath(inSchema, nil)
}

func sanitizeSchemaWithPath(inSchema *extv1.JSONSchemaProps, path []string) *extv1.JSONSchemaProps {
	schema := inSchema.DeepCopy()
	if schema.Description != "" {
		schema.Description = strings.ReplaceAll(schema.Description, "`", "'")
	}
	normalizeChecksumFieldSchema(schema, path)

	// Traverse Items
	if schema.Items != nil {
		if schema.Items.Schema != nil {
			schema.Items.Schema = sanitizeSchemaWithPath(schema.Items.Schema, append(path, "[]"))
		}
		if len(schema.Items.JSONSchemas) > 0 {
			sanitizedProps := make([]extv1.JSONSchemaProps, 0, len(schema.Items.JSONSchemas))
			for _, schema := range schema.Items.JSONSchemas {
				sanitizedProps = append(sanitizedProps, *sanitizeSchemaWithPath(&schema, append(path, "[]")))
			}
			schema.Items.JSONSchemas = sanitizedProps
		}
	}

	// Traverse Properties
	for name, prop := range schema.Properties {
		nextPath := append(append([]string{}, path...), name)
		schema.Properties[name] = *sanitizeSchemaWithPath(&prop, nextPath)
	}

	return schema
}

func normalizeChecksumFieldSchema(schema *extv1.JSONSchemaProps, path []string) {
	if !shouldNormalizeChecksum(path) {
		return
	}

	min := float64(0)
	max := float64(^uint32(0))
	schema.Format = "int64"
	schema.Minimum = &min
	schema.Maximum = &max
}

func shouldNormalizeChecksum(path []string) bool {
	if len(path) < 2 {
		return false
	}
	if path[len(path)-1] != "checksum" {
		return false
	}

	switch path[len(path)-2] {
	case "containerDiskVolume", "kernelInfo", "initrdInfo":
		return true
	default:
		return false
	}
}
