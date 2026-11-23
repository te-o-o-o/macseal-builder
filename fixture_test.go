package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func needMacOS(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "darwin" {
		t.Skip("drives codesign and security: macOS only")
	}
}

var (
	helloOnce sync.Once
	helloBin  string
	helloErr  error
)

// hello builds testdata/hello once per test run: a real Mach-O, which the Go
// linker already signs ad hoc, like any toolchain on Apple Silicon.
func hello(t *testing.T) string {
	t.Helper()
	helloOnce.Do(func() {
		dir, err := os.MkdirTemp("", "macseal-hello-")
		if err != nil {
			helloErr = err
			return
		}
		helloBin = filepath.Join(dir, "hello")
		out, err := exec.Command("go", "build", "-o", helloBin, "./testdata/hello").CombinedOutput()
		if err != nil {
			helloErr = err
			t.Logf("%s", out)
		}
	})
	if helloErr != nil {
		t.Fatal(helloErr)
	}
	return helloBin
}

const fixturePlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleIdentifier</key><string>io.github.te-o-o-o.hello</string>
<key>CFBundleExecutable</key><string>hello</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>VERSION</string>
<key>CFBundleVersion</key><string>VERSION</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
</dict></plist>
`

// fixtureApp assembles Hello.app around the hello binary, with a nested helper
// so the inside-out signing has something to do. Two different versions give
// two different cdhashes, as two releases would.
func fixtureApp(t *testing.T, version string) string {
	t.Helper()
	bin := hello(t)
	app := filepath.Join(t.TempDir(), "Hello.app")
	helper := filepath.Join(app, "Contents", "Library", "LoginItems", "Helper.app")
	for _, dir := range []string{filepath.Join(app, "Contents", "MacOS"), filepath.Join(helper, "Contents", "MacOS")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	plist := []byte(strings.ReplaceAll(fixturePlist, "VERSION", version))
	helperPlist := []byte(strings.ReplaceAll(fixturePlist, "VERSION", version+"-helper"))
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string][]byte{
		filepath.Join(app, "Contents", "Info.plist"):        plist,
		filepath.Join(app, "Contents", "MacOS", "hello"):    data,
		filepath.Join(helper, "Contents", "Info.plist"):     helperPlist,
		filepath.Join(helper, "Contents", "MacOS", "hello"): data,
	} {
		if err := os.WriteFile(path, content, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return app
}
