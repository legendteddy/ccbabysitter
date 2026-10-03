package buildinfo

import (
	"regexp"
	"testing"
)

func TestVersionIsSemver(t *testing.T) {
	// The release workflow accepts the same forms.
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$`).MatchString(Version) {
		t.Fatalf("Version %q is not MAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH-<pre>", Version)
	}
	if Name != "CC Babysitter" {
		t.Fatalf("Name = %q", Name)
	}
}
