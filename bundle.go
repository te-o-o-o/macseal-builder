package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type bundleOptions struct {
	exe, plist, icon, version, out string
}

// bundleApp assembles an unsigned .app: the executable under the name the
// Info.plist gives it, the plist with the version written in, and the icon.
// It refuses an existing destination rather than replace it: a mistyped -out
// must not delete an installed app.
func bundleApp(o bundleOptions) error {
	if filepath.Ext(o.out) != ".app" {
		return fmt.Errorf("-out %s: the bundle name must end in .app", o.out)
	}
	if _, err := os.Stat(o.out); err == nil {
		return fmt.Errorf("%s already exists: remove it first", o.out)
	}
	if !isMachO(o.exe) {
		return fmt.Errorf("%s is not a Mach-O executable", o.exe)
	}
	contents := filepath.Join(o.out, "Contents")
	plist := filepath.Join(contents, "Info.plist")
	if err := os.MkdirAll(filepath.Join(contents, "MacOS"), 0o755); err != nil {
		return err
	}
	if err := copyFile(o.plist, plist, 0o644); err != nil {
		return err
	}

	info, err := readPlist(plist)
	if err != nil {
		return err
	}
	// The plist names the executable; when it does not, the file's own name
	// is the only sensible one.
	name, _ := info["CFBundleExecutable"].(string)
	set := map[string]string{}
	if name == "" {
		name = filepath.Base(o.exe)
		set["CFBundleExecutable"] = name
	}
	if o.version != "" {
		set["CFBundleShortVersionString"] = o.version
		set["CFBundleVersion"] = o.version
	}
	if o.icon != "" {
		icon := filepath.Base(o.icon)
		if err := os.MkdirAll(filepath.Join(contents, "Resources"), 0o755); err != nil {
			return err
		}
		if err := copyFile(o.icon, filepath.Join(contents, "Resources", icon), 0o644); err != nil {
			return err
		}
		set["CFBundleIconFile"] = strings.TrimSuffix(icon, filepath.Ext(icon))
	}
	for key, value := range set {
		if _, err := run("plutil", "-replace", key, "-string", value, plist); err != nil {
			return err
		}
	}
	return copyFile(o.exe, filepath.Join(contents, "MacOS", name), 0o755)
}

func copyFile(from, to string, mode os.FileMode) error {
	src, err := os.Open(from)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

func cmdBundle(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("bundle", stderr)
	var o bundleOptions
	fs.StringVar(&o.exe, "exe", "", "the executable, already universal if it should be")
	fs.StringVar(&o.plist, "plist", "", "the Info.plist to start from")
	fs.StringVar(&o.icon, "icon", "", "an .icns icon (optional)")
	fs.StringVar(&o.version, "version", "", "written to CFBundleShortVersionString and CFBundleVersion (optional)")
	fs.StringVar(&o.out, "out", "", "the .app to create")
	if fs.Parse(args) != nil || fs.NArg() != 0 || o.exe == "" || o.plist == "" || o.out == "" {
		fmt.Fprint(stderr, "usage: macseal-builder bundle -exe FILE -plist Info.plist [-icon FILE.icns] [-version X.Y.Z] -out NAME.app\n")
		return exitUsage
	}
	if err := bundleApp(o); err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailed
	}
	fmt.Fprintf(stdout, "bundled %s, not signed yet: macseal-builder sign … %s\n", o.out, o.out)
	return exitOK
}
