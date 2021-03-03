// Copyright 2016 The prometheus-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// The changes can be made visible by diffing this file with the
// https://github.com/prometheus-operator/prometheus-operator/blob/8497a67b735e65ad779ed19c95a17ffd1c8fbb64/cmd/po-docgen/api.go version
//

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
)

const (
	firstParagraph = `
# API Docs

This Document documents the types introduced by the hyperconverged-cluster-operator to be consumed by users.

> Note this document is generated from code comments. When contributing a change to this document please do so by changing the code comments.`
)

var (
	links = map[string]string{
		"metav1.ObjectMeta":        "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#objectmeta-v1-meta",
		"metav1.ListMeta":          "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#listmeta-v1-meta",
		"metav1.LabelSelector":     "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#labelselector-v1-meta",
		"v1.ResourceRequirements":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#resourcerequirements-v1-core",
		"v1.LocalObjectReference":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#localobjectreference-v1-core",
		"v1.SecretKeySelector":     "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#secretkeyselector-v1-core",
		"v1.PersistentVolumeClaim": "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#persistentvolumeclaim-v1-core",
		"v1.EmptyDirVolumeSource":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#emptydirvolumesource-v1-core",
		"apiextensionsv1.JSON":     "https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.17/#json-v1-apiextensions-k8s-io",
	}

	selfLinks = map[string]string{
		"sdkapi.NodePlacement": "https://github.com/kubevirt/controller-lifecycle-operator-sdk/blob/bbf16167410b7a781c7b08a3f088fc39551c7a00/pkg/sdk/api/types.go#L49",
	}
	typesDoc = map[string]KubeTypes{}
)

const (
	kubebuilderDefaultPrefix = "// +kubebuilder:default="
)

func toSectionLink(name string) string {
	name = strings.ToLower(name)
	name = strings.Replace(name, " ", "-", -1)
	return name
}

func printTOC(types []KubeTypes) {
	fmt.Printf("\n## Table of Contents\n")
	for _, t := range types {
		strukt := t[0]
		if len(t) > 1 {
			fmt.Printf("* [%s](#%s)\n", strukt.Name, toSectionLink(strukt.Name))
		}
	}
}

func printAPIDocs(paths []string) {
	fmt.Println(firstParagraph)

	types := ParseDocumentationFrom(paths)
	for _, t := range types {
		strukt := t[0]
		selfLinks[strukt.Name] = "#" + strings.ToLower(strukt.Name)
		typesDoc[toLink(strukt.Name)] = t[1:]
	}

	// we need to parse once more to now add the self links and the inlined fields
	types = ParseDocumentationFrom(paths)

	printTOC(types)

	for _, t := range types {
		strukt := t[0]
		if len(t) > 1 {
			fmt.Printf("\n## %s\n\n%s\n\n", strukt.Name, strukt.Doc)

			fmt.Println("| Field | Description | Scheme | Default | Required |")
			fmt.Println("| ----- | ----------- | ------ | -------- |-------- |")
			fields := t[1:(len(t))]
			for _, f := range fields {
				fmt.Println("|", f.Name, "|", f.Doc, "|", f.Type, "|", f.Default, "|", f.Mandatory, "|")
			}
			fmt.Println("")
			fmt.Println("[Back to TOC](#table-of-contents)")
		}
	}
}

// Pair of strings. We keed the name of fields and the doc
type Pair struct {
	Name, Doc, Type, Default string
	Mandatory                bool
}

// KubeTypes is an array to represent all available types in a parsed file. [0] is for the type itself
type KubeTypes []Pair

// ParseDocumentationFrom gets all types' documentation and returns them as an
// array. Each type is again represented as an array (we have to use arrays as we
// need to be sure for the order of the fields). This function returns fields and
// struct definitions that have no documentation as {name, ""}.
func ParseDocumentationFrom(srcs []string) []KubeTypes {
	var docForTypes []KubeTypes

	for _, src := range srcs {
		pkg := astFrom(src)

		for _, kubType := range pkg.Types {
			if structType, ok := kubType.Decl.Specs[0].(*ast.TypeSpec).Type.(*ast.StructType); ok {
				var ks KubeTypes
				ks = append(ks, Pair{Name: kubType.Name, Doc: fmtRawDoc(kubType.Doc), Type: "", Default: "", Mandatory: false})

				for _, field := range structType.Fields.List {
					// Treat inlined fields separately as we don't want the original types to appear in the doc.
					if isInlined(field) {
						// Skip external types, as we don't want their content to be part of the API documentation.
						if isInternalType(field.Type) {
							ks = append(ks, typesDoc[fieldType(field.Type)]...)
						}
						continue
					}

					if n := fieldName(field); n != "-" {
						fieldDoc := fmtRawDoc(field.Doc.Text())
						ks = append(ks, Pair{
							Name:      n,
							Doc:       fieldDoc,
							Type:      fieldType(field.Type),
							Default:   fieldDefault(field),
							Mandatory: fieldRequired(field)})
					}
				}
				docForTypes = append(docForTypes, ks)
			}
		}
	}

	return docForTypes
}

