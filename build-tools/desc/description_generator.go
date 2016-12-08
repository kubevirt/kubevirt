package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func main() {
	sourceFile := flag.String("in", "", "golang file containing strucs for swagger")
	targetFile := flag.String("out", "", "target file where description should be written")
	flag.Parse()

	if *sourceFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *targetFile == "" {
		if strings.HasSuffix(*sourceFile, "_test.go") {
			*targetFile = strings.TrimSuffix(*sourceFile, "_test.go") + "_swagger_generated_test.go"
		} else {
			*targetFile = strings.TrimSuffix(*sourceFile, ".go") + "_swagger_generated.go"
		}
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, *sourceFile, nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	c := Parse(f)

	var file *os.File
	if *targetFile == "-" {
		file = os.Stdout
	} else {
		file, err = os.Create(*targetFile)
		defer file.Close()
		if err != nil {
			panic(err)
		}
	}
	for x := range c {
		fmt.Fprintln(file, x)
	}
}

func Parse(f *ast.File) chan string {
	c := make(chan string)
	go func() {
		defer close(c)
		re := regexp.MustCompile(`^([^,]+).*`)
		firstStruct := true
		for _, decl := range f.Decls {

			gendecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			if len(gendecl.Specs) == 0 {
				continue
			}
			typeSpec, ok := gendecl.Specs[0].(*ast.TypeSpec)
			if !ok {
				continue
			}
			structDecl, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			if firstStruct {
				firstStruct = false
				c <- "package " + f.Name.Name
			}
			c <- ""
			c <- fmt.Sprintf("func (%s) SwaggerDoc() map[string]string {", typeSpec.Name)
			c <- "\treturn map[string]string{"

			structDoc := filterDoc(gendecl.Doc.Text())
			if len(structDoc) > 0 {
				c <- "\t\t" + "\"\": " + strconv.Quote(structDoc) + ","
			}

			for _, field := range structDecl.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				fieldName := field.Names[0].Name
				if field.Tag != nil {
					tag := reflect.StructTag(field.Tag.Value)
					jsonTag := tag.Get("`json")
					if len(jsonTag) > 0 {
						matches := re.FindStringSubmatch(jsonTag)
						if len(matches) == 2 && len(matches[1]) > 0 {
							fieldName = matches[1]
						}
					}
				}

				docText := filterDoc(field.Doc.Text())
				if len(docText) > 0 {
					c <- "\t\t" + "\"" + fieldName + "\": " + strconv.Quote(docText) + ","
				}
			}
			c <- "\t}"
			c <- "}"
		}
	}()
	return c
}

func filterDoc(doc string) string {
	buf := ""
	scanner := bufio.NewScanner(strings.NewReader(doc))
	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "TODO") || strings.HasPrefix(trimmedLine, "FIXME") {
			continue
		}
		if strings.HasPrefix(trimmedLine, "---") {
			break
		}
		buf = buf + line + "\n"
	}
	return strings.TrimSpace(buf)
}
