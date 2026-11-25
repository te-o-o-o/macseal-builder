# Changelog

## 0.1.0 (2026-11-25)

### Added

- `cert init` creates a self-signed code signing identity as a PKCS#12 file and its password, never replacing an existing one.
- `bundle` assembles a `.app` from an executable, an Info.plist and an icon, with the version written in.
- `sign` creates its home folder, identity and config on first use (never in CI, never replacing a lost identity), then signs an app and everything nested in it, with that identity or ad hoc, in a throwaway keychain.
- `dmg` packs a signed app into a disk image with a link to `/Applications`.
- `doctor` reports what will break in an app: signature, seal, quarantine, Info.plist keys, architectures, as text or JSON, including libraries only the build machine has.
- A GitHub Action runs bundle, sign, doctor and dmg in one step.
- `verify-update` says whether a new build keeps the permissions granted to the installed one.
