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
	ExactTypeOnly bool
}

func Analyzer() *pswapAnalyzer {
	a := &pswapAnalyzer{
		Analyzer: &analysis.Analyzer{
			Name:      "varfmt",
			Doc:       doc,
			Requires:  []*analysis.Analyzer{inspect.Analyzer},
			FactTypes: []analysis.Fact{new(paramList)},
		},
	}
	a.Flags.BoolVar(&a.ExactTypeOnly, "exact", false, "suppress pswap reports when types aren't an exact match")

	a.Run = a.run

	return a
}

type (
	paramList []param
	param     struct {
		Name string
		Type types.Type
	}
)

func (*paramList) AFact() {}

func (v *pswapAnalyzer) run(pass *analysis.Pass) (any, error) {
	// track local function's parameters
	locals := make(map[types.Object]paramList)
	paramsOf := func(fun *ast.FuncType) (l paramList) {
		if fun.Params == nil || len(fun.Params.List) == 0 {
			return nil
		}

		for _, p := range fun.Params.List {
			t := pass.TypesInfo.TypeOf(p.Type)
			for _, n := range p.Names {
				l = append(l, param{n.Name, t})
			}
		}
		return l
	}

	callFunObj := func(c *ast.CallExpr) types.Object {
		switch f := c.Fun.(type) {
		case *ast.Ident:
			return pass.TypesInfo.ObjectOf(f)
		case *ast.SelectorExpr:
			return selobj(pass.TypesInfo, f)
			// case *ast.CallExpr, *ast.TypeAssertExpr, *ast.ParenExpr, *ast.IndexExpr:
			// case *ast.FuncLit, *ast.ArrayType, *ast.InterfaceType, *ast.MapType, *ast.ChanType, *ast.StructType:
			// default:
			// 	pt("unhandled callfun "+pp(n.Fun), n.Fun)
		}
		return nil
	}

	argName := func(x ast.Expr) (string, bool) {
		switch x := x.(type) {
		case *ast.Ident:
			return x.Name, true
		case *ast.SelectorExpr:
			return x.Sel.Name, true
		}
		return "", false
	}

	paramIndex := func(name string, pl paramList, match func(string, string) bool) (int, bool) {
		for i, p := range pl {
			if match(name, p.Name) {
				return i, true
			}
		}
		return -1, false
	}

	report := func(n ast.Node, name string, ai, pi int) {
		pass.Reportf(n.Pos(), "argument %s in position %d matches parameter in position %d", name, ai, pi)
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspect.Preorder([]ast.Node{new(ast.FuncDecl)}, func(n ast.Node) {
		f := n.(*ast.FuncDecl)
		obj := pass.TypesInfo.ObjectOf(f.Name)
		if obj != nil {
			if ps := paramsOf(f.Type); len(ps) > 0 {
				pass.ExportObjectFact(obj, &ps)
				locals[obj] = ps
			}
		}
	})
	inspect.Preorder([]ast.Node{new(ast.CallExpr)}, func(n ast.Node) {
		c := n.(*ast.CallExpr)
		funObj := callFunObj(c)
		if funObj == nil {
			return
		}
		funParams, ok := locals[funObj]
		if !ok {
			pass.ImportObjectFact(funObj, &funParams)
		}
		for ai, x := range c.Args {
			if name, ok := argName(x); ok {
				if pi, ok := paramIndex(name, funParams, func(a, b string) bool { return a == b }); ok {
					if ai != pi {
						report(n, name, ai, pi)
					}
				} else if pi, ok := paramIndex(name, funParams, strings.EqualFold); ok {
					if ai != pi {
						report(n, name, ai, pi)
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
