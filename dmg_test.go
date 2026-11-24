package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDMG(t *testing.T) {
	needMacOS(t)
	app := fixtureApp(t, "1.0.0")
	if err := signApp(app, "-", ""); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	out := filepath.Join(dir, "Hello.dmg")
	if err := makeDMG(app, out, ""); err != nil {
		t.Fatal(err)
	}
	if err := makeDMG(app, out, ""); err == nil {
		t.Error("an existing image should be refused")
	}

	mount := filepath.Join(dir, "mnt")
	if _, err := run("hdiutil", "attach", "-readonly", "-nobrowse", "-mountpoint", mount, out); err != nil {
		t.Fatal(err)
	}
	defer run("hdiutil", "detach", mount)

	if target, err := os.Readlink(filepath.Join(mount, "Applications")); err != nil || target != "/Applications" {
		t.Errorf("Applications link = %q, %v", target, err)
	}
	// The signature must survive the trip through the image.
	if _, err := run("codesign", "--verify", "--deep", "--strict", filepath.Join(mount, "Hello.app")); err != nil {
		t.Error(err)
	}
}
