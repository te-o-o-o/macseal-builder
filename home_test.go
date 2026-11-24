package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// sandbox points the home folder at a temporary directory and clears what
// GitHub Actions sets, so the first-use path runs in CI too.
func sandbox(t *testing.T) string {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("MACSEAL_HOME", home)
	t.Setenv("CI", "")
	t.Setenv("MACSEAL_P12_PASSWORD", "")
	return home
}

func TestSignCreatesHomeOnce(t *testing.T) {
	needMacOS(t)
	home := sandbox(t)
	v1, v2 := fixtureApp(t, "1.0.0"), fixtureApp(t, "2.0.0")
	var out, errs bytes.Buffer
	if code := cli([]string{"sign", "-name", "macseal test", v1}, &out, &errs); code != exitOK {
		t.Fatalf("first sign: %d %s", code, errs.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("created a signing identity")) {
		t.Errorf("first sign does not say it created the identity:\n%s", out.String())
	}
	for _, f := range []string{"", "config.json", "identity.p12", "identity.p12.password"} {
		info, err := os.Stat(filepath.Join(home, f))
		if err != nil {
			t.Fatal(err)
		}
		if want := os.FileMode(0o600); f == "" && info.Mode().Perm() != 0o700 || f != "" && info.Mode().Perm() != want {
			t.Errorf("%s: mode %v", f, info.Mode().Perm())
		}
	}
	var cfg config
	data, _ := os.ReadFile(filepath.Join(home, "config.json"))
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.Identity != filepath.Join(home, "identity.p12") {
		t.Errorf("config = %s (%v)", data, err)
	}

	out.Reset()
	if code := cli([]string{"sign", v2}, &out, &errs); code != exitOK {
		t.Fatalf("second sign: %d %s", code, errs.String())
	}
	if bytes.Contains(out.Bytes(), []byte("created")) {
		t.Error("the second sign created another identity")
	}
	if ok, why, err := verifyUpdate(v1, v2); !ok || err != nil {
		t.Errorf("two signs from the home folder must match: %s %v", why, err)
	}
}

func TestSignNeverRecreatesIdentity(t *testing.T) {
	needMacOS(t)
	home := sandbox(t)
	app := fixtureApp(t, "1.0.0")
	var out, errs bytes.Buffer
	if code := cli([]string{"sign", "-name", "macseal test", app}, &out, &errs); code != exitOK {
		t.Fatalf("first sign: %s", errs.String())
	}
	os.Remove(filepath.Join(home, "identity.p12"))
	if code := cli([]string{"sign", app}, &out, &errs); code != exitFailed {
		t.Errorf("sign with a lost identity: exit %d, want %d", code, exitFailed)
	}
	if _, err := os.Stat(filepath.Join(home, "identity.p12")); err == nil {
		t.Error("a lost identity was silently replaced")
	}
}

func TestSignNeverCreatesInCI(t *testing.T) {
	needMacOS(t)
	home := sandbox(t)
	t.Setenv("CI", "true")
	var out, errs bytes.Buffer
	if code := cli([]string{"sign", fixtureApp(t, "1.0.0")}, &out, &errs); code != exitFailed {
		t.Errorf("exit %d, want %d", code, exitFailed)
	}
	if _, err := os.Stat(home); err == nil {
		t.Error("CI created a home folder")
	}
}
