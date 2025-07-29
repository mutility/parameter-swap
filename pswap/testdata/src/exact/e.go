package exact

func abc(a, b, c string) {} // want abc:`&\[\{a string} \{b string} \{c string}]`
func anys(a, b, c any)   {} // want anys:`&\[\{a any} \{b any} \{c any}]`

func tests() {
	a, b, c := "a", "b", "c"

	abc(a, b, c) // good
	abc(a, a, c) // dup name is visible
	abc(b, a, c) // want `passes 'b' as 'a' in call to abc\(a string, b string, c string\) \(position 0 vs 1\)` `passes 'a' as 'b' in call to abc\(a string, b string, c string\) \(position 1 vs 0\)`

	anys(c, b, a) // ignored string->any
}

var _ = tests
