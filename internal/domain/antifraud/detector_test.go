package antifraud

import "testing"

func TestDetector(t *testing.T) {
	t.Parallel()

	detector := NewDetector(2, 10, 1)

	if got := detector.Weight("iphone"); got != 10 {
		t.Fatalf("first weight = %d, want 10", got)
	}
	if got := detector.Weight("iphone"); got != 10 {
		t.Fatalf("second weight = %d, want 10", got)
	}
	if got := detector.Weight("iphone"); got != 1 {
		t.Fatalf("third weight = %d, want 1", got)
	}

	detector.Reset()
	if got := detector.Weight("iphone"); got != 10 {
		t.Fatalf("weight after reset = %d, want 10", got)
	}
}
