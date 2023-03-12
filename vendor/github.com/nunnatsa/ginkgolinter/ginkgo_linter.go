package ginkgolinter

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/go-toolsmith/astcopy"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	gotypes "go/types"
	"golang.org/x/tools/go/analysis"

	"github.com/nunnatsa/ginkgolinter/gomegahandler"
	"github.com/nunnatsa/ginkgolinter/reverseassertion"
	"github.com/nunnatsa/ginkgolinter/types"
)

// The ginkgolinter enforces standards of using ginkgo and gomega.
//
// For more details, look at the README.md file

const (
	linterName                  = "ginkgo-linter"
	wrongLengthWarningTemplate  = linterName + ": wrong length assertion; consider using `%s` instead"
	wrongNilWarningTemplate     = linterName + ": wrong nil assertion; consider using `%s` instead"
	wrongBoolWarningTemplate    = linterName + ": wrong boolean assertion; consider using `%s` instead"
	wrongErrWarningTemplate     = linterName + ": wrong error assertion; consider using `%s` instead"
	wrongCompareWarningTemplate = linterName + ": wrong comparison assertion; consider using `%s` instead"
	beEmpty                     = "BeEmpty"
	beNil                       = "BeNil"
	beTrue                      = "BeTrue"
	beFalse                     = "BeFalse"
	beZero                      = "BeZero"
	beNumerically               = "BeNumerically"
	beIdenticalTo               = "BeIdenticalTo"
	equal                       = "Equal"
	not                         = "Not"
	haveLen                     = "HaveLen"
	succeed                     = "Succeed"
	haveOccurred                = "HaveOccurred"
	expect                      = "Expect"
	omega                       = "Ω"
	expectWithOffset            = "ExpectWithOffset"
)

// Analyzer is the interface to go_vet
var Analyzer = NewAnalyzer()

type ginkgoLinter struct {
	config *types.Config
}

// NewAnalyzer returns an Analyzer - the package interface with nogo
func NewAnalyzer() *analysis.Analyzer {
	linter := ginkgoLinter{
		config: &types.Config{
			SuppressLen:     false,
			SuppressNil:     false,
			SuppressErr:     false,
			SuppressCompare: false,
			AllowHaveLen0:   false,
		},
	}

	a := &analysis.Analyzer{
		Name:             "ginkgolinter",
		Doc:              doc,
		Run:              linter.run,
		RunDespiteErrors: true,
	}

	a.Flags.Init("ginkgolinter", flag.ExitOnError)
	a.Flags.Var(&linter.config.SuppressLen, "suppress-len-assertion", "Suppress warning for wrong length assertions")
	a.Flags.Var(&linter.config.SuppressNil, "suppress-nil-assertion", "Suppress warning for wrong nil assertions")
	a.Flags.Var(&linter.config.SuppressErr, "suppress-err-assertion", "Suppress warning for wrong error assertions")
	a.Flags.Var(&linter.config.SuppressCompare, "suppress-compare-assertion", "Suppress warning for wrong comparison assertions")
	a.Flags.Var(&linter.config.AllowHaveLen0, "allow-havelen-0", "Do not warn for HaveLen(0); default = false")

	return a
}

const doc = `enforces standards of using ginkgo and gomega
currently, the linter searches for following:

* wrong length assertions. We want to assert the item rather than its length.
For example:
	Expect(len(x)).Should(Equal(1))
This should be replaced with:
	Expect(x)).Should(HavelLen(1))
	
* wrong nil assertions. We want to assert the item rather than a comparison result.
For example:
	Expect(x == nil).Should(BeTrue())
This should be replaced with:
	Expect(x).Should(BeNil())

* wrong error assertions. For example:
	Expect(err == nil).Should(BeTrue())
This should be replaced with:
	Expect(err).ShouldNot(HaveOccurred())

* wrong boolean comparison, for example:
	Expect(x == 8).Should(BeTrue())
This should be replaced with:
	Expect(x).Should(BeEqual(8))

* replaces Equal(true/false) with BeTrue()/BeFalse()

* replaces HaveLen(0) with BeEmpty()
`