func astFrom(filePath string) *doc.Package {
	fset := token.NewFileSet()
	m := make(map[string]*ast.File)

	f, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	m[filePath] = f
	apkg, _ := ast.NewPackage(fset, m, nil, nil)

	return doc.New(apkg, "", 0)
}

func fmtRawDoc(rawDoc string) string {
	var buffer bytes.Buffer
	delPrevChar := func() {
		if buffer.Len() > 0 {
			buffer.Truncate(buffer.Len() - 1) // Delete the last " " or "\n"
		}
	}

	// Ignore all lines after ---
	rawDoc = strings.Split(rawDoc, "---")[0]

	for _, line := range strings.Split(rawDoc, "\n") {
		line = strings.TrimRight(line, " ")
		leading := strings.TrimLeft(line, " ")
		switch {
		case len(line) == 0: // Keep paragraphs
			delPrevChar()
			buffer.WriteString("\n\n")
		case strings.HasPrefix(leading, "TODO"): // Ignore one line TODOs
		case strings.HasPrefix(leading, "+"): // Ignore instructions to go2idl
		default:
			if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				delPrevChar()
				line = "\n" + line + "\n" // Replace it with newline. This is useful when we have a line with: "Example:\n\tJSON-someting..."
			} else {
				line += " "
			}
			buffer.WriteString(line)
		}
	}

	postDoc := strings.TrimRight(buffer.String(), "\n")
	postDoc = strings.Replace(postDoc, "\\\"", "\"", -1) // replace user's \" to "
	postDoc = strings.Replace(postDoc, "\"", "\\\"", -1) // Escape "
	postDoc = strings.Replace(postDoc, "\n", "\\n", -1)
	postDoc = strings.Replace(postDoc, "\t", "\\t", -1)
	postDoc = strings.Replace(postDoc, "|", "\\|", -1)

	return postDoc
}

func toLink(typeName string) string {
	selfLink, hasSelfLink := selfLinks[typeName]
	if hasSelfLink {
		return wrapInLink(typeName, selfLink)
	}

	link, hasLink := links[typeName]
	if hasLink {
		return wrapInLink(typeName, link)
	}

	return typeName
}

func wrapInLink(text, link string) string {
	return fmt.Sprintf("[%s](%s)", text, link)
}

func isInlined(field *ast.Field) bool {
	jsonTag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Get("json") // Delete first and last quotation
	return strings.Contains(jsonTag, "inline")
}

func isInternalType(typ ast.Expr) bool {
	switch typ.(type) {
	case *ast.SelectorExpr:
		e := typ.(*ast.SelectorExpr)
		pkg := e.X.(*ast.Ident)
		return strings.HasPrefix(pkg.Name, "monitoring")
	case *ast.StarExpr:
		return isInternalType(typ.(*ast.StarExpr).X)
	case *ast.ArrayType:
		return isInternalType(typ.(*ast.ArrayType).Elt)
	case *ast.MapType:
		mapType := typ.(*ast.MapType)
		return isInternalType(mapType.Key) && isInternalType(mapType.Value)
	default:
		return true
	}
}

// fieldName returns the name of the field as it should appear in JSON format
// "-" indicates that this field is not part of the JSON representation
func fieldName(field *ast.Field) string {
	jsonTag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Get("json") // Delete first and last quotation
	jsonTag = strings.Split(jsonTag, ",")[0]                                              // This can return "-"
	if jsonTag == "" {
		if field.Names != nil {
			return field.Names[0].Name
		}
		return field.Type.(*ast.Ident).Name
	}
	return jsonTag
}

// fieldRequired returns whether a field is a required field.
func fieldRequired(field *ast.Field) bool {
	jsonTag := ""
	if field.Tag != nil {
		jsonTag = reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1]).Get("json") // Delete first and last quotation
		return !strings.Contains(jsonTag, "omitempty")
	}

	return false
}

// fieldDefault returns the default value of the field set by kubebuilder:default
func fieldDefault(field *ast.Field) string {
	if field.Doc != nil {
		for _, doc := range field.Doc.List {
			if strings.HasPrefix(doc.Text, kubebuilderDefaultPrefix) {
				def := doc.Text[len(kubebuilderDefaultPrefix):]
				return def
			}
		}
	}
	return ""
}

func fieldType(typ ast.Expr) string {
	switch typ.(type) {
	case *ast.Ident:
		return toLink(typ.(*ast.Ident).Name)
	case *ast.StarExpr:
		return "*" + toLink(fieldType(typ.(*ast.StarExpr).X))
	case *ast.SelectorExpr:
		e := typ.(*ast.SelectorExpr)
		pkg := e.X.(*ast.Ident)
		t := e.Sel
		return toLink(pkg.Name + "." + t.Name)
	case *ast.ArrayType:
		return "[]" + toLink(fieldType(typ.(*ast.ArrayType).Elt))
	case *ast.MapType:
		mapType := typ.(*ast.MapType)
		return "map[" + toLink(fieldType(mapType.Key)) + "]" + toLink(fieldType(mapType.Value))
	default:
		return ""
	}
}

func main() {
	printAPIDocs(os.Args[1:])
}
