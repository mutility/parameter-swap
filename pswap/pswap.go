// Package pswap implements the parameter-swap analyzer. It reports when a
// a named parameter is passed to a function that offers a different parameter
// with the same name.

package pswap

import (
	"cmp"
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

func findParam(sig *types.Signature, argIndex int, matchers ...paramMatcher) (int, *param) {
	params := sig.Params()
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

func (a *arg) NoCaseMatch(p *param) bool {
	return strings.EqualFold(a.Name(), p.Name()) && types.AssignableTo(a.Type(), p.Type())
}

func (a *arg) CaseTypeMatch(p *param) bool {
	return a.Name() == p.Name() && a.Type() == p.Type()
}

func (a *arg) NoCaseTypeMatch(p *param) bool {
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
		// return typeutil.StaticCallee(pass.TypesInfo, c)
		switch f := c.Fun.(type) {
		case *ast.Ident:
			switch o := pass.TypesInfo.ObjectOf(f).(type) {
			case *types.Func:
				return o
			case *types.Var:
				// attempt to synthesize a Func matching the local name
				if sig, ok := o.Type().(*types.Signature); ok {
					name := cmp.Or(o.Origin().Name(), "func")
					return types.NewFunc(o.Pos(), nil, name, sig)
				}
			}
		case *ast.SelectorExpr:
			return selobj(pass.TypesInfo, f).(*types.Func)
		case *ast.FuncLit:
			// Can't find a *types.Func for a funclit; synthesize the parts we need.
			// This is presumably brittle, but works well enough.
			variadic := false
			paramsOf := func(fl *ast.FieldList) (vs []*types.Var) {
				if fl == nil {
					return nil
				}
				for _, p := range fl.List {
					ptype := p.Type
					if ell, ok := p.Type.(*ast.Ellipsis); ok {
						ptype = ell.Elt
						variadic = true
					}
					for _, n := range p.Names {
						vs = append(vs, types.NewVar(p.Pos(), nil, n.Name, pass.TypesInfo.TypeOf(ptype)))
					}
				}
				return vs
			}
			paramVars := paramsOf(f.Type.Params)
			resultVars := paramsOf(f.Type.Results)
			params := types.NewTuple(paramVars...)
			results := types.NewTuple(resultVars...)
			sig := types.NewSignatureType(nil, nil, nil, params, results, variadic)
			fun := types.NewFunc(c.Fun.Pos(), nil, "func", sig)
			return fun
			// default:
			// 	fmt.Printf("unhandled funOf %T %#[1]v\n", f)
		}
		return nil
	}

	sigOf := func(c *ast.CallExpr) *types.Signature {
		switch callee := pass.TypesInfo.TypeOf(c.Fun).(type) {
		case *types.Signature:
			if callee.TypeParams() == nil {
				return callee
			}
			var inst types.Instance
			switch fun := c.Fun.(type) {
			case *ast.Ident:
				inst = pass.TypesInfo.Instances[fun]
			case *ast.SelectorExpr:
				inst = pass.TypesInfo.Instances[fun.Sel]
			}

			if sig, ok := inst.Type.(*types.Signature); ok {
				return sig
			}
			return callee
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
		// fmt.Printf("unhandled %T %[1]v\n", x)
		return nil
	}

	report := func(n ast.Node, arg *arg, ai int, fun *types.Func, sig *types.Signature, pi int) {
		if fun == nil {
			return
		}
		// similar to t.String, but omits package names
		recvType := func(t types.Type) string {
			prefix := "("
			if p, ok := t.(*types.Pointer); ok {
				t = p.Elem()
				prefix = "(*"
			}
			if n, ok := t.(*types.Named); ok {
				infix := n.Obj().Name()
				if ta := n.TypeArgs(); ta != nil {
					tas := make([]string, ta.Len())
					for i := range tas {
						tas[i] = ta.At(i).String()
					}
					infix += "[" + strings.Join(tas, ", ") + "]"
				}
				return prefix + infix + ")."
			}
			return ""
		}
		funcType := ""
		if recv := fun.Signature().Recv(); recv != nil {
			funcType = recvType(recv.Type())
		}
		funcName, funcSig := fun.Name(), sig.Params()
		if funcName == "" {
			funcName = "func"
		}

		funcTP := ""
		if fun.Signature() != sig && fun.Signature().TypeParams() != nil {
			tparams := make([]string, fun.Signature().TypeParams().Len())
			for i := range fun.Signature().Params().Len() {
				pt := fun.Signature().Params().At(i).Type()
				for ti := range len(tparams) {
					if fun.Signature().TypeParams().At(ti) == pt {
						tparams[ti] = sig.Params().At(i).Type().String()
					}
				}
			}
			funcTP = "[" + strings.Join(tparams, ", ") + "]"
		}

		params := sig.Params()
		ppass := func() *types.Var {
			if ai >= params.Len() {
				return params.At(params.Len() - 1)
			} else {
				return params.At(ai)
			}
		}()

		pass.Reportf(
			n.Pos(),
			"passes '%s' as '%s' in call to %s%s%s%s (position %d vs %d)",
			arg.Name(), ppass.Name(),
			funcType, funcName, funcTP, funcSig,
			ai, pi,
		)
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{new(ast.CallExpr)}, func(n ast.Node) {
		if !v.IncludeGeneratedFiles && isCallGenerated(n) {
			return
		}
		call := n.(*ast.CallExpr)
		sig := sigOf(call)
		if sig == nil {
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
			if pi, _ := findParam(sig, ai, matchers()...); pi >= 0 {
				if pi != ai && pi < len(call.Args) {
					if v := varOf(call.Args[pi]); v == nil || v.Name() != a.Name() {
						report(x, a, ai, funOf(call), sig, pi)
					}
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
