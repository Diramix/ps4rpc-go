package ps4

import "testing"

func TestTitleIDRegex(t *testing.T) {
	cases := map[string]string{
		"CUSA10249": "CUSA10249",
		"SLUS12345": "SLUS12345",
		"garbage":   "",
	}
	for in, want := range cases {
		if got := titleIDRe.FindString(in); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"CUSA10249": "PS4",
		"SLUS12345": "PS1/2",
		"HOME00001": "Homebrew",
		"":          "",
	}
	for in, want := range cases {
		if got := classify(in); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestBaseName(t *testing.T) {
	if got := baseName("/mnt/sandbox/CUSA10249_000"); got != "CUSA10249_000" {
		t.Errorf("got %q", got)
	}
	if got := baseName("CUSA10249_000"); got != "CUSA10249_000" {
		t.Errorf("got %q", got)
	}
}
