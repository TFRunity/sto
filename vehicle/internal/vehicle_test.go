package vehicle

import "testing"

func TestNormalizePlate(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"a123bc77", "А123ВС77"},
		{"  а123\tвс 77 ", "А123ВС77"},
		{"А123ВС77", "А123ВС77"},
		{" \t ", ""},
	}
	for _, tc := range cases {
		if got := normalizePlate(tc.input); got != tc.want {
			t.Errorf("normalizePlate(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
