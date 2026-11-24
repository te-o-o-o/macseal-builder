// macseal-builder prepares a macOS app for distribution without an Apple
// Developer account: a stable self-signed identity, a clean inside-out
// signature, and checks that say what will break before users find out.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
)

// Exit codes: 0 everything passed, 1 a check failed, 2 the command could not run.
const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 2
)

const usage = `macseal-builder: sign and check macOS apps without an Apple Developer account

  macseal-builder cert init -name NAME -out FILE.p12
  macseal-builder bundle -exe FILE -plist Info.plist [-icon FILE.icns] [-version X.Y.Z] -out NAME.app
  macseal-builder sign (-p12 FILE.p12 | -adhoc) APP
  macseal-builder doctor [-json] APP
  macseal-builder verify-update OLD.app NEW.app

The PKCS#12 password is read from MACSEAL_P12_PASSWORD, never from a flag.
Every option: OPTIONS.md.
`

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr))
}

func cli(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "-help" || args[0] == "help" {
		fmt.Fprint(stderr, usage)
		return exitUsage
	}
	if runtime.GOOS != "darwin" {
		fmt.Fprintln(stderr, "macseal-builder drives codesign and security: it only runs on macOS")
		return exitUsage
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "cert":
		return cmdCert(rest, stdout, stderr)
	case "bundle":
		return cmdBundle(rest, stdout, stderr)
	case "sign":
		return cmdSign(rest, stdout, stderr)
	case "doctor":
		return cmdDoctor(rest, stdout, stderr)
	case "verify-update":
		return cmdVerify(rest, stdout, stderr)
	}
	fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, usage)
	return exitUsage
}

func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

func cmdCert(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "init" {
		fmt.Fprint(stderr, "usage: macseal-builder cert init -name NAME -out FILE.p12\n")
		return exitUsage
	}
	fs := newFlags("cert init", stderr)
	name := fs.String("name", "", "certificate common name, usually the developer's name")
	out := fs.String("out", "", "PKCS#12 file to create; the password goes to FILE.p12.password")
	if fs.Parse(args[1:]) != nil || *name == "" || *out == "" {
		fmt.Fprint(stderr, "usage: macseal-builder cert init -name NAME -out FILE.p12\n")
		return exitUsage
	}
	if err := certInit(*name, *out); err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	fmt.Fprintf(stdout, `created %[1]s and %[1]s.password

Back both up outside this machine now. Every app signed with this identity
refuses an update signed with another one, and macOS forgets its permissions.

For GitHub Actions:
  base64 -i %[1]s | gh secret set MACOS_CERT_P12
  gh secret set MACOS_CERT_PASSWORD < %[1]s.password
`, *out)
	return exitOK
}

func cmdSign(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("sign", stderr)
	p12 := fs.String("p12", "", "PKCS#12 identity to sign with (password in MACSEAL_P12_PASSWORD)")
	adhoc := fs.Bool("adhoc", false, "ad-hoc signature, for local builds: permissions do not survive a rebuild")
	if fs.Parse(args) != nil || fs.NArg() != 1 || (*p12 == "") == !*adhoc {
		fmt.Fprint(stderr, "usage: macseal-builder sign (-p12 FILE.p12 | -adhoc) APP\n")
		return exitUsage
	}
	app := fs.Arg(0)
	var err error
	if *adhoc {
		err = signApp(app, "-", "")
	} else {
		err = signWithP12(app, *p12, os.Getenv("MACSEAL_P12_PASSWORD"))
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailed
	}
	kind, req, err := designated(app)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitFailed
	}
	fmt.Fprintf(stdout, "signed %s (%s)\n  %s\n", app, kind, req)
	return exitOK
}
