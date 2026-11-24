<h1 align="center">macseal-builder</h1>

<p align="center">Ship a macOS app without the $99 Apple tax, and without users re-granting Accessibility after every update.</p>

<p align="center"><sub>v0.1.0 · Go, zero dependencies · macOS · MIT</sub></p>

<p align="center"><a href="#features">Features</a> · <a href="#install">Install</a> · <a href="#usage">Usage</a> · <a href="#development">Development</a> · <a href="README.fr.md">Français</a></p>

## Features

- **A signing identity that lasts.** `cert init` creates a self-signed code
  signing certificate. Sign every release with it and macOS recognises each
  update as the same app: permissions like Accessibility or Screen Recording
  survive, where an ad-hoc signature loses them at every build.
- **A bundle from what you already have.** `bundle` turns an executable, an
  Info.plist and an icon into a `.app`, version written in.
- **Signing done in the right order.** `sign` seals nested helpers, frameworks
  and dylibs before the app, without the deprecated `--deep`, in a throwaway
  keychain: your login keychain is never touched.
- **A disk image to drag from.** `dmg` packs the app with a link to
  `/Applications`, signature intact.
- **A doctor before your users find out.** `doctor` reads a `.app` and reports
  what will break: no signature, an ad-hoc one, a broken seal, a quarantine
  flag, missing Info.plist keys, a single architecture, a library only your Mac
  has. `-json` for scripts.
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
macseal-builder bundle -exe dist/myapp -plist Info.plist -icon MyApp.icns -version 1.2.0 -out MyApp.app
macseal-builder sign -name "Your Name" MyApp.app

macseal-builder doctor MyApp.app
macseal-builder dmg -out MyApp.dmg MyApp.app
macseal-builder verify-update /Applications/MyApp.app MyApp.app
```

The first `sign` creates your identity in
`~/Library/Application Support/macseal-builder/` and says so: back it up, it
signs all your apps from then on. In CI, pass it from secrets with
`-p12` and `MACSEAL_P12_PASSWORD`; `sign` never creates one there.
`cert init` creates an identity somewhere else, if you prefer.

Exit codes: `0` passed, `1` a check failed, `2` the command could not run.
Every option: [OPTIONS.md](OPTIONS.md). How it works and what is verified:
[docs/notes.md](docs/notes.md).

## GitHub Actions

```yaml
- uses: te-o-o-o/macseal-builder@main
  with:
    exe: dist/myapp                 # or  app: path/to/MyApp.app
    plist: packaging/Info.plist
    icon: MyApp.icns
    version: 1.2.0
    dmg: dist/MyApp.dmg
    p12: ${{ secrets.MACOS_CERT_P12 }}          # base64 -i identity.p12 | gh secret set MACOS_CERT_P12
    password: ${{ secrets.MACOS_CERT_PASSWORD }} # gh secret set MACOS_CERT_PASSWORD < identity.p12.password
```

It bundles (with `exe`), signs, runs `doctor`, which fails the job on a
`fail`, and packs the disk image. Outputs: `app`, `dmg`.

## Development

```bash
go vet ./... && go test ./...
go build -o macseal-builder .
```

The tests build a real app bundle, create identities and sign with them, so
most of them only run on macOS.

## License

[MIT](LICENSE)
