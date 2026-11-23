package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func level(checks []check, name string) string {
	for _, c := range checks {
		if c.Name == name {
			return c.Level
		}
	}
	return "absent"
}

func TestPlistChecks(t *testing.T) {
	good := map[string]any{
		"CFBundleIdentifier": "io.x.app", "CFBundleExecutable": "app",
		"CFBundleShortVersionString": "1.0", "LSMinimumSystemVersion": "11.0",
	}
	for _, c := range plistChecks(good) {
		if c.Level != levelOK {
			t.Errorf("good plist: %+v", c)
		}
	}
	bad := plistChecks(map[string]any{"CFBundleIdentifier": "app"})
	for name, want := range map[string]string{"identifier": levelFail, "executable": levelFail, "version": levelWarn, "minimum macOS": levelWarn} {
		if got := level(bad, name); got != want {
			t.Errorf("%s = %s, want %s", name, got, want)
		}
	}
}

func TestParseIdentity(t *testing.T) {
	out := "\nPolicy: Code Signing\n  Matching identities\n  1) EE23151BA47CFAA8B17F2CC793E58B5E22BA1BA1 \"x\" (CSSMERR_TP_NOT_TRUSTED)\n     1 identities found\n"
	if id, err := parseIdentity(out); err != nil || id != "EE23151BA47CFAA8B17F2CC793E58B5E22BA1BA1" {
		t.Errorf("parseIdentity = %q, %v", id, err)
	}
	if _, err := parseIdentity("  0 identities found\n"); err == nil {
		t.Error("no identity should be an error")
	}
}

func TestDoctor(t *testing.T) {
	needMacOS(t)
	app := fixtureApp(t, "1.0.0")
	if err := signApp(app, "-", ""); err != nil {
		t.Fatal(err)
	}
	checks := doctor(app)
	for name, want := range map[string]string{"signature": levelWarn, "seal": levelOK, "quarantine": levelOK, "identifier": levelOK} {
		if got := level(checks, name); got != want {
			t.Errorf("signed ad hoc: %s = %s, want %s", name, got, want)
		}
	}

	if _, err := run("xattr", "-w", "com.apple.quarantine", "0081;00000000;Safari;", app); err != nil {
		t.Fatal(err)
	}
	if got := level(doctor(app), "quarantine"); got != levelWarn {
		t.Errorf("quarantined: %s", got)
	}

	// Changing the plist after signing breaks the seal.
	plist := filepath.Join(app, "Contents", "Info.plist")
	data, _ := os.ReadFile(plist)
	write(t, plist, bytes.Replace(data, []byte("1.0.0"), []byte("1.0.1"), 1))
	if got := level(doctor(app), "seal"); got != levelFail {
		t.Errorf("tampered: seal = %s", got)
	}

	var stdout, stderr bytes.Buffer
	if code := cli([]string{"doctor", "-json", app}, &stdout, &stderr); code != exitFailed {
		t.Errorf("doctor exit = %d, want %d", code, exitFailed)
	}
	var parsed []check
	if err := json.Unmarshal(stdout.Bytes(), &parsed); err != nil || len(parsed) == 0 {
		t.Errorf("-json output: %v\n%s", err, stdout.String())
	}
}

func TestDoctorUnsigned(t *testing.T) {
	needMacOS(t)
	app := fixtureApp(t, "1.0.0")
	// The Go linker signs every arm64 binary ad hoc: strip it to get the
	// unsigned app an x86_64 toolchain or a patched binary leaves behind.
	if _, err := run("codesign", "--remove-signature", filepath.Join(app, "Contents", "MacOS", "hello")); err != nil {
		t.Fatal(err)
	}
	if got := level(doctor(app), "signature"); got != levelFail {
		t.Errorf("unsigned: signature = %s, want fail", got)
	}
}

func TestCertInitRefusesOverwrite(t *testing.T) {
	needMacOS(t)
	p := filepath.Join(t.TempDir(), "id.p12")
	if err := certInit("macseal test", p); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(p)
	if err := certInit("macseal test", p); err == nil {
		t.Fatal("second certInit should refuse")
	}
	after, _ := os.ReadFile(p)
	if !bytes.Equal(before, after) {
		t.Fatal("the identity was replaced")
	}
	if info, _ := os.Stat(p + ".password"); info == nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("password file mode: %v", info)
	}
}
