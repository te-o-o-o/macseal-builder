package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	levelOK   = "ok"
	levelWarn = "warn"
	levelFail = "fail"
)

type check struct {
	Name   string `json:"name"`
	Level  string `json:"level"`
	Detail string `json:"detail"`
	Fix    string `json:"fix,omitempty"`
}

// doctor audits an app bundle and says what will break for users. It only
// reads: fixing is the job of `sign`, or of the command given in Fix.
func doctor(app string) []check {
	var checks []check
	add := func(name, level, detail, fix string) {
		checks = append(checks, check{name, level, detail, fix})
	}

	info, err := readPlist(filepath.Join(app, "Contents", "Info.plist"))
	if err != nil {
		add("bundle", levelFail, "no readable Contents/Info.plist: "+err.Error(), "")
		return checks
	}
	checks = append(checks, plistChecks(info)...)

	if name, _ := info["CFBundleExecutable"].(string); name != "" {
		exe := filepath.Join(app, "Contents", "MacOS", name)
		if _, err := os.Stat(exe); err != nil {
			add("executable", levelFail, "Contents/MacOS/"+name+" is missing", "")
		} else {
			checks = append(checks, archCheck(exe))
		}
	}

	switch kind, req, err := designated(app); {
	case err != nil:
		add("signature", levelFail, err.Error(), "")
	case kind == reqUnsigned:
		add("signature", levelFail, "not signed: Apple Silicon kills an unsigned arm64 binary at launch",
			"macseal-builder sign -adhoc "+app)
	case kind == reqAdhoc:
		add("signature", levelWarn, "ad-hoc ("+req+"): macOS forgets granted permissions (Accessibility, Screen Recording…) at every update, and an updater cannot tell a genuine update apart",
			"macseal-builder sign -p12 FILE.p12 "+app)
	default:
		add("signature", levelOK, req, "")
	}

	if out, err := run("codesign", "--verify", "--deep", "--strict", app); err != nil {
		add("seal", levelFail, "the signature does not match the content: "+strings.TrimSpace(out), "macseal-builder sign … "+app)
	} else {
		add("seal", levelOK, "the signature matches the content, nested code included", "")
	}

	if _, err := run("xattr", "-p", "com.apple.quarantine", app); err == nil {
		add("quarantine", levelWarn, "downloaded by a browser: Gatekeeper blocks the first launch",
			"xattr -dr com.apple.quarantine "+app)
	} else {
		add("quarantine", levelOK, "no quarantine flag", "")
	}
	return checks
}

// plistChecks covers the Info.plist keys whose absence breaks the app or its
// updates, without reading anything else.
func plistChecks(info map[string]any) []check {
	str := func(k string) string { s, _ := info[k].(string); return s }
	var checks []check
	if id := str("CFBundleIdentifier"); !strings.Contains(id, ".") {
		checks = append(checks, check{"identifier", levelFail, "CFBundleIdentifier is missing or not reverse-DNS (" + id + "): macOS files permissions under it", ""})
	} else {
		checks = append(checks, check{"identifier", levelOK, id, ""})
	}
	if str("CFBundleExecutable") == "" {
		checks = append(checks, check{"executable", levelFail, "CFBundleExecutable is missing: the app cannot launch", ""})
	}
	if v := str("CFBundleShortVersionString"); v == "" {
		checks = append(checks, check{"version", levelWarn, "CFBundleShortVersionString is missing: Finder and updaters see no version", ""})
	} else {
		checks = append(checks, check{"version", levelOK, v, ""})
	}
	if v := str("LSMinimumSystemVersion"); v == "" {
		checks = append(checks, check{"minimum macOS", levelWarn, "LSMinimumSystemVersion is missing: older systems try to open it and fail without a clear message", ""})
	} else {
		checks = append(checks, check{"minimum macOS", levelOK, v, ""})
	}
	return checks
}

func archCheck(exe string) check {
	out, err := run("lipo", "-archs", exe)
	if err != nil {
		return check{"architectures", levelFail, "not a Mach-O executable: " + err.Error(), ""}
	}
	archs := strings.Fields(out)
	has := func(a string) bool {
		for _, x := range archs {
			if strings.HasPrefix(x, a) { // arm64e counts as arm64
				return true
			}
		}
		return false
	}
	switch {
	case has("arm64") && has("x86_64"):
		return check{"architectures", levelOK, "universal: " + strings.Join(archs, " "), ""}
	case has("arm64"):
		return check{"architectures", levelWarn, "arm64 only: Intel Macs cannot run it", "lipo -create arm64-binary x86_64-binary -output universal-binary"}
	default:
		return check{"architectures", levelWarn, strings.Join(archs, " ") + " only: Apple Silicon runs it under Rosetta, which Apple is phasing out", "lipo -create arm64-binary x86_64-binary -output universal-binary"}
	}
}

func cmdDoctor(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("doctor", stderr)
	asJSON := fs.Bool("json", false, "print the checks as JSON")
	if fs.Parse(args) != nil || fs.NArg() != 1 {
		fmt.Fprint(stderr, "usage: macseal-builder doctor [-json] APP\n")
		return exitUsage
	}
	checks := doctor(fs.Arg(0))
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		enc.Encode(checks)
	} else {
		for _, c := range checks {
			fmt.Fprintf(stdout, "%-4s  %-14s %s\n", c.Level, c.Name, c.Detail)
			if c.Fix != "" {
				fmt.Fprintf(stdout, "      %-14s → %s\n", "", c.Fix)
			}
		}
	}
	for _, c := range checks {
		if c.Level == levelFail {
			return exitFailed
		}
	}
	return exitOK
}
