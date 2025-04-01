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
	"embed"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"
	"text/template"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

var (
	links = map[string]string{
		"metav1.ObjectMeta":        "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#objectmeta-v1-meta",
		"metav1.ListMeta":          "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#listmeta-v1-meta",
		"metav1.LabelSelector":     "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#labelselector-v1-meta",
		"v1.ResourceRequirements":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#resourcerequirements-v1-core",
		"v1.LocalObjectReference":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#localobjectreference-v1-core",
		"v1.SecretKeySelector":     "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#secretkeyselector-v1-core",
		"v1.PersistentVolumeClaim": "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#persistentvolumeclaim-v1-core",
		"v1.EmptyDirVolumeSource":  "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#emptydirvolumesource-v1-core",
		"apiextensionsv1.JSON":     "https://kubernetes.io/docs/reference/generated/kubernetes-api/%s/#json-v1-apiextensions-k8s-io",
	}

	selfLinks = map[string]string{
		"sdkapi.NodePlacement": "https://github.com/kubevirt/controller-lifecycle-operator-sdk/blob/bbf16167410b7a781c7b08a3f088fc39551c7a00/pkg/sdk/api/types.go#L49",
	}

	typeFields = map[string][]*ast.Field{}
)

const (
	kubebuilderDefaultPrefix = "// +kubebuilder:default="
)

