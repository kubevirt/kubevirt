package gomegahandler

import "go/ast"

// Handler provide different handling, depend on the way gomega was imported, whether
// in imported with "." name, custome name or without any name.
type Handler interface {
	GetActualFuncName(*ast.CallExpr) (string, bool)
	ReplaceFunction(*ast.CallExpr, *ast.Ident)
}

// GetGomegaHandler returns a gomegar handler according to the way gomega was imported in the specific file
func GetGomegaHandler(file *ast.File) Handler {
	for _, imp := range file.Imports {
		if imp.Path.Value != `"github.com/onsi/gomega"` {
			continue
		}

		switch name := imp.Name.String(); {
		case name == ".":
			return dotHandler{}
		case name == "<nil>": // import with no local name
			return nameHandler("gomega")
		default:
			return nameHandler(name)
		}
	}

	return nil // no gomega import; this file does not use gomega
}

// dotHandler is used when importing gomega with dot; i.e.
// import . "github.com/onsi/gomega"
//
type dotHandler struct{}

func (dotHandler) GetActualFuncName(expr *ast.CallExpr) (string, bool) {
	actualFunc, ok := expr.Fun.(*ast.Ident)
	if !ok {
		return "", false
	}

	return actualFunc.Name, true
}

func (dotHandler) ReplaceFunction(caller *ast.CallExpr, newExpr *ast.Ident) {
	caller.Fun = newExpr
}

// nameHandler is used when importing gomega without name; i.e.
// import "github.com/onsi/gomega"
//
// or with a custom name; e.g.
// import customname "github.com/onsi/gomega"
//
type nameHandler string

func (g nameHandler) GetActualFuncName(expr *ast.CallExpr) (string, bool) {
	selector, ok := expr.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}

	x, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}

	if x.Name != string(g) {
		return "", false
	}

	return selector.Sel.Name, true
}

func (nameHandler) ReplaceFunction(caller *ast.CallExpr, newExpr *ast.Ident) {
	caller.Fun.(*ast.SelectorExpr).Sel = newExpr
}
