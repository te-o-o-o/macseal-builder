package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// run calls one of the macOS tools and returns stdout and stderr together:
// codesign prints its results on stderr. The error names the tool and its
// subcommand only, never the whole argument list, which can hold a password.
func run(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		sub := ""
		if len(args) > 0 {
			sub = " " + args[0]
		}
		return string(out), fmt.Errorf("%s%s: %v: %s", name, sub, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand does not fail on macOS
	}
	return hex.EncodeToString(b)
}

// keychain is a throwaway keychain in its own temporary directory. It is never
// added to the user's search list: codesign and security get its path
// explicitly, so the login keychain is never touched.
type keychain struct {
	dir, path, password string
}

func newKeychain() (*keychain, error) {
	dir, err := os.MkdirTemp("", "macseal-")
	if err != nil {
		return nil, err
	}
	k := &keychain{dir: dir, path: filepath.Join(dir, "sign.keychain-db"), password: randomHex(16)}
	for _, args := range [][]string{
		{"create-keychain", "-p", k.password, k.path},
		{"unlock-keychain", "-p", k.password, k.path},
		// No arguments means no auto-lock timeout: a long signing run must not
		// find the keychain locked halfway.
		{"set-keychain-settings", k.path},
	} {
		if _, err := run("security", args...); err != nil {
			k.close()
			return nil, err
		}
	}
	return k, nil
}

// importP12 adds a signing identity. The partition list lets codesign use the
// key without the "allow access" dialog, which would hang a CI run.
func (k *keychain) importP12(file, password string) error {
	if _, err := run("security", "import", file, "-k", k.path, "-P", password, "-T", "/usr/bin/codesign"); err != nil {
		return err
	}
	_, err := run("security", "set-key-partition-list", "-S", "apple-tool:,apple:", "-k", k.password, k.path)
	return err
}

var identityLine = regexp.MustCompile(`\d+\) ([0-9A-F]{40}) "`)

// identity returns the SHA-1 of the first code signing identity. A self-signed
// certificate is listed as CSSMERR_TP_NOT_TRUSTED, never as "valid", so the
// valid-only listing (-v) would find nothing; codesign signs with it anyway.
func (k *keychain) identity() (string, error) {
	out, err := run("security", "find-identity", "-p", "codesigning", k.path)
	if err != nil {
		return "", err
	}
	return parseIdentity(out)
}

func parseIdentity(out string) (string, error) {
	m := identityLine.FindStringSubmatch(out)
	if m == nil {
		return "", fmt.Errorf("no code signing identity in the keychain")
	}
	return m[1], nil
}

func (k *keychain) close() {
	run("security", "delete-keychain", k.path)
	os.RemoveAll(k.dir)
}