// main assertion function
func (l *ginkgoLinter) run(pass *analysis.Pass) (interface{}, error) {
	if l.config.AllTrue() {
		return nil, nil
	}

	for _, file := range pass.Files {
		fileConfig := l.config.Clone()

		cm := ast.NewCommentMap(pass.Fset, file, file.Comments)

		fileConfig.UpdateFromFile(cm)
		if fileConfig.AllTrue() {
			continue
		}

		handler := gomegahandler.GetGomegaHandler(file)
		if handler == nil { // no gomega import => no use in gomega in this file; nothing to do here
			continue
		}

		ast.Inspect(file, func(n ast.Node) bool {

			stmt, ok := n.(*ast.ExprStmt)
			if !ok {
				return true
			}

			config := fileConfig.Clone()

			if comments, ok := cm[stmt]; ok {
				config.UpdateFromComment(comments)
			}

			// search for function calls
			assertionExp, ok := stmt.X.(*ast.CallExpr)
			if !ok {
				return true
			}

			assertionFunc, ok := assertionExp.Fun.(*ast.SelectorExpr)
			if !ok || !isAssertionFunc(assertionFunc.Sel.Name) {
				return true
			}

			actualArg := getActualArg(assertionFunc, handler)
			if actualArg == nil {
				return true
			}

			return checkExpression(pass, config, actualArg, assertionExp, handler)

		})
	}
	return nil, nil
}

func checkExpression(pass *analysis.Pass, config types.Config, actualArg ast.Expr, assertionExp *ast.CallExpr, handler gomegahandler.Handler) bool {
	assertionExp = astcopy.CallExpr(assertionExp)
	oldExpr := goFmt(pass.Fset, assertionExp)
	if !bool(config.SuppressLen) && isActualIsLenFunc(actualArg) {

		return checkLengthMatcher(assertionExp, pass, handler, oldExpr)
	} else {
		if nilable, compOp := getNilableFromComparison(actualArg); nilable != nil {
			if isExprError(pass, nilable) {
				if config.SuppressErr {
					return true
				}
			} else if config.SuppressNil {
				return true
			}

			return checkNilMatcher(assertionExp, pass, nilable, handler, compOp == token.NEQ, oldExpr)

		} else if first, second, op, ok := isComparison(pass, actualArg); ok {
			return bool(config.SuppressCompare) || checkComparison(assertionExp, pass, handler, first, second, op, oldExpr)

		} else if isExprError(pass, actualArg) {
			return bool(config.SuppressErr) || checkNilError(pass, assertionExp, handler, actualArg, oldExpr)

		} else {
			return handleAssertionOnly(pass, config, assertionExp, handler, actualArg, oldExpr)
		}
	}
}

func checkComparison(exp *ast.CallExpr, pass *analysis.Pass, handler gomegahandler.Handler, first ast.Expr, second ast.Expr, op token.Token, oldExp string) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	matcherFuncName, ok := handler.GetActualFuncName(matcher)
	if !ok {
		return true
	}

	switch matcherFuncName {
	case beTrue:
	case beFalse:
		reverseAssertionFuncLogic(exp)
	case equal:
		boolean, found := matcher.Args[0].(*ast.Ident)
		if !found {
			return true
		}

		if boolean.Name == "false" {
			reverseAssertionFuncLogic(exp)
		} else if boolean.Name != "true" {
			return true
		}

	case not:
		reverseAssertionFuncLogic(exp)
		exp.Args[0] = exp.Args[0].(*ast.CallExpr).Args[0]
		return checkComparison(exp, pass, handler, first, second, op, oldExp)

	default:
		return true
	}

	fun, ok := exp.Fun.(*ast.SelectorExpr)
	if !ok {
		return true
	}

	call, ok := fun.X.(*ast.CallExpr)
	if !ok {
		return true
	}

	call.Args = []ast.Expr{first}
	switch op {
	case token.EQL:
		handleEqualComparison(pass, matcher, first, second, handler)

	case token.NEQ:
		reverseAssertionFuncLogic(exp)
		handleEqualComparison(pass, matcher, first, second, handler)
	case token.GTR, token.GEQ, token.LSS, token.LEQ:
		handler.ReplaceFunction(matcher, ast.NewIdent(beNumerically))
		matcher.Args = []ast.Expr{
			&ast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`"%s"`, op.String())},
			second,
		}
	default:
		return true
	}

	report(pass, exp, wrongCompareWarningTemplate, oldExp)
	return false
}

