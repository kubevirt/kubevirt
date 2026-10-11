package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

type definition struct {
	file string
	line int
	name string
	obj  *types.Func
}

func main() {
	domain := flag.String("domain", "", "path component owning the functions")
	flag.Parse()
	if *domain == "" || flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}

	patterns := make([]string, flag.NArg())
	for i, path := range flag.Args() {
		patterns[i] = "./" + strings.TrimSuffix(path, "/") + "/..."
	}
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(2)
	}

	var definitions []definition
	for _, pkg := range pkgs {
		for i, file := range pkg.CompiledGoFiles {
			if inDomain(file, *domain) || isExcluded(file) {
				continue
			}
			for _, decl := range pkg.Syntax[i].Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Recv != nil || !ast.IsExported(fn.Name.Name) {
					continue
				}
				if hasNolint(pkg.Fset, file, fn.Pos()) {
					continue
				}
				if obj, ok := pkg.TypesInfo.Defs[fn.Name].(*types.Func); ok {
					definitions = append(definitions, definition{
						file: file,
						line: pkg.Fset.Position(fn.Pos()).Line,
						name: fn.Name.Name,
						obj:  obj,
					})
				}
			}
		}
	}

	findings := map[*types.Func]map[string][]int{}
	for _, def := range definitions {
		findings[def.obj] = map[string][]int{}
	}
	for _, pkg := range pkgs {
		for id, obj := range pkg.TypesInfo.Uses {
			fn, ok := obj.(*types.Func)
			if !ok {
				continue
			}
			if _, tracked := findings[fn]; !tracked {
				continue
			}
			pos := pkg.Fset.Position(id.Pos())
			findings[fn][pos.Filename] = append(findings[fn][pos.Filename], pos.Line)
		}
	}

	count := 0
	for _, def := range definitions {
		hits := make([]string, 0)
		for file := range findings[def.obj] {
			// TypesInfo.Uses contains references only; the declaration itself is
			// recorded in TypesInfo.Defs. A hit in the defining file is therefore
			// a real same-file use and must not be discarded.
			hits = append(hits, file)
		}
		if len(hits) == 0 {
			continue
		}
		allInDomain := true
		for _, file := range hits {
			if !inDomain(file, *domain) {
				allInDomain = false
				break
			}
		}
		if !allInDomain {
			continue
		}
		sort.Strings(hits)
		count++
		fmt.Printf("\n%s:%d\n  %s\n  used only in %s; make it unexported or add //nolint:domain-funcs\n", relative(def.file), def.line, def.name, *domain)
		for _, file := range hits {
			lines := findings[def.obj][file]
			sort.Ints(lines)
			for _, line := range lines {
				fmt.Printf("    %s:%d\n", relative(file), line)
			}
		}
	}
	if count == 0 {
		fmt.Printf("domain-funcs: no %s-only functions outside %s/\n", *domain, *domain)
		return
	}
	fmt.Printf("\ndomain-funcs: %d function(s) should be localized into %s/\n", count, *domain)
	os.Exit(1)
}

func inDomain(file, domain string) bool {
	for _, part := range strings.Split(filepath.ToSlash(file), "/") {
		if part == domain {
			return true
		}
	}
	return false
}

func isExcluded(file string) bool {
	parts := strings.Split(filepath.ToSlash(file), "/")
	for i := 0; i+1 < len(parts); i++ {
		if (parts[i] == "tests" || parts[i] == "pkg") && strings.HasPrefix(parts[i+1], "lib") {
			return true
		}
	}
	return false
}

func relative(file string) string {
	base, baseErr := filepath.EvalSymlinks(".")
	target, targetErr := filepath.EvalSymlinks(file)
	if baseErr == nil && targetErr == nil {
		file = target
		rel, err := filepath.Rel(base, file)
		if err == nil {
			return filepath.ToSlash(rel)
		}
	}

	rel, err := filepath.Rel(".", file)
	if err == nil {
		if !strings.HasPrefix(rel, "../") {
			return filepath.ToSlash(rel)
		}
	}

	// Go may report files from a different checkout prefix (for example when
	// the repository is mounted into a canonical GOPATH). Strip that prefix
	// using the repository root directory as the anchor.
	file = filepath.ToSlash(file)
	if base, err := filepath.Abs("."); err == nil {
		repo := filepath.Base(filepath.ToSlash(base))
		if index := strings.LastIndex(file, "/"+repo+"/"); index >= 0 {
			return file[index+len(repo)+2:]
		}
	}
	return file
}

func hasNolint(fset *token.FileSet, file string, pos token.Pos) bool {
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	line := fset.Position(pos).Line
	lines := strings.Split(string(data), "\n")
	if line > 0 && line <= len(lines) && strings.Contains(lines[line-1], "//nolint:domain-funcs") {
		return true
	}
	return line > 1 && strings.Contains(lines[line-2], "//nolint:domain-funcs")
}
