# Notes

## How it works

- **macOS remembers permissions by designated requirement.** TCC (the
  permission database behind Accessibility, Screen Recording, Input
  Monitoring…) stores the requirement of the app at the moment the user
  granted it, and checks every later launch against it.
- **Ad hoc means `cdhash`.** An ad-hoc signature has no certificate, so its
  requirement is the hash of that exact build. Any rebuild is a stranger: the
  old entry stays ticked in System Settings and matches nothing.
- **A certificate means `identifier + certificate leaf`.** Every build signed
  with the same certificate and bundle identifier satisfies it. The
  certificate does not have to be trusted: codesign signs with a self-signed
  one (listed as `CSSMERR_TP_NOT_TRUSTED`) and the requirement check works.
- **One identity per developer, not per app.** The bundle identifier already
  tells apps apart. Never share an identity between developers: whoever holds
  the key can sign an update every one of their apps would accept.

## Choices

- **Throwaway keychains.** Signing imports the identity into a keychain in a
  temporary directory, deleted afterwards: the login keychain is never touched.
  It joins the user's search list only while codesign runs, and the previous
  list is put back: macOS 15 codesign cannot find the identity otherwise
  (`--keychain` alone works on macOS 27).
- **The PKCS#12 goes through `security export`.** Go's standard library has
  no PKCS#12 encoder, and a file written by OpenSSL 3 (AES by default) is
  refused by older macOS versions.
- **A 20-year certificate.** Replacing it changes the requirement, so users
  lose their permissions. A root with per-app leaves and a `certificate root`
  requirement would allow rotation; not done until that is verified with TCC.
- **No `--deep` to sign.** Apple deprecated it; nested code is signed one by
  one, deepest first. `--deep` is still used to verify.

## Verified, per macOS version

| Claim | How | Where |
|---|---|---|
| Two builds signed with one self-signed identity satisfy each other's requirement; an ad-hoc build or another identity does not | `TestUpdateKeepsRequirement`, and by hand on copies of copycopy 0.6.0 | macOS 27.2.1, Apple Silicon |
| Signing and identity creation run without any dialog | `go test`, and by hand | macOS 27.2.1, Apple Silicon |
| **TCC keeps the Accessibility grant across two builds signed with one identity** | **not verified yet**: install, grant, install the next build, use the permission | |
| Intel Macs, older macOS versions | not run | |

CI on `macos-15` runs the same tests: it shows they pass there, not that TCC
behaves.
