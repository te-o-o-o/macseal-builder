package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDesignated(t *testing.T) {
	for _, tc := range []struct{ out, kind, req string }{
		{
			"Executable=/A.app/Contents/MacOS/a\n# designated => cdhash H\"fedae127\"\n",
			reqAdhoc, `cdhash H"fedae127"`,
		},
		{
			"Executable=/A.app/Contents/MacOS/a\ndesignated => identifier \"x.y\" and certificate leaf = H\"ee23\"\n",
			reqCertificate, `identifier "x.y" and certificate leaf = H"ee23"`,
		},
		{
			"designated => anchor apple generic and identifier \"x.y\"\n",
			reqCertificate, `anchor apple generic and identifier "x.y"`,
		},
		{"Executable=/A.app/Contents/MacOS/a\n", "", ""},
	} {
		kind, req := parseDesignated(tc.out)
		if kind != tc.kind || req != tc.req {
			t.Errorf("parseDesignated(%q) = %q, %q; want %q, %q", tc.out, kind, req, tc.kind, tc.req)
		}
	}
}

// TestUpdateKeepsRequirement is the claim the whole tool rests on: two builds
// signed with one identity satisfy each other's requirement, while an ad-hoc
// build or another identity does not.
func TestUpdateKeepsRequirement(t *testing.T) {
	needMacOS(t)
	before, _ := run("security", "list-keychains", "-d", "user")
	t.Cleanup(func() {
		if after, _ := run("security", "list-keychains", "-d", "user"); after != before {
			t.Errorf("search list not restored:\nbefore %s\nafter  %s", before, after)
		}
	})
	dir := t.TempDir()
	mine, other := filepath.Join(dir, "mine.p12"), filepath.Join(dir, "other.p12")
	for _, p := range []string{mine, other} {
		if err := certInit("macseal test", p); err != nil {
			t.Fatal(err)
		}
	}
	sign := func(app, p12 string) string {
		t.Helper()
		pw, err := os.ReadFile(p12 + ".password")
		if err != nil {
			t.Fatal(err)
		}
		if err := signWithP12(app, p12, string(pw)); err != nil {
			t.Fatal(err)
		}
		return app
	}

	v1 := sign(fixtureApp(t, "1.0.0"), mine)
	v2 := sign(fixtureApp(t, "2.0.0"), mine)
	forged := sign(fixtureApp(t, "2.0.0"), other)
	adhoc := fixtureApp(t, "2.0.0")
	if err := signApp(adhoc, "-", ""); err != nil {
		t.Fatal(err)
	}

	if kind, _, _ := designated(v1); kind != reqCertificate {
		t.Fatalf("v1 is %s, want a certificate requirement", kind)
	}
	for _, tc := range []struct {
		name     string
		old, new string
		want     bool
	}{
		{"same identity", v1, v2, true},
		{"other identity", v1, forged, false},
		{"ad-hoc update", v1, adhoc, false},
		{"from ad-hoc", adhoc, v2, false},
	} {
		ok, why, err := verifyUpdate(tc.old, tc.new)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if ok != tc.want {
			t.Errorf("%s: ok = %v, want %v (%s)", tc.name, ok, tc.want, why)
		}
	}
}
