// Package pswap implements the parameter-swap analyzer. It reports when a
// a named parameter is passed to a function that offers a different parameter
// with the same name.

package pswap

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `pswap reports parameters that were likely swapped

While this isn't necessarily a problem, and sometimes is very intentional,
the results of crossing your parameters can be disastrous.`

type pswapAnalyzer struct {
	*analysis.Analyzer
	ExactTypeOnly         bool
	IncludeGeneratedFiles bool
}

func Analyzer() *pswapAnalyzer {
	a := &pswapAnalyzer{
		Analyzer: &analysis.Analyzer{
			Name:     "varfmt",
			Doc:      doc,
			Requires: []*analysis.Analyzer{inspect.Analyzer},
		},
	}
	a.Flags.BoolVar(&a.ExactTypeOnly, "exact", false, "suppress pswap reports when types aren't an exact match")
	a.Flags.BoolVar(&a.IncludeGeneratedFiles, "gen", false, "include reports from generated files")

	a.Run = a.run

	return a
}

type (
	arg          types.Var
	param        types.Var
	paramMatcher func(*param) bool
)

func findParam(fun *types.Func, argIndex int, matchers ...paramMatcher) (int, *param) {
	params := fun.Signature().Params()
	for _, match := range matchers {
		// prefer matching index when available over, e.g. similarly case mismatch in earlier param
		if argIndex < params.Len() && match((*param)(params.At(argIndex))) {
			return argIndex, (*param)(params.At(argIndex))
		}
		for i := range params.Len() {
			p := params.At(i)
			if match((*param)(p)) {
				return i, (*param)(p)
			}
		}
	}
	return -1, nil
}

func (a *arg) Name() string       { return (*types.Var)(a).Name() }
func (p *param) Name() string     { return (*types.Var)(p).Name() }
func (a *arg) Type() types.Type   { return (*types.Var)(a).Type() }
func (p *param) Type() types.Type { return (*types.Var)(p).Type() }

func (a *arg) CaseMatch(p *param) bool {
	return a.Name() == p.Name() && types.AssignableTo(a.Type(), p.Type())
}

func (a arg) NoCaseMatch(p *param) bool {
	return strings.EqualFold(a.Name(), p.Name()) && types.AssignableTo(a.Type(), p.Type())
}

func (a arg) CaseTypeMatch(p *param) bool {
	return a.Name() == p.Name() && a.Type() == p.Type()
}

func (a arg) NoCaseTypeMatch(p *param) bool {
	return strings.EqualFold(a.Name(), p.Name()) && a.Type() == p.Type()
}

func (v *pswapAnalyzer) run(pass *analysis.Pass) (any, error) {
	var lastCall struct {
		File      *ast.File
		Generated bool
	}
	isCallGenerated := func(n ast.Node) bool {
		pos := n.Pos()
		// opt: expect adjacent tokens to be from same file
		if lastCall.File != nil && lastCall.File.FileStart <= pos && pos <= lastCall.File.FileEnd {
			return lastCall.Generated
		}
		for _, f := range pass.Files {
			if f.FileStart <= pos && pos <= f.FileEnd {
				lastCall.File = f
				lastCall.Generated = ast.IsGenerated(f)
				return lastCall.Generated
			}
		}
		return false
	}

	funOf := func(c *ast.CallExpr) *types.Func {
		switch f := c.Fun.(type) {
		case *ast.Ident:
			return pass.TypesInfo.ObjectOf(f).(*types.Func)
		case *ast.SelectorExpr:
			return selobj(pass.TypesInfo, f).(*types.Func)
			// case *ast.CallExpr, *ast.TypeAssertExpr, *ast.ParenExpr, *ast.IndexExpr:
			// case *ast.FuncLit, *ast.ArrayType, *ast.InterfaceType, *ast.MapType, *ast.ChanType, *ast.StructType:
			// default:
			// 	pt("unhandled callfun "+pp(n.Fun), n.Fun)
		}
		return nil
	}

	varOf := func(x ast.Expr) *types.Var {
		switch x := x.(type) {
		case *ast.Ident:
			return pass.TypesInfo.ObjectOf(x).(*types.Var)
		case *ast.SelectorExpr:
			return pass.TypesInfo.ObjectOf(x.Sel).(*types.Var)
		case *ast.BasicLit:
			return nil
		}
		return nil
	}

	report := func(n ast.Node, arg *arg, ai int, f *types.Func, pi int) {
		// similar to t.String, but omits package names
		var recvType func(t types.Type) string
		recvType = func(t types.Type) string {
			switch t := t.(type) {
			case *types.Pointer:
				return "(*" + recvType(t.Elem()) + ")."
			case *types.Named:
				// what of t.TypeParams(), t.TypeArgs()?
				return t.Obj().Name()
			}
			return ""
		}
		funcType := ""
		if recv := f.Signature().Recv(); recv != nil {
			funcType = recvType(recv.Type())
		}
		funcName, funcSig := f.Name(), f.Signature().Params()
		if funcName == "" {
			funcName = "func"
		}

		params := f.Signature().Params()
		ppass := func() *types.Var {
			if ai >= params.Len() {
				return params.At(params.Len() - 1)
			} else {
				return params.At(ai)
			}
		}()

		pass.Reportf(
			n.Pos(),
			"passes '%s' as '%s' in call to %s%s%s (position %d vs %d)",
			arg.Name(), ppass.Name(),
			funcType, funcName, funcSig,
			ai, pi,
		)
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{new(ast.CallExpr)}, func(n ast.Node) {
		if !v.IncludeGeneratedFiles && isCallGenerated(n) {
			return
		}
		call := n.(*ast.CallExpr)
		fun := funOf(call)
		if fun == nil {
			return
		}
		for ai, x := range call.Args {
			argVar := varOf(x)
			if argVar == nil {
				continue
			}
			a := (*arg)(argVar)
			matchers := func() []paramMatcher {
				if v.ExactTypeOnly {
					return []paramMatcher{a.CaseTypeMatch, a.NoCaseTypeMatch}
				}
				return []paramMatcher{a.CaseMatch, a.NoCaseMatch}
			}
			if pi, _ := findParam(fun, ai, matchers()...); pi >= 0 {
				if pi != ai && pi < len(call.Args) && varOf(call.Args[pi]).Name() != a.Name() {
					report(x, a, ai, fun, pi)
				}
			}
		}
	})
	return nil, nil
}

func selobj(ti *types.Info, x *ast.SelectorExpr) types.Object {
	if obj := ti.ObjectOf(x.Sel); obj != nil {
		return obj
	}
	sel := ti.Selections[x]
	return sel.Obj()
}
