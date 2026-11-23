# Options

## `cert init`

- `-name NAME`: the certificate's common name, shown by `codesign -dvv` as the authority; usually the developer's name.
- `-out FILE.p12`: the PKCS#12 identity to create, with its random password written to `FILE.p12.password`; both are `0600`, and an existing file is never replaced.

## `sign`

- `-p12 FILE.p12`: sign with this identity; its password is read from the `MACSEAL_P12_PASSWORD` environment variable, never from a flag, so it stays out of shell history.
- `-adhoc`: sign ad hoc instead, for local builds; macOS forgets the app's permissions at every rebuild.
- `APP`: the `.app` bundle, re-signed in place, nested code first.

## `doctor`

- `-json`: print the checks as a JSON array of `{name, level, detail, fix}` instead of text.
- `APP`: the `.app` bundle to audit; nothing is modified.

## `verify-update`

- `OLD.app NEW.app`: passes when `NEW.app` satisfies the designated requirement of `OLD.app`, which is what macOS checks to keep the permissions granted to `OLD.app`.

## Exit codes

- `0`: everything passed.
- `1`: a check failed (`doctor` found a `fail`, `verify-update` refused, `sign` could not sign).
- `2`: the command could not run (usage, missing file, not on macOS).
