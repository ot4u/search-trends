package trending

import "testing"

func TestBuildTopN(t *testing.T) {
	t.Parallel()

	counts := map[string]uint64{
		"iphone":  40,
		"airpods": 10,
		"macbook": 40,
		"watch":   25,
	}

	items := BuildTopN(counts, 3)
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}

	want := []Item{
		{Query: "iphone", Score: 40, Rank: 1},
		{Query: "macbook", Score: 40, Rank: 2},
		{Query: "watch", Score: 25, Rank: 3},
	}

	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("items[%d] = %#v, want %#v", i, items[i], want[i])
		}
	}
}
