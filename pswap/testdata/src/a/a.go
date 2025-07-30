package a

import "a/pkg"

func abc(a, b, c string) {}
func ABC(A, B, C string) {}
func ghi(g, h, i string) {}
func AAaa(AA, aa string) {}
func anys(a, b, c any)   {}

type D struct {
	a, b, c string
	A, B, C string
}

func f() D { return D{} }

// generics
func TTT[T any](a, b, c T)           {}
func TUV[T, U, V any](a T, b U, c V) {}

type G[T any] struct{}

func (G[T]) abc(a, b, c T)   {}
func (*G[T]) pabc(a, b, c T) {}

func tests() {
	a, b, c, d := "a", "b", "c", D{}

	abc(a, b, c) // good
	abc(a, a, c) // dup name is visible
	abc(b, a, c) // want `passes 'b' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'a' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)`

	ghi(a, b, c) // good
	ghi(a, a, c) // good
	ghi(b, a, c) // good

	aA, Aa, AA, aa := "aA", "Aa", "AA", "aa"
	AAaa(aA, Aa) // good -- neither AA or aa is perfect, so accept matching index
	AAaa(AA, aa) // good
	AAaa(aa, AA) // want `passes 'aa' as 'AA' in call to AAaa\(AA string, aa string\) \(position 0 vs 1\)` `passes 'AA' as 'aa' in call to AAaa\(AA string, aa string\) \(position 1 vs 0\)`

	anys(c, b, a) // want `passes 'c' as 'a' in call to anys\(a any, b any, c any\) \(position 0 vs 2\)` `passes 'a' as 'c' in call to anys\(a any, b any, c any\) \(position 2 vs 0\)`
	// "anys argument c in position 0 matches parameter in position 2" "anys argument a in position 2 matches parameter in position 0"

	// TODO: should the message mention pkg.ABC instead of ABC?
	pkg.ABC(a, b, c) // good
	pkg.ABC(a, a, c) // dup name is visible
	pkg.ABC(b, a, c) // want `passes 'b' as 'a' in call to ABC\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'a' as 'b' in call to ABC\(a string, b string, c string\) \(position 1 vs 0\)`

	ABC(a, b, c) // good
	ABC(a, a, c) // dup name is visible
	ABC(b, a, c) // want `passes 'b' as 'A' in call to ABC\(A string, B string, C string\) \(position 0 vs 1\)` `passes 'a' as 'B' in call to ABC\(A string, B string, C string\) \(position 1 vs 0\)`

	// TODO: should the message mention d.a instead of a?
	abc(d.a, d.b, d.c) // good
	abc(d.a, d.a, d.c) // dup name is visible
	abc(d.b, d.a, d.c) // want `passes 'b' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'a' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)`

	ABC(d.a, d.b, d.c) // good
	ABC(d.a, d.a, d.c) // dup name is visible
	ABC(d.b, d.a, d.c) // want `passes 'b' as 'A' in call to ABC\(A string, B string, C string\) \(position 0 vs 1\)` `passes 'a' as 'B' in call to ABC\(A string, B string, C string\) \(position 1 vs 0\)`

	abc(d.A, d.B, d.C) // good
	abc(d.A, d.A, d.C) // dup name is visible
	abc(d.B, d.A, d.C) // want `passes 'B' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'A' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)`

	ABC(d.A, d.B, d.C) // good
	ABC(d.A, d.A, d.C) // dup name is visible
	ABC(d.B, d.A, d.C) // want `passes 'B' as 'A' in call to ABC\(A string, B string, C string\) \(position 0 vs 1\)` `passes 'A' as 'B' in call to ABC\(A string, B string, C string\) \(position 1 vs 0\)`

	// TODO: should the message mention f().a instead of a?
	abc(f().a, f().b, f().c) // good
	abc(f().a, f().a, f().c) // dup name is visible
	abc(f().A, f().a, f().c) // want `passes 'a' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)`
	abc(f().b, f().a, f().c) // want `passes 'a' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)`

	ABC(f().a, f().b, f().c) // good
	ABC(f().a, f().a, f().c) // dup name is visible
	ABC(f().b, f().a, f().c) // want `passes 'b' as 'A' in call to ABC\(A string, B string, C string\) \(position 0 vs 1\)` `passes 'a' as 'B' in call to ABC\(A string, B string, C string\) \(position 1 vs 0\)`

	abc(f().A, f().B, f().C) // good
	abc(f().A, f().A, f().C) // dup name is visible
	abc(f().B, f().A, f().C) // want `passes 'B' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'A' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)`

	ABC(f().A, f().B, f().C) // good
	ABC(f().A, f().A, f().C) // dup name is visible
	ABC(f().a, f().A, f().C) // want `passes 'A' as 'B' in call to ABC\(A string, B string, C string\) \(position 1 vs 0\)`
	ABC(f().B, f().A, f().C) // want `passes 'A' as 'B' in call to ABC\(A string, B string, C string\) \(position 1 vs 0\)` `passes 'B' as 'A' in call to ABC\(A string, B string, C string\) \(position 0 vs 1\)`

	TTT(a, b, c) // good
	TTT(a, a, c) // dup name is visible
	TTT(b, a, c) // want `passes 'a' as 'b' in call to TTT\[string\]\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to TTT\[string\]\(a string, b string, c string\) \(position 0 vs 1\)`

	TUV(a, b, c) // good
	TUV(a, a, c) // dup name is visible
	TUV(b, a, c) // want `passes 'a' as 'b' in call to TUV\[string, string, string\]\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to TUV\[string, string, string\]\(a string, b string, c string\) \(position 0 vs 1\)`

	g := G[string]{}
	g.abc(a, b, c) // good
	g.abc(a, a, c) // dup name is visible
	g.abc(b, a, c) // want `passes 'a' as 'b' in call to \(G\[string\]\).abc\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to \(G\[string\]\).abc\(a string, b string, c string\) \(position 0 vs 1\)`

	g.pabc(a, b, c) // good
	g.pabc(a, a, c) // dup name is visible
	g.pabc(b, a, c) // want `passes 'a' as 'b' in call to \(\*G\[string\]\).pabc\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to \(\*G\[string\]\).pabc\(a string, b string, c string\) \(position 0 vs 1\)`

	func(a, b, c string) {}(a, b, c) // good
	func(a, b, c string) {}(a, a, c) // dup name is visible
	func(a, b, c string) {}(b, a, c) // want `passes 'a' as 'b' in call to func\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to func\(a string, b string, c string\) \(position 0 vs 1\)`

	f := func(a, b, c string) {}
	f(a, b, c) // good
	f(a, a, c) // dup name is visible
	f(b, a, c) // want `passes 'a' as 'b' in call to f\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to f\(a string, b string, c string\) \(position 0 vs 1\)`

	f = g.pabc
	f(a, b, c) // good
	f(a, a, c) // dup name is visible
	f(b, a, c) // want `passes 'a' as 'b' in call to f\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to f\(a string, b string, c string\) \(position 0 vs 1\)`

	var i interface {
		blank(string, string, string)
		abc(a, b, c string)
	}
	i.blank(a, b, c) // fine
	i.blank(c, b, a) // fine
	i.abc(a, b, c)   // good
	i.abc(b, a, c)   // want `passes 'a' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)` `passes 'b' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)`

	func(c string, _ ...string) {}(a, b, c) // want `passes 'c' as '_' in call to func\(c string, _ \[\]string\) \(position 2 vs 0\)`
}

type mock struct {
	x *expectations
}

func (m *mock) ExpectFoo(t TB, fn func(int, string) string) {
	m.x.Expect(t, "Foo", fn) // skip mismatched type (string != func)
}

type expectations struct{}

func (x *expectations) Expect(t TB, fn string, e ...any) {}

type TB interface{ Helper() }

var _ = tests
