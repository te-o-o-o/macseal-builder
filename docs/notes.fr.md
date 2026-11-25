# Notes

## Fonctionnement

- **macOS retient les autorisations par exigence désignée.** TCC (la base
  derrière Accessibilité, Enregistrement de l'écran, Surveillance de
  l'entrée…) enregistre l'exigence de l'app au moment où l'utilisateur
  l'autorise, et compare chaque lancement suivant à elle.
- **Ad hoc veut dire `cdhash`.** Une signature ad hoc n'a pas de certificat :
  son exigence est le hash de cette build précise. Toute nouvelle build est
  une inconnue : l'ancienne entrée reste cochée dans les Réglages et ne
  correspond plus à rien.
- **Un certificat veut dire `identifiant + certificate leaf`.** Toute build
  signée avec le même certificat et le même identifiant de bundle la
  satisfait. Le certificat n'a pas besoin d'être approuvé : codesign signe
  avec un certificat auto-signé (listé `CSSMERR_TP_NOT_TRUSTED`) et la
  vérification de l'exigence fonctionne.
- **Une identité par développeur, pas par app.** L'identifiant de bundle
  distingue déjà les apps. Ne jamais partager une identité entre
  développeurs : qui détient la clé peut signer une mise à jour que toutes
  leurs apps accepteraient.

## Choix

- **Des trousseaux jetables.** La signature importe l'identité dans un
  trousseau d'un dossier temporaire, supprimé ensuite : le trousseau de session
  n'est jamais touché. Il rejoint la liste de recherche de l'utilisateur le
  temps de codesign, puis la liste d'avant est remise : codesign sous
  macOS 15 ne trouve pas l'identité sinon (`--keychain` seul suffit sous
  macOS 27).
- **`sign` s'installe tout seul.** Sans `-p12`, il prend l'identité que
  désigne `config.json` dans `~/Library/Application Support/macseal-builder`,
  créée au premier usage. La config ne contient que ce chemin : l'identité
  peut aller dans un dossier sauvegardé. Une identité perdue est une erreur,
  pas une nouvelle : une nouvelle identité ferait refuser la prochaine mise à
  jour par toutes les copies installées.
- **Le PKCS#12 passe par `security export`.** La bibliothèque standard de Go
  n'a pas d'encodeur PKCS#12, et un fichier écrit par OpenSSL 3 (AES par
  défaut) est refusé par les anciennes versions de macOS.
- **Un certificat de 20 ans.** Le remplacer change l'exigence, donc les
  utilisateurs perdent leurs autorisations. Une racine avec un certificat par
  app et une exigence `certificate root` permettrait la rotation ; pas fait
  tant que ce n'est pas vérifié avec TCC.
- **Pas de `--deep` pour signer.** Apple l'a déprécié ; le code imbriqué est
  signé un par un, le plus profond d'abord. `--deep` sert encore à vérifier.

## Vérifié, par version de macOS

| Affirmation | Comment | Où |
|---|---|---|
| Deux builds signées avec la même identité auto-signée satisfont l'exigence l'une de l'autre ; une build ad hoc ou une autre identité, non | `TestUpdateKeepsRequirement`, et à la main sur des copies de copycopy 0.6.0 | macOS 27.2.1, Apple Silicon |
| La signature et la création d'identité se font sans aucun dialogue | `go test`, et à la main | macOS 27.2.1, Apple Silicon |
| TCC garde l'autorisation Accessibilité entre deux builds signées avec la même identité | à la main : wrkspaces 0.1.8 installée et autorisée, 0.1.9 installée par-dessus, `verify-update` passé, l'Accessibilité marche toujours sans rien redemander | macOS 27.2.1, Apple Silicon |
| Une release signée par la GitHub Action et téléchargée par un navigateur garde aussi l'autorisation ; Gatekeeper bloque toujours son premier lancement (seulement « Terminé » ou « Placer dans la corbeille », puis Réglages Système → Confidentialité et sécurité → Ouvrir quand même) | à la main : le `.dmg` de wrkspaces 0.1.10 par-dessus la 0.1.9, l'Accessibilité n'est pas redemandée | macOS 27.2.1, Apple Silicon |
| Mac Intel, anciennes versions de macOS | pas lancé | |

La CI sur `macos-15` lance les mêmes tests : elle montre qu'ils passent là,
pas que TCC se comporte bien.