func handleEqualComparison(pass *analysis.Pass, matcher *ast.CallExpr, first ast.Expr, second ast.Expr, handler gomegahandler.Handler) {
	if isZero(pass, second) {
		handler.ReplaceFunction(matcher, ast.NewIdent(beZero))
		matcher.Args = nil
	} else {
		t := pass.TypesInfo.TypeOf(first)
		if gotypes.IsInterface(t) {
			handler.ReplaceFunction(matcher, ast.NewIdent(beIdenticalTo))
		} else if _, ok := t.(*gotypes.Pointer); ok {
			handler.ReplaceFunction(matcher, ast.NewIdent(beIdenticalTo))
		} else {
			handler.ReplaceFunction(matcher, ast.NewIdent(equal))
		}

		matcher.Args = []ast.Expr{second}
	}
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
func checkLengthMatcher(exp *ast.CallExpr, pass *analysis.Pass, handler gomegahandler.Handler, oldExp string) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	matcherFuncName, ok := handler.GetActualFuncName(matcher)
	if !ok {
		return true
	}

	switch matcherFuncName {
	case equal:
		handleEqualMatcher(matcher, pass, exp, handler, oldExp)
		return false

	case beZero:
		handleBeZero(pass, exp, handler, oldExp)
		return false

	case beNumerically:
		return handleBeNumerically(matcher, pass, exp, handler, oldExp)

	case not:
		reverseAssertionFuncLogic(exp)
		exp.Args[0] = exp.Args[0].(*ast.CallExpr).Args[0]
		return checkLengthMatcher(exp, pass, handler, oldExp)

	default:
		return true
	}
}

// Check if matcher function is in one of the patterns we want to avoid
func checkNilMatcher(exp *ast.CallExpr, pass *analysis.Pass, nilable ast.Expr, handler gomegahandler.Handler, notEqual bool, oldExp string) bool {
	matcher, ok := exp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	matcherFuncName, ok := handler.GetActualFuncName(matcher)
	if !ok {
		return true
	}

	switch matcherFuncName {
	case equal:
		handleEqualNilMatcher(matcher, pass, exp, handler, nilable, notEqual, oldExp)

	case beTrue:
		handleNilBeBoolMatcher(pass, exp, handler, nilable, notEqual, oldExp)

	case beFalse:
		reverseAssertionFuncLogic(exp)
		handleNilBeBoolMatcher(pass, exp, handler, nilable, notEqual, oldExp)

	case not:
		reverseAssertionFuncLogic(exp)
		exp.Args[0] = exp.Args[0].(*ast.CallExpr).Args[0]
		return checkNilMatcher(exp, pass, nilable, handler, notEqual, oldExp)

	default:
		return true
	}
	return false
}

func checkNilError(pass *analysis.Pass, assertionExp *ast.CallExpr, handler gomegahandler.Handler, actualArg ast.Expr, oldExpr string) bool {
	if len(assertionExp.Args) == 0 {
		return true
	}

	equalFuncExpr, ok := assertionExp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	funcName, ok := handler.GetActualFuncName(equalFuncExpr)
	if !ok {
		return true
	}

	switch funcName {
	case beNil: // no additional processing needed.
	case equal:

		if len(equalFuncExpr.Args) == 0 {
			return true
		}

		nilable, ok := equalFuncExpr.Args[0].(*ast.Ident)
		if !ok || nilable.Name != "nil" {
			return true
		}

	case not:
		reverseAssertionFuncLogic(assertionExp)
		assertionExp.Args[0] = assertionExp.Args[0].(*ast.CallExpr).Args[0]
		return checkNilError(pass, assertionExp, handler, actualArg, oldExpr)
	default:
		return true
	}

	var newFuncName string
	if _, ok := actualArg.(*ast.CallExpr); ok {
		newFuncName = succeed
	} else {
		reverseAssertionFuncLogic(assertionExp)
		newFuncName = haveOccurred
	}

	handler.ReplaceFunction(equalFuncExpr, ast.NewIdent(newFuncName))
	equalFuncExpr.Args = nil

	report(pass, assertionExp, wrongErrWarningTemplate, oldExpr)
	return false
}

