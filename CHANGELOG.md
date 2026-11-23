# Changelog

## 0.1.0 (unreleased)

### Added

- `cert init` creates a self-signed code signing identity as a PKCS#12 file and its password, never replacing an existing one.
- `sign` signs an app and everything nested in it, with that identity or ad hoc, in a throwaway keychain.
- `doctor` reports what will break in an app: signature, seal, quarantine, Info.plist keys, architectures, as text or JSON.
- `verify-update` says whether a new build keeps the permissions granted to the installed one.
