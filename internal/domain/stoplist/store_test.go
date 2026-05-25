package stoplist

import "testing"

func TestStore(t *testing.T) {
	t.Parallel()

	store := NewStore([]string{"spam"})
	if !store.Has("spam") {
		t.Fatal("expected initial entry to exist")
	}
	if !store.ContainsAny([]string{"ham", "spam"}) {
		t.Fatal("expected token lookup to match")
	}
	if store.Add("spam") {
		t.Fatal("expected duplicate add to be ignored")
	}
	if !store.Add("scam") {
		t.Fatal("expected new add to succeed")
	}
	if !store.Remove("spam") {
		t.Fatal("expected remove to succeed")
	}
	if store.Remove("missing") {
		t.Fatal("expected missing remove to be ignored")
	}

	got := store.List()
	if len(got) != 1 || got[0] != "scam" {
		t.Fatalf("unexpected list: %#v", got)
	}
}
