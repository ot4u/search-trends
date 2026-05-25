package trending

import "testing"

func TestNormalizeQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "lowercase trim punctuation and intent words",
			input: "  Купить   Айфон-17!!! ",
			want:  "айфон 17",
		},
		{
			name:  "collapse spaces",
			input: "iphone     16     pro",
			want:  "iphone 16 pro",
		},
		{
			name:  "drops only configured intent words",
			input: "новый дешево заказать xbox",
			want:  "xbox",
		},
		{
			name:  "empty after cleanup",
			input: "!!!   ",
			want:  "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeQuery(tt.input); got != tt.want {
				t.Fatalf("NormalizeQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeTextKeepsStopWordsUntouched(t *testing.T) {
	t.Parallel()

	got := NormalizeText(" Купить !!! IPHONE ")
	if got != "купить iphone" {
		t.Fatalf("NormalizeText() = %q, want %q", got, "купить iphone")
	}
}
