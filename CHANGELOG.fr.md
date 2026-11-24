# Changements

## 0.1.0 (pas encore publiée)

### Ajouté

- `cert init` crée une identité de signature de code auto-signée en PKCS#12, avec son mot de passe, sans jamais en remplacer une existante.
- `bundle` assemble un `.app` à partir d'un exécutable, d'un Info.plist et d'une icône, version comprise.
- `sign` signe une app et tout ce qu'elle contient, avec cette identité ou ad hoc, dans un trousseau jetable.
- `dmg` emballe une app signée dans une image disque avec un lien vers `/Applications`.
- `doctor` dit ce qui va casser dans une app : signature, sceau, quarantaine, clés Info.plist, architectures, en texte ou en JSON.
- `verify-update` dit si une nouvelle build garde les autorisations données à celle installée.
