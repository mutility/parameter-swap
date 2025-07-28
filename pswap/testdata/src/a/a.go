package a

import "a/pkg"

func abc(a, b, c string) {} // want abc:`&\[\{a string} \{b string} \{c string}]`
func ABC(A, B, C string) {} // want ABC:`&\[\{A string} \{B string} \{C string}]`
func ghi(g, h, i string) {} // want ghi:`&\[\{g string} \{h string} \{i string}]`
func AAaa(AA, aa string) {} // want AAaa:`&\[\{AA string} \{aa string}]`

type D struct {
	a, b, c string
	A, B, C string
}

func f() D { return D{} }

func tests() {
	a, b, c, d := "a", "b", "c", D{}

	abc(a, b, c) // good
	abc(a, a, c) // dup name is visible
	abc(b, a, c) // want "abc argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	ghi(a, b, c) // good
	ghi(a, a, c) // good
	ghi(b, a, c) // good

	aA, Aa, AA, aa := "aA", "Aa", "AA", "aa"
	AAaa(aA, Aa) // good -- neither AA or aa is perfect, so accept matching index
	AAaa(AA, aa) // good
	AAaa(aa, AA) // want "AAaa argument aa in position 0 matches parameter in position 1" "AAaa argument AA in position 1 matches parameter in position 0"

	pkg.ABC(a, b, c) // good
	pkg.ABC(a, a, c) // dup name is visible
	pkg.ABC(b, a, c) // want "ABC argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	ABC(a, b, c) // good
	ABC(a, a, c) // dup name is visible
	ABC(b, a, c) // want "ABC argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	abc(d.a, d.b, d.c) // good
	abc(d.a, d.a, d.c) // dup name is visible
	abc(d.b, d.a, d.c) // want "abc argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	ABC(d.a, d.b, d.c) // good
	ABC(d.a, d.a, d.c) // dup name is visible
	ABC(d.b, d.a, d.c) // want "ABC argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	abc(d.A, d.B, d.C) // good
	abc(d.A, d.A, d.C) // dup name is visible
	abc(d.B, d.A, d.C) // want "abc argument A in position 1 matches parameter in position 0" "argument B in position 0 matches parameter in position 1"

	ABC(d.A, d.B, d.C) // good
	ABC(d.A, d.A, d.C) // dup name is visible
	ABC(d.B, d.A, d.C) // want "ABC argument A in position 1 matches parameter in position 0" "argument B in position 0 matches parameter in position 1"

	abc(f().a, f().b, f().c) // good
	abc(f().a, f().a, f().c) // dup name is visible
	abc(f().A, f().a, f().c) // want "abc argument a in position 1 matches parameter in position 0"
	abc(f().b, f().a, f().c) // want "abc argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	ABC(f().a, f().b, f().c) // good
	ABC(f().a, f().a, f().c) // dup name is visible
	ABC(f().b, f().a, f().c) // want "ABC argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	abc(f().A, f().B, f().C) // good
	abc(f().A, f().A, f().C) // dup name is visible
	abc(f().B, f().A, f().C) // want "abc argument A in position 1 matches parameter in position 0" "argument B in position 0 matches parameter in position 1"

	ABC(f().A, f().B, f().C) // good
	ABC(f().A, f().A, f().C) // dup name is visible
	ABC(f().a, f().A, f().C) // want "ABC argument A in position 1 matches parameter in position 0"
	ABC(f().B, f().A, f().C) // want "ABC argument A in position 1 matches parameter in position 0" "argument B in position 0 matches parameter in position 1"
}

type mock struct {
	x *expectations
}

func (m *mock) ExpectFoo(t TB, fn func(int, string) string) { // want ExpectFoo:`&\[\{t a.TB\} \{fn func\(int, string\) string}]`
	m.x.Expect(t, "Foo", fn) // skip mismatched type (string != func)
}

type expectations struct{}

func (x *expectations) Expect(t TB, fn string, e ...any) {} // want Expect:`&\[\{t a.TB\} \{fn string\} \{e \[\]any}]`

type TB interface{ Helper() }

var _ = tests
