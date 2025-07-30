package a

import (
	ren "a/pkg"
)

// confirm that renamed imports report with renamed value

func rentests() {
	a, b, c := "a", "b", "c"

	ren.ABC(a, b, c) // good
	ren.ABC(a, a, c) // dup name is visible
	ren.ABC(b, a, c) // want `passes 'b' as 'a' in call to func ren.ABC\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'a' as 'b' in call to func ren.ABC\(a string, b string, c string\) \(position 1 vs 0\)`

	anys(ren.ABC, con, nil) // provokes *ast.SelectorExpr -> *types.Func
	// *ast.Ident -> *types.Const, and *ast.Ident -> *types.Nil in varOf.

	func(pro, con string) {}(con, "")   // want `passes 'con' as 'pro' in call to func\(pro string, con string\) \(position 0 vs 1\)`
	func(abc, xyz any) {}(nil, ren.ABC) // want `passes 'ABC' as 'xyz' in call to func\(abc any, xyz any\) \(position 1 vs 0\)`
}

var _ = rentests