// handleAssertionOnly checks use-cases when the actual value is valid, but only the assertion should be fixed
// it handles:
//
//	Equal(nil) => BeNil()
//	Equal(true) => BeTrue()
//	Equal(false) => BeFalse()
//	HaveLen(0) => BeEmpty()
func handleAssertionOnly(pass *analysis.Pass, config types.Config, assertionExp *ast.CallExpr, handler gomegahandler.Handler, actualArg ast.Expr, oldExpr string) bool {
	if len(assertionExp.Args) == 0 {
		return true
	}

	equalFuncExpr, ok := assertionExp.Args[0].(*ast.CallExpr)
	if !ok {
		return true
	}

	funcName, ok := handler.GetActualFuncName(equalFuncExpr)
	if !ok {
		return true
	}

	switch funcName {
	case equal:
		if len(equalFuncExpr.Args) == 0 {
			return true
		}

		tkn, ok := equalFuncExpr.Args[0].(*ast.Ident)
		if !ok {
			return true
		}

		var replacement string
		var template string
		switch tkn.Name {
		case "nil":
			if config.SuppressNil {
				return true
			}
			replacement = beNil
			template = wrongNilWarningTemplate
		case "true":
			replacement = beTrue
			template = wrongBoolWarningTemplate
		case "false":
			replacement = beFalse
			template = wrongBoolWarningTemplate
		default:
			return true
		}

		handler.ReplaceFunction(equalFuncExpr, ast.NewIdent(replacement))
		equalFuncExpr.Args = nil

		report(pass, assertionExp, template, oldExpr)

		return false

	case haveLen:
		if config.AllowHaveLen0 {
			return true
		}

		if len(equalFuncExpr.Args) > 0 {
			if isZero(pass, equalFuncExpr.Args[0]) {
				handler.ReplaceFunction(equalFuncExpr, ast.NewIdent(beEmpty))
				equalFuncExpr.Args = nil
				report(pass, assertionExp, wrongLengthWarningTemplate, oldExpr)
				return false
			}
		}

		return true

	case not:
		reverseAssertionFuncLogic(assertionExp)
		assertionExp.Args[0] = assertionExp.Args[0].(*ast.CallExpr).Args[0]
		return handleAssertionOnly(pass, config, assertionExp, handler, actualArg, oldExpr)
	default:
		return true
	}
}

func isZero(pass *analysis.Pass, arg ast.Expr) bool {
	if val, ok := arg.(*ast.BasicLit); ok && val.Kind == token.INT && val.Value == "0" {
		return true
	}
	info, ok := pass.TypesInfo.Types[arg]
	if ok {
		if t, ok := info.Type.(*gotypes.Basic); ok && t.Kind() == gotypes.Int && info.Value != nil {
			if i, ok := constant.Int64Val(info.Value); ok && i == 0 {
				return true
			}
		}
	} else if val, ok := arg.(*ast.Ident); ok && val.Obj != nil && val.Obj.Kind == ast.Con {
		if spec, ok := val.Obj.Decl.(*ast.ValueSpec); ok {
			if len(spec.Values) == 1 {
				if value, ok := spec.Values[0].(*ast.BasicLit); ok && value.Kind == token.INT && value.Value == "0" {
					return true
				}
			}
		}
	}

	return false
}

// checks that the function is an assertion's actual function and return the "actual" parameter. If the function
// is not assertion's actual function, return nil.
func getActualArg(assertionFunc *ast.SelectorExpr, handler gomegahandler.Handler) ast.Expr {
	actualExpr, ok := assertionFunc.X.(*ast.CallExpr)
	if !ok {
		return nil
	}

	funcName, ok := handler.GetActualFuncName(actualExpr)
	if !ok {
		return nil
	}

	switch funcName {
	case expect, omega:
		return actualExpr.Args[0]
	case expectWithOffset:
		return actualExpr.Args[1]
	default:
		return nil
	}
}

