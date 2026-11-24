package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDeps(t *testing.T) {
	out := "/A.app/Contents/MacOS/a (architecture x86_64):\n" +
		"\t/usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1351.0.0)\n" +
		"\t@rpath/libfoo.dylib (compatibility version 1.0.0, current version 1.0.0)\n" +
		"/A.app/Contents/MacOS/a (architecture arm64):\n" +
		"\t/usr/lib/libSystem.B.dylib (compatibility version 1.0.0, current version 1351.0.0)\n"
	got := parseDeps(out)
	if len(got) != 2 || got[0] != "/usr/lib/libSystem.B.dylib" || got[1] != "@rpath/libfoo.dylib" {
		t.Errorf("parseDeps = %q", got)
	}
}

func TestParseRpaths(t *testing.T) {
	// The wrkspaces 0.1.6 binary: Swift adds the build machine's toolchain.
	out := "Load command 20\n          cmd LC_RPATH\n      cmdsize 32\n         path /usr/lib/swift (offset 12)\n" +
		"Load command 21\n          cmd LC_RPATH\n      cmdsize 32\n         path @loader_path (offset 12)\n" +
		"Load command 22\n          cmd LC_RPATH\n      cmdsize 80\n         path /Library/Developer/CommandLineTools/usr/lib/swift-6.2/macosx (offset 12)\n"
	got := parseRpaths(out)
	if len(got) != 3 || got[0] != "/usr/lib/swift" || got[2] != "/Library/Developer/CommandLineTools/usr/lib/swift-6.2/macosx" {
		t.Errorf("parseRpaths = %q", got)
	}
}

func TestResolveDep(t *testing.T) {
	app := "/A.app"
	main := "/A.app/Contents/MacOS"
	helper := "/A.app/Contents/Frameworks/A Helper (GPU).app/Contents/MacOS"
	files := map[string]bool{
		"/A.app/Contents/Frameworks/E.framework/E":                                        true,
		"/A.app/Contents/Frameworks/libfoo.dylib":                                         true,
		"/opt/homebrew/lib/libpng.dylib":                                                  true,
		"/Library/Developer/CommandLineTools/usr/lib/swift-6.2/macosx/libswiftCore.dylib": true,
	}
	exists := func(p string) bool { return files[p] }
	for _, tc := range []struct {
		dep    string
		rpaths []string
		want   string
	}{
		{"/usr/lib/libSystem.B.dylib", nil, "system"},
		{"/opt/homebrew/lib/libpng.dylib", nil, "outside"},
		{"/opt/homebrew/lib/libgone.dylib", nil, "missing"},
		{"@executable_path/../Frameworks/libfoo.dylib", nil, "bundle"},
		{"@rpath/libfoo.dylib", []string{"@loader_path/../Frameworks"}, "bundle"},
		// /usr/lib/swift comes first: the toolchain rpath is never reached.
		{"@rpath/libswiftCore.dylib", []string{"/usr/lib/swift", "/Library/Developer/CommandLineTools/usr/lib/swift-6.2/macosx"}, "system"},
		{"@rpath/libswiftCore.dylib", []string{"/Library/Developer/CommandLineTools/usr/lib/swift-6.2/macosx"}, "outside"},
		{"@rpath/libnone.dylib", []string{"@loader_path"}, "missing"},
		// An Electron helper: @executable_path is the helper, three levels below Frameworks.
		{"@rpath/E.framework/E", []string{"@executable_path/../../.."}, "bundle"},
	} {
		if got, _ := resolveDep(tc.dep, main, []string{main, helper}, app, tc.rpaths, exists); got != tc.want {
			t.Errorf("resolveDep(%s, %v) = %s, want %s", tc.dep, tc.rpaths, got, tc.want)
		}
	}
}

func TestLinkCheck(t *testing.T) {
	needMacOS(t)
	app := fixtureApp(t, "1.0.0")
	exe := filepath.Join(app, "Contents", "MacOS", "hello")
	if c := linkCheck(app, exe); c.Level != levelOK {
		t.Errorf("a Go binary only needs libSystem: %+v", c)
	}
	// Electron names its helpers "App Helper (GPU)": otool must still read them.
	paren := filepath.Join(app, "Contents", "MacOS", "Hello Helper (GPU)")
	if err := copyFile(exe, paren, 0o755); err != nil {
		t.Fatal(err)
	}
	if c := linkCheck(app, exe); c.Level != levelOK {
		t.Errorf("a helper with parentheses: %+v", c)
	}
	// Point the binary at a library outside the bundle, as a Homebrew build would.
	ext := filepath.Join(t.TempDir(), "libext.dylib")
	if err := os.WriteFile(ext, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run("install_name_tool", "-change", "/usr/lib/libSystem.B.dylib", ext, exe); err != nil {
		t.Fatal(err)
	}
	if c := linkCheck(app, exe); c.Level != levelFail {
		t.Errorf("an external library must fail: %+v", c)
	}
}
