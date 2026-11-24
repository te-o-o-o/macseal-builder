package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// config is macseal-builder's own settings, in its home folder. It only says
// where the signing identity is, so the identity can move to a folder that is
// backed up without anything else changing.
type config struct {
	Identity string `json:"identity"`
}

// homeDir is ~/Library/Application Support/macseal-builder, or MACSEAL_HOME.
func homeDir() (string, error) {
	if d := os.Getenv("MACSEAL_HOME"); d != "" {
		return d, nil
	}
	d, err := os.UserConfigDir()
	return filepath.Join(d, "macseal-builder"), err
}

// isCI is true on GitHub Actions and most other CI services, which set CI.
func isCI() bool { return os.Getenv("CI") != "" }

// defaultIdentity returns the identity `sign` uses when given no -p12. On the
// first run it creates the home folder, the identity and the config; after
// that it only reads them. It never creates an identity in CI, nor a second
// one when the config names a file that is gone: either would sign a release
// that no installed copy accepts as an update.
func defaultIdentity(name string) (p12 string, created bool, err error) {
	home, err := homeDir()
	if err != nil {
		return "", false, err
	}
	cfgPath := filepath.Join(home, "config.json")
	data, err := os.ReadFile(cfgPath)
	switch {
	case err == nil:
		var cfg config
		if err := json.Unmarshal(data, &cfg); err != nil || cfg.Identity == "" {
			return "", false, fmt.Errorf("%s: no identity in it (%v)", cfgPath, err)
		}
		if _, err := os.Stat(cfg.Identity); err != nil {
			return "", false, fmt.Errorf("%s names %s, which is missing: restore it from your backup; a new identity would break every update", cfgPath, cfg.Identity)
		}
		return cfg.Identity, false, nil
	case !errors.Is(err, fs.ErrNotExist):
		return "", false, err
	case isCI():
		return "", false, fmt.Errorf("no signing identity in CI: pass -p12 from a secret, never one created on the runner")
	}

	if name == "" {
		u, err := user.Current()
		if err != nil {
			return "", false, err
		}
		name = u.Username
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", false, err
	}
	p12 = filepath.Join(home, "identity.p12")
	if err := certInit(name, p12); err != nil {
		return "", false, err
	}
	data, _ = json.MarshalIndent(config{Identity: p12}, "", "  ")
	return p12, true, os.WriteFile(cfgPath, append(data, '\n'), 0o600)
}

// p12Password comes from MACSEAL_P12_PASSWORD (CI, from a secret), or else
// from the FILE.p12.password that cert init wrote next to it.
func p12Password(p12 string) string {
	if pw := os.Getenv("MACSEAL_P12_PASSWORD"); pw != "" {
		return pw
	}
	data, err := os.ReadFile(p12 + ".password")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
