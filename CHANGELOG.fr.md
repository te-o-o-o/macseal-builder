# Changements

## 0.1.0 (pas encore publiée)

### Ajouté

- `cert init` crée une identité de signature de code auto-signée en PKCS#12, avec son mot de passe, sans jamais en remplacer une existante.
- `sign` signe une app et tout ce qu'elle contient, avec cette identité ou ad hoc, dans un trousseau jetable.
- `doctor` dit ce qui va casser dans une app : signature, sceau, quarantaine, clés Info.plist, architectures, en texte ou en JSON.
- `verify-update` dit si une nouvelle build garde les autorisations données à celle installée.
