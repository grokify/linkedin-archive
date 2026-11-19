package search

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	terms, err := Parse(`acme (shares OR "stock options" OR invest*) -recruiter`)
	if err != nil {
		t.Fatal(err)
	}
	if len(terms) != 3 {
		t.Fatalf("terms = %+v", terms)
	}
	if terms[0].Text != "acme" {
		t.Errorf("term 0 = %+v", terms[0])
	}
	g := terms[1].Group
	if len(g) != 3 || g[0].Text != "shares" || !g[1].Phrase || !g[1].Or || !g[2].Prefix || g[2].Text != "invest" {
		t.Errorf("group = %+v", g)
	}
	if !terms[2].Negate || terms[2].Text != "recruiter" {
		t.Errorf("term 2 = %+v", terms[2])
	}
}

func TestParseEdgeCases(t *testing.T) {
	cases := map[string]error{
		"":                      nil,
		"   ":                   nil,
		"-only":                 ErrOnlyNegated,
		"a (-b)":                ErrOnlyNegated,
		"(a":                    ErrUnbalanced,
		"a)":                    ErrUnbalanced,
		"OR a":                  nil,
		`"unterminated`:         nil,
		"((((((((((a))))))))))": ErrNestingTooDeep,
	}
	for in, want := range cases {
		_, err := Parse(in)
		if !errors.Is(err, want) {
			t.Errorf("Parse(%q) error = %v, want %v", in, err, want)
		}
	}
	terms, err := Parse("OR a")
	if err != nil {
		t.Fatal(err)
	}
	if len(terms) != 1 || terms[0].Or {
		t.Errorf("leading OR should be ignored: %+v", terms)
	}
}

func TestEffectiveLimit(t *testing.T) {
	if (Query{}).EffectiveLimit() != DefaultLimit || (Query{Limit: 5}).EffectiveLimit() != 5 {
		t.Error("EffectiveLimit")
	}
}
