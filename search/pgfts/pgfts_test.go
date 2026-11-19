package pgfts

import (
	"strconv"
	"testing"

	"github.com/grokify/linkedin-archive/search"
)

func TestToTSQuery(t *testing.T) {
	terms, err := search.Parse(`a b OR "c d" -e* (f OR g)`)
	if err != nil {
		t.Fatal(err)
	}
	var args []any
	arg := func(v any) string {
		args = append(args, v)
		return "$" + strconv.Itoa(len(args))
	}
	got := ToTSQuery(terms, arg)
	want := "(((plainto_tsquery('english', $1) && plainto_tsquery('english', $2)) || " +
		"(phraseto_tsquery('english', $3) && ((plainto_tsquery('english', $5)) || (plainto_tsquery('english', $6))))) " +
		"&& !!to_tsquery('english', $4))"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
	wantArgs := []any{"a", "b", "c d", "e:*", "f", "g"}
	for i, a := range wantArgs {
		if args[i] != a {
			t.Errorf("arg %d = %v, want %v", i, args[i], a)
		}
	}
}
