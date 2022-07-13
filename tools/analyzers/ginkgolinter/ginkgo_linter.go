package ginkgolinter

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

// The ginkgolinter enforces standards of using ginkgo and gomega.
//
// The current checks are:
// * enforce right length check - warn for assertion of len(something):
//
//   This check finds the following patterns and suggests an alternative
//   * Expect(len(something)).To(Equal(number)) ===> Expect(x).To(HaveLen(number))
//   * ExpectWithOffset(1, len(something)).ShouldNot(Equal(0)) ===> ExpectWithOffset(1, something).ShouldNot(BeEmpty())
//   * Ω(len(something)).NotTo(BeZero()) ===> Ω(something).NotTo(BeEmpty())
//   * Expect(len(something)).To(BeNumerically(">", 0)) ===> Expect(something).ToNot(BeEmpty())
//   * Expect(len(something)).To(BeNumerically(">=", 1)) ===> Expect(something).ToNot(BeEmpty())
//   * Expect(len(something)).To(BeNumerically("==", number)) ===> Expect(something).To(HaveLen(number))

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

const (
	linterName                 = "ginkgo-linter"
	wrongLengthWarningTemplate = "wrong length check; consider using `%s` instead"
	beEmpty                    = "BeEmpty"
	haveLen                    = "HaveLen"
	expect                     = "Expect"
	omega                      = "Ω"
	expectWithOffset           = "ExpectWithOffset"
)

// main assertion function
func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// search for function calls
			assertionExp, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			assertionFunc, ok := assertionExp.Fun.(*ast.SelectorExpr)
			if !ok || !isAssertionFunc(assertionFunc.Sel.Name) {
				return true
			}

			actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
			if !ok {
				return true
			}

			actualArg := getActualArg(actualExpr)
			if actualArg == nil || !isActualIsLenFunc(actualArg) {
				return true
			}

			return checkMatcher(assertionExp, pass)
		})
	}
	return nil, nil
}

// Check if the "actual" argument is a call to the golang built-in len() function
func isActualIsLenFunc(actualArg ast.Expr) bool {
	lenArgExp, ok := actualArg.(*ast.CallExpr)
	if !ok {
		return false
	}

	lenFunc, ok := lenArgExp.Fun.(*ast.Ident)
	return ok && lenFunc.Name == "len"
}

// Check if matcher function is in one of the patterns we want to avoid
func checkMatcher(exp *ast.CallExpr, pass *analysis.Pass) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	matcherFunc, ok := matcher.Fun.(*ast.Ident)
	if !ok {
		return true
	}

	switch matcherFunc.Name {
	case "Equal":
		handleEqualMatcher(matcher, pass, exp)
		return false

	case "BeZero":
		handleBeZero(pass, exp)
		return false

	case "BeNumerically":
		return handleBeNumerically(matcher, pass, exp)

	case "Not":
		reverseAssertionFuncLogic(exp)
		exp.Args[0] = exp.Args[0].(*ast.CallExpr).Args[0]
		return checkMatcher(exp, pass)

	default:
		return true
	}
}

// checks that the function is an assertion's actual function and return the "actual" parameter. If the function
// is not assertion's actual function, return nil.
func getActualArg(actualExpr *ast.CallExpr) ast.Expr {
	actualFunc, ok := actualExpr.Fun.(*ast.Ident)
	if !ok {
		return nil
	}

	switch actualFunc.Name {
	case expect, omega:
		return actualExpr.Args[0]
	case expectWithOffset:
		return actualExpr.Args[1]
	default:
		return nil
	}
}

// Replace the len function call by its parameter, to create a fix suggestion
func replaceLenActualArg(actualExpr *ast.CallExpr) {
	actualFunc, ok := actualExpr.Fun.(*ast.Ident)
	if !ok {
		return
	}

	switch actualFunc.Name {
	case expect, omega:
		arg := actualExpr.Args[0]
		if isActualIsLenFunc(arg) {
			// replace the len function call by its parameter, to create a fix suggestion
			actualExpr.Args[0] = arg.(*ast.CallExpr).Args[0]
		}
	case expectWithOffset:
		arg := actualExpr.Args[1]
		if isActualIsLenFunc(arg) {
			// replace the len function call by its parameter, to create a fix suggestion
			actualExpr.Args[1] = arg.(*ast.CallExpr).Args[0]
		}
	}
}

