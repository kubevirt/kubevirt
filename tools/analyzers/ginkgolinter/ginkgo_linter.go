package ginkgolinter

import (
	"go/ast"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/analysis"
)

// Analyzer is the package interface with nogo
var Analyzer = &analysis.Analyzer{
	Name: "ginkgolinter",
	Doc: `enforces standards of using ginkgo and gomega
currently, the linter searches for wrong length checks; e.g.
	Expect(len(x).Should(Equal(1))
This should be replaced with:
	Expect(x).Should(HavelLen(1)`,
	Run: run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// search for function calls
			exp, ok := n.(*ast.CallExpr)
			if ok {
				selected, ok := exp.Fun.(*ast.SelectorExpr)
				if ok {
					if !isCheckFunc(selected.Sel.Name) {
						return true
					}

					caller, ok := selected.X.(*ast.CallExpr)
					if ok {
						callerFunc, ok := caller.Fun.(*ast.Ident)
						if ok {
							arg := getActualArg(callerFunc, caller.Args)
							if arg == nil {
								return true
							}

							// check that the "actual" is a function call
							lenArgExp, ok := arg.(*ast.CallExpr)
							if ok {
								lenFunc, ok := lenArgExp.Fun.(*ast.Ident)
								// check that the "actual" function is the built-in len() function
								if ok && lenFunc.Name == "len" {
									return checkMatcher(exp, pass)
								}
							}
						}
					}
				}
			}
			return true
		})
	}
	return nil, nil
}

func checkMatcher(exp *ast.CallExpr, pass *analysis.Pass) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if ok {
		matcherFunc, ok := matcher.Fun.(*ast.Ident)
		if ok {
			switch matcherFunc.Name {
			case "Equal":
				handleEqualMatcher(matcher, pass, exp)
				return false
			case "BeZero":
				pass.Reportf(exp.Pos(), "ginkgo-linter: wrong length check; consider using BeEmpty() instead")
				return false
			case "BeNumerically":
				return handleBeNumerically(matcher, pass, exp)
			}
		}
	}
	return false
}

func getActualArg(callerFunc *ast.Ident, callerArgs []ast.Expr) ast.Expr {
	switch callerFunc.Name {
	case "Expect", "Ω":
		return callerArgs[0]
	case "ExpectWithOffset":
		return callerArgs[1]
	default:
		return nil
	}
}

func handleBeNumerically(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr) bool {
	op, ok1 := matcher.Args[0].(*ast.BasicLit)
	val, ok2 := matcher.Args[1].(*ast.BasicLit)

	if ok1 && ok2 {
		if (op.Value == `">"` && val.Value == "0") || (op.Value == `">="` && val.Value == "1") {
			pass.Reportf(exp.Pos(), "ginkgo-linter: wrong length check; consider using Not(BeEmpty()) instead")
			return false
		} else if op.Value == `"=="` {
			pass.Reportf(exp.Pos(), "ginkgo-linter: wrong length check; consider using HaveLen() instead")
			return false
		}
	}
	return true
}

func handleEqualMatcher(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr) {
	suggest := "HaveLen()"

	equalTo, ok := matcher.Args[0].(*ast.BasicLit)
	if ok && equalTo.Kind == token.INT {
		val, err := strconv.Atoi(equalTo.Value)
		if err != nil {
			// should never get here; this is just for case
			pass.Reportf(exp.Pos(), "ginkgo-linter: wrong data. '%s' should be integer", equalTo.Value)
			return
		} else if val == 0 {
			suggest = "BeEmpty()"
		}
	}

	pass.Reportf(exp.Pos(), "ginkgo-linter: wrong length check; consider using %s instead", suggest)
}

func isCheckFunc(name string) bool {
	switch name {
	case "To", "ToNot", "NotTo", "Should", "ShouldNot":
		return true
	}
	return false
}
