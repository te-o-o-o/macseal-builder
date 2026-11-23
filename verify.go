package main

import (
	"fmt"
	"io"
	"strings"
)

// Kinds of designated requirement, which is what macOS checks a new build
// against: TCC for the permissions it granted, an updater before replacing
// the app.
const (
	reqUnsigned    = "unsigned"
	reqAdhoc       = "ad-hoc"      // cdhash: that exact build, and nothing else
	reqCertificate = "certificate" // identifier + certificate: any build signed with it
)

// designated reads the designated requirement of signed code.
func designated(path string) (kind, req string, err error) {
	out, err := run("codesign", "-d", "-r-", path)
	if err != nil {
		if strings.Contains(out, "not signed at all") {
			return reqUnsigned, "", nil
		}
		return "", "", err
	}
	kind, req = parseDesignated(out)
	if kind == "" {
		return "", "", fmt.Errorf("%s: no designated requirement in codesign output: %s", path, strings.TrimSpace(out))
	}
	return kind, req, nil
}

// parseDesignated reads `codesign -d -r-`. An ad-hoc signature has no stored
// requirement: codesign prints the implicit one commented out,
// `# designated => cdhash H"…"`.
func parseDesignated(out string) (kind, req string) {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		adhoc := strings.HasPrefix(line, "#")
		_, after, found := strings.Cut(line, "designated => ")
		if !found {
			continue
		}
		if adhoc || strings.HasPrefix(after, "cdhash ") {
			return reqAdhoc, after
		}
		return reqCertificate, after
	}
	return "", ""
}

// verifyUpdate says whether newApp can replace oldApp and keep what users
// granted to it: newApp must satisfy oldApp's designated requirement.
func verifyUpdate(oldApp, newApp string) (ok bool, why string, err error) {
	kind, req, err := designated(oldApp)
	if err != nil {
		return false, "", err
	}
	switch kind {
	case reqUnsigned:
		return false, oldApp + " is not signed", nil
	case reqAdhoc:
		return false, oldApp + " is ad-hoc signed: its requirement is that exact build, no update can satisfy it", nil
	}
	if _, err := run("codesign", "--verify", "--strict", "-R", "="+req, newApp); err != nil {
		return false, newApp + " does not satisfy " + req, nil
	}
	return true, newApp + " satisfies " + req, nil
}

func cmdVerify(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprint(stderr, "usage: macseal-builder verify-update OLD.app NEW.app\n")
		return exitUsage
	}
	ok, why, err := verifyUpdate(args[0], args[1])
	if err != nil {
		fmt.Fprintln(stderr, err)
		return exitUsage
	}
	if !ok {
		fmt.Fprintln(stdout, "fail:", why)
		return exitFailed
	}
	fmt.Fprintln(stdout, "ok:", why)
	return exitOK
}
