// Package pswap implements the parameter-swap analyzer. It reports when a
// a named parameter is passed to a function that offers a different parameter
// with the same name.

package pswap

import (
	"cmp"
	"go/ast"
	"go/types"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
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
			Name:     "pswap",
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
	param        types.Var
	paramMatcher func(*param) bool
	nameType     struct {
		Pkg  *types.Package
		Name string
		Type types.Type
	}
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

func (p *param) Name() string     { return (*types.Var)(p).Name() }
func (p *param) Type() types.Type { return (*types.Var)(p).Type() }

func (a *nameType) CaseMatch(p *param) bool {
	return a.Name == p.Name() && types.AssignableTo(a.Type, p.Type())
}

func (a *nameType) NoCaseMatch(p *param) bool {
	return strings.EqualFold(a.Name, p.Name()) && types.AssignableTo(a.Type, p.Type())
}

func (a *nameType) CaseTypeMatch(p *param) bool {
	return a.Name == p.Name() && a.Type == p.Type()
}

func (a *nameType) NoCaseTypeMatch(p *param) bool {
	return strings.EqualFold(a.Name, p.Name()) && a.Type == p.Type()
}

func (v *pswapAnalyzer) run(pass *analysis.Pass) (any, error) {
	var lastCall struct {
		File      *ast.File
		Generated bool
	}
	var qualify types.Qualifier
	allPkgNames := make(map[string]string, len(pass.Pkg.Imports()))
	for _, imp := range pass.Pkg.Imports() {
		allPkgNames[imp.Path()] = imp.Name()
	}
	relativeTo := func(f *ast.File) types.Qualifier {
		localPkgNames := make(map[string]string, len(f.Imports))
		for _, imp := range f.Imports {
			path, _ := strconv.Unquote(imp.Path.Value)
			localPkgNames[path] = allPkgNames[path]
			if imp.Name != nil {
				localPkgNames[path] = imp.Name.Name
			}
		}
		return func(other *types.Package) string {
			if other == pass.Pkg {
				return ""
			}
			return localPkgNames[other.Path()]
		}
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
				qualify = relativeTo(f)
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
			return typeutil.Callee(pass.TypesInfo, c).(*types.Func)
		case *ast.FuncLit:
			// Can't find a *types.Func for a funclit; synthesize the parts we need.
			// This is presumably brittle, but works well enough.
			variadic := false
			paramsOf := func(fl *ast.FieldList) (vs []*types.Var) {
				if fl == nil {
					return nil
				}
				for _, p := range fl.List {
					if _, ok := p.Type.(*ast.Ellipsis); ok {
						variadic = true
					}
					for _, n := range p.Names {
						vs = append(vs, types.NewVar(p.Pos(), nil, n.Name, pass.TypesInfo.TypeOf(p.Type)))
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

	nameTypeOf := func(x ast.Expr) nameType {
		var o types.Object
		switch x := x.(type) {
		case *ast.Ident:
			o = pass.TypesInfo.ObjectOf(x)
		case *ast.SelectorExpr:
			o = pass.TypesInfo.ObjectOf(x.Sel)
		case *ast.BasicLit:
			return nameType{nil, "", pass.TypesInfo.TypeOf(x)}
			// default:
			// 	fmt.Printf("%s unhandled %T %[1]v\n", pass.Fset.Position(x.Pos()), x)
		}
		if o != nil {
			return nameType{o.Pkg(), o.Name(), o.Type()}
		}
		return nameType{}
	}

	report := func(n ast.Node, arg nameType, ai int, call *ast.CallExpr, pi int) {
		fun := funOf(call)
		sig, _ := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
		if fun == nil || sig == nil {
			return
		}

		signature := func() string {
			if fun.Name() == "func" {
				return types.TypeString(sig, qualify)
			} else if fun.Signature() != sig && fun.Signature().TypeParams() != nil {
				// Partly resolve type parameters in our message.
				// For example take func Foo[T any](foo T)
				// When called with a string, its signature is func(foo string)
				// Make a new func with new signature that merges these, yielding:
				// func Foo[T = string](foo T)
				resolveTypeParams := func(tup *types.TypeParamList) []*types.TypeParam {
					l := make([]*types.TypeParam, tup.Len())
					for i := range l {
						tp := tup.At(i)
						for i := range fun.Signature().Params().Len() {
							if tv := fun.Signature().Params().At(i); tv.Type() == tp && i < sig.Params().Len() {
								cv := sig.Params().At(i)
								witheq := types.NewTypeName(tp.Obj().Pos(), tp.Obj().Pkg(), tp.Obj().Name()+" =", tp.Obj().Type())
								tp = types.NewTypeParam(witheq, cv.Type())
								break
							}
						}
						l[i] = tp
					}
					return l
				}

				rsig := types.NewSignatureType(
					fun.Signature().Recv(),
					nil,
					resolveTypeParams(fun.Signature().TypeParams()),
					fun.Signature().Params(),
					fun.Signature().Results(),
					sig.Variadic(),
				)
				return types.ObjectString(types.NewFunc(fun.Pos(), fun.Pkg(), fun.Name(), rsig), qualify)
			} else {
				return types.ObjectString(fun, qualify)
			}
		}()

		argName := qualify(arg.Pkg)
		if argName != "" {
			argName += "."
		}
		argName += arg.Name
		pass.Reportf(
			n.Pos(),
			"passes '%s' as '%s' in call to %s (position %d vs %d)",
			argName, sig.Params().At(min(sig.Params().Len()-1, ai)).Name(),
			signature, ai, pi,
		)
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{new(ast.CallExpr)}, func(n ast.Node) {
		if !v.IncludeGeneratedFiles && isCallGenerated(n) {
			return
		}
		call := n.(*ast.CallExpr)
		sig, _ := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
		if sig == nil {
			return
		}
		for ai, x := range call.Args {
			arg := nameTypeOf(x)
			if arg.Name == "" {
				continue
			}
			matchers := func() []paramMatcher {
				if v.ExactTypeOnly {
					return []paramMatcher{arg.CaseTypeMatch, arg.NoCaseTypeMatch}
				}
				return []paramMatcher{arg.CaseMatch, arg.NoCaseMatch}
			}
			if pi, _ := findParam(sig, ai, matchers()...); pi >= 0 {
				if pi != ai && pi < len(call.Args) {
					if v := nameTypeOf(call.Args[pi]); v.Name != arg.Name {
						report(x, arg, ai, call, pi)
					}
				}
			}
		}
	})
	return nil, nil
}
