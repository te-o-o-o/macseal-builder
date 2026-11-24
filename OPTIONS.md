# Options

## `cert init`

- `-name NAME`: the certificate's common name, shown by `codesign -dvv` as the authority; usually the developer's name.
- `-out FILE.p12`: the PKCS#12 identity to create, with its random password written to `FILE.p12.password`; both are `0600`, and an existing file is never replaced.

## `bundle`

- `-exe FILE`: the executable, copied to `Contents/MacOS` under the name `CFBundleExecutable` gives, or its own name when the plist has none; glue it with `lipo` first for a universal app.
- `-plist FILE`: the Info.plist to start from, copied as is apart from the keys the other options set.
- `-icon FILE.icns`: optional icon, copied to `Contents/Resources` and named in `CFBundleIconFile`.
- `-version X.Y.Z`: optional, written to `CFBundleShortVersionString` and `CFBundleVersion`, so the plist can keep a placeholder.
- `-out NAME.app`: the bundle to create; an existing one is refused, never replaced.

## `sign`

- `-p12 FILE.p12`: sign with this identity; its password is read from the `MACSEAL_P12_PASSWORD` environment variable, never from a flag, so it stays out of shell history.
- `-adhoc`: sign ad hoc instead, for local builds; macOS forgets the app's permissions at every rebuild.
- `APP`: the `.app` bundle, re-signed in place, nested code first.

## `dmg`

- `-out FILE.dmg`: the compressed image to create, holding the app and a link to `/Applications`; an existing one is refused.
- `-volname NAME`: the name the mounted volume shows, the app's name by default.
- `APP`: the `.app` to put in it, signature kept intact; the image itself is not signed.

## `doctor`

- `-json`: print the checks as a JSON array of `{name, level, detail, fix}` instead of text.
- `APP`: the `.app` bundle to audit; nothing is modified.

## `verify-update`

- `OLD.app NEW.app`: passes when `NEW.app` satisfies the designated requirement of `OLD.app`, which is what macOS checks to keep the permissions granted to `OLD.app`.

## Exit codes

- `0`: everything passed.
- `1`: a check failed (`doctor` found a `fail`, `verify-update` refused, `sign` could not sign).
- `2`: the command could not run (usage, missing file, not on macOS).
