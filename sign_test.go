package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestIsMachO(t *testing.T) {
	dir := t.TempDir()
	for name, tc := range map[string]struct {
		data []byte
		want bool
	}{
		"thin64": {[]byte{0xCF, 0xFA, 0xED, 0xFE, 0, 0, 0, 0}, true},
		"fat":    {[]byte{0xCA, 0xFE, 0xBA, 0xBE, 0, 0, 0, 2}, true},
		"java":   {[]byte{0xCA, 0xFE, 0xBA, 0xBE, 0, 0, 0, 52}, false},
		"script": {[]byte("#!/bin/sh\necho hi\n"), false},
		"short":  {[]byte{0xCF}, false},
	} {
		p := filepath.Join(dir, name)
		write(t, p, tc.data)
		if got := isMachO(p); got != tc.want {
			t.Errorf("isMachO(%s) = %v, want %v", name, got, tc.want)
		}
	}
}

func TestNestedDeepestFirst(t *testing.T) {
	macho := []byte{0xCF, 0xFA, 0xED, 0xFE, 0, 0, 0, 0}
	app := filepath.Join(t.TempDir(), "A.app")
	c := filepath.Join(app, "Contents")
	main := filepath.Join(c, "MacOS", "a")
	fw := filepath.Join(c, "Frameworks", "F.framework")
	write(t, main, macho)
	write(t, filepath.Join(c, "MacOS", "helper"), macho)
	write(t, filepath.Join(c, "MacOS", "run.sh"), []byte("#!/bin/sh\n"))
	write(t, filepath.Join(c, "Frameworks", "libx.dylib"), macho)
	write(t, filepath.Join(fw, "Versions", "A", "F"), macho)
	if err := os.Symlink("A", filepath.Join(fw, "Versions", "Current")); err != nil {
		t.Fatal(err)
	}

	parts, err := nested(app, main)
	if err != nil {
		t.Fatal(err)
	}
	pos := map[string]int{}
	for i, p := range parts {
		rel, _ := filepath.Rel(app, p)
		pos[rel] = i
	}
	want := []string{"Contents/Frameworks/F.framework/Versions/A/F", "Contents/Frameworks/F.framework", "Contents/Frameworks/libx.dylib", "Contents/MacOS/helper"}
	if len(parts) != len(want) {
		t.Fatalf("nested = %v, want %v in some order", parts, want)
	}
	for _, w := range want {
		if _, ok := pos[filepath.FromSlash(w)]; !ok {
			t.Errorf("%s missing from %v", w, parts)
		}
	}
	if pos[filepath.FromSlash(want[0])] > pos[filepath.FromSlash(want[1])] {
		t.Errorf("the framework is sealed before its binary: %v", parts)
	}
}

func TestSignAdhocThenCertificate(t *testing.T) {
	needMacOS(t)
	app := fixtureApp(t, "1.0.0")
	if err := signApp(app, "-", ""); err != nil {
		t.Fatal(err)
	}
	if kind, _, _ := designated(app); kind != reqAdhoc {
		t.Fatalf("after -adhoc: %s", kind)
	}
	helper := filepath.Join(app, "Contents", "Library", "LoginItems", "Helper.app")
	if kind, _, _ := designated(helper); kind != reqAdhoc {
		t.Fatalf("nested helper not signed: %s", kind)
	}
	if err := signWithP12(app, "missing.p12", ""); err == nil {
		t.Fatal("signing without a password should fail")
	}
}