// For the BeNumerically matcher, we want to avoid the assertion of length to be > 0 or >= 1, or just == number
func handleBeNumerically(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr) bool {
	opExp, ok1 := matcher.Args[0].(*ast.BasicLit)
	valExp, ok2 := matcher.Args[1].(*ast.BasicLit)

	if ok1 && ok2 {
		op := opExp.Value
		val := valExp.Value

		if (op == `">"` && val == "0") || (op == `">="` && val == "1") {
			reverseAssertionFuncLogic(exp)
			exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(beEmpty)
			exp.Args[0].(*ast.CallExpr).Args = nil
			reportLengthCheck(pass, exp)
			return false
		} else if op == `"=="` {
			if val == "0" {
				exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(beEmpty)
				exp.Args[0].(*ast.CallExpr).Args = nil
			} else {
				exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(haveLen)
				exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{valExp}
			}

			reportLengthCheck(pass, exp)
			return false
		} else if op == `"!="` {
			reverseAssertionFuncLogic(exp)

			if val == "0" {
				exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(beEmpty)
				exp.Args[0].(*ast.CallExpr).Args = nil
			} else {
				exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(haveLen)
				exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{valExp}
			}

			reportLengthCheck(pass, exp)
			return false
		}
	}
	return true
}

func reverseAssertionFuncLogic(exp *ast.CallExpr) {
	assertionFunc := exp.Fun.(*ast.SelectorExpr).Sel
	assertionFunc.Name = changeAssertionLogic(assertionFunc.Name)
}

func handleEqualMatcher(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr) {
	equalTo, ok := matcher.Args[0].(*ast.BasicLit)
	if ok {
		if equalTo.Value == "0" {
			exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(beEmpty)
			exp.Args[0].(*ast.CallExpr).Args = nil
		} else {
			exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(haveLen)
			exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{equalTo}
		}
	} else {
		exp.Args[0].(*ast.CallExpr).Fun = ast.NewIdent(haveLen)
		exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{matcher.Args[0]}
	}
	reportLengthCheck(pass, exp)
}

func handleBeZero(pass *analysis.Pass, exp *ast.CallExpr) {
	exp.Args[0].(*ast.CallExpr).Args = nil
	exp.Args[0].(*ast.CallExpr).Fun.(*ast.Ident).Name = beEmpty

	reportLengthCheck(pass, exp)
}

func report(pass *analysis.Pass, pos token.Pos, warning string) {
	pass.Reportf(pos, "%s: %s", linterName, warning)
}

func reportLengthCheck(pass *analysis.Pass, expr *ast.CallExpr) {
	replaceLenActualArg(expr.Fun.(*ast.SelectorExpr).X.(*ast.CallExpr))
	report(pass, expr.Pos(), fmt.Sprintf(wrongLengthWarningTemplate, goFmt(pass.Fset, expr)))
}

func goFmt(fset *token.FileSet, x ast.Expr) string {
	var b bytes.Buffer
	printer.Fprint(&b, fset, x)
	return b.String()
}

var reverseLogicAssertions = map[string]string{
	"To":        "ToNot",
	"ToNot":     "To",
	"NotTo":     "To",
	"Should":    "ShouldNot",
	"ShouldNot": "Should",
}

// ChangeAssertionLogic get gomega assertion function name, and returns the reverse logic function name
func changeAssertionLogic(funcName string) string {
	if revFunc, ok := reverseLogicAssertions[funcName]; ok {
		return revFunc
	}
	return funcName
}

func isAssertionFunc(name string) bool {
	_, ok := reverseLogicAssertions[name]
	return ok
}
