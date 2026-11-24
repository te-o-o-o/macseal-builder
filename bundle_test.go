package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundle(t *testing.T) {
	needMacOS(t)
	dir := t.TempDir()
	plist := filepath.Join(dir, "Info.plist")
	// No CFBundleExecutable and a stale version: bundle must fill both.
	data := strings.Replace(strings.ReplaceAll(fixturePlist, "VERSION", "0.0.0"), "<key>CFBundleExecutable</key><string>hello</string>\n", "", 1)
	write(t, plist, []byte(data))
	icon := filepath.Join(dir, "Hello.icns")
	write(t, icon, []byte("icns"))
	out := filepath.Join(dir, "Hello.app")

	o := bundleOptions{exe: hello(t), plist: plist, icon: icon, version: "1.2.3", out: out}
	if err := bundleApp(o); err != nil {
		t.Fatal(err)
	}
	info, err := readPlist(filepath.Join(out, "Contents", "Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"CFBundleExecutable": "hello", "CFBundleShortVersionString": "1.2.3", "CFBundleVersion": "1.2.3", "CFBundleIconFile": "Hello"} {
		if got, _ := info[key].(string); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
	if !isMachO(filepath.Join(out, "Contents", "MacOS", "hello")) {
		t.Error("executable not copied")
	}
	if _, err := os.Stat(filepath.Join(out, "Contents", "Resources", "Hello.icns")); err != nil {
		t.Error(err)
	}

	// The bundle is ready to sign, and then doctor has nothing to fail on.
	if err := signApp(out, "-", ""); err != nil {
		t.Fatal(err)
	}
	for _, c := range doctor(out) {
		if c.Level == levelFail {
			t.Errorf("doctor after bundle + sign: %+v", c)
		}
	}

	if err := bundleApp(o); err == nil {
		t.Error("bundling over an existing app should be refused")
	}
	o.out, o.exe = filepath.Join(dir, "Other.app"), plist
	if err := bundleApp(o); err == nil {
		t.Error("a non Mach-O executable should be refused")
	}
	var stderr bytes.Buffer
	if code := cli([]string{"bundle", "-exe", hello(t)}, &stderr, &stderr); code != exitUsage {
		t.Errorf("missing flags: exit %d", code)
	}
}
