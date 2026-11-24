package main

import (
	"os"
	"path/filepath"
	"strings"
)

// isSystemPath is where macOS ships its own libraries. Most of them are not
// files on disk any more (they live in the dyld shared cache), so a path
// there counts as found without looking.
func isSystemPath(p string) bool {
	return strings.HasPrefix(p, "/usr/lib/") || strings.HasPrefix(p, "/System/")
}

// parseDeps reads `otool -L`: one dependency per tab-indented line. A fat
// binary lists them once per architecture, hence the dedup.
func parseDeps(out string) []string {
	var deps []string
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "\t") {
			continue
		}
		dep, _, _ := strings.Cut(strings.TrimSpace(line), " (compatibility")
		if dep != "" && !seen[dep] {
			seen[dep] = true
			deps = append(deps, dep)
		}
	}
	return deps
}

// parseRpaths reads the LC_RPATH entries of `otool -l`, in order: dyld tries
// them in that order.
func parseRpaths(out string) []string {
	var rpaths []string
	seen := map[string]bool{}
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "cmd LC_RPATH" {
			continue
		}
		for _, next := range lines[i+1 : min(i+4, len(lines))] {
			next = strings.TrimSpace(next)
			if p, ok := strings.CutPrefix(next, "path "); ok {
				p, _, _ = strings.Cut(p, " (offset")
				if !seen[p] {
					seen[p] = true
					rpaths = append(rpaths, p)
				}
				break
			}
		}
	}
	return rpaths
}

// resolveDep says where dyld would find dep for a binary in loaderDir, as one
// of "bundle", "system", "outside" (a file only this machine has) or
// "missing". @executable_path is the process's executable, which for a
// framework can be the app or any of its helpers: every one in exeDirs is
// tried. exists is os.Stat, swapped in tests.
func resolveDep(dep, loaderDir string, exeDirs []string, app string, rpaths []string, exists func(string) bool) (where, path string) {
	expand := func(p string) []string {
		p = strings.Replace(p, "@loader_path", loaderDir, 1)
		if !strings.Contains(p, "@executable_path") {
			return []string{filepath.Clean(p)}
		}
		var out []string
		for _, d := range exeDirs {
			out = append(out, filepath.Clean(strings.Replace(p, "@executable_path", d, 1)))
		}
		return out
	}
	var candidates []string
	if rest, ok := strings.CutPrefix(dep, "@rpath/"); ok {
		for _, r := range rpaths {
			for _, base := range expand(r) {
				candidates = append(candidates, filepath.Join(base, rest))
			}
		}
	} else {
		candidates = expand(dep)
	}
	inBundle := func(p string) bool { return strings.HasPrefix(p, filepath.Clean(app)+string(filepath.Separator)) }
	for _, c := range candidates {
		switch {
		case isSystemPath(c):
			return "system", c
		case !exists(c):
		case inBundle(c):
			return "bundle", c
		default:
			return "outside", c
		}
	}
	return "missing", dep
}

// isExecutable tells a bundle's executable (Contents/MacOS/x) from a library.
func isExecutable(bin string) bool {
	dir := filepath.Dir(bin)
	return filepath.Base(dir) == "MacOS" && filepath.Base(filepath.Dir(dir)) == "Contents"
}

// linkCheck finds the libraries that load on this Mac and nowhere else: a
// Homebrew dylib, a build machine's toolchain. The app works for its
// developer and crashes at launch for everyone else.
func linkCheck(app, mainExe string) check {
	bins := []string{mainExe}
	parts, _ := nested(app, mainExe)
	for _, p := range parts {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			bins = append(bins, p)
		}
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }

	// The shim in /usr/bin exists without the Command Line Tools and fails:
	// the main executable is the test, since every app has one.
	if _, err := otool("-L", mainExe); err != nil {
		return check{"libraries", levelWarn, "otool cannot run (xcode-select --install): libraries not checked", ""}
	}
	// A library also searches the rpaths of the executable that loads it:
	// Electron Framework finds Squirrel.framework through the app's own
	// @executable_path/../Frameworks. Those are expanded once, up front.
	var exeDirs, inherited []string
	for _, bin := range bins {
		if !isExecutable(bin) {
			continue
		}
		exeDirs = append(exeDirs, filepath.Dir(bin))
		out, _ := otool("-l", bin)
		for _, r := range parseRpaths(out) {
			r = strings.Replace(r, "@loader_path", filepath.Dir(bin), 1)
			inherited = append(inherited, filepath.Clean(strings.Replace(r, "@executable_path", filepath.Dir(bin), 1)))
		}
	}

	var problems []string
	for _, bin := range bins {
		rel, _ := filepath.Rel(app, bin)
		depsOut, err := otool("-L", bin)
		if err != nil {
			problems = append(problems, rel+": otool cannot read it")
			continue
		}
		loadOut, _ := otool("-l", bin)
		id, _ := otool("-D", bin) // a dylib lists its own install name first
		rpaths := append(parseRpaths(loadOut), inherited...)
		for _, dep := range parseDeps(depsOut) {
			if strings.Contains(id, "\n"+dep+"\n") {
				continue
			}
			where, path := resolveDep(dep, filepath.Dir(bin), exeDirs, app, rpaths, exists)
			switch where {
			case "outside":
				problems = append(problems, rel+" loads "+path+", which only this Mac has")
			case "missing":
				problems = append(problems, rel+" needs "+dep+", found nowhere")
			}
		}
	}
	if len(problems) > 0 {
		return check{"libraries", levelFail, strings.Join(problems, "; "), "copy each library into Contents/Frameworks and point to it with @rpath (install_name_tool), then sign again"}
	}
	return check{"libraries", levelOK, "every library comes from the bundle or from macOS", ""}
}

// otool reads "name(member)" as a member of a static archive, so it cannot
// open "Obsidian Helper (GPU)", the name every Electron helper has. It gets a
// symlink without parentheses instead.
func otool(flag, bin string) (string, error) {
	dir, err := os.MkdirTemp("", "macseal-otool-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	link := filepath.Join(dir, "bin")
	if err := os.Symlink(bin, link); err != nil {
		return "", err
	}
	out, err := run("otool", flag, link)
	return strings.ReplaceAll(out, link, bin), err
}
