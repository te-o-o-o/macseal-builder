<h1 align="center">macseal-builder</h1>

<p align="center">Distribuer une app macOS sans la taxe Apple à 99 $, et sans réautoriser Accessibilité à chaque mise à jour.</p>

<p align="center"><sub>v0.1.0 · Go, zéro dépendance · macOS · MIT</sub></p>

<p align="center"><a href="#fonctionnalités">Fonctionnalités</a> · <a href="#installation">Installation</a> · <a href="#utilisation">Utilisation</a> · <a href="#développement">Développement</a> · <a href="README.md">English</a></p>

## Fonctionnalités

- **Une identité de signature qui dure.** `cert init` crée un certificat de
  signature de code auto-signé. Toutes les releases signées avec lui sont
  reconnues par macOS comme la même app : les autorisations comme
  Accessibilité ou Enregistrement de l'écran survivent, là où une signature
  ad hoc les perd à chaque build.
- **Un bundle à partir de ce que vous avez déjà.** `bundle` fait un `.app`
  d'un exécutable, d'un Info.plist et d'une icône, version comprise.
- **Une signature dans le bon ordre.** `sign` scelle les helpers, frameworks
  et dylibs imbriqués avant l'app, sans le `--deep` déprécié, dans un trousseau
  jetable : votre trousseau de session n'est jamais touché.
- **Un diagnostic avant vos utilisateurs.** `doctor` lit un `.app` et dit ce
  qui va casser : pas de signature, une signature ad hoc, un sceau cassé, un
  drapeau de quarantaine, des clés Info.plist manquantes, une seule
  architecture. `-json` pour les scripts.
- **Une vérification de mise à jour sûre.** `verify-update ANCIEN.app NOUVEAU.app`
  dit si la nouvelle build peut remplacer l'ancienne en gardant ce que les
  utilisateurs ont autorisé. C'est le test qu'un updater doit faire avant de
  remplacer l'app.

Ce qu'il ne fait pas : passer Gatekeeper. Sans notarisation, l'utilisateur
confirme encore le premier lancement une fois (Réglages Système →
Confidentialité et sécurité → Ouvrir quand même, ou
`xattr -dr com.apple.quarantine`).

## Installation

Depuis la [dernière release](https://github.com/te-o-o-o/macseal-builder/releases/latest),
avec `curl`, qui ne pose pas de drapeau de quarantaine :

```bash
curl -fLo macseal-builder https://github.com/te-o-o-o/macseal-builder/releases/latest/download/macseal-builder-darwin-arm64
chmod +x macseal-builder
```

`darwin-amd64` pour les Mac Intel. Téléchargé avec un navigateur :
`xattr -d com.apple.quarantine macseal-builder`.

Depuis les sources : `go install github.com/te-o-o-o/macseal-builder@latest`.

## Utilisation

```bash
# Une fois par développeur, pour toutes vos apps. Sauvegardez les deux fichiers.
macseal-builder cert init -name "Votre Nom" -out ~/secure/signing.p12

# À chaque release
macseal-builder bundle -exe dist/monapp -plist Info.plist -icon MonApp.icns -version 1.2.0 -out MonApp.app
MACSEAL_P12_PASSWORD="$(cat ~/secure/signing.p12.password)" \
  macseal-builder sign -p12 ~/secure/signing.p12 MonApp.app

macseal-builder doctor MonApp.app
macseal-builder verify-update /Applications/MonApp.app MonApp.app
```

Codes de sortie : `0` réussi, `1` une vérification a échoué, `2` la commande
n'a pas pu s'exécuter. Toutes les options : [OPTIONS.md](OPTIONS.md).
Fonctionnement et vérifications : [docs/notes.fr.md](docs/notes.fr.md).

## Développement

```bash
go vet ./... && go test ./...
go build -o macseal-builder .
```

Les tests construisent un vrai bundle, créent des identités et signent avec :
la plupart ne tournent que sur macOS.

## Licence

[MIT](LICENSE)
