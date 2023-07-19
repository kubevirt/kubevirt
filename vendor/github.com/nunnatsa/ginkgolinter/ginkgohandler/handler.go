package ginkgohandler

import (
	"go/ast"
)

// Handler provide different handling, depend on the way ginkgo was imported, whether
// in imported with "." name, custom name or without any name.
type Handler interface {
	GetFocusContainerName(*ast.CallExpr) (bool, *ast.Ident)
}

// GetGinkgoHandler returns a ginkgor handler according to the way ginkgo was imported in the specific file
func GetGinkgoHandler(file *ast.File) Handler {
	for _, imp := range file.Imports {
		if imp.Path.Value != `"github.com/onsi/ginkgo"` && imp.Path.Value != `"github.com/onsi/ginkgo/v2"` {
			continue
		}

		switch name := imp.Name.String(); {
		case name == ".":
			return dotHandler{}
		case name == "<nil>": // import with no local name
			return nameHandler("ginkgo")
		default:
			return nameHandler(name)
		}
	}

	return nil // no ginkgo import; this file does not use ginkgo
}

// dotHandler is used when importing ginkgo with dot; i.e.
// import . "github.com/onsi/ginkgo"
type dotHandler struct{}

func (h dotHandler) GetFocusContainerName(exp *ast.CallExpr) (bool, *ast.Ident) {
	if fun, ok := exp.Fun.(*ast.Ident); ok {
		return isFocusContainer(fun.Name), fun
	}
	return false, nil
}

// nameHandler is used when importing ginkgo without name; i.e.
// import "github.com/onsi/ginkgo"
//
// or with a custom name; e.g.
// import customname "github.com/onsi/ginkgo"
type nameHandler string

func (h nameHandler) GetFocusContainerName(exp *ast.CallExpr) (bool, *ast.Ident) {
	if sel, ok := exp.Fun.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == string(h) {
			return isFocusContainer(sel.Sel.Name), sel.Sel
		}
	}
	return false, nil
}

func isFocusContainer(name string) bool {
	switch name {
	case "FDescribe", "FContext", "FWhen", "FIt":
		return true
	}
	return false
}
