# Code tiers : Swagger UI

Les fichiers de `assets/` sont ceux de la distribution officielle **swagger-ui-dist**, copiés sans
aucune modification.

| | |
|---|---|
| Paquet | [`swagger-ui-dist`](https://www.npmjs.com/package/swagger-ui-dist) (SmartBear) |
| Version | 5.34.0 |
| Licence | Apache-2.0 (fichiers `LICENSE`, `NOTICE` et `swagger-ui-bundle.js.LICENSE.txt` conservés) |
| Intégrité de l'archive | `sha512-H6tAIjr3XqE31PbkZl0OFya1ziRLDfTwh3J6OHdumPtPCLfjTnyLeagl/55kFNZcTOz9a9QI9WP0FXldwiGJag==` (celle du registre npm) |

Seuls les fichiers utiles sont embarqués : `swagger-ui-bundle.js`, `swagger-ui.css`,
`favicon-32x32.png` et les trois fichiers de licence. Les `index.html` et `initializer.js` du
dossier parent sont les nôtres (la politique de sécurité de la page interdit les scripts en ligne).

Les empreintes SHA-256 de chaque fichier sont dans `TestSwaggerUIFilesAreUnmodified`
(`apidocs_test.go`) : le test échoue si un fichier est modifié.

## Mettre à jour la version

Dans un dossier temporaire **hors du dépôt** :

```bash
npm view swagger-ui-dist version dist.integrity   # version et empreinte annoncées par le registre
npm pack swagger-ui-dist                          # télécharge l'archive
openssl dgst -sha512 -binary swagger-ui-dist-*.tgz | openssl base64 -A   # doit égaler dist.integrity
tar -xzf swagger-ui-dist-*.tgz                    # extrait dans package/
```

Puis, dans le dépôt :

1. copier les six fichiers listés ci-dessus dans `assets/` ;
2. mettre à jour les empreintes (`shasum -a 256 assets/*`) dans le test, et la version et l'intégrité
   ici ;
3. `make test`, puis ouvrir la page (`make up-dev`, `/api/v1/docs`) et vérifier que les opérations
   s'affichent et que la console du navigateur ne signale aucune violation de la politique de
   sécurité.

`govulncheck` ne voit pas ces fichiers : surveiller les avis de sécurité de Swagger UI.
