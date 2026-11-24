# macseal-builder

Signs and checks macOS apps without an Apple Developer account. Go, stdlib
only, one `main` package; it drives `codesign`, `security`, `plutil`, `lipo`
and `xattr`, which ship with macOS. Decisions and what is verified:
`docs/notes.md`.

## Layout

| File | Role | Rule |
|---|---|---|
| `main.go` | `cli`, one `cmd*` per subcommand, exit codes | Commands write to the given writers, so tests call `cli` directly. |
| `tools.go` | `run`, throwaway `keychain` | `run` errors name the tool and subcommand only: arguments can hold passwords. |
| `cert.go` | `cert init`: self-signed identity → PKCS#12 | Never overwrite an identity. |
| `bundle.go` | `bundle`: executable + Info.plist + icon → `.app` | Never replace an existing destination. |
| `sign.go` | inside-out signing, `nested`, `isMachO`, `readPlist` | Deepest first; no `--deep` to sign. |
| `verify.go` | designated requirement, `verify-update` | |
| `doctor.go` | read-only audit | Never modifies the app: fixes go in `Fix`. |
| `fixture_test.go` | builds `testdata/hello` into a real `Hello.app` | |
| `OPTIONS.md` | every option in one sentence | Update it with every flag. |

## Commands

```bash
go vet ./... && go test ./...        # before every commit
go build -o macseal-builder .
```

## Conventions

- Commits: Conventional Commits, one line, no body, no trailer. Scopes:
  `cert`, `sign`, `doctor`, `verify`. None when nothing fits, none on
  `chore: release X.Y.Z`.
- Docs come in pairs: `README.md` / `README.fr.md`, `docs/notes.md` /
  `docs/notes.fr.md`, `CHANGELOG.md` / `CHANGELOG.fr.md`.
- Release: bump the README badge (both READMEs), date the CHANGELOG, then
  `chore: release X.Y.Z` and `git tag vX.Y.Z && git push origin vX.Y.Z`.
  `release.yml` refuses a tag that does not match the badge.

## Traps

- **A relative keychain name lands in `~/Library/Keychains/`.** `security
  create-keychain test.keychain-db` does not create it in the current
  directory. Always pass an absolute path.
- **macOS 15 codesign ignores `--keychain` alone**: "The specified item could
  not be found in the keychain". The keychain must be in the search list while
  signing (`keychain.searchable`), and the old list put back. macOS 27 signs
  without it, so only CI showed it.
- **A `lipo`-glued universal binary is half signed.** The Go and Rust linkers
  sign arm64 ad hoc, not x86_64, and nothing seals the bundle: `codesign`
  says "not signed at all" for the app, yet it launches (copycopy 0.7.0).
  Judge each slice (`unsignedCheck`).
- **Quarantine flag `0x40` means already approved** (`01c1;…` after the user
  opened it, `0081;…` before). Only warn without it.
- **`security find-identity -v` hides self-signed identities** (they are
  `CSSMERR_TP_NOT_TRUSTED`). Parse the full listing.
- **`codesign -R` takes `=requirement` for text**, otherwise it reads a file:
  `-R "=$req"`, never `-R="=…"`.
- **An ad-hoc signature prints its requirement commented out**:
  `# designated => cdhash H"…"`.
- **Editing a signed binary in place makes it invalid Mach-O for codesign**:
  to get a different cdhash in a test, change the Info.plist, not the binary.
