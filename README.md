<h1 align="center">macseal-builder</h1>

<p align="center">Ship a macOS app without the $99 Apple tax, and without users re-granting Accessibility after every update.</p>

<p align="center"><sub>v0.1.0 · Go, zero dependencies · macOS · MIT</sub></p>

<p align="center"><a href="#features">Features</a> · <a href="#install">Install</a> · <a href="#usage">Usage</a> · <a href="#development">Development</a> · <a href="README.fr.md">Français</a></p>

## Features

- **A signing identity that lasts.** `cert init` creates a self-signed code
  signing certificate. Sign every release with it and macOS recognises each
  update as the same app: permissions like Accessibility or Screen Recording
  survive, where an ad-hoc signature loses them at every build.
- **Signing done in the right order.** `sign` seals nested helpers, frameworks
  and dylibs before the app, without the deprecated `--deep`, in a throwaway
  keychain: your login keychain is never touched.
- **A doctor before your users find out.** `doctor` reads a `.app` and reports
  what will break: no signature, an ad-hoc one, a broken seal, a quarantine
  flag, missing Info.plist keys, a single architecture. `-json` for scripts.
- **A safe updater check.** `verify-update OLD.app NEW.app` says whether the
  new build can replace the old one and keep what users granted. It is the
  check an updater should run before swapping the app.

What it cannot do: get past Gatekeeper. Without notarisation, users still
confirm the first launch once (System Settings → Privacy & Security → Open
Anyway, or `xattr -dr com.apple.quarantine`).

## Install

From the [latest release](https://github.com/te-o-o-o/macseal-builder/releases/latest),
with `curl`, which sets no quarantine flag:

```bash
curl -fLo macseal-builder https://github.com/te-o-o-o/macseal-builder/releases/latest/download/macseal-builder-darwin-arm64
chmod +x macseal-builder
```

`darwin-amd64` for Intel Macs. Downloaded with a browser instead:
`xattr -d com.apple.quarantine macseal-builder`.

From source: `go install github.com/te-o-o-o/macseal-builder@latest`.

## Usage

```bash
# Once per developer, for all your apps. Back up both files.
macseal-builder cert init -name "Your Name" -out ~/secure/signing.p12

# Each release
MACSEAL_P12_PASSWORD="$(cat ~/secure/signing.p12.password)" \
  macseal-builder sign -p12 ~/secure/signing.p12 MyApp.app

macseal-builder doctor MyApp.app
macseal-builder verify-update /Applications/MyApp.app MyApp.app
```

Exit codes: `0` passed, `1` a check failed, `2` the command could not run.
Every option: [OPTIONS.md](OPTIONS.md). How it works and what is verified:
[docs/notes.md](docs/notes.md).

## Development

```bash
go vet ./... && go test ./...
go build -o macseal-builder .
```

The tests build a real app bundle, create identities and sign with them, so
most of them only run on macOS.

## License

[MIT](LICENSE)
