# Changements

## 0.1.1 (pas encore publiée)

### Corrigé

- `doctor` vérifie de nouveau les bibliothèques quand l'app est donnée par un chemin relatif, comme le fait la GitHub Action : il disait qu'otool ne pouvait pas tourner.

## 0.1.0 (2026-11-25)

### Ajouté

- `cert init` crée une identité de signature de code auto-signée en PKCS#12, avec son mot de passe, sans jamais en remplacer une existante.
- `bundle` assemble un `.app` à partir d'un exécutable, d'un Info.plist et d'une icône, version comprise.
- `sign` crée son dossier, son identité et sa config au premier usage (jamais en CI, sans jamais remplacer une identité perdue), puis signe une app et tout ce qu'elle contient, avec cette identité ou ad hoc, dans un trousseau jetable.
- `dmg` emballe une app signée dans une image disque avec un lien vers `/Applications`.
- `doctor` dit ce qui va casser dans une app : signature, sceau, quarantaine, clés Info.plist, architectures, en texte ou en JSON, y compris les bibliothèques que seule la machine de build possède.
- Une GitHub Action enchaîne bundle, sign, doctor et dmg en une étape.
- `verify-update` dit si une nouvelle build garde les autorisations données à celle installée.
