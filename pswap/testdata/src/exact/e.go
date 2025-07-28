package exact

func abc(a, b, c string) {} // want abc:`&\[\{a string} \{b string} \{c string}]`
func anys(a, b, c any)   {} // want anys:`&\[\{a any} \{b any} \{c any}]`

func tests() {
	a, b, c := "a", "b", "c"

	abc(a, b, c) // good
	abc(a, a, c) // dup name is visible
	abc(b, a, c) // want "abc argument a in position 1 matches parameter in position 0" "argument b in position 0 matches parameter in position 1"

	anys(c, b, a) // ignored string->any
}

var _ = tests
