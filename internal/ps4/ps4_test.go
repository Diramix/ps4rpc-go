package ps4

import (
	"regexp"
	"testing"
)

func classify(titleID string) string {
	if titleID == "" {
		return ""
	}
	prefix := titleID[:4]
	switch {
	case ps4Prefixes[prefix]:
		return "PS4"
	case classicPrefixes[prefix]:
		return "PS1/2"
	default:
		return "Homebrew"
	}
}

func TestTitleIDRegex(t *testing.T) {
	re := regexp.MustCompile(`^NPXS`)
	cases := map[string]string{
		"CUSA10249": "CUSA10249",
		"SLUS12345": "SLUS12345",
		"NPXS20001": "",
		"garbage":   "",
	}
	for in, want := range cases {
		if re.MatchString(in) {
			if want != "" {
				t.Errorf("%s: skipped NPXS but wanted %s", in, want)
			}
			continue
		}
		got := titleIDRe.FindString(in)
		if got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"CUSA10249": "PS4",
		"SLUS12345": "PS1/2",
		"HOME00001": "Homebrew",
	}
	for in, want := range cases {
		if got := classify(in); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}