func main() {
	err := setK8sLinks()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	types, err := parseFiles(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = printAPIDocs(os.Stdout, types)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseFiles(paths []string) ([]KubeTypes, error) {
	initial, err := getInitialInfo(paths)
	if err != nil {
		return nil, err
	}

	for _, strukt := range initial {
		selfLinks[strukt.name] = "#" + strings.ToLower(strukt.name)
		typeFields[toLink(strukt.name)] = strukt.fields
	}

	var types []KubeTypes
	for _, info := range initial {
		types = handleType(types, info.name, info.strct, info.doc)
	}

	return types, nil
}

type typeInfo struct {
	Name         string
	Doc          string
	PrintedType  string
	DefaultValue string
	Mandatory    bool
}

// KubeTypes is an array to represent all available types in a parsed file. [0] is for the type itself
type KubeTypes []typeInfo

type initialInfo struct {
	name   string
	strct  *ast.StructType
	doc    string
	fields []*ast.Field
}

func getInitialInfo(srcs []string) ([]initialInfo, error) {
	var initial []initialInfo
	for _, src := range srcs {
		fset := token.NewFileSet()

		file, err := parser.ParseFile(fset, src, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}

		initial = parseAST(initial, file.Decls)
	}

	slices.SortFunc(initial, func(a, b initialInfo) int {
		return strings.Compare(a.name, b.name)
	})

	return initial, nil
}

func parseAST(initial []initialInfo, decls []ast.Decl) []initialInfo {
	for _, decl := range decls {
		strct, name, doc := getStructFromDecl(decl)
		if strct != nil {
			initial = append(initial, initialInfo{
				name:   name,
				strct:  strct,
				doc:    doc,
				fields: strct.Fields.List,
			})
		}
	}

	return initial
}

func getStructFromDecl(decl ast.Decl) (*ast.StructType, string, string) {
	d, ok := decl.(*ast.GenDecl)
	if !ok || d.Tok != token.TYPE || len(d.Specs) != 1 {
		return nil, "", ""
	}

	s, ok := d.Specs[0].(*ast.TypeSpec)
	if !ok {
		return nil, "", ""
	}

	strct, ok := s.Type.(*ast.StructType)
	if !ok {
		return nil, "", ""
	}

	return strct, s.Name.Name, d.Doc.Text()
}

func handleType(docForTypes []KubeTypes, name string, st *ast.StructType, doc string) []KubeTypes {
	var ks KubeTypes
	ks = append(ks, typeInfo{Name: name, Doc: fmtRawDoc(doc), PrintedType: "", DefaultValue: "", Mandatory: false})

	for _, field := range st.Fields.List {
		ks = handleField(ks, field)
	}
	return append(docForTypes, ks)
}

func handleField(ks KubeTypes, field *ast.Field) KubeTypes {
	// Treat inlined fields separately as we don't want the original types to appear in the doc.
	if isInlined(field) {
		// Skip external types, as we don't want their content to be part of the API documentation.
		if isInternalType(field.Type) {
			var flds KubeTypes
			for _, fld := range typeFields[fieldType(field.Type)] {
				flds = handleField(flds, fld)
			}
			ks = append(ks, flds...)
		}
	} else if n := fieldName(field); n != "-" {
		fieldDoc := fmtRawDoc(field.Doc.Text())
		ks = append(ks, typeInfo{
			Name:         n,
			Doc:          fieldDoc,
			PrintedType:  fieldType(field.Type),
			DefaultValue: fieldDefault(field),
			Mandatory:    fieldRequired(field)})
	}
	return ks
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
	postDoc = strings.ReplaceAll(postDoc, "\\\"", "\"") // replace user's \" to "
	postDoc = strings.ReplaceAll(postDoc, "\"", "\\\"") // Escape "
	postDoc = strings.ReplaceAll(postDoc, "\n", "\\n")
	postDoc = strings.ReplaceAll(postDoc, "\t", "\\t")
	postDoc = strings.ReplaceAll(postDoc, "|", "\\|")

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
	switch v := typ.(type) {
	case *ast.SelectorExpr:
		pkg := v.X.(*ast.Ident)
		return strings.HasPrefix(pkg.Name, "hco.kubevirt.io")
	case *ast.StarExpr:
		return isInternalType(v.X)
	case *ast.ArrayType:
		return isInternalType(v.Elt)
	case *ast.MapType:
		return isInternalType(v.Key) && isInternalType(v.Value)
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
	switch v := typ.(type) {
	case *ast.Ident:
		return toLink(v.Name)
	case *ast.StarExpr:
		return "*" + toLink(fieldType(v.X))
	case *ast.SelectorExpr:
		pkg := v.X.(*ast.Ident)
		t := v.Sel
		return toLink(pkg.Name + "." + t.Name)
	case *ast.ArrayType:
		return "[]" + toLink(fieldType(v.Elt))
	case *ast.MapType:
		return "map[" + toLink(fieldType(v.Key)) + "]" + toLink(fieldType(v.Value))
	default:
		return ""
	}
}

func getK8sAPIVersion() (string, error) {
	data, err := os.ReadFile("./go.mod")
	if err != nil {
		return "", err
	}

	gomod, err := modfile.Parse("./go.mod", data, nil)
	if err != nil {
		return "", err
	}

	for _, req := range gomod.Require {
		if req.Mod.Path == "k8s.io/api" {
			v := strings.ReplaceAll(req.Mod.Version, "v0.", "v1.")
			return semver.MajorMinor(v), nil
		}
	}

	return "", errors.New("couldn't find the Kubernetes version in go.mod")
}

func setK8sLinks() error {
	k8sVer, err := getK8sAPIVersion()
	if err != nil {
		return err
	}

	for pkg, link := range links {
		links[pkg] = fmt.Sprintf(link, k8sVer)
	}

	return nil
}

//go:embed api.md.gotemplate
var templateFile embed.FS

func printAPIDocs(w io.Writer, types []KubeTypes) error {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"FirstItem": func(kubeTypes KubeTypes) typeInfo {
			return kubeTypes[0]
		},
		"ItemFields": func(kubeTypes KubeTypes) KubeTypes {
			return kubeTypes[1:]
		},
	}

	tmplt, err := template.New("api.md.gotemplate").Funcs(funcMap).ParseFS(templateFile, "api.md.gotemplate")
	if err != nil {
		return err
	}

	err = tmplt.Execute(w, types)
	if err != nil {
		return err
	}

	return nil
}
