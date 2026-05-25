package stoplist

import "testing"

func TestServiceNormalizesAndValidatesWord(t *testing.T) {
	t.Parallel()

	service := NewService(nil)

	word, changed, err := service.Add("  SPAM!!! ")
	if err != nil {
		t.Fatalf("Add() error = %v", err)
	}
	if word != "spam" || !changed {
		t.Fatalf("unexpected add result: word=%q changed=%t", word, changed)
	}

	if _, _, err := service.Add("iphone pro"); err != ErrWordMustBeToken {
		t.Fatalf("error = %v, want %v", err, ErrWordMustBeToken)
	}
}
