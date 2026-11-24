package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// makeDMG writes a compressed disk image holding the app and a link to
// /Applications, so it can be dragged into place. Without the link, users run
// the app from the read-only volume, where clearing the quarantine fails.
// The app goes in through ditto, which keeps its signature intact; the image
// itself is not signed: without notarisation that changes nothing for
// Gatekeeper.
func makeDMG(app, out, volname string) error {
	if _, err := os.Stat(out); err == nil {
		return fmt.Errorf("%s already exists: remove it first", out)
	}
	if volname == "" {
		volname = strings.TrimSuffix(filepath.Base(app), ".app")
	}
	staging, err := os.MkdirTemp("", "macseal-dmg-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	if _, err := run("ditto", app, filepath.Join(staging, filepath.Base(app))); err != nil {
		return err
	}
	if err := os.Symlink("/Applications", filepath.Join(staging, "Applications")); err != nil {
		return err
	}
	// UDZO (zlib) opens on every macOS version an app may still target.
	_, err = run("hdiutil", "create", "-volname", volname, "-srcfolder", staging, "-format", "UDZO", out)
	return err
}

func cmdDMG(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("dmg", stderr)
	out := fs.String("out", "", "the .dmg to create")
	volname := fs.String("volname", "", "the volume name, the app's name by default")
	if fs.Parse(args) != nil || fs.NArg() != 1 || *out == "" {
		fmt.Fprint(stderr, "usage: macseal-builder dmg -out FILE.dmg [-volname NAME] APP\n")
		return exitUsage
	}
	if err := makeDMG(fs.Arg(0), *out, *volname); err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailed
	}
	fmt.Fprintln(stdout, "created", *out)
	return exitOK
}