// Replace the len function call by its parameter, to create a fix suggestion
func replaceLenActualArg(actualExpr *ast.CallExpr, handler gomegahandler.Handler) {
	name, ok := handler.GetActualFuncName(actualExpr)
	if !ok {
		return
	}

	switch name {
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

// Replace the nil comparison with the compared object, to create a fix suggestion
func replaceNilActualArg(actualExpr *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr) bool {
	actualFuncName, ok := handler.GetActualFuncName(actualExpr)
	if !ok {
		return false
	}

	switch actualFuncName {
	case expect, omega:
		actualExpr.Args[0] = nilable
		return true

	case expectWithOffset:
		actualExpr.Args[1] = nilable
		return true

	default:
		return false
	}
}

// For the BeNumerically matcher, we want to avoid the assertion of length to be > 0 or >= 1, or just == number
func handleBeNumerically(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, oldExp string) bool {
	opExp, ok1 := matcher.Args[0].(*ast.BasicLit)
	valExp, ok2 := matcher.Args[1].(*ast.BasicLit)

	if ok1 && ok2 {
		op := opExp.Value
		val := valExp.Value

		if (op == `">"` && val == "0") || (op == `">="` && val == "1") {
			reverseAssertionFuncLogic(exp)
			handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(beEmpty))
			exp.Args[0].(*ast.CallExpr).Args = nil
			reportLengthAssertion(pass, exp, handler, oldExp)
			return false
		} else if op == `"=="` {
			chooseNumericMatcher(pass, exp, handler, valExp)
			reportLengthAssertion(pass, exp, handler, oldExp)

			return false
		} else if op == `"!="` {
			reverseAssertionFuncLogic(exp)
			chooseNumericMatcher(pass, exp, handler, valExp)
			reportLengthAssertion(pass, exp, handler, oldExp)

			return false
		}
	}
	return true
}

func chooseNumericMatcher(pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, valExp ast.Expr) {
	caller := exp.Args[0].(*ast.CallExpr)
	if isZero(pass, valExp) {
		handler.ReplaceFunction(caller, ast.NewIdent(beEmpty))
		exp.Args[0].(*ast.CallExpr).Args = nil
	} else {
		handler.ReplaceFunction(caller, ast.NewIdent(haveLen))
		exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{valExp}
	}
}

func reverseAssertionFuncLogic(exp *ast.CallExpr) {
	assertionFunc := exp.Fun.(*ast.SelectorExpr).Sel
	assertionFunc.Name = reverseassertion.ChangeAssertionLogic(assertionFunc.Name)
}

func handleEqualMatcher(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, oldExp string) {
	equalTo, ok := matcher.Args[0].(*ast.BasicLit)
	if ok {
		chooseNumericMatcher(pass, exp, handler, equalTo)
	} else {
		handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(haveLen))
		exp.Args[0].(*ast.CallExpr).Args = []ast.Expr{matcher.Args[0]}
	}
	reportLengthAssertion(pass, exp, handler, oldExp)
}

func handleBeZero(pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, oldExp string) {
	exp.Args[0].(*ast.CallExpr).Args = nil
	handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(beEmpty))
	reportLengthAssertion(pass, exp, handler, oldExp)
}

func handleEqualNilMatcher(matcher *ast.CallExpr, pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr, notEqual bool, oldExp string) {
	equalTo, ok := matcher.Args[0].(*ast.Ident)
	if !ok {
		return
	}

	if equalTo.Name == "false" {
		reverseAssertionFuncLogic(exp)
	} else if equalTo.Name != "true" {
		return
	}

	newFuncName, isItError := handleNilComparisonErr(pass, exp, nilable)

	handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(newFuncName))
	exp.Args[0].(*ast.CallExpr).Args = nil

	reportNilAssertion(pass, exp, handler, nilable, notEqual, oldExp, isItError)
}

func handleNilBeBoolMatcher(pass *analysis.Pass, exp *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr, notEqual bool, oldExp string) {
	newFuncName, isItError := handleNilComparisonErr(pass, exp, nilable)
	handler.ReplaceFunction(exp.Args[0].(*ast.CallExpr), ast.NewIdent(newFuncName))
	exp.Args[0].(*ast.CallExpr).Args = nil

	reportNilAssertion(pass, exp, handler, nilable, notEqual, oldExp, isItError)
}

func handleNilComparisonErr(pass *analysis.Pass, exp *ast.CallExpr, nilable ast.Expr) (string, bool) {
	newFuncName := beNil
	isItError := isExprError(pass, nilable)
	if isItError {
		if _, ok := nilable.(*ast.CallExpr); ok {
			newFuncName = succeed
		} else {
			reverseAssertionFuncLogic(exp)
			newFuncName = haveOccurred
		}
	}

	return newFuncName, isItError
}
func isAssertionFunc(name string) bool {
	switch name {
	case "To", "ToNot", "NotTo", "Should", "ShouldNot":
		return true
	}
	return false
}

