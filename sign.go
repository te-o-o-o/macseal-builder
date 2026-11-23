package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func signWithP12(app, p12, password string) error {
	if password == "" {
		return fmt.Errorf("MACSEAL_P12_PASSWORD is not set")
	}
	k, err := newKeychain()
	if err != nil {
		return err
	}
	defer k.close()
	if err := k.importP12(p12, password); err != nil {
		return err
	}
	id, err := k.identity()
	if err != nil {
		return err
	}
	restore, err := k.searchable()
	if err != nil {
		return err
	}
	defer restore()
	return signApp(app, id, k.path)
}

// signApp signs every piece of code in the bundle, innermost first, then the
// bundle itself. `codesign --deep` is not used: Apple deprecated it for
// signing, and it misses code outside the standard locations. identity is "-"
// for an ad-hoc signature, with no keychain.
func signApp(app, identity, keychain string) error {
	if _, err := os.Stat(filepath.Join(app, "Contents", "Info.plist")); err != nil {
		return fmt.Errorf("%s is not an app bundle: %v", app, err)
	}
	main, err := mainExecutable(app)
	if err != nil {
		return err
	}
	parts, err := nested(app, main)
	if err != nil {
		return err
	}
	for _, p := range append(parts, app) {
		args := []string{"--force", "--sign", identity, "--preserve-metadata=entitlements"}
		if keychain != "" {
			args = append(args, "--keychain", keychain)
		}
		if _, err := run("codesign", append(args, p)...); err != nil {
			return err
		}
	}
	_, err = run("codesign", "--verify", "--deep", "--strict", app)
	return err
}

// bundleExts are the nested bundles that carry their own signature.
var bundleExts = map[string]bool{".app": true, ".framework": true, ".xpc": true, ".appex": true, ".bundle": true, ".plugin": true}

// nested lists the code inside an app that must be signed before the app:
// nested bundles and loose Mach-O files (helpers, dylibs), deepest first so a
// bundle is sealed after what it contains. The main executable is left out:
// signing the app signs it. Symlinks are skipped, as a framework's
// Versions/Current links back to code that is already listed.
func nested(app, main string) ([]string, error) {
	var parts []string
	err := filepath.WalkDir(filepath.Join(app, "Contents"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.Type()&fs.ModeSymlink != 0:
		case d.IsDir():
			if bundleExts[filepath.Ext(p)] {
				parts = append(parts, p)
			}
		case p != main && isMachO(p):
			parts = append(parts, p)
		}
		return nil
	})
	sort.SliceStable(parts, func(i, j int) bool {
		return strings.Count(parts[i], string(filepath.Separator)) > strings.Count(parts[j], string(filepath.Separator))
	})
	return parts, err
}

// isMachO reads the magic number. 0xCAFEBABE is also a Java class file: a fat
// Mach-O follows it with a small architecture count, a class file with its
// version (45 and up).
func isMachO(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	var head [8]byte
	if _, err := io.ReadFull(f, head[:]); err != nil {
		return false
	}
	switch binary.BigEndian.Uint32(head[:4]) {
	case 0xFEEDFACE, 0xFEEDFACF, 0xCEFAEDFE, 0xCFFAEDFE:
		return true
	case 0xCAFEBABE, 0xCAFEBABF:
		return binary.BigEndian.Uint32(head[4:]) < 45
	}
	return false
}

// mainExecutable reads CFBundleExecutable from the bundle's Info.plist.
func mainExecutable(app string) (string, error) {
	info, err := readPlist(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil {
		return "", err
	}
	name, _ := info["CFBundleExecutable"].(string)
	if name == "" {
		return "", fmt.Errorf("%s: Info.plist has no CFBundleExecutable", app)
	}
	return filepath.Join(app, "Contents", "MacOS", name), nil
}

// readPlist goes through plutil, which reads the XML and binary formats
// alike, so no plist parser is needed here.
func readPlist(path string) (map[string]any, error) {
	out, err := run("plutil", "-convert", "json", "-o", "-", path)
	if err != nil {
		return nil, err
	}
	var info map[string]any
	if err := json.Unmarshal([]byte(out), &info); err != nil {
		return nil, fmt.Errorf("%s: %v", path, err)
	}
	return info, nil
}
