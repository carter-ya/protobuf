package version

import "testing"

func TestAtLeastOrdering(t *testing.T) {
	for _, test := range []struct {
		minimum, actual string
		want            bool
	}{
		{"3.14.0", "3.21.8", true}, {"3.0.0", "36.2", true},
		{"36.2.0", "36.2", true}, {"36.2", "36.1", false},
		{"4.0.0", "3.10.0", false}, {"3.9.9", "3.10.0", true},
		{"3.21.8", "3.21.7", false}, {"36.2.1", "36.2", false},
	} {
		if got := less(test.minimum, test.actual); got != test.want {
			t.Errorf("minimum %s, actual %s: got %v, want %v", test.minimum, test.actual, got, test.want)
		}
	}
}

func TestMalformedVersions(t *testing.T) {
	for _, input := range []string{"", "36", "36.2.0.1", "36..2", "36.-1", "36.+1", "36.2-rc1"} {
		if _, err := parseVersion(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}

func TestCompilerReleaseCandidateOrdering(t *testing.T) {
	for _, test := range []struct {
		minimum, actual string
		want            bool
	}{
		{"3.0.0", "37.0-rc1", true}, {"36.2", "37.0-rc1", true},
		{"37.0", "37.0-rc1", false}, {"37.0-rc1", "37.0", true},
		{"37.0-rc1", "37.0.0-rc1", true}, {"37.0-rc2", "37.0-rc1", false},
		{"37.0-rc2", "37.0-rc10", true}, {"37.0-rc10", "37.0-rc2", false},
		{"37.0-rc-1", "37.0-rc1", true}, {"37.0.1-rc1", "37.0", false},
	} {
		if got := less(test.minimum, test.actual); got != test.want {
			t.Errorf("minimum %s actual %s: %v want %v", test.minimum, test.actual, got, test.want)
		}
	}
	for _, input := range []string{"37.0-rc", "37.0-rc-", "37.0-rcx", "37.0-beta1", "37.0-rc1-extra", "37.0-rc999999999999999999999999999999"} {
		if _, err := parseCompilerVersion(input); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
}