func reportLengthAssertion(pass *analysis.Pass, expr *ast.CallExpr, handler gomegahandler.Handler, oldExpr string) {
	replaceLenActualArg(expr.Fun.(*ast.SelectorExpr).X.(*ast.CallExpr), handler)

	report(pass, expr, wrongLengthWarningTemplate, oldExpr)
}

func reportNilAssertion(pass *analysis.Pass, expr *ast.CallExpr, handler gomegahandler.Handler, nilable ast.Expr, notEqual bool, oldExpr string, isItError bool) {
	changed := replaceNilActualArg(expr.Fun.(*ast.SelectorExpr).X.(*ast.CallExpr), handler, nilable)
	if !changed {
		return
	}

	if notEqual {
		reverseAssertionFuncLogic(expr)
	}
	template := wrongNilWarningTemplate
	if isItError {
		template = wrongErrWarningTemplate
	}

	report(pass, expr, template, oldExpr)
}

func report(pass *analysis.Pass, expr *ast.CallExpr, messageTemplate, oldExpr string) {
	newExp := goFmt(pass.Fset, expr)
	pass.Report(analysis.Diagnostic{
		Pos:     expr.Pos(),
		Message: fmt.Sprintf(messageTemplate, newExp),
		SuggestedFixes: []analysis.SuggestedFix{
			{
				Message: fmt.Sprintf("should replace %s with %s", oldExpr, newExp),
				TextEdits: []analysis.TextEdit{
					{
						Pos:     expr.Pos(),
						End:     expr.End(),
						NewText: []byte(newExp),
					},
				},
			},
		},
	})
}

func getNilableFromComparison(actualArg ast.Expr) (ast.Expr, token.Token) {
	bin, ok := actualArg.(*ast.BinaryExpr)
	if !ok {
		return nil, token.ILLEGAL
	}

	if bin.Op == token.EQL || bin.Op == token.NEQ {
		if isNil(bin.Y) {
			return bin.X, bin.Op
		} else if isNil(bin.X) {
			return bin.Y, bin.Op
		}
	}

	return nil, token.ILLEGAL
}

func isNil(expr ast.Expr) bool {
	nilObject, ok := expr.(*ast.Ident)
	return ok && nilObject.Name == "nil" && nilObject.Obj == nil
}

func isComparison(pass *analysis.Pass, actualArg ast.Expr) (ast.Expr, ast.Expr, token.Token, bool) {
	bin, ok := actualArg.(*ast.BinaryExpr)
	if !ok {
		return nil, nil, token.ILLEGAL, false
	}

	first, second, op := bin.X, bin.Y, bin.Op
	replace := false
	switch realFirst := first.(type) {
	case *ast.Ident: // check if const
		info, ok := pass.TypesInfo.Types[realFirst]
		if ok {
			if _, ok := info.Type.(*gotypes.Basic); ok && info.Value != nil {
				replace = true
			}
		}

	case *ast.BasicLit:
		replace = true
	}

	if replace {
		first, second = second, first
	}

	switch op {
	case token.EQL:
	case token.NEQ:
	case token.GTR, token.GEQ, token.LSS, token.LEQ:
		if replace {
			op = reverseassertion.ChangeCompareOperator(op)
		}
	default:
		return nil, nil, token.ILLEGAL, false
	}
	return first, second, op, true
}

func goFmt(fset *token.FileSet, x ast.Expr) string {
	var b bytes.Buffer
	_ = printer.Fprint(&b, fset, x)
	return b.String()
}

var errorType *gotypes.Interface

func init() {
	errorType = gotypes.Universe.Lookup("error").Type().Underlying().(*gotypes.Interface)
}

func isError(t gotypes.Type) bool {
	return gotypes.Implements(t, errorType)
}

func isExprError(pass *analysis.Pass, expr ast.Expr) bool {
	actualArgType := pass.TypesInfo.TypeOf(expr)
	switch t := actualArgType.(type) {
	case *gotypes.Named:
		if isError(actualArgType) {
			return true
		}
	case *gotypes.Tuple:
		if t.Len() > 0 {
			switch t0 := t.At(0).Type().(type) {
			case *gotypes.Named, *gotypes.Pointer:
				if isError(t0) {
					return true
				}
			}
		}
	}
	return false
}
