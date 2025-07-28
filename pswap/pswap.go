// Package pswap implements the parameter-swap analyzer. It reports when a
// a named parameter is passed to a function that offers a different parameter
// with the same name.

package pswap

import (
	"fmt"
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
			Name:      "varfmt",
			Doc:       doc,
			Requires:  []*analysis.Analyzer{inspect.Analyzer},
			FactTypes: []analysis.Fact{new(paramList)},
		},
	}
	a.Flags.BoolVar(&a.ExactTypeOnly, "exact", false, "suppress pswap reports when types aren't an exact match")
	a.Flags.BoolVar(&a.IncludeGeneratedFiles, "gen", false, "include reports from generated files")

	a.Run = a.run

	return a
}

type (
	paramList []param
	param     struct {
		Name string
		Type types.Type
	}
	arg          param
	paramMatcher func(param) bool
)

func (*paramList) AFact() {}

func (pl *paramList) Index(ai int, matchers ...paramMatcher) (int, param) {
	for _, match := range matchers {
		// prefer matching index when available over, e.g. similarly case mismatch in earlier param
		if ai < len(*pl) && match((*pl)[ai]) {
			return ai, (*pl)[ai]
		}
		for i, p := range *pl {
			if match(p) {
				return i, p
			}
		}
	}
	return -1, param{}
}

func (a arg) CaseMatch(p param) bool {
	return a.Name == p.Name && types.AssignableTo(a.Type, p.Type)
}

func (a arg) NoCaseMatch(p param) bool {
	return strings.EqualFold(a.Name, p.Name) && types.AssignableTo(a.Type, p.Type)
}

func (a arg) CaseTypeMatch(p param) bool {
	return a.Name == p.Name && a.Type == p.Type
}

func (a arg) NoCaseTypeMatch(p param) bool {
	return strings.EqualFold(a.Name, p.Name) && a.Type == p.Type
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

	argName := func(x ast.Expr) string {
		switch x := x.(type) {
		case *ast.Ident:
			return x.Name
		case *ast.SelectorExpr:
			return x.Sel.Name
		}
		return ""
	}

	report := func(n ast.Node, name string, ai int, f types.Object, pi int) {
		d := analysis.Diagnostic{Pos: n.Pos()}
		if funName := f.Name(); name != "" {
			d.Message = fmt.Sprintf("%s argument %s in position %d matches parameter in position %d", funName, name, ai, pi)
		} else {
			d.Message = fmt.Sprintf("argument %s in position %d matches parameter in position %d", name, ai, pi)
		}
		if f != nil && f.Pos() >= 0 {
			d.Related = []analysis.RelatedInformation{{
				Pos:     f.Pos(),
				Message: fmt.Sprintf("signature: %s", f),
			}}
		}
		pass.Report(d)
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
		if !v.IncludeGeneratedFiles && isCallGenerated(n) {
			return
		}
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
			if name := argName(x); name != "" {
				a := arg{Name: name, Type: pass.TypesInfo.TypeOf(x)}
				matchers := func() []paramMatcher {
					if v.ExactTypeOnly {
						return []paramMatcher{a.CaseTypeMatch, a.NoCaseTypeMatch}
					}
					return []paramMatcher{a.CaseMatch, a.NoCaseMatch}
				}
				if pi, _ := funParams.Index(ai, matchers()...); pi >= 0 {
					if pi != ai && pi < len(c.Args) && argName(c.Args[pi]) != name {
						report(x, name, ai, funObj, pi)
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
