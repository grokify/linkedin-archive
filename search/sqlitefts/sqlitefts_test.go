package sqlitefts

import (
	"testing"

	"github.com/grokify/linkedin-archive/search"
)

func TestToFTS5(t *testing.T) {
	cases := map[string]string{
		`a b`:         `("a" AND "b")`,
		`a b OR c`:    `("a" AND "b" OR "c")`,
		`a (b OR c*)`: `("a" AND ("b" OR "c"*))`,
		`"x y" -z`:    `(("x y") NOT "z")`,
		`say"hi`:      `("say" AND "hi")`,
		`a NEAR b`:    `("a" AND "NEAR" AND "b")`,
		`a -(b OR c)`: `(("a") NOT ("b" OR "c"))`,
	}
	for in, want := range cases {
		terms, err := search.Parse(in)
		if err != nil {
			t.Fatalf("Parse(%q): %v", in, err)
		}
		if got := ToFTS5(terms); got != want {
			t.Errorf("ToFTS5(%q) = %s, want %s", in, got, want)
		}
	}
}
